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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	vc "github.com/containers/virtcontainers"
	specs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/containers/virtcontainers/pkg/oci"
	"github.com/stretchr/testify/assert"
)

func makeRuntimeConfig(prefixDir string) (configFile string, config oci.RuntimeConfig, err error) {
	const proxyURL = "file:///proxyURL"
	const logPath = "/log/path"
	hypervisorPath := filepath.Join(prefixDir, "hypervisor")
	kernelPath := filepath.Join(prefixDir, "kernel")
	imagePath := filepath.Join(prefixDir, "image")
	shimPath := filepath.Join(prefixDir, "shim")
	agentPauseRoot := filepath.Join(prefixDir, "agentPauseRoot")
	agentPauseRootBin := filepath.Join(agentPauseRoot, "bin")

	err = os.MkdirAll(agentPauseRootBin, testDirMode)
	if err != nil {
		return "", oci.RuntimeConfig{}, err
	}

	pauseBinPath := path.Join(agentPauseRootBin, "pause")

	filesToCreate := []string{
		hypervisorPath,
		kernelPath,
		imagePath,
		shimPath,
		pauseBinPath,
	}

	for _, file := range filesToCreate {
		err := createEmptyFile(file)
		if err != nil {
			return "", oci.RuntimeConfig{}, err
		}
	}

	runtimeConfig := makeRuntimeConfigFileData(
		hypervisorPath,
		kernelPath,
		imagePath,
		shimPath,
		agentPauseRoot,
		proxyURL,
		logPath)

	configFile, err = createConfig("runtime.toml", runtimeConfig)
	if err != nil {
		return "", oci.RuntimeConfig{}, err
	}

	_, _, config, err = loadConfiguration(configFile, true)
	if err != nil {
		return "", oci.RuntimeConfig{}, err
	}

	return configFile, config, nil
}

func getExpectedProxyDetails(config oci.RuntimeConfig) (ProxyInfo, error) {
	proxyConfig, ok := config.ProxyConfig.(vc.CCProxyConfig)
	if !ok {
		return ProxyInfo{}, fmt.Errorf("failed to get proxy config")
	}

	return ProxyInfo{
		Type:    string(config.ProxyType),
		Version: unknown,
		URL:     proxyConfig.URL,
	}, nil
}

func getExpectedShimDetails(config oci.RuntimeConfig) (ShimInfo, error) {
	shimConfig, ok := config.ShimConfig.(vc.CCShimConfig)
	if !ok {
		return ShimInfo{}, fmt.Errorf("failed to get shim config")
	}

	shimPath := shimConfig.Path

	return ShimInfo{
		Type:    string(config.ShimType),
		Version: unknown,
		Location: PathInfo{
			Path:     shimPath,
			Resolved: shimPath,
		},
	}, nil
}

func getExpectedAgentDetails(config oci.RuntimeConfig) (AgentInfo, error) {
	agentConfig, ok := config.AgentConfig.(vc.HyperConfig)
	if !ok {
		return AgentInfo{}, fmt.Errorf("failed to get agent config")
	}

	agentBinPath := agentConfig.PauseBinPath

	return AgentInfo{
		Type:    string(config.AgentType),
		Version: unknown,
		PauseBin: PathInfo{
			Path:     agentBinPath,
			Resolved: agentBinPath,
		},
	}, nil
}

func getExpectedHostDetails(tmpdir string) (HostInfo, error) {
	type filesToCreate struct {
		file     string
		contents string
	}

	const expectedKernelVersion = "99.1"

	expectedDistro := DistroInfo{
		Name:    "Foo",
		Version: "42",
	}

	expectedCPU := CPUInfo{
		Vendor: "moi",
		Model:  "awesome XI",
	}

	expectedHostDetails := HostInfo{
		Kernel:    expectedKernelVersion,
		Distro:    expectedDistro,
		CPU:       expectedCPU,
		CCCapable: false,
	}

	testProcCPUInfo := filepath.Join(tmpdir, "cpuinfo")
	testOSRelease := filepath.Join(tmpdir, "os-release")
	testProcVersion := filepath.Join(tmpdir, "proc-version")

	// override
	procVersion = testProcVersion
	osRelease = testOSRelease
	procCPUInfo = testProcCPUInfo

	procVersionContents := fmt.Sprintf("Linux version %s a b c",
		expectedKernelVersion)

	osReleaseContents := fmt.Sprintf(`
NAME="%s"
VERSION_ID="%s"
`, expectedDistro.Name, expectedDistro.Version)

	procCPUInfoContents := fmt.Sprintf(`
vendor_id	: %s
model name	: %s
`, expectedCPU.Vendor, expectedCPU.Model)

	data := []filesToCreate{
		{procVersion, procVersionContents},
		{osRelease, osReleaseContents},
		{procCPUInfo, procCPUInfoContents},
	}

	for _, d := range data {
		err := createFile(d.file, d.contents)
		if err != nil {
			return HostInfo{}, err
		}
	}

	return expectedHostDetails, nil
}

func getExpectedHypervisor(config oci.RuntimeConfig) PathInfo {
	return PathInfo{
		Path:     config.HypervisorConfig.HypervisorPath,
		Resolved: config.HypervisorConfig.HypervisorPath,
	}
}

func getExpectedImage(config oci.RuntimeConfig) PathInfo {
	return PathInfo{
		Path:     config.HypervisorConfig.ImagePath,
		Resolved: config.HypervisorConfig.ImagePath,
	}
}

