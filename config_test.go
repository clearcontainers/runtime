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
	goruntime "runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
	"github.com/stretchr/testify/assert"
)

const proxyURL = "foo:///foo/clear-containers/proxy.sock"

type testRuntimeConfig struct {
	RuntimeConfig     oci.RuntimeConfig
	RuntimeConfigFile string
	ConfigPath        string
	ConfigPathLink    string
	LogDir            string
	LogPath           string
}

func makeRuntimeConfigFileData(hypervisor, hypervisorPath, kernelPath, imagePath, kernelParams, machineType, shimPath, agentPauseRootPath, proxyURL, logPath string, disableBlock bool) string {
	return `
	# Clear Containers runtime configuration file

	[hypervisor.` + hypervisor + `]
	path = "` + hypervisorPath + `"
	kernel = "` + kernelPath + `"
	kernel_params = "` + kernelParams + `"
	image = "` + imagePath + `"
	machine_type = "` + machineType + `"
	default_vcpus = ` + strconv.FormatUint(uint64(defaultVCPUCount), 10) + `
	default_memory = ` + strconv.FormatUint(uint64(defaultMemSize), 10) + `
	disable_block_device_use =  ` + strconv.FormatBool(disableBlock) + `

	[proxy.cc]
	url = "` + proxyURL + `"

	[shim.cc]
	path = "` + shimPath + `"

	[agent.hyperstart]
	pause_root_path = "` + agentPauseRootPath + `"

        [runtime]
        global_log_path = "` + logPath + `"
	`
}

func createConfig(configPath string, fileData string) error {

	err := ioutil.WriteFile(configPath, []byte(fileData), testFileMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create config file %s %v\n", configPath, err)
		return err
	}

	return nil
}

// createAllRuntimeConfigFiles creates all files necessary to call
// loadConfiguration().
func createAllRuntimeConfigFiles(dir, hypervisor string) (config testRuntimeConfig, err error) {
	if dir == "" {
		return config, fmt.Errorf("BUG: need directory")
	}

	if hypervisor == "" {
		return config, fmt.Errorf("BUG: need hypervisor")
	}

	hypervisorPath := path.Join(dir, "hypervisor")
	kernelPath := path.Join(dir, "kernel")
	kernelParams := "foo=bar xyz"
	imagePath := path.Join(dir, "image")
	shimPath := path.Join(dir, "shim")
	agentPauseRootPath := path.Join(dir, "agentPauseRoot")
	agentPauseRootBin := path.Join(agentPauseRootPath, "bin")
	pauseBinPath := path.Join(agentPauseRootBin, "pause")
	logDir := path.Join(dir, "logs")
	logPath := path.Join(logDir, "runtime.log")
	machineType := "machineType"
	disableBlockDevice := true

	runtimeConfigFileData := makeRuntimeConfigFileData(hypervisor, hypervisorPath, kernelPath, imagePath, kernelParams, machineType, shimPath, agentPauseRootPath, proxyURL, logPath, disableBlockDevice)

	configPath := path.Join(dir, "runtime.toml")
	err = createConfig(configPath, runtimeConfigFileData)
	if err != nil {
		return config, err
	}

	configPathLink := path.Join(filepath.Dir(configPath), "link-to-configuration.toml")

	// create a link to the config file
	err = syscall.Symlink(configPath, configPathLink)
	if err != nil {
		return config, err
	}

	err = os.MkdirAll(agentPauseRootBin, testDirMode)
	if err != nil {
		return config, err
	}

	files := []string{pauseBinPath, hypervisorPath, kernelPath, imagePath, shimPath}

	for _, file := range files {
		// create the resource
		err = createEmptyFile(file)
		if err != nil {
			return config, err
		}
	}

	hypervisorConfig := vc.HypervisorConfig{
		HypervisorPath:        hypervisorPath,
		KernelPath:            kernelPath,
		ImagePath:             imagePath,
		KernelParams:          vc.DeserializeParams(strings.Fields(kernelParams)),
		HypervisorMachineType: machineType,
		DefaultVCPUs:          defaultVCPUCount,
		DefaultMemSz:          defaultMemSize,
		DisableBlockDeviceUse: disableBlockDevice,
		Mlock: !defaultEnableSwap,
	}

	agentConfig := vc.HyperConfig{
		PauseBinPath: filepath.Join(agentPauseRootPath, pauseBinRelativePath),
	}

	proxyConfig := vc.CCProxyConfig{
		URL: proxyURL,
	}

	shimConfig := vc.CCShimConfig{
		Path: shimPath,
	}

	runtimeConfig := oci.RuntimeConfig{
		HypervisorType:   defaultHypervisor,
		HypervisorConfig: hypervisorConfig,

		AgentType:   defaultAgent,
		AgentConfig: agentConfig,

		ProxyType:   defaultProxy,
		ProxyConfig: proxyConfig,

		ShimType:   defaultShim,
		ShimConfig: shimConfig,
	}

	config = testRuntimeConfig{
		RuntimeConfig:     runtimeConfig,
		RuntimeConfigFile: configPath,
		ConfigPath:        configPath,
		ConfigPathLink:    configPathLink,
		LogDir:            logDir,
		LogPath:           logPath,
	}

	return config, nil
}

