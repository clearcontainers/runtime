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

// Ready represents the unmarshalled version of the contents of an SSNTP READY
// payload.  The structure contains information about the state of an NN or a CN
// on which ciao-launcher is running.
type Ready struct {
	// NodeUUID is the SSNTP UUID assigned to the ciao-launcher instance
	// that transmitted the READY frame.
	NodeUUID string `yaml:"node_uuid"`

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
	// reported).
	Load int `yaml:"load"`

	// Number of CPUs present in the CN/NN.  Derived from the number of
	// cpu[0-9]+ entries in /proc/stat.
	CpusOnline int `yaml:"cpus_online"`

	// Any changes to this struct should be accompanied by a change to
	// the ciao-scheduler/scheduler.go:updateNodeStat() function
}

// Init initialises the Ready structure.
func (s *Ready) Init() {
	s.NodeUUID = ""
	s.MemTotalMB = -1
	s.MemAvailableMB = -1
	s.DiskTotalMB = -1
	s.DiskAvailableMB = -1
	s.Load = -1
	s.CpusOnline = -1
}
