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

package payloads

// Persistence represents the persistency of an instance, i.e., whether that
// instance should be restarted after certain events have occurred, e.g., the
// node on which the instance runs is rebooted. It's not currently implemented
// in ciao-launcher.
type Persistence string

// Firmware represents the type of firmware used to boot a VM
type Firmware string

// Resource represents the name of a resource in a StartCmd structure
type Resource string

// Hypervisor indicates the type of hypervisor used to run a given instance
type Hypervisor string

const (
	// All used to indicate all persistent scenario, in this case it
	// indicates to act in all instances.
	All Persistence = "all"

	// VM used to indicate in a instance persistent scenario, in this case it
	// indicates to act in only one instance.
	VM = "vm"

	// Host used to indicate in a Host persistent scenario, in this case it
	// indicates to act in only in the instances of a host.
	Host = "host"
)

const (
	// EFI indicates that EFI firmware, e.g., OVMF.fd, should be used to
	// boot a VM
	EFI Firmware = "efi"

	// Legacy indicates that legacy firmware, e.g., BIOS should be used
	// to boot a VM
	Legacy = "legacy"
)

const (
	// VCPUs indicates that a particular resource struct contains a count
	// of VCPUs
	VCPUs Resource = "vcpus"

	// MemMB indicates that a resource struct specifies a quantity of memory
	// in MBs
	MemMB = "mem_mb"

	// NetworkNode indicates that a resource struct specifies whether the
	// command in which it is embedded applies to a network node.
	NetworkNode = "network_node"

	// ComputeNode indicates that a resource struct specifies whether the
	// command in which it is embedded applies to a compute node.
	ComputeNode = "compute_node"
)

const (
	// QEMU specifies that an instance is to be booted on QEMU KVM VM.
	QEMU Hypervisor = "qemu"

	// Docker specifies that an instance is to be launched inside a Docker
	// container.
	Docker = "docker"
)

// StorageResource represents a requested storage resource for a workload.
type StorageResource struct {
	// ID is passed to the Block Driver to operate on the resource
	ID string `yaml:"id,omitempty"`

	// Bootable indicates that this is a bootable storage device.
	Bootable bool `yaml:"boot,omitempty"`

	// BootIndex hints to the hypervisor a bootable disk order among
	// multple storage resources
	BootIndex int `yaml:"boot_index,omitempty"`

	// Ephemeral indicates whether this storage should only last as long as
	// the instance
	Ephemeral bool `yaml:"ephemeral,omitempty"`

	// Local indicates if the storage resource is local (ephemeral,
	// auto-created by launcher implied) or backed by the volume service
	Local bool `yaml:"local,omitempty"`

	// Swap optionally indicates the storage resource is intended for
	// use as a linux swap device (attempt swapon via cloud-init
	Swap bool `yaml:"swap,omitempty"`

	// Tag is an arbitrary text identifier
	Tag string `yaml:"tag,omitempty"`

	// Size is the requested size for an auto-created storage resource
	Size int `yaml:"size,omitempty"`
}

// RequestedResource is used to specify an individual resource contained within
// a Start or Restart command.  Example of resources include number of VCPUs or
// MBs of RAM to assign to an instance
type RequestedResource struct {
	// Type specifies the type of the resource, e.g., VCPUs.
	Type Resource `yaml:"type"`

	// Value specifies the integer value associated with that resource.
	Value int `yaml:"value"`

	// Mandatory indicates whether a resource is mandatory or not.
	Mandatory bool `yaml:"mandatory"`
}

// EstimatedResource contains the definition of estimated value of a resource.
type EstimatedResource struct {
	// Type is the resource type.
	Type Resource `yaml:"type"`

	// value is the value of the resource.
	Value int `yaml:"value"`
}

