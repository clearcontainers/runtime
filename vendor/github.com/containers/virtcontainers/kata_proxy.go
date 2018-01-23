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
	"context"
	"fmt"
	"os/exec"
	"syscall"

	kataclient "github.com/kata-containers/agent/protocols/client"
	"github.com/kata-containers/agent/protocols/grpc"
)

// This is the Kata Containers implementation of the proxy interface.
// This is pretty simple since it provides the same interface to both
// runtime and shim as if they were talking directly to the agent.
type kataProxy struct {
	proxyURL string
	client   *kataclient.AgentClient
}

// start is kataProxy start implementation for proxy interface.
func (p *kataProxy) start(pod Pod) (int, string, error) {
	if pod.agent == nil {
		return -1, "", fmt.Errorf("No agent")
	}

	config, err := newProxyConfig(pod.config)
	if err != nil {
		return -1, "", err
	}

	// construct the socket path the proxy instance will use
	proxyURL, err := defaultAgentURL(&pod, SocketTypeUNIX)
	if err != nil {
		return -1, "", err
	}

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
		args = append(args, "-agent-logs-socket", pod.hypervisor.getPodConsole(pod.id))
	}

	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		return -1, "", err
	}

	return cmd.Process.Pid, p.proxyURL, nil
}

// register is kataProxy register implementation for proxy interface.
func (p *kataProxy) register(pod Pod) ([]ProxyInfo, string, error) {
	client, err := kataclient.NewAgentClient(p.proxyURL)
	if err != nil {
		return []ProxyInfo{}, "", err
	}
	p.client = client

	var proxyInfos []ProxyInfo

	for i := 0; i < len(pod.containers); i++ {
		proxyInfo := ProxyInfo{}

		proxyInfos = append(proxyInfos, proxyInfo)
	}

	if p.proxyURL == "" {
		// construct the socket path the proxy instance will use
		proxyURL, err := defaultAgentURL(&pod, SocketTypeUNIX)
		if err != nil {
			return []ProxyInfo{}, "", err
		}

		p.proxyURL = proxyURL
	}

	return proxyInfos, p.proxyURL, nil
}

// unregister is kataProxy unregister implementation for proxy interface.
func (p *kataProxy) unregister(pod Pod) error {
	// Kill the proxy. This should ideally be dealt with from a stop method.
	return syscall.Kill(pod.state.ProxyPid, syscall.SIGKILL)
}

// connect is kataProxy connect implementation for proxy interface.
func (p *kataProxy) connect(pod Pod, createToken bool) (ProxyInfo, string, error) {
	client, err := kataclient.NewAgentClient(pod.state.URL)
	if err != nil {
		return ProxyInfo{}, "", err
	}

	p.client = client

	if p.proxyURL == "" {
		// construct the socket path the proxy instance will use
		proxyURL, err := defaultAgentURL(&pod, SocketTypeUNIX)
		if err != nil {
			return ProxyInfo{}, "", err
		}

		p.proxyURL = proxyURL
	}

	return ProxyInfo{}, p.proxyURL, nil
}

// disconnect is kataProxy disconnect implementation for proxy interface.
func (p *kataProxy) disconnect() error {
	if p.client == nil {
		return fmt.Errorf("Client is nil, we can't interact with kata-proxy")
	}

	p.client.Close()

	return nil
}

// sendCmd is kataProxy sendCmd implementation for proxy interface.
func (p *kataProxy) sendCmd(cmd interface{}) (interface{}, error) {
	if p.client == nil {
		return nil, fmt.Errorf("Client is nil, we can't interact with kata-proxy")
	}

	switch c := cmd.(type) {
	case *grpc.ExecProcessRequest:
		_, err := p.client.ExecProcess(context.Background(), c)
		return nil, err
	case *grpc.CreateSandboxRequest:
		_, err := p.client.CreateSandbox(context.Background(), c)
		return nil, err
	case *grpc.DestroySandboxRequest:
		_, err := p.client.DestroySandbox(context.Background(), c)
		return nil, err
	case *grpc.CreateContainerRequest:
		_, err := p.client.CreateContainer(context.Background(), c)
		return nil, err
	case *grpc.StartContainerRequest:
		_, err := p.client.StartContainer(context.Background(), c)
		return nil, err
	case *grpc.RemoveContainerRequest:
		_, err := p.client.RemoveContainer(context.Background(), c)
		return nil, err
	case *grpc.SignalProcessRequest:
		_, err := p.client.SignalProcess(context.Background(), c)
		return nil, err
	default:
		return nil, fmt.Errorf("Unknown gRPC type %T", c)
	}
}
