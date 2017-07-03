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
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	vc "github.com/containers/virtcontainers"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli"

	"github.com/containers/virtcontainers/pkg/oci"
	"github.com/stretchr/testify/assert"
)

const testProxyURL = "file:///proxyURL"
const testProxyVersion = "proxy version 0.1"
const testShimVersion = "shim version 0.1"

func makeRuntimeConfig(prefixDir string) (configFile string, config oci.RuntimeConfig, err error) {
	const logPath = "/log/path"
	hypervisorPath := filepath.Join(prefixDir, "hypervisor")
	kernelPath := filepath.Join(prefixDir, "kernel")
	imagePath := filepath.Join(prefixDir, "image")
	shimPath := filepath.Join(prefixDir, "cc-shim")
	proxyPath := filepath.Join(prefixDir, "cc-proxy")

	// override
	defaultProxyPath = proxyPath

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
		pauseBinPath,
	}

	for _, file := range filesToCreate {
		err := createEmptyFile(file)
		if err != nil {
			return "", oci.RuntimeConfig{}, err
		}
	}

	err = createFile(shimPath,
		fmt.Sprintf(`#!/bin/sh
	[ "$1" = "--version" ] && echo "%s"`, testShimVersion))
	if err != nil {
		return "", oci.RuntimeConfig{}, err
	}

	err = os.Chmod(shimPath, testExeFileMode)
	if err != nil {
		return "", oci.RuntimeConfig{}, err
	}

	err = createFile(proxyPath,
		fmt.Sprintf(`#!/bin/sh
	[ "$1" = "--version" ] && echo "%s"`, testProxyVersion))
	if err != nil {
		return "", oci.RuntimeConfig{}, err
	}

	err = os.Chmod(proxyPath, testExeFileMode)
	if err != nil {
		return "", oci.RuntimeConfig{}, err
	}

	runtimeConfig := makeRuntimeConfigFileData(
		"qemu-lite",
		hypervisorPath,
		kernelPath,
		imagePath,
		shimPath,
		agentPauseRoot,
		testProxyURL,
		logPath)

	configFile = path.Join(prefixDir, "runtime.toml")
	err = createConfig(configFile, runtimeConfig)
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
		Version: testProxyVersion,
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
		Version: testShimVersion,
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

	// XXX: This file is *NOT* created by this function on purpose
	// (to ensure the only file checked by the tests is
	// testOSRelease). osReleaseClr handling is tested in
	// utils_test.go.
	testOSReleaseClr := filepath.Join(tmpdir, "os-release-clr")

	testProcVersion := filepath.Join(tmpdir, "proc-version")

	// override
	procVersion = testProcVersion
	osRelease = testOSRelease
	osReleaseClr = testOSReleaseClr
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
	meta := getExpectedMetaInfo()

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
		Meta:       meta,
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

func getExpectedMetaInfo() MetaInfo {
	return MetaInfo{
		Version: formatVersion,
	}
}

