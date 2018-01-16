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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	vcAnnotations "github.com/containers/virtcontainers/pkg/annotations"

	kataclient "github.com/kata-containers/agent/protocols/client"
	"github.com/kata-containers/agent/protocols/grpc"

	"github.com/opencontainers/runtime-spec/specs-go"
)

var (
	defaultKataSockPathTemplate = "%s/%s/kata.sock"
	defaultKataChannel          = "io.katacontainers.channel"
	defaultKataDeviceID         = "channel0"
	defaultKataID               = "charch0"
	errorMissingGRPClient       = errors.New("Missing gRPC client")
	errorMissingOCISpec         = errors.New("Missing OCI specification")
	kataHostSharedDir           = "/tmp/kata-containers/shared/pods/"
	kataGuestSharedDir          = "/tmp/kata-containers/shared/pods/"
	mountGuest9pTag             = "kataShared"
	type9pFs                    = "9p"
	devPath                     = "/dev"
	vsockSocketScheme           = "vsock"
)

// KataAgentConfig is a structure storing information needed
// to reach the Kata Containers agent.
type KataAgentConfig struct {
	Volumes []Volume
}

type kataVSOCK struct {
	contextID uint32
	port      uint32
}

func (s *kataVSOCK) String() string {
	return fmt.Sprintf("%s://%d:%d", vsockSocketScheme, s.contextID, s.port)
}

type kataAgent struct {
	config KataAgentConfig

	client     *kataclient.AgentClient
	gRPCSocket string
	vmSocket   interface{}
}

func parseVSOCKAddr(sock string) (uint32, uint32, error) {
	sp := strings.Split(sock, ":")
	if len(sp) != 3 {
		return 0, 0, fmt.Errorf("Invalid vsock address: %s", sock)
	}
	if sp[0] != vsockSocketScheme {
		return 0, 0, fmt.Errorf("Invalid vsock URL scheme: %s", sp[0])
	}

	cid, err := strconv.ParseUint(sp[1], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("Invalid vsock cid: %s", sp[1])
	}
	port, err := strconv.ParseUint(sp[2], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("Invalid vsock port: %s", sp[2])
	}

	return uint32(cid), uint32(port), nil
}

func (k *kataAgent) validate(pod *Pod) error {
	if k.gRPCSocket == "" {
		return fmt.Errorf("Empty gRPC socket path")
	}

	cid, port, err := parseVSOCKAddr(k.gRPCSocket)
	if err != nil {
		// We need to generate a host UNIX socket path for the emulated serial port.
		k.vmSocket = Socket{
			DeviceID: defaultKataDeviceID,
			ID:       defaultKataID,
			HostPath: fmt.Sprintf(defaultKataSockPathTemplate, runStoragePath, pod.id),
			Name:     defaultKataChannel,
		}
	} else {
		// We want to go through VSOCK. The VM VSOCK endpoint will be our gRPC.
		k.vmSocket = kataVSOCK{
			contextID: cid,
			port:      port,
		}
	}

	return nil
}

func (k *kataAgent) init(pod *Pod, config interface{}) error {
	switch c := config.(type) {
	case KataAgentConfig:
		if err := k.validate(pod); err != nil {
			return err
		}
		k.config = c
	default:
		return fmt.Errorf("Invalid config type")
	}

	// Override pod agent configuration
	pod.config.AgentConfig = k.config

	client, err := kataclient.NewAgentClient(k.gRPCSocket)
	if err != nil {
		return err
	}

	k.client = client

	return nil
}

func (k *kataAgent) vmURL() (string, error) {
	switch s := k.vmSocket.(type) {
	case Socket:
		return s.HostPath, nil
	case kataVSOCK:
		return s.String(), nil
	default:
		return "", fmt.Errorf("Invalid socket type")
	}
}

func (k *kataAgent) setProxyURL(url string) error {
	k.gRPCSocket = url

	return nil
}

func (k *kataAgent) capabilities() capabilities {
	return capabilities{}
}