// testLoadConfiguration accepts an optional function that can be used
// to modify the test: if a function is specified, it indicates if the
// subsequent call to loadConfiguration() is expected to fail by
// returning a bool. If the function itself fails, that is considered an
// error.
func testLoadConfiguration(t *testing.T, dir string,
	fn func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error)) {
	subDir := path.Join(dir, "test")

	for _, hypervisor := range []string{"qemu"} {
	Loop:
		for _, ignoreLogging := range []bool{true, false} {
			err := os.RemoveAll(subDir)
			assert.NoError(t, err)

			err = os.MkdirAll(subDir, testDirMode)
			assert.NoError(t, err)

			testConfig, err := createAllRuntimeConfigFiles(subDir, hypervisor)
			assert.NoError(t, err)

			configFiles := []string{testConfig.ConfigPath, testConfig.ConfigPathLink, ""}

			// override
			defaultRuntimeConfiguration = testConfig.ConfigPath
			defaultSysConfRuntimeConfiguration = ""

			for _, file := range configFiles {
				var err error
				expectFail := false

				if fn != nil {
					expectFail, err = fn(testConfig, file, ignoreLogging)
					assert.NoError(t, err)
				}

				resolvedConfigPath, logfilePath, config, err := loadConfiguration(file, ignoreLogging)
				if expectFail {
					assert.Error(t, err)

					// no point proceeding in the error scenario.
					break Loop
				} else {
					assert.NoError(t, err)
				}

				if file == "" {
					assert.Equal(t, defaultRuntimeConfiguration, resolvedConfigPath)
				} else {
					assert.Equal(t, testConfig.ConfigPath, resolvedConfigPath)
				}

				assert.NotEqual(t, "", logfilePath)
				assert.Equal(t, logfilePath, testConfig.LogPath)

				if ignoreLogging {
					assert.False(t, fileExists(testConfig.LogDir))
					assert.False(t, fileExists(testConfig.LogPath))
				} else {
					assert.True(t, fileExists(testConfig.LogDir))
					assert.True(t, fileExists(testConfig.LogPath))
				}

				assert.Equal(t, defaultRuntimeConfiguration, resolvedConfigPath)
				result := reflect.DeepEqual(config, testConfig.RuntimeConfig)
				if !result {
					t.Fatalf("Expected\n%+v\nGot\n%+v", config, testConfig.RuntimeConfig)
				}
				assert.True(t, result)

				err = os.RemoveAll(testConfig.LogDir)
				assert.NoError(t, err)
			}
		}
	}
}

