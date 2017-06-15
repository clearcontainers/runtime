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
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type controllerClient interface {
	ssntp.ClientNotifier
	StartTracedWorkload(config string, startTime time.Time, label string) error
	StartWorkload(config string) error
	DeleteInstance(instanceID string, nodeID string) error
	StopInstance(instanceID string, nodeID string) error
	RestartInstance(i *types.Instance, w *types.Workload, t *types.Tenant) error
	EvacuateNode(nodeID string) error
	Disconnect()
	mapExternalIP(t types.Tenant, m types.MappedIP) error
	unMapExternalIP(t types.Tenant, m types.MappedIP) error
	attachVolume(volID string, instanceID string, nodeID string) error
	detachVolume(volID string, instanceID string, nodeID string) error
	ssntpClient() *ssntp.Client
}

type ssntpClient struct {
	ctl   *controller
	ssntp ssntp.Client
	name  string
}

func (client *ssntpClient) ConnectNotify() {
	glog.Info(client.name, " connected")
}

func (client *ssntpClient) DisconnectNotify() {
	glog.Info(client.name, " disconnected")
}

func (client *ssntpClient) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
	glog.Info("STATUS for ", client.name)
}

func (client *ssntpClient) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	var stats payloads.Stat
	payload := frame.Payload

	glog.Info("COMMAND ", command, " for ", client.name)

	if command == ssntp.STATS {
		stats.Init()
		err := yaml.Unmarshal(payload, &stats)
		if err != nil {
			glog.Warningf("Error unmarshalling STATS: %v", err)
			return
		}
		err = client.ctl.ds.HandleStats(stats)
		if err != nil {
			glog.Warningf("Error updating stats in datastore: %v", err)
		}
	}
	glog.V(1).Info(string(payload))
}

func (client *ssntpClient) deleteEphemeralStorage(instanceID string) {
	err := client.ctl.deleteEphemeralStorage(instanceID)
	if err != nil {
		glog.Warningf("Error deleting ephemeral storage for instance: %s: %v", instanceID, err)
	}
}

func (client *ssntpClient) releaseResources(instanceID string) error {
	i, err := client.ctl.ds.GetInstance(instanceID)
	if err != nil {
		return errors.Wrapf(err, "error getting instance from datastore")
	}

	// CNCI resources are not quota tracked
	if i.CNCI {
		return nil
	}

	wl, err := client.ctl.ds.GetWorkload(i.TenantID, i.WorkloadID)
	if err != nil {
		return errors.Wrapf(err, "error getting workload for instance from datastore")
	}

	resources := []payloads.RequestedResource{{Type: payloads.Instance, Value: 1}}
	resources = append(resources, wl.Defaults...)
	client.ctl.qs.Release(i.TenantID, resources...)
	return nil
}

func (client *ssntpClient) removeInstance(instanceID string) {
	err := client.releaseResources(instanceID)
	if err != nil {
		glog.Warningf("Error when releasing resources for deleted instance: %v", err)
	}
	client.deleteEphemeralStorage(instanceID)
	err = client.ctl.ds.DeleteInstance(instanceID)
	if err != nil {
		glog.Warningf("Error deleting instance from datastore: %v", err)
	}
}

func (client *ssntpClient) instanceDeleted(payload []byte) {
	var event payloads.EventInstanceDeleted
	err := yaml.Unmarshal(payload, &event)
	if err != nil {
		glog.Warning("Error unmarshalling InstanceDeleted: %v")
		return
	}
	client.removeInstance(event.InstanceDeleted.InstanceUUID)
}

func (client *ssntpClient) instanceStopped(payload []byte) {
	var event payloads.EventInstanceStopped
	err := yaml.Unmarshal(payload, &event)
	if err != nil {
		glog.Warning("Error unmarshalling InstanceStopped: %v")
		return
	}
	glog.Infof("Stopped instance %s", event.InstanceStopped.InstanceUUID)
	err = client.ctl.ds.StopInstance(event.InstanceStopped.InstanceUUID)
	if err != nil {
		glog.Warningf("Error stopping instance from datastore: %v", err)
	}
}

