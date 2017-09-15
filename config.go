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
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/BurntSushi/toml"
	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
)

const (
	defaultHypervisor = vc.QemuHypervisor
	defaultProxy      = vc.CCProxyType
	defaultShim       = vc.CCShimType
	defaultAgent      = vc.HyperstartAgent
)

// The TOML configuration file contains a number of sections (or
// tables). The names of these tables are in dotted ("nested table")
// form:
//
//   [<component>.<type>]
//
// The components are hypervisor, proxy, shim and agent. For example,
//
//   [proxy.cc]
//
// The currently supported types are listed below:
const (
	// supported hypervisor component types
	qemuHypervisorTableType = "qemu"

	// supported proxy component types
	ccProxyTableType = "cc"

	// supported shim component types
	ccShimTableType = "cc"

	// supported agent component types
	hyperstartAgentTableType = "hyperstart"
)

var (
	errUnknownHypervisor = errors.New("unknown hypervisor")
	errUnknownAgent      = errors.New("unknown agent")
)

type tomlConfig struct {
	Hypervisor map[string]hypervisor
	Proxy      map[string]proxy
	Shim       map[string]shim
	Agent      map[string]agent
	Runtime    runtime
}

type hypervisor struct {
	Path                  string `toml:"path"`
	Kernel                string `toml:"kernel"`
	Image                 string `toml:"image"`
	KernelParams          string `toml:"kernel_params"`
	MachineType           string `toml:"machine_type"`
	DefaultVCPUs          int32  `toml:"default_vcpus"`
	DefaultMemSz          uint32 `toml:"default_memory"`
	DisableBlockDeviceUse bool   `toml:"disable_block_device_use"`
}

type proxy struct {
	URL string `toml:"url"`
}

type runtime struct {
	GlobalLogPath string `toml:"global_log_path"`
}

type shim struct {
	Path string `toml:"path"`
}

type agent struct {
	PauseRootPath string `toml:"pause_root_path"`
}

func (h hypervisor) path() string {
	if h.Path == "" {
		return defaultHypervisorPath
	}

	return h.Path
}

func (h hypervisor) kernel() string {
	if h.Kernel == "" {
		return defaultKernelPath
	}

	return h.Kernel
}

func (h hypervisor) image() string {
	if h.Image == "" {
		return defaultImagePath
	}

	return h.Image
}

func (h hypervisor) kernelParams() string {
	if h.KernelParams == "" {
		return defaultKernelParams
	}

	return h.KernelParams
}

func (h hypervisor) machineType() string {
	if h.MachineType == "" {
		return defaultMachineType
	}

	return h.MachineType
}

func (h hypervisor) defaultVCPUs() uint32 {
	if h.DefaultVCPUs < 0 {
		return uint32(goruntime.NumCPU())
	}
	if h.DefaultVCPUs == 0 { // or unspecified
		return defaultVCPUCount
	}
	if h.DefaultVCPUs > 255 { // qemu supports max 255
		return 255
	}

	return uint32(h.DefaultVCPUs)
}

func (h hypervisor) defaultMemSz() uint32 {
	if h.DefaultMemSz < 8 {
		return defaultMemSize // MiB
	}

	return h.DefaultMemSz
}

func (p proxy) url() string {
	if p.URL == "" {
		return defaultProxyURL
	}

	return p.URL
}

func (s shim) path() string {
	if s.Path == "" {
		return defaultShimPath
	}

	return s.Path
}

func (a agent) pauseRootPath() string {
	if a.PauseRootPath == "" {
		return defaultPauseRootPath
	}

	return a.PauseRootPath
}

func newQemuHypervisorConfig(h hypervisor) (vc.HypervisorConfig, error) {
	hypervisor := h.path()
	kernel := h.kernel()
	image := h.image()
	kernelParams := h.kernelParams()
	machineType := h.machineType()

	for _, file := range []string{hypervisor, kernel, image} {
		if !fileExists(file) {
			return vc.HypervisorConfig{},
				fmt.Errorf("File does not exist: %v", file)
		}
	}
	return vc.HypervisorConfig{
		HypervisorPath:        hypervisor,
		KernelPath:            kernel,
		ImagePath:             image,
		KernelParams:          vc.DeserializeParams(strings.Fields(kernelParams)),
		HypervisorMachineType: machineType,
		DefaultVCPUs:          h.defaultVCPUs(),
		DefaultMemSz:          h.defaultMemSz(),
		DisableBlockDeviceUse: h.DisableBlockDeviceUse,
	}, nil
}

func newHyperstartAgentConfig(a agent) (vc.HyperConfig, error) {
	dir := a.pauseRootPath()

	if !fileExists(dir) {
		return vc.HyperConfig{}, fmt.Errorf("Directory does not exist: %v", dir)
	}

	path := filepath.Join(dir, pauseBinRelativePath)

	if !fileExists(path) {
		return vc.HyperConfig{}, fmt.Errorf("File does not exist: %v", path)
	}

	return vc.HyperConfig{
		PauseBinPath: path,
	}, nil
}

