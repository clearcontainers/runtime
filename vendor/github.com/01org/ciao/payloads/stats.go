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

// InstanceStat contains information about the state of an indiviual
// instance in a ciao cluster.
type InstanceStat struct {

	// UUID of the instance to which this stats structure pertains
	InstanceUUID string `yaml:"instance_uuid"`

	// State of the instance, e.g., running, pending, exited
	State string `yaml:"state"`

	// IP address to use to connect to instance via SSH.  This
	// is actually the IP address of the CNCI VM.
	// Will be "" if the instance is itself a CNCI VM.
	SSHIP string `yaml:"ssh_ip"`

	// Port number used to access the SSH service running on the
	// VM.  This number is computed from the VM's IP address.
	// Will be 0 if the instance is itself a CNCI VM.
	SSHPort int `yaml:"ssh_port"`

	// Memory usage in MB.  May be -1 if State != Running.
	MemoryUsageMB int `yaml:"memory_usage_mb"`

	// Disk usage in MB.  May be -1 if State = Pending.
	DiskUsageMB int `yaml:"disk_usage_mb"`

	// Percentage of CPU Usage for VM, normalized for VCPUs.
	// May be -1 if State != Running or if launcher has not
	// acquired enough samples to compute the CPU usage.
	// Assuming CPU usage can be computed it will be a value
	// between 0 and 100% regardless of the number of VPCUs.
	// 100% means all your VCPUs are maxed out.
	CPUUsage int `yaml:"cpu_usage"`

	// List of volumes attached to the instance.
	Volumes []string `yaml:"volumes"`
}

// NetworkStat contains information about a single network interface present on
// a ciao compute or network node.
type NetworkStat struct {
	NodeIP  string `yaml:"ip"`
	NodeMAC string `yaml:"mac"`
}

// Stat represents a snapshot of the state of a compute or a network node.  This
// information is sent periodically by ciao-launcher to the scheduler.
type Stat struct {
	// The UUID of the launcher instance from which the Stats structure
	// originated
	NodeUUID string `yaml:"node_uuid"`

	// The Status of the node, e.g., READY or FULL
	Status string `yaml:"status"`

	// Total amount of RAM available on a CN or NN
	MemTotalMB int `yaml:"mem_total_mb"`

	// Memory currently available on a CN or NN, computed from
	// proc/meminfo:MemFree + Active(file) + Inactive(file)
	MemAvailableMB int `yaml:"mem_available_mb"`

	// Size of the CN/NN RootFS in MB
	DiskTotalMB int `yaml:"disk_total_mb"`

	// MBs available in the RootFS of the CN/NN
	DiskAvailableMB int `yaml:"disk_available_mb"`

	// Load of CN/NN, taken from /proc/loadavg (Average over last minute
	// reported
	Load int `yaml:"load"`

	// Number of CPUs present in the CN/NN.  Derived from the number of
	// cpu[0-9]+ entries in /proc/stat
	CpusOnline int `yaml:"cpus_online"`

	// Hostname of the CN/NN
	NodeHostName string `yaml:"hostname"`

	// Array containing one entry for each network interface present on the
	// CN/NN
	Networks []NetworkStat

	// Array containing statistics information for each instance hosted by
	// the CN/NN
	Instances []InstanceStat
}

const (
	// ComputeStatusPending is a filter that used to select pending
	// instances in requests to the controller.
	ComputeStatusPending = "pending"

	// ComputeStatusRunning is a filter that used to select running
	// instances in requests to the controller.
	ComputeStatusRunning = "active"

	// ComputeStatusStopped is a filter that used to select exited
	// instances in requests to the controller.
	ComputeStatusStopped = "exited"
)

const (
	// Pending indicates that ciao-launcher has not yet ascertained the
	// state of a given instance.  This can happen, either because the
	// instance is in the process of being created, or ciao-launcher itself
	// has just started and is still gathering information about the
	// existing instances.
	Pending = ComputeStatusPending

	// Running indicates an instance is running
	Running = ComputeStatusRunning

	// Exited indicates that an instance has been successfully created but
	// is not currently running, either because it failed to start or was
	// explicitly stopped by a STOP command or perhaps by a CN reboot.
	Exited = ComputeStatusStopped
	// ExitFailed is not currently used
	ExitFailed = "exit_failed"
	// ExitPaused is not currently used
	ExitPaused = "exit_paused"
)

// Init initialises instances of the Stat structure.
func (s *Stat) Init() {
	s.NodeUUID = ""
	s.MemTotalMB = -1
	s.MemAvailableMB = -1
	s.DiskTotalMB = -1
	s.DiskAvailableMB = -1
	s.Load = -1
	s.CpusOnline = -1
}
