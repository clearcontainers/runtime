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

package payloads_test

import (
	"testing"

	. "github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestReadyUnmarshal(t *testing.T) {
	var cmd Ready
	err := yaml.Unmarshal([]byte(testutil.ReadyYaml), &cmd)
	if err != nil {
		t.Error(err)
	}
}

func TestReadyMarshal(t *testing.T) {
	cmd := Ready{
		NodeUUID:        testutil.AgentUUID,
		MemTotalMB:      3896,
		MemAvailableMB:  3896,
		DiskTotalMB:     500000,
		DiskAvailableMB: 256000,
		Load:            0,
		CpusOnline:      4,
		Networks: []NetworkStat{
			{NodeIP: "192.168.1.1", NodeMAC: "02:00:15:03:6f:49"},
			{NodeIP: "10.168.1.1", NodeMAC: "02:00:8c:ba:f9:45"},
		},
	}

	y, err := yaml.Marshal(&cmd)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.ReadyYaml {
		t.Errorf("Ready marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.ReadyYaml)
	}
}

// make sure the yaml can be unmarshaled into the Ready struct
// when only some node stats are present
func TestReadyNodeNotAllStats(t *testing.T) {
	var cmd Ready
	cmd.Init()

	err := yaml.Unmarshal([]byte(testutil.PartialReadyYaml), &cmd)
	if err != nil {
		t.Error(err)
	}

	expectedCmd := Ready{
		NodeUUID:        testutil.AgentUUID,
		MemTotalMB:      -1,
		MemAvailableMB:  -1,
		DiskTotalMB:     -1,
		DiskAvailableMB: -1,
		Load:            1,
		CpusOnline:      -1,
	}
	if cmd.NodeUUID != expectedCmd.NodeUUID ||
		cmd.MemTotalMB != expectedCmd.MemTotalMB ||
		cmd.MemAvailableMB != expectedCmd.MemAvailableMB ||
		cmd.DiskTotalMB != expectedCmd.DiskTotalMB ||
		cmd.DiskAvailableMB != expectedCmd.DiskAvailableMB ||
		cmd.Load != expectedCmd.Load ||
		cmd.CpusOnline != expectedCmd.CpusOnline ||
		len(cmd.Networks) != 0 {
		t.Error("Unexpected values in Ready")
	}
}
