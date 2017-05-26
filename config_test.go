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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
)

const proxyURL = "foo:///foo/clear-containers/proxy.sock"

func makeRuntimeConfigFileData(hypervisorPath, kernelPath, imagePath, shimPath, agentPauseRootPath, proxyURL string) string {
	return `
	# Clear Containers runtime configuration file

	[hypervisor.qemu-lite]
	path = "` + hypervisorPath + `"
	kernel = "` + kernelPath + `"
	image = "` + imagePath + `"

	[proxy.cc]
	url = "` + proxyURL + `"

	[shim.cc]
	path = "` + shimPath + `"

	[agent.hyperstart]
	pause_root_path = "` + agentPauseRootPath + `"
	`
}

func createConfig(fileName string, fileData string) (string, error) {
	configPath := path.Join(testDir, fileName)

	err := ioutil.WriteFile(configPath, []byte(fileData), testFileMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create config file %s %v\n", configPath, err)
		return "", err
	}

	return configPath, nil
}

func TestRuntimeConfig(t *testing.T) {
	dir, err := ioutil.TempDir(testDir, "runtime-config-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	hypervisorPath := path.Join(dir, "hypervisor")
	kernelPath := path.Join(dir, "kernel")
	imagePath := path.Join(dir, "image")
	shimPath := path.Join(dir, "shim")
	agentPauseRootPath := path.Join(dir, "agentPauseRoot")
	agentPauseRootBin := path.Join(agentPauseRootPath, "bin")
	pauseBinPath := path.Join(agentPauseRootBin, "pause")

	runtimeConfig := makeRuntimeConfigFileData(hypervisorPath, kernelPath, imagePath, shimPath, agentPauseRootPath, proxyURL)

	configPath, err := createConfig("runtime.toml", runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}

	config, err := loadConfiguration(configPath)
	if err == nil {
		t.Fatalf("Expected loadConfiguration to fail as no paths exist: %+v", config)
	}

	err = os.MkdirAll(agentPauseRootBin, testDirMode)
	if err != nil {
		t.Fatal(err)
	}

	config, err = loadConfiguration(configPath)
	if err == nil {
		t.Fatalf("Expected loadConfiguration to fail as only pause path exists: %+v", config)
	}

	files := []string{pauseBinPath, hypervisorPath, kernelPath, imagePath, shimPath}
	filesLen := len(files)

	for i, file := range files {
		_, err = loadConfiguration(configPath)
		if err == nil {
			t.Fatalf("Expected loadConfiguration to fail as not all paths exist (not created %v)",
				strings.Join(files[i:filesLen], ","))
		}

		// create the resource
		err = createEmptyFile(file)
		if err != nil {
			t.Error(err)
		}
	}

	// all paths exist now
	config, err = loadConfiguration(configPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedHypervisorConfig := vc.HypervisorConfig{
		HypervisorPath: hypervisorPath,
		KernelPath:     kernelPath,
		ImagePath:      imagePath,
	}

	expectedAgentConfig := vc.HyperConfig{
		PauseBinPath: filepath.Join(agentPauseRootPath, pauseBinRelativePath),
	}

	expectedProxyConfig := vc.CCProxyConfig{
		URL: proxyURL,
	}

	expectedShimConfig := vc.CCShimConfig{
		Path: shimPath,
	}

	expectedConfig := oci.RuntimeConfig{
		HypervisorType:   defaultHypervisor,
		HypervisorConfig: expectedHypervisorConfig,

		AgentType:   defaultAgent,
		AgentConfig: expectedAgentConfig,

		ProxyType:   defaultProxy,
		ProxyConfig: expectedProxyConfig,

		ShimType:   defaultShim,
		ShimConfig: expectedShimConfig,
	}

	if reflect.DeepEqual(config, expectedConfig) == false {
		t.Fatalf("Got %v\n expecting %v", config, expectedConfig)
	}

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestMinimalRuntimeConfig(t *testing.T) {
	dir, err := ioutil.TempDir(testDir, "minimal-runtime-config-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	shimPath := path.Join(dir, "shim")

	runtimeMinimalConfig := `
	# Clear Containers runtime configuration file

	[proxy.cc]
	url = "` + proxyURL + `"

	[shim.cc]
	path = "` + shimPath + `"
`

	configPath, err := createConfig("runtime.toml", runtimeMinimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	config, err := loadConfiguration(configPath)
	if err == nil {
		t.Fatalf("Expected loadConfiguration to fail as shim path does not exist: %+v", config)
	}

	err = createEmptyFile(shimPath)
	if err != nil {
		t.Error(err)
	}

	config, err = loadConfiguration(configPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedHypervisorConfig := vc.HypervisorConfig{
		HypervisorPath: defaultHypervisorPath,
		KernelPath:     defaultKernelPath,
		ImagePath:      defaultImagePath,
	}

	expectedAgentConfig := vc.HyperConfig{
		PauseBinPath: filepath.Join(defaultPauseRootPath, pauseBinRelativePath),
	}

	expectedProxyConfig := vc.CCProxyConfig{
		URL: proxyURL,
	}

	expectedShimConfig := vc.CCShimConfig{
		Path: shimPath,
	}

	expectedConfig := oci.RuntimeConfig{
		HypervisorType:   defaultHypervisor,
		HypervisorConfig: expectedHypervisorConfig,

		AgentType:   defaultAgent,
		AgentConfig: expectedAgentConfig,

		ProxyType:   defaultProxy,
		ProxyConfig: expectedProxyConfig,

		ShimType:   defaultShim,
		ShimConfig: expectedShimConfig,
	}

	if reflect.DeepEqual(config, expectedConfig) == false {
		t.Fatalf("Got %v\n expecting %v", config, expectedConfig)
	}

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestNewQemuHypervisorConfig(t *testing.T) {
	dir, err := ioutil.TempDir(testDir, "hypervisor-config-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	hypervisorPath := path.Join(dir, "hypervisor")
	kernelPath := path.Join(dir, "kernel")
	imagePath := path.Join(dir, "image")

	hypervisor := hypervisor{
		Path:   hypervisorPath,
		Kernel: kernelPath,
		Image:  imagePath,
	}

	files := []string{hypervisorPath, kernelPath, imagePath}
	filesLen := len(files)

	for i, file := range files {
		_, err := newQemuHypervisorConfig(hypervisor)
		if err == nil {
			t.Fatalf("Expected newQemuHypervisorConfig to fail as not all paths exist (not created %v)",
				strings.Join(files[i:filesLen], ","))
		}

		// create the resource
		err = createEmptyFile(file)
		if err != nil {
			t.Error(err)
		}
	}

	// all paths exist now
	config, err := newQemuHypervisorConfig(hypervisor)
	if err != nil {
		t.Fatal(err)
	}

	if config.HypervisorPath != hypervisor.Path {
		t.Errorf("Expected hypervisor path %v, got %v", hypervisor.Path, config.HypervisorPath)
	}

	if config.KernelPath != hypervisor.Kernel {
		t.Errorf("Expected kernel path %v, got %v", hypervisor.Kernel, config.KernelPath)
	}

	if config.ImagePath != hypervisor.Image {
		t.Errorf("Expected image path %v, got %v", hypervisor.Image, config.ImagePath)
	}
}

func TestNewHyperstartAgentConfig(t *testing.T) {
	dir, err := ioutil.TempDir(testDir, "hyperstart-agent-config-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	agentPauseRootPath := path.Join(dir, "agentPauseRoot")
	agentPauseRootBin := path.Join(agentPauseRootPath, "bin")
	pauseBinPath := path.Join(agentPauseRootBin, "pause")

	agent := agent{
		PauseRootPath: agentPauseRootPath,
	}

	_, err = newHyperstartAgentConfig(agent)
	if err == nil {
		t.Fatalf("Expected newHyperstartAgentConfig to fail as no paths exist")
	}

	err = os.MkdirAll(agentPauseRootPath, testDirMode)
	if err != nil {
		t.Fatal(err)
	}

	_, err = newHyperstartAgentConfig(agent)
	if err == nil {
		t.Fatalf("Expected newHyperstartAgentConfig to fail as only pause root path exists")
	}

	err = os.MkdirAll(agentPauseRootBin, testDirMode)
	if err != nil {
		t.Fatal(err)
	}

	_, err = newHyperstartAgentConfig(agent)
	if err == nil {
		t.Fatalf("Expected newHyperstartAgentConfig to fail as only pause bin path exists")
	}

	err = createEmptyFile(pauseBinPath)
	if err != nil {
		t.Error(err)
	}

	agentConfig, err := newHyperstartAgentConfig(agent)
	if err != nil {
		t.Fatalf("newHyperstartAgentConfig failed unexpectedly: %v", err)
	}

	if agentConfig.PauseBinPath != pauseBinPath {
		t.Errorf("Expected pause bin path %v, got %v", pauseBinPath, agentConfig.PauseBinPath)
	}
}

func TestNewCCShimConfig(t *testing.T) {
	dir, err := ioutil.TempDir(testDir, "shim-config-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	shimPath := path.Join(dir, "shim")

	shim := shim{
		Path: shimPath,
	}

	_, err = newCCShimConfig(shim)
	if err == nil {
		t.Fatalf("Expected newCCShimConfig to fail as no paths exist")
	}

	err = createEmptyFile(shimPath)
	if err != nil {
		t.Error(err)
	}

	shConfig, err := newCCShimConfig(shim)
	if err != nil {
		t.Fatalf("newCCShimConfig failed unexpectedly: %v", err)
	}

	if shConfig.Path != shimPath {
		t.Errorf("Expected shim path %v, got %v", shimPath, shConfig.Path)
	}
}
