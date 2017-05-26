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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
	"github.com/stretchr/testify/assert"
)

const (
	testFileMode = os.FileMode(0640)
)

type TestFileWriter struct {
	Name string
	File *os.File
}

var testStatuses = []fullContainerState{
	{
		containerState: containerState{
			Version:        "",
			ID:             "1",
			InitProcessPid: 1234,
			Status:         "running",
			Bundle:         "/somewhere/over/the/rainbow",
			Created:        time.Now().UTC(),
			Annotations:    map[string]string(nil),
			Owner:          "",
		},
		hypervisorDetails: hypervisorDetails{
			HypervisorPath: "/hypervisor/path",
			ImagePath:      "/image/path",
			KernelPath:     "/kernel/path",
		},
	},
	{
		containerState: containerState{
			Version:        "",
			ID:             "2",
			InitProcessPid: 2345,
			Status:         "stopped",
			Bundle:         "/this/path/is/invalid",
			Created:        time.Now().UTC(),
			Annotations:    map[string]string(nil),
			Owner:          "",
		},
		hypervisorDetails: hypervisorDetails{
			HypervisorPath: "/hypervisor/path2",
			ImagePath:      "/image/path2",
			KernelPath:     "/kernel/path2",
		},
	},
	{
		containerState: containerState{
			Version:        "",
			ID:             "3",
			InitProcessPid: 9999,
			Status:         "ready",
			Bundle:         "/foo/bar/baz",
			Created:        time.Now().UTC(),
			Annotations:    map[string]string(nil),
			Owner:          "",
		},
		hypervisorDetails: hypervisorDetails{
			HypervisorPath: "/hypervisor/path3",
			ImagePath:      "/image/path3",
			KernelPath:     "/kernel/path3",
		},
	},
}

// Implement the io.Writer interface
func (w *TestFileWriter) Write(bytes []byte) (n int, err error) {
	return w.File.Write(bytes)
}

func NewTestFileWriter(name string) (w *TestFileWriter, err error) {
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND | os.O_SYNC

	file, err := os.OpenFile(name, flags, testFileMode)
	if err != nil {
		return nil, err
	}

	w = &TestFileWriter{
		Name: name,
		File: file,
	}

	return w, nil
}

func TestGetHypervisorDetails(t *testing.T) {
	tmpDir, err := ioutil.TempDir(testDir, "hypervisor-details-")
	if err != nil {
		t.Error(err)
	}

	kernel := path.Join(tmpDir, "kernel")
	image := path.Join(tmpDir, "image")
	hypervisor := path.Join(tmpDir, "image")

	kernelLink := path.Join(tmpDir, "link-to-kernel")
	imageLink := path.Join(tmpDir, "link-to-image")
	hypervisorLink := path.Join(tmpDir, "link-to-hypervisor")

	type testData struct {
		file    string
		symLink string
	}

	for _, d := range []testData{
		{kernel, kernelLink},
		{image, imageLink},
		{hypervisor, hypervisorLink},
	} {
		err = createEmptyFile(d.file)
		if err != nil {
			t.Error(err)
		}

		err = syscall.Symlink(d.file, d.symLink)
		if err != nil {
			t.Error(err)
		}
	}

	hypervisorConfig := vc.HypervisorConfig{
		KernelPath:     kernelLink,
		ImagePath:      imageLink,
		HypervisorPath: hypervisorLink,
	}

	runtimeConfig := oci.RuntimeConfig{
		HypervisorConfig: hypervisorConfig,
	}

	expected := hypervisorDetails{
		KernelPath:     kernel,
		ImagePath:      image,
		HypervisorPath: hypervisor,
	}

	result, err := getHypervisorDetails(runtimeConfig)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, result, expected, "hypervisor configs")

	os.RemoveAll(tmpDir)
}

func formatListDataAsBytes(formatter formatState, state []fullContainerState, showAll bool) (bytes []byte, err error) {
	tmpfile, err := ioutil.TempFile("", "formatListData-")
	if err != nil {
		return nil, err
	}

	defer os.Remove(tmpfile.Name())

	err = formatter.Write(state, showAll, tmpfile)
	if err != nil {
		return nil, err
	}

	tmpfile.Close()

	return ioutil.ReadFile(tmpfile.Name())
}

func formatListDataAsString(formatter formatState, state []fullContainerState, showAll bool) (lines []string, err error) {
	bytes, err := formatListDataAsBytes(formatter, state, showAll)
	if err != nil {
		return nil, err
	}

	lines = strings.Split(string(bytes), "\n")

	// Remove last line if empty
	length := len(lines)
	last := lines[length-1]
	if last == "" {
		lines = lines[:length-1]
	}

	return lines, nil
}

func TestStateToIDList(t *testing.T) {

	// no header
	expectedLength := len(testStatuses)

	// showAll should not affect the output
	for _, showAll := range []bool{true, false} {
		lines, err := formatListDataAsString(&formatIDList{}, testStatuses, showAll)
		if err != nil {
			t.Fatal(err)
		}

		var expected []string
		for _, s := range testStatuses {
			expected = append(expected, s.ID)
		}

		length := len(lines)

		if length != expectedLength {
			t.Fatalf("Expected %d lines, got %d: %v", expectedLength, length, lines)
		}

		assert.Equal(t, lines, expected, "lines + expected")
	}
}

