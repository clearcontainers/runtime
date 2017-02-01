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

package storagebat

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/01org/ciao/bat"
)

const standardTimeout = time.Second * 300

// TODO: We're launching an instance from a known workload UUID that
// that has no data volumes attached.

func createSpecificInstance(ctx context.Context, t *testing.T, tenant, workloadID string,
	mustBeActive bool) string {
	wklds, err := bat.GetAllWorkloads(ctx, tenant)
	if err != nil {
		t.Fatalf("Unable to retrieve workload list : %v", err)
	}

	i := 0
	for ; i < len(wklds); i++ {
		if wklds[i].ID == workloadID {
			break
		}
	}

	if i == len(wklds) {
		t.Skip()
	}

	instances, err := bat.LaunchInstances(ctx, tenant, workloadID, 1)
	if err != nil {
		t.Fatalf("Unable to launch instance : %v", err)
	}

	_, err = bat.WaitForInstancesLaunch(ctx, tenant, instances, mustBeActive)
	if err != nil {
		_ = bat.DeleteInstance(ctx, tenant, instances[0])
		t.Fatalf("Instance %s did not start correctly : %v",
			instances[0], err)
	}

	return instances[0]
}

func createVMInstance(ctx context.Context, t *testing.T, tenant string) string {
	const testWorkloadID = "79034317-3beb-447e-987d-4e310a8cf410"
	return createSpecificInstance(ctx, t, tenant, testWorkloadID, true)
}

func createContainerInstance(ctx context.Context, t *testing.T, tenant string) string {
	const testWorkloadID = "ca957444-fa46-11e5-94f9-38607786d9ec"
	return createSpecificInstance(ctx, t, tenant, testWorkloadID, true)
}

func checkBootedVolume(ctx context.Context, t *testing.T, tenant, instanceID string) string {
	instance, err := bat.GetInstance(ctx, tenant, instanceID)
	if err != nil {
		t.Fatalf("Unable to retrieve instance %s details : %v",
			instanceID, err)
	}

	if len(instance.Volumes) == 0 {
		t.Fatalf("No volumes are attached to the launched instance %s",
			instanceID)
	}

	volumeID := instance.Volumes[0]
	vol, err := bat.GetVolume(ctx, tenant, volumeID)
	if err != nil {
		t.Fatalf("Failed to retrieve volume information for %s: %v",
			volumeID, err)
	}

	if vol.TenantID != instance.TenantID {
		t.Fatalf("Volume and instance tenant ids do not match %s != %s",
			vol.TenantID, instance.TenantID)
	}

	if vol.Status != "in-use" {
		t.Fatalf("Incorrect volume status in-use expected found '%s'", vol.Status)
	}

	return volumeID
}

// Test bootable volumes are created and deleted correctly
//
// Boot a VM which has no data volumes attached, retrieve the volume ID from
// ciao-cli, retrieve information about the volume, delete the instance and
// check that the volume has also been deleted.
//
// The instance should be created successfully.  It should have one volume attached.
// The status of the volume should be in-use.  The volume should be deleted with
// the instance.
func TestBootFromVolume(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instanceID := createVMInstance(ctx, t, "")
	defer func() {
		if instanceID != "" {
			err := bat.DeleteInstance(ctx, "", instanceID)
			if err != nil {
				t.Errorf("Failed to delete instance %s : %v",
					instanceID, err)
			}
		}
	}()

	volumeID := checkBootedVolume(ctx, t, "", instanceID)

	err := bat.DeleteInstanceAndWait(ctx, "", instanceID)
	if err != nil {
		t.Fatalf("Unable to delete instance %s : %v", instanceID, err)
	}

	instanceID = ""
	_, err = bat.GetVolume(ctx, "", volumeID)
	if err == nil {
		t.Errorf("Volume %s not deleted", volumeID)
	}
}

// Check bootable volumes of stopped instances are in-use
//
// Boot a VM which has no data volumes attached and stop the instance.  Check
// the status of the bootable volume.  Delete the instance.
//
// The instance should be created successfully.  It should have one volume attached.
// The status of the volume should be in-use.
func TestStoppedInstance(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instanceID := createVMInstance(ctx, t, "")
	defer func() {
		err := bat.DeleteInstance(ctx, "", instanceID)
		if err != nil {
			t.Errorf("Failed to delete instance %s : %v",
				instanceID, err)
		}
	}()

	err := bat.StopInstanceAndWait(ctx, "", instanceID)
	if err != nil {
		t.Fatalf("Failed to stop instance %s : %v", instanceID, err)
	}

	_ = checkBootedVolume(ctx, t, "", instanceID)
}

