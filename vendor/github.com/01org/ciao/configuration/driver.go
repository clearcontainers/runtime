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
	"github.com/01org/ciao/payloads"
)

type driver interface {
	// fetchConfiguration should implement a backend
	// which produces a correct payloads.Configure structure
	// from a URI which will be handled by a specific
	// driver, depending on the URI's scheme given.
	// e.g: 'file' scheme is handled by the 'file' driver.
	fetchConfiguration(uri string) (payloads.Configure, error)

	// storeConfiguration should implement a backend
	// responsible to save the new configuration provided
	// in a payloads.Configure structure.
	// e.g: 'file' driver will save new configuration in a
	// YAML file.
	storeConfiguration(payloads.Configure) error
}
