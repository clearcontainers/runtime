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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type testData struct {
	path          string
	expectFailure bool
}

func init() {
	// Ensure all log levels are logged
	ccLog.Logger.Level = logrus.DebugLevel

	// Discard "normal" log output: this test only cares about the
	// (additional) global log output
	ccLog.Logger.Out = ioutil.Discard
}

func grep(pattern, file string) error {
	if file == "" {
		return errors.New("need file")
	}

	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(string(bytes), -1)

	if matches == nil {
		return fmt.Errorf("pattern %q not found in file %q", pattern, file)
	}

	return nil
}

func TestNewGlobalLogHook(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	tmpfile := path.Join(tmpdir, "global.log")

	data := []testData{
		{"", true},
		{tmpfile, false},
	}

	for _, d := range data {
		hook, err := newGlobalLogHook(d.path)
		if d.expectFailure {
			if err == nil {
				t.Fatal(fmt.Sprintf("unexpected succes from newGlobalLogHook(path=%v)", d.path))
			}
		} else {
			if err != nil {
				t.Fatal(fmt.Sprintf("unexpected failure from newGlobalLogHook(path=%q): %v", d.path, err))
			}
			if hook.path != d.path {
				t.Fatal(fmt.Sprintf("expected hook to contain path %q, found %q", d.path, hook.path))
			}
		}
	}
}

func TestHandleGlobalLog(t *testing.T) {
	assert := assert.New(t)

	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	subDir := path.Join(tmpdir, "a/b/global.log")
	err = os.MkdirAll(subDir, testDirMode)
	assert.NoError(err)

	existingFile := path.Join(tmpdir, "c")
	err = createEmptyFile(existingFile)
	assert.NoError(err)

	tmpfile := path.Join(tmpdir, "global.log")

	data := []testData{
		{"", false},

		// path must be absolute, so these should fail
		{"foo/bar/global.log", true},
		{"../foo/bar/global.log", true},
		{"./foo/bar/global.log", true},
		{subDir, true},
		{path.Join(existingFile, "global.log"), true},

		{tmpfile, false},
	}

	for _, d := range data {
		// clear any existing hooks
		ccLog.Logger.Hooks = make(map[logrus.Level][]logrus.Hook)

		err := handleGlobalLog(d.path)
		if d.expectFailure {
			if err == nil {
				t.Fatal(fmt.Sprintf("unexpected success from handleGlobalLog(path=%q)", d.path))
			}
			continue
		}

		if err != nil {
			t.Fatal(fmt.Sprintf("unexpected failure from handleGlobalLog(path=%q): %v", d.path, err))
		}

		// It's valid to pass a blank path to handleGlobalLog(),
		// but no point in checking for log entries in that
		// case!
		if d.path == "" {
			continue
		}

		// Add a log entry
		str := "hello. foo bar baz!"
		ccLog.Debug(str)

		// Check that the string was logged
		err = grep(fmt.Sprintf("level=\"debug\".*%s", str), d.path)
		assert.NoError(err)

		// Check expected perms
		st, err := os.Stat(d.path)
		if err != nil {
			t.Fatal(err)
		}

		expectedPerms := "-rw-r-----"
		actualPerms := st.Mode().String()
		if expectedPerms != actualPerms {
			t.Fatal(fmt.Sprintf("logfile %v should have perms %v, but found %v",
				d.path, expectedPerms, actualPerms))
		}
	}
}

func TestHandleGlobalLogEnvVar(t *testing.T) {
	envvar := "CC_RUNTIME_GLOBAL_LOG"
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	tmpfile := path.Join(tmpdir, "global.log")
	tmpfile2 := path.Join(tmpdir, "global-envvar.log")

	os.Setenv(envvar, tmpfile2)
	defer os.Unsetenv(envvar)

	err = handleGlobalLog(tmpfile)
	if err != nil {
		t.Fatal(err)
	}

	str := "foo or moo?"
	ccLog.Debug(str)
	tmpfileExists := fileExists(tmpfile)
	tmpfile2Exists := fileExists(tmpfile2)

	if tmpfileExists == true {
		t.Fatal(fmt.Sprintf("tmpfile %q exists unexpectedly", tmpfile))
	}

	if tmpfile2Exists == false {
		t.Fatal(fmt.Sprintf("tmpfile2 %q does not exist unexpectedly", tmpfile2))
	}

	// Check that the string was logged
	err = grep(fmt.Sprintf("level=\"debug\".*%s", str), tmpfile2)
	assert.NoError(t, err)
}

func TestLoggerFire(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	ccLog = logrus.NewEntry(logrus.New())

	logFile := path.Join(tmpdir, "a/b/global.log")
	err = handleGlobalLog(logFile)
	assert.NoError(t, err)

	entry := &logrus.Entry{
		Logger:  ccLog.Logger,
		Time:    time.Now().UTC(),
		Level:   logrus.DebugLevel,
		Message: "foo",
	}

	err = ccLog.Logger.Hooks.Fire(logrus.DebugLevel, entry)
	assert.NoError(t, err)

	assert.Equal(t, len(ccLog.Logger.Hooks[logrus.DebugLevel]), 1)
	hook, ok := ccLog.Logger.Hooks[logrus.DebugLevel][0].(*GlobalLogHook)
	assert.True(t, ok)

	err = hook.file.Close()
	assert.NoError(t, err)

	err = os.RemoveAll(tmpdir)
	assert.NoError(t, err)

	err = ccLog.Logger.Hooks.Fire(logrus.DebugLevel, entry)
	assert.Error(t, err)
}