// NetworkResources contains all the networking information for an instance.
type NetworkResources struct {

	// VnicMAC contains the MAC address of an instance's VNIC.
	VnicMAC string `yaml:"vnic_mac"`

	// VnicUUID is a cluster unique UUID assigned to an instance's VNIC.
	VnicUUID string `yaml:"vnic_uuid"`

	// ConcentratorUUID is the UUID of the CNCI instance.  Only specified
	// when creating CN instances.
	ConcentratorUUID string `yaml:"concentrator_uuid"`

	// ConcentratorIP is the IP address of the CNCI.  Only specified
	// when creating CN instances.
	ConcentratorIP string `yaml:"concentrator_ip"`

	// Subnet is the subnet to which the instance is assigned.  Only
	// specified when creating CN instances.
	Subnet string `yaml:"subnet"`

	// SubnetKey is the subnet identifier to which the instance
	// is assigned.
	SubnetKey string `yaml:"subnet_key"`

	// SubnetUUID is the subnet ID of the subnet to which the instance is
	//  assinged.
	SubnetUUID string `yaml:"subnet_uuid"`

	// PrivateIP represents the private IP address of an instance.  Only
	// specified when creating CN instances.
	PrivateIP string `yaml:"private_ip"`

	// PublicIP represents the current statu of the assignation of a Public
	// IP.
	PublicIP bool `yaml:"public_ip"`
}

// StartCmd contains the information needed to start a new instance.
type StartCmd struct {
	// TenantUUID is the UUID of the tenant to which the new instance will
	// belong.
	TenantUUID string `yaml:"tenant_uuid"`

	// InstanceUUID is the UUID of the instance itself.
	InstanceUUID string `yaml:"instance_uuid"`

	// ImageUUID is the UUID of the image upon which the RootFS of this
	// instance will be based.  Only used for qemu instances.
	ImageUUID string `yaml:"image_uuid"`

	// DockerImage is the name of the docker base image from which the
	// container will be created.  It should match the name of an
	// existing image in the docker registry.  Only used for docker
	// instances.
	DockerImage string `yaml:"docker_image"`

	// FWType indicates the type of firmware needed to boot the instance.
	// Only used for qemu instances.
	FWType Firmware `yaml:"fw_type"`

	// InstancePersistence is the persistence type for the instance.
	InstancePersistence Persistence `yaml:"persistence"`

	// VMType indicates whether we are creating a qemu or docker instance.
	VMType Hypervisor `yaml:"vm_type"`

	// RequestedResources contains a list of the resources that are to be
	// assigned to the new instance.
	RequestedResources []RequestedResource `yaml:"requested_resources"`

	// EstimatedResources represents the estimated value of the instance resource.
	EstimatedResources []EstimatedResource `yaml:"estimated_resources"`

	// Networking contains all the information required to set up networking
	// for the new instance.
	Networking NetworkResources `yaml:"networking"`

	// Storage contains all the information required to attach or boot
	// from storage for the new instance.
	Storage []StorageResource `yaml:"storage,omitempty"`
}

// Start represents the unmarshalled version of the contents of a SSNTP START
// payload.  The structure contains enough information to create and launch a
// new CN or NN instance.
type Start struct {
	// Start contains information about the instance to create.
	Start StartCmd `yaml:"start"`
}

// RestartCmd contains information needed to restart an instance.
type RestartCmd struct {
	// TenantUUID is the tenant ID of the instance to restart.
	TenantUUID string `yaml:"tenant_uuid"`

	// InstanceUUID is the UUID of the instance to restart.
	InstanceUUID string `yaml:"instance_uuid"`

	// ImageUUID  is the image ID fo the instance to restart.
	ImageUUID string `yaml:"image_uuid"`

	// WorkloadAgentUUID identifies the node on which the instance is
	// running.  This information is needed by the scheduler to route
	// the command to the correct CN/NN.
	WorkloadAgentUUID string `yaml:"workload_agent_uuid"`

	// FWType indicates the type of firmware needed to boot the instance.
	FWType Firmware `yaml:"fw_type"`

	// InstancePersistence is the persistence type for the instance.
	InstancePersistence Persistence `yaml:"persistence"`

	// RequestedResources contains a list of the resources that are to be
	// assigned to the new instance.
	RequestedResources []RequestedResource `yaml:"requested_resources"`

	// EstimatedResources represents the estimated value of the instance resource.
	EstimatedResources []EstimatedResource `yaml:"estimated_resources"`

	// Networking contains all the information required to set up networking
	// for the new instance.
	Networking NetworkResources `yaml:"networking"`
}

// Restart represents the unmarshalled version of the contents of a SSNTP
// RESTART payload.  The structure contains enough information to restart a CN
// or NN instance.
type Restart struct {
	Restart RestartCmd `yaml:"restart"`
}
