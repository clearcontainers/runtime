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

package oci

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	vc "github.com/containers/virtcontainers"
	spec "github.com/opencontainers/runtime-spec/specs-go"
)

var (
	// ErrNoLinux is an error for missing Linux sections in the OCI configuration file.
	ErrNoLinux = errors.New("missing Linux section")
)

// RuntimeConfig aggregates all runtime specific settings
type RuntimeConfig struct {
	VMConfig vc.Resources

	HypervisorType   vc.HypervisorType
	HypervisorConfig vc.HypervisorConfig

	AgentType   vc.AgentType
	AgentConfig interface{}

	ProxyType   vc.ProxyType
	ProxyConfig interface{}
}

func cmdEnvs(spec spec.Spec, envs []vc.EnvVar) []vc.EnvVar {
	for _, env := range spec.Process.Env {
		kv := strings.Split(env, "=")
		if len(kv) < 2 {
			continue
		}

		envs = append(envs,
			vc.EnvVar{
				Var:   kv[0],
				Value: kv[1],
			})
	}

	return envs
}

func newHook(h spec.Hook) vc.Hook {
	timeout := 0
	if h.Timeout != nil {
		timeout = *h.Timeout
	}

	return vc.Hook{
		Path:    h.Path,
		Args:    h.Args,
		Env:     h.Env,
		Timeout: timeout,
	}
}

func containerHooks(spec spec.Spec) vc.Hooks {
	ociHooks := spec.Hooks
	if ociHooks == nil {
		return vc.Hooks{}
	}

	var hooks vc.Hooks

	for _, h := range ociHooks.Prestart {
		hooks.PreStartHooks = append(hooks.PreStartHooks, newHook(h))
	}

	for _, h := range ociHooks.Poststart {
		hooks.PostStartHooks = append(hooks.PostStartHooks, newHook(h))
	}

	for _, h := range ociHooks.Poststop {
		hooks.PostStopHooks = append(hooks.PostStopHooks, newHook(h))
	}

	return hooks
}

func networkConfig(ocispec spec.Spec) (vc.NetworkConfig, error) {
	linux := ocispec.Linux
	if linux == nil {
		return vc.NetworkConfig{}, ErrNoLinux
	}

	var netConf vc.NetworkConfig

	for _, n := range linux.Namespaces {
		if n.Type != spec.NetworkNamespace {
			continue
		}

		netConf.NumInterfaces = 1
		if n.Path != "" {
			netConf.NetNSPath = n.Path
		}
	}

	return netConf, nil
}

// PodConfig converts an OCI compatible runtime configuration file
// to a virtcontainers pod configuration structure.
func PodConfig(runtime RuntimeConfig, bundlePath, cid, console string) (*vc.PodConfig, error) {
	configPath := filepath.Join(bundlePath, "config.json")
	log.Debugf("converting %s", configPath)

	configByte, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var ocispec spec.Spec
	if err = json.Unmarshal(configByte, &ocispec); err != nil {
		return nil, err
	}

	rootfs := filepath.Join(bundlePath, ocispec.Root.Path)
	log.Debugf("container rootfs: %s", rootfs)

	cmd := vc.Cmd{
		Args:    ocispec.Process.Args,
		Envs:    cmdEnvs(ocispec, []vc.EnvVar{}),
		WorkDir: ocispec.Process.Cwd,
		User:    strconv.FormatUint(uint64(ocispec.Process.User.UID), 10),
		Group:   strconv.FormatUint(uint64(ocispec.Process.User.GID), 10),
	}

	containerConfig := vc.ContainerConfig{
		RootFs:      rootfs,
		Interactive: ocispec.Process.Terminal,
		Console:     console,
		Cmd:         cmd,
	}

	networkConfig, err := networkConfig(ocispec)
	if err != nil {
		return nil, err
	}

	podConfig := vc.PodConfig{
		ID: cid,

		Hooks: containerHooks(ocispec),

		VMConfig: runtime.VMConfig,

		HypervisorType:   runtime.HypervisorType,
		HypervisorConfig: runtime.HypervisorConfig,

		AgentType:   runtime.AgentType,
		AgentConfig: runtime.AgentConfig,

		ProxyType:   runtime.ProxyType,
		ProxyConfig: runtime.ProxyConfig,

		NetworkModel:  vc.CNMNetworkModel,
		NetworkConfig: networkConfig,

		Containers: []vc.ContainerConfig{containerConfig},
	}

	return &podConfig, nil
}
