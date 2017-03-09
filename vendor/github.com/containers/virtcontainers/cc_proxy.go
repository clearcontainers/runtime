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
	"net"
	"net/url"

	"github.com/01org/cc-oci-runtime/proxy/api"
)

var defaultCCProxyURL = "unix:///run/cc-oci-runtime/proxy.sock"

type ccProxy struct {
	client *api.Client
}

// CCProxyConfig is a structure storing information needed for
// the Clear Containers proxy initialization.
type CCProxyConfig struct {
	URL string
}

func (p *ccProxy) connectProxy(proxyURL string) (*api.Client, error) {
	if proxyURL == "" {
		proxyURL = defaultCCProxyURL
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "" {
		return nil, fmt.Errorf("URL scheme cannot be empty")
	}

	address := u.Host
	if address == "" {
		if u.Path == "" {
			return nil, fmt.Errorf("URL host and path cannot be empty")
		}

		address = u.Path
	}

	conn, err := net.Dial(u.Scheme, address)
	if err != nil {
		return nil, err
	}

	return api.NewClient(conn.(*net.UnixConn)), nil
}

func (p *ccProxy) allocateProxyInfo() (ProxyInfo, error) {
	if p.client == nil {
		return ProxyInfo{}, fmt.Errorf("allocateIOStream: Client is nil, we can't interact with cc-proxy")
	}

	ioBase, _, err := p.client.AllocateIo(2)
	if err != nil {
		return ProxyInfo{}, err
	}

	proxyInfo := ProxyInfo{
		StdioID:  ioBase,
		StderrID: ioBase + 1,
	}

	return proxyInfo, nil
}

// register is the proxy register implementation for ccProxy.
func (p *ccProxy) register(pod Pod) ([]ProxyInfo, string, error) {
	var err error
	var proxyInfos []ProxyInfo

	ccConfig, ok := newProxyConfig(*(pod.config)).(CCProxyConfig)
	if !ok {
		return []ProxyInfo{}, "", fmt.Errorf("Wrong proxy config type, should be CCProxyConfig type")
	}

	p.client, err = p.connectProxy(ccConfig.URL)
	if err != nil {
		return []ProxyInfo{}, "", err
	}

	hyperConfig, ok := newAgentConfig(*(pod.config)).(HyperConfig)
	if !ok {
		return []ProxyInfo{}, "", fmt.Errorf("Wrong agent config type, should be HyperConfig type")
	}

	_, err = p.client.Hello(pod.id, hyperConfig.SockCtlName, hyperConfig.SockTtyName, nil)
	if err != nil {
		return []ProxyInfo{}, "", err
	}

	// TODO: url will be given by the RegisterVM of the new proxy
	url := ""

	if url == "" {
		url = defaultCCProxyURL
	}

	for i := 0; i < len(pod.containers); i++ {
		proxyInfo, err := p.allocateProxyInfo()
		if err != nil {
			return []ProxyInfo{}, "", err
		}

		proxyInfos = append(proxyInfos, proxyInfo)
	}

	return proxyInfos, url, nil
}

// unregister is the proxy unregister implementation for ccProxy.
func (p *ccProxy) unregister(pod Pod) error {
	if p.client == nil {
		return fmt.Errorf("unregister: Client is nil, we can't interact with cc-proxy")
	}

	return p.client.Bye(pod.id)
}

// connect is the proxy connect implementation for ccProxy.
func (p *ccProxy) connect(pod Pod, createToken bool) (ProxyInfo, string, error) {
	var err error

	ccConfig, ok := newProxyConfig(*(pod.config)).(CCProxyConfig)
	if !ok {
		return ProxyInfo{}, "", fmt.Errorf("Wrong proxy config type, should be CCProxyConfig type")
	}

	p.client, err = p.connectProxy(ccConfig.URL)
	if err != nil {
		return ProxyInfo{}, "", err
	}

	_, err = p.client.Attach(pod.id, nil)
	if err != nil {
		return ProxyInfo{}, "", err
	}

	// TODO: url will be given by the AttachVM of the new proxy
	url := ""

	if url == "" {
		url = defaultCCProxyURL
	}

	// createToken is intended to be true in case we don't want
	// the proxy to create a new token, but instead only get a handle
	// to be able to communicate with the agent inside the VM.
	if createToken == false {
		return ProxyInfo{}, url, nil
	}

	proxyInfo, err := p.allocateProxyInfo()
	if err != nil {
		return ProxyInfo{}, "", err
	}

	return proxyInfo, url, nil
}

// disconnect is the proxy disconnect implementation for ccProxy.
func (p *ccProxy) disconnect() error {
	if p.client == nil {
		return fmt.Errorf("disconnect: Client is nil, we can't interact with cc-proxy")
	}

	p.client.Close()

	return nil
}

// sendCmd is the proxy sendCmd implementation for ccProxy.
func (p *ccProxy) sendCmd(cmd interface{}) (interface{}, error) {
	if p.client == nil {
		return nil, fmt.Errorf("sendCmd: Client is nil, we can't interact with cc-proxy")
	}

	proxyCmd, ok := cmd.(hyperstartProxyCmd)
	if !ok {
		return nil, fmt.Errorf("Wrong command type, should be hyperstartProxyCmd type")
	}

	err := p.client.Hyper(proxyCmd.cmd, proxyCmd.message)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
