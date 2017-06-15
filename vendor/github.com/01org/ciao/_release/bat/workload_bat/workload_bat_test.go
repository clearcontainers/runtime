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

package workloadbat

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/01org/ciao/bat"
)

const standardTimeout = time.Second * 300

const vmCloudInit = `---
#cloud-config
users:
  - name: demouser
    geocos: CIAO Demo User
    lock-passwd: false
    passwd: %s
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
      - %s
...
`

const vmWorkloadImageName = "Ubuntu Server 16.04"

func getWorkloadSource(ctx context.Context, t *testing.T, tenant string) bat.Source {
	// get the Image ID to use.
	source := bat.Source{
		Type: "image",
	}

	// if we pass in "" for tenant, we get whatever the CIAO_USERNAME value
	// is set to.
	images, err := bat.GetImages(ctx, tenant)
	if err != nil {
		t.Fatal(err)
	}

	for ID, image := range images {
		if image.Name != vmWorkloadImageName {
			continue
		}
		source.ID = ID
	}

	if source.ID == "" {
		t.Fatalf("vm Image %s not available", vmWorkloadImageName)
	}

	return source
}

func testCreateWorkload(t *testing.T, public bool) {
	// we'll use empty string for now
	tenant := ""

	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	// generate ssh test keys?

	source := getWorkloadSource(ctx, t, tenant)

	// fill out the opt structure for this workload.
	defaults := bat.DefaultResources{
		VCPUs: 2,
		MemMB: 128,
	}

	disk := bat.Disk{
		Bootable:  true,
		Source:    &source,
		Ephemeral: true,
	}

	opt := bat.WorkloadOptions{
		Description: "BAT VM Test",
		VMType:      "qemu",
		FWType:      "legacy",
		Defaults:    defaults,
		Disks:       []bat.Disk{disk},
	}

	var ID string
	var err error
	if public {
		ID, err = bat.CreatePublicWorkload(ctx, tenant, opt, vmCloudInit)
	} else {
		ID, err = bat.CreateWorkload(ctx, tenant, opt, vmCloudInit)
	}

	if err != nil {
		t.Fatal(err)
	}

	// now retrieve the workload from controller.
	w, err := bat.GetWorkloadByID(ctx, "", ID)
	if err != nil {
		t.Fatal(err)
	}

	if w.Name != opt.Description || w.CPUs != opt.Defaults.VCPUs || w.Mem != opt.Defaults.MemMB {
		t.Fatalf("Workload not defined correctly")
	}

	// delete the workload.
	if public {
		err = bat.DeletePublicWorkload(ctx, w.ID)
	} else {
		err = bat.DeleteWorkload(ctx, tenant, w.ID)
	}

	if err != nil {
		t.Fatal(err)
	}

	// now try to retrieve the workload from controller.
	_, err = bat.GetWorkloadByID(ctx, "", ID)
	if err == nil {
		t.Fatalf("Workload not deleted correctly")
	}
}

// Check that a tenant workload can be created.
//
// Create a tenant workload and confirm that the workload exists.
//
// The new workload should be visible to the tenant and contain
// the correct resources and description.
func TestCreateTenantWorkload(t *testing.T) {
	testCreateWorkload(t, false)
}

// Check that a public workload can be created.
//
// Create a public workload and confirm that the workload exists.
//
// The new public workload should be visible to the tenant and contain
// the correct resources and description.
func TestCreatePublicWorkload(t *testing.T) {
	testCreateWorkload(t, true)
}

func findQuota(qds []bat.QuotaDetails, name string) *bat.QuotaDetails {
	for i := range qds {
		if qds[i].Name == name {
			return &qds[i]
		}
	}
	return nil
}

// Check workload creation with a sized volume.
//
// Create a workload with a storage specification that has a size, boot
// an instance from that workload and check that the storage usage goes
// up. Then delete the instance and the created workload.
//
// The new workload is created successfully and the storage used by the
// instance created from the workload matches the requested size.
func TestCreateWorkloadWithSizedVolume(t *testing.T) {
	tenant := ""

	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	source := getWorkloadSource(ctx, t, tenant)

	defaults := bat.DefaultResources{
		VCPUs: 2,
		MemMB: 128,
	}

	disk := bat.Disk{
		Bootable:  true,
		Source:    &source,
		Ephemeral: true,
		Size:      10,
	}

	opt := bat.WorkloadOptions{
		Description: "BAT VM Test",
		VMType:      "qemu",
		FWType:      "legacy",
		Defaults:    defaults,
		Disks:       []bat.Disk{disk},
	}

	workloadID, err := bat.CreateWorkload(ctx, tenant, opt, vmCloudInit)

	if err != nil {
		t.Fatal(err)
	}

	w, err := bat.GetWorkloadByID(ctx, tenant, workloadID)
	if err != nil {
		t.Fatal(err)
	}

	initalQuotas, err := bat.ListQuotas(ctx, tenant, "")
	if err != nil {
		t.Error(err)
	}

	instances, err := bat.LaunchInstances(ctx, tenant, w.ID, 1)
	if err != nil {
		t.Error(err)
	}

	scheduled, err := bat.WaitForInstancesLaunch(ctx, tenant, instances, false)
	if err != nil {
		t.Errorf("Instances failed to launch: %v", err)
	}

	updatedQuotas, err := bat.ListQuotas(ctx, tenant, "")
	if err != nil {
		t.Error(err)
	}

	storageBefore := findQuota(initalQuotas, "tenant-storage-quota")
	storageAfter := findQuota(updatedQuotas, "tenant-storage-quota")

	if storageBefore == nil || storageAfter == nil {
		t.Errorf("Quota not found for storage")
	}

	before, _ := strconv.Atoi(storageBefore.Usage)
	after, _ := strconv.Atoi(storageAfter.Usage)

	if after-before < 10 {
		t.Errorf("Storage usage not increased by expected amount")
	}

	for _, i := range scheduled {
		err = bat.DeleteInstanceAndWait(ctx, "", i)
		if err != nil {
			t.Errorf("Failed to delete instances: %v", err)
		}
	}

	err = bat.DeleteWorkload(ctx, tenant, w.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = bat.GetWorkloadByID(ctx, tenant, workloadID)
	if err == nil {
		t.Fatalf("Workload not deleted correctly")
	}
}
