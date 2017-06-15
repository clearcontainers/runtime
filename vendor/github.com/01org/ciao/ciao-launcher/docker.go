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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"sync"
	"syscall"
	"time"

	"context"

	storage "github.com/01org/ciao/ciao-storage"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/network"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

const volumesDir = "volumes"

type dockerMounter struct{}

func (m dockerMounter) Mount(source, destination string) error {
	return exec.Command("mount", source, destination).Run()
}

func (m dockerMounter) Unmount(destination string, flags int) error {
	return syscall.Unmount(destination, flags)
}

type docker struct {
	cfg            *vmConfig
	instanceDir    string
	dockerID       string
	prevCPUTime    int64
	prevSampleTime time.Time
	storageDriver  storage.BlockDriver
	mount          mounter
	cli            containerManager
}

type mounter interface {
	Mount(source, destination string) error
	Unmount(path string, flags int) error
}

// BUG(markus): We shouldn't report ssh ports for docker instances

func getDockerClient() (cli *client.Client, err error) {
	return client.NewClient("unix:///var/run/docker.sock", "v1.22", nil,
		map[string]string{
			"User-Agent": "ciao-1.0",
		})
}

func (d *docker) init(cfg *vmConfig, instanceDir string) {
	d.cfg = cfg
	d.instanceDir = instanceDir
	if d.mount == nil {
		d.mount = dockerMounter{}
	}
	_ = d.initDockerClient()
}

func (d *docker) initDockerClient() error {
	if d.cli != nil {
		return nil
	}
	cli, err := getDockerClient()
	if err != nil {
		return fmt.Errorf("Unable to init docker client: %v", err)
	}
	d.cli = cli
	return nil
}

func (d *docker) checkBackingImage() error {
	glog.Infof("Checking backing docker image %s", d.cfg.DockerImage)

	args := filters.NewArgs()
	images, err := d.cli.ImageList(context.Background(),
		types.ImageListOptions{
			MatchName: d.cfg.DockerImage,
			All:       false,
			Filters:   args,
		})

	if err != nil {
		glog.Infof("Called to ImageList for %s failed: %v", d.cfg.DockerImage, err)
		return err
	}

	if len(images) == 0 {
		glog.Infof("Docker Image not found %s", d.cfg.DockerImage)
		return errImageNotFound
	}

	glog.Infof("Docker Image %s is present on node", d.cfg.DockerImage)

	return nil
}

func (d *docker) ensureBackingImage() error {
	glog.Infof("Downloading backing docker image %s", d.cfg.DockerImage)

	err := d.initDockerClient()
	if err != nil {
		return err
	}

	err = d.checkBackingImage()
	if err == nil {
		return nil
	} else if err != errImageNotFound {
		glog.Errorf("Backing image check failed")
		return err
	}

	glog.Infof("Backing image not found.  Trying to download")

	prog, err := d.cli.ImagePull(context.Background(), types.ImagePullOptions{ImageID: d.cfg.DockerImage}, nil)
	if err != nil {
		glog.Errorf("Unable to download image %s: %v\n", d.cfg.DockerImage, err)
		return err

	}
	defer func() { _ = prog.Close() }()

	dec := json.NewDecoder(prog)
	var msg jsonmessage.JSONMessage
	err = dec.Decode(&msg)
	for err == nil {
		if msg.Error != nil {
			err = msg.Error
			break
		}

		err = dec.Decode(&msg)
	}

	if err != nil && err != io.EOF {
		glog.Errorf("Unable to download image : %v\n", err)
		return err
	}

	return nil
}