// Check that in-use volumes cannot be deleted
//
// Boot a VM and try to delete the volume from which it booted.  Delete the instance.
//
// The instance should be created, the attempt to delete the volume should fail, and
// the instance should be correctly deleted.
func TestDeleteInUse(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instanceID := createVMInstance(ctx, t, "")
	defer func() {
		err := bat.DeleteInstance(ctx, "", instanceID)
		if err != nil {
			t.Errorf("Failed to delete instance %s : %v",
				instanceID, err)
		}
	}()

	volumeID := checkBootedVolume(ctx, t, "", instanceID)
	err := bat.DeleteVolume(ctx, "", volumeID)
	if err == nil {
		t.Fatalf("Succeeded in deleting in-use volume %s", volumeID)
	} else if err == context.Canceled {
		t.Fatalf("Attempt to delete volume %s cancelled or timed out : %v",
			volumeID, err)
	}
}

// Check that the rootfs volumes cannot be detached
//
// Boot a VM and try to detach the volume from which it booted.  Delete the instance.
//
// The instance should be created, the attempt to detach the volume should fail, and
// the instance and volume should be correctly deleted.
func TestDetachRootFS(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instanceID := createVMInstance(ctx, t, "")
	defer func() {
		err := bat.DeleteInstance(ctx, "", instanceID)
		if err != nil {
			t.Errorf("Failed to delete instance %s : %v",
				instanceID, err)
		}
	}()

	volumeID := checkBootedVolume(ctx, t, "", instanceID)
	err := bat.DetachVolume(ctx, "", volumeID)
	if err == nil {
		t.Fatalf("Succeeded in detaching in-use volume %s", volumeID)
	} else if err == context.Canceled {
		t.Fatalf("Attempt to detach volume %s cancelled or timed out : %v",
			volumeID, err)
	}
}

// Check that volumes can be added, listed and deleted
//
// This test creates 10 new volumes of 1GB each.  It then retrieves the meta
// data for each volume, checks it matches the meta data specified at volume
// creation and then deletes the volumes.
//
// The volumes should be created and enumerated correctly.  The enumerated
// meta data should match the meta data specified during volume creation
// and the volumes should be deleted without error.
func TestAddListDelete(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	const volumeCount = 10
	volumeIDs := make([]string, 0, volumeCount)

	defer func() {
		for _, id := range volumeIDs {
			if err := bat.DeleteVolume(ctx, "", id); err != nil {
				t.Errorf("Failed to delete volume %s : %v", id, err)
			}
		}
	}()

	for i := 0; i < volumeCount; i++ {
		id, err := bat.AddVolume(ctx, "", "", "", &bat.VolumeOptions{
			Size:        1,
			Name:        fmt.Sprintf("%d", i+1),
			Description: fmt.Sprintf("%d description", i+1),
		})
		if err != nil {
			t.Fatalf("Unable to add volume %d :%v", i, err)
		}
		volumeIDs = append(volumeIDs, id)
	}

	volumes, err := bat.GetAllVolumes(ctx, "")
	if err != nil {
		t.Fatalf("Unable to retrieve list of errors %v", err)
	}

	for i, id := range volumeIDs {
		vol := volumes[id]
		if vol == nil {
			t.Fatalf("Unable to find volume %s", id)
		}
		if vol.Size != 1 || vol.Name != fmt.Sprintf("%d", i+1) ||
			vol.Description != fmt.Sprintf("%d description", i+1) {
			t.Fatalf("Incorrect meta data for %s: size %d, name %s, description %s",
				vol.ID, vol.Size, vol.Name, vol.Description)
		}

		if vol.Status != "available" {
			t.Fatalf("Incorrect status %s for volume %s.  Expected available",
				vol.Status, id)
		}
	}
}

