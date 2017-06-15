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
	"os"
	"path"
	"testing"

	"github.com/dlespiau/covertool/pkg/cover"
	"github.com/dlespiau/covertool/pkg/exit"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{0, 0, 0},
		{1, 1, 2},
		{-2, 1, -1},
		{-1, -10, -11},
	}

	for i := range tests {
		test := &tests[i]
		got := add(test.a, test.b)
		if test.expected != got {
			t.Fatalf("expected %d but got %d", test.expected, got)
		}
	}
}

func TestMain(m *testing.M) {
	cover.ParseAndStripTestFlags()

	// Make sure we have the opportunity to flush the coverage report to disk when
	// terminating the process.
	exit.AtExit(cover.FlushProfiles)

	// If the test binary name is "calc" we've are being asked to run the
	// coverage-instrumented calc.
	if path.Base(os.Args[0]) == "calc" {
		main()
		exit.Exit(0)
	}

	os.Exit(m.Run())
}
