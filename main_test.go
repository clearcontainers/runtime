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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dlespiau/covertool/pkg/cover"

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
)

const (
	testDisabledNeedRoot    = "Test disabled as requires root user"
	testDisabledNeedNonRoot = "Test disabled as requires non-root user"
	testDirMode             = os.FileMode(0750)
	testFileMode            = os.FileMode(0640)
	testExeFileMode         = os.FileMode(0750)

	testPodID       = "99999999-9999-9999-99999999999999999"
	testContainerID = "1"
	testBundle      = "bundle"
	testKernel      = "kernel"
	testImage       = "image"
	testHypervisor  = "hypervisor"

	MockHypervisor vc.HypervisorType = "mock"
	NoopAgentType  vc.AgentType      = "noop"
)

// package variables set in TestMain
var testDir = ""

var testPodAnnotations = map[string]string{
	"pod.foo":   "pod.bar",
	"pod.hello": "pod.world",
}

var testContainerAnnotations = map[string]string{
	"container.foo":   "container.bar",
	"container.hello": "container.world",
}

func runUnitTests(m *testing.M) {
	var err error

	testDir, err = ioutil.TempDir("", fmt.Sprintf("%s-", name))
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(testDir, testDirMode)
	if err != nil {
		fmt.Printf("Could not create test directory %s: %s\n", testDir, err)
		os.Exit(1)
	}

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

func newTestPodConfigNoop() vc.PodConfig {
	// Define the container command and bundle.
	container := vc.ContainerConfig{
		ID:          testContainerID,
		RootFs:      filepath.Join(testDir, testBundle),
		Cmd:         newTestCmd(),
		Annotations: testContainerAnnotations,
	}

	// Sets the hypervisor configuration.
	hypervisorConfig := vc.HypervisorConfig{
		KernelPath:     filepath.Join(testDir, testKernel),
		ImagePath:      filepath.Join(testDir, testImage),
		HypervisorPath: filepath.Join(testDir, testHypervisor),
	}

	podConfig := vc.PodConfig{
		ID:               testPodID,
		HypervisorType:   MockHypervisor,
		HypervisorConfig: hypervisorConfig,

		AgentType: NoopAgentType,

		Containers: []vc.ContainerConfig{container},

		Annotations: testPodAnnotations,
	}

	return podConfig
}

func newTestHypervisorConfig(dir string) (vc.HypervisorConfig, error) {
	if dir == "" {
		return vc.HypervisorConfig{}, fmt.Errorf("BUG: need directory")
	}

	kernelPath := path.Join(dir, "kernel")
	imagePath := path.Join(dir, "image")
	hypervisorPath := path.Join(dir, "hypervisor")

	for _, file := range []string{kernelPath, imagePath, hypervisorPath} {
		err := createEmptyFile(file)
		if err != nil {
			return vc.HypervisorConfig{}, err
		}
	}

	return vc.HypervisorConfig{
		KernelPath:            kernelPath,
		ImagePath:             imagePath,
		HypervisorPath:        hypervisorPath,
		HypervisorMachineType: "pc-lite",
	}, nil
}

func newRuntimeConfig(dir, consolePath string) (oci.RuntimeConfig, error) {
	hypervisorConfig, err := newTestHypervisorConfig(dir)
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

// createOCIConfig creates an OCI configuration file ("config.json") in
// the bundle directory specified (which must exist).
func createOCIConfig(bundleDir string) error {

	if bundleDir == "" {
		return fmt.Errorf("Need bundle directory")
	}

	if !fileExists(bundleDir) {
		return fmt.Errorf("Bundle directory %s does not exist", bundleDir)
	}

	var configCmd string

	for _, cmd := range []string{"docker-runc", "runc"} {
		fullPath, err := exec.LookPath(cmd)
		if err == nil {
			configCmd = fullPath
			break
		}
	}

	if configCmd == "" {
		return fmt.Errorf("Cannot find command to generate OCI config file")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	err = os.Chdir(bundleDir)
	if err != nil {
		return err
	}

	defer func() {
		err = os.Chdir(cwd)
	}()

	_, err = runCommand([]string{configCmd, "spec"})
	if err != nil {
		return err
	}

	specFile := filepath.Join(bundleDir, "config.json")
	if !fileExists(specFile) {
		return fmt.Errorf("generated OCI config file does not exist: %v", specFile)
	}

	return nil
}

// makeOCIBundle will create an OCI bundle (including the "config.json"
// config file)  in the directory specified (which must already exist).
func makeOCIBundle(bundleDir string) error {
	if bundleDir == "" {
		return fmt.Errorf("Need bundle directory")
	}

	if defaultPauseRootPath == "" {
		return fmt.Errorf("BUG: defaultPauseRootPath unset")
	}

	// make use of the existing pause bundle
	_, err := runCommand([]string{"cp", "-a", defaultPauseRootPath, bundleDir})
	if err != nil {
		return err
	}

	err = createOCIConfig(bundleDir)
	if err != nil {
		return err
	}

	// Note the unusual parameter!
	spec, err := oci.ParseConfigJSON(bundleDir)
	if err != nil {
		return err
	}

	// Determine the rootfs directory name the OCI config refers to
	rootDir := spec.Root.Path

	base := filepath.Base(defaultPauseRootPath)
	from := filepath.Join(bundleDir, base)
	to := rootDir

	if !strings.HasPrefix(rootDir, "/") {
		to = filepath.Join(bundleDir, rootDir)
	}

	_, err = runCommand([]string{"mv", from, to})
	if err != nil {
		return err
	}

	return nil
}