// Check attaching volumes become available when deleting their instance
//
// Create a VM and a volume and then attach the volume to the VM.  Don't wait for the
// volume to attach.  Instead delete the instance straight away and then delete the
// volume.
//
// Instance and volume should be created and the volume should transition to the
// attaching state without issue.  The volume should become available after the
// the instance has been deleted and should be deleted correctly.
func TestDeleteInstanceWhileAttaching(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instanceID := createVMInstance(ctx, t, "")
	var volumeID string
	defer func() {
		if instanceID != "" {
			err := bat.DeleteInstanceAndWait(ctx, "", instanceID)
			if err != nil {
				t.Errorf("Failed to delete instance %s : %v",
					instanceID, err)
			}
		}
		if volumeID != "" {
			err := bat.DeleteVolume(ctx, "", volumeID)
			if err != nil {
				t.Errorf("Failed to delete volume %s : %v", volumeID, err)
			}
		}
	}()

	volumeID, err := bat.AddVolume(ctx, "", "", "", &bat.VolumeOptions{
		Size: 1,
	})

	if err != nil {
		t.Fatalf("Unable to add volume :%v", err)
	}

	defer func() {
	}()

	err = bat.AttachVolume(ctx, "", instanceID, volumeID)
	if err != nil {
		t.Fatalf("Unable to attach volume %s to instance %s : %v",
			volumeID, instanceID, err)
	}

	err = bat.DeleteInstance(ctx, "", instanceID)
	if err != nil {
		t.Errorf("Failed to delete instance %s : %v", instanceID, err)
	}
	instanceID = ""

	err = bat.WaitForVolumeStatus(ctx, "", volumeID, "available")
	if err != nil {
		t.Errorf("Timed out waiting for volume %s to become available : %v",
			volumeID, err)
	}
}

// Check that a deleting a detaching volume fails
//
// Create an instance and a volume, attach the volume to the instance and wait
// for it to transition to available.  Detach and delete the volume before the
// state of the volume has transitioned back to available.
//
// Volume and instance should be created correctly and the volume should be
// attached without issue.  The first attempt to delete the detaching volume
// may fail, but the subsequent attempt made after the volume has become
// available again, should succeed.
func TestDeleteBeforeDetached(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instanceID := createVMInstance(ctx, t, "")
	var volumeID string
	defer func() {
		err := bat.DeleteInstanceAndWait(ctx, "", instanceID)
		if err != nil {
			t.Errorf("Failed to delete instance %s : %v",
				instanceID, err)
		}
		if volumeID != "" {
			err = bat.DeleteVolume(ctx, "", volumeID)
			if err != nil {
				t.Errorf("Failed to delete volume %s : %v", volumeID, err)
			}
		}
	}()

	volumeID, err := bat.AddVolume(ctx, "", "", "", &bat.VolumeOptions{
		Size: 1,
	})

	if err != nil {
		t.Fatalf("Unable to add volume :%v", err)
	}

	err = bat.AttachVolumeAndWait(ctx, "", instanceID, volumeID)
	if err != nil {
		t.Fatalf("Unable to attach volume %s to instance %s : %v",
			volumeID, instanceID, err)
	}

	err = bat.DetachVolume(ctx, "", volumeID)
	if err != nil {
		t.Fatalf("Unable to detach volume %s from instance %s : %v",
			volumeID, instanceID, err)
	}

	err = bat.DeleteVolume(ctx, "", volumeID)
	if err != nil {
		err = bat.WaitForVolumeStatus(ctx, "", volumeID, "available")
		if err != nil {
			t.Fatalf("Volume %s status not available", volumeID)
		}
		err = bat.DeleteVolume(ctx, "", volumeID)
		if err != nil {
			t.Fatalf("Failed to delete volume %s : %v", volumeID, err)
		}
	}

	volumeID = ""
}

