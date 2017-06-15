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
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/01org/ciao/networking/libsnnet"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
)

var startTests = []struct {
	payload string
	config  *vmConfig
}{
	{
		`
start:
  requested_resources:
     - type: vcpus
       value: 2
     - type: mem_mb
       value: 370
  instance_uuid: d7d86208-b46c-4465-9018-ee14087d415f
  tenant_uuid: 67d86208-000-4465-9018-fe14087d415f
  fw_type: legacy
  vm_type: qemu
  networking:
    vnic_mac: 02:00:e6:f5:af:f9
    vnic_uuid: 67d86208-b46c-0000-9018-fe14087d415f
    concentrator_ip: 192.168.42.21
    concentrator_uuid: 67d86208-b46c-4465-0000-fe14087d415f
    subnet: 192.168.8.0/21
    private_ip: 192.168.8.2
  storage:
     - id: 69e84267-ed01-4738-b15f-b47de06b62e7
       boot: true
`,
		&vmConfig{
			Cpus:       2,
			Mem:        370,
			Instance:   "d7d86208-b46c-4465-9018-ee14087d415f",
			Legacy:     true,
			VnicMAC:    "02:00:e6:f5:af:f9",
			VnicIP:     "192.168.8.2",
			ConcIP:     "192.168.42.21",
			SubnetIP:   "192.168.8.0/21",
			TenantUUID: "67d86208-000-4465-9018-fe14087d415f",
			ConcUUID:   "67d86208-b46c-4465-0000-fe14087d415f",
			VnicUUID:   "67d86208-b46c-0000-9018-fe14087d415f",
			SSHPort:    35050,
			Volumes: []volumeConfig{
				{
					"69e84267-ed01-4738-b15f-b47de06b62e7",
					true,
				},
			},
		},
	},
	{
		"start",
		nil,
	},
	{
		`
start:
  requested_resources:
     - type: vcpus
       value: 2
     - type: mem_mb
       value: 370
  instance_uuid: imnotvalid
  tenant_uuid: 67d86208-000-4465-9018-fe14087d415f
  fw_type: legacy
  networking:
    vnic_mac: 02:00:e6:f5:af:f9
    vnic_uuid: 67d86208-b46c-0000-9018-fe14087d415f
    concentrator_ip: 192.168.42.21
    concentrator_uuid: 67d86208-b46c-4465-0000-fe14087d415f
    subnet: 192.168.8.0/21
    private_ip: 192.168.8.2
  storage:
     - id: 69e84267-ed01-4738-b15f-b47de06b62e7
       boot: true
`,
		nil,
	},
	{
		`
start:
  requested_resources:
     - type: vcpus
       value: 2
     - type: mem_mb
       value: 370
  instance_uuid: d7d86208-b46c-4465-9018-ee14087d415f
  tenant_uuid: 67d86208-000-4465-9018-fe14087d415f
  fw_type: imnotvalid
  networking:
    vnic_mac: 02:00:e6:f5:af:f9
    vnic_uuid: 67d86208-b46c-0000-9018-fe14087d415f
    concentrator_ip: 192.168.42.21
    concentrator_uuid: 67d86208-b46c-4465-0000-fe14087d415f
    subnet: 192.168.8.0/21
    private_ip: 192.168.8.2
  storage:
     - id: 69e84267-ed01-4738-b15f-b47de06b62e7
       boot: true
`,
		nil,
	},
	{
		`
start:
  requested_resources:
     - type: vcpus
       value: 2
     - type: mem_mb
       value: 370
  instance_uuid: d7d86208-b46c-4465-9018-ee14087d415f
  vm_type: askajajlsj
  tenant_uuid: 67d86208-000-4465-9018-fe14087d415f
  fw_type: legacy
  networking:
    vnic_mac: 02:00:e6:f5:af:f9
    vnic_uuid: 67d86208-b46c-0000-9018-fe14087d415f
    concentrator_ip: 192.168.42.21
    concentrator_uuid: 67d86208-b46c-4465-0000-fe14087d415f
    subnet: 192.168.8.0/21
    private_ip: 192.168.8.2
  storage:
     - id: 69e84267-ed01-4738-b15f-b47de06b62e7
       boot: true
`,
		nil,
	},
}

// Verify the parseAttachVolumePayload function.
//
// The function is passed one valid payload and two invalid payloads.
//
// No error should be returned for the valid payload and the returned instance
// and volume UUIDs should match what is in the payload.  Errors should be
// returned for the invalid payloads.
func TestParseAttachVolumePayload(t *testing.T) {
	instance, volume, err := parseAttachVolumePayload([]byte(testutil.AttachVolumeYaml))
	if err != nil {
		t.Fatalf("parseAttachVolumePayload failed: %v", err)
	}
	if instance != testutil.InstanceUUID || volume != testutil.VolumeUUID {
		t.Fatalf("VolumeUUID or InstanceUUID is invalid")
	}

	_, _, err = parseAttachVolumePayload([]byte("  -"))
	if err == nil || err.code != payloads.AttachVolumeInvalidPayload {
		t.Fatalf("AttachVolumeInvalidPayload error expected")
	}

	_, _, err = parseAttachVolumePayload([]byte(testutil.BadAttachVolumeYaml))
	if err == nil || err.code != payloads.AttachVolumeInvalidData {
		t.Fatalf("AttachVolumeInvalidData error expected")
	}
}

