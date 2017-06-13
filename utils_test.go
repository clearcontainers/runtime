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

	"github.com/stretchr/testify/assert"
)

func TestFileExists(t *testing.T) {
	dir, err := ioutil.TempDir(testDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "foo")

	assert.False(t, fileExists(file),
		fmt.Sprintf("File %q should not exist", file))

	err = createEmptyFile(file)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, fileExists(file),
		fmt.Sprintf("File %q should exist", file))
}

func TestGetFileContents(t *testing.T) {
	type testData struct {
		contents string
	}

	data := []testData{
		{""},
		{" "},
		{"\n"},
		{"\n\n"},
		{"\n\n\n"},
		{"foo"},
		{"foo\nbar"},
		{"processor   : 0\nvendor_id   : GenuineIntel\n"},
	}

	dir, err := ioutil.TempDir(testDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "foo")

	// file doesn't exist
	_, err = getFileContents(file)
	assert.Error(t, err)

	for _, d := range data {
		// create the file
		err = ioutil.WriteFile(file, []byte(d.contents), testFileMode)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file)

		contents, err := getFileContents(file)
		assert.NoError(t, err)
		assert.Equal(t, contents, d.contents)
	}
}
