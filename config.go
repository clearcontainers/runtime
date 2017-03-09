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
	"io/ioutil"

	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
)

const (
	defaultRuntimeLib           = "/var/lib/clear-containers"
	defaultRuntimeRun           = "/var/run/clear-containers"
	defaultRuntimeConfiguration = "/etc/clear-containers/configuration.toml"
	defaultHypervisor           = vc.QemuHypervisor
	defaultProxy                = vc.CCProxyType
	defaultAgent                = vc.HyperstartAgent
	defaultKernelPath           = "/usr/share/clear-containers/vmlinux.container"
	defaultImagePath            = "/usr/share/clear-containers/clear-containers.img"
	defaultHypervisorPath       = "/usr/bin/qemu-lite-system-x86_64"
	defaultShimPath             = "/usr/local/libexec/cc-shim"
	defaultProxyURL             = "unix:///run/cc-oci-runtime/proxy.sock"
)

const (
	qemuLite = "qemu-lite"
	qemu     = "qemu"
)

const (
	ccProxy = "cc"
)

const (
	ccShim = "cc"
)

var (
	errUnknownHypervisor  = errors.New("unknown hypervisor")
	errUnknownAgent       = errors.New("unknown agent")
	errTooManyHypervisors = errors.New("too many hypervisor sections")
	errTooManyProxies     = errors.New("too many proxy sections")
	errTooManyShims       = errors.New("too many shim sections")
)

type tomlConfig struct {
	Hypervisor map[string]hypervisor
	Proxy      map[string]proxy
	Shim       map[string]shim
}

type hypervisor struct {
	Path   string
	Kernel string
	Image  string
}

type proxy struct {
	URL string `toml:"url"`
}

type shim struct {
	Path string `toml:"path"`
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

func loadConfiguration(configPath string) (oci.RuntimeConfig, ShimConfig, error) {
	defaultHypervisorConfig := vc.HypervisorConfig{
		HypervisorPath: defaultHypervisorPath,
		KernelPath:     defaultKernelPath,
		ImagePath:      defaultImagePath,
	}

	config := oci.RuntimeConfig{
		HypervisorType:   defaultHypervisor,
		HypervisorConfig: defaultHypervisorConfig,
		AgentType:        defaultAgent,
		ProxyType:        defaultProxy,
	}

	shimConfig := ShimConfig{
		Path: defaultShimPath,
	}

	if configPath == "" {
		configPath = defaultRuntimeConfiguration
	}

	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, shimConfig, err
	}

	var tomlConf tomlConfig
	_, err = toml.Decode(string(configData), &tomlConf)
	if err != nil {
		return config, shimConfig, err
	}

	log.Debugf("TOML configuration: %v", tomlConf)

	if len(tomlConf.Hypervisor) > 1 {
		return config, shimConfig, errTooManyHypervisors
	}

	if len(tomlConf.Proxy) > 1 {
		return config, shimConfig, errTooManyProxies
	}

	if len(tomlConf.Shim) > 1 {
		return config, shimConfig, errTooManyShims
	}

	for k, hypervisor := range tomlConf.Hypervisor {
		switch k {
		case qemu:
			fallthrough
		case qemuLite:
			hConfig := vc.HypervisorConfig{
				HypervisorPath: hypervisor.path(),
				KernelPath:     hypervisor.kernel(),
				ImagePath:      hypervisor.image(),
			}

			config.HypervisorConfig = hConfig
			break
		}
	}

	for k, proxy := range tomlConf.Proxy {
		switch k {
		case ccProxy:
			pConfig := vc.CCProxyConfig{
				URL: proxy.url(),
			}

			config.ProxyType = vc.CCProxyType
			config.ProxyConfig = pConfig
			break
		}
	}

	for k, shim := range tomlConf.Shim {
		switch k {
		case ccShim:
			shimConfig = ShimConfig{
				Path: shim.path(),
			}
			break
		}
	}

	return config, shimConfig, nil
}
