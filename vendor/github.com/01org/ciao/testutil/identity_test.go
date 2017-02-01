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

package testutil_test

import (
	"testing"

	. "github.com/01org/ciao/testutil"
)

func TestStartIdentityServer(t *testing.T) {
	config := IdentityConfig{
		ComputeURL: "https://localhost:8888",
		ProjectID:  "30dedd5c-48d9-45d3-8b44-f973e4f35e48",
	}

	id := StartIdentityServer(config)
	if id == nil {
		t.Fatal("Could not start test identity server")
	}
	defer id.Close()
}