func (d *docker) createConfigs(bridge string, userData, metaData []byte, volumes []string) (config *container.Config,
	hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig) {

	var hostname string
	var cmd []string
	md := &struct {
		Hostname string `json:"hostname"`
	}{}
	err := json.Unmarshal(metaData, md)
	if err != nil {
		glog.Info("Start command does not contain hostname. Setting to instance UUID")
		hostname = d.cfg.Instance
	} else {
		glog.Infof("Found hostname %s", md.Hostname)
		hostname = md.Hostname
	}

	ud := &struct {
		Cmds [][]string `yaml:"runcmd"`
	}{}
	err = yaml.Unmarshal(userData, ud)
	if err != nil {
		glog.Info("Start command does not contain a run command")
	} else {
		if len(ud.Cmds) >= 1 {
			cmd = ud.Cmds[0]
			if len(ud.Cmds) > 1 {
				glog.Warningf("Only one command supported.  Found %d in userdata", len(ud.Cmds))
			}
		}
	}

	config = &container.Config{
		Hostname: hostname,
		Image:    d.cfg.DockerImage,
		Cmd:      cmd,
	}

	hostConfig = &container.HostConfig{Binds: volumes}

	if d.cfg.Mem > 0 {
		// Docker memory limit is in bytes.
		hostConfig.Memory = int64(1024 * 1024 * d.cfg.Mem)
	}

	if d.cfg.Cpus > 0 {
		// CFS quota period - default to 100ms.
		hostConfig.CPUPeriod = 100 * 1000
		hostConfig.CPUQuota = hostConfig.CPUPeriod * int64(d.cfg.Cpus)
	}

	networkConfig = &network.NetworkingConfig{}
	if bridge != "" {
		config.MacAddress = d.cfg.VnicMAC
		hostConfig.NetworkMode = container.NetworkMode(bridge)
		networkConfig.EndpointsConfig = map[string]*network.EndpointSettings{
			bridge: {
				IPAMConfig: &network.EndpointIPAMConfig{
					IPv4Address: d.cfg.VnicIP,
				},
			},
		}
	}

	return
}

func (d *docker) umountVolumes(vols []volumeConfig) {
	for _, vol := range vols {
		vd := path.Join(d.instanceDir, volumesDir, vol.UUID)
		if err := d.mount.Unmount(vd, 0); err != nil {
			glog.Warningf("Unable to unmount %s: %v", vd, err)
			continue
		}
		glog.Infof("%s successfully unmounted", vol.UUID)
	}
}

func (d *docker) unmapVolumes() {
	for _, vol := range d.cfg.Volumes {
		if err := d.storageDriver.UnmapVolumeFromNode(vol.UUID); err != nil {
			glog.Warningf("Unable to unmap %s: %v", vol.UUID, err)
			continue
		}
		glog.Infof("Unmapping volume %s", vol.UUID)
	}
}

func (d *docker) mapAndMountVolumes() error {
	for mapped, vol := range d.cfg.Volumes {
		var devName string
		var err error
		if devName, err = d.storageDriver.MapVolumeToNode(vol.UUID); err != nil {
			d.umountVolumes(d.cfg.Volumes[:mapped])
			return fmt.Errorf("Unable to map (%s) %v", vol.UUID, err)
		}

		vd := path.Join(d.instanceDir, volumesDir, vol.UUID)
		if err = d.mount.Mount(devName, vd); err != nil {
			d.umountVolumes(d.cfg.Volumes[:mapped])
			return fmt.Errorf("Unable to mount (%s) %v", vol.UUID, err)
		}
	}

	return nil
}

func (d *docker) prepareVolumes() ([]string, error) {
	var err error
	volumes := make([]string, len(d.cfg.Volumes))

	for _, vol := range d.cfg.Volumes {
		if vol.Bootable {
			return nil, fmt.Errorf("Cannot attach bootable volumes to containers")
		}
	}
	for i, vol := range d.cfg.Volumes {
		vd := path.Join(d.instanceDir, volumesDir, vol.UUID)
		if err = os.MkdirAll(vd, 0777); err != nil {
			return nil, fmt.Errorf("Unable to create instances directory (%s) %v",
				instancesDir, err)
		}

		volumes[i] = fmt.Sprintf("%s:/volumes/%s", vd, vol.UUID)
	}

	return volumes, nil
}

