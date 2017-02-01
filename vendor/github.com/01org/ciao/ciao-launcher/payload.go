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
	"bufio"
	"bytes"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/01org/ciao/networking/libsnnet"
	"github.com/01org/ciao/payloads"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

type payloadError struct {
	err  error
	code string
}

type extractedDoc struct {
	doc       []string
	realStart int
	realEnd   int
}

var indentedRegexp *regexp.Regexp
var startRegexp *regexp.Regexp
var uuidRegexp *regexp.Regexp

func init() {
	indentedRegexp = regexp.MustCompile("\\s+.*")
	startRegexp = regexp.MustCompile("^start\\s*:\\s*$")
	uuidRegexp = regexp.MustCompile("^[0-9a-fA-F]+(-[0-9a-fA-F]+)*$")
}

func printCloudinit(data *payloads.Start) {
	start := &data.Start
	glog.Info("cloud-init file content")
	glog.Info("-----------------------")
	glog.Infof("Instance UUID:        %v", start.InstanceUUID)
	glog.Infof("Disk image UUID:      %v", start.ImageUUID)
	glog.Infof("FW Type:              %v", start.FWType)
	glog.Infof("VM Type:              %v", start.VMType)
	glog.Infof("TenantUUID:          %v", start.TenantUUID)
	net := &start.Networking
	glog.Infof("VnicMAC:              %v", net.VnicMAC)
	glog.Infof("VnicIP:               %v", net.PrivateIP)
	glog.Infof("ConcIP:               %v", net.ConcentratorIP)
	glog.Infof("SubnetIP:             %v", net.Subnet)
	glog.Infof("ConcUUID:             %v", net.ConcentratorUUID)
	glog.Infof("VnicUUID:             %v", net.VnicUUID)

	glog.Info("Requested resources:")
	for i := range start.RequestedResources {
		glog.Infof("%8s:     %v", start.RequestedResources[i].Type,
			start.RequestedResources[i].Value)
	}

	for _, storage := range start.Storage {
		if storage.ID != "" {
			glog.Info("Volumes:")
			glog.Infof("  %s Bootable=%t", storage.ID, storage.Bootable)
		}
	}
}

func computeSSHPort(networkNode bool, vnicIP string) int {
	if networkNode || vnicIP == "" {
		return 0
	}

	ip := net.ParseIP(vnicIP)
	if ip == nil {
		return 0
	}

	ip = ip.To4()
	if ip == nil {
		return 0
	}

	port, err := libsnnet.DebugSSHPortForIP(ip)
	if err != nil {
		return 0
	}

	return port
}

func parseVMTtype(start *payloads.StartCmd) (container bool, image string, err error) {
	vmType := start.VMType
	if vmType != "" && vmType != payloads.QEMU && vmType != payloads.Docker {
		err = fmt.Errorf("Invalid vmtype received: %s", vmType)
		return
	}

	container = vmType == payloads.Docker
	if container {
		image = start.DockerImage
	} else {
		image = start.ImageUUID
	}

	return
}

