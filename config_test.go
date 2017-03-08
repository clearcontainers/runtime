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
	"reflect"
	"testing"

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
)

const hypervisorPath = "/foo/qemu-lite-system-x86_64"
const kernelPath = "/foo/clear-containers/vmlinux.container"
const imagePath = "/foo/clear-containers/clear-containers.img"
const runtimePath = "/foo/clear-containers/runtime.sock"
const shimPath = "/foo/clear-containers/cc-shim"

const runtimeConfig = `
# Clear Containers runtime configuration file

[hypervisor.qemu-lite]
path = "` + hypervisorPath + `"
kernel = "` + kernelPath + `"
image = "` + imagePath + `"

[proxy.cc]
runtime_sock = "` + runtimePath + `"

[shim.cc]
path = "` + shimPath + `"
`

const runtimeMinimalConfig = `
# Clear Containers runtime configuration file

[proxy.cc]
runtime_sock = "` + runtimePath + `"
shim_sock = "` + shimPath + `"
`

const tempRuntimePath = "/tmp/cc-runtime/"

func createConfig(fileName string, fileData string) (string, error) {
	configPath := path.Join(tempRuntimePath, fileName)

	err := ioutil.WriteFile(configPath, []byte(fileData), 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create config file %s %v\n", configPath, err)
		return "", err
	}

	return configPath, nil
}

func TestRuntimeConfig(t *testing.T) {
	configPath, err := createConfig("runtime.toml", runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}

	config, shimConfig, err := loadConfiguration(configPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedHypervisorConfig := vc.HypervisorConfig{
		HypervisorPath: hypervisorPath,
		KernelPath:     kernelPath,
		ImagePath:      imagePath,
	}

	expectedProxyConfig := vc.CCProxyConfig{
		URL: runtimePath,
	}

	expectedConfig := oci.RuntimeConfig{
		HypervisorType:   defaultHypervisor,
		HypervisorConfig: expectedHypervisorConfig,

		AgentType: defaultAgent,

		ProxyType:   defaultProxy,
		ProxyConfig: expectedProxyConfig,
	}

	expectedShimConfig := ShimConfig{
		Path: shimPath,
	}

	if reflect.DeepEqual(config, expectedConfig) == false {
		t.Fatalf("Got %v\n expecting %v", config, expectedConfig)
	}

	if reflect.DeepEqual(shimConfig, expectedShimConfig) == false {
		t.Fatalf("Got %v\n expecting %v", shimConfig, expectedShimConfig)
	}

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestMinimalRuntimeConfig(t *testing.T) {
	configPath, err := createConfig("runtime.toml", runtimeMinimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	config, shimConfig, err := loadConfiguration(configPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedHypervisorConfig := vc.HypervisorConfig{
		HypervisorPath: defaultHypervisorPath,
		KernelPath:     defaultKernelPath,
		ImagePath:      defaultImagePath,
	}

	expectedProxyConfig := vc.CCProxyConfig{
		URL: runtimePath,
	}

	expectedConfig := oci.RuntimeConfig{
		HypervisorType:   defaultHypervisor,
		HypervisorConfig: expectedHypervisorConfig,

		AgentType: defaultAgent,

		ProxyType:   defaultProxy,
		ProxyConfig: expectedProxyConfig,
	}

	expectedShimConfig := ShimConfig{
		Path: defaultShimPath,
	}

	if reflect.DeepEqual(config, expectedConfig) == false {
		t.Fatalf("Got %v\n expecting %v", config, expectedConfig)
	}

	if reflect.DeepEqual(shimConfig, expectedShimConfig) == false {
		t.Fatalf("Got %v\n expecting %v", shimConfig, expectedShimConfig)
	}

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	/* Create temp bundle directory if necessary */
	err := os.MkdirAll(tempRuntimePath, 0755)
	if err != nil {
		fmt.Printf("Unable to create %s %v\n", tempRuntimePath, err)
		os.Exit(1)
	}

	defer os.RemoveAll(tempRuntimePath)

	os.Exit(m.Run())
}