func newCCShimConfig(s shim) (vc.CCShimConfig, error) {
	path := s.path()

	if !fileExists(path) {
		return vc.CCShimConfig{}, fmt.Errorf("File does not exist: %v", path)
	}

	return vc.CCShimConfig{
		Path: path,
	}, nil
}

func updateRuntimeConfig(configPath string, tomlConf tomlConfig, config *oci.RuntimeConfig) error {
	for k, hypervisor := range tomlConf.Hypervisor {
		switch k {
		case qemuHypervisorTableType:
			hConfig, err := newQemuHypervisorConfig(hypervisor)
			if err != nil {
				return fmt.Errorf("%v: %v", configPath, err)
			}

			config.HypervisorConfig = hConfig

			break
		}
	}

	for k, proxy := range tomlConf.Proxy {
		switch k {
		case ccProxyTableType:
			pConfig := vc.CCProxyConfig{
				URL: proxy.url(),
			}

			config.ProxyType = vc.CCProxyType
			config.ProxyConfig = pConfig

			break
		}
	}

	for k, agent := range tomlConf.Agent {
		switch k {
		case hyperstartAgentTableType:
			agentConfig, err := newHyperstartAgentConfig(agent)
			if err != nil {
				return fmt.Errorf("%v: %v", configPath, err)
			}

			config.AgentConfig = agentConfig

			break
		}
	}

	for k, shim := range tomlConf.Shim {
		switch k {
		case ccShimTableType:
			shConfig, err := newCCShimConfig(shim)
			if err != nil {
				return fmt.Errorf("%v: %v", configPath, err)
			}

			config.ShimType = vc.CCShimType
			config.ShimConfig = shConfig

			break
		}
	}

	return nil
}

// loadConfiguration loads the configuration file and converts it into a
// runtime configuration.
//
// If ignoreLogging is true, the global log will not be initialised nor
// will this function make any log calls.
func loadConfiguration(configPath string, ignoreLogging bool) (resolvedConfigPath, logfilePath string, config oci.RuntimeConfig, err error) {
	defaultHypervisorConfig := vc.HypervisorConfig{
		HypervisorPath:        defaultHypervisorPath,
		KernelPath:            defaultKernelPath,
		ImagePath:             defaultImagePath,
		HypervisorMachineType: defaultMachineType,
		DefaultVCPUs:          defaultVCPUCount,
		DefaultMemSz:          defaultMemSize,
	}

	defaultAgentConfig := vc.HyperConfig{
		PauseBinPath: filepath.Join(defaultPauseRootPath,
			pauseBinRelativePath),
	}

	config = oci.RuntimeConfig{
		HypervisorType:   defaultHypervisor,
		HypervisorConfig: defaultHypervisorConfig,
		AgentType:        defaultAgent,
		AgentConfig:      defaultAgentConfig,
		ProxyType:        defaultProxy,
		ShimType:         defaultShim,
	}

	var resolved string

	if configPath == "" {
		resolved, err = getDefaultConfigFile()
	} else {
		resolved, err = resolvePath(configPath)
	}

	if err != nil {
		return "", "", config, fmt.Errorf("Cannot find usable config file (%v)", err)
	}

	configData, err := ioutil.ReadFile(resolved)
	if err != nil {
		return "", "", config, err
	}

	var tomlConf tomlConfig
	_, err = toml.Decode(string(configData), &tomlConf)
	if err != nil {
		return "", "", config, err
	}

	logfilePath = tomlConf.Runtime.GlobalLogPath

	if !ignoreLogging {
		// The configuration file may have enabled global logging,
		// so handle that before any log calls.
		err = handleGlobalLog(logfilePath)
		if err != nil {
			return "", "", config, err
		}

		ccLog.Debugf("TOML configuration: %v", tomlConf)
	}

	if err := updateRuntimeConfig(resolved, tomlConf, &config); err != nil {
		return "", "", config, err
	}

	return resolved, logfilePath, config, nil
}

// getDefaultConfigFilePaths returns a list of paths that will be
// considered as configuration files in priority order.
func getDefaultConfigFilePaths() []string {
	return []string{
		// normally below "/etc"
		defaultSysConfRuntimeConfiguration,

		// normally below "/usr/share"
		defaultRuntimeConfiguration,
	}
}

// getDefaultConfigFile looks in multiple default locations for a
// configuration file and returns the resolved path for the first file
// found, or an error if no config files can be found.
func getDefaultConfigFile() (string, error) {
	var errs []string

	for _, file := range getDefaultConfigFilePaths() {
		resolved, err := resolvePath(file)
		if err != nil {
			s := fmt.Sprintf("config file %q unresolvable: %v", file, err)
			errs = append(errs, s)
			continue
		}

		return resolved, nil
	}

	return "", errors.New(strings.Join(errs, ", "))
}
