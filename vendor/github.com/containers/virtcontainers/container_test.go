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

func TestGetAnnotations(t *testing.T) {
	annotations := map[string]string{
		"annotation1": "abc",
		"annotation2": "xyz",
		"annotation3": "123",
	}

	container := Container{
		config: &ContainerConfig{
			Annotations: annotations,
		},
	}

	containerAnnotations := container.GetAnnotations()

	for k, v := range containerAnnotations {
		if annotations[k] != v {
			t.Fatalf("Expecting ['%s']='%s', Got ['%s']='%s'\n", k, annotations[k], k, v)
		}
	}
}

func TestContainerPod(t *testing.T) {
	expectedPod := &Pod{}

	container := Container{
		pod: expectedPod,
	}

	pod := container.Pod()

	if !reflect.DeepEqual(pod, expectedPod) {
		t.Fatalf("Expecting %+v\nGot %+v", expectedPod, pod)
	}
}
