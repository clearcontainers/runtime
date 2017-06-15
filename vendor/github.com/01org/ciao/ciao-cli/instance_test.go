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

import "testing"

func TestVolumeBoolSubArgs(t *testing.T) {
	var stringTests = []struct {
		subArg   string
		expected bool
	}{
		{"swap", true},
		{"local", true},
		{"ephemeral", true},
		{"invalid", false},
		{"swap=true", true},
		{"local=true", true},
		{"ephemeral=true", true},
		{"swap=false", true},
		{"local=false", true},
		{"ephemeral=false", true},
		{"ephemeral=foo", false},
		{"swap=bar", false},
		{"invalid=baz", false},
	}
	for _, test := range stringTests {
		ok := isInstanceVolumeBoolArg(test.subArg)
		if ok != test.expected {
			t.Errorf("expected \"%t\" for \"%s\", got \"%t\"",
				test.expected, test.subArg, ok)
		}
	}
}

func TestVolumeIntSubArgs(t *testing.T) {
	var stringTests = []struct {
		subArg   string
		expected bool
	}{
		{"size=1", true},
		{"size=42", true},
		{"size", false},
		{"size=1G", false},
		{"invalid", false},
		{"invalid=foo", false},
	}
	for _, test := range stringTests {
		ok := isInstanceVolumeIntArg(test.subArg)
		if ok != test.expected {
			t.Errorf("expected \"%t\" for \"%s\", got \"%t\"",
				test.expected, test.subArg, ok)
		}
	}
}

func TestVolumeStringSubArgs(t *testing.T) {
	var stringTests = []struct {
		subArg   string
		expected bool
	}{
		{"uuid", false},
		{"uuid=foo", true},
		{"boot_index", false},
		{"boot_index=none", true},
		{"boot_index=-1", true},
		{"boot_index=0", true},
		{"boot_index=1", true},
		{"tag", false},
		{"tag=bar", true},
		{"tag=baz", true},
		{"tag='something more complicated'", true},
		{"invalid=foo", false},
	}
	for _, test := range stringTests {
		ok := isInstanceVolumeStringArg(test.subArg)
		if ok != test.expected {
			t.Errorf("expected \"%t\" for \"%s\", got \"%t\"",
				test.expected, test.subArg, ok)
		}
	}
}

func TestVolumeSubArgSet(t *testing.T) {
	var stringTestsGood = []struct {
		subArg string
	}{
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,boot_index=none"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,boot_index=-1"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,boot_index=0"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,boot_index=1"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,swap"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,swap=true"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,ephemeral"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,ephemeral=false"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,ephemeral,boot_index=1"},
		{"size=42"},
		{"ephemeral,size=42"},
		{"local,ephemeral,size=42"},
		{"swap,size=42"},
		{"local,swap,size=42"},
	}
	var stringTestsBad = []struct {
		subArg string
	}{
		{""},
		{"size"},
		{"size=42,size=41"},
		{"uuid"},
		{"swap"},
		{"local"},
		{"boot-index"},
		{"boot-index=foo"},
		{"boot-index=1"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,boot_index"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,boot_index=foo"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,swap=true,swap=false"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,swap=invalid"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,boot_index=none,boot_index=1"},
		{"uuid=d033fdcf-f2a2-4bf4-8f5c-0a935c5c7c65,size=1"},
	}

	for _, test := range stringTestsGood {
		var v volumeFlagSlice
		err := v.Set(test.subArg)
		if err != nil {
			t.Errorf("valid sub arg string incorrectly refused: \"%s\"", test.subArg)
		}
	}
	for _, test := range stringTestsBad {
		var v volumeFlagSlice
		err := v.Set(test.subArg)
		if err == nil {
			t.Errorf("invalid sub arg string incorrectly accepted: \"%s\"", test.subArg)
		}
	}
}