func TestConfigLoadConfiguration(t *testing.T) {
	tmpdir, err := ioutil.TempDir(testDir, "load-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir, nil)
}

func TestConfigLoadConfigurationFailBrokenSymLink(t *testing.T) {
	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := false

			if configFile == config.ConfigPathLink {
				// break the symbolic link
				err = os.Remove(config.ConfigPathLink)
				if err != nil {
					return expectFail, err
				}

				expectFail = true
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailSymLinkLoop(t *testing.T) {
	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := false

			if configFile == config.ConfigPathLink {
				// remove the config file
				err = os.Remove(config.ConfigPath)
				if err != nil {
					return expectFail, err
				}

				// now, create a sym-link loop
				err := os.Symlink(config.ConfigPathLink, config.ConfigPath)
				if err != nil {
					return expectFail, err
				}

				expectFail = true
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailMissingPauseBinary(t *testing.T) {
	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := true

			hyperConfig, ok := config.RuntimeConfig.AgentConfig.(vc.HyperConfig)
			if !ok {
				return expectFail, fmt.Errorf("cannot determine agent config")
			}

			err = os.Remove(hyperConfig.PauseBinPath)
			if err != nil {
				return expectFail, err
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailMissingPauseDir(t *testing.T) {
	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := true

			hyperConfig, ok := config.RuntimeConfig.AgentConfig.(vc.HyperConfig)
			if !ok {
				return expectFail, fmt.Errorf("cannot determine agent config")
			}

			err = os.RemoveAll(filepath.Dir(hyperConfig.PauseBinPath))
			if err != nil {
				return expectFail, err
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailMissingHypervisor(t *testing.T) {
	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := true

			err = os.Remove(config.RuntimeConfig.HypervisorConfig.HypervisorPath)
			if err != nil {
				return expectFail, err
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailMissingImage(t *testing.T) {
	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := true

			err = os.Remove(config.RuntimeConfig.HypervisorConfig.ImagePath)
			if err != nil {
				return expectFail, err
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailMissingKernel(t *testing.T) {
	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := true

			err = os.Remove(config.RuntimeConfig.HypervisorConfig.KernelPath)
			if err != nil {
				return expectFail, err
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailMissingShim(t *testing.T) {
	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := true

			shimConfig, ok := config.RuntimeConfig.ShimConfig.(vc.CCShimConfig)
			if !ok {
				return expectFail, fmt.Errorf("cannot determine shim config")
			}
			err = os.Remove(shimConfig.Path)
			if err != nil {
				return expectFail, err
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailUnreadableConfig(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip(testDisabledNeedNonRoot)
	}

	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := true

			// make file unreadable by non-root user
			err = os.Chmod(config.ConfigPath, 0000)
			if err != nil {
				return expectFail, err
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailInvalidLogPath(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip(testDisabledNeedNonRoot)
	}

	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			if ignoreLogging {
				return false, nil
			}

			expectFail := true

			err := os.RemoveAll(config.LogDir)
			if err != nil {
				return expectFail, err
			}

			parentDir := filepath.Dir(config.LogDir)

			err = os.MkdirAll(parentDir, testDirMode)
			if err != nil {
				return expectFail, err
			}

			err = createEmptyFile(config.LogDir)
			if err != nil {
				return expectFail, err
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailTOMLConfigFileInvalidContents(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip(testDisabledNeedNonRoot)
	}

	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := true

			err := createFile(config.ConfigPath,
				`<?xml version="1.0"?>
			<foo>I am not TOML! ;-)</foo>
			<bar>I am invalid XML!`)

			if err != nil {
				return expectFail, err
			}

			return expectFail, nil
		})
}

func TestConfigLoadConfigurationFailTOMLConfigFileDuplicatedData(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip(testDisabledNeedNonRoot)
	}

	tmpdir, err := ioutil.TempDir(testDir, "runtime-config-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	testLoadConfiguration(t, tmpdir,
		func(config testRuntimeConfig, configFile string, ignoreLogging bool) (bool, error) {
			expectFail := true

			text, err := getFileContents(config.ConfigPath)
			if err != nil {
				return expectFail, err
			}

			// create a config file containing two sets of
			// data.
			err = createFile(config.ConfigPath, fmt.Sprintf("%s\n%s\n", text, text))
			if err != nil {
				return expectFail, err
			}

			return expectFail, nil
		})
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

	configPath := path.Join(dir, "runtime.toml")
	err = createConfig(configPath, runtimeMinimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, _, config, err := loadConfiguration(configPath, false)
	if err == nil {
		t.Fatalf("Expected loadConfiguration to fail as shim path does not exist: %+v", config)
	}

	err = createEmptyFile(shimPath)
	if err != nil {
		t.Error(err)
	}

	_, _, config, err = loadConfiguration(configPath, false)
	if err != nil {
		t.Fatal(err)
	}

	expectedHypervisorConfig := vc.HypervisorConfig{
		HypervisorPath:        defaultHypervisorPath,
		KernelPath:            defaultKernelPath,
		ImagePath:             defaultImagePath,
		HypervisorMachineType: defaultMachineType,
		DefaultVCPUs:          defaultVCPUCount,
		DefaultMemSz:          defaultMemSize,
		DisableBlockDeviceUse: defaultDisableBlockDeviceUse,
		Mlock: !defaultEnableSwap,
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
	machineType := "machineType"
	disableBlock := true

	hypervisor := hypervisor{
		Path:                  hypervisorPath,
		Kernel:                kernelPath,
		Image:                 imagePath,
		MachineType:           machineType,
		DisableBlockDeviceUse: disableBlock,
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

	if config.DisableBlockDeviceUse != disableBlock {
		t.Errorf("Expected value for disable block usage %v, got %v", disableBlock, config.DisableBlockDeviceUse)
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

func TestHypervisorDefaults(t *testing.T) {
	h := hypervisor{}

	assert.Equal(t, h.path(), defaultHypervisorPath, "default hypervisor path wrong")
	assert.Equal(t, h.kernel(), defaultKernelPath, "default hypervisor kernel wrong")
	assert.Equal(t, h.image(), defaultImagePath, "default hypervisor image wrong")
	assert.Equal(t, h.kernelParams(), defaultKernelParams, "default hypervisor image wrong")
	assert.Equal(t, h.machineType(), defaultMachineType, "default hypervisor machine type wrong")
	assert.Equal(t, h.defaultVCPUs(), defaultVCPUCount, "default vCPU number is wrong")
	assert.Equal(t, h.defaultMemSz(), defaultMemSize, "default memory size is wrong")

	path := "/foo"
	h.Path = path
	assert.Equal(t, h.path(), path, "custom hypervisor path wrong")

	kernel := "wibble"
	h.Kernel = kernel
	assert.Equal(t, h.kernel(), kernel, "custom hypervisor kernel wrong")

	kernelParams := "foo=bar xyz"
	h.KernelParams = kernelParams
	assert.Equal(t, h.kernelParams(), kernelParams, "custom hypervisor kernel parameterms wrong")

	image := "foo"
	h.Image = image
	assert.Equal(t, h.image(), image, "custom hypervisor image wrong")

	machineType := "foo"
	h.MachineType = machineType
	assert.Equal(t, h.machineType(), machineType, "custom hypervisor machine type wrong")

	// auto inferring
	h.DefaultVCPUs = -1
	assert.Equal(t, h.defaultVCPUs(), uint32(goruntime.NumCPU()), "default vCPU number is wrong")

	h.DefaultVCPUs = 2
	assert.Equal(t, h.defaultVCPUs(), uint32(2), "default vCPU number is wrong")

	// qemu supports max 255
	h.DefaultVCPUs = 8086
	assert.Equal(t, h.defaultVCPUs(), uint32(255), "default vCPU number is wrong")

	h.DefaultMemSz = 1024
	assert.Equal(t, h.defaultMemSz(), uint32(1024), "default memory size is wrong")
}

func TestProxyDefaults(t *testing.T) {
	p := proxy{}

	assert.Equal(t, p.url(), defaultProxyURL, "default proxy url wrong")

	url := "unix:///hello/world.sock"
	p.URL = url
	assert.Equal(t, p.url(), url, "custom proxy url wrong")

}

func TestShimDefaults(t *testing.T) {
	s := shim{}

	assert.Equal(t, s.path(), defaultShimPath, "default shim path wrong")

	path := "/foo/bar"
	s.Path = path
	assert.Equal(t, s.path(), path, "custom shim path wrong")
}

func TestAgentDefaults(t *testing.T) {
	a := agent{}

	assert.Equal(t, a.pauseRootPath(), defaultPauseRootPath, "default agent pause root path wrong")

	path := "/foo/bar/baz"
	a.PauseRootPath = path
	assert.Equal(t, a.pauseRootPath(), path, "custom agent pause root path wrong")
}

func TestGetDefaultConfigFilePaths(t *testing.T) {
	assert := assert.New(t)

	results := getDefaultConfigFilePaths()
	// There should be atleast two config file locations
	assert.True(len(results) >= 2)

	for _, f := range results {
		// Paths cannot be empty
		assert.NotNil(f)
	}
}

func TestGetDefaultConfigFile(t *testing.T) {
	assert := assert.New(t)

	tmpdir, err := ioutil.TempDir(testDir, "")
	assert.NoError(err)
	defer os.RemoveAll(tmpdir)

	hypervisor := "qemu"
	confDir := filepath.Join(tmpdir, "conf")
	sysConfDir := filepath.Join(tmpdir, "sysconf")

	for _, dir := range []string{confDir, sysConfDir} {
		err = os.MkdirAll(dir, testDirMode)
		assert.NoError(err)
	}

	confDirConfig, err := createAllRuntimeConfigFiles(confDir, hypervisor)
	assert.NoError(err)

	sysConfDirConfig, err := createAllRuntimeConfigFiles(sysConfDir, hypervisor)
	assert.NoError(err)

	savedConf := defaultRuntimeConfiguration
	savedSysConf := defaultSysConfRuntimeConfiguration

	defaultRuntimeConfiguration = confDirConfig.ConfigPath
	defaultSysConfRuntimeConfiguration = sysConfDirConfig.ConfigPath

	defer func() {
		defaultRuntimeConfiguration = savedConf
		defaultSysConfRuntimeConfiguration = savedSysConf

	}()

	got, err := getDefaultConfigFile()
	assert.NoError(err)
	// defaultSysConfRuntimeConfiguration has priority over defaultRuntimeConfiguration
	assert.Equal(got, defaultSysConfRuntimeConfiguration)

	// force defaultRuntimeConfiguration to be returned
	os.Remove(defaultSysConfRuntimeConfiguration)

	got, err = getDefaultConfigFile()
	assert.NoError(err)
	assert.Equal(got, defaultRuntimeConfiguration)

	// force error
	os.Remove(defaultRuntimeConfiguration)

	_, err = getDefaultConfigFile()
	assert.Error(err)
}