func (k *kataAgent) createPod(pod *Pod) error {
	for _, volume := range k.config.Volumes {
		err := pod.hypervisor.addDevice(volume, fsDev)
		if err != nil {
			return err
		}
	}

	switch s := k.vmSocket.(type) {
	case Socket:
		err := pod.hypervisor.addDevice(s, serialPortDev)
		if err != nil {
			return err
		}
	case kataVSOCK:
		// TODO Add an hypervisor vsock
	default:
		return fmt.Errorf("Invalid config type")
	}

	// Adding the shared volume.
	// This volume contains all bind mounted container bundles.
	sharedVolume := Volume{
		MountTag: mountGuest9pTag,
		HostPath: filepath.Join(kataHostSharedDir, pod.id),
	}

	if err := os.MkdirAll(sharedVolume.HostPath, dirMode); err != nil {
		return err
	}

	return pod.hypervisor.addDevice(sharedVolume, fsDev)
}

func cmdToKataProcess(cmd Cmd) (process *grpc.Process, err error) {
	var i uint64
	var extraGids []uint32

	// Number of bits used to store user+group values in
	// the gRPC "User" type.
	const grpcUserBits = 32

	i, err = strconv.ParseUint(cmd.User, 10, grpcUserBits)
	if err != nil {
		return nil, err
	}

	uid := uint32(i)

	i, err = strconv.ParseUint(cmd.PrimaryGroup, 10, grpcUserBits)
	if err != nil {
		return nil, err
	}

	gid := uint32(i)

	for _, g := range cmd.SupplementaryGroups {
		var extraGid uint64

		extraGid, err = strconv.ParseUint(g, 10, grpcUserBits)
		if err != nil {
			return nil, err
		}

		extraGids = append(extraGids, uint32(extraGid))
	}

	process = &grpc.Process{
		Terminal: cmd.Interactive,
		User: grpc.User{
			UID:            uid,
			GID:            gid,
			AdditionalGids: extraGids,
		},
		Args: cmd.Args,
		Env:  cmdEnvsToStringSlice(cmd.Envs),
		Cwd:  cmd.WorkDir,
	}

	return process, nil
}

func cmdEnvsToStringSlice(ev []EnvVar) []string {
	var env []string

	for _, e := range ev {
		pair := []string{e.Var, e.Value}
		env = append(env, strings.Join(pair, "="))
	}

	return env
}

func (k *kataAgent) exec(pod *Pod, c Container, process Process, cmd Cmd) (err error) {
	var kataProcess *grpc.Process

	kataProcess, err = cmdToKataProcess(cmd)
	if err != nil {
		return err
	}

	req := &grpc.ExecProcessRequest{
		ContainerId: c.id,
		Process:     kataProcess,
	}

	_, err = k.client.ExecProcess(context.Background(), req)
	return err
}

func (k *kataAgent) startPod(pod Pod) error {
	if k.client == nil {
		return errorMissingGRPClient
	}

	hostname := pod.config.Hostname
	if len(hostname) > maxHostnameLen {
		hostname = hostname[:maxHostnameLen]
	}

	// We mount the shared directory in a predefined location
	// in the guest.
	// This is where at least some of the host config files
	// (resolv.conf, etc...) and potentially all container
	// rootfs will reside.
	sharedVolume := &grpc.Storage{
		Source:     mountGuest9pTag,
		MountPoint: kataGuestSharedDir,
		Fstype:     type9pFs,
		Options:    []string{"trans=virtio", "nodev"},
	}

	req := &grpc.CreateSandboxRequest{
		Hostname:     hostname,
		Storages:     []*grpc.Storage{sharedVolume},
		SandboxPidns: true,
	}
	_, err := k.client.CreateSandbox(context.Background(), req)
	return err
}

func (k *kataAgent) stopPod(pod Pod) error {
	if k.client == nil {
		return errorMissingGRPClient
	}

	req := &grpc.DestroySandboxRequest{}
	_, err := k.client.DestroySandbox(context.Background(), req)
	return err
}

func appendStorageFromMounts(storage []*grpc.Storage, mounts []*Mount) []*grpc.Storage {
	for _, m := range mounts {
		s := &grpc.Storage{
			Source:     m.Source,
			MountPoint: m.Destination,
		}

		storage = append(storage, s)
	}

	return storage
}