func TestCCEnvGetMetaInfo(t *testing.T) {
	expectedMeta := getExpectedMetaInfo()

	meta := getMetaInfo()

	assert.Equal(t, expectedMeta, meta)
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

func TestCCEnvGetHostInfoNoProcCPUInfo(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, err = getExpectedHostDetails(tmpdir)
	assert.NoError(t, err)

	err = os.Remove(procCPUInfo)
	assert.NoError(t, err)

	_, err = getHostInfo()
	assert.Error(t, err)
}

func TestCCEnvGetHostInfoNoOSRelease(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, err = getExpectedHostDetails(tmpdir)
	assert.NoError(t, err)

	err = os.Remove(osRelease)
	assert.NoError(t, err)

	_, err = getHostInfo()
	assert.Error(t, err)
}

func TestCCEnvGetHostInfoNoProcVersion(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, err = getExpectedHostDetails(tmpdir)
	assert.NoError(t, err)

	err = os.Remove(procVersion)
	assert.NoError(t, err)

	_, err = getHostInfo()
	assert.Error(t, err)
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

func TestCCEnvGetEnvInfoNoOSRelease(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	err = os.Remove(osRelease)
	assert.NoError(t, err)

	_, err = getEnvInfo(configFile, logFile, config)
	assert.Error(t, err)
}

func TestCCEnvGetEnvInfoNoProcCPUInfo(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	err = os.Remove(procCPUInfo)
	assert.NoError(t, err)

	_, err = getEnvInfo(configFile, logFile, config)
	assert.Error(t, err)
}

func TestCCEnvGetEnvInfoNoProcVersion(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	err = os.Remove(procVersion)
	assert.NoError(t, err)

	_, err = getEnvInfo(configFile, logFile, config)
	assert.Error(t, err)
}

func TestCCEnvGetEnvInfoNoHypervisor(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expected, err := getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	err = os.Remove(expected.Hypervisor.Resolved)
	assert.NoError(t, err)

	_, err = getEnvInfo(configFile, logFile, config)
	assert.Error(t, err)
}

func TestCCEnvGetEnvInfoNoImage(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expected, err := getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	err = os.Remove(expected.Image.Resolved)
	assert.NoError(t, err)

	_, err = getEnvInfo(configFile, logFile, config)
	assert.Error(t, err)
}

func TestCCEnvGetEnvInfoNoKernel(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expected, err := getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	err = os.Remove(expected.Kernel.Resolved)
	assert.NoError(t, err)

	_, err = getEnvInfo(configFile, logFile, config)
	assert.Error(t, err)
}

func TestCCEnvGetEnvInfoNoShim(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expected, err := getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	err = os.Remove(expected.Shim.Location.Resolved)
	assert.NoError(t, err)

	_, err = getEnvInfo(configFile, logFile, config)
	assert.Error(t, err)
}

func TestCCEnvGetEnvInfoInvalidAgent(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	//err = os.Remove(expected.Shim.Location.Resolved)
	//assert.NoError(t, err)

	cwd, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(cwd)

	agentConfig, ok := config.AgentConfig.(vc.HyperConfig)
	assert.True(t, ok)

	dir := filepath.Dir(agentConfig.PauseBinPath)

	// remove the pause bins parent directory
	err = os.RemoveAll(dir)
	assert.NoError(t, err)

	_, err = getEnvInfo(configFile, logFile, config)
	assert.Error(t, err)
}

func TestCCEnvGetEnvInfoInvalidProxy(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")

	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	// Not directly used, *BUT* must be called as it sets
	// procVersion!
	_, err = getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	configData, err := getFileContents(configFile)
	assert.NoError(t, err)

	// convert to an invalid proxy type
	replacer := strings.NewReplacer(
		`proxy.cc`, `proxy.foo`,
		`url = "`+testProxyURL+`"`, `bar = "wibble"`)
	newConfigData := replacer.Replace(configData)

	err = createFile(configFile, newConfigData)
	assert.NoError(t, err)

	// reload the now invalid config file
	_, _, newConfig, err := loadConfiguration(configFile, true)
	assert.NoError(t, err)

	_, err = getEnvInfo(configFile, logFile, newConfig)
	assert.Error(t, err)
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

	ccRuntime := getRuntimeInfo(configFile, logFile, config)

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

func TestCCEnvGetProxyInfoNoVersion(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expectedProxy, err := getExpectedProxyDetails(config)
	assert.NoError(t, err)

	// remove the proxy ensuring its version cannot be queried
	err = os.Remove(defaultProxyPath)
	assert.NoError(t, err)

	expectedProxy.Version = unknown

	ccProxy, err := getProxyInfo(config)
	assert.NoError(t, err)

	assert.Equal(t, expectedProxy, ccProxy)
}

func TestCCEnvGetProxyInfoInvalidType(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedProxyDetails(config)
	assert.NoError(t, err)

	config.ProxyConfig = "foo"
	_, err = getProxyInfo(config)
	assert.Error(t, err)
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

func TestCCEnvGetShimInfoENOENT(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expectedShim, err := getExpectedShimDetails(config)
	assert.NoError(t, err)

	// remove the shim ensuring its version cannot be queried
	shim := expectedShim.Location.Resolved
	err = os.Remove(shim)
	assert.NoError(t, err)

	_, err = getShimInfo(config)
	assert.Error(t, err)
}

func TestCCEnvGetShimInfoNoVersion(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	expectedShim, err := getExpectedShimDetails(config)
	assert.NoError(t, err)

	shim := expectedShim.Location.Resolved

	// ensure querying the shim version fails
	err = createFile(shim, `#!/bin/sh
	exit 1`)
	assert.NoError(t, err)

	expectedShim.Version = unknown

	ccShim, err := getShimInfo(config)
	assert.NoError(t, err)

	assert.Equal(t, expectedShim, ccShim)
}

func TestCCEnvGetShimInfoInvalidType(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedShimDetails(config)
	assert.NoError(t, err)

	config.ShimConfig = "foo"
	_, err = getShimInfo(config)
	assert.Error(t, err)
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

func TestCCEnvGetAgentInfoInvalidType(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedAgentDetails(config)
	assert.NoError(t, err)

	config.AgentConfig = "foo"
	_, err = getAgentInfo(config)
	assert.Error(t, err)
}

func TestCCEnvGetAgentInfoUnableToResolvePath(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	_, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedAgentDetails(config)
	assert.NoError(t, err)

	cwd, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(cwd)

	agentConfig, ok := config.AgentConfig.(vc.HyperConfig)
	assert.True(t, ok)

	dir := filepath.Dir(agentConfig.PauseBinPath)

	// remove the pause bins parent directory
	err = os.RemoveAll(dir)
	assert.NoError(t, err)

	_, err = getAgentInfo(config)
	assert.Error(t, err)
}

func testCCEnvShowSettings(t *testing.T, tmpdir string, tmpfile *os.File) error {

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

	err = showSettings(ccEnv, tmpfile)
	if err != nil {
		return err
	}

	contents, err := getFileContents(tmpfile.Name())
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	encoder := toml.NewEncoder(buf)
	err = encoder.Encode(ccEnv)
	assert.NoError(t, err)

	expectedContents := buf.String()

	assert.Equal(t, expectedContents, contents)

	return nil
}

func TestCCEnvShowSettings(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	tmpfile, err := ioutil.TempFile("", "ccEnvShowSettings-")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = testCCEnvShowSettings(t, tmpdir, tmpfile)
	assert.NoError(t, err)
}

func TestCCEnvShowSettingsInvalidFile(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	tmpfile, err := ioutil.TempFile("", "ccEnvShowSettings-")
	assert.NoError(t, err)

	// close the file
	tmpfile.Close()

	err = testCCEnvShowSettings(t, tmpdir, tmpfile)
	assert.Error(t, err)
}

func TestCCEnvHandleSettings(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	m := map[string]interface{}{
		"configFile":    configFile,
		"logfilePath":   logFile,
		"runtimeConfig": config,
	}

	tmpfile, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = handleSettings(tmpfile, m)
	assert.NoError(t, err)

	var ccEnv EnvInfo

	_, err = toml.DecodeFile(tmpfile.Name(), &ccEnv)
	assert.NoError(t, err)
}

func TestCCEnvHandleSettingsGetEnvInfoFailure(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	m := map[string]interface{}{
		"configFile":    configFile,
		"logfilePath":   logFile,
		"runtimeConfig": config,
	}

	tmpfile, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = os.Remove(config.HypervisorConfig.HypervisorPath)
	assert.NoError(t, err)

	err = handleSettings(tmpfile, m)
	assert.Error(t, err)
}

func TestCCEnvHandleSettingsInvalidParams(t *testing.T) {
	err := handleSettings(nil, map[string]interface{}{})
	assert.Error(t, err)
}

func TestCCEnvHandleSettingsEmptyMap(t *testing.T) {
	err := handleSettings(os.Stdout, map[string]interface{}{})
	assert.Error(t, err)
}

func TestCCEnvHandleSettingsInvalidFile(t *testing.T) {
	m := map[string]interface{}{
		"configFile":    "foo",
		"logfilePath":   "bar",
		"runtimeConfig": oci.RuntimeConfig{},
	}

	err := handleSettings(nil, m)
	assert.Error(t, err)
}

func TestCCEnvHandleSettingsInvalidConfigFileType(t *testing.T) {
	m := map[string]interface{}{
		"configFile":    123,
		"logfilePath":   "bar",
		"runtimeConfig": oci.RuntimeConfig{},
	}

	err := handleSettings(os.Stderr, m)
	assert.Error(t, err)
}

func TestCCEnvHandleSettingsInvalidLogfileType(t *testing.T) {
	m := map[string]interface{}{
		"configFile":    "/some/where",
		"logfilePath":   42,
		"runtimeConfig": oci.RuntimeConfig{},
	}

	err := handleSettings(os.Stderr, m)
	assert.Error(t, err)
}

func TestCCEnvHandleSettingsInvalidRuntimeConfigType(t *testing.T) {
	m := map[string]interface{}{
		"configFile":    "/some/where",
		"logfilePath":   "some/where/else",
		"runtimeConfig": true,
	}

	err := handleSettings(os.Stderr, m)
	assert.Error(t, err)
}

func TestCCEnvCLIFunction(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	app := cli.NewApp()
	ctx := cli.NewContext(app, nil, nil)
	app.Name = "foo"

	ctx.App.Metadata = map[string]interface{}{
		"configFile":    configFile,
		"logfilePath":   logFile,
		"runtimeConfig": config,
	}

	fn, ok := envCLICommand.Action.(func(context *cli.Context) error)
	assert.True(t, ok)

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
	assert.NoError(t, err)

	// throw away output
	savedOutputFile := defaultOutputFile
	defaultOutputFile = devNull

	defer func() {
		defaultOutputFile = savedOutputFile
	}()

	err = fn(ctx)
	assert.NoError(t, err)
}

func TestCCEnvCLIFunctionFail(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	const logFile = "/tmp/file.log"

	configFile, config, err := makeRuntimeConfig(tmpdir)
	assert.NoError(t, err)

	_, err = getExpectedSettings(config, tmpdir, configFile, logFile)
	assert.NoError(t, err)

	app := cli.NewApp()
	ctx := cli.NewContext(app, nil, nil)
	app.Name = "foo"

	ctx.App.Metadata = map[string]interface{}{
		"configFile":    configFile,
		"logfilePath":   logFile,
		"runtimeConfig": config,
	}

	fn, ok := envCLICommand.Action.(func(context *cli.Context) error)
	assert.True(t, ok)

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
	assert.NoError(t, err)

	// throw away output
	savedOutputFile := defaultOutputFile
	defaultOutputFile = devNull

	defer func() {
		defaultOutputFile = savedOutputFile
	}()

	// cause a failure
	err = os.Remove(config.HypervisorConfig.HypervisorPath)
	assert.NoError(t, err)

	err = fn(ctx)
	assert.Error(t, err)
}