func (client *ssntpClient) instanceAdded(payload []byte) {
	var event payloads.EventConcentratorInstanceAdded
	err := yaml.Unmarshal(payload, &event)
	if err != nil {
		glog.Warning("Error unmarshalling EventConcentratorInstanceAdded: %v", err)
		return
	}
	newCNCI := event.CNCIAdded
	err = client.ctl.ds.AddCNCIIP(newCNCI.ConcentratorMAC, newCNCI.ConcentratorIP)
	if err != nil {
		glog.Warningf("Error adding CNCI IP to datastore: %v", err)
	}
}

func (client *ssntpClient) traceReport(payload []byte) {
	var trace payloads.Trace
	err := yaml.Unmarshal(payload, &trace)
	if err != nil {
		glog.Warningf("Error unmarshalling TraceReport: %v", err)
		return
	}
	err = client.ctl.ds.HandleTraceReport(trace)
	if err != nil {
		glog.Warningf("Error updating trace report in datastore: %v", err)
	}
}

func (client *ssntpClient) nodeConnected(payload []byte) {
	var nodeConnected payloads.NodeConnected
	err := yaml.Unmarshal(payload, &nodeConnected)
	if err != nil {
		glog.Warningf("Error unmarshalling NodeConnected: %v", err)
		return
	}
	glog.Infof("Node %s connected", nodeConnected.Connected.NodeUUID)

	client.ctl.ds.AddNode(nodeConnected.Connected.NodeUUID, nodeConnected.Connected.NodeType)
}

func (client *ssntpClient) nodeDisconnected(payload []byte) {
	var nodeDisconnected payloads.NodeDisconnected
	err := yaml.Unmarshal(payload, &nodeDisconnected)
	if err != nil {
		glog.Warningf("Error unmarshalling NodeDisconnected: %v", err)
		return
	}

	glog.Infof("Node %s disconnected", nodeDisconnected.Disconnected.NodeUUID)
	client.ctl.ds.DeleteNode(nodeDisconnected.Disconnected.NodeUUID)
}

func (client *ssntpClient) unassignEvent(payload []byte) {
	var event payloads.EventPublicIPUnassigned
	err := yaml.Unmarshal(payload, &event)
	if err != nil {
		glog.Warningf("Error unmarshalling EventPublicIPUnassigned: %v", err)
		return
	}

	i, err := client.ctl.ds.GetInstance(event.UnassignedIP.InstanceUUID)
	if err != nil {
		glog.Warningf("Error getting instance from datastore: %v", err)
		return
	}

	err = client.ctl.ds.UnMapExternalIP(event.UnassignedIP.PublicIP)
	if err != nil {
		glog.Warning("Error unmapping external IP: %v", err)
		return
	}

	client.ctl.qs.Release(i.TenantID, payloads.RequestedResource{Type: payloads.ExternalIP, Value: 1})

	msg := fmt.Sprintf("Unmapped %s from %s", event.UnassignedIP.PublicIP, event.UnassignedIP.PrivateIP)
	client.ctl.ds.LogEvent(i.TenantID, msg)
}

func (client *ssntpClient) assignEvent(payload []byte) {
	var event payloads.EventPublicIPAssigned
	err := yaml.Unmarshal(payload, &event)
	if err != nil {
		glog.Warningf("Error unmarshalling EventPublicIPAssigned: %v", err)
		return
	}

	i, err := client.ctl.ds.GetInstance(event.AssignedIP.InstanceUUID)
	if err != nil {
		glog.Warningf("Error getting instance from datastore: %v", err)
		return
	}

	msg := fmt.Sprintf("Mapped %s to %s", event.AssignedIP.PublicIP, event.AssignedIP.PrivateIP)
	client.ctl.ds.LogEvent(i.TenantID, msg)
}

