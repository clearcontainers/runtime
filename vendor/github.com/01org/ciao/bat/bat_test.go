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

package bat

import (
	"reflect"
	"testing"
)

var instances = []string{
	"d258443c-72c7-4971-8c2b-cb9925522c3e",
	"64a0cca9-85a2-4733-988b-b4fe9a72dd0e",
}

func TestGoodCheckStatuses(t *testing.T) {
	goodStatus := map[string]string{
		"d258443c-72c7-4971-8c2b-cb9925522c3e": "active",
		"64a0cca9-85a2-4733-988b-b4fe9a72dd0e": "active",
	}

	scheduled, finished, err := checkStatuses(instances, goodStatus, true)
	if len(scheduled) != 2 || !finished || err != nil {
		t.Errorf("goodStatus check failed")
	}
}

func TestPendingCheckStatuses(t *testing.T) {
	pendingStatus := map[string]string{
		"d258443c-72c7-4971-8c2b-cb9925522c3e": "pending",
		"64a0cca9-85a2-4733-988b-b4fe9a72dd0e": "pending",
	}

	scheduled, finished, err := checkStatuses(instances, pendingStatus, true)
	if len(scheduled) != 2 || finished || err != nil {
		t.Errorf("pendingStatus check failed")
	}

	partialPendingStatus := map[string]string{
		"d258443c-72c7-4971-8c2b-cb9925522c3e": "active",
		"64a0cca9-85a2-4733-988b-b4fe9a72dd0e": "pending",
	}

	scheduled, finished, err = checkStatuses(instances, partialPendingStatus, true)
	if len(scheduled) != 2 || finished || err != nil {
		t.Errorf("pendingStatus check failed")
	}
}

func TestExitedCheckStatuses(t *testing.T) {
	exitedStatus := map[string]string{
		"d258443c-72c7-4971-8c2b-cb9925522c3e": "active",
		"64a0cca9-85a2-4733-988b-b4fe9a72dd0e": "exited",
	}

	scheduled, finished, err := checkStatuses(instances, exitedStatus, true)
	if len(scheduled) != 2 || !finished || err == nil {
		t.Errorf("pendingStatus mustActive=true check failed")
	}

	scheduled, finished, err = checkStatuses(instances, exitedStatus, false)
	if len(scheduled) != 2 || !finished || err != nil {
		t.Errorf("pendingStatus mustActive=false check failed")
	}
}

func TestMissingCheckStatuses(t *testing.T) {
	missingStatus := map[string]string{
		"d258443c-72c7-4971-8c2b-cb9925522c3e": "active",
	}

	scheduled, finished, err := checkStatuses(instances, missingStatus, false)
	if len(scheduled) != 1 || !finished || err == nil {
		t.Errorf("pendingStatus mustActive=false check failed")
	}
}

func TestImageOptions(t *testing.T) {
	opts := &ImageOptions{
		ContainerFormat:  "ovf",
		DiskFormat:       "qcow2",
		ID:               "test-id",
		MinDiskGigabytes: 1,
		MinRAMMegabytes:  2,
		Name:             "test-name",
		Protected:        true,
		Tags:             []string{"tag1", "tag2"},
		Visibility:       "private",
	}

	computedArgs := computeImageAddArgs(opts)
	expectedArgs := []string{
		"-container-format", "ovf",
		"-disk-format", "qcow2",
		"-id", "test-id",
		"-min-disk-size", "1",
		"-min-ram-size", "2",
		"-name", "test-name",
		"-protected",
		"-tags", "tag1,tag2",
		"-visibility", "private",
	}

	if !reflect.DeepEqual(computedArgs, expectedArgs) {
		t.Fatalf("Compute image arguments are incorrect")
	}
}
