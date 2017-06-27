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

type noopProxy struct{}

var noopProxyURL = "noopProxyURL"

// register is the proxy register implementation for testing purpose.
// It does nothing.
func (p *noopProxy) register(pod Pod) ([]ProxyInfo, string, error) {
	var proxyInfos []ProxyInfo

	for i := 0; i < len(pod.containers); i++ {
		proxyInfo := ProxyInfo{}

		proxyInfos = append(proxyInfos, proxyInfo)
	}

	return proxyInfos, noopProxyURL, nil
}

// unregister is the proxy unregister implementation for testing purpose.
// It does nothing.
func (p *noopProxy) unregister(pod Pod) error {
	return nil
}

// connect is the proxy connect implementation for testing purpose.
// It does nothing.
func (p *noopProxy) connect(pod Pod, createToken bool) (ProxyInfo, string, error) {
	return ProxyInfo{}, noopProxyURL, nil
}

// disconnect is the proxy disconnect implementation for testing purpose.
// It does nothing.
func (p *noopProxy) disconnect() error {
	return nil
}

// sendCmd is the proxy sendCmd implementation for testing purpose.
// It does nothing.
func (p *noopProxy) sendCmd(cmd interface{}) (interface{}, error) {
	return nil, nil
}