func (client *ssntpClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	payload := frame.Payload

	glog.Info("EVENT ", event, " for ", client.name)

	glog.V(1).Info(string(payload))

	switch event {
	case ssntp.InstanceDeleted:
		client.instanceDeleted(payload)

	case ssntp.InstanceStopped:
		client.instanceStopped(payload)

	case ssntp.ConcentratorInstanceAdded:
		client.instanceAdded(payload)

	case ssntp.TraceReport:
		client.traceReport(payload)

	case ssntp.NodeConnected:
		client.nodeConnected(payload)

	case ssntp.NodeDisconnected:
		client.nodeDisconnected(payload)

	case ssntp.PublicIPAssigned:
		client.assignEvent(payload)

	case ssntp.PublicIPUnassigned:
		client.unassignEvent(payload)

	}
}

func (client *ssntpClient) startFailure(payload []byte) {
	var failure payloads.ErrorStartFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warningf("Error unmarshalling StartFailure: %v", err)
		return
	}
	if failure.Reason.IsFatal() && !failure.Restart {
		client.deleteEphemeralStorage(failure.InstanceUUID)
		err = client.releaseResources(failure.InstanceUUID)
		if err != nil {
			glog.Warningf("Error when releasing resources for start failed instance: %v", err)
		}
	}
	err = client.ctl.ds.StartFailure(failure.InstanceUUID, failure.Reason, failure.Restart)
	if err != nil {
		glog.Warningf("Error adding StartFailure to datastore: %v", err)
	}
}

func (client *ssntpClient) stopFailure(payload []byte) {
	var failure payloads.ErrorStopFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warningf("Error unmarshalling StopFailure: %v", err)
		return
	}
	client.ctl.ds.StopFailure(failure.InstanceUUID, failure.Reason)
	if err != nil {
		glog.Warningf("Error adding StopFailure to datastore: %v", err)
	}
}

func (client *ssntpClient) restartFailure(payload []byte) {
	var failure payloads.ErrorRestartFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warning("Error unmarshalling RestartFailure: %v", err)
		return
	}
	err = client.ctl.ds.RestartFailure(failure.InstanceUUID, failure.Reason)
	if err != nil {
		glog.Warningf("Error adding RestartFailure to datastore: %v", err)
	}
}

func (client *ssntpClient) attachVolumeFailure(payload []byte) {
	var failure payloads.ErrorAttachVolumeFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warningf("Error unmarshalling AttachVolumeFailure: %v", err)
		return
	}
	err = client.ctl.ds.AttachVolumeFailure(failure.InstanceUUID, failure.VolumeUUID, failure.Reason)
	if err != nil {
		glog.Warningf("Error handling AttachVolumeFailure in datastore: %v", err)
	}
}

func (client *ssntpClient) detachVolumeFailure(payload []byte) {
	var failure payloads.ErrorDetachVolumeFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warningf("Error unmarshalling DetachVolumeFailure: %v", err)
		return
	}

	err = client.ctl.ds.DetachVolumeFailure(failure.InstanceUUID, failure.VolumeUUID, failure.Reason)
	if err != nil {
		glog.Warningf("Error handling DetachVolumeFailure in datastore: %v", err)
	}
}

func (client *ssntpClient) assignError(payload []byte) {
	var failure payloads.ErrorPublicIPFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warningf("Error unmarshalling ErrorPublicIPFailure:: %v", err)
		return
	}

	err = client.ctl.ds.UnMapExternalIP(failure.PublicIP)
	if err != nil {
		glog.Warningf("Error unmapping external IP: %v", err)
	}

	client.ctl.qs.Release(failure.TenantUUID, payloads.RequestedResource{Type: payloads.ExternalIP, Value: 1})

	msg := fmt.Sprintf("Failed to map %s to %s: %s", failure.PublicIP, failure.InstanceUUID, failure.Reason.String())
	client.ctl.ds.LogEvent(failure.TenantUUID, msg)
}

func (client *ssntpClient) unassignError(payload []byte) {
	var failure payloads.ErrorPublicIPFailure
	err := yaml.Unmarshal(payload, &failure)
	if err != nil {
		glog.Warning("Error unmarshalling ErrorPublicIPFailure: %v", err)
		return
	}

	// we can't unmap the IP - all we can do is log.
	msg := fmt.Sprintf("Failed to unmap %s from %s: %s", failure.PublicIP, failure.InstanceUUID, failure.Reason.String())
	client.ctl.ds.LogEvent(failure.TenantUUID, msg)
}