func parseStartPayload(data []byte) (*vmConfig, *payloadError) {
	var clouddata payloads.Start

	err := yaml.Unmarshal(data, &clouddata)
	if err != nil {
		return nil, &payloadError{err, payloads.InvalidPayload}
	}
	printCloudinit(&clouddata)

	start := &clouddata.Start

	instance := strings.TrimSpace(start.InstanceUUID)
	if !uuidRegexp.MatchString(instance) {
		err = fmt.Errorf("Invalid instance id received: %s", instance)
		return nil, &payloadError{err, payloads.InvalidData}
	}

	fwType := start.FWType
	if fwType != "" && fwType != payloads.Legacy && fwType != payloads.EFI {
		err = fmt.Errorf("Invalid fwtype received: %s", fwType)
		return nil, &payloadError{err, payloads.InvalidData}
	}
	legacy := fwType == payloads.Legacy

	var cpus, mem int
	var networkNode bool
	container, image, err := parseVMTtype(start)
	if err != nil {
		return nil, &payloadError{err, payloads.InvalidData}
	}

	for i := range start.RequestedResources {
		switch start.RequestedResources[i].Type {
		case payloads.VCPUs:
			cpus = start.RequestedResources[i].Value
		case payloads.MemMB:
			mem = start.RequestedResources[i].Value
		case payloads.NetworkNode:
			networkNode = start.RequestedResources[i].Value != 0
		}
	}

	net := &start.Networking
	vnicIP := strings.TrimSpace(net.PrivateIP)
	sshPort := computeSSHPort(networkNode, vnicIP)
	var volumes []volumeConfig
	for _, storage := range start.Storage {
		if storage.ID != "" {
			volumes = append(volumes, volumeConfig{
				UUID:     storage.ID,
				Bootable: storage.Bootable,
			})
		} else {
			/* See github issue #972:
			   A storage.ID == "" implies an auto-created-by-launcher
			   local disk.  This is not yet supported. */
			return nil, &payloadError{err, payloads.InvalidData}
		}
	}

	return &vmConfig{Cpus: cpus,
		Mem:         mem,
		Instance:    instance,
		Image:       image,
		Legacy:      legacy,
		Container:   container,
		NetworkNode: networkNode,
		VnicMAC:     strings.TrimSpace(net.VnicMAC),
		VnicIP:      vnicIP,
		ConcIP:      strings.TrimSpace(net.ConcentratorIP),
		SubnetIP:    strings.TrimSpace(net.Subnet),
		TenantUUID:  strings.TrimSpace(start.TenantUUID),
		ConcUUID:    strings.TrimSpace(net.ConcentratorUUID),
		VnicUUID:    strings.TrimSpace(net.VnicUUID),
		SSHPort:     sshPort,
		Volumes:     volumes,
	}, nil
}

func generateStartError(instance string, startErr *startError) (out []byte, err error) {
	sf := &payloads.ErrorStartFailure{
		InstanceUUID: instance,
		Reason:       startErr.code,
	}
	return yaml.Marshal(sf)
}

func generateStopError(instance string, stopErr *stopError) (out []byte, err error) {
	sf := &payloads.ErrorStopFailure{
		InstanceUUID: instance,
		Reason:       stopErr.code,
	}
	return yaml.Marshal(sf)
}

func generateRestartError(instance string, restartErr *restartError) (out []byte, err error) {
	rf := &payloads.ErrorRestartFailure{
		InstanceUUID: instance,
		Reason:       restartErr.code,
	}
	return yaml.Marshal(rf)
}

func generateDeleteError(instance string, deleteErr *deleteError) (out []byte, err error) {
	df := &payloads.ErrorDeleteFailure{
		InstanceUUID: instance,
		Reason:       deleteErr.code,
	}
	return yaml.Marshal(df)
}

func generateAttachVolumeError(instance, volume string, ave *attachVolumeError) (out []byte, err error) {
	avf := &payloads.ErrorAttachVolumeFailure{
		InstanceUUID: instance,
		VolumeUUID:   volume,
		Reason:       ave.code,
	}
	return yaml.Marshal(avf)
}

func generateDetachVolumeError(instance, volume string, dve *detachVolumeError) (out []byte, err error) {
	dvf := &payloads.ErrorDetachVolumeFailure{
		InstanceUUID: instance,
		VolumeUUID:   volume,
		Reason:       dve.code,
	}
	return yaml.Marshal(dvf)
}

func generateNetEventPayload(ssntpEvent *libsnnet.SsntpEventInfo, agentUUID string) ([]byte, error) {
	var event interface{}
	var eventData *payloads.TenantAddedEvent

	switch ssntpEvent.Event {
	case libsnnet.SsntpTunAdd:
		add := &payloads.EventTenantAdded{}
		event = add
		eventData = &add.TenantAdded
	case libsnnet.SsntpTunDel:
		del := &payloads.EventTenantRemoved{}
		event = del
		eventData = &del.TenantRemoved
	default:
		return nil, fmt.Errorf("Unsupported ssntpEventInfo type: %d",
			ssntpEvent.Event)
	}

	eventData.AgentUUID = agentUUID
	eventData.AgentIP = ssntpEvent.CnIP
	eventData.TenantUUID = ssntpEvent.TenantID
	eventData.TenantSubnet = ssntpEvent.SubnetID
	eventData.ConcentratorUUID = ssntpEvent.ConcID
	eventData.ConcentratorIP = ssntpEvent.CnciIP
	eventData.SubnetKey = ssntpEvent.SubnetKey

	return yaml.Marshal(event)
}

