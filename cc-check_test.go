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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

type testModuleData struct {
	path     string
	isDir    bool
	contents string
}

type testCPUData struct {
	vendorID    string
	flags       string
	expectError bool
}

func createFile(file, contents string) error {
	return ioutil.WriteFile(file, []byte(contents), testFileMode)
}

func TestCheckGetCPUInfo(t *testing.T) {
	assert := assert.New(t)

	type testData struct {
		contents       string
		expectedResult string
		expectError    bool
	}

	data := []testData{
		{"", "", true},
		{" ", "", true},
		{"\n", "", true},
		{"\n\n", "", true},
		{"hello\n", "hello", false},
		{"foo\n\n", "foo", false},
		{"foo\n\nbar\n\n", "foo", false},
		{"foo\n\nbar\nbaz\n\n", "foo", false},
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "cpuinfo")
	// file doesn't exist
	_, err = getCPUInfo(file)
	assert.Error(err)

	for _, d := range data {
		err = ioutil.WriteFile(file, []byte(d.contents), testFileMode)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file)

		contents, err := getCPUInfo(file)
		if d.expectError {
			assert.Error(err, fmt.Sprintf("got %q, test data: %+v", contents, d))
		} else {
			assert.NoError(err, fmt.Sprintf("got %q, test data: %+v", contents, d))
		}

		assert.Equal(d.expectedResult, contents)
	}
}

func TestCheckFindAnchoredString(t *testing.T) {
	assert := assert.New(t)

	type testData struct {
		haystack      string
		needle        string
		expectSuccess bool
	}

	data := []testData{
		{"", "", false},
		{"", "foo", false},
		{"foo", "", false},
		{"food", "foo", false},
		{"foo", "foo", true},
		{"foo bar", "foo", true},
		{"foo bar baz", "bar", true},
	}

	for _, d := range data {
		result := findAnchoredString(d.haystack, d.needle)

		if d.expectSuccess {
			assert.True(result)
		} else {
			assert.False(result)
		}
	}
}

func TestCheckGetCPUFlags(t *testing.T) {
	assert := assert.New(t)

	type testData struct {
		cpuinfo       string
		expectedFlags string
	}

	data := []testData{
		{"", ""},
		{"foo", ""},
		{"foo bar", ""},
		{":", ""},
		{"flags", ""},
		{"flags:", ""},
		{"flags: a b c", "a b c"},
		{"flags: a b c foo bar d", "a b c foo bar d"},
	}

	for _, d := range data {
		result := getCPUFlags(d.cpuinfo)
		assert.Equal(d.expectedFlags, result)
	}
}

func TestCheckCheckCPUFlags(t *testing.T) {
	assert := assert.New(t)

	type testData struct {
		cpuflags    string
		required    map[string]string
		expectError bool
	}

	data := []testData{
		{
			"",
			map[string]string{},
			true,
		},
		{
			"",
			map[string]string{
				"a": "A flag",
			},
			true,
		},
		{
			"",
			map[string]string{
				"a": "A flag",
				"b": "B flag",
			},
			true,
		},
		{
			"a b c",
			map[string]string{
				"b": "B flag",
			},
			false,
		},
	}

	for _, d := range data {
		err := checkCPUFlags(d.cpuflags, d.required)
		if d.expectError {
			assert.Error(err)
		} else {
			assert.NoError(err)
		}
	}
}

func TestCheckCheckCPUAttribs(t *testing.T) {
	assert := assert.New(t)

	type testData struct {
		cpuinfo     string
		required    map[string]string
		expectError bool
	}

	data := []testData{
		{
			"",
			map[string]string{},
			true,
		},
		{
			"",
			map[string]string{
				"a": "",
			},
			true,
		},
		{
			"a: b",
			map[string]string{
				"b": "B attribute",
			},
			false,
		},
		{
			"a: b\nc: d\ne: f",
			map[string]string{
				"b": "B attribute",
			},
			false,
		},
		{
			"a: b\n",
			map[string]string{
				"b": "B attribute",
				"c": "C attribute",
				"d": "D attribute",
			},
			true,
		},
		{
			"a: b\nc: d\ne: f",
			map[string]string{
				"b": "B attribute",
				"d": "D attribute",
				"f": "F attribute",
			},
			false,
		},
	}

	for _, d := range data {
		err := checkCPUAttribs(d.cpuinfo, d.required)
		if d.expectError {
			assert.Error(err)
		} else {
			assert.NoError(err)
		}
	}
}

