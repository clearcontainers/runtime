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

var testKataProxyURL = "vmURL"

func TestKataProxyRegister(t *testing.T) {
	p := &kataProxy{
		proxyURL: testKataProxyURL,
	}

	_, proxyURL, err := p.register(Pod{})
	if err != nil {
		t.Fatal(err)
	}

	if proxyURL != testKataProxyURL {
		t.Fatalf("Got URL %q, expecting %q", proxyURL, testKataProxyURL)
	}
}

func TestKataProxyUnregister(t *testing.T) {
	p := &kataProxy{}

	if err := p.unregister(Pod{}); err != nil {
		t.Fatal(err)
	}
}

func TestKataProxyConnect(t *testing.T) {
	p := &kataProxy{
		proxyURL: testKataProxyURL,
	}

	_, proxyURL, err := p.connect(Pod{}, false)
	if err != nil {
		t.Fatal(err)
	}

	if proxyURL != testKataProxyURL {
		t.Fatalf("Got URL %q, expecting %q", proxyURL, testKataProxyURL)
	}
}

func TestKataProxyDisconnect(t *testing.T) {
	p := &kataProxy{}

	if err := p.disconnect(); err != nil {
		t.Fatal(err)
	}
}

func TestKataProxySendCmd(t *testing.T) {
	p := &kataProxy{}

	if _, err := p.sendCmd(nil); err != nil {
		t.Fatal(err)
	}
}
