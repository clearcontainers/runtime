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

func TestStatsUnmarshal(t *testing.T) {
	var cmd Stat
	err := yaml.Unmarshal([]byte(testutil.StatsYaml), &cmd)
	if err != nil {
		t.Error(err)
	}
}

func TestStatsMarshal(t *testing.T) {
	instances := []InstanceStat{
		testutil.InstanceStat001,
		testutil.InstanceStat002,
		testutil.InstanceStat003,
	}
	networks := []NetworkStat{
		testutil.NetworkStat001,
		testutil.NetworkStat002,
	}
	p := testutil.StatsPayload(testutil.AgentUUID, "test", instances, networks)

	y, err := yaml.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.StatsYaml {
		t.Errorf("Stats marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.StatsYaml)
	}
}

// make sure the yaml can be unmarshaled into the Stat struct with
// no instances present
func TestStatsNodeOnly(t *testing.T) {
	var cmd Stat
	err := yaml.Unmarshal([]byte(testutil.NodeOnlyStatsYaml), &cmd)
	if err != nil {
		t.Error(err)
	}

	expectedCmd := Stat{
		NodeUUID:        testutil.AgentUUID,
		MemTotalMB:      3896,
		MemAvailableMB:  3896,
		DiskTotalMB:     500000,
		DiskAvailableMB: 256000,
		Load:            0,
		CpusOnline:      4,
		NodeHostName:    "test",
		Networks: []NetworkStat{
			{
				NodeIP:  "192.168.1.1",
				NodeMAC: "02:00:15:03:6f:49",
			},
		},
	}
	if cmd.NodeUUID != expectedCmd.NodeUUID ||
		cmd.MemTotalMB != expectedCmd.MemTotalMB ||
		cmd.MemAvailableMB != expectedCmd.MemAvailableMB ||
		cmd.DiskTotalMB != expectedCmd.DiskTotalMB ||
		cmd.DiskAvailableMB != expectedCmd.DiskAvailableMB ||
		cmd.Load != expectedCmd.Load ||
		cmd.CpusOnline != expectedCmd.CpusOnline ||
		cmd.NodeHostName != expectedCmd.NodeHostName ||
		len(cmd.Networks) != 1 ||
		cmd.Networks[0] != expectedCmd.Networks[0] ||
		cmd.Instances != nil {
		t.Error("Unexpected values in Stat")
	}
}

// make sure the yaml can be unmarshaled into the Stat struct
// when only some node stats are present
func TestStatsNodeNotAllStats(t *testing.T) {
	var cmd Stat
	cmd.Init()

	err := yaml.Unmarshal([]byte(testutil.PartialStatsYaml), &cmd)
	if err != nil {
		t.Error(err)
	}

	expectedCmd := Stat{
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
		cmd.NodeHostName != expectedCmd.NodeHostName ||
		cmd.Networks != nil ||
		cmd.Instances != nil {
		t.Error("Unexpected values in Stat")
	}
}
