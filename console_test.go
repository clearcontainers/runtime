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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsoleFromFile(t *testing.T) {
	assert := assert.New(t)

	console := ConsoleFromFile(os.Stdout)

	assert.NotNil(console.File(), "console file is nil")
}

func TestNewConsole(t *testing.T) {
	assert := assert.New(t)

	console, err := newConsole()
	assert.NoError(err, "failed to create a new console: %s", err)
	defer console.Close()

	assert.NotEmpty(console.Path(), "console path is empty")

	assert.NotNil(console.File(), "console file is nil")
}

func TestIsTerminal(t *testing.T) {
	assert := assert.New(t)

	var fd uintptr = 4
	assert.False(isTerminal(fd), "Fd %d is not a terminal", fd)

	console, err := newConsole()
	assert.NoError(err, "failed to create a new console: %s", err)
	defer console.Close()

	fd = console.File().Fd()
	assert.True(isTerminal(fd), "Fd %d is a terminal", fd)
}
