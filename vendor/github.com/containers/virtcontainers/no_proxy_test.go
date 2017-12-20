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
	"testing"
)

var testNoProxyVMURL = "vmURL"

func TestNoProxyStart(t *testing.T) {
	pod := Pod{
		agent: newAgent(NoopAgentType),
	}

	p := &noProxy{}

	pid, vmURL, err := p.start(pod)
	if err != nil {
		t.Fatal(err)
	}

	if vmURL != "" {
		t.Fatalf("Got URL %q, expecting empty URL", vmURL)
	}

	if pid != 0 {
		t.Fatal("Failure since returned PID should be 0")
	}
}

func TestNoProxyRegister(t *testing.T) {
	p := &noProxy{
		vmURL: testNoProxyVMURL,
	}

	_, vmURL, err := p.register(Pod{})
	if err != nil {
		t.Fatal(err)
	}

	if vmURL != testNoProxyVMURL {
		t.Fatalf("Got URL %q, expecting %q", vmURL, testNoProxyVMURL)
	}
}

func TestNoProxyUnregister(t *testing.T) {
	p := &noProxy{}

	if err := p.unregister(Pod{}); err != nil {
		t.Fatal(err)
	}
}

func TestNoProxyConnect(t *testing.T) {
	p := &noProxy{
		vmURL: testNoProxyVMURL,
	}

	_, vmURL, err := p.connect(Pod{}, false)
	if err != nil {
		t.Fatal(err)
	}

	if vmURL != testNoProxyVMURL {
		t.Fatalf("Got URL %q, expecting %q", vmURL, testNoProxyVMURL)
	}
}

func TestNoProxyDisconnect(t *testing.T) {
	p := &noProxy{}

	if err := p.disconnect(); err != nil {
		t.Fatal(err)
	}
}

func TestNoProxySendCmd(t *testing.T) {
	p := &noProxy{}

	if _, err := p.sendCmd(nil); err != nil {
		t.Fatal(err)
	}
}