func (d *docker) createImage(bridge string, userData, metaData []byte) error {
	err := d.initDockerClient()
	if err != nil {
		return err
	}

	volumes, err := d.prepareVolumes()
	if err != nil {
		glog.Errorf("Unable to mount container volumes %v", err)
		return err
	}

	config, hostConfig, networkConfig := d.createConfigs(bridge, userData, metaData, volumes)

	resp, err := d.cli.ContainerCreate(context.Background(), config, hostConfig, networkConfig,
		d.cfg.Instance)
	if err != nil {
		glog.Errorf("Unable to create container %v", err)
		return err
	}

	idPath := path.Join(d.instanceDir, "docker-id")
	err = ioutil.WriteFile(idPath, []byte(resp.ID), 0600)
	if err != nil {
		glog.Errorf("Unable to store docker container ID %v", err)
		_ = dockerDeleteContainer(d.cli, resp.ID, d.cfg.Instance)
		return err
	}

	d.dockerID = resp.ID

	// This value is configurable.  Need to figure out how to get it from docker.

	d.cfg.Disk = 10000

	return nil
}

func dockerDeleteContainer(cli containerManager, dockerID, instanceUUID string) error {
	err := cli.ContainerRemove(context.Background(),
		types.ContainerRemoveOptions{
			ContainerID: dockerID,
			Force:       true})
	if err != nil {
		glog.Warningf("Unable to delete docker instance %s:%s err %v",
			instanceUUID, dockerID, err)
	}

	return err
}

func (d *docker) deleteImage() error {
	if d.dockerID == "" {
		return nil
	}

	err := d.initDockerClient()
	if err != nil {
		return err
	}

	return dockerDeleteContainer(d.cli, d.dockerID, d.cfg.Instance)
}

func (d *docker) startVM(vnicName, ipAddress, cephID string) error {
	err := d.initDockerClient()
	if err != nil {
		return err
	}

	err = d.mapAndMountVolumes()
	if err != nil {
		glog.Errorf("Unable to map container volumes: %v", err)
		return err
	}

	err = d.cli.ContainerStart(context.Background(), d.dockerID)
	if err != nil {
		d.umountVolumes(d.cfg.Volumes)
		d.unmapVolumes()
		glog.Errorf("Unable to start container %v", err)
		return err
	}
	return nil
}

func dockerCommandLoop(cli containerManager, dockerChannel chan interface{}, instance, dockerID string) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	lostContainerCh := make(chan struct{})
	go func() {
		defer close(lostContainerCh)
		ret, err := cli.ContainerWait(ctx, dockerID)
		glog.Infof("Instance %s:%s exitted with code %d err %v",
			instance, dockerID, ret, err)
	}()

DONE:
	for {
		select {
		case _, _ = <-lostContainerCh:
			break DONE
		case cmd, ok := <-dockerChannel:
			if !ok {
				glog.Info("Cancelling Wait")
				cancelFunc()
				_ = <-lostContainerCh
				break DONE
			}
			switch cmd := cmd.(type) {
			case virtualizerStopCmd:
				err := cli.ContainerKill(context.Background(), dockerID, "KILL")
				if err != nil {
					glog.Errorf("Unable to stop instance %s:%s: %v", instance, dockerID, err)
				}
			case virtualizerAttachCmd:
				err := fmt.Errorf("Live Attach of volumes not supported for containers")
				cmd.responseCh <- err
			case virtualizerDetachCmd:
				err := fmt.Errorf("Live Detach of volumes not supported for containers")
				cmd.responseCh <- err
			}
		}
	}
	cancelFunc()

	glog.Infof("Docker Instance %s:%s shut down", instance, dockerID)
}

