//
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
//

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
	"github.com/containers/virtcontainers/pkg/vcMock"
	"github.com/dlespiau/covertool/pkg/cover"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

const (
	testDisabledNeedRoot    = "Test disabled as requires root user"
	testDisabledNeedNonRoot = "Test disabled as requires non-root user"
	testDirMode             = os.FileMode(0750)
	testFileMode            = os.FileMode(0640)
	testExeFileMode         = os.FileMode(0750)

	// small docker image used to create root filesystems from
	testDockerImage = "busybox"

	testPodID       = "99999999-9999-9999-99999999999999999"
	testContainerID = "1"
	testBundle      = "bundle"
	testKernel      = "kernel"
	testImage       = "image"
	testHypervisor  = "hypervisor"

	MockHypervisor vc.HypervisorType = "mock"
	NoopAgentType  vc.AgentType      = "noop"
)

var (
	// package variables set by calling TestMain()
	testDir       = ""
	testBundleDir = ""
)

// testingImpl is a concrete mock RVC implementation used for testing
var testingImpl = &vcMock.VCMock{}

func init() {
	fmt.Printf("INFO: switching to fake virtcontainers implementation for testing\n")
	vci = testingImpl

	var err error

	fmt.Printf("INFO: creating test directory\n")
	testDir, err = ioutil.TempDir("", fmt.Sprintf("%s-", name))
	if err != nil {
		panic(err)
	}

	fmt.Printf("INFO: test directory is %v\n", testDir)

	// Do this now to avoid hitting the test timeout value due to
	// slow network response.
	fmt.Printf("INFO: ensuring required docker image (%v) is available\n", testDockerImage)
	_, err = runCommand([]string{"docker", "pull", testDockerImage})
	if err != nil {
		panic(err)
	}

	testBundleDir = filepath.Join(testDir, testBundle)
	err = os.MkdirAll(testBundleDir, testDirMode)
	if err != nil {
		panic(err)
	}

	fmt.Printf("INFO: creating OCI bundle in %v for tests to use\n", testBundleDir)
	err = realMakeOCIBundle(testBundleDir)
	if err != nil {
		panic(err)
	}
}

var testPodAnnotations = map[string]string{
	"pod.foo":   "pod.bar",
	"pod.hello": "pod.world",
}

var testContainerAnnotations = map[string]string{
	"container.foo":   "container.bar",
	"container.hello": "container.world",
}

func runUnitTests(m *testing.M) {
	ret := m.Run()

	os.RemoveAll(testDir)

	os.Exit(ret)
}

// TestMain is the common main function used by ALL the test functions
// for this package.
func TestMain(m *testing.M) {
	// Parse the command line using the stdlib flag package so the flags defined
	// in the testing package get populated.
	cover.ParseAndStripTestFlags()

	// Make sure we have the opportunity to flush the coverage report to disk when
	// terminating the process.
	atexit(cover.FlushProfiles)

	// If the test binary name is cc-runtime.coverage, we've are being asked to
	// run the coverage-instrumented cc-runtime.
	if path.Base(os.Args[0]) == name+".coverage" ||
		path.Base(os.Args[0]) == name {
		main()
		exit(0)
	}

	runUnitTests(m)
}

func createEmptyFile(path string) (err error) {
	return ioutil.WriteFile(path, []byte(""), testFileMode)
}

// newTestCmd creates a new virtcontainers Cmd to run a shell
func newTestCmd() vc.Cmd {
	envs := []vc.EnvVar{
		{
			Var:   "PATH",
			Value: "/bin:/usr/bin:/sbin:/usr/sbin",
		},
	}

	cmd := vc.Cmd{
		Args:    strings.Split("/bin/sh", " "),
		Envs:    envs,
		WorkDir: "/",
	}

	return cmd
}

// newTestContainerConfig returns a new ContainerConfig
func newTestContainerConfig(dir string) vc.ContainerConfig {
	return vc.ContainerConfig{
		ID:          testContainerID,
		RootFs:      filepath.Join(dir, testBundle),
		Cmd:         newTestCmd(),
		Annotations: testContainerAnnotations,
	}
}

// newTestPodConfigNoop creates a new virtcontainers PodConfig
// (of the most basic type). If create is true, create the required
// resources.
//
// Note: no parameter validation in case caller wishes to create an invalid
// object.
func newTestPodConfigNoop(dir string, create bool) (vc.PodConfig, error) {
	// Sets the hypervisor configuration.
	hypervisorConfig, err := newTestHypervisorConfig(dir, create)
	if err != nil {
		return vc.PodConfig{}, err
	}

	container := newTestContainerConfig(dir)

	podConfig := vc.PodConfig{
		ID:               testPodID,
		HypervisorType:   MockHypervisor,
		HypervisorConfig: hypervisorConfig,

		AgentType: NoopAgentType,

		Containers: []vc.ContainerConfig{container},

		Annotations: testPodAnnotations,
	}

	return podConfig, nil
}

