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

package hyperstart

import (
	"syscall"
)

// Defines all available commands to communicate with hyperstart agent.
const (
	VersionCode = iota
	StartPodCode
	GetPodDeprecatedCode
	StopPodDeprecatedCode
	DestroyPodCode
	RestartContainerDeprecatedCode
	ExecCmdCode
	FinishCmdDeprecatedCode
	ReadyCode
	AckCode
	ErrorCode
	WinsizeCode
	PingCode
	FinishPodDeprecatedCode
	NextCode
	WriteFileCode
	ReadFileCode
	NewContainerCode
	KillContainerCode
	OnlineCPUMemCode
	SetupInterfaceCode
	SetupRouteCode
	RemoveContainerCode
	ProcessAsyncEventCode
)

// FileCommand is the structure corresponding to the format expected by
// hyperstart to interact with files.
type FileCommand struct {
	Container string `json:"container"`
	File      string `json:"file"`
}

// KillCommand is the structure corresponding to the format expected by
// hyperstart to kill a container on the guest.
type KillCommand struct {
	Container    string         `json:"container"`
	Signal       syscall.Signal `json:"signal"`
	AllProcesses bool           `json:"allProcesses"`
}

// ExecCommand is the structure corresponding to the format expected by
// hyperstart to execute a command on the guest.
type ExecCommand struct {
	Container string  `json:"container,omitempty"`
	Process   Process `json:"process"`
}

// RemoveCommand is the structure corresponding to the format expected by
// hyperstart to remove a container on the guest.
type RemoveCommand struct {
	Container string `json:"container"`
}

// PAECommand is the structure hyperstart can expects to
// receive after a process has been started/executed on a container.
type PAECommand struct {
	Container string `json:"container"`
	Process   string `json:"process"`
	Event     string `json:"event"`
	Info      string `json:"info,omitempty"`
	Status    int    `json:"status,omitempty"`
}

// DecodedMessage is the structure holding messages coming from CTL channel.
type DecodedMessage struct {
	Code    uint32
	Message []byte
}

// TtyMessage is the structure holding messages coming from TTY channel.
type TtyMessage struct {
	Session uint64
	Message []byte
}

// WindowSizeMessage is the structure corresponding to the format expected by
// hyperstart to resize a container's window.
type WindowSizeMessage struct {
	Container string `json:"container"`
	Process   string `json:"process"`
	Row       uint16 `json:"row"`
	Column    uint16 `json:"column"`
}

// VolumeDescriptor describes a volume related to a container.
type VolumeDescriptor struct {
	Device       string `json:"device"`
	Addr         string `json:"addr,omitempty"`
	Mount        string `json:"mount"`
	Fstype       string `json:"fstype,omitempty"`
	ReadOnly     bool   `json:"readOnly"`
	DockerVolume bool   `json:"dockerVolume"`
}

// FsmapDescriptor describes a filesystem map related to a container.
type FsmapDescriptor struct {
	Source       string `json:"source"`
	Path         string `json:"path"`
	ReadOnly     bool   `json:"readOnly"`
	DockerVolume bool   `json:"dockerVolume"`
}

// EnvironmentVar holds an environment variable and its value.
type EnvironmentVar struct {
	Env   string `json:"env"`
	Value string `json:"value"`
}

// Rlimit describes a resource limit.
type Rlimit struct {
	// Type of the rlimit to set
	Type string `json:"type"`
	// Hard is the hard limit for the specified type
	Hard uint64 `json:"hard"`
	// Soft is the soft limit for the specified type
	Soft uint64 `json:"soft"`
}

// Process describes a process running on a container inside a pod.
type Process struct {
	User             string   `json:"user,omitempty"`
	Group            string   `json:"group,omitempty"`
	AdditionalGroups []string `json:"additionalGroups,omitempty"`
	// Terminal creates an interactive terminal for the process.
	Terminal bool `json:"terminal"`
	// Sequeue number for stdin and stdout
	Stdio uint64 `json:"stdio,omitempty"`
	// Sequeue number for stderr if it is not shared with stdout
	Stderr uint64 `json:"stderr,omitempty"`
	// Args specifies the binary and arguments for the application to execute.
	Args []string `json:"args"`
	// Envs populates the process environment for the process.
	Envs []EnvironmentVar `json:"envs,omitempty"`
	// Workdir is the current working directory for the process and must be
	// relative to the container's root.
	Workdir string `json:"workdir"`
	// Rlimits specifies rlimit options to apply to the process.
	Rlimits []Rlimit `json:"rlimits,omitempty"`
}

// Container describes a container running on a pod.
type Container struct {
	ID            string              `json:"id"`
	Rootfs        string              `json:"rootfs"`
	Fstype        string              `json:"fstype,omitempty"`
	Image         string              `json:"image"`
	Addr          string              `json:"addr,omitempty"`
	Volumes       []*VolumeDescriptor `json:"volumes,omitempty"`
	Fsmap         []*FsmapDescriptor  `json:"fsmap,omitempty"`
	Sysctl        map[string]string   `json:"sysctl,omitempty"`
	Process       *Process            `json:"process"`
	RestartPolicy string              `json:"restartPolicy"`
	Initialize    bool                `json:"initialize"`
}

// IPAddress describes an IP address and its network mask.
type IPAddress struct {
	IPAddress string `json:"ipAddress"`
	NetMask   string `json:"netMask"`
}

// NetworkIface describes a network interface to setup on the host.
type NetworkIface struct {
	Device      string      `json:"device,omitempty"`
	NewDevice   string      `json:"newDeviceName,omitempty"`
	IPAddresses []IPAddress `json:"ipAddresses"`
	MTU         string      `json:"mtu"`
	MACAddr     string      `json:"macAddr"`
}

// Route describes a route to setup on the host.
type Route struct {
	Dest    string `json:"dest"`
	Gateway string `json:"gateway,omitempty"`
	Device  string `json:"device,omitempty"`
}

// Pod describes the pod configuration to start inside the VM.
type Pod struct {
	Hostname   string         `json:"hostname"`
	Containers []Container    `json:"containers,omitempty"`
	Interfaces []NetworkIface `json:"interfaces,omitempty"`
	DNS        []string       `json:"dns,omitempty"`
	Routes     []Route        `json:"routes,omitempty"`
	ShareDir   string         `json:"shareDir"`
}
