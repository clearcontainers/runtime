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

package testutil

import "github.com/01org/ciao/payloads"

// AgentIP is a test agent IP address
const AgentIP = "10.2.3.4"

//TenantSubnet is a test tenant subnet
const TenantSubnet = "10.2.0.0/16"

// SubnetKey is a test tenant subnet key
const SubnetKey = "8"

// InstancePublicIP is a test instance public IP
const InstancePublicIP = "10.1.2.3"

// InstancePrivateIP is a test instance private IP
const InstancePrivateIP = "192.168.1.2"

// VNICMAC is a test instance VNIC MAC address
const VNICMAC = "aa:bb:cc:01:02:03"

// VNICUUID is a test instance VNIC UUID
const VNICUUID = "7f49d00d-1995-4156-8c79-5f5ab24ce138"

// TenantUUID is a test tenant UUID
const TenantUUID = "2491851d-dce9-48d6-b83a-a717417072ce"

// CNCIUUID is a test CNCI instance UUID
const CNCIUUID = "7e84c2d6-5a84-4f9b-98e3-38980f722d1b"

// CNCIIP is a test CNCI instance IP address
const CNCIIP = "10.1.2.3"

// CNCIMAC is a test CNCI instance MAC address
const CNCIMAC = "CA:FE:C0:00:01:02"

// SchedulerAddr is a test scheduler address
const SchedulerAddr = "192.168.42.5"

// KeystoneURL is a test Keystone identity server url
const KeystoneURL = "http://keystone.example.com"

// GlanceURL is a test Glance image server url
const GlanceURL = "http://glance.example.com"

// ComputeNet is a test compute network
const ComputeNet = "192.168.1.110"

// MgmtNet is a test management network
const MgmtNet = "192.168.1.111"

// ManagementID is a test identifier for a Ceph ID
const ManagementID = "ciao"

// StorageURI is a test storage URI
const StorageURI = "/etc/ciao/ciao.json"

// IdentityUser is a test user for the test identity server
const IdentityUser = "controller"

// IdentityPassword is a test password for the test identity server
const IdentityPassword = "ciao"

// VolumePort is a test port for the compute service
const VolumePort = "446"

// ComputePort is a test port for the compute service
const ComputePort = "443"

// CiaoPort is a test port for ciao's api service
const CiaoPort = "447"

// HTTPSKey is a path to a key for the compute service
const HTTPSKey = "/etc/pki/ciao/compute_key.pem"

// HTTPSCACert is a path to the CA cert used to sign the HTTPSKey
const HTTPSCACert = "/etc/pki/ciao/compute_ca.pem"

// DockerImage is a docker image name for use in start/restart tests
const DockerImage = "docker/latest"

// ImageUUID is a disk image UUID for use in start/restart tests
const ImageUUID = "59460b8a-5f53-4e3e-b5ce-b71fed8c7e64"

// InstanceUUID is an instance UUID for use in start/stop/restart/delete tests
const InstanceUUID = "3390740c-dce9-48d6-b83a-a717417072ce"

// CNCIInstanceUUID is a CNCI instance UUID for use in start/stop/restart/delete tests
const CNCIInstanceUUID = "c6beb8b5-0bfc-43fd-9638-7dd788179fd8"

// NetAgentUUID is a network node UUID for coordinated tests
const NetAgentUUID = "6be56328-92e2-4ecd-b426-8fe529c04e0c"

// AgentUUID is a node UUID for coordinated stop/restart/delete tests
const AgentUUID = "4cb19522-1e18-439a-883a-f9b2a3a95f5e"

// VolumeUUID is a node UUID for storage tests
const VolumeUUID = "67d86208-b46c-4465-9018-e14187d4010"

var computeNetwork001 = payloads.NetworkStat{
	NodeIP:  "198.51.100.1",
	NodeMAC: "02:00:aa:cb:84:41",
}
var computeNetwork002 = payloads.NetworkStat{
	NodeIP:  "10.168.1.1",
	NodeMAC: "02:00:8c:ba:f9:45",
}
var computeNetwork003 = payloads.NetworkStat{
	NodeIP:  ComputeNet,
	NodeMAC: "02:00:15:03:6f:49",
}

// PartialComputeNetworks is meant to represent a node with partial access to
// compute networks in the testutil cluster
var PartialComputeNetworks = []payloads.NetworkStat{
	computeNetwork002,
}

