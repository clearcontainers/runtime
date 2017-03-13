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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var testPID = 100
var testStrPID = fmt.Sprintf("%d", testPID)

func testCreateCgroupsFilesSuccessful(t *testing.T, cgroupsPath string, pid int) {
	if err := createCgroupsFiles(cgroupsPath, pid); err != nil {
		t.Fatalf("This test should succeed (cgroupsPath %q, pid %d): %s", cgroupsPath, pid, err)
	}
}

func TestCgroupsFilesEmptyCgroupsPathSuccessful(t *testing.T) {
	testCreateCgroupsFilesSuccessful(t, "", testPID)
}

func TestCgroupsFilesNonEmptyCgroupsPathSuccessful(t *testing.T) {
	cgroupsPath, err := ioutil.TempDir(testDir, "cgroups-path-")
	if err != nil {
		t.Fatalf("Could not create temporary cgroups directory: %s", err)
	}

	testCreateCgroupsFilesSuccessful(t, cgroupsPath, testPID)

	defer os.RemoveAll(cgroupsPath)

	tasksPath := filepath.Join(cgroupsPath, cgroupsTasksFile)
	procsPath := filepath.Join(cgroupsPath, cgroupsProcsFile)

	for _, path := range []string{tasksPath, procsPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Path %q should have been created: %s", path, err)
		}

		fileBytes, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatalf("Could not read %q previously created: %s", path, err)
		}

		if string(fileBytes) != testStrPID {
			t.Fatalf("PID %s read from %q different from expected PID %s", string(fileBytes), path, testStrPID)
		}
	}
}

func TestCreatePIDFileSuccessful(t *testing.T) {
	pidDirPath, err := ioutil.TempDir(testDir, "pid-path-")
	if err != nil {
		t.Fatalf("Could not create temporary PID directory: %s", err)
	}

	pidFilePath := filepath.Join(pidDirPath, "pid-file-path")
	if err := createPIDFile(pidFilePath, testPID); err != nil {
		t.Fatal(err)
	}

	fileBytes, err := ioutil.ReadFile(pidFilePath)
	if err != nil {
		os.RemoveAll(pidFilePath)
		t.Fatalf("Could not read %q: %s", pidFilePath, err)
	}

	if string(fileBytes) != testStrPID {
		os.RemoveAll(pidFilePath)
		t.Fatalf("PID %s read from %q different from expected PID %s", string(fileBytes), pidFilePath, testStrPID)
	}

	os.RemoveAll(pidFilePath)
}

func TestCreatePIDFileEmptyPathFailure(t *testing.T) {
	file := ""
	if err := createPIDFile(file, testPID); err == nil {
		t.Fatalf("This test should fail (pidFilePath %q, pid %d)", file, testPID)
	}
}