func (k *kataAgent) createContainer(pod *Pod, c *Container) error {
	if k.client == nil {
		return errorMissingGRPClient
	}

	ociSpecJSON, ok := c.config.Annotations[vcAnnotations.ConfigJSONKey]
	if !ok {
		return errorMissingOCISpec
	}

	var ociSpec specs.Spec
	if err := json.Unmarshal([]byte(ociSpecJSON), &ociSpec); err != nil {
		return err
	}

	grpcSpec, err := grpc.OCItoGRPC(&ociSpec)
	if err != nil {
		return err
	}

	var containerStorage []*grpc.Storage

	// The rootfs storage volume represents the container rootfs
	// mount point inside the guest.
	// It can be a block based device (when using block based container
	// overlay on the host) mount or a 9pfs one (for all other overlay
	// implementations).
	rootfs := &grpc.Storage{}

	// First we need to give the OCI spec our absolute path in the guest.
	grpcSpec.Root.Path = filepath.Join(kataGuestSharedDir, pod.id, c.id, rootfsDir)

	if c.state.Fstype != "" {
		// This is a block based device rootfs.
		// driveName is the predicted virtio-block guest name (the vd* in /dev/vd*).
		driveName, err := getVirtDriveName(c.state.BlockIndex)
		if err != nil {
			return err
		}

		rootfs.Source = filepath.Join(devPath, driveName)
		rootfs.MountPoint = grpcSpec.Root.Path // Should we remove the "rootfs" suffix?
		rootfs.Fstype = c.state.Fstype

		// Add rootfs to the list of container storage.
		// We only need to do this for block based rootfs, as we
		// want the agent to mount it into the right location
		// (/tmp/kata-containers/shared/pods/podID/ctrID/
		containerStorage = append(containerStorage, rootfs)

	} else {
		// This is not a block based device rootfs.
		// We are going to bind mount it into the 9pfs
		// shared drive between the host and the guest.
		// With 9pfs we don't need to ask the agent to
		// mount the rootfs as the shared directory
		// (/tmp/kata-containers/shared/pods/) is already
		// mounted in the guest. We only need to mount the
		// rootfs from the host and it will show up in the guest.
		if err := bindMountContainerRootfs(kataHostSharedDir, pod.id, c.id, c.rootFs, false); err != nil {
			bindUnmountAllRootfs(kataHostSharedDir, *pod)
			return err
		}
	}

	// Handle container mounts
	newMounts, err := bindMountContainerMounts(kataHostSharedDir, pod.id, c.id, c.mounts)
	if err != nil {
		bindUnmountAllRootfs(kataHostSharedDir, *pod)
		return err
	}
	containerStorage = appendStorageFromMounts(containerStorage, newMounts)

	// Append container mounts for block devices passed with --device.
	for _, device := range c.devices {
		d, ok := device.(*BlockDevice)

		if !ok {
			continue
		}

		deviceStorage := &grpc.Storage{
			Source:     d.VirtPath,
			MountPoint: d.DeviceInfo.ContainerPath,
		}

		containerStorage = append(containerStorage, deviceStorage)
	}

	req := &grpc.CreateContainerRequest{
		ContainerId: c.id,
		Storages:    containerStorage,
		OCI:         grpcSpec,
	}

	_, err = k.client.CreateContainer(context.Background(), req)
	return err
}

func (k *kataAgent) startContainer(pod Pod, c Container) error {
	if k.client == nil {
		return errorMissingGRPClient
	}

	req := &grpc.StartContainerRequest{
		ContainerId: c.id,
	}

	_, err := k.client.StartContainer(context.Background(), req)
	if err != nil {
		return err
	}

	// The Kata shim wants to be signaled when the init container
	// is created. Sending the signal for all containers is harmless.
	return signalShim(c.process.Pid, syscall.SIGUSR1)
}

func (k *kataAgent) stopContainer(pod Pod, c Container) error {
	req := &grpc.RemoveContainerRequest{
		ContainerId: c.id,
	}

	_, err := k.client.RemoveContainer(context.Background(), req)
	return err
}

func (k *kataAgent) killContainer(pod Pod, c Container, signal syscall.Signal, all bool) error {
	req := &grpc.SignalProcessRequest{
		ContainerId: c.id,
		ExecId:      c.process.Token,
		Signal:      uint32(signal),
	}

	_, err := k.client.SignalProcess(context.Background(), req)
	return err
}

func (k *kataAgent) processListContainer(pod Pod, c Container, options ProcessListOptions) (ProcessList, error) {
	return nil, nil
}