func parseRestartPayload(data []byte) (string, *payloadError) {
	var clouddata payloads.Restart

	err := yaml.Unmarshal(data, &clouddata)
	if err != nil {
		return "", &payloadError{err, payloads.RestartInvalidPayload}
	}

	instance := strings.TrimSpace(clouddata.Restart.InstanceUUID)
	if !uuidRegexp.MatchString(instance) {
		err = fmt.Errorf("Invalid instance id received: %s", instance)
		return "", &payloadError{err, payloads.RestartInvalidData}
	}
	return instance, nil
}

func parseDeletePayload(data []byte) (string, *payloadError) {
	var clouddata payloads.Delete

	err := yaml.Unmarshal(data, &clouddata)
	if err != nil {
		return "", &payloadError{err, payloads.DeleteInvalidPayload}
	}

	instance := strings.TrimSpace(clouddata.Delete.InstanceUUID)
	if !uuidRegexp.MatchString(instance) {
		err = fmt.Errorf("Invalid instance id received: %s", instance)
		return "", &payloadError{err, payloads.DeleteInvalidData}
	}
	return instance, nil
}

func parseStopPayload(data []byte) (string, *payloadError) {
	var clouddata payloads.Stop

	err := yaml.Unmarshal(data, &clouddata)
	if err != nil {
		glog.Errorf("YAML error: %v", err)
		return "", &payloadError{err, payloads.StopInvalidPayload}
	}

	instance := strings.TrimSpace(clouddata.Stop.InstanceUUID)
	if !uuidRegexp.MatchString(instance) {
		err = fmt.Errorf("Invalid instance id received: %s", instance)
		return "", &payloadError{err, payloads.StopInvalidData}
	}
	return instance, nil
}

func extractVolumeInfo(cmd *payloads.VolumeCmd, errString string) (string, string, *payloadError) {
	instance := strings.TrimSpace(cmd.InstanceUUID)
	if !uuidRegexp.MatchString(instance) {
		err := fmt.Errorf("Invalid instance id received: %s", instance)
		return "", "", &payloadError{err, errString}
	}

	volume := strings.TrimSpace(cmd.VolumeUUID)
	if !uuidRegexp.MatchString(volume) {
		err := fmt.Errorf("Invalid volume id received: %s", volume)
		return "", "", &payloadError{err, errString}
	}
	return instance, volume, nil
}

func parseAttachVolumePayload(data []byte) (string, string, *payloadError) {
	var clouddata payloads.AttachVolume

	err := yaml.Unmarshal(data, &clouddata)
	if err != nil {
		glog.Errorf("YAML error: %v", err)
		return "", "", &payloadError{err, payloads.AttachVolumeInvalidPayload}
	}

	return extractVolumeInfo(&clouddata.Attach, payloads.AttachVolumeInvalidData)
}

func parseDetachVolumePayload(data []byte) (string, string, *payloadError) {
	var clouddata payloads.DetachVolume

	err := yaml.Unmarshal(data, &clouddata)
	if err != nil {
		glog.Errorf("YAML error: %v", err)
		return "", "", &payloadError{err, payloads.DetachVolumeInvalidPayload}
	}

	return extractVolumeInfo(&clouddata.Detach, payloads.DetachVolumeInvalidData)
}

func linesToBytes(doc []string, buf *bytes.Buffer) {
	for _, line := range doc {
		_, _ = buf.WriteString(line)
		_, _ = buf.WriteString("\n")
	}
}

func extractDocument(doc *extractedDoc, buf *bytes.Buffer) {
	linesToBytes(doc.doc[doc.realStart:doc.realEnd], buf)
}