// Check that we can attach and detach volumes
//
// Create an instance and a volume.  Attach the volume and wait for its status to
// change to in-use.  Detach the volume and wait for its status to change back to
// available.  Delete the instance and volume.
//
// The instance and volume should be created correctly.  The volume should be attached
// and detached without issue and its status should transition from available to in-use
// and back to available.  The instance and the volume should be deleted without issue.
func TestAttachDetach(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instanceID := createVMInstance(ctx, t, "")
	var volumeID string
	defer func() {
		err := bat.DeleteInstanceAndWait(ctx, "", instanceID)
		if err != nil {
			t.Errorf("Failed to delete instance %s : %v",
				instanceID, err)
		}
		if volumeID != "" {
			err = bat.DeleteVolume(ctx, "", volumeID)
			if err != nil {
				t.Errorf("Failed to delete volume %s : %v", volumeID, err)
			}
		}

	}()

	volumeID, err := bat.AddVolume(ctx, "", "", "", &bat.VolumeOptions{
		Size: 1,
	})

	if err != nil {
		t.Fatalf("Unable to add volume :%v", err)
	}

	err = bat.AttachVolumeAndWait(ctx, "", instanceID, volumeID)
	if err != nil {
		t.Fatalf("Unable to attach volume %s to instance %s : %v",
			volumeID, instanceID, err)
	}

	err = bat.DetachVolumeAndWait(ctx, "", volumeID)
	if err != nil {
		t.Fatalf("Unable to detach volume %s from instance %s : %v",
			volumeID, instanceID, err)
	}
}

// Check that we cannot attach to a running container
//
// Create a container instance and a volume.  Attach the volume.  Delete the container
// and the volume.
//
// The container and the volume should be correctly created.  The attempt to
// attach the volume to the container should initially succeed, but the volume should
// quickly return to available state as it cannot be attached to a container.  The
// container and the volume should be deleted correctly.
func TestAttachToRunningContainer(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instanceID := createContainerInstance(ctx, t, "")
	var volumeID string
	defer func() {
		err := bat.DeleteInstanceAndWait(ctx, "", instanceID)
		if err != nil {
			t.Errorf("Failed to delete instance %s : %v",
				instanceID, err)
		}
		if volumeID != "" {
			err = bat.DeleteVolume(ctx, "", volumeID)
			if err != nil {
				t.Errorf("Failed to delete volume %s : %v", volumeID, err)
			}
		}
	}()

	volumeID, err := bat.AddVolume(ctx, "", "", "", &bat.VolumeOptions{
		Size: 1,
	})

	if err != nil {
		t.Fatalf("Unable to add volume :%v", err)
	}

	err = bat.AttachVolume(ctx, "", instanceID, volumeID)
	if err != nil {
		t.Fatalf("Failed to attach volume %s to instance %s : %v",
			volumeID, instanceID, err)
	}

	err = bat.WaitForVolumeStatus(ctx, "", volumeID, "available")
	if err != nil {
		t.Fatalf("An error occurred waiting for volume %s to become available : %v",
			volumeID, err)
	}
}

// Check that we cannot attach a volume to two instances
//
// Create two VM instances and a volume.  Attach the volume to both instances.
// Delete the instances and the volume
//
// The instances and the volume should be correctly created.  However, the attempt to
// attach the volume to the second instance should fail.  The volume and the instances
// should be deleted without any problems.
func TestDoubleAttach(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	var volumeID string

	defer func() {
		if volumeID != "" {
			err := bat.DeleteVolume(ctx, "", volumeID)
			if err != nil {
				t.Errorf("Failed to delete volume %s : %v", volumeID, err)
			}
		}
	}()

	instanceID1 := createVMInstance(ctx, t, "")
	defer func() {
		err := bat.DeleteInstanceAndWait(ctx, "", instanceID1)
		if err != nil {
			t.Errorf("Failed to delete instance %s : %v",
				instanceID1, err)
		}
	}()

	instanceID2 := createVMInstance(ctx, t, "")
	defer func() {
		err := bat.DeleteInstance(ctx, "", instanceID2)
		if err != nil {
			t.Errorf("Failed to delete instance %s : %v",
				instanceID2, err)
		}
	}()

	volumeID, err := bat.AddVolume(ctx, "", "", "", &bat.VolumeOptions{
		Size: 1,
	})

	if err != nil {
		t.Fatalf("Unable to add volume :%v", err)
	}

	err = bat.AttachVolume(ctx, "", instanceID1, volumeID)
	if err != nil {
		t.Fatalf("Unable to attach volume %s to instance %s : %v",
			volumeID, instanceID1, err)
	}

	err = bat.AttachVolume(ctx, "", instanceID2, volumeID)
	if err == nil {
		t.Fatalf("Attempt to attach volume %s to instance %s should have failed",
			volumeID, instanceID2)
	} else if err == context.Canceled {
		t.Fatalf("Attempt to attach volume %s to instance %s timed out : %v",
			volumeID, instanceID2, err)
	}

	err = bat.WaitForVolumeStatus(ctx, "", volumeID, "in-use")
	if err != nil {
		t.Fatalf("Volume %s did not attach correctly : %v", volumeID, err)
	}

	err = bat.DetachVolumeAndWait(ctx, "", volumeID)
	if err != nil {
		t.Fatalf("Volume %s did not detach correctly : %v", volumeID, err)
	}
}

