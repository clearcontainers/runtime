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
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	datastore "github.com/01org/ciao/ciao-controller/internal/datastore"
	"github.com/01org/ciao/ciao-controller/types"
	image "github.com/01org/ciao/ciao-image/client"
	"github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/openstack/block"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/ssntp/uuid"
	"github.com/01org/ciao/testutil"
)

func addTestTenant() (tenant *types.Tenant, err error) {
	/* add a new tenant */
	tuuid := uuid.Generate()
	tenant, err = ctl.ds.AddTenant(tuuid.String())
	if err != nil {
		return
	}

	// Add fake CNCI
	err = ctl.ds.AddTenantCNCI(tuuid.String(), uuid.Generate().String(), tenant.CNCIMAC)
	if err != nil {
		return
	}
	err = ctl.ds.AddCNCIIP(tenant.CNCIMAC, "192.168.0.1")
	if err != nil {
		return
	}
	return
}

func addComputeTestTenant() (tenant *types.Tenant, err error) {
	/* add a new tenant */
	tenant, err = ctl.ds.AddTenant(testutil.ComputeUser)
	if err != nil {
		return
	}

	// Add fake CNCI
	err = ctl.ds.AddTenantCNCI(testutil.ComputeUser, uuid.Generate().String(), tenant.CNCIMAC)
	if err != nil {
		return
	}

	err = ctl.ds.AddCNCIIP(tenant.CNCIMAC, "192.168.0.2")
	if err != nil {
		return
	}

	return
}