func TestCheckHaveKernelModule(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	savedModInfoCmd := modInfoCmd
	savedSysModuleDir := sysModuleDir

	// XXX: override (fake the modprobe command failing)
	modInfoCmd = "false"
	sysModuleDir = filepath.Join(dir, "sys/module")

	defer func() {
		modInfoCmd = savedModInfoCmd
		sysModuleDir = savedSysModuleDir
	}()

	err = os.MkdirAll(sysModuleDir, testDirMode)
	if err != nil {
		t.Fatal(err)
	}

	module := "foo"

	result := haveKernelModule(module)
	assert.False(result)

	// XXX: override - make our fake "modprobe" succeed
	modInfoCmd = "true"

	result = haveKernelModule(module)
	assert.True(result)

	// disable "modprobe" again
	modInfoCmd = "false"

	fooDir := filepath.Join(sysModuleDir, module)
	err = os.MkdirAll(fooDir, testDirMode)
	if err != nil {
		t.Fatal(err)
	}

	result = haveKernelModule(module)
	assert.True(result)
}

func TestCheckCheckKernelModules(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	savedModInfoCmd := modInfoCmd
	savedSysModuleDir := sysModuleDir

	// XXX: override (fake the modprobe command failing)
	modInfoCmd = "false"
	sysModuleDir = filepath.Join(dir, "sys/module")

	defer func() {
		modInfoCmd = savedModInfoCmd
		sysModuleDir = savedSysModuleDir
	}()

	err = os.MkdirAll(sysModuleDir, testDirMode)
	if err != nil {
		t.Fatal(err)
	}

	testData := map[string]kernelModule{
		"foo": {
			desc:       "desc",
			parameters: map[string]string{},
		},
		"bar": {
			desc: "desc",
			parameters: map[string]string{
				"param1": "hello",
				"param2": "world",
				"param3": "a",
				"param4": ".",
			},
		},
	}

	err = checkKernelModules(map[string]kernelModule{})
	// No required modules means no error
	assert.NoError(err)

	err = checkKernelModules(testData)
	// No modules exist
	assert.Error(err)

	for module, details := range testData {
		path := filepath.Join(sysModuleDir, module)
		err = os.MkdirAll(path, testDirMode)
		if err != nil {
			t.Fatal(err)
		}

		paramDir := filepath.Join(path, "parameters")
		err = os.MkdirAll(paramDir, testDirMode)
		if err != nil {
			t.Fatal(err)
		}

		for param, value := range details.parameters {
			paramPath := filepath.Join(paramDir, param)
			err = createFile(paramPath, value)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	err = checkKernelModules(testData)
	assert.NoError(err)
}

func TestCheckCheckKernelModulesNoUnrestrictedGuest(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	savedSysModuleDir := sysModuleDir
	savedProcCPUInfo := procCPUInfo

	cpuInfoFile := filepath.Join(dir, "cpuinfo")

	// XXX: override
	sysModuleDir = filepath.Join(dir, "sys/module")
	procCPUInfo = cpuInfoFile

	defer func() {
		sysModuleDir = savedSysModuleDir
		procCPUInfo = savedProcCPUInfo
	}()

	err = os.MkdirAll(sysModuleDir, testDirMode)
	if err != nil {
		t.Fatal(err)
	}

	requiredModules := map[string]kernelModule{
		"kvm_intel": {
			desc: "Intel KVM",
			parameters: map[string]string{
				"nested":             "Y",
				"unrestricted_guest": "Y",
			},
		},
	}

	actualModuleData := []testModuleData{
		{filepath.Join(sysModuleDir, "kvm"), true, ""},
		{filepath.Join(sysModuleDir, "kvm_intel"), true, ""},
		{filepath.Join(sysModuleDir, "kvm_intel/parameters/nested"), false, "Y"},

		// XXX: force a failure on non-VMM systems
		{filepath.Join(sysModuleDir, "kvm_intel/parameters/unrestricted_guest"), false, "N"},
	}

	vendor := "GenuineIntel"
	flags := "vmx lm sse4_1"

	err = checkKernelModules(requiredModules)
	// no cpuInfoFile yet
	assert.Error(err)

	err = makeCPUInfoFile(cpuInfoFile, vendor, flags)
	assert.NoError(err)

	createModules(assert, cpuInfoFile, actualModuleData)

	err = checkKernelModules(requiredModules)

	// fails due to unrestricted_guest not being available
	assert.Error(err)
	assert.True(strings.Contains(err.Error(), "unrestricted_guest"))

	// pretend test is running under a hypervisor
	flags += " hypervisor"

	// recreate
	err = makeCPUInfoFile(cpuInfoFile, vendor, flags)
	assert.NoError(err)

	// create buffer to save logger output
	buf := &bytes.Buffer{}

	savedLogOutput := ccLog.Out

	defer func() {
		ccLog.Out = savedLogOutput
	}()

	ccLog.Out = buf

	err = checkKernelModules(requiredModules)

	// no error now because running under a hypervisor
	assert.NoError(err)

	re := regexp.MustCompile(`\bwarning\b.*\bunrestricted_guest\b`)
	matches := re.FindAllStringSubmatch(buf.String(), -1)
	assert.NotEmpty(matches)
}

func TestCheckCheckKernelModulesUnreadableFile(t *testing.T) {
	assert := assert.New(t)

	if os.Geteuid() == 0 {
		t.Skip(testDisabledNeedNonRoot)
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	testData := map[string]kernelModule{
		"foo": {
			desc: "desc",
			parameters: map[string]string{
				"param1": "wibble",
			},
		},
	}

	savedModInfoCmd := modInfoCmd
	savedSysModuleDir := sysModuleDir

	// XXX: override (fake the modprobe command failing)
	modInfoCmd = "false"
	sysModuleDir = filepath.Join(dir, "sys/module")

	defer func() {
		modInfoCmd = savedModInfoCmd
		sysModuleDir = savedSysModuleDir
	}()

	modPath := filepath.Join(sysModuleDir, "foo/parameters")
	err = os.MkdirAll(modPath, testDirMode)
	assert.NoError(err)

	modParamFile := filepath.Join(modPath, "param1")

	err = createEmptyFile(modParamFile)
	assert.NoError(err)

	// make file unreadable by non-root user
	err = os.Chmod(modParamFile, 0000)
	assert.NoError(err)

	err = checkKernelModules(testData)
	assert.Error(err)
}

func TestCheckCheckKernelModulesInvalidFileContents(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	testData := map[string]kernelModule{
		"foo": {
			desc: "desc",
			parameters: map[string]string{
				"param1": "wibble",
			},
		},
	}

	savedModInfoCmd := modInfoCmd
	savedSysModuleDir := sysModuleDir

	// XXX: override (fake the modprobe command failing)
	modInfoCmd = "false"
	sysModuleDir = filepath.Join(dir, "sys/module")

	defer func() {
		modInfoCmd = savedModInfoCmd
		sysModuleDir = savedSysModuleDir
	}()

	modPath := filepath.Join(sysModuleDir, "foo/parameters")
	err = os.MkdirAll(modPath, testDirMode)
	assert.NoError(err)

	modParamFile := filepath.Join(modPath, "param1")

	err = createFile(modParamFile, "burp")
	assert.NoError(err)

	err = checkKernelModules(testData)
	assert.Error(err)
}

func createModules(assert *assert.Assertions, cpuInfoFile string, moduleData []testModuleData) {
	for _, d := range moduleData {
		var dir string

		if d.isDir {
			dir = d.path
		} else {
			dir = path.Dir(d.path)
		}

		err := os.MkdirAll(dir, testDirMode)
		assert.NoError(err)

		if !d.isDir {
			err = createFile(d.path, d.contents)
			assert.NoError(err)
		}

		err = hostIsClearContainersCapable(cpuInfoFile)
		// cpuInfoFile doesn't exist
		assert.Error(err)
	}
}

func setupCheckHostIsClearContainersCapable(assert *assert.Assertions, cpuInfoFile string, cpuData []testCPUData, moduleData []testModuleData) {
	createModules(assert, cpuInfoFile, moduleData)

	// all the modules files have now been created, so deal with the
	// cpuinfo data.
	for _, d := range cpuData {
		err := makeCPUInfoFile(cpuInfoFile, d.vendorID, d.flags)
		assert.NoError(err)

		err = hostIsClearContainersCapable(cpuInfoFile)
		if d.expectError {
			assert.Error(err)
		} else {
			assert.NoError(err)
		}
	}
}

func TestCheckHostIsClearContainersCapable(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	savedSysModuleDir := sysModuleDir
	savedProcCPUInfo := procCPUInfo

	cpuInfoFile := filepath.Join(dir, "cpuinfo")

	// XXX: override
	sysModuleDir = filepath.Join(dir, "sys/module")
	procCPUInfo = cpuInfoFile

	defer func() {
		sysModuleDir = savedSysModuleDir
		procCPUInfo = savedProcCPUInfo
	}()

	err = os.MkdirAll(sysModuleDir, testDirMode)
	if err != nil {
		t.Fatal(err)
	}

	cpuData := []testCPUData{
		{"", "", true},
		{"Intel", "", true},
		{"GenuineIntel", "", true},
		{"GenuineIntel", "lm", true},
		{"GenuineIntel", "lm vmx", true},
		{"GenuineIntel", "lm vmx sse4_1", false},
	}

	moduleData := []testModuleData{
		{filepath.Join(sysModuleDir, "kvm"), true, ""},
		{filepath.Join(sysModuleDir, "kvm_intel"), true, ""},
		{filepath.Join(sysModuleDir, "kvm_intel/parameters/nested"), false, "Y"},
		{filepath.Join(sysModuleDir, "kvm_intel/parameters/unrestricted_guest"), false, "Y"},
	}

	setupCheckHostIsClearContainersCapable(assert, cpuInfoFile, cpuData, moduleData)

	// remove the modules to force a failure
	err = os.RemoveAll(sysModuleDir)
	assert.NoError(err)

	err = hostIsClearContainersCapable(cpuInfoFile)
	assert.Error(err)
}

func TestCCCheckCLIFunction(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	logfile := filepath.Join(dir, "global.log")

	savedSysModuleDir := sysModuleDir
	savedProcCPUInfo := procCPUInfo

	cpuInfoFile := filepath.Join(dir, "cpuinfo")

	// XXX: override
	sysModuleDir = filepath.Join(dir, "sys/module")
	procCPUInfo = cpuInfoFile

	defer func() {
		sysModuleDir = savedSysModuleDir
		procCPUInfo = savedProcCPUInfo
	}()

	err = os.MkdirAll(sysModuleDir, testDirMode)
	if err != nil {
		t.Fatal(err)
	}

	cpuData := []testCPUData{
		{"GenuineIntel", "lm vmx sse4_1", false},
	}

	moduleData := []testModuleData{
		{filepath.Join(sysModuleDir, "kvm_intel/parameters/unrestricted_guest"), false, "Y"},
		{filepath.Join(sysModuleDir, "kvm_intel/parameters/nested"), false, "Y"},
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
	assert.NoError(err)

	savedLogOutput := ccLog.Out

	// discard normal output
	ccLog.Out = devNull

	defer func() {
		ccLog.Out = savedLogOutput
	}()

	assert.False(fileExists(logfile))

	err = handleGlobalLog(logfile)
	assert.NoError(err)

	setupCheckHostIsClearContainersCapable(assert, cpuInfoFile, cpuData, moduleData)

	assert.True(fileExists(logfile))

	app := cli.NewApp()
	ctx := cli.NewContext(app, nil, nil)
	app.Name = "foo"

	fn, ok := checkCLICommand.Action.(func(context *cli.Context) error)
	assert.True(ok)

	err = fn(ctx)
	assert.NoError(err)

	err = grep(successMessage, logfile)
	assert.NoError(err)
}

func TestCCCheckCLIFunctionFail(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	oldProcCPUInfo := procCPUInfo

	// doesn't exist
	procCPUInfo = filepath.Join(dir, "cpuinfo")

	defer func() {
		procCPUInfo = oldProcCPUInfo
	}()

	app := cli.NewApp()
	ctx := cli.NewContext(app, nil, nil)
	app.Name = "foo"

	fn, ok := checkCLICommand.Action.(func(context *cli.Context) error)
	assert.True(ok)

	err = fn(ctx)
	assert.Error(err)
}
