package dockerfile

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/backend"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/builder"
	"github.com/docker/docker/builder/dockerfile/command"
	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/docker/docker/builder/remotecontext"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/sync/syncmap"
)

var validCommitCommands = map[string]bool{
	"cmd":         true,
	"entrypoint":  true,
	"healthcheck": true,
	"env":         true,
	"expose":      true,
	"label":       true,
	"onbuild":     true,
	"user":        true,
	"volume":      true,
	"workdir":     true,
}

// BuildManager is shared across all Builder objects
type BuildManager struct {
	backend   builder.Backend
	pathCache pathCache // TODO: make this persistent
}

// NewBuildManager creates a BuildManager
func NewBuildManager(b builder.Backend) *BuildManager {
	return &BuildManager{
		backend:   b,
		pathCache: &syncmap.Map{},
	}
}

// Build starts a new build from a BuildConfig
func (bm *BuildManager) Build(ctx context.Context, config backend.BuildConfig) (*builder.Result, error) {
	buildsTriggered.Inc()
	if config.Options.Dockerfile == "" {
		config.Options.Dockerfile = builder.DefaultDockerfileName
	}

	source, dockerfile, err := remotecontext.Detect(config)
	if err != nil {
		return nil, err
	}
	if source != nil {
		defer func() {
			if err := source.Close(); err != nil {
				logrus.Debugf("[BUILDER] failed to remove temporary context: %v", err)
			}
		}()
	}

	builderOptions := builderOptions{
		Options:        config.Options,
		ProgressWriter: config.ProgressWriter,
		Backend:        bm.backend,
		PathCache:      bm.pathCache,
	}
	return newBuilder(ctx, builderOptions).build(source, dockerfile)
}

// builderOptions are the dependencies required by the builder
type builderOptions struct {
	Options        *types.ImageBuildOptions
	Backend        builder.Backend
	ProgressWriter backend.ProgressWriter
	PathCache      pathCache
}

// Builder is a Dockerfile builder
// It implements the builder.Backend interface.
type Builder struct {
	options *types.ImageBuildOptions

	Stdout io.Writer
	Stderr io.Writer
	Aux    *streamformatter.AuxFormatter
	Output io.Writer

	docker    builder.Backend
	clientCtx context.Context

	buildStages      *buildStages
	disableCommit    bool
	buildArgs        *buildArgs
	imageSources     *imageSources
	pathCache        pathCache
	containerManager *containerManager
	imageProber      ImageProber
}

// newBuilder creates a new Dockerfile builder from an optional dockerfile and a Options.
func newBuilder(clientCtx context.Context, options builderOptions) *Builder {
	config := options.Options
	if config == nil {
		config = new(types.ImageBuildOptions)
	}
	b := &Builder{
		clientCtx:        clientCtx,
		options:          config,
		Stdout:           options.ProgressWriter.StdoutFormatter,
		Stderr:           options.ProgressWriter.StderrFormatter,
		Aux:              options.ProgressWriter.AuxFormatter,
		Output:           options.ProgressWriter.Output,
		docker:           options.Backend,
		buildArgs:        newBuildArgs(config.BuildArgs),
		buildStages:      newBuildStages(),
		imageSources:     newImageSources(clientCtx, options),
		pathCache:        options.PathCache,
		imageProber:      newImageProber(options.Backend, config.CacheFrom, config.NoCache),
		containerManager: newContainerManager(options.Backend),
	}
	return b
}

// Build runs the Dockerfile builder by parsing the Dockerfile and executing
// the instructions from the file.
func (b *Builder) build(source builder.Source, dockerfile *parser.Result) (*builder.Result, error) {
	defer b.imageSources.Unmount()

	addNodesForLabelOption(dockerfile.AST, b.options.Labels)

	if err := checkDispatchDockerfile(dockerfile.AST); err != nil {
		buildsFailed.WithValues(metricsDockerfileSyntaxError).Inc()
		return nil, err
	}

	dispatchState, err := b.dispatchDockerfileWithCancellation(dockerfile, source)
	if err != nil {
		return nil, err
	}

	if b.options.Target != "" && !dispatchState.isCurrentStage(b.options.Target) {
		buildsFailed.WithValues(metricsBuildTargetNotReachableError).Inc()
		return nil, errors.Errorf("failed to reach build target %s in Dockerfile", b.options.Target)
	}

	b.buildArgs.WarnOnUnusedBuildArgs(b.Stderr)

	if dispatchState.imageID == "" {
		buildsFailed.WithValues(metricsDockerfileEmptyError).Inc()
		return nil, errors.New("No image was generated. Is your Dockerfile empty?")
	}
	return &builder.Result{ImageID: dispatchState.imageID, FromImage: dispatchState.baseImage}, nil
}

