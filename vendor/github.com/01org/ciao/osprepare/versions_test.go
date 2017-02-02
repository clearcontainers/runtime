//
// Copyright © 2016 Intel Corporation
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

package osprepare

import (
	"testing"
)

func TestGetDocker(t *testing.T) {
	if pathExists("/usr/bin/docker") == false {
		t.Skip("Docker not installed, cannot validate version get")
	}
	if vers := getDockerVersion(nil); vers == "" {
		t.Fatal("Cannot determine docker version")
	}
}

func TestGetQemu(t *testing.T) {
	if pathExists("/usr/bin/qemu-system-x86_64") == false {
		t.Skip("Qemu not installed, cannot validate version get")
	}
	if vers := getQemuVersion(ospNullLogger{}); vers == "" {
		t.Fatal("Cannot determine qemu version")
	}
}

// TestVersionLessThanEqualVersion tests than VersionLessThan returns
// false when given same version to tests. e.g: VersionLessThan("1.11.0", "1.11.0")
// this tests is expected to pass
func TestVersionLessThanEqualVersion(t *testing.T) {
	if res := versionLessThan(MinQemuVersion, MinQemuVersion); res != false {
		t.Fatalf("expected false, got %v\n", res)
	}
}

// TestVersionLessThanGreaterVersion tests than VersionLessThan returns
// false when given greater version. e.g: VersionLessThan("1.11.0", "0.0.1")
// this tests is expected to pass
func TestVersionLessThanGreaterVersion(t *testing.T) {
	if res := versionLessThan(MinQemuVersion, "0.0.1"); res != false {
		t.Fatalf("expected false, got %v\n", res)
	}
}

// TestVersionLessThanLowerVersion tests than VersionLessThan returns
// true when given lower version. e.g: VersionLessThan("0.0.1", "99.9.9")
// this tests is expected to pass
func TestVersionLessThanLowerVersion(t *testing.T) {
	if res := versionLessThan("0.0.1", MinQemuVersion); res != true {
		t.Fatalf("expected true, got %v\n", res)
	}
}
