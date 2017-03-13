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

package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func testRemoveCgroupsPathSuccessful(t *testing.T, cgroupsPath string) {
	if err := removeCgroupsPath(cgroupsPath); err != nil {
		t.Fatalf("This test should succeed (cgroupsPath = %s): %s", cgroupsPath, err)
	}
}

func TestRemoveCgroupsPathEmptyPathSuccessful(t *testing.T) {
	testRemoveCgroupsPathSuccessful(t, "")
}

func TestRemoveCgroupsPathNonEmptyPathSuccessful(t *testing.T) {
	cgroupsPath, err := ioutil.TempDir(testDir, "cgroups-path-")
	if err != nil {
		t.Fatalf("Could not create temporary cgroups directory: %s", err)
	}

	if err := os.MkdirAll(cgroupsPath, testDirMode); err != nil {
		t.Fatalf("CgroupsPath directory %q could not be created: %s", cgroupsPath, err)
	}

	testRemoveCgroupsPathSuccessful(t, cgroupsPath)

	if _, err := os.Stat(cgroupsPath); err == nil {
		t.Fatalf("CgroupsPath directory %q should have been removed: %s", cgroupsPath, err)
	}
}
