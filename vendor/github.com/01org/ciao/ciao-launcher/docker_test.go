/*
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
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"

	storage "github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/testutil"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
)

type dockerTestMounter struct {
	mounts map[string]string
}

func (m dockerTestMounter) Mount(source, destination string) error {
	m.mounts[path.Base(destination)] = source
	return nil
}

func (m dockerTestMounter) Unmount(destination string, flags int) error {
	delete(m.mounts, path.Base(destination))
	return nil
}

type dockerTestStorage struct {
	root      string
	failAfter int
	count     int
}

func (s dockerTestStorage) MapVolumeToNode(volumeUUID string) (string, error) {
	if s.failAfter != -1 && s.failAfter >= s.count {
		return "", fmt.Errorf("MapVolumeToNode failure forced")
	}
	s.count++

	return "", nil
}

func (s dockerTestStorage) CreateBlockDevice(volumeUUID string, image string, sizeGB int) (storage.BlockDevice, error) {
	return storage.BlockDevice{}, nil
}

func (s dockerTestStorage) CreateBlockDeviceFromSnapshot(volumeUUID string, snapshotID string) (storage.BlockDevice, error) {
	return storage.BlockDevice{}, nil
}

func (s dockerTestStorage) CreateBlockDeviceSnapshot(volumeUUID string, snapshotID string) error {
	return nil
}

func (s dockerTestStorage) DeleteBlockDevice(string) error {
	return nil
}

func (s dockerTestStorage) DeleteBlockDeviceSnapshot(volumeUUID string, snapshotID string) error {
	return nil
}

func (s dockerTestStorage) UnmapVolumeFromNode(volumeUUID string) error {
	return nil
}

func (s dockerTestStorage) GetVolumeMapping() (map[string][]string, error) {
	return nil, nil
}

func (s dockerTestStorage) CopyBlockDevice(volumeUUID string) (storage.BlockDevice, error) {
	return storage.BlockDevice{}, nil
}

func (s dockerTestStorage) GetBlockDeviceSize(volumeUUID string) (uint64, error) {
	return 0, nil
}

func (s dockerTestStorage) IsValidSnapshotUUID(string) error {
	return nil
}

func (s dockerTestStorage) Resize(string, int) (int, error) {
	return 0, nil
}

type dockerTestClient struct {
	err               error
	images            []types.Image
	imagePullProgress bytes.Buffer
	config            *container.Config
	hostConfig        *container.HostConfig
	networkConfig     *network.NetworkingConfig
	containerWaitCh   chan struct{}
}

func (d *dockerTestClient) ImageList(context.Context, types.ImageListOptions) ([]types.Image, error) {
	if d.err != nil {
		return nil, d.err
	}

	return d.images, nil
}

func (d *dockerTestClient) ImagePull(context.Context, types.ImagePullOptions,
	client.RequestPrivilegeFunc) (io.ReadCloser, error) {
	if d.err != nil {
		return nil, d.err
	}

	return ioutil.NopCloser(&d.imagePullProgress), nil
}

func (d *dockerTestClient) ContainerCreate(ctx context.Context, config *container.Config,
	hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig,
	instance string) (types.ContainerCreateResponse, error) {
	if d.err != nil {
		return types.ContainerCreateResponse{}, d.err
	}
	d.config = config
	d.hostConfig = hostConfig
	d.networkConfig = networkConfig
	return types.ContainerCreateResponse{ID: testutil.InstanceUUID}, nil
}

func (d *dockerTestClient) ContainerRemove(context.Context, types.ContainerRemoveOptions) error {
	if d.err != nil {
		return d.err
	}
	d.config = nil
	d.hostConfig = nil
	d.networkConfig = nil

	return nil
}

func (d *dockerTestClient) ContainerStart(context.Context, string) error {
	return nil
}

func (d *dockerTestClient) ContainerInspectWithRaw(context.Context, string, bool) (types.ContainerJSON, []byte, error) {
	i := int64(10000000)
	return types.ContainerJSON{ContainerJSONBase: &types.ContainerJSONBase{SizeRootFs: &i}}, nil, nil
}

func (d *dockerTestClient) ContainerInspect(context.Context, string) (types.ContainerJSON, error) {
	return types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Running: true,
			},
		},
	}, nil
}

func (d *dockerTestClient) ContainerStats(context.Context, string, bool) (io.ReadCloser, error) {
	var buf bytes.Buffer

	buf.WriteString(`
{
  "cpu_stats" : {
    "cpu_usage" : {
      "total_usage" : 100000000
    }
  },
  "memory_stats" : {
     "usage" : 104857600
  }
}`)

	return ioutil.NopCloser(&buf), nil
}

func (d *dockerTestClient) ContainerKill(context.Context, string, string) error {
	close(d.containerWaitCh)
	return nil
}

func (d *dockerTestClient) ContainerWait(ctx context.Context, id string) (int, error) {
	select {
	case <-d.containerWaitCh:
	case <-ctx.Done():
	}

	return 0, nil
}

// Checks that the logic of the code that mounts and unmounts ceph volumes in
// docker containers.
//
// We mount 4 volumes, check the mount commands are received correctly and check
// the correct directories are created.  We then unmount, and check that everything
// gets unmounted as expected.
//
// Calls to docker.mountVolumes and docker.unmountVolumes should succeed and the
// mounted volumes should be correctly cleaned up.
func TestDockerMountUnmount(t *testing.T) {
	root, err := ioutil.TempDir("", "mount-unmount")
	if err != nil {
		t.Fatalf("Unable to create temporary directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(root) }()

	instanceDir := path.Join(root, "volumes")

	s := dockerTestStorage{
		root:      root,
		failAfter: -1,
	}
	mounts := make(map[string]string)
	d := docker{
		cfg: &vmConfig{
			Volumes: []volumeConfig{
				{UUID: "92a1e4fa-8448-4260-adb1-4d2dd816cc7c"},
				{UUID: "5ce2c5bf-58d9-4573-b433-05550b945866"},
				{UUID: "11773eac-6b27-4bc1-8717-02e75ae5e063"},
				{UUID: "590603fb-c73e-4efa-941e-454b5d4f9857"},
			},
		},
		instanceDir:   root,
		storageDriver: s,
		mount:         dockerTestMounter{mounts: mounts},
	}

	_, err = d.prepareVolumes()
	if err != nil {
		t.Fatalf("Unable to prepare volumes: %v", err)
	}
	err = d.mapAndMountVolumes()
	if err != nil {
		t.Fatalf("Unable to map and mount volumes: %v", err)
	}

	dirInfo, err := ioutil.ReadDir(instanceDir)
	if err != nil {
		t.Fatalf("Unable to readdir %s: %v", instanceDir, err)
	}

	if len(dirInfo) != len(d.cfg.Volumes) {
		t.Fatalf("Unexpected number of volumes directories.  Found %d, expected %d",
			len(dirInfo), len(d.cfg.Volumes))
	}

	if len(dirInfo) != len(mounts) {
		t.Fatalf("Unexpected number of volumes mounted.  Found %d, expected %d",
			len(dirInfo), len(mounts))
	}

	for _, vol := range d.cfg.Volumes {
		var i int
		for i = 0; i < len(dirInfo); i++ {
			if vol.UUID == dirInfo[i].Name() {
				break
			}
		}
		if i == len(dirInfo) {
			t.Fatalf("%s not mounted", vol.UUID)
		}
		if _, ok := mounts[vol.UUID]; !ok {
			t.Fatalf("%s does not seem to have been mounted", vol.UUID)
		}
	}

	d.umountVolumes(d.cfg.Volumes)
	if len(mounts) != 0 {
		t.Fatalf("Not all volumes have been unmounted")
	}
}

// Checks that everything is cleaned up correctly when a call to
// docker.mountVolumes fails.
//
// We call docker.mountVolumes with 4 volumes but arrange for the call to fail
// after the second volume has been created.
//
// docker.mountVolumes should fail but everything should be cleaned up despite
// the failure.
func TestDockerBadMount(t *testing.T) {
	root, err := ioutil.TempDir("", "mount-unmount")
	if err != nil {
		t.Fatalf("Unable to create temporary directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(root) }()

	s := dockerTestStorage{
		root:      root,
		failAfter: 2,
	}

	mounts := make(map[string]string)
	d := docker{
		cfg: &vmConfig{
			Volumes: []volumeConfig{
				{UUID: "92a1e4fa-8448-4260-adb1-4d2dd816cc7c"},
				{UUID: "5ce2c5bf-58d9-4573-b433-05550b945866"},
				{UUID: "11773eac-6b27-4bc1-8717-02e75ae5e063"},
				{UUID: "590603fb-c73e-4efa-941e-454b5d4f9857"},
			},
		},
		instanceDir:   root,
		storageDriver: s,
		mount:         dockerTestMounter{mounts: mounts},
	}

	_, err = d.prepareVolumes()
	if err != nil {
		t.Fatalf("Unable to prepare volumes: %v", err)
	}

	err = d.mapAndMountVolumes()
	if err == nil {
		t.Fatal("d.mountVolumes was expected to fail")
	}

	if len(mounts) != 0 {
		t.Fatal("mounts not cleaned up correctly")
	}
}

// Check that the docker.checkBackingImage works correctly
//
// We call checkBackingImage 3 times.  Each time checkBackingImage calls
// containerManager.ImageList.  The first call to ImageList returns an
// empty list of images, the second returns an error and the third returns
// no error and a slice of images.
//
// The first call should fail with errImageNotFound, the second with the
// error we create and the third should succeed.
func TestDockerCheckBackingImage(t *testing.T) {
	tc := &dockerTestClient{}
	d := &docker{cfg: &vmConfig{}, cli: tc}
	err := d.checkBackingImage()
	if err != errImageNotFound {
		t.Errorf("Expected image not found error")
	}

	tc.err = fmt.Errorf("ImageList fail forced")
	err = d.checkBackingImage()
	if err == nil {
		t.Errorf("Expected checkBackingImage to fail")
	}

	tc.err = nil
	tc.images = make([]types.Image, 1)
	err = d.checkBackingImage()
	if err != nil {
		t.Errorf("Expected checkBackingImage to succeeded but it failed with : %v", err)
	}
}

// Check that docker.downloaBackingImage works correctly.
//
// We call ensureBackingImage 4 times.  The first time we provide it with some
// valid progress information via containerManager.downloadImage, the second
// time we provide progress information that contains an error, the third time
// we force downloadImage to return an error and the fourth time downloadImage
// doesn't return any errors but also doesn't return any progress buffers.
//
// The first and last call to downloadBackingImage should succeed.  The second
// and third should fail.
func TestDockerEnsureBackingImage(t *testing.T) {
	tc := &dockerTestClient{}
	d := &docker{cfg: &vmConfig{}, cli: tc}

	var msg jsonmessage.JSONMessage
	enc := json.NewEncoder(&tc.imagePullProgress)
	for i := 0; i < 5; i++ {
		if err := enc.Encode(&msg); err != nil {
			t.Fatalf("Failed to encode JSONMessage : %v", err)
		}
	}

	err := d.ensureBackingImage()
	if err != nil {
		t.Errorf("Failed to download backing image : %v", err)
	}

	tc.imagePullProgress.Reset()
	msg.Error = &jsonmessage.JSONError{
		Code:    1,
		Message: "Forced error retrieving image",
	}
	if err := enc.Encode(&msg); err != nil {
		t.Fatalf("Failed to encode JSONMessage : %v", err)
	}
	err = d.ensureBackingImage()
	if err == nil {
		t.Errorf("Error expected downloading backing image")
	}

	tc.err = fmt.Errorf("Force failing of downloading backing image")
	err = d.ensureBackingImage()
	if err == nil {
		t.Errorf("Error expected downloading backing image")
	}

	// TODO:  Need to decide if this is the correct behaviour.  Here we
	// are testing the case where we get no progress information from the
	// docker daemon.  We currently treat this as success but this might
	// not be correct. As there are no docs it's hard to know.

	tc.imagePullProgress.Reset()
	tc.err = nil
	err = d.ensureBackingImage()
	if err != nil {
		t.Errorf("Error downloading image with no progress : %v", err)
	}
}

// Check that docker.deleteImage works correctly.
//
// We call deleteImage 3 times, the first time with a blank dockerID,
// the second time with a non blank docker ID, and the third time,
// we arrange for containerManager.ContainerRemove to fail.
//
// The first two calls to docker.deleteImage should succeed, the third
// should fail.
func TestDockerDeleteImage(t *testing.T) {
	tc := &dockerTestClient{}
	d := &docker{cfg: &vmConfig{}, cli: tc}

	err := d.deleteImage()
	if err != nil {
		t.Errorf("Unable to Delete Image : %v", err)
	}

	d.dockerID = testutil.InstanceUUID
	err = d.deleteImage()
	if err != nil {
		t.Errorf("Unable to Delete Image : %v", err)
	}

	tc.err = fmt.Errorf("Forcing failure of Container Remove")
	err = d.deleteImage()
	if err == nil {
		t.Errorf("deleteImage was expected to fail")
	}
}

// Verify that docker.createImage works as expected.
//
// The test calls createImage with a sample user data that contains a single
// command and some meta data that contains a host name.  The container is
// deleted after it has been successfully created.
//
// The call to createImage should succeed, the hostname and the command
// to execute should be successfully extracted and the id of the new
// container should be stored in the docker-id file.  The container should
// be deleted without error.
func TestDockerCreateImage(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ciao-docker-tests")
	if err != nil {
		t.Fatal("Unable to create temporary directory")
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	tc := &dockerTestClient{}
	d := &docker{instanceDir: tmpDir, cfg: &vmConfig{}, cli: tc}

	const cmd = `["touch", "/etc/bootdone"]`
	const ud = `
runcmd:
 - ` + cmd + `
`
	const md = `
{
  "uuid": "ciao",
  "hostname": "ciao"
}
`
	err = d.createImage("", []byte(ud), []byte(md))
	if err != nil {
		t.Fatalf("Unable to create image : %v", err)
	}

	if tc.config.Hostname != "ciao" {
		t.Errorf("Unexpected hostname.  Expected ciao found %s", tc.config.Hostname)
	}

	if reflect.DeepEqual(tc.config.Cmd, cmd) {
		t.Errorf("Unexpected command.  Expected %s found %s", cmd, tc.config.Cmd)
	}

	if d.dockerID != testutil.InstanceUUID {
		t.Errorf("Incorrect container ID %s expected, found %s",
			testutil.InstanceUUID, d.dockerID)
	}

	readID, err := ioutil.ReadFile(path.Join(tmpDir, "docker-id"))
	if err != nil {
		t.Errorf("Unable to read docker-id file : %v", err)
	}

	readIDstr := string(bytes.TrimSpace(readID))
	if readIDstr != testutil.InstanceUUID {
		t.Errorf("Incorrect container ID read from docker-id file %s expected, found %s",
			testutil.InstanceUUID, readIDstr)
	}

	err = d.deleteImage()
	if err != nil {
		t.Errorf("Unable to delete container : %v", err)
	}
}

// Verify that docker.createImage handles errors gracefully
//
// The test calls createImage twice.  The first time it arranges for
// ContainerCreate to fail. The second time it specifies an invalid
// temporary directory, so the docker-id file does not get written.
//
// Both calls to createImage should fail and no containers should be
// created.
func TestDockerCreateImageFail(t *testing.T) {
	tc := &dockerTestClient{err: fmt.Errorf("ContainerCreate failure forced")}
	d := &docker{instanceDir: "/tmp/i/dont/exist", cfg: &vmConfig{}, cli: tc}

	for i := 0; i < 2; i++ {
		if err := d.createImage("", nil, nil); err == nil {
			t.Errorf("createImage should have failed")
		}

		if tc.config != nil || tc.hostConfig != nil || tc.networkConfig != nil ||
			d.dockerID != "" {
			t.Errorf("dockerTestClient status not clean")
		}
		tc.err = nil
	}
}

// Verify that docker.createImage handles volumes correctly
//
// The test calls createImage with two pre-configured volumes.  It checks that
// the directories in which the volumes are to be mounted have been created and
// then tries to create an image with a bootable volume.
//
// The first call to createImage should succeed and directories should be created
// for each volume.  The second call to createImage should fail as it's not
// possible to create an image with a bootable volume.
func TestDockerCreateImageWithVolumes(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ciao-docker-tests")
	if err != nil {
		t.Fatal("Unable to create temporary directory")
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	tc := &dockerTestClient{}
	d := &docker{
		instanceDir: tmpDir,
		cli:         tc,
		cfg: &vmConfig{
			Volumes: []volumeConfig{
				{UUID: "92a1e4fa-8448-4260-adb1-4d2dd816cc7c"},
				{UUID: "5ce2c5bf-58d9-4573-b433-05550b945866"},
			},
		}}

	if err := d.createImage("", nil, nil); err != nil {
		t.Fatalf("Unable to create image : %v", err)
	}

	for _, vol := range tc.hostConfig.Binds {
		volInfo := strings.Split(vol, ":")
		fi, err := os.Stat(volInfo[0])
		if err != nil {
			t.Errorf("Unable to retrieve information about volume directory : %v", err)
			continue
		}
		if !fi.IsDir() {
			t.Errorf("%s is not a directory", vol)
		}
	}

	err = d.deleteImage()
	if err != nil {
		t.Errorf("Unable to delete container : %v", err)
	}

	d.cfg.Volumes[0].Bootable = true
	if err = d.createImage("", nil, nil); err == nil {
		t.Fatalf("Attempt to create image with a bootable volume should fail")
	}
}

// Check createImage processes memory, cpu and IP resources correctly
//
// Create an image with memory, CPU and IP resources.  Delete the image.
//
// The image is correctly created, the resources are computed/allocated correctly,
// and the image is deleted correctly.
func TestDockerCreateImageWithResources(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ciao-docker-tests")
	if err != nil {
		t.Fatal("Unable to create temporary directory")
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	tc := &dockerTestClient{}
	d := &docker{instanceDir: tmpDir, cli: tc,
		cfg: &vmConfig{
			Mem:     10,
			Cpus:    2,
			VnicMAC: testutil.VNICMAC,
			VnicIP:  testutil.AgentIP,
		}}

	if err := d.createImage("bridge", nil, nil); err != nil {
		t.Fatalf("Unable to create image : %v", err)
	}

	if tc.hostConfig.Memory != int64(1024*1024*d.cfg.Mem) {
		t.Errorf("Wrong memory value %d ", tc.hostConfig.Memory)
	}

	if tc.hostConfig.CPUQuota != int64(d.cfg.Cpus)*tc.hostConfig.CPUPeriod {
		t.Errorf("Wrong CPU Quota %d ", d.cfg.Cpus)
	}

	if tc.networkConfig.EndpointsConfig["bridge"].IPAMConfig.IPv4Address !=
		testutil.AgentIP {
		t.Errorf("Wrong IP address %s ",
			tc.networkConfig.EndpointsConfig["bridge"].IPAMConfig.IPv4Address)
	}

	err = d.deleteImage()
	if err != nil {
		t.Errorf("Unable to delete container : %v", err)
	}
}

// Checks the monitorVM function works correctly.
//
// This test creates a new instance, calls monitor VM, waits for the connected
// channel to be closed and then sends the stop command to the container.
//
// The instance should be created correctly, the connected channel should be
// closed and the closedChannel should be closed after we send the virtualizerStopCmd.
func TestDockerMonitorVM(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ciao-docker-tests")
	if err != nil {
		t.Fatal("Unable to create temporary directory")
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	tc := &dockerTestClient{containerWaitCh: make(chan struct{})}
	d := &docker{instanceDir: tmpDir, dockerID: testutil.InstanceUUID, cfg: &vmConfig{}, cli: tc}

	err = d.createImage("", nil, nil)
	if err != nil {
		t.Fatalf("Unable to create image : %v", err)
	}

	defer func() {
		if d.deleteImage() != nil {
			t.Errorf("Unable to delete container : %v", err)
		}
	}()

	// Force monitorVM to re-read instance-ID

	d.dockerID = ""

	closedCh := make(chan struct{})
	connectedCh := make(chan struct{})

	var wg sync.WaitGroup

	dockerCh := d.monitorVM(closedCh, connectedCh, &wg, false)

	select {
	case <-connectedCh:
	case <-closedCh:
		t.Errorf("Failed to connect to container")
	case <-time.After(time.Second):
		t.Fatalf("Timed out waiting to connect to container")
	}

	dockerCh <- virtualizerStopCmd{}

	select {
	case <-closedCh:
	case <-time.After(time.Second):
		t.Fatalf("Timed out waiting to connect to container")
	}

	wg.Wait()
}

// Check that closing the dockerChannel quits the monitor routine.
//
// This test calls monitorVM, waits for the connectedCh channel to be closed and
// then closes the dockerCh channel returned by monitorVM, simulating a launcher
// exit.
//
// The connectedCh should be closed as expected.  The closedCh channel should then
// be closed after the test function has closed the dockerCh channel.
func TestDockerMonitorVMClose(t *testing.T) {
	tc := &dockerTestClient{containerWaitCh: make(chan struct{})}
	d := &docker{dockerID: testutil.InstanceUUID, cfg: &vmConfig{}, cli: tc}

	closedCh := make(chan struct{})
	connectedCh := make(chan struct{})

	var wg sync.WaitGroup

	dockerCh := d.monitorVM(closedCh, connectedCh, &wg, false)

	select {
	case <-connectedCh:
	case <-closedCh:
		t.Errorf("Failed to connect to container")
	case <-time.After(time.Second):
		t.Fatalf("Timed out waiting to connect to container")
	}

	close(dockerCh)

	select {
	case <-closedCh:
	case <-time.After(time.Second):
		t.Fatalf("Timed out waiting to connect to container")
	}

	wg.Wait()
}

// Check container statistics are computed correctly.
//
// Call the stats method twice.  The second call is required to retrieve cpu stats.
//
// The stats method should return the statistics provisioned in the dockerTestClient
// ContainerInspectWithRaw and ContainerStats methods.
func TestDockerStats(t *testing.T) {
	tc := &dockerTestClient{}
	d := &docker{dockerID: testutil.InstanceUUID, cfg: &vmConfig{}, cli: tc, prevCPUTime: -1}

	disk, mem, cpu := d.stats()
	if mem != 100 {
		t.Errorf("Expected memory usage of 100.  Got %d", mem)
	}

	if disk != 9 {
		t.Errorf("Expected disk usage of 9.  Got %d", disk)
	}

	if cpu != -1 {
		t.Errorf("Expected cpu usage of -1.  Got %d", cpu)
	}

	_, _, cpu = d.stats()
	if cpu != 0 {
		t.Errorf("Expected cpu usage of 0.  Got %d", cpu)
	}
}