// MultipleComputeNetworks is meant to represent a node with full access to
// all compute networks in the testutil cluster
var MultipleComputeNetworks = []payloads.NetworkStat{
	computeNetwork001,
	computeNetwork002,
	computeNetwork003,
}

//////////////////////////////////////////////////////////////////////////////

// StartYaml is a sample workload START ssntp.Command payload for test usage
const StartYaml = `start:
  tenant_uuid: ` + TenantUUID + `
  instance_uuid: ` + InstanceUUID + `
  image_uuid: ` + ImageUUID + `
  docker_image: ` + DockerImage + `
  fw_type: efi
  persistence: host
  vm_type: qemu
  requested_resources:
  - type: vcpus
    value: 2
    mandatory: true
  - type: mem_mb
    value: 4096
    mandatory: true
  estimated_resources:
  - type: vcpus
    value: 1
  - type: mem_mb
    value: 128
  networking:
    vnic_mac: ""
    vnic_uuid: ""
    concentrator_uuid: ""
    concentrator_ip: ""
    subnet: ""
    subnet_key: ""
    subnet_uuid: ""
    private_ip: ""
    public_ip: false
  restart: false
`

// CNCIStartYaml is a sample CNCI workload START ssntp.Command payload for test cases
const CNCIStartYaml = `start:
  instance_uuid: ` + CNCIInstanceUUID + `
  image_uuid: ` + ImageUUID + `
  fw_type: efi
  persistence: host
  vm_type: qemu
  requested_resources:
    - type: vcpus
      value: 4
      mandatory: true
    - type: mem_mb
      value: 4096
      mandatory: true
    - type: network_node
      value: 1
      mandatory: true
    - type: physical_network
      value_string: ` + ComputeNet + `
  networking:
    vnic_mac: ` + VNICMAC + `
    vnic_uuid: ` + VNICUUID + `
    concentrator_uuid: ` + CNCIUUID + `
    concentrator_ip: ` + CNCIIP + `
    subnet: ` + TenantSubnet + `
    subnet_key: ` + SubnetKey + `
    subnet_uuid: ""
    private_ip: ""
    public_ip: false
`

// PartialStartYaml is a sample minimal workload START ssntp.Command payload for test cases
const PartialStartYaml = `start:
  instance_uuid: ` + InstanceUUID + `
  image_uuid: ` + ImageUUID + `
  docker_image: ` + DockerImage + `
  fw_type: efi
  persistence: host
  vm_type: qemu
  requested_resources:
    - type: vcpus
      value: 2
      mandatory: true
`

// StartFailureYaml is a sample workload StartFailure ssntp.Error payload for test cases
const StartFailureYaml = `instance_uuid: ` + InstanceUUID + `
reason: full_cloud
restart: false
`

// RestartYaml is a sample workload RESTART ssntp.Command payload for test cases
const RestartYaml = `restart:
  tenant_uuid: ` + TenantUUID + `
  instance_uuid: ` + InstanceUUID + `
  image_uuid: ` + ImageUUID + `
  workload_agent_uuid: ` + AgentUUID + `
  fw_type: efi
  persistence: host
  requested_resources:
  - type: vcpus
    value: 2
    mandatory: true
  - type: mem_mb
    value: 4096
    mandatory: true
  estimated_resources:
  - type: vcpus
    value: 1
  - type: mem_mb
    value: 128
  networking:
    vnic_mac: ""
    vnic_uuid: ""
    concentrator_uuid: ""
    concentrator_ip: ""
    subnet: ""
    subnet_key: ""
    subnet_uuid: ""
    private_ip: ""
    public_ip: false
`

// PartialRestartYaml is a sample minimal workload RESTART ssntp.Command payload for test cases
const PartialRestartYaml = `restart:
  instance_uuid: ` + InstanceUUID + `
  workload_agent_uuid: ` + AgentUUID + `
  fw_type: efi
  persistence: host
  requested_resources:
    - type: vcpus
      value: 2
      mandatory: true
`

// RestartFailureYaml is a sample workload RestartFailure ssntp.Error payload for test cases
const RestartFailureYaml = `instance_uuid: ` + InstanceUUID + `
reason: already_running
`

// StopYaml is a sample workload STOP ssntp.Command payload for test cases
const StopYaml = `stop:
  instance_uuid: ` + InstanceUUID + `
  workload_agent_uuid: ` + AgentUUID + `
  stop: false
`

// StopFailureYaml is a sample workload StopFailure ssntp.Error payload for test cases
const StopFailureYaml = `instance_uuid: ` + InstanceUUID + `
reason: already_stopped
`

