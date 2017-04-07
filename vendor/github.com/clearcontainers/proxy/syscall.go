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

package main

import (
	"net"
	"os"
	"syscall"
)

// Socketpair wraps the eponymous syscall but gives go friendly objects instead
// of the raw file descriptors.
func Socketpair() (*net.UnixConn, *net.UnixConn, error) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, err
	}

	// First end
	f0 := os.NewFile(uintptr(fds[0]), "")
	// os.NewFile() dups the fd and we're responsible for closing it
	defer f0.Close()
	c0, err := net.FileConn(f0)
	if err != nil {
		return nil, nil, err
	}

	// Second end
	f1 := os.NewFile(uintptr(fds[1]), "")
	defer f1.Close()
	c1, err := net.FileConn(f1)
	if err != nil {
		return nil, nil, err
	}

	return c0.(*net.UnixConn), c1.(*net.UnixConn), nil
}
