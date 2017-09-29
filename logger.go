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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	// globalLogEnv is the name of the environment variable that
	// specifies the full path to a global log.
	//
	// This variable takes priority over global_log_path in the
	// config file.
	globalLogEnv = "CC_RUNTIME_GLOBAL_LOG"

	// globalLogMode is the mode used to create the global log file.
	globalLogMode = os.FileMode(0640)

	// globalLogDirMode is the mode used to create the directory
	// to hold the global log.
	globalLogDirMode = os.FileMode(0750)

	// globalLogFlags are the flags used to open the global log
	// file.
	globalLogFlags = (os.O_CREATE | os.O_WRONLY | os.O_APPEND | os.O_SYNC)
)

var (
	errNeedGlobalLogPath = errors.New("Global log path cannot be empty")
)

// GlobalLogHook represents a "global logfile" that is appended to by all
// runtimes.
//
// A global log is useful since although container managers
// such as docker pass a path for the runtime to use for logging, if the
// runtime fails, the directory used for logging may be deleted. A
// global log can be specified in a location distinct from such
// container-specific paths to provide a persistent log of all runtime
// activity, including debugging failures.
type GlobalLogHook struct {
	path string
	file *os.File
}

// handleGlobalLog sets up the global logger.
//
// Note that the logfile path may be blank since this function also
// checks the environment to see whether global logging is required.
func handleGlobalLog(logfilePath string) error {

	// the environment variable takes priority
	path := os.Getenv(globalLogEnv)

	if path == "" {
		path = logfilePath
	}

	if path == "" {
		// global logging not required
		return nil
	}

	if strings.HasPrefix(path, "/") == false {
		return fmt.Errorf("Global log path must be absolute: %v", path)
	}

	dir := filepath.Dir(logfilePath)

	err := os.MkdirAll(dir, globalLogDirMode)
	if err != nil {
		return err
	}

	hook, err := newGlobalLogHook(path)
	if err != nil {
		return err
	}

	ccLog.Logger.Hooks.Add(hook)

	return nil
}

// newGlobalLogHook creates a new hook that can be used by a logrus
// logger.
func newGlobalLogHook(logfilePath string) (*GlobalLogHook, error) {
	if logfilePath == "" {
		return nil, errNeedGlobalLogPath
	}

	f, err := os.OpenFile(logfilePath, globalLogFlags, globalLogMode)
	if err != nil {
		return nil, err
	}

	hook := &GlobalLogHook{
		path: logfilePath,
		file: f,
	}

	return hook, nil
}

// Levels informs the logrus Logger which log levels this hook supports.
func (hook *GlobalLogHook) Levels() []logrus.Level {
	// Log at all levels
	return logrus.AllLevels
}

// formatFields returns a "name=value" formatted string of sorted map keys
func formatFields(fields map[string]interface{}) string {
	var keys []string

	for key := range fields {
		keys = append(keys, key)
	}

	sort.Sort(sort.StringSlice(keys))

	var sorted []string

	for _, k := range keys {
		sorted = append(sorted, fmt.Sprintf("%s=%q", k, fields[k]))
	}

	return strings.Join(sorted, " ")
}

// Fire is called by the logrus logger when data is available for the
// hook.
func (hook *GlobalLogHook) Fire(entry *logrus.Entry) error {
	// Ignore any formatter that has been used and log in a custom format
	// to the global log.

	fields := formatFields(entry.Data)

	str := fmt.Sprintf("time=%q pid=%d name=%q level=%q",
		entry.Time,
		os.Getpid(),
		name,
		entry.Level)

	if fields != "" {
		str += " " + fields
	}

	if entry.Message != "" {
		str += " " + fmt.Sprintf("msg=%q", entry.Message)
	}

	str += "\n"

	_, err := hook.file.WriteString(str)
	if err != nil {
		return err
	}

	return nil
}