func emitImageID(aux *streamformatter.AuxFormatter, state *dispatchState) error {
	if aux == nil || state.imageID == "" {
		return nil
	}
	return aux.Emit(types.BuildResult{ID: state.imageID})
}

func (b *Builder) dispatchDockerfileWithCancellation(dockerfile *parser.Result, source builder.Source) (*dispatchState, error) {
	shlex := NewShellLex(dockerfile.EscapeToken)
	state := newDispatchState()
	total := len(dockerfile.AST.Children)
	var err error
	for i, n := range dockerfile.AST.Children {
		select {
		case <-b.clientCtx.Done():
			logrus.Debug("Builder: build cancelled!")
			fmt.Fprint(b.Stdout, "Build cancelled")
			buildsFailed.WithValues(metricsBuildCanceled).Inc()
			return nil, errors.New("Build cancelled")
		default:
			// Not cancelled yet, keep going...
		}

		// If this is a FROM and we have a previous image then
		// emit an aux message for that image since it is the
		// end of the previous stage
		if n.Value == command.From {
			if err := emitImageID(b.Aux, state); err != nil {
				return nil, err
			}
		}

		if n.Value == command.From && state.isCurrentStage(b.options.Target) {
			break
		}

		opts := dispatchOptions{
			state:   state,
			stepMsg: formatStep(i, total),
			node:    n,
			shlex:   shlex,
			source:  source,
		}
		if state, err = b.dispatch(opts); err != nil {
			if b.options.ForceRemove {
				b.containerManager.RemoveAll(b.Stdout)
			}
			return nil, err
		}

		fmt.Fprintf(b.Stdout, " ---> %s\n", stringid.TruncateID(state.imageID))
		if b.options.Remove {
			b.containerManager.RemoveAll(b.Stdout)
		}
	}

	// Emit a final aux message for the final image
	if err := emitImageID(b.Aux, state); err != nil {
		return nil, err
	}

	return state, nil
}

func addNodesForLabelOption(dockerfile *parser.Node, labels map[string]string) {
	if len(labels) == 0 {
		return
	}

	node := parser.NodeFromLabels(labels)
	dockerfile.Children = append(dockerfile.Children, node)
}

// BuildFromConfig builds directly from `changes`, treating it as if it were the contents of a Dockerfile
// It will:
// - Call parse.Parse() to get an AST root for the concatenated Dockerfile entries.
// - Do build by calling builder.dispatch() to call all entries' handling routines
//
// BuildFromConfig is used by the /commit endpoint, with the changes
// coming from the query parameter of the same name.
//
// TODO: Remove?
func BuildFromConfig(config *container.Config, changes []string) (*container.Config, error) {
	if len(changes) == 0 {
		return config, nil
	}

	b := newBuilder(context.Background(), builderOptions{
		Options: &types.ImageBuildOptions{NoCache: true},
	})

	dockerfile, err := parser.Parse(bytes.NewBufferString(strings.Join(changes, "\n")))
	if err != nil {
		return nil, err
	}

	// ensure that the commands are valid
	for _, n := range dockerfile.AST.Children {
		if !validCommitCommands[n.Value] {
			return nil, fmt.Errorf("%s is not a valid change command", n.Value)
		}
	}

	b.Stdout = ioutil.Discard
	b.Stderr = ioutil.Discard
	b.disableCommit = true

	if err := checkDispatchDockerfile(dockerfile.AST); err != nil {
		return nil, err
	}
	dispatchState := newDispatchState()
	dispatchState.runConfig = config
	return dispatchFromDockerfile(b, dockerfile, dispatchState, nil)
}

func checkDispatchDockerfile(dockerfile *parser.Node) error {
	for _, n := range dockerfile.Children {
		if err := checkDispatch(n); err != nil {
			return errors.Wrapf(err, "Dockerfile parse error line %d", n.StartLine)
		}
	}
	return nil
}

func dispatchFromDockerfile(b *Builder, result *parser.Result, dispatchState *dispatchState, source builder.Source) (*container.Config, error) {
	shlex := NewShellLex(result.EscapeToken)
	ast := result.AST
	total := len(ast.Children)

	for i, n := range ast.Children {
		opts := dispatchOptions{
			state:   dispatchState,
			stepMsg: formatStep(i, total),
			node:    n,
			shlex:   shlex,
			source:  source,
		}
		if _, err := b.dispatch(opts); err != nil {
			return nil, err
		}
	}
	return dispatchState.runConfig, nil
}
