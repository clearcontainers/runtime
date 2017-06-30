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
	"os"
	"path/filepath"

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
	qemuLiteHypervisorTableType = "qemu-lite"
	qemuHypervisorTableType     = "qemu"

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
	Path   string
	Kernel string
	Image  string
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

	for _, file := range []string{hypervisor, kernel, image} {
		if !fileExists(file) {
			return vc.HypervisorConfig{},
				fmt.Errorf("File does not exist: %v", file)
		}
	}

	return vc.HypervisorConfig{
		HypervisorPath: hypervisor,
		KernelPath:     kernel,
		ImagePath:      image,
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
			fallthrough
		case qemuLiteHypervisorTableType:
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
		HypervisorPath: defaultHypervisorPath,
		KernelPath:     defaultKernelPath,
		ImagePath:      defaultImagePath,
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

	if configPath == "" {
		configPath = defaultRuntimeConfiguration
	}

	resolved, err := resolvePath(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Make the error clearer than the one returned
			// by EvalSymlinks().
			return "", "", config, fmt.Errorf("Config file %v does not exist", configPath)
		}

		return "", "", config, err
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