func getExpectedKernel(config oci.RuntimeConfig) PathInfo {
	return PathInfo{
		Path:     config.HypervisorConfig.KernelPath,
		Resolved: config.HypervisorConfig.KernelPath,
	}
}

func getExpectedRuntimeDetails(configFile, logFile string) RuntimeInfo {
	return RuntimeInfo{
		Version: RuntimeVersionInfo{
			Semver: version,
			Commit: commit,
			OCI:    specs.Version,
		},
		Config: RuntimeConfigInfo{
			GlobalLogPath: logFile,
			Location: PathInfo{
				Path:     configFile,
				Resolved: configFile,
			},
		},
	}
}

func getExpectedSettings(config oci.RuntimeConfig, tmpdir, configFile, logFile string) (EnvInfo, error) {
	runtime := getExpectedRuntimeDetails(configFile, logFile)

	proxy, err := getExpectedProxyDetails(config)
	if err != nil {
		return EnvInfo{}, err
	}

	shim, err := getExpectedShimDetails(config)
	if err != nil {
		return EnvInfo{}, err
	}

	agent, err := getExpectedAgentDetails(config)
	if err != nil {
		return EnvInfo{}, err
	}

	host, err := getExpectedHostDetails(tmpdir)
	if err != nil {
		return EnvInfo{}, err
	}

	hypervisor := getExpectedHypervisor(config)
	kernel := getExpectedKernel(config)
	image := getExpectedImage(config)

	ccEnv := EnvInfo{
		Runtime:    runtime,
		Hypervisor: hypervisor,
		Image:      image,
		Kernel:     kernel,
		Proxy:      proxy,
		Shim:       shim,
		Agent:      agent,
		Host:       host,
	}

	return ccEnv, nil
}

func TestCCEnvGetHostInfo(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	expectedHostDetails, err := getExpectedHostDetails(tmpdir)
	assert.NoError(t, err)

	ccHost, err := getHostInfo()
	assert.NoError(t, err)

	assert.Equal(t, expectedHostDetails, ccHost)
}

func TestCCEnvGetEnvInfo(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expectedCCEnv, err := getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	ccEnv, err := getEnvInfo(configFile, logFile, config)
	assert.NoError(t, err)

	assert.Equal(t, expectedCCEnv, ccEnv)
}

func TestCCEnvGetRuntimeInfo(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	const logFile = "/log/file/path/foo.log"

	expectedRuntime := getExpectedRuntimeDetails(configFile, logFile)

	ccRuntime, err := getRuntimeInfo(configFile, logFile, config)
	assert.NoError(t, err)

	assert.Equal(t, expectedRuntime, ccRuntime)
}

func TestCCEnvGetProxyInfo(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expectedProxy, err := getExpectedProxyDetails(config)
	assert.NoError(t, err)

	ccProxy, err := getProxyInfo(config)
	assert.NoError(t, err)

	assert.Equal(t, expectedProxy, ccProxy)
}

func TestCCEnvGetShimInfo(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expectedShim, err := getExpectedShimDetails(config)
	assert.NoError(t, err)

	ccShim, err := getShimInfo(config)
	assert.NoError(t, err)

	assert.Equal(t, expectedShim, ccShim)
}

func TestCCEnvGetAgentInfo(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expectedAgent, err := getExpectedAgentDetails(config)
	assert.NoError(t, err)

	ccAgent, err := getAgentInfo(config)
	assert.NoError(t, err)

	assert.Equal(t, expectedAgent, ccAgent)
}

func TestCCEnvShowSettings(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	ccRuntime := RuntimeInfo{}

	ccHypervisor := PathInfo{
		Path:     "/hypervisor/path",
		Resolved: "/resolved/hypervisor/path",
	}

	ccImage := PathInfo{
		Path:     "/image/path",
		Resolved: "/resolved/image/path",
	}

	ccKernel := PathInfo{
		Path:     "/kernel/path",
		Resolved: "/resolved/kernel/path",
	}

	ccProxy := ProxyInfo{
		Type:    "proxy-type",
		Version: "proxy-version",
		URL:     "file:///proxy-url",
	}

	ccShim := ShimInfo{
		Type:    "shim-type",
		Version: "shim-version",
		Location: PathInfo{
			Path:     "/shim/path",
			Resolved: "/resolved/shim/path",
		},
	}

	ccAgent := AgentInfo{
		Type:    "agent-type",
		Version: "agent-version",
		PauseBin: PathInfo{
			Path:     "/agent/path",
			Resolved: "/resolved/agent/path",
		},
	}

	expectedHostDetails, err := getExpectedHostDetails(tmpdir)
	assert.NoError(t, err)

	ccEnv := EnvInfo{
		Runtime:    ccRuntime,
		Hypervisor: ccHypervisor,
		Image:      ccImage,
		Kernel:     ccKernel,
		Proxy:      ccProxy,
		Shim:       ccShim,
		Agent:      ccAgent,
		Host:       expectedHostDetails,
	}

	tmpfile, err := ioutil.TempFile("", "ccEnvShowSettings-")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = showSettings(ccEnv, tmpfile)
	assert.NoError(t, err)

	contents, err := getFileContents(tmpfile.Name())
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	encoder := toml.NewEncoder(buf)
	err = encoder.Encode(ccEnv)
	assert.NoError(t, err)

	expectedContents := buf.String()

	assert.Equal(t, expectedContents, contents)
}
