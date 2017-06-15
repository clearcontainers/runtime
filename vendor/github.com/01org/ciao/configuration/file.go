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
	"errors"
	"io/ioutil"
	"net/url"

	"github.com/01org/ciao/payloads"
)

type file struct {
	file string
}

// handle the yaml configuration file read
// only 'file' URI Scheme is supported at the moment
func loadFile(path string) (yamlConf []byte, err error) {
	yamlConf, err = ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return yamlConf, nil
}

func (f *file) fetchConfiguration(uriStr string) (conf payloads.Configure, err error) {
	uri, err := url.Parse(uriStr)
	if err != nil {
		return conf, err
	}
	if uri.Path == "" {
		return conf, errors.New("configuration URI path is empty")
	}
	yamlConf, err := loadFile(uri.Path)
	if err != nil {
		return conf, err
	}

	conf, err = Payload(yamlConf)
	if err != nil {
		return conf, err
	}
	// save configuration file URI
	f.file = uriStr
	return conf, nil
}

func (f *file) storeConfiguration(payloads.Configure) error {
	//empty for now
	return nil
}
