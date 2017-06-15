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

package libsnnet

import (
	"testing"
)

func TestEqualNetSlice(t *testing.T) {
	netSlice1 := []string{"192.168.0.0/24", "192.168.5.0/24", "192.168.42.0/24"}
	netSlice2 := []string{"192.168.0.0/24", "192.168.5.0/24", "192.168.42.0/24"}

	equalSlices := EqualNetSlice(netSlice1, netSlice2)
	if equalSlices == false {
		t.Fatalf("Expected true, got %v", equalSlices)
	}
}