// DeleteYaml is a sample workload DELETE ssntp.Command payload for test cases
const DeleteYaml = `delete:
  instance_uuid: ` + InstanceUUID + `
  workload_agent_uuid: ` + AgentUUID + `
  stop: false
`

// MigrateYaml is a sample workload DELETE ssntp.Command payload for test cases
// that indicates that an instance is to be migrated rather than deleted.
const MigrateYaml = `delete:
  instance_uuid: ` + InstanceUUID + `
  workload_agent_uuid: ` + AgentUUID + `
  stop: true
`

// EvacuateYaml is a sample node EVACUATE ssntp.Command payload for test cases
const EvacuateYaml = `evacuate:
  workload_agent_uuid: ` + AgentUUID + `
`

// CNCIAddedYaml is a sample ConcentratorInstanceAdded ssntp.Event payload for test cases
const CNCIAddedYaml = `concentrator_instance_added:
  instance_uuid: ` + CNCIUUID + `
  tenant_uuid: ` + TenantUUID + `
  concentrator_ip: ` + CNCIIP + `
  concentrator_mac: ` + CNCIMAC + `
`

// AssignIPYaml is a sample AssignPublicIP ssntp.Command payload for test cases
const AssignIPYaml = `assign_public_ip:
  concentrator_uuid: ` + CNCIUUID + `
  tenant_uuid: ` + TenantUUID + `
  instance_uuid: ` + InstanceUUID + `
  public_ip: ` + InstancePublicIP + `
  private_ip: ` + InstancePrivateIP + `
  vnic_mac: ` + VNICMAC + `
`

// ReleaseIPYaml is a sample ReleasePublicIP ssntp.Command payload for test cases
const ReleaseIPYaml = `release_public_ip:
  concentrator_uuid: ` + CNCIUUID + `
  tenant_uuid: ` + TenantUUID + `
  instance_uuid: ` + InstanceUUID + `
  public_ip: ` + InstancePublicIP + `
  private_ip: ` + InstancePrivateIP + `
  vnic_mac: ` + VNICMAC + `
`

// AssignedIPYaml is a sample PublicIPAssigned ssntp.Event payload for test cases
const AssignedIPYaml = `public_ip_assigned:
  concentrator_uuid: ` + CNCIUUID + `
  instance_uuid: ` + InstanceUUID + `
  public_ip: ` + InstancePublicIP + `
  private_ip: ` + InstancePrivateIP + `
`

// UnassignedIPYaml is a sample PublicIPUnassigned ssntp.Event payload for test cases
const UnassignedIPYaml = `public_ip_unassigned:
  concentrator_uuid: ` + CNCIUUID + `
  instance_uuid: ` + InstanceUUID + `
  public_ip: ` + InstancePublicIP + `
  private_ip: ` + InstancePrivateIP + `
`

// TenantAddedYaml is a sample TenantAdded ssntp.Event payload for test cases
const TenantAddedYaml = `tenant_added:
  agent_uuid: ` + AgentUUID + `
  agent_ip: ` + AgentIP + `
  tenant_uuid: ` + TenantUUID + `
  tenant_subnet: ` + TenantSubnet + `
  concentrator_uuid: ` + CNCIUUID + `
  concentrator_ip: ` + CNCIIP + `
  subnet_key: ` + SubnetKey + `
`

// TenantRemovedYaml is a sample TenantRemove ssntp.Event payload for test cases
const TenantRemovedYaml = `tenant_removed:
  agent_uuid: ` + AgentUUID + `
  agent_ip: ` + AgentIP + `
  tenant_uuid: ` + TenantUUID + `
  tenant_subnet: ` + TenantSubnet + `
  concentrator_uuid: ` + CNCIUUID + `
  concentrator_ip: ` + CNCIIP + `
  subnet_key: ` + SubnetKey + `
`

// CNCIInstanceData is a sample CNCIInstanceConfig payload for test cases
const CNCIInstanceData = `scheduler_addr: 192.168.42.5
`

