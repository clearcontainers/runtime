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
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
)

// ProxyConfig is a structure storing information needed from any
// proxy in order to be properly initialized.
type ProxyConfig struct {
	Path  string
	Debug bool
}

// ProxyType describes a proxy type.
type ProxyType string

const (
	// NoopProxyType is the noopProxy.
	NoopProxyType ProxyType = "noopProxy"

	// NoProxyType is the noProxy.
	NoProxyType ProxyType = "noProxy"

	// CCProxyType is the ccProxy.
	CCProxyType ProxyType = "ccProxy"

	// KataProxyType is the kataProxy.
	KataProxyType ProxyType = "kataProxy"
)

const (
	// Number of seconds to wait for the proxy to respond to a connection
	// request.
	waitForProxyTimeoutSecs = 5.0
)

func proxyLogger() *logrus.Entry {
	return virtLog.WithField("subsystem", "proxy")
}

// Set sets a proxy type based on the input string.
func (pType *ProxyType) Set(value string) error {
	switch value {
	case "noopProxy":
		*pType = NoopProxyType
		return nil
	case "noProxy":
		*pType = NoProxyType
		return nil
	case "ccProxy":
		*pType = CCProxyType
		return nil
	case "kataProxy":
		*pType = KataProxyType
		return nil
	default:
		return fmt.Errorf("Unknown proxy type %s", value)
	}
}

// String converts a proxy type to a string.
func (pType *ProxyType) String() string {
	switch *pType {
	case NoopProxyType:
		return string(NoopProxyType)
	case NoProxyType:
		return string(NoProxyType)
	case CCProxyType:
		return string(CCProxyType)
	case KataProxyType:
		return string(KataProxyType)
	default:
		return ""
	}
}

// newProxy returns a proxy from a proxy type.
func newProxy(pType ProxyType) (proxy, error) {
	switch pType {
	case NoopProxyType:
		return &noopProxy{}, nil
	case NoProxyType:
		return &noProxy{}, nil
	case CCProxyType:
		return &ccProxy{}, nil
	case KataProxyType:
		return &kataProxy{}, nil
	default:
		return &noopProxy{}, nil
	}
}

// newProxyConfig returns a proxy config from a generic PodConfig handler,
// after it properly checked the configuration was valid.
func newProxyConfig(podConfig *PodConfig) (ProxyConfig, error) {
	if podConfig == nil {
		return ProxyConfig{}, fmt.Errorf("Pod config cannot be nil")
	}

	var config ProxyConfig
	switch podConfig.ProxyType {
	case KataProxyType:
		fallthrough
	case CCProxyType:
		if err := mapstructure.Decode(podConfig.ProxyConfig, &config); err != nil {
			return ProxyConfig{}, err
		}
	}

	if config.Path == "" {
		return ProxyConfig{}, fmt.Errorf("Proxy path cannot be empty")
	}

	return config, nil
}

// ProxyInfo holds the token returned by the proxy.
// Each ProxyInfo relates to a process running inside a container.
type ProxyInfo struct {
	Token string
}

// connectProxyRetry repeatedly tries to connect to the proxy on the specified
// address until a timeout state is reached, when it will fail.
func connectProxyRetry(scheme, address string) (conn net.Conn, err error) {
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

func connectProxy(uri string) (net.Conn, error) {
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

	return connectProxyRetry(u.Scheme, address)
}

// proxy is the virtcontainers proxy interface.
type proxy interface {
	// start launches a proxy instance for the specified pod, returning
	// the PID of the process and the URL used to connect to it.
	start(pod Pod) (int, string, error)

	// register connects and registers the proxy to the given VM.
	// It also returns information related to containers workloads.
	register(pod Pod) ([]ProxyInfo, string, error)

	// unregister unregisters and disconnects the proxy from the given VM.
	unregister(pod Pod) error

	// connect gets the proxy a handle to a previously registered VM.
	// It also returns information related to containers workloads.
	//
	// createToken is intended to be true in case we don't want
	// the proxy to create a new token, but instead only get a handle
	// to be able to communicate with the agent inside the VM.
	connect(pod Pod, createToken bool) (ProxyInfo, string, error)

	// disconnect disconnects from the proxy.
	disconnect() error

	// sendCmd sends a command to the agent inside the VM through the
	// proxy.
	// This function will always be used from a specific agent
	// implementation because a proxy type is always tied to an agent
	// type. That's the reason why it takes an interface as parameter
	// and it returns another interface.
	// Those interfaces allows consumers (agent implementations) of this
	// proxy interface to be able to use specific structures that can only
	// be understood by a specific agent<=>proxy pair.
	sendCmd(cmd interface{}) (interface{}, error)
}
