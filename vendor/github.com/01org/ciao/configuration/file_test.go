//
// Copyright (c) 2016 Intel Corporation
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

package configuration

import (
	"io/ioutil"
	"syscall"
	"testing"

	"github.com/01org/ciao/payloads"
	"github.com/google/gofuzz"
)

// missing the 'configure' identation
const malformedConf = `---
scheduler:
    storage_uri: /etc/ciao/configuration.yaml
controller:
    compute_ca: /etc/pki/ciao/compute_ca.pem
    compute_cert: /etc/pki/ciao/compute_key.pem
    identity_user: controller
    identity_password: ciao
launcher:
    compute_net: 192.168.1.110
    mgmt_net: 192.168.1.111
image_service:
    url: http://glance.example.com
identity_service:
    url: http://keystone.example.com
`

func TestLoadFile(t *testing.T) {
	var fuzzyStr string
	f := fuzz.New()
	f.Fuzz(&fuzzyStr)

	// create temp file where we can read the conf
	tmpf, err := ioutil.TempFile("", "configuration.yaml")
	if err != nil {
		panic(err)
	}
	defer syscall.Unlink(tmpf.Name())
	ioutil.WriteFile(tmpf.Name(), []byte(minValidConf), 0644)

	_, err = loadFile(tmpf.Name())
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	yamlConf, _ := loadFile(fuzzyStr)
	if yamlConf != nil {
		t.Fatalf("expected nil, got %v", yamlConf)
	}

}

func testFileFetchConfiguration(t *testing.T, uri string, drv driver,
	expectedPayload payloads.Configure, positive bool) {
	payload, err := drv.fetchConfiguration(uri)
	if positive == true && err != nil {
		t.Fatalf("%s", err)
	}
	if positive == false && err == nil {
		t.Fatalf("%s", err)
	}

	if positive == true && emptyPayload(expectedPayload) == false {
		if equalPayload(payload, expectedPayload) == false {
			t.Fatalf("expected:%v got %v",
				expectedPayload, payload)
		}
	}
}

func TestFileFetchConfigurationFuzzyPath(t *testing.T) {
	var fuzzyStr string

	f := fuzz.New()
	f.Fuzz(&fuzzyStr)
	// file doesn't exists
	uri := emptyPathURI + "/" + fuzzyStr
	testFileFetchConfiguration(t, uri, &file{}, payloads.Configure{}, false)
}

func TestFileFetchConfigurationBadScheme(t *testing.T) {
	// uri parse error due to  "%z " on string
	uri := "%z" + emptyPathURI
	testFileFetchConfiguration(t, uri, &file{}, payloads.Configure{}, false)
}

func TestFileFetchConfigurationCorrectURI(t *testing.T) {
	var expectedPayload payloads.Configure

	fillPayload(&expectedPayload)

	// create temp file where we can read the conf
	tmpf, err := ioutil.TempFile("", "configuration.yaml")
	if err != nil {
		panic(err)
	}
	defer syscall.Unlink(tmpf.Name())
	ioutil.WriteFile(tmpf.Name(), []byte(fullValidConf), 0644)

	uri := "file://" + tmpf.Name()
	testFileFetchConfiguration(t, uri, &file{}, expectedPayload, true)
}

func TestFileStoreConfiguration(t *testing.T) {
	var d driver
	var conf payloads.Configure

	conf.InitDefaults()
	d = &file{}

	err := d.storeConfiguration(conf)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
