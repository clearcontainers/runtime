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

package virtcontainers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestHyperstartValidateNoSocketsSuccessful(t *testing.T) {
	config := &HyperConfig{
		SockCtlName: "ctlSock",
		SockTtyName: "ttySock",
	}

	pod := Pod{
		id: testPodID,
	}

	ok := config.validate(pod)
	if ok != true {
		t.Fatal()
	}
}

func testHyperstartValidateNSocket(t *testing.T, socketAmount int, expected bool) {
	sockets := make([]Socket, socketAmount)

	config := &HyperConfig{
		SockCtlName: "ctlSock",
		SockTtyName: "ttySock",
		Sockets:     sockets,
	}

	pod := Pod{
		id: testPodID,
	}

	ok := config.validate(pod)
	if ok != expected {
		t.Fatal()
	}
}

func TestHyperstartValidateOneSocketFailing(t *testing.T) {
	testHyperstartValidateNSocket(t, 1, false)

	for i := 3; i < 1000; i++ {
		testHyperstartValidateNSocket(t, i, false)
	}
}

func TestHyperstartValidateNSocketSuccessful(t *testing.T) {
	testHyperstartValidateNSocket(t, 0, true)
	testHyperstartValidateNSocket(t, 2, true)
}

func TestCopyPauseBinarySuccessful(t *testing.T) {
	tmpDirPath, err := ioutil.TempDir("", "test_shared_dir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDirPath)

	defaultSharedDir = tmpDirPath

	srcFile, err := ioutil.TempFile("", "test_src_copy")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcFile.Name())
	defer srcFile.Close()

	h := &hyper{
		config: HyperConfig{
			PauseBinPath: srcFile.Name(),
		},
	}

	dstPath := filepath.Join(defaultSharedDir, testPodID,
		pauseContainerName, rootfsDir, pauseBinName)

	if err := h.copyPauseBinary(testPodID); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcFile.Name())

	if _, err := os.Stat(dstPath); err != nil {
		t.Fatal(err)
	}
}
