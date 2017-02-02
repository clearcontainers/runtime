//
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
//

package virtcontainers

import (
	"fmt"
	"reflect"
	"testing"
)

func testSetHypervisorType(t *testing.T, value string, expected HypervisorType) {
	var hypervisorType HypervisorType

	err := (&hypervisorType).Set(value)
	if err != nil {
		t.Fatal(err)
	}

	if hypervisorType != expected {
		t.Fatal()
	}
}

func TestSetQemuHypervisorType(t *testing.T) {
	testSetHypervisorType(t, "qemu", QemuHypervisor)
}

func TestSetMockHypervisorType(t *testing.T) {
	testSetHypervisorType(t, "mock", MockHypervisor)
}

func TestSetUnknownHypervisorType(t *testing.T) {
	var hypervisorType HypervisorType

	err := (&hypervisorType).Set("unknown")
	if err == nil {
		t.Fatal()
	}

	if hypervisorType == QemuHypervisor ||
		hypervisorType == MockHypervisor {
		t.Fatal()
	}
}

func testStringFromHypervisorType(t *testing.T, hypervisorType HypervisorType, expected string) {
	hypervisorTypeStr := (&hypervisorType).String()
	if hypervisorTypeStr != expected {
		t.Fatal()
	}
}

func TestStringFromQemuHypervisorType(t *testing.T) {
	hypervisorType := QemuHypervisor
	testStringFromHypervisorType(t, hypervisorType, "qemu")
}

func TestStringFromMockHypervisorType(t *testing.T) {
	hypervisorType := MockHypervisor
	testStringFromHypervisorType(t, hypervisorType, "mock")
}

func TestStringFromUnknownHypervisorType(t *testing.T) {
	var hypervisorType HypervisorType
	testStringFromHypervisorType(t, hypervisorType, "")
}

func testNewHypervisorFromHypervisorType(t *testing.T, hypervisorType HypervisorType, expected hypervisor) {
	hy, err := newHypervisor(hypervisorType)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(hy, expected) == false {
		t.Fatal()
	}
}

func TestNewHypervisorFromQemuHypervisorType(t *testing.T) {
	hypervisorType := QemuHypervisor
	expectedHypervisor := &qemu{}
	testNewHypervisorFromHypervisorType(t, hypervisorType, expectedHypervisor)
}

func TestNewHypervisorFromMockHypervisorType(t *testing.T) {
	hypervisorType := MockHypervisor
	expectedHypervisor := &mockHypervisor{}
	testNewHypervisorFromHypervisorType(t, hypervisorType, expectedHypervisor)
}

func TestNewHypervisorFromUnknownHypervisorType(t *testing.T) {
	var hypervisorType HypervisorType

	hy, err := newHypervisor(hypervisorType)
	if err == nil {
		t.Fatal()
	}

	if hy != nil {
		t.Fatal()
	}
}

func testHypervisorConfigValid(t *testing.T, hypervisorConfig *HypervisorConfig, expected bool) {
	ret, _ := hypervisorConfig.valid()
	if ret != expected {
		t.Fatal()
	}
}

func TestHypervisorConfigNoKernelPath(t *testing.T) {
	hypervisorConfig := &HypervisorConfig{
		KernelPath:     "",
		ImagePath:      fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath: fmt.Sprintf("%s/%s", testDir, testHypervisor),
	}

	testHypervisorConfigValid(t, hypervisorConfig, false)
}

func TestHypervisorConfigNoImagePath(t *testing.T) {
	hypervisorConfig := &HypervisorConfig{
		KernelPath:     fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:      "",
		HypervisorPath: fmt.Sprintf("%s/%s", testDir, testHypervisor),
	}

	testHypervisorConfigValid(t, hypervisorConfig, false)
}

func TestHypervisorConfigNoHypervisorPath(t *testing.T) {
	hypervisorConfig := &HypervisorConfig{
		KernelPath:     fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:      fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath: "",
	}

	testHypervisorConfigValid(t, hypervisorConfig, false)
}

func TestHypervisorConfigIsValid(t *testing.T) {
	hypervisorConfig := &HypervisorConfig{
		KernelPath:     fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:      fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath: fmt.Sprintf("%s/%s", testDir, testHypervisor),
	}

	testHypervisorConfigValid(t, hypervisorConfig, true)
}

func TestAppendParams(t *testing.T) {
	paramList := []Param{
		{
			parameter: "param1",
			value:     "value1",
		},
	}

	expectedParams := []Param{
		{
			parameter: "param1",
			value:     "value1",
		},
		{
			parameter: "param2",
			value:     "value2",
		},
	}

	paramList = appendParam(paramList, "param2", "value2")
	if reflect.DeepEqual(paramList, expectedParams) == false {
		t.Fatal()
	}
}

func testSerializeParams(t *testing.T, params []Param, delim string, expected []string) {
	result := serializeParams(params, delim)
	if reflect.DeepEqual(result, expected) == false {
		t.Fatal()
	}
}

func TestSerializeParamsNoParamNoValue(t *testing.T) {
	params := []Param{
		{
			parameter: "",
			value:     "",
		},
	}
	var expected []string

	testSerializeParams(t, params, "", expected)
}

func TestSerializeParamsNoParam(t *testing.T) {
	params := []Param{
		{
			value: "value1",
		},
	}

	expected := []string{"value1"}

	testSerializeParams(t, params, "", expected)
}

func TestSerializeParamsNoValue(t *testing.T) {
	params := []Param{
		{
			parameter: "param1",
		},
	}

	expected := []string{"param1"}

	testSerializeParams(t, params, "", expected)
}

func TestSerializeParamsNoDelim(t *testing.T) {
	params := []Param{
		{
			parameter: "param1",
			value:     "value1",
		},
	}

	expected := []string{"param1", "value1"}

	testSerializeParams(t, params, "", expected)
}

func TestSerializeParams(t *testing.T) {
	params := []Param{
		{
			parameter: "param1",
			value:     "value1",
		},
	}

	expected := []string{"param1=value1"}

	testSerializeParams(t, params, "=", expected)
}
