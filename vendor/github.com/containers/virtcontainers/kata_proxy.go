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

package virtcontainers

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// This is the Kata Containers implementation of the proxy interface.
// This is pretty simple since it provides the same interface to both
// runtime and shim as if they were talking directly to the agent.
type kataProxy struct {
	proxyURL string
}

// start is kataProxy start implementation for proxy interface.
func (p *kataProxy) start(pod Pod) (int, string, error) {
	config, err := newProxyConfig(pod.config)
	if err != nil {
		return -1, "", err
	}

	// construct the socket path the proxy instance will use
	socketPath := filepath.Join(runStoragePath, pod.id, "kata_proxy.sock")
	proxyURL := fmt.Sprintf("unix://%s", socketPath)

	vmURL, err := pod.agent.vmURL()
	if err != nil {
		return -1, "", err
	}

	if err := pod.agent.setProxyURL(proxyURL); err != nil {
		return -1, "", err
	}

	p.proxyURL = proxyURL

	args := []string{config.Path, "-listen-socket", proxyURL, "-mux-socket", vmURL}
	if config.Debug {
		args = append(args, "-log", "debug")
	}

	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		return -1, "", err
	}

	return cmd.Process.Pid, p.proxyURL, nil
}

// register is kataProxy register implementation for proxy interface.
func (p *kataProxy) register(pod Pod) ([]ProxyInfo, string, error) {
	var proxyInfos []ProxyInfo

	for i := 0; i < len(pod.containers); i++ {
		proxyInfo := ProxyInfo{}

		proxyInfos = append(proxyInfos, proxyInfo)
	}

	return proxyInfos, p.proxyURL, nil
}

// unregister is kataProxy unregister implementation for proxy interface.
func (p *kataProxy) unregister(pod Pod) error {
	return nil
}

// connect is kataProxy connect implementation for proxy interface.
func (p *kataProxy) connect(pod Pod, createToken bool) (ProxyInfo, string, error) {
	return ProxyInfo{}, p.proxyURL, nil
}

// disconnect is kataProxy disconnect implementation for proxy interface.
func (p *kataProxy) disconnect() error {
	return nil
}

// sendCmd is kataProxy sendCmd implementation for proxy interface.
func (p *kataProxy) sendCmd(cmd interface{}) (interface{}, error) {
	return nil, nil
}