// ConfigureYaml is a sample CONFIGURE ssntp.Command payload for test cases
const ConfigureYaml = `configure:
  scheduler:
    storage_uri: ` + StorageURI + `
  storage:
    ceph_id: ` + ManagementID + `
  controller:
    volume_port: ` + VolumePort + `
    compute_port: ` + ComputePort + `
    ciao_port: ` + CiaoPort + `
    compute_fqdn: ""
    compute_ca: ` + HTTPSCACert + `
    compute_cert: ` + HTTPSKey + `
    identity_user: ` + IdentityUser + `
    identity_password: ` + IdentityPassword + `
    cnci_vcpus: 0
    cnci_mem: 0
    cnci_disk: 0
    admin_ssh_key: ""
    admin_password: ""
  launcher:
    compute_net:
    - ` + ComputeNet + `
    mgmt_net:
    - ` + MgmtNet + `
    disk_limit: false
    mem_limit: false
  identity_service:
    type: keystone
    url: ` + KeystoneURL + `
`

// DeleteFailureYaml is a sample workload DeleteFailure ssntp.Error payload for test cases
const DeleteFailureYaml = `instance_uuid: ` + InstanceUUID + `
reason: no_instance
`

// InsDelYaml is a sample workload InstanceDeleted ssntp.Event payload for test cases
const InsDelYaml = `instance_deleted:
  instance_uuid: ` + InstanceUUID + `
`

// InsStopYaml is a sample workload InstanceStopped ssntp.Event payload for test cases
const InsStopYaml = `instance_stopped:
  instance_uuid: ` + InstanceUUID + `
`

// NodeConnectedYaml is a sample node NodeConnected ssntp.Event payload for test cases
const NodeConnectedYaml = `node_connected:
  node_uuid: ` + AgentUUID + `
  node_type: ` + payloads.NetworkNode + `
`

// ReadyPayload is a helper to craft a mostly fixed ssntp.READY status
// payload, with parameters to specify the source node uuid and available resources
func ReadyPayload(uuid string, memTotal int, memAvail int, networks []payloads.NetworkStat) payloads.Ready {
	p := payloads.Ready{
		NodeUUID:        uuid,
		MemTotalMB:      memTotal,
		MemAvailableMB:  memAvail,
		DiskTotalMB:     500000,
		DiskAvailableMB: 256000,
		Load:            0,
		CpusOnline:      4,
		Networks:        networks,
	}
	return p
}

// ReadyYaml is a sample node READY ssntp.Status payload for test cases
const ReadyYaml = `node_uuid: ` + AgentUUID + `
mem_total_mb: 3896
mem_available_mb: 3896
disk_total_mb: 500000
disk_available_mb: 256000
load: 0
cpus_online: 4
networks:
- ip: 192.168.1.1
  mac: 02:00:15:03:6f:49
- ip: 10.168.1.1
  mac: 02:00:8c:ba:f9:45
`

// PartialReadyYaml is a sample minimal node READY ssntp.Status payload for test cases
const PartialReadyYaml = `node_uuid: ` + AgentUUID + `
load: 1
`

// InstanceStat001 is a sample payloads.InstanceStat
var InstanceStat001 = payloads.InstanceStat{
	InstanceUUID:  "fe2970fa-7b36-460b-8b79-9eb4745e62f2",
	State:         payloads.Running,
	MemoryUsageMB: 40,
	DiskUsageMB:   2,
	CPUUsage:      90,
	SSHIP:         "",
	SSHPort:       0,
}

// InstanceStat002 is a sample payloads.InstanceStat
var InstanceStat002 = payloads.InstanceStat{
	InstanceUUID:  "cbda5bd8-33bd-4d39-9f52-ace8c9f0b99c",
	State:         payloads.Running,
	MemoryUsageMB: 50,
	DiskUsageMB:   10,
	CPUUsage:      0,
	SSHIP:         "172.168.2.2",
	SSHPort:       8768,
}

// InstanceStat003 is a sample payloads.InstanceStat
var InstanceStat003 = payloads.InstanceStat{
	InstanceUUID:  "1f5b2fe6-4493-4561-904a-8f4e956218d9",
	State:         payloads.Exited,
	MemoryUsageMB: -1,
	DiskUsageMB:   2,
	CPUUsage:      -1,
	Volumes:       []string{VolumeUUID},
}

// NetworkStat001 is a sample payloads.NetworkStat
var NetworkStat001 = payloads.NetworkStat{
	NodeIP:  "192.168.1.1",
	NodeMAC: "02:00:15:03:6f:49",
}

// NetworkStat002 is a sample payloads.NetworkStat
var NetworkStat002 = payloads.NetworkStat{
	NodeIP:  "10.168.1.1",
	NodeMAC: "02:00:8c:ba:f9:45",
}

