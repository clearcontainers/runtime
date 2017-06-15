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
	"fmt"
	"os"
	"strconv"

	"github.com/dlespiau/covertool/pkg/exit"
)

// add is tested with a unit test
func add(a, b int) int {
	return a + b
}

// sub is tested by running "calc sub 2 3".
func sub(a, b int) int {
	return a - b
}

// The error paths in main are tested by invoking calc.
func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "expected 3 arguments, got %d\n", len(os.Args)-1)
		exit.Exit(1)
	}

	// No error checking here, it's a contrived example after all.
	a, _ := strconv.Atoi(os.Args[2])
	b, _ := strconv.Atoi(os.Args[3])

	var op func(int, int) int
	switch os.Args[1] {
	case "add":
		op = add
	case "sub":
		op = sub
	default:
		fmt.Fprintf(os.Stderr, "unknown operation: %s\n", os.Args[1])
		exit.Exit(1)
	}

	fmt.Println(op(a, b))
}