func extractStartYaml(lines []string, start int, s, ci *bytes.Buffer) {
	cnStart := 0

	docStartFound := false
	for ; cnStart < start; cnStart++ {
		line := lines[cnStart]
		if strings.HasPrefix(line, "---") {
			docStartFound = true
			cnStart++
			break
		}
	}

	if !docStartFound {
		cnStart = 0
	}

	linesToBytes(lines[cnStart:start], ci)

	i := start
	if i < len(lines) {
		_, _ = s.WriteString(lines[i])
		_, _ = s.WriteString("\n")
		i++
	}
	for ; i < len(lines) && (indentedRegexp.MatchString(lines[i]) || lines[i] == ""); i++ {
		_, _ = s.WriteString(lines[i])
		_, _ = s.WriteString("\n")
	}

	if i < len(lines) && !strings.HasPrefix(lines[i], "...") {
		linesToBytes(lines[i:], ci)
	}
}

func findDocument(lines []string) (doc *extractedDoc, endOfNextDoc int) {
	var realStart int
	var realEnd int
	docStartFound := false
	docEndFound := false

	start := len(lines) - 1
	line := lines[start]
	if strings.HasPrefix(line, "...") {
		docEndFound = true
		realEnd = start
		start--
	}

	for ; start >= 0; start-- {
		line := lines[start]
		if strings.HasPrefix(line, "---") {
			docStartFound = true
			break
		}
		if strings.HasPrefix(line, "...") {
			start++
			break
		}
	}

	if docStartFound {
		realStart = start + 1
		for start = start - 1; start >= 0; start-- {
			line := lines[start]
			if !strings.HasPrefix(line, "%") {
				break
			}
		}
		start++
	} else {
		if start < 0 {
			start = 0
		}
		realStart = start
	}

	if !docEndFound {
		realEnd = len(lines)
	}

	realStart -= start
	realEnd -= start

	return &extractedDoc{lines[start:], realStart, realEnd}, start
}

func splitYaml(data []byte) ([]byte, []byte, []byte) {

	var s bytes.Buffer
	var ci bytes.Buffer
	var md bytes.Buffer

	foundStart := -1
	lines := make([]string, 0, 256)
	docs := make([]*extractedDoc, 0, 3)

	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if foundStart == -1 && startRegexp.MatchString(line) {
			foundStart = len(lines)
		}
		lines = append(lines, line)
	}

	endOfNextDoc := len(lines)

	for endOfNextDoc > 0 {
		var doc *extractedDoc
		doc, endOfNextDoc = findDocument(lines[:endOfNextDoc])
		docs = append([]*extractedDoc{doc}, docs...)
	}

	if len(docs) == 1 {
		if foundStart != -1 {
			extractStartYaml(docs[0].doc, foundStart, &s, &ci)
		} else {
			extractDocument(docs[0], &ci)
		}
	} else if len(docs) == 2 {
		if foundStart != -1 {
			if foundStart < len(docs[0].doc) {
				extractStartYaml(docs[0].doc, foundStart, &s, &ci)
				extractDocument(docs[1], &md)
			} else {
				extractStartYaml(docs[1].doc, foundStart-len(docs[0].doc), &s, &ci)
				extractDocument(docs[0], &md)
			}
		} else {
			extractDocument(docs[0], &ci)
			extractDocument(docs[1], &md)
		}
	} else if foundStart != -1 && foundStart < len(docs[0].doc)+len(docs[1].doc)+len(docs[2].doc) {
		notStart := make([]*extractedDoc, 0, 2)
		sum := 0
		for i := 0; i < 3; i++ {
			newSum := sum + len(docs[i].doc)
			if foundStart >= sum && foundStart < newSum {
				extractDocument(docs[i], &s)
			} else {
				notStart = append(notStart, docs[i])
			}
			sum = newSum
		}
		extractDocument(notStart[0], &ci)
		extractDocument(notStart[1], &md)
	} else {
		glog.Warning("Unable to split payload into documents")
	}

	return s.Bytes(), ci.Bytes(), md.Bytes()
}