// Check we can attach a volume to a stopped instance
//
// Boot a VM which has no data volumes attached and stop the instance.  Create
// and attach a volume.  Delete the instance and the volume.
//
// The instance should be created and stopped successfully.  The volume should be
// created and attached successfully.  The volume and instance should be deleted
// without error.
func TestAttachToStoppedInstance(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instanceID := createVMInstance(ctx, t, "")
	var volumeID string
	defer func() {
		err := bat.DeleteInstanceAndWait(ctx, "", instanceID)
		if err != nil {
			t.Errorf("Failed to delete instance %s : %v",
				instanceID, err)
		}
		if volumeID != "" {
			err = bat.DeleteVolume(ctx, "", volumeID)
			if err != nil {
				t.Errorf("Failed to delete volume %s : %v", volumeID, err)
			}
		}
	}()

	err := bat.StopInstanceAndWait(ctx, "", instanceID)
	if err != nil {
		t.Fatalf("Failed to stop instance %s : %v", instanceID, err)
	}

	volumeID, err = bat.AddVolume(ctx, "", "", "", &bat.VolumeOptions{
		Size: 1,
	})

	if err != nil {
		t.Fatalf("Unable to add volume :%v", err)
	}

	err = bat.AttachVolumeAndWait(ctx, "", instanceID, volumeID)
	if err != nil {
		t.Fatalf("Unable to attach volume %s to instance %s : %v",
			volumeID, instanceID, err)
	}
}

// Check that volumes created from image work.
//
// Create a volume from an image and check that the sizes match.
//
// The created volume should be the same size as the image to the nearest GiB
// rounded up.
func TestCreateVolumeFromImage(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	options := bat.ImageOptions{
		Name: "test-image",
	}

	image, err := bat.AddRandomImage(ctx, "", 10, &options)
	if err != nil {
		t.Fatalf("Unable to add image %v", err)
	}

	volumeUUID, err := bat.AddVolume(ctx, "", image.ID, "image", &bat.VolumeOptions{})
	if err != nil {
		t.Error(err)
	}

	volume, err := bat.GetVolume(ctx, "", volumeUUID)
	if err != nil {
		t.Error(err)
	}

	if volume.Size != int(math.Ceil(float64(image.SizeBytes)/(1024*1024*1024))) {
		t.Errorf("Expected created volume same as image size: %v bytes vs %v GiB", image.SizeBytes, volume.Size)
	}

	err = bat.DeleteVolume(ctx, "", volumeUUID)
	if err != nil {
		t.Fatal(err)
	}

	err = bat.DeleteImage(ctx, "", image.ID)
	if err != nil {
		t.Fatal(err)
	}
}

// Check that volumes created from volumes work.
//
// Create a volume from another volume and check that the sizes match.
//
// The created volume should be the same size as the source volume.
func TestCreateVolumeFromVolume(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	sourceVolumeUUID, err := bat.AddVolume(ctx, "", "", "", &bat.VolumeOptions{Size: 2})
	if err != nil {
		t.Fatal(err)
	}

	sourceVolume, err := bat.GetVolume(ctx, "", sourceVolumeUUID)
	if err != nil {
		t.Error(err)
	}

	volumeUUID, err := bat.AddVolume(ctx, "", sourceVolumeUUID, "volume", &bat.VolumeOptions{})
	if err != nil {
		t.Error(err)
	}

	volume, err := bat.GetVolume(ctx, "", volumeUUID)
	if err != nil {
		t.Error(err)
	}

	if volume.Size != sourceVolume.Size {
		t.Errorf("Expected volume sizes to match after clone: %v GiB vs %v", volume.Size, sourceVolume.Size)
	}

	err = bat.DeleteVolume(ctx, "", sourceVolumeUUID)
	if err != nil {
		t.Error(err)
	}

	err = bat.DeleteVolume(ctx, "", volumeUUID)
	if err != nil {
		t.Error(err)
	}
}