func (client *ssntpClient) ErrorNotify(err ssntp.Error, frame *ssntp.Frame) {
	payload := frame.Payload

	glog.Info("ERROR (", err, ") for ", client.name)
	glog.V(1).Info(string(payload))

	switch err {
	case ssntp.StartFailure:
		client.startFailure(payload)

	case ssntp.StopFailure:
		client.stopFailure(payload)

	case ssntp.RestartFailure:
		client.restartFailure(payload)

	case ssntp.AttachVolumeFailure:
		client.attachVolumeFailure(payload)

	case ssntp.DetachVolumeFailure:
		client.detachVolumeFailure(payload)

	case ssntp.AssignPublicIPFailure:
		client.assignError(payload)

	case ssntp.UnassignPublicIPFailure:
		client.unassignError(payload)

	}
}

func newSSNTPClient(ctl *controller, config *ssntp.Config) (controllerClient, error) {
	client := &ssntpClient{name: "ciao Controller", ctl: ctl}

	err := client.ssntp.Dial(config, client)
	return client, err
}

func (client *ssntpClient) StartTracedWorkload(config string, startTime time.Time, label string) error {
	glog.V(1).Info("START TRACED config:")
	glog.V(1).Info(config)

	traceConfig := &ssntp.TraceConfig{
		PathTrace: true,
		Start:     startTime,
		Label:     []byte(label),
	}

	_, err := client.ssntp.SendTracedCommand(ssntp.START, []byte(config), traceConfig)

	return err
}

func (client *ssntpClient) StartWorkload(config string) error {
	glog.V(1).Info("START config:")
	glog.V(1).Info(config)

	_, err := client.ssntp.SendCommand(ssntp.START, []byte(config))

	return err
}

func (client *ssntpClient) deleteInstance(payload *payloads.Delete, instanceID string, nodeID string) error {
	y, err := yaml.Marshal(*payload)
	if err != nil {
		return err
	}

	glog.Info("DELETE instance_id: ", instanceID, "node_id ", nodeID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.DELETE, y)

	return err
}

func (client *ssntpClient) DeleteInstance(instanceID string, nodeID string) error {
	if nodeID == "" {
		// This instance is not running and not assigned to a node.  We
		// can just remove its details from controller's db and delete
		// any ephemeral storage.
		glog.Info("Deleting unassigned instance")
		client.removeInstance(instanceID)
		return nil
	}

	payload := payloads.Delete{
		Delete: payloads.StopCmd{
			InstanceUUID:      instanceID,
			WorkloadAgentUUID: nodeID,
		},
	}

	return client.deleteInstance(&payload, instanceID, nodeID)
}

func (client *ssntpClient) StopInstance(instanceID string, nodeID string) error {
	payload := payloads.Delete{
		Delete: payloads.StopCmd{
			InstanceUUID:      instanceID,
			WorkloadAgentUUID: nodeID,
			Stop:              true,
		},
	}

	return client.deleteInstance(&payload, instanceID, nodeID)
}

