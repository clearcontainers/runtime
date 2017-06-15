/*
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
*/

package main

import (
	"sync"
	"testing"
)

// Test the port grabbing and release code
//
// Grab all the available ports concurrently, check that they are all
// allocated and then release all the ports concurrently.
//
// All ports should be grabbed correctly, all ports should be reported
// to be allocated, i.e., uiPortGrabber.grabPort should return 0, and
// all ports should be released correctly.  At the end of the test the
// size of uiPortGrabber.free should equal the maximum number of ports
// available.
func TestPortGrabber(t *testing.T) {
	ports := make([]int, portGrabberMax-portGrabberStart)

	var wg sync.WaitGroup
	wg.Add(len(ports))
	for i := 0; i < len(ports); i++ {
		go func(i int) {
			ports[i] = uiPortGrabber.grabPort()
			if ports[i] == 0 {
				t.Error("Expected grabPort to return non zero value")
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	if len(uiPortGrabber.free) != 0 {
		t.Error("Expected all ports to be allocated")
	}

	if uiPortGrabber.grabPort() != 0 {
		t.Error("Expected grabPort to return 0")
	}

	wg.Add(len(ports))
	for i := 0; i < len(ports); i++ {
		go func(port int) {
			uiPortGrabber.releasePort(port)
			wg.Done()
		}(ports[i])
	}
	wg.Wait()

	if len(uiPortGrabber.free) != len(ports) {
		t.Error("Expected no ports to be allocated")
	}
}
