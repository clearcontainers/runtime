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
	"bytes"
	"fmt"
	"strings"
	"testing"
)

type testint int

var templateUsageTests = []struct {
	obj      interface{}
	expected string
}{
	{int(0), "int"},
	{testint(0), "main.testint"},
	{[]int{}, "[]int"},
	{false, "bool"},
	{[5]int{}, "[5]int"},
	{func(int) (int, error) { return 0, nil }, "func(int) (int, error)"},
	{"", "string"},
	{struct {
		X       int
		Y       string
		hidden  float64
		Invalid chan int
		Empty   struct{}
	}{}, "struct {\nX int\nY string\nEmpty struct {\n}\n}"},
	{map[string]struct{ X int }{}, "map[string]struct {\nX int\n}"},
	{map[string]struct{ x int }{}, "map[string]struct {\n}"},
	{struct {
		hidden int
		Embed  struct {
			X int
		}
	}{}, "struct {\nEmbed struct {\nX int\n}\n}"},
	{struct{ hidden int }{}, "struct {\n}"},
	{struct{ Empty struct{} }{}, "struct { Empty struct{\n}}"},
	{struct{}{}, "struct {\n}"},
	{struct {
		X int            `test:"tag"`
		Y []int          `test:"tag"`
		Z map[string]int `test:"tag"`
		B struct {
			A int
		} `test:"tag"`
	}{}, "struct { X int `test:\"tag\"`; Y []int `test:\"tag\"`; Z map[string]int `test:\"tag\"`; B struct {\nA int\n} `test:\"tag\"`} "},
}

func TestTemplateGenerateUsage(t *testing.T) {
	for _, s := range templateUsageTests {
		gen := generateUsageUndecorated(s.obj)
		var buf bytes.Buffer
		formatType(&buf, []byte(s.expected))
		trimmedGen := strings.TrimSpace(gen)
		trimmedExpected := strings.TrimSpace(buf.String())
		if trimmedGen != trimmedExpected {
			t.Errorf("Bad template usage. Found\n%s, expected\n%s",
				trimmedGen, trimmedExpected)
		}
	}
	fmt.Println()
}
