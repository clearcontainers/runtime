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

package datastore

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp/uuid"
)

var dbCount = 1

func getPersistentStore() (persistentStore, error) {
	ps := &sqliteDB{}
	config := Config{
		PersistentURI:     "file:memdb" + string(dbCount) + "?mode=memory&cache=shared",
		TransientURI:      "file:memdb" + string(dbCount+1) + "?mode=memory&cache=shared",
		InitWorkloadsPath: *workloadsPath,
	}
	err := ps.init(config)
	dbCount = dbCount + 2
	return ps, err
}

func TestSQLiteDBGetWorkloadStorage(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.getWorkloadStorage("validid")
	if err != nil {
		t.Fatal(err)
	}

	db.disconnect()
}

func TestSQLiteDBGetTenantDevices(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	blockDevice := storage.BlockDevice{
		ID: uuid.Generate().String(),
	}

	data := types.BlockData{
		BlockDevice: blockDevice,
		State:       types.Available,
		TenantID:    uuid.Generate().String(),
		CreateTime:  time.Now(),
	}

	err = db.addBlockData(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure our query works.
	devices, err := db.getTenantDevices(data.TenantID)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := devices[data.ID]
	if !ok {
		t.Fatal("device not in map")
	}

	db.disconnect()
}

func TestSQLiteDBGetTenantWithStorage(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	// add a tenant.
	tenantID := uuid.Generate().String()
	mac := "validmac"

	err = db.addTenant(tenantID, mac)
	if err != nil {
		t.Fatal(err)
	}

	blockDevice := storage.BlockDevice{
		ID: uuid.Generate().String(),
	}

	data := types.BlockData{
		BlockDevice: blockDevice,
		State:       types.Available,
		TenantID:    tenantID,
		CreateTime:  time.Now(),
	}

	err = db.addBlockData(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure our query works.
	tenant, err := db.getTenant(data.TenantID)
	if err != nil {
		t.Fatal(err)
	}

	if tenant.devices == nil {
		t.Fatal("devices is nil")
	}

	d := tenant.devices[data.ID]
	if d.ID != data.ID {
		t.Fatal("device not correct")
	}

	db.disconnect()
}

func TestSQLiteDBGetAllBlockData(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	blockDevice := storage.BlockDevice{
		ID: uuid.Generate().String(),
	}

	data := types.BlockData{
		BlockDevice: blockDevice,
		State:       types.Available,
		TenantID:    uuid.Generate().String(),
		CreateTime:  time.Now(),
	}

	err = db.addBlockData(data)
	if err != nil {
		t.Fatal(err)
	}

	devices, err := db.getAllBlockData()
	if err != nil {
		t.Fatal(err)
	}

	_, ok := devices[data.ID]
	if !ok {
		t.Fatal(err)
	}

	db.disconnect()
}

func TestSQLiteDBDeleteBlockData(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	blockDevice := storage.BlockDevice{
		ID: uuid.Generate().String(),
	}

	data := types.BlockData{
		BlockDevice: blockDevice,
		State:       types.Available,
		TenantID:    uuid.Generate().String(),
		CreateTime:  time.Now(),
	}

	err = db.addBlockData(data)
	if err != nil {
		t.Fatal(err)
	}

	err = db.deleteBlockData(data.ID)
	if err != nil {
		t.Fatal(err)
	}

	devices, err := db.getAllBlockData()
	if err != nil {
		t.Fatal(err)
	}

	_, ok := devices[data.ID]
	if ok {
		t.Fatal("block devices not deleted")
	}

	db.disconnect()
}

func TestSQLiteDBGetAllStorageAttachments(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	a := types.StorageAttachment{
		ID:         uuid.Generate().String(),
		InstanceID: uuid.Generate().String(),
		BlockID:    uuid.Generate().String(),
		Ephemeral:  false,
	}

	err = db.addStorageAttachment(a)
	if err != nil {
		t.Fatal(err)
	}

	attachments, err := db.getAllStorageAttachments()
	if err != nil {
		t.Fatal(err)
	}

	if len(attachments) != 1 {
		t.Fatal(err)
	}

	alpha := attachments[a.ID]

	if alpha != a {
		t.Fatal("Attachment from DB doesn't match original attachment")
	}

	b := types.StorageAttachment{
		ID:         uuid.Generate().String(),
		InstanceID: uuid.Generate().String(),
		BlockID:    uuid.Generate().String(),
		Ephemeral:  true,
	}

	err = db.addStorageAttachment(b)
	if err != nil {
		t.Fatal(err)
	}

	attachments, err = db.getAllStorageAttachments()
	if err != nil {
		t.Fatal(err)
	}

	if len(attachments) != 2 {
		t.Fatal(err)
	}

	err = db.deleteStorageAttachment(a.ID)
	if err != nil {
		t.Fatal(err)
	}

	attachments, err = db.getAllStorageAttachments()
	if err != nil {
		t.Fatal(err)
	}

	if len(attachments) != 1 {
		t.Fatal(err)
	}

	beta := attachments[b.ID]

	if beta != b {
		t.Fatal("Attachment from DB doesn't match original attachment")
	}
	db.disconnect()
}

func TestCreatePool(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.addPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || (p.Name != "test") {
		t.Fatal("pool not stored")
	}

	db.disconnect()
}

func TestUpdatePool(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.addPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pool.Free = 2
	pool.TotalIPs = 10

	err = db.updatePool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || p.Free != 2 || p.TotalIPs != 10 {
		t.Fatal("pool not updated")
	}

	db.disconnect()
}

func TestDeletePool(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.addPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	_, ok := pools[pool.ID]
	if !ok {
		t.Fatal("pool not updated")
	}

	err = db.deletePool(pool.ID)
	if err != nil {
		t.Fatal("pool not deleted")
	}

	pools = db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	_, ok = pools[pool.ID]
	if ok {
		t.Fatal("pool not deleted")
	}

	db.disconnect()
}

func TestCreateSubnet(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.addPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	subnet := types.ExternalSubnet{
		ID:   uuid.Generate().String(),
		CIDR: "192.168.0.0/24",
	}

	pool.Subnets = append(pool.Subnets, subnet)

	err = db.updatePool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || (p.Name != "test") {
		t.Fatal("pool not stored")
	}

	subs := p.Subnets
	if len(subs) != 1 {
		t.Fatal("subnet not saved")
	}

	if subs[0].CIDR != subnet.CIDR || subs[0].ID != subnet.ID {
		t.Fatal("subnet not saved correctly")
	}

	db.disconnect()
}

func TestDeleteSubnet(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	subnet := types.ExternalSubnet{
		ID:   uuid.Generate().String(),
		CIDR: "192.168.0.0/24",
	}

	pool.Subnets = append(pool.Subnets, subnet)

	err = db.addPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pool.Subnets = []types.ExternalSubnet{}
	err = db.updatePool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || (p.Name != "test") {
		t.Fatal("pool not stored")
	}

	subs := p.Subnets
	if len(subs) != 0 {
		t.Fatal("subnet not deleted")
	}

	db.disconnect()
}

func TestCreateAddress(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	IP := types.ExternalIP{
		ID:      uuid.Generate().String(),
		Address: "192.168.0.1",
	}

	pool.IPs = append(pool.IPs, IP)

	err = db.addPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || (p.Name != "test") {
		t.Fatal("pool not stored")
	}

	addrs := p.IPs
	if len(addrs) != 1 || addrs[0].ID != IP.ID || addrs[0].Address != IP.Address {
		t.Fatal("address not stored correctly")
	}

	db.disconnect()
}

func TestDeleteAddress(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	IP := types.ExternalIP{
		ID:      uuid.Generate().String(),
		Address: "192.168.0.1",
	}

	pool.IPs = append(pool.IPs, IP)

	err = db.addPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pool.IPs = []types.ExternalIP{}

	err = db.updatePool(pool)
	if err != nil {
		t.Fatal(err)
	}

	pools := db.getAllPools()
	if pools == nil {
		t.Fatal("pool not stored")
	}

	p, ok := pools[pool.ID]
	if !ok || (p.Name != "test") {
		t.Fatal("pool not stored")
	}

	addrs := p.IPs
	if len(addrs) != 0 {
		t.Fatal("address not deleted")
	}

	db.disconnect()
}

func TestCreateMappedIP(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	i := types.Instance{
		ID:         uuid.Generate().String(),
		TenantID:   uuid.Generate().String(),
		WorkloadID: uuid.Generate().String(),
		IPAddress:  "172.16.0.2",
	}

	err = db.addInstance(&i)
	if err != nil {
		t.Fatal("unable to store instance")
	}

	instances, err := db.getInstances()
	if err != nil || len(instances) != 1 {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.addPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	m := types.MappedIP{
		ID:         uuid.Generate().String(),
		ExternalIP: "192.168.0.1",
		InternalIP: i.IPAddress,
		InstanceID: i.ID,
		TenantID:   i.TenantID,
		PoolID:     pool.ID,
		PoolName:   pool.Name,
	}

	err = db.addMappedIP(m)
	if err != nil {
		t.Fatal(err)
	}

	IPs := db.getMappedIPs()
	if len(IPs) != 1 {
		t.Fatal("could not get mapped IP")
	}

	if reflect.DeepEqual(IPs[m.ExternalIP], m) == false {
		t.Fatalf("expected %v, got %v\n", m, IPs[m.ExternalIP])
	}
}

func TestDeleteMappedIP(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	i := types.Instance{
		ID:         uuid.Generate().String(),
		TenantID:   uuid.Generate().String(),
		WorkloadID: uuid.Generate().String(),
		IPAddress:  "172.16.0.2",
	}

	err = db.addInstance(&i)
	if err != nil {
		t.Fatal("unable to store instance")
	}

	instances, err := db.getInstances()
	if err != nil || len(instances) != 1 {
		t.Fatal(err)
	}

	pool := types.Pool{
		ID:   uuid.Generate().String(),
		Name: "test",
	}

	err = db.addPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	m := types.MappedIP{
		ID:         uuid.Generate().String(),
		ExternalIP: "192.168.0.1",
		InternalIP: i.IPAddress,
		InstanceID: i.ID,
		TenantID:   i.TenantID,
		PoolID:     pool.ID,
		PoolName:   pool.Name,
	}

	err = db.addMappedIP(m)
	if err != nil {
		t.Fatal(err)
	}

	IPs := db.getMappedIPs()
	if len(IPs) != 1 {
		t.Fatal("could not get mapped IP")
	}

	if reflect.DeepEqual(IPs[m.ExternalIP], m) == false {
		t.Fatalf("expected %v, got %v\n", m, IPs[m.ExternalIP])
	}

	err = db.deleteMappedIP(m.ID)
	if err != nil {
		t.Fatal(err)
	}

	IPs = db.getMappedIPs()
	if len(IPs) != 0 {
		t.Fatal("IP not deleted")
	}
}

func createTestTenant(db persistentStore, t *testing.T) *tenant {
	tid := uuid.Generate().String()
	thw, err := newHardwareAddr()
	if err != nil {
		t.Fatal(err)
	}
	err = db.addTenant(tid, thw.String())
	if err != nil {
		t.Fatal(err)
	}

	tn, err := db.getTenant(tid)
	if err != nil {
		t.Fatal(err)
	}
	if tn == nil {
		t.Fatal("Expected added tenant")
	}

	if tn.CNCIMAC != thw.String() {
		t.Fatal("Expected added tenant CNCI MACs to be equal")
	}
	return tn
}

func TestSQLiteDBTestTenants(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	tns, err := db.getTenants()
	if err != nil {
		t.Fatal(err)
	}

	if len(tns) != 0 {
		t.Fatal("No tenants expected")
	}

	_ = createTestTenant(db, t)
	_ = createTestTenant(db, t)

	tns, err = db.getTenants()
	if err != nil {
		t.Fatal(err)
	}

	if len(tns) != 2 {
		t.Fatal("2 tenants expected")
	}

	for _, tn := range tns {
		tn2, err := db.getTenant(tn.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tn, tn2) {
			t.Fatal("Expected tenant equality")
		}
	}
}

func TestSQLiteDBTestUpdateTenant(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	tn := createTestTenant(db, t)
	tn.CNCIIP = "127.0.0.2"

	err = db.updateTenant(tn)
	if err != nil {
		t.Fatal(err)
	}

	if tn.CNCIIP != "127.0.0.2" {
		t.Fatal("Tenant not updated")
	}
}

func TestSQLiteDBGetBatchFrameStatistics(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	frames := createTestFrameTraces("batch_frame_test")
	for _, frame := range frames {
		err := db.addFrameStat(frame)
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = db.getBatchFrameStatistics("batch_frame_test")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteDBGetBatchFrameSummary(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	frames := createTestFrameTraces("batch_summary_test")
	for _, frame := range frames {
		err := db.addFrameStat(frame)
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = db.getBatchFrameSummary()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteDBEventLog(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	log, err := db.getEventLog()
	if err != nil {
		t.Fatal(err)
	}

	if len(log) != 0 {
		t.Fatal("Expected no log messages")
	}

	tn := createTestTenant(db, t)

	err = db.logEvent(tn.ID, string(userError), "test message 1")
	if err != nil {
		t.Fatal(err)
	}

	log, err = db.getEventLog()
	if err != nil {
		t.Fatal(err)
	}
	if len(log) != 1 {
		t.Fatal("Expected 1 log message")
	}

	err = db.logEvent(tn.ID, string(userError), "test message 2")
	if err != nil {
		t.Fatal(err)
	}

	log, err = db.getEventLog()
	if err != nil {
		t.Fatal(err)
	}
	if len(log) != 2 {
		t.Fatal("Expected 2 log message")
	}

	err = db.clearLog()
	if err != nil {
		t.Fatal(err)
	}

	log, err = db.getEventLog()
	if err != nil {
		t.Fatal(err)
	}
	if len(log) != 0 {
		t.Fatal("Expected no log messages")
	}
}

func TestSQLiteDBInstanceStats(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	var stats []payloads.InstanceStat

	for i := 0; i < 3; i++ {
		stat := payloads.InstanceStat{
			InstanceUUID:  uuid.Generate().String(),
			State:         payloads.ComputeStatusRunning,
			SSHIP:         "192.168.0.1",
			SSHPort:       34567,
			MemoryUsageMB: 0,
			DiskUsageMB:   0,
			CPUUsage:      0,
		}
		stats = append(stats, stat)
	}

	nodeID := uuid.Generate().String()

	err = db.addInstanceStats(stats, nodeID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.getNodeSummary()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteDBUpdateDeleteWorkload(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	tn := createTestTenant(db, t)

	testConfig := `
---
#cloud-config
users:
  - name: demouser
    gecos: CIAO Demo User
    lock-passwd: false
    passwd: $6$rounds=4096$w9I3hR4g/hu$AnYjaC2DfznbPSG3vxsgtgAS4mJwWBkcR74Y/KHNB5OsfAlA4gpU5j6CHWMOkkt9j.9d7OYJXJ4icXHzKXTAO.
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
    - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDerQfD+qkb0V0XdQs8SBWqy4sQmqYFP96n/kI4Cq162w4UE8pTxy0ozAPldOvBJjljMvgaNKSAddknkhGcrNUvvJsUcZFm2qkafi32WyBdGFvIc45A+8O7vsxPXgHEsS9E3ylEALXAC3D0eX7pPtRiAbasLlY+VcACRqr3bPDSZTfpCmIkV2334uZD9iwOvTVeR+FjGDqsfju4DyzoAIqpPasE0+wk4Vbog7osP+qvn1gj5kQyusmr62+t0wx+bs2dF5QemksnFOswUrv9PGLhZgSMmDQrRYuvEfIAC7IdN/hfjTn0OokzljBiuWQ4WIIba/7xTYLVujJV65qH3heaSMxJJD7eH9QZs9RdbbdTXMFuJFsHV2OF6wZRp18tTNZZJMqiHZZSndC5WP1WrUo3Au/9a+ighSaOiVddHsPG07C/TOEnr3IrwU7c9yIHeeRFHmcQs9K0+n9XtrmrQxDQ9/mLkfje80Ko25VJ/QpAQPzCKh2KfQ4RD+/PxBUScx/lHIHOIhTSCh57ic629zWgk0coSQDi4MKSa5guDr3cuDvt4RihGviDM6V68ewsl0gh6Z9c0Hw7hU0vky4oxak5AiySiPz0FtsOnAzIL0UON+yMuKzrJgLjTKodwLQ0wlBXu43cD+P8VXwQYeqNSzfrhBnHqsrMf4lTLtc7kDDTcw== ciao@ciao
...
	`
	cpus := payloads.RequestedResource{
		Type:      payloads.VCPUs,
		Value:     2,
		Mandatory: false,
	}

	mem := payloads.RequestedResource{
		Type:      payloads.MemMB,
		Value:     512,
		Mandatory: false,
	}

	storage := types.StorageResource{
		ID:        "",
		Ephemeral: false,
		Size:      20,
	}

	wl := types.Workload{
		ID:          uuid.Generate().String(),
		TenantID:    tn.ID,
		Description: "testWorkload",
		FWType:      string(payloads.EFI),
		VMType:      payloads.QEMU,
		ImageID:     uuid.Generate().String(),
		ImageName:   "",
		Config:      testConfig,
		Defaults:    []payloads.RequestedResource{mem, cpus},
		Storage:     []types.StorageResource{storage},
	}

	// file will be added, so we will want to remove it.
	filename := fmt.Sprintf("%s/%s_config.yaml", *workloadsPath, wl.ID)
	defer os.Remove(filename)

	err = db.updateWorkload(wl)
	if err != nil {
		t.Fatal(err)
	}

	tenant, err := db.getTenant(tn.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(tenant.workloads) != 1 {
		t.Fatal("Expected a workload associated with tenant")
	}

	wl2 := tenant.workloads[0]

	if !reflect.DeepEqual(wl, wl2) {
		fmt.Fprintf(os.Stderr, "got %v\n", wl2)
		fmt.Fprintf(os.Stderr, "expected %v\n", wl)
		t.Fatal("Expected workload equality")
	}

	// now try to delete the workload
	err = db.deleteWorkload(wl.ID)
	if err != nil {
		t.Fatal(err)
	}

	tenant, err = db.getTenant(tn.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(tenant.workloads) != 0 {
		t.Fatal("Expected no workloads associated with tenant")
	}

	db.disconnect()
}

func findQuota(qds []types.QuotaDetails, name string, value int) bool {
	for _, qd := range qds {
		if qd.Name == name && qd.Value == value {
			return true
		}
	}

	return false
}

func TestSQLiteDBAddQuotas(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	qds, err := db.getQuotas("test-tenand-id")
	if err != nil {
		t.Fatal(err)
	}

	if len(qds) != 0 {
		t.Fatalf("Expected zero quota entries: got %d", len(qds))
	}

	err = db.updateQuotas("test-tenant-id", []types.QuotaDetails{{Name: "test-quota-name", Value: 10}})
	if err != nil {
		t.Fatal(err)
	}

	qds, err = db.getQuotas("test-tenant-id")
	if err != nil {
		t.Fatal(err)
	}

	if !findQuota(qds, "test-quota-name", 10) {
		t.Fatal("Added quota not found")
	}

	err = db.updateQuotas("test-tenant-id", []types.QuotaDetails{{Name: "test-quota-name-2", Value: 20}})
	if err != nil {
		t.Fatal(err)
	}

	qds, err = db.getQuotas("test-tenant-id")
	if err != nil {
		t.Fatal(err)
	}

	if len(qds) != 2 {
		t.Fatalf("Expected 2 quotas: got %d", len(qds))
	}

	if !findQuota(qds, "test-quota-name-2", 20) {
		t.Fatal("Added quota not found")
	}
}

func TestSQLiteDBUpdateQuotas(t *testing.T) {
	db, err := getPersistentStore()
	if err != nil {
		t.Fatal(err)
	}

	qds, err := db.getQuotas("test-tenand-id")
	if err != nil {
		t.Fatal(err)
	}

	if len(qds) != 0 {
		t.Fatalf("Expected zero quota entries: got %d", len(qds))
	}

	err = db.updateQuotas("test-tenant-id", []types.QuotaDetails{{Name: "test-quota-name", Value: 10}})
	if err != nil {
		t.Fatal(err)
	}

	qds, err = db.getQuotas("test-tenant-id")
	if err != nil {
		t.Fatal(err)
	}

	if !findQuota(qds, "test-quota-name", 10) {
		t.Fatal("Added quota not found")
	}

	err = db.updateQuotas("test-tenant-id", []types.QuotaDetails{{Name: "test-quota-name", Value: 20}})
	if err != nil {
		t.Fatal(err)
	}

	qds, err = db.getQuotas("test-tenant-id")
	if err != nil {
		t.Fatal(err)
	}

	if len(qds) != 1 {
		t.Fatalf("Expected 1 quotas: got %d", len(qds))
	}

	if !findQuota(qds, "test-quota-name", 20) {
		t.Fatal("Added quota not found")
	}
}