func BenchmarkStartSingleWorkload(b *testing.B) {
	var err error

	/* add a new tenant */
	tuuid := uuid.Generate()
	tenant, err := ctl.ds.AddTenant(tuuid.String())
	if err != nil {
		b.Error(err)
	}

	// Add fake CNCI
	err = ctl.ds.AddTenantCNCI(tuuid.String(), uuid.Generate().String(), tenant.CNCIMAC)
	if err != nil {
		b.Error(err)
	}
	err = ctl.ds.AddCNCIIP(tenant.CNCIMAC, "192.168.0.1")
	if err != nil {
		b.Error(err)
	}

	// get workload ID
	wls, err := ctl.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		b.Fatal(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		w := types.WorkloadRequest{
			WorkloadID: wls[0].ID,
			TenantID:   tuuid.String(),
			Instances:  1,
		}
		_, err = ctl.startWorkload(w)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkStart1000Workload(b *testing.B) {
	var err error

	/* add a new tenant */
	tuuid := uuid.Generate()
	tenant, err := ctl.ds.AddTenant(tuuid.String())
	if err != nil {
		b.Error(err)
	}

	// Add fake CNCI
	err = ctl.ds.AddTenantCNCI(tuuid.String(), uuid.Generate().String(), tenant.CNCIMAC)
	if err != nil {
		b.Error(err)
	}
	err = ctl.ds.AddCNCIIP(tenant.CNCIMAC, "192.168.0.1")
	if err != nil {
		b.Error(err)
	}

	// get workload ID
	wls, err := ctl.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		b.Fatal(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		w := types.WorkloadRequest{
			WorkloadID: wls[0].ID,
			TenantID:   tuuid.String(),
			Instances:  1000,
		}
		_, err = ctl.startWorkload(w)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkNewConfig(b *testing.B) {
	var err error

	tenant, err := addTestTenant()
	if err != nil {
		b.Error(err)
	}

	// get workload ID
	wls, err := ctl.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		b.Fatal(err)
	}

	id := uuid.Generate()

	b.ResetTimer()
	noVolumes := []storage.BlockDevice{}
	for n := 0; n < b.N; n++ {
		_, err := newConfig(ctl, wls[0], id.String(), tenant.ID, noVolumes)
		if err != nil {
			b.Error(err)
		}
	}
}

func TestTenantWithinBounds(t *testing.T) {
	var err error

	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	/* put tenant limit of 1 instance */
	err = ctl.ds.AddLimit(tenant.ID, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	wls, err := ctl.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	w := types.WorkloadRequest{
		WorkloadID: wls[0].ID,
		TenantID:   tenant.ID,
		Instances:  1,
	}
	_, err = ctl.startWorkload(w)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTenantOutOfBounds(t *testing.T) {
	var err error

	/* add a new tenant */
	tenant, err := addTestTenant()
	if err != nil {
		t.Error(err)
	}

	/* put tenant limit of 1 instance */
	err = ctl.ds.AddLimit(tenant.ID, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	wls, err := ctl.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	/* try to send 2 workload start commands */
	w := types.WorkloadRequest{
		WorkloadID: wls[0].ID,
		TenantID:   tenant.ID,
		Instances:  2,
	}
	_, err = ctl.startWorkload(w)
	if err == nil {
		t.Errorf("Not tracking limits correctly")
	}
}

// TestNewTenantHardwareAddr
// Confirm that the mac addresses generated from a given
// IP address is as expected.
func TestNewTenantHardwareAddr(t *testing.T) {
	ip := net.ParseIP("172.16.0.2")
	expectedMAC := "02:00:ac:10:00:02"
	hw := newTenantHardwareAddr(ip)
	if hw.String() != expectedMAC {
		t.Error("Expected: ", expectedMAC, " Received: ", hw.String())
	}
}

func TestStartWorkload(t *testing.T) {
	var reason payloads.StartFailureReason

	client, _ := testStartWorkload(t, 1, false, reason)
	defer client.Shutdown()
}

func TestStartTracedWorkload(t *testing.T) {
	client := testStartTracedWorkload(t)
	defer client.Shutdown()
}

func TestStartWorkloadLaunchCNCI(t *testing.T) {
	netClient, instances := testStartWorkloadLaunchCNCI(t, 1)
	defer netClient.Shutdown()

	id := instances[0].TenantID

	tenant, err := ctl.ds.GetTenant(id)
	if err != nil {
		t.Fatal(err)
	}

	if tenant.CNCIIP == "" {
		t.Fatal("CNCI Info not updated")
	}

}

func sendTraceReportEvent(client *testutil.SsntpTestClient, t *testing.T) {
	clientCh := client.AddEventChan(ssntp.TraceReport)
	serverCh := server.AddEventChan(ssntp.TraceReport)
	go client.SendTrace()
	_, err := client.GetEventChanResult(clientCh, ssntp.TraceReport)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetEventChanResult(serverCh, ssntp.TraceReport)
	if err != nil {
		t.Fatal(err)
	}
}

func sendStatsCmd(client *testutil.SsntpTestClient, t *testing.T) {
	clientCh := client.AddCmdChan(ssntp.STATS)
	serverCh := server.AddCmdChan(ssntp.STATS)
	controllerCh := wrappedClient.addCmdChan(ssntp.STATS)
	go client.SendStatsCmd()
	_, err := client.GetCmdChanResult(clientCh, ssntp.STATS)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetCmdChanResult(serverCh, ssntp.STATS)
	if err != nil {
		t.Fatal(err)
	}
	err = wrappedClient.getCmdChan(controllerCh, ssntp.STATS)
	if err != nil {
		t.Fatal(err)
	}
}

// TBD: for the launch CNCI tests, I really need to create a fake
// network node and test that way.

func TestDeleteInstance(t *testing.T) {
	var reason payloads.StartFailureReason

	client, instances := testStartWorkload(t, 1, false, reason)
	defer client.Shutdown()

	sendStatsCmd(client, t)

	serverCh := server.AddCmdChan(ssntp.DELETE)

	err := ctl.deleteInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := server.GetCmdChanResult(serverCh, ssntp.DELETE)
	if err != nil {
		t.Fatal(err)
	}
	if result.InstanceUUID != instances[0].ID {
		t.Fatal("Did not get correct Instance ID")
	}
}

func TestStopInstance(t *testing.T) {
	var reason payloads.StartFailureReason

	client, instances := testStartWorkload(t, 1, false, reason)
	defer client.Shutdown()

	sendStatsCmd(client, t)

	serverCh := server.AddCmdChan(ssntp.STOP)

	err := ctl.stopInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := server.GetCmdChanResult(serverCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
	if result.InstanceUUID != instances[0].ID {
		t.Fatal("Did not get correct Instance ID")
	}
}

func TestRestartInstance(t *testing.T) {
	var reason payloads.StartFailureReason

	client, instances := testStartWorkload(t, 1, false, reason)
	defer client.Shutdown()

	sendStatsCmd(client, t)

	serverCh := server.AddCmdChan(ssntp.STOP)
	clientCh := client.AddCmdChan(ssntp.STOP)

	err := ctl.stopInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := server.GetCmdChanResult(serverCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetCmdChanResult(clientCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
	if result.InstanceUUID != instances[0].ID {
		t.Fatal("Did not get correct Instance ID")
	}

	// now attempt to restart

	sendStatsCmd(client, t)

	serverCh = server.AddCmdChan(ssntp.RESTART)

	err = ctl.restartInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	result, err = server.GetCmdChanResult(serverCh, ssntp.RESTART)
	if err != nil {
		t.Fatal(err)
	}
	if result.InstanceUUID != instances[0].ID {
		t.Fatal("Did not get correct Instance ID")
	}
}

func TestEvacuateNode(t *testing.T) {
	client, err := testutil.NewSsntpTestClientConnection("EvacuateNode", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Shutdown()

	serverCh := server.AddCmdChan(ssntp.EVACUATE)

	// ok to not send workload first?

	err = ctl.evacuateNode(client.UUID)
	if err != nil {
		t.Error(err)
	}

	result, err := server.GetCmdChanResult(serverCh, ssntp.EVACUATE)
	if err != nil {
		t.Fatal(err)
	}
	if result.NodeUUID != client.UUID {
		t.Fatal("Did not get node ID")
	}
}

func TestAttachVolume(t *testing.T) {
	client, err := testutil.NewSsntpTestClientConnection("AttachVolume", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Ssntp.Close()

	serverCh := server.AddCmdChan(ssntp.AttachVolume)

	// ok to not send workload first?

	err = ctl.client.attachVolume("volID", "instanceID", client.UUID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := server.GetCmdChanResult(serverCh, ssntp.AttachVolume)
	if err != nil {
		t.Fatal(err)
	}

	if result.NodeUUID != client.UUID {
		t.Fatal("Did not get node ID")
	}

	if result.VolumeUUID != "volID" {
		t.Fatal("Did not get volume ID")
	}

	if result.InstanceUUID != "instanceID" {
		t.Fatal("Did not get instance ID")
	}
}

func TestDetachVolume(t *testing.T) {
	client, err := testutil.NewSsntpTestClientConnection("DetachVolume", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Ssntp.Close()

	serverCh := server.AddCmdChan(ssntp.DetachVolume)

	// ok to not send workload first?

	err = ctl.client.detachVolume("volID", "instanceID", client.UUID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := server.GetCmdChanResult(serverCh, ssntp.DetachVolume)
	if err != nil {
		t.Fatal(err)
	}

	if result.NodeUUID != client.UUID {
		t.Fatal("Did not get node ID")
	}

	if result.VolumeUUID != "volID" {
		t.Fatal("Did not get volume ID")
	}

	if result.InstanceUUID != "instanceID" {
		t.Fatal("Did not get instance ID")
	}
}

func addTestBlockDevice(t *testing.T, tenantID string) types.BlockData {
	bd, err := ctl.CreateBlockDevice("", "", 0)
	if err != nil {
		t.Fatal(err)
	}

	data := types.BlockData{
		BlockDevice: bd,
		CreateTime:  time.Now(),
		TenantID:    tenantID,
		State:       types.Available,
	}

	err = ctl.ds.AddBlockDevice(data)
	if err != nil {
		ctl.DeleteBlockDevice(bd.ID)
		t.Fatal(err)
	}

	return data
}

// Note: caller should close ssntp client
func doAttachVolumeCommand(t *testing.T, fail bool) (client *testutil.SsntpTestClient, tenant string, volume string) {
	var reason payloads.StartFailureReason

	client, instances := testStartWorkload(t, 1, false, reason)

	tenantID := instances[0].TenantID

	sendStatsCmd(client, t)

	data := addTestBlockDevice(t, tenantID)

	serverCh := server.AddCmdChan(ssntp.AttachVolume)
	agentCh := client.AddCmdChan(ssntp.AttachVolume)
	var serverErrorCh chan testutil.Result
	var controllerCh chan struct{}

	if fail == true {
		serverErrorCh = server.AddErrorChan(ssntp.AttachVolumeFailure)
		controllerCh = wrappedClient.addErrorChan(ssntp.AttachVolumeFailure)
		client.AttachFail = true
		client.AttachVolumeFailReason = payloads.AttachVolumeAlreadyAttached

		defer func() {
			client.AttachFail = false
			client.AttachVolumeFailReason = ""
		}()
	}

	err := ctl.AttachVolume(tenantID, data.ID, instances[0].ID, "")
	if err != nil {
		t.Fatal(err)
	}

	result, err := server.GetCmdChanResult(serverCh, ssntp.AttachVolume)
	if err != nil {
		t.Fatal(err)
	}

	if result.InstanceUUID != instances[0].ID ||
		result.NodeUUID != client.UUID ||
		result.VolumeUUID != data.ID {
		t.Fatalf("expected %s %s %s, got %s %s %s", instances[0].ID, client.UUID, data.ID, result.InstanceUUID, result.NodeUUID, result.VolumeUUID)
	}

	if fail == true {
		_, err = client.GetCmdChanResult(agentCh, ssntp.AttachVolume)
		if err == nil {
			t.Fatal("Success when Failure expected")
		}

		_, err = server.GetErrorChanResult(serverErrorCh, ssntp.AttachVolumeFailure)
		if err != nil {
			t.Fatal(err)
		}

		err = wrappedClient.getErrorChan(controllerCh, ssntp.AttachVolumeFailure)
		if err != nil {
			t.Fatal(err)
		}

		// at this point, the state of the block device should
		// be set back to available.
		data2, err := ctl.ds.GetBlockDevice(data.ID)
		if err != nil {
			t.Fatal(err)
		}

		if data2.State != types.Available {
			t.Fatalf("block device state not updated")
		}
	} else {
		_, err = client.GetCmdChanResult(agentCh, ssntp.AttachVolume)
		if err != nil {
			t.Fatal(err)
		}
	}

	return client, tenantID, data.ID
}

func TestAttachVolumeCommand(t *testing.T) {
	client, _, _ := doAttachVolumeCommand(t, false)
	client.Ssntp.Close()
}

func TestAttachVolumeFailure(t *testing.T) {
	client, _, _ := doAttachVolumeCommand(t, true)
	client.Ssntp.Close()
}

func doDetachVolumeCommand(t *testing.T, fail bool) {
	// attach volume should succeed for this test
	client, tenantID, volume := doAttachVolumeCommand(t, false)
	defer client.Ssntp.Close()

	sendStatsCmd(client, t)

	serverCh := server.AddCmdChan(ssntp.DetachVolume)
	agentCh := client.AddCmdChan(ssntp.DetachVolume)
	var serverErrorCh chan testutil.Result
	var controllerCh chan struct{}

	if fail == true {
		serverErrorCh = server.AddErrorChan(ssntp.DetachVolumeFailure)
		controllerCh = wrappedClient.addErrorChan(ssntp.DetachVolumeFailure)
		client.DetachFail = true
		client.DetachVolumeFailReason = payloads.DetachVolumeNotAttached

		defer func() {
			client.DetachFail = false
			client.DetachVolumeFailReason = ""
		}()
	}

	err := ctl.DetachVolume(tenantID, volume, "")
	if err != nil {
		t.Fatal(err)
	}

	result, err := server.GetCmdChanResult(serverCh, ssntp.DetachVolume)
	if err != nil {
		t.Fatal(err)
	}

	if result.NodeUUID != client.UUID ||
		result.VolumeUUID != volume {
		t.Fatalf("expected %s %s , got %s %s ", client.UUID, volume, result.NodeUUID, result.VolumeUUID)
	}

	// at this point, the state of the volume should be "detaching"
	data, err := ctl.ds.GetBlockDevice(volume)
	if err != nil {
		t.Fatal(err)
	}

	if data.State != types.Detaching {
		t.Fatalf("expected state %s, got %s\n", types.Detaching, data.State)
	}

	if fail == true {
		_, err = client.GetCmdChanResult(agentCh, ssntp.DetachVolume)
		if err == nil {
			t.Fatal("Success when Failure expected")
		}

		_, err = server.GetErrorChanResult(serverErrorCh, ssntp.DetachVolumeFailure)
		if err != nil {
			t.Fatal(err)
		}

		err = wrappedClient.getErrorChan(controllerCh, ssntp.DetachVolumeFailure)
		if err != nil {
			t.Fatal(err)
		}

		// at this point, the state of the block device should
		// be set back to InUse

		data2, err := ctl.ds.GetBlockDevice(volume)
		if err != nil {
			t.Fatal(err)
		}

		if data2.State != types.InUse {
			t.Fatalf("expected state %s, got %s\n", types.InUse, data2.State)
		}
	} else {
		_, err = client.GetCmdChanResult(agentCh, ssntp.DetachVolume)
		if err != nil {
			t.Fatal(err)
		}
	}

	return
}

func TestDetachVolumeCommand(t *testing.T) {
	doDetachVolumeCommand(t, false)
}

func TestDetachVolumeFailure(t *testing.T) {
	doDetachVolumeCommand(t, true)
}

func TestDetachVolumeByAttachment(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	err = ctl.DetachVolume(tenant.ID, "invalidVolume", "attachmentID")
	if err == nil {
		t.Fatal("Detach by attachment ID not supported yet")
	}
}

func TestInstanceDeletedEvent(t *testing.T) {
	var reason payloads.StartFailureReason

	client, instances := testStartWorkload(t, 1, false, reason)
	defer client.Shutdown()

	sendStatsCmd(client, t)

	serverCh := server.AddCmdChan(ssntp.DELETE)

	err := ctl.deleteInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = server.GetCmdChanResult(serverCh, ssntp.DELETE)
	if err != nil {
		t.Fatal(err)
	}

	clientEvtCh := client.AddEventChan(ssntp.InstanceDeleted)
	serverEvtCh := server.AddEventChan(ssntp.InstanceDeleted)
	controllerCh := wrappedClient.addEventChan(ssntp.InstanceDeleted)
	go client.SendDeleteEvent(instances[0].ID)
	_, err = client.GetEventChanResult(clientEvtCh, ssntp.InstanceDeleted)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetEventChanResult(serverEvtCh, ssntp.InstanceDeleted)
	if err != nil {
		t.Fatal(err)
	}
	err = wrappedClient.getEventChan(controllerCh, ssntp.InstanceDeleted)
	if err != nil {
		t.Fatal(err)
	}

	// try to get instance info
	_, err = ctl.ds.GetInstance(instances[0].ID)
	if err == nil {
		t.Error("Instance not deleted")
	}
}

func TestStartFailure(t *testing.T) {
	reason := payloads.FullCloud

	client, _ := testStartWorkload(t, 1, true, reason)
	defer client.Shutdown()

	// since we had a start failure, we should confirm that the
	// instance is no longer pending in the database
}

func TestStopFailure(t *testing.T) {
	ctl.ds.ClearLog()

	var reason payloads.StartFailureReason

	client, instances := testStartWorkload(t, 1, false, reason)
	defer client.Shutdown()

	client.StopFail = true
	client.StopFailReason = payloads.StopNoInstance

	sendStatsCmd(client, t)

	serverCh := server.AddCmdChan(ssntp.STOP)
	controllerCh := wrappedClient.addErrorChan(ssntp.StopFailure)

	err := ctl.stopInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := server.GetCmdChanResult(serverCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
	err = wrappedClient.getErrorChan(controllerCh, ssntp.StopFailure)
	if err != nil {
		t.Fatal(err)
	}
	if result.InstanceUUID != instances[0].ID {
		t.Fatal("Did not get correct Instance ID")
	}

	// the response to a stop failure is to log the failure
	entries, err := ctl.ds.GetEventLog()
	if err != nil {
		t.Fatal(err)
	}

	expectedMsg := fmt.Sprintf("Stop Failure %s: %s", instances[0].ID, client.StopFailReason.String())

	for i := range entries {
		if entries[i].Message == expectedMsg {
			return
		}
	}
	t.Error("Did not find failure message in Log")
}

func TestRestartFailure(t *testing.T) {
	ctl.ds.ClearLog()

	var reason payloads.StartFailureReason

	client, instances := testStartWorkload(t, 1, false, reason)
	defer client.Shutdown()

	client.RestartFail = true
	client.RestartFailReason = payloads.RestartLaunchFailure

	sendStatsCmd(client, t)

	serverCh := server.AddCmdChan(ssntp.STOP)
	clientCh := client.AddCmdChan(ssntp.STOP)

	err := ctl.stopInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GetCmdChanResult(clientCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
	result, err := server.GetCmdChanResult(serverCh, ssntp.STOP)
	if err != nil {
		t.Fatal(err)
	}
	if result.InstanceUUID != instances[0].ID {
		t.Fatal("Did not get correct Instance ID")
	}

	sendStatsCmd(client, t)

	serverCh = server.AddCmdChan(ssntp.RESTART)
	controllerCh := wrappedClient.addErrorChan(ssntp.RestartFailure)

	err = ctl.restartInstance(instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	result, err = server.GetCmdChanResult(serverCh, ssntp.RESTART)
	if err != nil {
		t.Fatal(err)
	}
	err = wrappedClient.getErrorChan(controllerCh, ssntp.RestartFailure)
	if err != nil {
		t.Fatal(err)
	}
	if result.InstanceUUID != instances[0].ID {
		t.Fatal("Did not get correct Instance ID")
	}

	// the response to a restart failure is to log the failure
	entries, err := ctl.ds.GetEventLog()
	if err != nil {
		t.Fatal(err)
	}

	expectedMsg := fmt.Sprintf("Restart Failure %s: %s", instances[0].ID, client.RestartFailReason.String())

	for i := range entries {
		if entries[i].Message == expectedMsg {
			return
		}
	}
	t.Error("Did not find failure message in Log")
}

func TestNoNetwork(t *testing.T) {
	nn := true

	noNetwork = &nn

	var reason payloads.StartFailureReason

	client, _ := testStartWorkload(t, 1, false, reason)
	defer client.Shutdown()
}

// NOTE: the caller is responsible for calling Shutdown() on the *SsntpTestClient
func testStartTracedWorkload(t *testing.T) *testutil.SsntpTestClient {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	client, err := testutil.NewSsntpTestClientConnection("StartTracedWorkload", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	// caller of TestStartTracedWorkload() owns doing the close
	//defer client.Shutdown()

	wls, err := ctl.ds.GetWorkloads()
	if err != nil {
		t.Fatal(err)
	}
	if len(wls) == 0 {
		t.Fatal("No workloads, expected len(wls) > 0, got len(wls) == 0")
	}

	clientCh := client.AddCmdChan(ssntp.START)
	serverCh := server.AddCmdChan(ssntp.START)

	w := types.WorkloadRequest{
		WorkloadID: wls[0].ID,
		TenantID:   tenant.ID,
		Instances:  1,
		TraceLabel: "testtrace",
	}
	instances, err := ctl.startWorkload(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 {
		t.Fatalf("Wrong number of instances, expected 1, got %d", len(instances))
	}

	_, err = client.GetCmdChanResult(clientCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
	result, err := server.GetCmdChanResult(serverCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
	if result.InstanceUUID != instances[0].ID {
		t.Fatal("Did not get correct Instance ID")
	}

	return client
}

// NOTE: the caller is responsible for calling Shutdown() on the *SsntpTestClient
func testStartWorkload(t *testing.T, num int, fail bool, reason payloads.StartFailureReason) (*testutil.SsntpTestClient, []*types.Instance) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	client, err := testutil.NewSsntpTestClientConnection("StartWorkload", ssntp.AGENT, testutil.AgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	// caller of TestStartWorkload() owns doing the close
	//defer client.Shutdown()

	wls, err := ctl.ds.GetWorkloads()
	if err != nil {
		t.Fatal(err)
	}
	if len(wls) == 0 {
		t.Fatal("No workloads, expected len(wls) > 0, got len(wls) == 0")
	}

	clientCmdCh := client.AddCmdChan(ssntp.START)
	clientErrCh := client.AddErrorChan(ssntp.StartFailure)
	client.StartFail = fail
	client.StartFailReason = reason

	w := types.WorkloadRequest{
		WorkloadID: wls[0].ID,
		TenantID:   tenant.ID,
		Instances:  num,
	}
	instances, err := ctl.startWorkload(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != num {
		t.Fatalf("Wrong number of instances, expected %d, got %d", len(instances), num)
	}

	if fail == true {
		_, err := client.GetErrorChanResult(clientErrCh, ssntp.StartFailure)
		if err == nil { // unexpected success
			t.Fatal(err)
		}
	}

	result, err := client.GetCmdChanResult(clientCmdCh, ssntp.START)
	if fail == true && err == nil { // unexpected success
		t.Fatal(err)
	}
	if fail == false && err != nil { // unexpected failure
		t.Fatal(err)
	}
	if result.InstanceUUID != instances[0].ID {
		t.Fatal("Did not get correct Instance ID")
	}

	return client, instances
}

// NOTE: the caller is responsible for calling Shutdown() on the *SsntpTestClient
func testStartWorkloadLaunchCNCI(t *testing.T, num int) (*testutil.SsntpTestClient, []*types.Instance) {
	netClient, err := testutil.NewSsntpTestClientConnection("StartWorkloadLaunchCNCI", ssntp.NETAGENT, testutil.NetAgentUUID)
	if err != nil {
		t.Fatal(err)
	}
	// caller of testStartWorkloadLaunchCNCI() owns doing the close
	//defer netClient.Shutdown()

	wls, err := ctl.ds.GetWorkloads()
	if err != nil {
		t.Fatal(err)
	}
	if len(wls) == 0 {
		t.Fatal("No workloads, expected len(wls) > 0, got len(wls) == 0")
	}

	serverCmdCh := server.AddCmdChan(ssntp.START)
	netClientCmdCh := netClient.AddCmdChan(ssntp.START)

	newTenant := uuid.Generate().String() // random ~= new tenant and thus triggers start of a CNCI

	// trigger the START command flow, and await results
	instanceCh := make(chan []*types.Instance)

	go func() {
		w := types.WorkloadRequest{
			WorkloadID: wls[0].ID,
			TenantID:   newTenant,
			Instances:  1,
		}
		instances, err := ctl.startWorkload(w)
		if err != nil {
			t.Fatal(err)
		}

		if len(instances) != 1 {
			t.Fatalf("Wrong number of instances, expected 1, got %d", len(instances))
		}

		instanceCh <- instances
	}()

	_, err = netClient.GetCmdChanResult(netClientCmdCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}
	result, err := server.GetCmdChanResult(serverCmdCh, ssntp.START)
	if err != nil {
		t.Fatal(err)
	}

	if result.TenantUUID != newTenant {
		t.Fatal("Did not get correct tenant ID")
	}

	if !result.CNCI {
		t.Fatal("this is not a CNCI launch request")
	}

	// start a test CNCI client
	cnciClient, err := testutil.NewSsntpTestClientConnection("StartWorkloadLaunchCNCI", ssntp.CNCIAGENT, newTenant)
	if err != nil {
		t.Fatal(err)
	}

	// make CNCI send an ssntp.ConcentratorInstanceAdded event, and await results
	cnciEventCh := cnciClient.AddEventChan(ssntp.ConcentratorInstanceAdded)
	serverEventCh := server.AddEventChan(ssntp.ConcentratorInstanceAdded)
	tenantCNCI, _ := ctl.ds.GetTenantCNCISummary(result.InstanceUUID)
	go cnciClient.SendConcentratorAddedEvent(result.InstanceUUID, newTenant, testutil.CNCIIP, tenantCNCI[0].MACAddress)
	result, err = cnciClient.GetEventChanResult(cnciEventCh, ssntp.ConcentratorInstanceAdded)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetEventChanResult(serverEventCh, ssntp.ConcentratorInstanceAdded)
	if err != nil {
		t.Fatal(err)
	}

	// shutdown the test CNCI client
	cnciClient.Shutdown()

	if result.InstanceUUID != tenantCNCI[0].InstanceID {
		t.Fatalf("Did not get correct Instance ID, got %s, expected %s", result.InstanceUUID, tenantCNCI[0].InstanceID)
	}

	instances := <-instanceCh
	if instances == nil {
		t.Fatal("did not receive instance")
	}

	return netClient, instances
}

func TestGetStorageForVolume(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	sourceVolume := addTestBlockDevice(t, tenant.ID)
	defer ctl.DeleteBlockDevice(sourceVolume.ID)

	// a temporary in memory filesystem?
	s := types.StorageResource{
		ID:         "",
		Bootable:   true,
		Ephemeral:  false,
		SourceType: types.VolumeService,
		SourceID:   sourceVolume.ID,
	}

	pl, err := getStorage(ctl, s, tenant.ID, "")
	if err != nil {
		t.Fatal(err)
	}

	if pl.ID == "" {
		t.Errorf("storage ID does not exist")
	}

	if pl.Bootable != true {
		t.Errorf("bootable flag not correct")
	}

	if pl.Ephemeral != false {
		t.Errorf("ephemeral flag not correct")
	}

	createdVolume, err := ctl.ds.GetBlockDevice(pl.ID)

	if err != nil {
		t.Fatal(err)
	}

	if len(createdVolume.Name) == 0 {
		t.Errorf("block device name not set")
	}
}

func TestGetStorageForImage(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	// add fake image to images store
	//
	tmpfile, err := ioutil.TempFile(ctl.image.MountPoint, "testImage")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// a temporary in memory filesystem?
	s := types.StorageResource{
		ID:         "",
		Bootable:   true,
		Ephemeral:  false,
		SourceType: types.ImageService,
		SourceID:   filepath.Base(tmpfile.Name()),
	}

	pl, err := getStorage(ctl, s, tenant.ID, "")
	if err != nil {
		t.Fatal(err)
	}

	if pl.ID == "" {
		t.Errorf("storage ID does not exist")
	}

	if pl.Bootable != true {
		t.Errorf("bootable flag not correct")
	}

	if pl.Ephemeral != false {
		t.Errorf("ephemeral flag not correct")
	}

	createdVolume, err := ctl.ds.GetBlockDevice(pl.ID)

	if err != nil {
		t.Fatal(err)
	}

	if len(createdVolume.Name) == 0 {
		t.Errorf("block device name not set")
	}
}

func TestStorageConfig(t *testing.T) {
	var err error

	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	// get workload ID
	wls, err := ctl.ds.GetWorkloads()
	if err != nil || len(wls) == 0 {
		t.Fatal(err)
	}

	tmpfile, err := ioutil.TempFile(ctl.image.MountPoint, "test-image")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	info, err := tmpfile.Stat()
	if err != nil {
		t.Fatal(err)
	}

	// a temporary in memory filesystem?
	s := types.StorageResource{
		ID:         "",
		Bootable:   true,
		Ephemeral:  false,
		SourceType: types.ImageService,
		SourceID:   info.Name(),
	}

	wls[0].Storage = []types.StorageResource{s}

	id := uuid.Generate()

	noVolumes := []storage.BlockDevice{}
	_, err = newConfig(ctl, wls[0], id.String(), tenant.ID, noVolumes)
	if err != nil {
		t.Fatal(err)
	}

	wls[0].Storage = []types.StorageResource{}
}

func createTestVolume(tenantID string, size int, t *testing.T) string {
	req := block.RequestedVolume{
		Size: size,
	}

	vol, err := ctl.CreateVolume(tenantID, req)
	if err != nil {
		t.Fatal(err)
	}

	if vol.UserID != tenantID || vol.Status != block.Available ||
		vol.Size != size || vol.Bootable != "false" {
		t.Fatalf("incorrect volume returned\n")
	}

	return vol.ID
}

func TestCreateVolume(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	volID := createTestVolume(tenant.ID, 20, t)

	// confirm that we can retrieve the volume from
	// the datastore.
	bd, err := ctl.ds.GetBlockDevice(volID)
	if err != nil {
		t.Fatal(err)
	}

	if bd.State != types.Available || bd.TenantID != tenant.ID {
		t.Fatalf("incorrect volume information stored\n")
	}
}

func TestDeleteVolume(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	volID := createTestVolume(tenant.ID, 20, t)

	// confirm that we can retrieve the volume from
	// the datastore.
	_, err = ctl.ds.GetBlockDevice(volID)
	if err != nil {
		t.Fatal(err)
	}

	// attempt to delete invalid volume
	err = ctl.DeleteVolume(tenant.ID, "badID")
	if err != datastore.ErrNoBlockData {
		t.Fatal("Incorrect error")
	}

	// attempt to delete with bad tenant ID
	err = ctl.DeleteVolume("badID", volID)
	if err != block.ErrVolumeOwner {
		t.Fatal("Incorrect error")
	}

	// this should work
	err = ctl.DeleteVolume(tenant.ID, volID)
	if err != nil {
		t.Fatal(err)
	}

	// confirm that we cannot retrieve the volume from
	// the datastore.
	_, err = ctl.ds.GetBlockDevice(volID)
	if err != datastore.ErrNoBlockData {
		t.Fatal(err)
	}
}

func TestListVolumes(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	_ = createTestVolume(tenant.ID, 20, t)

	vols, err := ctl.ListVolumes(tenant.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(vols) != 1 {
		t.Fatal("Incorrect number of volumes returned")
	}
}

func TestShowVolumeDetails(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	volID := createTestVolume(tenant.ID, 20, t)

	vol, err := ctl.ShowVolumeDetails(tenant.ID, volID)
	if err != nil {
		t.Fatal(err)
	}

	if vol.ID != volID {
		t.Fatal("wrong volume retrieved")
	}
}

func TestListVolumesDetail(t *testing.T) {
	tenant, err := addTestTenant()
	if err != nil {
		t.Fatal(err)
	}

	_ = createTestVolume(tenant.ID, 20, t)

	vols, err := ctl.ListVolumesDetail(tenant.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(vols) != 1 {
		t.Fatal("Incorrect number of volumes returned")
	}
}

func testAddPool(t *testing.T, name string, subnet *string, ips []string) {
	pool, err := ctl.AddPool(name, subnet, ips)
	if err != nil {
		t.Fatal(err)
	}

	if pool.ID == "" {
		t.Fatal("id not set")
	}

	expected := types.Pool{
		ID:   pool.ID,
		Name: name,
	}

	if subnet != nil {
		if pool.Subnets[0].ID == "" {
			t.Fatal("subnet id not created")
		}

		sub := types.ExternalSubnet{
			ID:   pool.Subnets[0].ID,
			CIDR: *subnet,
		}

		expected.Subnets = []types.ExternalSubnet{sub}

		_, ipNet, err := net.ParseCIDR(*subnet)
		if err != nil {
			t.Fatal(err)
		}

		ones, bits := ipNet.Mask.Size()
		expected.TotalIPs = (1 << uint32(bits-ones)) - 2
		expected.Free = expected.TotalIPs
	} else if len(ips) > 0 {
		// not an easy way to check this, so we're going to
		// do some manual tests
		if pool.TotalIPs != len(ips) ||
			pool.Free != len(ips) ||
			len(pool.IPs) != len(ips) {
			t.Fatal("External IPs not handled correctly")
		}
		return
	}

	if reflect.DeepEqual(expected, pool) == false {
		t.Fatalf("expected %v, got %v\n", expected, pool)
	}
}

func deletePool(name string) error {
	pools, err := ctl.ListPools()
	if err != nil {
		return err
	}

	if len(pools) < 1 {
		return types.ErrPoolNotFound
	}

	for _, pool := range pools {
		if pool.Name == name {
			return ctl.DeletePool(pool.ID)
		}
	}

	return types.ErrPoolNotFound
}

func TestAddPoolWithSubnet(t *testing.T) {
	subnet := "192.168.0.0/16"
	testAddPool(t, "test1", &subnet, []string{})
	deletePool("test1")
}

func TestAddPoolWithIPs(t *testing.T) {
	ips := []string{"10.10.0.1", "10.10.0.2"}
	testAddPool(t, "test2", nil, ips)
	deletePool("test2")
}

func TestAddPool(t *testing.T) {
	testAddPool(t, "test3", nil, []string{})
	deletePool("test3")
}

func TestListPools(t *testing.T) {
	testAddPool(t, "listPoolTest", nil, []string{})

	pools, err := ctl.ListPools()
	if err != nil {
		t.Fatal(err)
	}

	if len(pools) < 1 {
		t.Fatal("Unable to retrieve pools")
	}

	for _, pool := range pools {
		if pool.Name == "listPoolTest" {
			err := ctl.DeletePool(pool.ID)
			if err != nil {
				t.Fatal(err)
			}
			return
		}
	}

	t.Fatal("Could not list pools")
}

func TestShowPool(t *testing.T) {
	testAddPool(t, "showPoolTest", nil, []string{})

	pools, err := ctl.ListPools()
	if err != nil {
		t.Fatal(err)
	}

	if len(pools) < 1 {
		t.Fatal("Unable to retrieve pools")
	}

	for _, pool := range pools {
		if pool.Name == "showPoolTest" {
			_, err := ctl.ShowPool(pool.ID)
			if err != nil {
				t.Fatal(err)
			}

			err = ctl.DeletePool(pool.ID)
			if err != nil {
				t.Fatal(err)
			}

			return
		}
	}

	t.Fatal("Could not show pool")
}

func TestDeletePool(t *testing.T) {
	testAddPool(t, "deletePoolTest", nil, []string{})

	pools, err := ctl.ListPools()
	if err != nil {
		t.Fatal(err)
	}

	if len(pools) < 1 {
		t.Fatal("Unable to retrieve pools")
	}

	for _, pool := range pools {
		if pool.Name == "deletePoolTest" {
			err := ctl.DeletePool(pool.ID)
			if err != nil {
				t.Fatal(err)
			}

			_, err = ctl.ShowPool(pool.ID)
			if err != types.ErrPoolNotFound {
				t.Fatal("Pool not deleted")
			}
			return
		}
	}

	t.Fatal("Could not delete pool")
}

func TestAddPoolSubnet(t *testing.T) {
	subnet := "192.168.0.0/24"

	testAddPool(t, "addsubnet", nil, []string{})

	pools, err := ctl.ListPools()
	if err != nil {
		t.Fatal(err)
	}

	if len(pools) < 1 {
		t.Fatal("Unable to retrieve pools")
	}

	for _, pool := range pools {
		if pool.Name == "addsubnet" {
			err := ctl.AddAddress(pool.ID, &subnet, []string{})
			if err != nil {
				t.Fatal(err)
			}

			p1, err := ctl.ShowPool(pool.ID)
			if err != nil {
				t.Fatal(err)
			}

			// we should have a our subnet.
			if p1.Subnets[0].CIDR != subnet {
				t.Fatalf("expectd %s subnet got %s", subnet, p1.Subnets[0].CIDR)
			}

			err = ctl.DeletePool(pool.ID)
			if err != nil {
				t.Fatal(err)
			}
			return
		}
	}

}

func TestAddPoolAddress(t *testing.T) {
	address := "192.168.1.1"

	testAddPool(t, "addaddress", nil, []string{})

	pools, err := ctl.ListPools()
	if err != nil {
		t.Fatal(err)
	}

	if len(pools) < 1 {
		t.Fatal("Unable to retrieve pools")
	}

	for _, pool := range pools {
		if pool.Name == "addaddress" {
			err := ctl.AddAddress(pool.ID, nil, []string{address})
			if err != nil {
				t.Fatal(err)
			}

			p1, err := ctl.ShowPool(pool.ID)
			if err != nil {
				t.Fatal(err)
			}

			// we should have a our subnet.
			if p1.IPs[0].Address != address {
				t.Fatalf("expected %s address got %s", address, p1.IPs[0].Address)
			}

			err = ctl.DeletePool(pool.ID)
			if err != nil {
				t.Fatal(err)
			}
			return
		}
	}

}

func TestRemovePoolSubnet(t *testing.T) {
	subnet := "192.168.0.0/24"
	address := "192.168.1.1"

	testAddPool(t, "addsubnet", &subnet, []string{})

	pools, err := ctl.ListPools()
	if err != nil {
		t.Fatal(err)
	}

	var pool types.Pool

	for _, pool = range pools {
		if pool.Name == "addsubnet" {
			// make sure the subnet is there.
			for _, sub := range pool.Subnets {
				if sub.CIDR == subnet {
					err := ctl.RemoveAddress(pool.ID, &sub.ID, nil)
					if err != nil {
						t.Fatalf("%s: %v\n", err, pool.Subnets)
					}
				}
			}
		}
		break
	}

	p1, err := ctl.ShowPool(pool.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(p1.Subnets) != 0 || p1.TotalIPs != 0 || p1.Free != 0 {
		fmt.Printf("pool %v\n", p1)
		t.Fatal("subnet not deleted")
	}

	err = ctl.AddAddress(pool.ID, nil, []string{address})
	if err != nil {
		t.Fatal(err)
	}

	p1, err = ctl.ShowPool(pool.ID)
	if err != nil {
		t.Fatal(err)
	}

	err = ctl.RemoveAddress(pool.ID, nil, &p1.IPs[0].ID)
	if err != nil {
		t.Fatalf("%s: %v\n", err, pool.IPs)
	}

	err = ctl.RemoveAddress(pool.ID, nil, nil)
	if err != types.ErrBadRequest {
		t.Fatal("invalid remove address request allowed")
	}
}

func TestMapAddress(t *testing.T) {
	var reason payloads.StartFailureReason

	client, instances := testStartWorkload(t, 1, false, reason)
	defer client.Shutdown()

	ips := []string{"10.10.0.1"}
	poolName := "testmap"

	testAddPool(t, poolName, nil, ips)

	pools, err := ctl.ListPools()
	if err != nil {
		t.Fatal(err)
	}

	if len(pools) < 1 {
		t.Fatal("Unable to retrieve pools")
	}

	for _, pool := range pools {
		if pool.Name == "testmap" {
			if pool.Free != 1 {
				t.Fatal("Pool Free not correct")
			}
		}
	}

	err = ctl.MapAddress(&poolName, instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	pools, err = ctl.ListPools()
	if err != nil {
		t.Fatal(err)
	}

	if len(pools) < 1 {
		t.Fatal("Unable to retrieve pools")
	}

	for _, pool := range pools {
		if pool.Name == "testmap" {
			if pool.Free != 0 {
				fmt.Printf("%v", pool)
				t.Fatal("Pool Free not decremented")
			}
		}
	}
}

func TestMapAddressNoPool(t *testing.T) {
	var reason payloads.StartFailureReason

	client, instances := testStartWorkload(t, 1, false, reason)
	defer client.Shutdown()

	ips := []string{"10.10.0.2"}
	poolName := "testmapnopool"

	testAddPool(t, poolName, nil, ips)

	err := ctl.MapAddress(nil, instances[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	pools, err := ctl.ListPools()
	if err != nil {
		t.Fatal(err)
	}

	if len(pools) < 1 {
		t.Fatal("Unable to retrieve pools")
	}

	for _, pool := range pools {
		if pool.Name == "testmapnopool" {
			if pool.Free != 0 {
				fmt.Printf("%v", pool)
				t.Fatal("Pool Free not decremented")
			}
		}
	}

	mappedIPs := ctl.ListMappedAddresses(&instances[0].TenantID)
	if len(mappedIPs) != 1 {
		t.Fatal("mapped IP not in list")
	}
}

var testClients []*testutil.SsntpTestClient
var ctl *controller
var server *testutil.SsntpTestServer
var wrappedClient *ssntpClientWrapper

func TestMain(m *testing.M) {
	flag.Parse()

	// create fake ssntp server
	server = testutil.StartTestServer()

	ctl = new(controller)
	ctl.tenantReadiness = make(map[string]*tenantConfirmMemo)
	ctl.ds = new(datastore.Datastore)

	ctl.BlockDriver = func() storage.BlockDriver {
		return &storage.NoopDriver{}
	}()

	dir, err := ioutil.TempDir("", "controller_test")
	if err != nil {
		os.Exit(1)
	}
	fakeImage := fmt.Sprintf("%s/73a86d7e-93c0-480e-9c41-ab42f69b7799", dir)

	f, err := os.Create(fakeImage)
	if err != nil {
		os.RemoveAll(dir)
		os.Exit(1)
	}

	ctl.image = image.Client{MountPoint: dir}

	dsConfig := datastore.Config{
		PersistentURI:     "file:memdb1?mode=memory&cache=shared",
		TransientURI:      "file:memdb2?mode=memory&cache=shared",
		InitTablesPath:    *tablesInitPath,
		InitWorkloadsPath: *workloadsPath,
	}

	err = ctl.ds.Init(dsConfig)
	if err != nil {
		f.Close()
		os.RemoveAll(dir)
		os.Exit(1)
	}

	config := &ssntp.Config{
		URI:    "localhost",
		CAcert: ssntp.DefaultCACert,
		Cert:   ssntp.RoleToDefaultCertName(ssntp.Controller),
	}

	wrappedClient, err = newWrappedSSNTPClient(ctl, config)
	if err != nil {
		os.Exit(1)
	}
	ctl.client = wrappedClient

	testIdentityConfig := testutil.IdentityConfig{
		ComputeURL: testutil.ComputeURL,
		ProjectID:  testutil.ComputeUser,
	}

	id := testutil.StartIdentityServer(testIdentityConfig)

	idConfig := identityConfig{
		endpoint:        id.URL,
		serviceUserName: "test",
		servicePassword: "iheartciao",
	}

	ctl.id, err = newIdentityClient(idConfig)
	if err != nil {
		fmt.Println(err)
		// keep going anyway - any compute api tests will fail.
	}

	_, _ = addComputeTestTenant()
	go ctl.startComputeService()
	time.Sleep(1 * time.Second)
	go ctl.startVolumeService()
	time.Sleep(1 * time.Second)

	code := m.Run()

	ctl.client.Disconnect()
	ctl.ds.Exit()
	id.Close()
	server.Shutdown()
	f.Close()
	os.RemoveAll(dir)

	os.Exit(code)
}