func dockerConnect(cli containerManager, dockerChannel chan interface{}, instance,
	dockerID string, closedCh chan struct{}, connectedCh chan struct{},
	wg *sync.WaitGroup, boot bool) {

	defer func() {
		if closedCh != nil {
			close(closedCh)
		}
		glog.Infof("Monitor function for %s exitting", instance)
		wg.Done()
	}()

	// BUG(markus): Need a way to cancel this.  Can't do this until we have contexts

	con, err := cli.ContainerInspect(context.Background(), dockerID)
	if err != nil {
		glog.Errorf("Unable to determine status of instance %s:%s: %v", instance, dockerID, err)
		return
	}

	if !con.State.Running && !con.State.Paused && !con.State.Restarting {
		glog.Infof("Docker Instance %s:%s is not running", instance, dockerID)
		return
	}

	close(connectedCh)

	dockerCommandLoop(cli, dockerChannel, instance, dockerID)
}

func (d *docker) monitorVM(closedCh chan struct{}, connectedCh chan struct{},
	wg *sync.WaitGroup, boot bool) chan interface{} {

	if d.dockerID == "" {
		idPath := path.Join(d.instanceDir, "docker-id")
		data, err := ioutil.ReadFile(idPath)
		if err != nil {
			// We'll return an error later on in dockerConnect
			glog.Errorf("Unable to read docker container ID %v", err)
		} else {
			d.dockerID = string(data)
			glog.Infof("Instance UUID %s -> Docker UUID %s", d.cfg.Instance, d.dockerID)
		}
	}
	dockerChannel := make(chan interface{})
	wg.Add(1)
	go dockerConnect(d.cli, dockerChannel, d.cfg.Instance, d.dockerID, closedCh, connectedCh, wg, boot)
	return dockerChannel
}

func (d *docker) computeInstanceDiskspace() int {
	if d.dockerID == "" {
		return -1
	}

	err := d.initDockerClient()
	if err != nil {
		return -1
	}

	con, _, err := d.cli.ContainerInspectWithRaw(context.Background(), d.dockerID, true)
	if err != nil {
		glog.Errorf("Unable to determine status of instance %s:%s: %v", d.cfg.Instance,
			d.dockerID, err)
		return -1
	}

	if con.SizeRootFs == nil {
		return -1
	}

	return int(*con.SizeRootFs / (1024 * 1024))
}

func (d *docker) stats() (disk, memory, cpu int) {
	disk = d.computeInstanceDiskspace()
	memory = -1
	cpu = -1

	if d.cfg == nil {
		return
	}

	err := d.initDockerClient()
	if err != nil {
		glog.Errorf("Unable to get docker client: %v", err)
		return
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := d.cli.ContainerStats(ctx, d.dockerID, false)
	cancelFunc()
	if err != nil {
		glog.Errorf("Unable to get stats from container: %s:%s %v", d.cfg.Instance, d.dockerID, err)
		return
	}
	defer func() { _ = resp.Close() }()

	var stats types.Stats
	err = json.NewDecoder(resp).Decode(&stats)
	if err != nil {
		glog.Errorf("Unable to get stats from container: %s:%s %v", d.cfg.Instance, d.dockerID, err)
		return
	}

	// The value from docker comes in bytes
	memory = int(stats.MemoryStats.Usage / 1024 / 1024)

	cpuTime := int64(stats.CPUStats.CPUUsage.TotalUsage)
	now := time.Now()
	if d.prevCPUTime != -1 {
		cpu = int((100 * (cpuTime - d.prevCPUTime) /
			now.Sub(d.prevSampleTime).Nanoseconds()))
		if d.cfg.Cpus > 1 {
			cpu /= d.cfg.Cpus
		}
		// if glog.V(1) {
		//     glog.Infof("cpu %d%%\n", cpu)
		// }
	}
	d.prevCPUTime = cpuTime
	d.prevSampleTime = now

	return
}

func (d *docker) connected() {
	d.prevCPUTime = -1
}

func (d *docker) lostVM() {
	d.prevCPUTime = -1

	d.umountVolumes(d.cfg.Volumes)
}