// newTestHypervisorConfig creaets a new virtcontainers
// HypervisorConfig, ensuring that the required resources are also
// created.
//
// Note: no parameter validation in case caller wishes to create an invalid
// object.
func newTestHypervisorConfig(dir string, create bool) (vc.HypervisorConfig, error) {
	kernelPath := path.Join(dir, "kernel")
	imagePath := path.Join(dir, "image")
	hypervisorPath := path.Join(dir, "hypervisor")

	if create {
		for _, file := range []string{kernelPath, imagePath, hypervisorPath} {
			err := createEmptyFile(file)
			if err != nil {
				return vc.HypervisorConfig{}, err
			}
		}
	}

	return vc.HypervisorConfig{
		KernelPath:            kernelPath,
		ImagePath:             imagePath,
		HypervisorPath:        hypervisorPath,
		HypervisorMachineType: "pc-lite",
	}, nil
}

// newTestRuntimeConfig creates a new RuntimeConfig
func newTestRuntimeConfig(dir, consolePath string, create bool) (oci.RuntimeConfig, error) {
	if dir == "" {
		return oci.RuntimeConfig{}, errors.New("BUG: need directory")
	}

	hypervisorConfig, err := newTestHypervisorConfig(dir, create)
	if err != nil {
		return oci.RuntimeConfig{}, err
	}

	return oci.RuntimeConfig{
		HypervisorType:   vc.QemuHypervisor,
		HypervisorConfig: hypervisorConfig,
		AgentType:        vc.HyperstartAgent,
		ProxyType:        vc.CCProxyType,
		ShimType:         vc.CCShimType,
		Console:          consolePath,
	}, nil
}

// createOCIConfig creates an OCI configuration (spec) file in
// the bundle directory specified (which must exist).
func createOCIConfig(bundleDir string) error {
	if bundleDir == "" {
		return errors.New("BUG: Need bundle directory")
	}

	if !fileExists(bundleDir) {
		return fmt.Errorf("BUG: Bundle directory %s does not exist", bundleDir)
	}

	var configCmd string

	// Search for a suitable version of runc to use to generate
	// the OCI config file.
	for _, cmd := range []string{"docker-runc", "runc"} {
		fullPath, err := exec.LookPath(cmd)
		if err == nil {
			configCmd = fullPath
			break
		}
	}

	if configCmd == "" {
		return errors.New("Cannot find command to generate OCI config file")
	}

	_, err := runCommand([]string{configCmd, "spec", "--bundle", bundleDir})
	if err != nil {
		return err
	}

	specFile := filepath.Join(bundleDir, specConfig)
	if !fileExists(specFile) {
		return fmt.Errorf("generated OCI config file does not exist: %v", specFile)
	}

	return nil
}

// createRootfs creates a minimal root filesystem below the specified
// directory.
func createRootfs(dir string) error {
	err := os.MkdirAll(dir, testDirMode)
	if err != nil {
		return err
	}

	container, err := runCommand([]string{"docker", "create", testDockerImage})
	if err != nil {
		return err
	}

	cmd1 := exec.Command("docker", "export", container)
	cmd2 := exec.Command("tar", "-C", dir, "-xvf", "-")

	cmd1Stdout, err := cmd1.StdoutPipe()
	if err != nil {
		return err
	}

	cmd2.Stdin = cmd1Stdout

	err = cmd2.Start()
	if err != nil {
		return err
	}

	err = cmd1.Run()
	if err != nil {
		return err
	}

	err = cmd2.Wait()
	if err != nil {
		return err
	}

	// Clean up
	_, err = runCommand([]string{"docker", "rm", container})
	if err != nil {
		return err
	}

	return nil
}

// makeOCIBundle will create an OCI bundle (including the "config.json"
// config file) in the directory specified (which must already exist).
func realMakeOCIBundle(bundleDir string) error {
	if bundleDir == "" {
		return errors.New("BUG: Need bundle directory")
	}

	if !fileExists(bundleDir) {
		return fmt.Errorf("BUG: Bundle directory %v does not exist", bundleDir)
	}

	err := createOCIConfig(bundleDir)
	if err != nil {
		return err
	}

	// Note the unusual parameter (a directory, not the config
	// file to parse!)
	spec, err := oci.ParseConfigJSON(bundleDir)
	if err != nil {
		return err
	}

	// Determine the rootfs directory name the OCI config refers to
	ociRootPath := spec.Root.Path

	rootfsDir := filepath.Join(bundleDir, ociRootPath)

	if strings.HasPrefix(ociRootPath, "/") {
		return fmt.Errorf("Cannot handle absolute rootfs as bundle must be unique to each test")
	}

	err = createRootfs(rootfsDir)
	if err != nil {
		return err
	}

	return nil
}