func TestStateToTabular(t *testing.T) {
	// +1 for header line
	expectedLength := len(testStatuses) + 1

	expectedDefaultHeaderPattern := `\AID\s+PID\s+STATUS\s+BUNDLE\s+CREATED\s+OWNER`
	expectedExtendedHeaderPattern := `HYPERVISOR\s+KERNEL\s+IMAGE`
	endingPattern := `\s*\z`

	lines, err := formatListDataAsString(&formatTabular{}, testStatuses, false)
	if err != nil {
		t.Fatal(err)
	}

	length := len(lines)

	expectedHeaderPattern := expectedDefaultHeaderPattern + endingPattern
	expectedHeaderRE := regexp.MustCompile(expectedHeaderPattern)

	if length != expectedLength {
		t.Fatalf("Expected %d lines, got %d", expectedLength, length)
	}

	header := lines[0]

	matches := expectedHeaderRE.FindAllStringSubmatch(header, -1)
	if matches == nil {
		t.Fatalf("Header line failed to match:\n"+
			"pattern : %v\n"+
			"line    : %v\n",
			expectedDefaultHeaderPattern,
			header)
	}

	for i, status := range testStatuses {
		lineIndex := i + 1
		line := lines[lineIndex]

		expectedLinePattern := fmt.Sprintf(`\A%s\s+%d\s+%s\s+%s\s+%s\s+%s\s*\z`,
			regexp.QuoteMeta(status.ID),
			status.InitProcessPid,
			regexp.QuoteMeta(status.Status),
			regexp.QuoteMeta(status.Bundle),
			regexp.QuoteMeta(status.Created.Format(time.RFC3339Nano)),
			regexp.QuoteMeta(status.Owner))

		expectedLineRE := regexp.MustCompile(expectedLinePattern)

		matches := expectedLineRE.FindAllStringSubmatch(line, -1)
		if matches == nil {
			t.Fatalf("Data line failed to match:\n"+
				"pattern : %v\n"+
				"line    : %v\n",
				expectedLinePattern,
				line)
		}
	}

	// Try again with full details this time
	lines, err = formatListDataAsString(&formatTabular{}, testStatuses, true)
	if err != nil {
		t.Fatal(err)
	}

	length = len(lines)

	expectedHeaderPattern = expectedDefaultHeaderPattern + `\s+` + expectedExtendedHeaderPattern + endingPattern
	expectedHeaderRE = regexp.MustCompile(expectedHeaderPattern)

	if length != expectedLength {
		t.Fatalf("Expected %d lines, got %d", expectedLength, length)
	}

	header = lines[0]

	matches = expectedHeaderRE.FindAllStringSubmatch(header, -1)
	if matches == nil {
		t.Fatalf("Header line failed to match:\n"+
			"pattern : %v\n"+
			"line    : %v\n",
			expectedDefaultHeaderPattern,
			header)
	}

	for i, status := range testStatuses {
		lineIndex := i + 1
		line := lines[lineIndex]

		expectedLinePattern := fmt.Sprintf(`\A%s\s+%d\s+%s\s+%s\s+%s\s+%s\s+%s\s+%s\s+%s\s*\z`,
			regexp.QuoteMeta(status.ID),
			status.InitProcessPid,
			regexp.QuoteMeta(status.Status),
			regexp.QuoteMeta(status.Bundle),
			regexp.QuoteMeta(status.Created.Format(time.RFC3339Nano)),
			regexp.QuoteMeta(status.Owner),
			regexp.QuoteMeta(status.hypervisorDetails.HypervisorPath),
			regexp.QuoteMeta(status.hypervisorDetails.KernelPath),
			regexp.QuoteMeta(status.hypervisorDetails.ImagePath))

		expectedLineRE := regexp.MustCompile(expectedLinePattern)

		matches := expectedLineRE.FindAllStringSubmatch(line, -1)
		if matches == nil {
			t.Fatalf("Data line failed to match:\n"+
				"pattern : %v\n"+
				"line    : %v\n",
				expectedLinePattern,
				line)
		}
	}
}

func TestStateToJSON(t *testing.T) {
	expectedLength := len(testStatuses)

	// showAll should not affect the output
	for _, showAll := range []bool{true, false} {
		bytes, err := formatListDataAsBytes(&formatJSON{}, testStatuses, showAll)
		if err != nil {
			t.Fatal(err)
		}

		// Force capacity to match the original otherwise assert.Equal() complains.
		states := make([]fullContainerState, 0, len(testStatuses))

		err = json.Unmarshal(bytes, &states)
		if err != nil {
			t.Fatal(err)
		}

		length := len(states)

		if length != expectedLength {
			t.Fatalf("Expected %d lines, got %d", expectedLength, length)
		}

		// golang tip (what will presumably become v1.9) now
		// stores a monotonic clock value as part of time.Time's
		// internal representation (this is shown by a suffix in
		// the form "m=±ddd.nnnnnnnnn" when calling String() on
		// the time.Time object). However, this monotonic value
		// is stripped out when marshaling.
		//
		// This behaviour change makes comparing the original
		// object and the marshaled-and-then-unmarshaled copy of
		// the object doomed to failure.
		//
		// The solution? Manually strip the monotonic time out
		// of the original before comparison (yuck!)
		//
		// See:
		//
		// - https://go-review.googlesource.com/c/36255/7/src/time/time.go#54
		//
		for i := 0; i < expectedLength; i++ {
			// remove monotonic time part
			testStatuses[i].Created = testStatuses[i].Created.Truncate(0)
		}

		assert.Equal(t, states, testStatuses, "states + testStatuses")
	}
}