// StatsPayload is a factory function for a node STATS ssntp.Command payload for test cases
// The StatsPayload() factory function returns a payloads.Stat object.
// If passed uuid==AgentUUID, instances==[InstanceStat001,InstanceStat002,InstanceStat003]
// and networks==[NetworkStat001,NetworkStat002], then StatsPayload() will
// return a payloads.Stat matching the StatsYaml string.
func StatsPayload(uuid string, name string, instances []payloads.InstanceStat, networks []payloads.NetworkStat) payloads.Stat {
	p := payloads.Stat{
		NodeUUID:        uuid,
		Status:          "READY",
		MemTotalMB:      3896,
		MemAvailableMB:  3896,
		DiskTotalMB:     500000,
		DiskAvailableMB: 256000,
		Load:            0,
		CpusOnline:      4,
		NodeHostName:    name,
		Instances:       instances,
		Networks:        networks,
	}

	return p
}

// StatsYaml is a sample node STATS ssntp.Command payload for test cases
const StatsYaml = `node_uuid: ` + AgentUUID + `
status: READY
mem_total_mb: 3896
mem_available_mb: 3896
disk_total_mb: 500000
disk_available_mb: 256000
load: 0
cpus_online: 4
hostname: test
networks:
- ip: 192.168.1.1
  mac: 02:00:15:03:6f:49
- ip: 10.168.1.1
  mac: 02:00:8c:ba:f9:45
instances:
- instance_uuid: fe2970fa-7b36-460b-8b79-9eb4745e62f2
  state: active
  ssh_ip: ""
  ssh_port: 0
  memory_usage_mb: 40
  disk_usage_mb: 2
  cpu_usage: 90
  volumes: []
- instance_uuid: cbda5bd8-33bd-4d39-9f52-ace8c9f0b99c
  state: active
  ssh_ip: 172.168.2.2
  ssh_port: 8768
  memory_usage_mb: 50
  disk_usage_mb: 10
  cpu_usage: 0
  volumes: []
- instance_uuid: 1f5b2fe6-4493-4561-904a-8f4e956218d9
  state: exited
  ssh_ip: ""
  ssh_port: 0
  memory_usage_mb: -1
  disk_usage_mb: 2
  cpu_usage: -1
  volumes:
  - 67d86208-b46c-4465-9018-e14187d4010
`

// NodeOnlyStatsYaml is a sample minimal node STATS ssntp.Command payload for test cases
// with no per-instance statistics
const NodeOnlyStatsYaml = `node_uuid: ` + AgentUUID + `
mem_total_mb: 3896
mem_available_mb: 3896
disk_total_mb: 500000
disk_available_mb: 256000
load: 0
cpus_online: 4
hostname: test
networks:
- ip: 192.168.1.1
  mac: 02:00:15:03:6f:49
`

// PartialStatsYaml is a sample minimal node STATS ssntp.Command payload for test cases
// with limited node statistics and no per-instance statistics
const PartialStatsYaml = `node_uuid: ` + AgentUUID + `
load: 1
`

// AttachVolumeYaml is a sample yaml payload for the ssntp Attach Volume command.
const AttachVolumeYaml = `attach_volume:
  instance_uuid: ` + InstanceUUID + `
  volume_uuid: ` + VolumeUUID + `
  workload_agent_uuid: ` + AgentUUID + `
`

// BadAttachVolumeYaml is a corrupt yaml payload for the ssntp Attach Volume command.
const BadAttachVolumeYaml = `attach_volume:
  volume_uuid: ` + VolumeUUID + `
`

// DetachVolumeYaml is a sample yaml payload for the ssntp Detach Volume command.
const DetachVolumeYaml = `detach_volume:
  instance_uuid: ` + InstanceUUID + `
  volume_uuid: ` + VolumeUUID + `
  workload_agent_uuid: ` + AgentUUID + `
`

// BadDetachVolumeYaml is a corrupt yaml payload for the ssntp Detach Volume command.
const BadDetachVolumeYaml = `detach_volume:
  instance_uuid: ` + InstanceUUID + `
`

// AttachVolumeFailureYaml is a sample AttachVolumeFailure ssntp.Error payload for test cases
const AttachVolumeFailureYaml = `instance_uuid: ` + InstanceUUID + `
volume_uuid: ` + VolumeUUID + `
reason: attach_failure
`

// DetachVolumeFailureYaml is a sample DetachVolumeFailure ssntp.Error payload for test cases
const DetachVolumeFailureYaml = `instance_uuid: ` + InstanceUUID + `
volume_uuid: ` + VolumeUUID + `
reason: detach_failure
`