// Create an OCI bundle in the specified directory.
//
// Note that the directory will be created, but it's parent is expected to exist.
//
// This function works by copying the already-created test bundle. Ideally,
// the bundle would be recreated for each test, but createRootfs() uses
// docker which on some systems is too slow, resulting in the tests timing
// out.
func makeOCIBundle(bundleDir string) error {
	from := testBundleDir
	to := bundleDir

	// only the basename of bundleDir needs to exist as bundleDir
	// will get created by cp(1).
	base := filepath.Dir(bundleDir)

	for _, dir := range []string{from, base} {
		if !fileExists(dir) {
			return fmt.Errorf("BUG: directory %v should exist", dir)
		}
	}

	output, err := runCommandFull([]string{"cp", "-a", from, to}, true)
	if err != nil {
		return fmt.Errorf("failed to copy test OCI bundle from %v to %v: %v (output: %v)", from, to, err, output)
	}

	return nil
}

// readOCIConfig returns an OCI spec.
func readOCIConfigFile(configPath string) (oci.CompatOCISpec, error) {
	if configPath == "" {
		return oci.CompatOCISpec{}, errors.New("BUG: need config file path")
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return oci.CompatOCISpec{}, err
	}

	var ociSpec oci.CompatOCISpec
	if err := json.Unmarshal(data, &ociSpec); err != nil {
		return oci.CompatOCISpec{}, err
	}

	return ociSpec, nil
}

func writeOCIConfigFile(spec oci.CompatOCISpec, configPath string) error {
	if configPath == "" {
		return errors.New("BUG: need config file path")
	}

	bytes, err := json.MarshalIndent(spec, "", "\t")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, bytes, testFileMode)
}

func newSingleContainerPodStatusList(podID, containerID string, podState, containerState vc.State, annotations map[string]string) []vc.PodStatus {
	return []vc.PodStatus{
		{
			ID:    podID,
			State: podState,
			ContainersStatus: []vc.ContainerStatus{
				{
					ID:          containerID,
					State:       containerState,
					Annotations: annotations,
				},
			},
		},
	}
}

func execCLICommandFunc(assertHandler *assert.Assertions, cliCommand cli.Command, set *flag.FlagSet, expectedErr bool) {
	app := cli.NewApp()
	ctx := cli.NewContext(app, set, nil)
	app.Name = "foo"

	fn, ok := cliCommand.Action.(func(context *cli.Context) error)
	assertHandler.True(ok)

	err := fn(ctx)

	if expectedErr {
		assertHandler.Error(err)
	} else {
		assertHandler.Nil(err)
	}
}

func TestMakeOCIBundle(t *testing.T) {
	assert := assert.New(t)

	tmpdir, err := ioutil.TempDir(testDir, "")
	assert.NoError(err)
	defer os.RemoveAll(tmpdir)

	bundleDir := filepath.Join(tmpdir, "bundle")

	err = makeOCIBundle(bundleDir)
	assert.NoError(err)

	specFile := filepath.Join(bundleDir, specConfig)
	assert.True(fileExists(specFile))
}

func TestCreateOCIConfig(t *testing.T) {
	assert := assert.New(t)

	tmpdir, err := ioutil.TempDir(testDir, "")
	assert.NoError(err)
	defer os.RemoveAll(tmpdir)

	bundleDir := filepath.Join(tmpdir, "bundle")

	err = createOCIConfig(bundleDir)
	// ENOENT
	assert.Error(err)

	err = os.MkdirAll(bundleDir, testDirMode)
	assert.NoError(err)

	err = createOCIConfig(bundleDir)
	assert.NoError(err)

	specFile := filepath.Join(bundleDir, specConfig)
	assert.True(fileExists(specFile))
}

func TestCreateRootfs(t *testing.T) {
	assert := assert.New(t)

	tmpdir, err := ioutil.TempDir(testDir, "")
	assert.NoError(err)
	defer os.RemoveAll(tmpdir)

	rootfsDir := filepath.Join(tmpdir, "rootfs")
	assert.False(fileExists(rootfsDir))

	err = createRootfs(rootfsDir)
	assert.NoError(err)

	// non-comprehensive list of expected directories
	expectedDirs := []string{"bin", "dev", "etc", "usr", "var"}

	assert.True(fileExists(rootfsDir))

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(rootfsDir, dir)
		assert.True(fileExists(dirPath))
	}
}