func (client *ssntpClient) RestartInstance(i *types.Instance, w *types.Workload,
	t *types.Tenant) error {

	err := client.ctl.ds.RestartInstance(i.ID)
	if err != nil {
		return errors.Wrapf(err, "Unable to update instance state before restarting")
	}

	hostname := i.ID
	if i.Name != "" {
		hostname = i.Name
	}

	metaData := userData{
		UUID:     i.ID,
		Hostname: hostname,
	}

	restartCmd := payloads.StartCmd{
		TenantUUID:          i.TenantID,
		InstanceUUID:        i.ID,
		FWType:              payloads.Firmware(w.FWType),
		VMType:              w.VMType,
		InstancePersistence: payloads.Host,
		RequestedResources:  w.Defaults,
		Networking: payloads.NetworkResources{
			VnicMAC:          i.MACAddress,
			VnicUUID:         i.VnicUUID,
			ConcentratorUUID: t.CNCIID,
			ConcentratorIP:   t.CNCIIP,
			Subnet:           i.Subnet,
			PrivateIP:        i.IPAddress,
		},
		Storage: make([]payloads.StorageResource, len(i.Attachments)),
		Restart: true,
	}

	if w.VMType == payloads.Docker {
		restartCmd.DockerImage = w.ImageName
	}

	for k := range i.Attachments {
		vol := &restartCmd.Storage[k]
		vol.ID = i.Attachments[k].BlockID
		vol.Bootable = i.Attachments[k].Boot
		vol.Ephemeral = i.Attachments[k].Ephemeral
	}

	payload := payloads.Start{
		Start: restartCmd,
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	b, err := json.Marshal(&metaData)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	_, _ = buf.WriteString("---\n")
	_, _ = buf.Write(y)
	_, _ = buf.WriteString("...\n")
	_, _ = buf.WriteString(w.Config)
	_, _ = buf.WriteString("---\n")
	_, _ = buf.Write(b)
	_, _ = buf.WriteString("\n...\n")

	glog.Info("RESTART instance: ", i.ID)
	glog.V(1).Info(buf.String())

	_, err = client.ssntp.SendCommand(ssntp.START, buf.Bytes())

	return err
}

func (client *ssntpClient) EvacuateNode(nodeID string) error {
	evacuateCmd := payloads.EvacuateCmd{
		WorkloadAgentUUID: nodeID,
	}

	payload := payloads.Evacuate{
		Evacuate: evacuateCmd,
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Info("EVACUATE node: ", nodeID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.EVACUATE, y)

	return err
}

func (client *ssntpClient) attachVolume(volID string, instanceID string, nodeID string) error {
	payload := payloads.AttachVolume{
		Attach: payloads.VolumeCmd{
			InstanceUUID:      instanceID,
			VolumeUUID:        volID,
			WorkloadAgentUUID: nodeID,
		},
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Infof("AttachVolume %s to %s\n", volID, instanceID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.AttachVolume, y)

	return err
}

func (client *ssntpClient) detachVolume(volID string, instanceID string, nodeID string) error {
	payload := payloads.DetachVolume{
		Detach: payloads.VolumeCmd{
			InstanceUUID:      instanceID,
			VolumeUUID:        volID,
			WorkloadAgentUUID: nodeID,
		},
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Infof("DetachVolume %s to %s\n", volID, instanceID)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.DetachVolume, y)

	return err
}

func (client *ssntpClient) ssntpClient() *ssntp.Client {
	return &client.ssntp
}

func (client *ssntpClient) Disconnect() {
	client.ssntp.Close()
}

func (client *ssntpClient) mapExternalIP(t types.Tenant, m types.MappedIP) error {
	payload := payloads.CommandAssignPublicIP{
		AssignIP: payloads.PublicIPCommand{
			ConcentratorUUID: t.CNCIID,
			TenantUUID:       m.TenantID,
			InstanceUUID:     m.InstanceID,
			PublicIP:         m.ExternalIP,
			PrivateIP:        m.InternalIP,
			VnicMAC:          t.CNCIMAC,
		},
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Infof("Request Map of %s to %s\n", m.ExternalIP, m.InternalIP)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.AssignPublicIP, y)
	return err
}

func (client *ssntpClient) unMapExternalIP(t types.Tenant, m types.MappedIP) error {
	payload := payloads.CommandReleasePublicIP{
		ReleaseIP: payloads.PublicIPCommand{
			ConcentratorUUID: t.CNCIID,
			TenantUUID:       m.TenantID,
			InstanceUUID:     m.InstanceID,
			PublicIP:         m.ExternalIP,
			PrivateIP:        m.InternalIP,
			VnicMAC:          t.CNCIMAC,
		},
	}

	y, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	glog.Infof("Request unmap of %s from %s\n", m.ExternalIP, m.InternalIP)
	glog.V(1).Info(string(y))

	_, err = client.ssntp.SendCommand(ssntp.ReleasePublicIP, y)
	return err
}
