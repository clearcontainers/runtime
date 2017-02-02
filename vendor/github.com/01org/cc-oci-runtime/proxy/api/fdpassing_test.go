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

package api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Ensure the message we send along with the fd is only a single byte
func TestTagLength(t *testing.T) {
	assert.Equal(t, 1, len(fileTagMsg))
}

func TestFdPassing(t *testing.T) {
	reader, writer, err := os.Pipe()
	assert.Nil(t, err)

	// Passes the reader end of the pipe fd through a AF_UNIX connection
	// and recreate an os.File from the received fd
	c0, c1, err := socketpair()
	assert.Nil(t, err)

	err = WriteFd(c0, int(reader.Fd()))
	assert.Nil(t, err)

	newFd, err := ReadFd(c1)
	assert.Nil(t, err)
	assert.NotEqual(t, newFd, -1)

	newReader := os.NewFile(uintptr(newFd), "")

	// write into the pipe and check reading from newReader gives the
	// expected result
	var data = []byte("foo")

	n, err := writer.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, n, len(data))

	buf := make([]byte, 512)
	n, err = newReader.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, n, len(data))
	assert.Equal(t, data, buf[:n])

	// cleanup
	reader.Close()
	writer.Close()
	c0.Close()
	c1.Close()
	newReader.Close()
}
