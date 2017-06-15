//
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
//

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/opencontainers/runc/libcontainer/configs"
)

var logFile = "/tmp/mock_hook.log"
var testKey = "test-key"
var testContainerID = "test-container-id"
var testControllerID = "test-controller-id"

func main() {
	err := os.RemoveAll(logFile)
	if err != nil {
		os.Exit(1)
	}

	f, err := os.Create(logFile)
	if err != nil {
		os.Exit(1)
	}
	defer f.Close()

	fmt.Fprintf(f, "args = %s\n", os.Args)

	if len(os.Args) < 3 {
		fmt.Fprintf(f, "At least 3 args expected, only %d received\n", len(os.Args))
		os.Exit(1)
	}

	if os.Args[0] != testKey {
		fmt.Fprintf(f, "args[0] should be \"%s\", received \"%s\" instead\n", testKey, os.Args[0])
		os.Exit(1)
	}

	if os.Args[1] != testContainerID {
		fmt.Fprintf(f, "argv[1] should be \"%s\", received \"%s\" instead\n", testContainerID, os.Args[1])
		os.Exit(1)
	}

	if os.Args[2] != testControllerID {
		fmt.Fprintf(f, "argv[2] should be \"%s\", received \"%s\" instead\n", testControllerID, os.Args[2])
		os.Exit(1)
	}

	stateBuf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(f, "Could not read on stdin: %s\n", err)
		os.Exit(1)
	}

	var state configs.HookState
	err = json.Unmarshal(stateBuf, &state)
	if err != nil {
		fmt.Fprintf(f, "Could not unmarshal HookState json: %s\n", err)
		os.Exit(1)
	}

	if state.Pid < 1 {
		fmt.Fprintf(f, "Invalid PID: %d\n", state.Pid)
		os.Exit(1)
	}

	// Intended to sleep, so as to make the test passing/failing.
	if len(os.Args) == 4 {
		timeout, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Fprintf(f, "Could not retrieve timeout %s from args[3]\n", os.Args[3])
			os.Exit(1)
		}
		time.Sleep(time.Duration(timeout) * time.Second)
	}
}
