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
	"syscall"
	"testing"
)

func TestProcessSignal(t *testing.T) {
	tests := []struct {
		signal string
		valid  bool
		signum syscall.Signal
	}{
		{"SIGDCKBY", false, 0}, //invalid signal
		{"DCKBY", false, 0},    //invalid signal
		{"99999", false, 0},    //invalid signal
		{"SIGTERM", true, syscall.SIGTERM},
		{"TERM", true, syscall.SIGTERM},
		{"15", true, syscall.SIGTERM},
	}

	for _, test := range tests {
		signum, err := processSignal(test.signal)
		if signum != test.signum {
			t.Fatalf("signal received: %d expected signal: %d\n", signum, test.signum)
		}
		if test.valid && err != nil {
			t.Fatalf("signal %s is a valid but a error was received: %s\n", test.signal, err)
		}
		if !test.valid && err == nil {
			t.Fatalf("signal %s is not a valid signal and no error was reported\n", test.signal)
		}
	}
}
