// Copyright (c) 2014,2015,2016 Docker, Inc.
// Copyright (c) 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	vc "github.com/containers/virtcontainers"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	// name holds the name of this program
	name    = "cc-runtime"
	project = "Intel® Clear Containers"
)

// specConfig is the name of the file holding the containers configuration
const specConfig = "config.json"

const usage = project + ` runtime

cc-runtime is a command line program for running applications packaged
according to the Open Container Initiative (OCI).`

const notes = `
NOTES:

- Commands starting "cc-" and options starting "--cc-" are ` + project + ` extensions.

`

var ccLog = logrus.WithField("source", "runtime")

// concrete virtcontainer implementation
var virtcontainersImpl = &vc.VCImpl{}

// vci is used to access a particular virtcontainers implementation.
// Normally, it refers to the official package, but is re-assigned in
// the tests to allow virtcontainers to be mocked.
var vci vc.VC = virtcontainersImpl

// defaultOutputFile is the default output file to write the gathered
// information to.
var defaultOutputFile = os.Stdout

// defaultErrorFile is the default output file to write error
// messages to.
var defaultErrorFile = os.Stderr

// runtimeFlags is the list of supported global command-line flags
var runtimeFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "cc-config",
		Usage: project + " config file path",
	},
	cli.BoolFlag{
		Name:  "debug",
		Usage: "enable debug output for logging",
	},
	cli.StringFlag{
		Name:  "log",
		Value: "/dev/null",
		Usage: "set the log file path where internal debug information is written",
	},
	cli.StringFlag{
		Name:  "log-format",
		Value: "text",
		Usage: "set the format used by logs ('text' (default), or 'json')",
	},
	cli.StringFlag{
		Name:  "root",
		Value: defaultRootDirectory,
		Usage: "root directory for storage of container state (this should be located in tmpfs)",
	},
	cli.BoolFlag{
		Name:  "cc-show-default-config-paths",
		Usage: "show config file paths that will be checked for (in order)",
	},
}

// runtimeCommands is the list of supported command-line (sub-)
// commands.
var runtimeCommands = []cli.Command{
	createCLICommand,
	deleteCLICommand,
	execCLICommand,
	killCLICommand,
	listCLICommand,
	pauseCLICommand,
	resumeCLICommand,
	runCLICommand,
	startCLICommand,
	stateCLICommand,
	versionCLICommand,

	// Clear Containers specific extensions
	ccCheckCLICommand,
	ccEnvCLICommand,
}

// runtimeBeforeSubcommands is the function to run before command-line
// parsing occurs.
var runtimeBeforeSubcommands = beforeSubcommands

// runtimeCommandNotFound is the function to handle an invalid sub-command.
var runtimeCommandNotFound = commandNotFound

// runtimeVersion is the function that returns the full version
// string describing the runtime.
var runtimeVersion = makeVersionString

// saved default cli package values (for testing).
var savedCLIAppHelpTemplate = cli.AppHelpTemplate
var savedCLIVersionPrinter = cli.VersionPrinter
var savedCLIErrWriter = cli.ErrWriter

