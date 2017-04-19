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

package virtcontainers

import (
	"reflect"
	"testing"
)

func testSetShimType(t *testing.T, value string, expected ShimType) {
	var shimType ShimType

	err := (&shimType).Set(value)
	if err != nil {
		t.Fatal(err)
	}

	if shimType != expected {
		t.Fatalf("Got %s\nExpecting %s", shimType, expected)
	}
}

func TestSetCCShimType(t *testing.T) {
	testSetShimType(t, "ccShim", CCShimType)
}

func TestSetNoopShimType(t *testing.T) {
	testSetShimType(t, "noopShim", NoopShimType)
}

func TestSetUnknownShimType(t *testing.T) {
	var shimType ShimType

	unknownType := "unknown"

	err := (&shimType).Set(unknownType)
	if err == nil {
		t.Fatalf("Should fail because %s type used", unknownType)
	}

	if shimType == CCShimType || shimType == NoopShimType {
		t.Fatalf("%s shim type was not expected", shimType)
	}
}

func testStringFromShimType(t *testing.T, shimType ShimType, expected string) {
	shimTypeStr := (&shimType).String()
	if shimTypeStr != expected {
		t.Fatalf("Got %s\nExpecting %s", shimTypeStr, expected)
	}
}

func TestStringFromCCShimType(t *testing.T) {
	shimType := CCShimType
	testStringFromShimType(t, shimType, "ccShim")
}

func TestStringFromNoopShimType(t *testing.T) {
	shimType := NoopShimType
	testStringFromShimType(t, shimType, "noopShim")
}

func TestStringFromUnknownShimType(t *testing.T) {
	var shimType ShimType
	testStringFromShimType(t, shimType, "")
}

func testNewShimFromShimType(t *testing.T, shimType ShimType, expected shim) {
	result, err := newShim(shimType)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(result, expected) == false {
		t.Fatalf("Got %+v\nExpecting %+v", result, expected)
	}
}

func TestNewShimFromCCShimType(t *testing.T) {
	shimType := CCShimType
	expectedShim := &ccShim{}
	testNewShimFromShimType(t, shimType, expectedShim)
}

func TestNewShimFromNoopShimType(t *testing.T) {
	shimType := NoopShimType
	expectedShim := &noopShim{}
	testNewShimFromShimType(t, shimType, expectedShim)
}

func TestNewShimFromUnknownShimType(t *testing.T) {
	var shimType ShimType

	_, err := newShim(shimType)
	if err != nil {
		t.Fatal(err)
	}
}

func testNewShimConfigFromPodConfig(t *testing.T, podConfig PodConfig, expected interface{}) {
	result := newShimConfig(podConfig)

	if reflect.DeepEqual(result, expected) == false {
		t.Fatalf("Got %+v\nExpecting %+v", result, expected)
	}
}

func TestNewShimConfigFromCCShimPodConfig(t *testing.T) {
	shimConfig := CCShimConfig{}

	podConfig := PodConfig{
		ShimType:   CCShimType,
		ShimConfig: shimConfig,
	}

	testNewShimConfigFromPodConfig(t, podConfig, shimConfig)
}

func TestNewShimConfigFromNoopShimPodConfig(t *testing.T) {
	podConfig := PodConfig{
		ShimType: NoopShimType,
	}

	testNewShimConfigFromPodConfig(t, podConfig, nil)
}

func TestNewShimConfigFromUnknownShimPodConfig(t *testing.T) {
	var shimType ShimType

	podConfig := PodConfig{
		ShimType: shimType,
	}

	testNewShimConfigFromPodConfig(t, podConfig, nil)
}