// Verify the parseDetachVolumePayload function.
//
// The function is passed one valid payload and two invalid payloads.
//
// No error should be returned for the valid payload and the returned instance
// and volume UUIDs should match what is in the payload.  Errors should be
// returned for the invalid payloads.
func TestParseDetachVolumePayload(t *testing.T) {
	instance, volume, err := parseDetachVolumePayload([]byte(testutil.DetachVolumeYaml))
	if err != nil {
		t.Fatalf("parseDetachVolumePayload failed: %v", err)
	}
	if instance != testutil.InstanceUUID || volume != testutil.VolumeUUID {
		t.Fatalf("VolumeUUID or InstanceUUID is invalid")
	}

	_, _, err = parseDetachVolumePayload([]byte("  -"))
	if err == nil || err.code != payloads.DetachVolumeInvalidPayload {
		t.Fatalf("AttachVolumeInvalidPayload error expected")
	}

	_, _, err = parseDetachVolumePayload([]byte(testutil.BadDetachVolumeYaml))
	if err == nil || err.code != payloads.DetachVolumeInvalidData {
		t.Fatalf("DetachVolumeInvalidData error expected")
	}
}

// Verify the parseStartPayload function.
//
// The function is passed one valid payload and a number of invalid payloads.
//
// No error should be returned for the valid payload.  The resulting vmConfig
// structure should match the handcrafted structure associated with the
// payload.  The invalid payloads should fail to parse.
func TestParseStartPayload(t *testing.T) {
	for i, st := range startTests {
		cfg, err := parseStartPayload([]byte(st.payload))
		if st.config == nil {
			if cfg != nil {
				t.Errorf("Expected nil config due to bad payload %d", i)
			}
			continue
		}
		if err != nil {
			t.Errorf("Failed to parse payload %d : %v", i, err)
			continue
		}
		if !reflect.DeepEqual(cfg, st.config) {
			t.Errorf("Start payload %d does not match", i)
		}
	}
}

func compareNetEvents(t *testing.T, ev *libsnnet.SsntpEventInfo, eventData *payloads.TenantAddedEvent) {
	if eventData.AgentUUID != testutil.AgentUUID ||
		eventData.AgentIP != ev.CnIP ||
		eventData.TenantUUID != ev.TenantID ||
		eventData.TenantSubnet != ev.SubnetID ||
		eventData.ConcentratorUUID != ev.ConcID ||
		eventData.ConcentratorIP != ev.CnciIP ||
		eventData.SubnetKey != ev.SubnetKey {
		t.Errorf("payloads and ssntp events do not match")
	}
}

// Verify that the generateNetEventPayload parses payloads correctly.
//
// Two valid payloads are passed to generateNetEventPayload, the first
// representing a payloads.EventTenantAdded event, the second a
// payloads.EventTenantRemoved event.  Finally, an event with an invalid
// id is parsed.
//
// Both valid payloads should be parsed correctly and the resulting
// payloads data structures should have the correct contents.  An
// error should be generated when parsing the payload with the invalid
// event id.
func TestGenerateNetEventPayload(t *testing.T) {
	ev := &libsnnet.SsntpEventInfo{
		Event:     libsnnet.SsntpTunAdd,
		CnciIP:    testutil.CNCIIP,
		CnIP:      testutil.InstancePrivateIP,
		Subnet:    testutil.TenantSubnet,
		TenantID:  testutil.TenantUUID,
		SubnetID:  "",
		ConcID:    testutil.CNCIUUID,
		CnID:      testutil.AgentUUID,
		SubnetKey: 1,
	}

	pl, err := generateNetEventPayload(ev, testutil.AgentUUID)
	if err != nil {
		t.Fatalf("Failed to generate payload : %v", err)
	}
	var eventData payloads.EventTenantAdded
	err = yaml.Unmarshal(pl, &eventData)
	if err != nil {
		t.Fatalf("Unable to unmarshall event : %v", err)
	}

	compareNetEvents(t, ev, &eventData.TenantAdded)

	ev.Event = libsnnet.SsntpTunDel
	pl, err = generateNetEventPayload(ev, testutil.AgentUUID)
	if err != nil {
		t.Fatalf("Failed to generate payload : %v", err)
	}
	var eventData2 payloads.EventTenantRemoved
	err = yaml.Unmarshal(pl, &eventData2)
	if err != nil {
		t.Fatalf("Unable to unmarshall event : %v", err)
	}

	compareNetEvents(t, ev, &eventData2.TenantRemoved)

	ev.Event = 666
	_, err = generateNetEventPayload(ev, testutil.AgentUUID)
	if err == nil {
		t.Errorf("generateNetEventPayload should have failed on invalid event type")
	}
}

// Check that parseDeletePayload works correctly.
//
// Parse a valid delete payload.
//
// The payload should parse without any error and the instance UUID in the
// resulting payloads data structure should be as expected.
func TestParseDeletePayload(t *testing.T) {
	instance, stop, err := parseDeletePayload([]byte(testutil.DeleteYaml))
	if err != nil {
		t.Fatalf("Failed to parse delete payload : %v", err.err)
	}
	if instance != testutil.InstanceUUID {
		t.Errorf("Wrong instance UUID.  Expected %s found %s", instance,
			testutil.InstanceUUID)
	}
	if stop {
		t.Errorf("Expected stop to be false")
	}
}