// beforeSubcommands is the function to perform preliminary checks
// before command-line parsing occurs.
func beforeSubcommands(context *cli.Context) error {
	if context.GlobalBool("cc-show-default-config-paths") {
		files := getDefaultConfigFilePaths()

		for _, file := range files {
			fmt.Fprintf(defaultOutputFile, "%s\n", file)
		}

		exit(0)
	}

	if userWantsUsage(context) || (context.NArg() == 1 && (context.Args()[0] == "cc-check")) {
		// No setup required if the user just
		// wants to see the usage statement or are
		// running a command that does not manipulate
		// containers.
		return nil
	}

	if context.GlobalBool("debug") {
		ccLog.Logger.Level = logrus.DebugLevel
	}

	if path := context.GlobalString("log"); path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_SYNC, 0640)
		if err != nil {
			return err
		}
		ccLog.Logger.Out = f
	}

	switch context.GlobalString("log-format") {
	case "text":
		// retain logrus's default.
	case "json":
		ccLog.Logger.Formatter = new(logrus.JSONFormatter)
	default:
		return fmt.Errorf("unknown log-format %q", context.GlobalString("log-format"))
	}

	// Set virtcontainers logger.
	vci.SetLogger(ccLog)

	ignoreLogging := false
	if context.NArg() == 1 && context.Args()[0] == "cc-env" {
		// "cc-env" should simply report the logging setup
		ignoreLogging = true
	}

	configFile, logfilePath, runtimeConfig, err := loadConfiguration(context.GlobalString("cc-config"), ignoreLogging)
	if err != nil {
		fatal(err)
	}

	args := strings.Join(context.Args(), " ")

	fields := logrus.Fields{
		"name":      name,
		"version":   version,
		"commit":    commit,
		"arguments": `"` + args + `"`,
	}

	ccLog.WithFields(fields).Info()

	// make the data accessible to the sub-commands.
	context.App.Metadata = map[string]interface{}{
		"runtimeConfig": runtimeConfig,
		"configFile":    configFile,
		"logfilePath":   logfilePath,
	}

	return nil
}

// function called when an invalid command is specified which causes the
// runtime to error.
func commandNotFound(c *cli.Context, command string) {
	err := fmt.Errorf("Invalid command %q", command)
	fatal(err)
}

// makeVersionString returns a multi-line string describing the runtime
// version along with the version of the OCI specification it supports.
func makeVersionString() string {
	v := make([]string, 0, 3)

	versionStr := version
	if versionStr == "" {
		versionStr = unknown
	}

	v = append(v, name+"  : "+versionStr)

	commitStr := commit
	if commitStr == "" {
		commitStr = unknown
	}

	v = append(v, "   commit   : "+commitStr)

	specVersionStr := specs.Version
	if specVersionStr == "" {
		specVersionStr = unknown
	}

	v = append(v, "   OCI specs: "+specVersionStr)

	return strings.Join(v, "\n")
}

// setCLIGlobals modifies various cli package global variables
func setCLIGlobals() {
	cli.AppHelpTemplate = fmt.Sprintf(`%s%s`, cli.AppHelpTemplate, notes)

	// Override the default function to display version details to
	// ensure the "--version" option and "version" command are identical.
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintln(defaultOutputFile, c.App.Version)
	}

	// If the command returns an error, cli takes upon itself to print
	// the error on cli.ErrWriter and exit.
	// Use our own writer here to ensure the log gets sent to the right
	// location.
	cli.ErrWriter = &fatalWriter{cli.ErrWriter}
}

// createRuntimeApp creates an application to process the command-line
// arguments and invoke the requested runtime command.
func createRuntimeApp(args []string) error {
	app := cli.NewApp()

	app.Name = name
	app.Writer = defaultOutputFile
	app.Usage = usage
	app.CommandNotFound = runtimeCommandNotFound
	app.Version = runtimeVersion()
	app.Flags = runtimeFlags
	app.Commands = runtimeCommands
	app.Before = runtimeBeforeSubcommands
	app.EnableBashCompletion = true

	return app.Run(args)
}

// userWantsUsage determines if the user only wishes to see the usage
// statement.
func userWantsUsage(context *cli.Context) bool {
	if context.NArg() == 0 {
		return true
	}

	if context.NArg() == 1 && (context.Args()[0] == "help" || context.Args()[0] == "version") {
		return true
	}

	if context.NArg() >= 2 && (context.Args()[1] == "-h" || context.Args()[1] == "--help") {
		return true
	}

	return false
}

// fatal prints the error's details exits the program.
func fatal(err error) {
	ccLog.Error(err)
	fmt.Fprintln(defaultErrorFile, err)
	exit(1)
}

type fatalWriter struct {
	cliErrWriter io.Writer
}

func (f *fatalWriter) Write(p []byte) (n int, err error) {
	// Ensure error is logged before displaying to the user
	ccLog.Error(string(p))
	return f.cliErrWriter.Write(p)
}

func createRuntime() {
	setCLIGlobals()

	err := createRuntimeApp(os.Args)
	if err != nil {
		fatal(err)
	}
}

func main() {
	createRuntime()
}
