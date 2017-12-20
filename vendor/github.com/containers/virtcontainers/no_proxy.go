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

// This is the no proxy implementation of the proxy interface. This
// is a generic implementation for any case (basically any agent),
// where no actual proxy is needed. This happens when the combination
// of the VM and the agent can handle multiple connections without
// additional component to handle the multiplexing. Both the runtime
// and the shim will connect to the agent through the VM, bypassing
// the proxy model.
// That's why this implementation is very generic, and all it does
// is to provide both shim and runtime the correct URL to connect
// directly to the VM.
type noProxy struct {
	vmURL string
}

// start is noProxy start implementation for proxy interface.
func (p *noProxy) start(pod Pod) (int, string, error) {
	url, err := pod.agent.vmURL()
	if err != nil {
		return -1, "", err
	}

	p.vmURL = url

	if err := pod.agent.setProxyURL(url); err != nil {
		return -1, "", err
	}

	return 0, p.vmURL, nil
}

// register is noProxy register implementation for proxy interface.
func (p *noProxy) register(pod Pod) ([]ProxyInfo, string, error) {
	var proxyInfos []ProxyInfo

	for i := 0; i < len(pod.containers); i++ {
		proxyInfo := ProxyInfo{}

		proxyInfos = append(proxyInfos, proxyInfo)
	}

	return proxyInfos, p.vmURL, nil
}

// unregister is noProxy unregister implementation for proxy interface.
func (p *noProxy) unregister(pod Pod) error {
	return nil
}

// connect is noProxy connect implementation for proxy interface.
func (p *noProxy) connect(pod Pod, createToken bool) (ProxyInfo, string, error) {
	return ProxyInfo{}, p.vmURL, nil
}

// disconnect is noProxy disconnect implementation for proxy interface.
func (p *noProxy) disconnect() error {
	return nil
}

// sendCmd is noProxy sendCmd implementation for proxy interface.
func (p *noProxy) sendCmd(cmd interface{}) (interface{}, error) {
	return nil, nil
}
