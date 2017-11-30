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
	"os/exec"
	"path/filepath"
	"time"

	"github.com/clearcontainers/proxy/client"
	"github.com/sirupsen/logrus"
)

var defaultCCProxyURL = "unix:///var/run/clear-containers/proxy.sock"

const (
	// Number of seconds to wait for the proxy to respond to a connection
	// request.
	waitForProxyTimeoutSecs = 5.0
)

type ccProxy struct {
	client *client.Client
}

// CCProxyConfig is a structure storing information needed for
// the Clear Containers proxy initialization.
type CCProxyConfig struct {
	Path  string
	Debug bool
}

// connectProxyRetry repeatedly tries to connect to the proxy on the specified
// address until a timeout state is reached, when it will fail.
func (p *ccProxy) connectProxyRetry(scheme, address string) (conn net.Conn, err error) {
	attempt := 1

	timeoutSecs := time.Duration(waitForProxyTimeoutSecs * time.Second)

	startTime := time.Now()
	lastLogTime := startTime

	for {
		conn, err = net.Dial(scheme, address)
		if err == nil {
			// If the initial connection was unsuccessful,
			// ensure a log message is generated when successfully
			// connected.
			if attempt > 1 {
				proxyLogger().WithField("attempt", fmt.Sprintf("%d", attempt)).Info("Connected to proxy")
			}

			return conn, nil
		}

		attempt++

		now := time.Now()

		delta := now.Sub(startTime)
		remaining := timeoutSecs - delta

		if remaining <= 0 {
			return nil, fmt.Errorf("failed to connect to proxy after %v: %v", timeoutSecs, err)
		}

		logDelta := now.Sub(lastLogTime)
		logDeltaSecs := logDelta / time.Second

		if logDeltaSecs >= 1 {
			proxyLogger().WithError(err).WithFields(logrus.Fields{
				"attempt":             fmt.Sprintf("%d", attempt),
				"proxy-network":       scheme,
				"proxy-address":       address,
				"remaining-time-secs": fmt.Sprintf("%2.2f", remaining.Seconds()),
			}).Warning("Retrying proxy connection")

			lastLogTime = now
		}

		time.Sleep(time.Duration(100) * time.Millisecond)
	}
}

func (p *ccProxy) connectProxy(uri string) (*client.Client, error) {
	if uri == "" {
		return nil, fmt.Errorf("no proxy URI")
	}

	u, err := url.Parse(uri)
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

	conn, err := p.connectProxyRetry(u.Scheme, address)
	if err != nil {
		return nil, err
	}

	return client.NewClient(conn), nil
}

// start is the proxy start implementation for ccProxy.
func (p *ccProxy) start(pod Pod) (int, string, error) {
	if pod.config == nil {
		return -1, "", fmt.Errorf("Pod config cannot be nil")
	}

	config, ok := newProxyConfig(*(pod.config)).(CCProxyConfig)
	if !ok {
		return -1, "", fmt.Errorf("Wrong proxy config type, should be CCProxyConfig type")
	}

	if config.Path == "" {
		return -1, "", fmt.Errorf("Proxy path cannot be empty")
	}

	// construct the socket path the proxy instance will use
	socketPath := filepath.Join(runStoragePath, pod.id, "proxy.sock")
	uri := fmt.Sprintf("unix://%s", socketPath)

	args := []string{config.Path, "-uri", uri}
	if config.Debug {
		args = append(args, "-log", "debug")
	}

	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		return -1, "", err
	}

	return cmd.Process.Pid, uri, nil
}

// register is the proxy register implementation for ccProxy.
func (p *ccProxy) register(pod Pod) ([]ProxyInfo, string, error) {
	var err error
	var proxyInfos []ProxyInfo

	p.client, err = p.connectProxy(pod.state.URL)
	if err != nil {
		return []ProxyInfo{}, "", err
	}

	hyperConfig, ok := newAgentConfig(*(pod.config)).(HyperConfig)
	if !ok {
		return []ProxyInfo{}, "", fmt.Errorf("Wrong agent config type, should be HyperConfig type")
	}

	registerVMOptions := &client.RegisterVMOptions{
		Console:      pod.hypervisor.getPodConsole(pod.id),
		NumIOStreams: len(pod.containers),
	}

	registerVMReturn, err := p.client.RegisterVM(pod.id, hyperConfig.SockCtlName,
		hyperConfig.SockTtyName, registerVMOptions)
	if err != nil {
		return []ProxyInfo{}, "", err
	}

	url := registerVMReturn.IO.URL
	if url == "" {
		url = defaultCCProxyURL
	}

	if len(registerVMReturn.IO.Tokens) != len(pod.containers) {
		return []ProxyInfo{}, "", fmt.Errorf("%d tokens retrieved out of %d expected",
			len(registerVMReturn.IO.Tokens),
			len(pod.containers))
	}

	for _, token := range registerVMReturn.IO.Tokens {
		proxyInfo := ProxyInfo{
			Token: token,
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

	return p.client.UnregisterVM(pod.id)
}

// connect is the proxy connect implementation for ccProxy.
func (p *ccProxy) connect(pod Pod, createToken bool) (ProxyInfo, string, error) {
	var err error

	p.client, err = p.connectProxy(pod.state.URL)
	if err != nil {
		return ProxyInfo{}, "", err
	}

	// In case we are asked to create a token, this means the caller
	// expects only one token to be generated.
	numTokens := 0
	if createToken {
		numTokens = 1
	}

	attachVMOptions := &client.AttachVMOptions{
		NumIOStreams: numTokens,
	}

	attachVMReturn, err := p.client.AttachVM(pod.id, attachVMOptions)
	if err != nil {
		return ProxyInfo{}, "", err
	}

	url := attachVMReturn.IO.URL
	if url == "" {
		url = defaultCCProxyURL
	}

	if len(attachVMReturn.IO.Tokens) != numTokens {
		return ProxyInfo{}, "", fmt.Errorf("%d tokens retrieved out of %d expected",
			len(attachVMReturn.IO.Tokens), numTokens)
	}

	if !createToken {
		return ProxyInfo{}, url, nil
	}

	proxyInfo := ProxyInfo{
		Token: attachVMReturn.IO.Tokens[0],
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

	var tokens []string
	if proxyCmd.token != "" {
		tokens = append(tokens, proxyCmd.token)
	}

	return p.client.HyperWithTokens(proxyCmd.cmd, tokens, proxyCmd.message)
}
