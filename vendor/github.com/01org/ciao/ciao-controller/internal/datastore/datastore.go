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

// Package datastore retrieves stores data for the ciao controller.
// This package caches most data in memory, and uses a sql
// database as persistent storage.
package datastore

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp/uuid"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// custom errors
var (
	ErrNoTenant            = errors.New("Tenant not found")
	ErrNoBlockData         = errors.New("Block Device not found")
	ErrNoStorageAttachment = errors.New("No Volume Attached")
)

// Config contains configuration information for the datastore.
type Config struct {
	DBBackend         persistentStore
	PersistentURI     string
	TransientURI      string
	InitWorkloadsPath string
}

type userEventType string

const (
	userInfo  userEventType = "info"
	userWarn  userEventType = "warn"
	userError userEventType = "error"
)

type tenant struct {
	types.Tenant
	network   map[int]map[int]bool
	subnets   []int
	instances map[string]*types.Instance
	devices   map[string]types.BlockData
	workloads []types.Workload
}

type node struct {
	types.Node
	instances map[string]*types.Instance
}

type attachment struct {
	instanceID string
	volumeID   string
}

type persistentStore interface {
	init(config Config) error
	disconnect()

	// interfaces related to logging
	logEvent(tenantID string, eventType string, message string) error
	clearLog() error
	getEventLog() (logEntries []*types.LogEntry, err error)

	// interfaces related to workloads
	updateWorkload(wl types.Workload) error

	// interfaces related to tenants
	addLimit(tenantID string, resourceID int, limit int) (err error)
	addTenant(id string, MAC string) (err error)
	getTenant(id string) (t *tenant, err error)
	getTenants() ([]*tenant, error)
	updateTenant(t *tenant) (err error)
	releaseTenantIP(tenantID string, subnetInt int, rest int) (err error)
	claimTenantIP(tenantID string, subnetInt int, rest int) (err error)

	// interfaces related to instances
	getInstances() (instances []*types.Instance, err error)
	addInstance(instance *types.Instance) (err error)
	deleteInstance(instanceID string) (err error)

	// interfaces related to statistics
	addNodeStat(stat payloads.Stat) (err error)
	getNodeSummary() (Summary []*types.NodeSummary, err error)
	addInstanceStats(stats []payloads.InstanceStat, nodeID string) (err error)
	addFrameStat(stat payloads.FrameTrace) (err error)
	getBatchFrameSummary() (stats []types.BatchFrameSummary, err error)
	getBatchFrameStatistics(label string) (stats []types.BatchFrameStat, err error)

	// storage interfaces
	getWorkloadStorage(ID string) ([]types.StorageResource, error)
	getAllBlockData() (map[string]types.BlockData, error)
	addBlockData(data types.BlockData) error
	updateBlockData(data types.BlockData) error
	deleteBlockData(string) error
	getTenantDevices(tenantID string) (map[string]types.BlockData, error)
	addStorageAttachment(a types.StorageAttachment) error
	getAllStorageAttachments() (map[string]types.StorageAttachment, error)
	deleteStorageAttachment(ID string) error

	// external IP interfaces
	addPool(pool types.Pool) error
	updatePool(pool types.Pool) error
	getAllPools() map[string]types.Pool
	deletePool(ID string) error

	addMappedIP(m types.MappedIP) error
	deleteMappedIP(ID string) error
	getMappedIPs() map[string]types.MappedIP

	// quotas
	updateQuotas(tenantID string, qds []types.QuotaDetails) error
	getQuotas(tenantID string) ([]types.QuotaDetails, error)
}

// Datastore provides context for the datastore package.
type Datastore struct {
	db persistentStore

	cnciAddedChans map[string]chan bool
	cnciAddedLock  *sync.Mutex

	nodeLastStat     map[string]types.CiaoComputeNode
	nodeLastStatLock *sync.RWMutex

	instanceLastStat     map[string]types.CiaoServerStats
	instanceLastStatLock *sync.RWMutex

	tenants     map[string]*tenant
	tenantsLock *sync.RWMutex

	cnciWorkload types.Workload

	nodes     map[string]*node
	nodesLock *sync.RWMutex

	instances     map[string]*types.Instance
	instancesLock *sync.RWMutex

	tenantUsage     map[string][]types.CiaoUsage
	tenantUsageLock *sync.RWMutex

	blockDevices map[string]types.BlockData
	bdLock       *sync.RWMutex

	attachments     map[string]types.StorageAttachment
	instanceVolumes map[attachment]string
	attachLock      *sync.RWMutex
	// maybe add a map[instanceid][]types.StorageAttachment
	// to make retrieval of volumes faster.

	pools           map[string]types.Pool
	externalSubnets map[string]bool
	externalIPs     map[string]bool
	mappedIPs       map[string]types.MappedIP
	poolsLock       *sync.RWMutex
}

func (ds *Datastore) initExternalIPs() {
	ds.poolsLock = &sync.RWMutex{}
	ds.externalSubnets = make(map[string]bool)
	ds.externalIPs = make(map[string]bool)

	ds.pools = ds.db.getAllPools()

	for _, pool := range ds.pools {
		for _, subnet := range pool.Subnets {
			ds.externalSubnets[subnet.CIDR] = true
		}

		for _, IP := range pool.IPs {
			ds.externalIPs[IP.Address] = true
		}
	}

	ds.mappedIPs = ds.db.getMappedIPs()
}

// Init initializes the private data for the Datastore object.
// The sql tables are populated with initial data from csv
// files if this is the first time the database has been
// created.  The datastore caches are also filled.
func (ds *Datastore) Init(config Config) error {
	ps := config.DBBackend

	if ps == nil {
		ps = &sqliteDB{}
	}

	err := ps.init(config)
	if err != nil {
		return errors.Wrap(err, "error initialising persistent store")
	}

	ds.db = ps

	ds.cnciAddedChans = make(map[string]chan bool)
	ds.cnciAddedLock = &sync.Mutex{}

	ds.nodeLastStat = make(map[string]types.CiaoComputeNode)
	ds.nodeLastStatLock = &sync.RWMutex{}

	ds.instanceLastStat = make(map[string]types.CiaoServerStats)
	ds.instanceLastStatLock = &sync.RWMutex{}

	// warning, do not use the tenant cache to get
	// networking information right now.  that is not
	// updated, just the resources
	ds.tenants = make(map[string]*tenant)
	ds.tenantsLock = &sync.RWMutex{}

	// cache all our instances prior to getting tenants
	ds.instancesLock = &sync.RWMutex{}
	ds.instances = make(map[string]*types.Instance)

	instances, err := ds.db.getInstances()
	if err != nil {
		return errors.Wrap(err, "error getting instances from database")
	}

	for i := range instances {
		ds.instances[instances[i].ID] = instances[i]
	}

	// cache our current tenants into a map that we can
	// quickly index
	tenants, err := ds.getTenants()
	if err != nil {
		return errors.Wrap(err, "error getting tenants from database")
	}
	for i := range tenants {
		ds.tenants[tenants[i].ID] = tenants[i]
	}

	ds.nodesLock = &sync.RWMutex{}
	ds.nodes = make(map[string]*node)

	for key, i := range ds.instances {
		_, ok := ds.nodes[i.NodeID]
		if !ok {
			newNode := types.Node{
				ID: i.NodeID,
			}
			n := &node{
				Node:      newNode,
				instances: make(map[string]*types.Instance),
			}
			ds.nodes[i.NodeID] = n
		}
		ds.nodes[i.NodeID].instances[key] = i
	}

	ds.tenantUsage = make(map[string][]types.CiaoUsage)
	ds.tenantUsageLock = &sync.RWMutex{}

	ds.blockDevices, err = ds.db.getAllBlockData()
	if err != nil {
		return errors.Wrap(err, "error getting block devices from database")
	}

	ds.bdLock = &sync.RWMutex{}

	ds.attachments, err = ds.db.getAllStorageAttachments()
	if err != nil {
		return errors.Wrap(err, "error getting storage attachments from database")
	}

	ds.instanceVolumes = make(map[attachment]string)

	for key, value := range ds.attachments {
		link := attachment{
			instanceID: value.InstanceID,
			volumeID:   value.BlockID,
		}

		ds.instanceVolumes[link] = key
	}

	ds.attachLock = &sync.RWMutex{}

	ds.initExternalIPs()

	return nil
}

// Exit will disconnect the backing database.
func (ds *Datastore) Exit() {
	ds.db.disconnect()
}

// AddTenantChan allows a caller to pass in a channel for CNCI Launch status.
// When a CNCI has been added to the datastore and a channel exists,
// success will be indicated on the channel.  If a CNCI failure occurred
// and a channel exists, failure will be indicated on the channel.
func (ds *Datastore) AddTenantChan(c chan bool, tenantID string) {
	ds.cnciAddedLock.Lock()
	ds.cnciAddedChans[tenantID] = c
	ds.cnciAddedLock.Unlock()
}

// AddLimit allows the caller to store a limt for a specific resource for a tenant.
func (ds *Datastore) AddLimit(tenantID string, resourceID int, limit int) error {
	err := ds.db.addLimit(tenantID, resourceID, limit)
	if err != nil {
		return errors.Wrap(err, "error adding limit to database")
	}

	// update cache
	ds.tenantsLock.Lock()

	tenant := ds.tenants[tenantID]
	if tenant != nil {
		resources := tenant.Resources

		for i := range resources {
			if resources[i].Rtype == resourceID {
				resources[i].Limit = limit
				break
			}
		}
	}

	ds.tenantsLock.Unlock()

	return nil
}

func newHardwareAddr() (net.HardwareAddr, error) {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, errors.Wrap(err, "error reading random data")
	}

	// vnic creation seems to require not just the
	// bit 1 to be set, but the entire byte to be
	// set to 2.  Also, ensure that we get no
	// overlap with tenant mac addresses by not allowing
	// byte 1 to ever be zero.
	buf[0] = 2
	if buf[1] == 0 {
		buf[1] = 3
	}

	hw := net.HardwareAddr(buf)

	return hw, nil
}

// AddTenant stores information about a tenant into the datastore.
// it creates a MAC address for the tenant network and makes sure
// that this new tenant is cached.
func (ds *Datastore) AddTenant(id string) (*types.Tenant, error) {
	hw, err := newHardwareAddr()
	if err != nil {
		return nil, errors.Wrap(err, "error creating MAC address")
	}

	err = ds.db.addTenant(id, hw.String())
	if err != nil {
		return nil, errors.Wrapf(err, "error adding tenant (%v) to database", id)
	}

	t, err := ds.getTenant(id)
	if err != nil || t == nil {
		return nil, err
	}

	ds.tenantsLock.Lock()
	ds.tenants[id] = t
	ds.tenantsLock.Unlock()

	return &t.Tenant, nil
}

func (ds *Datastore) getTenant(id string) (*tenant, error) {
	// check cache first
	ds.tenantsLock.RLock()
	t := ds.tenants[id]
	ds.tenantsLock.RUnlock()

	if t != nil {
		return t, nil
	}

	t, err := ds.db.getTenant(id)
	return t, errors.Wrapf(err, "error getting tenant (%v) from database", id)
}

// GetTenant returns details about a tenant referenced by the uuid
func (ds *Datastore) GetTenant(id string) (*types.Tenant, error) {
	t, err := ds.getTenant(id)
	if err != nil || t == nil {
		return nil, err
	}

	return &t.Tenant, nil
}

// AddWorkload is used to add a new workload to the datastore.
// Both cache and persistent store are updated.
func (ds *Datastore) AddWorkload(w types.Workload) error {
	ds.tenantsLock.Lock()
	defer ds.tenantsLock.Unlock()

	tenant, ok := ds.tenants[w.TenantID]
	if !ok {
		return ErrNoTenant
	}

	err := ds.db.updateWorkload(w)
	if err != nil {
		return errors.Wrapf(err, "error updating workload (%v) in database", w.ID)
	}

	// cache it.
	ds.tenants[w.TenantID].workloads = append(tenant.workloads, w)

	return nil
}

// GetWorkload returns details about a specific workload referenced by id
func (ds *Datastore) GetWorkload(tenantID string, ID string) (types.Workload, error) {
	if ID == ds.cnciWorkload.ID {
		return ds.cnciWorkload, nil
	}

	ds.tenantsLock.RLock()
	defer ds.tenantsLock.RUnlock()

	// get any public workloads. These are part of our
	// dummy tenant "public".
	public, ok := ds.tenants["public"]
	if ok {
		for _, wl := range public.workloads {
			if wl.ID == ID {
				return wl, nil
			}
		}
	}

	tenant, ok := ds.tenants[tenantID]
	if !ok {
		return types.Workload{}, ErrNoTenant
	}

	for _, wl := range tenant.workloads {
		if wl.ID == ID {
			return wl, nil
		}
	}

	return types.Workload{}, types.ErrWorkloadNotFound
}

// GetWorkloads retrieves the list of workloads for a particular tenant.
// if there are any public workloads, they will be included in the returned list.
func (ds *Datastore) GetWorkloads(tenantID string) ([]types.Workload, error) {
	var workloads []types.Workload

	// check the cache first
	ds.tenantsLock.RLock()
	defer ds.tenantsLock.RUnlock()

	// get any public workloads. These are part of our
	// dummy tenant "public".
	public, ok := ds.tenants["public"]
	if ok {
		for _, wl := range public.workloads {
			workloads = append(workloads, wl)
		}
	}

	// if there isn't a tenant here, it isn't necessarily an
	// error.
	tenant, ok := ds.tenants[tenantID]
	if !ok {
		return workloads, nil
	}

	workloads = append(workloads, tenant.workloads...)

	return workloads, nil
}

// AddCNCIIP will associate a new IP address with an existing CNCI
// via the mac address
func (ds *Datastore) AddCNCIIP(cnciMAC string, ip string) error {
	var ok bool
	var tenantID string
	var tenant *tenant

	ds.tenantsLock.Lock()

	for tenantID, tenant = range ds.tenants {
		if tenant.CNCIMAC == cnciMAC {
			ok = true
			break
		}
	}

	if !ok {
		ds.tenantsLock.Unlock()
		return ErrNoTenant
	}

	tenant.CNCIIP = ip

	ds.tenantsLock.Unlock()

	err := ds.db.updateTenant(tenant)

	ds.cnciAddedLock.Lock()

	c, ok := ds.cnciAddedChans[tenantID]
	if ok {
		delete(ds.cnciAddedChans, tenantID)
	}

	ds.cnciAddedLock.Unlock()

	if c != nil {
		c <- true
	}

	return errors.Wrap(err, "error updating tenant in database")
}

// AddTenantCNCI will associate a new CNCI instance with a specific tenant.
// The instanceID of the new CNCI instance and the MAC address of the new instance
// are stored in the sql database and updated in the cache.
func (ds *Datastore) AddTenantCNCI(tenantID string, instanceID string, mac string) error {
	// update tenants cache
	ds.tenantsLock.Lock()

	tenant, ok := ds.tenants[tenantID]
	if !ok {
		ds.tenantsLock.Unlock()
		return ErrNoTenant
	}

	tenant.CNCIID = instanceID
	tenant.CNCIMAC = mac

	ds.tenantsLock.Unlock()

	return ds.db.updateTenant(tenant)
}

func (ds *Datastore) removeTenantCNCI(tenantID string) error {
	// update tenants cache
	ds.tenantsLock.Lock()

	tenant, ok := ds.tenants[tenantID]
	if !ok {
		ds.tenantsLock.Unlock()
		return ErrNoTenant
	}

	tenant.CNCIID = ""
	tenant.CNCIIP = ""

	ds.tenantsLock.Unlock()

	return errors.Wrap(ds.db.updateTenant(tenant), "error updating tenant in database")
}

func (ds *Datastore) getTenants() ([]*tenant, error) {
	var tenants []*tenant

	// check the cache first
	ds.tenantsLock.RLock()

	if len(ds.tenants) > 0 {
		for _, value := range ds.tenants {
			tenants = append(tenants, value)
		}

		ds.tenantsLock.RUnlock()

		return tenants, nil
	}

	ds.tenantsLock.RUnlock()

	tenants, err := ds.db.getTenants()
	return tenants, errors.Wrap(err, "error getting tenants from database")
}

// GetAllTenants returns all the tenants from the datastore.
func (ds *Datastore) GetAllTenants() ([]*types.Tenant, error) {
	var tenants []*types.Tenant

	// yes, this makes it so we have to loop through
	// tenants twice, but there probably aren't huge
	// numbers of tenants. I'd rather reuse the code
	// than make this more efficient at this point.
	ts, err := ds.getTenants()
	if err != nil {
		return nil, err
	}

	if len(ts) > 0 {
		for _, value := range ts {
			tenants = append(tenants, &value.Tenant)
		}
	}

	return tenants, nil
}

// ReleaseTenantIP will return an IP address previously allocated to the pool.
// Once a tenant IP address is released, it can be reassigned to another
// instance.
func (ds *Datastore) ReleaseTenantIP(tenantID string, ip string) error {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return errors.New("Invalid IPv4 Address")
	}

	ipBytes := ipAddr.To4()
	if ipBytes == nil {
		return errors.New("Unable to convert ip to bytes")
	}

	subnetInt := binary.BigEndian.Uint16(ipBytes[1:3])

	// clear from cache
	ds.tenantsLock.Lock()

	if ds.tenants[tenantID] != nil {
		ds.tenants[tenantID].network[int(subnetInt)][int(ipBytes[3])] = false
	}

	ds.tenantsLock.Unlock()

	return ds.db.releaseTenantIP(tenantID, int(subnetInt), int(ipBytes[3]))
}

// AllocateTenantIP will find a free IP address within a tenant network.
// For now we make each tenant have unique subnets even though it
// isn't actually needed because of a docker issue.
func (ds *Datastore) AllocateTenantIP(tenantID string) (net.IP, error) {
	var subnetInt uint16
	subnetInt = 0

	ds.tenantsLock.Lock()

	network := ds.tenants[tenantID].network
	subnets := ds.tenants[tenantID].subnets

	// find any subnet assigned to this tenant with available addresses
	sort.Ints(subnets)

	for _, k := range subnets {
		if len(network[k]) < 253 {
			subnetInt = uint16(k)
		}
	}

	var subnetBytes = []byte{16, 0}

	if subnetInt == 0 {
		i := binary.BigEndian.Uint16(subnetBytes)

		for {
			// check for new subnet.
			_, ok := network[int(i)]
			if !ok {
				sub := make(map[int]bool)
				network[int(i)] = sub

				break
			}

			if subnetBytes[1] == 255 {
				if subnetBytes[0] == 31 {
					// out of possible subnets
					glog.Warning("Out of Subnets")
					ds.tenantsLock.Unlock()
					return nil, errors.New("Out of subnets")
				}
				subnetBytes[0]++
				subnetBytes[1] = 0
			} else {
				subnetBytes[1]++
			}

			i = binary.BigEndian.Uint16(subnetBytes)
		}

		subnetInt = i

		ds.tenants[tenantID].subnets = append(subnets, int(subnetInt))
	} else {
		binary.BigEndian.PutUint16(subnetBytes, subnetInt)
	}

	hosts := network[int(subnetInt)]

	rest := 2

	for {
		if hosts[rest] == false {
			hosts[rest] = true
			break
		}

		if rest == 255 {
			// this should never happen
			glog.Warning("ran out of host numbers")

			ds.tenantsLock.Unlock()

			return nil, errors.New("rand out of host numbers")
		}

		rest++
	}

	ds.tenantsLock.Unlock()

	go ds.db.claimTenantIP(tenantID, int(subnetInt), rest)

	// convert to IP type.
	next := net.IPv4(172, subnetBytes[0], subnetBytes[1], byte(rest))

	return next, nil
}

// GetAllInstances retrieves all instances out of the datastore.
func (ds *Datastore) GetAllInstances() ([]*types.Instance, error) {
	var instances []*types.Instance

	// always get from cache
	ds.instancesLock.RLock()

	if len(ds.instances) > 0 {
		for _, val := range ds.instances {
			instances = append(instances, val)
		}
	}

	ds.instancesLock.RUnlock()

	return instances, nil
}

// GetInstance retrieves an instance out of the datastore.
func (ds *Datastore) GetInstance(id string) (*types.Instance, error) {
	// always get from cache
	ds.instancesLock.RLock()

	value, ok := ds.instances[id]

	ds.instancesLock.RUnlock()

	if !ok {
		return nil, types.ErrInstanceNotFound
	}

	return value, nil
}

// GetAllInstancesFromTenant will retrieve all instances belonging to a specific tenant
func (ds *Datastore) GetAllInstancesFromTenant(tenantID string) ([]*types.Instance, error) {
	var instances []*types.Instance

	ds.tenantsLock.RLock()

	t, ok := ds.tenants[tenantID]
	if ok {
		for _, val := range t.instances {
			instances = append(instances, val)
		}

		ds.tenantsLock.RUnlock()

		return instances, nil
	}

	ds.tenantsLock.RUnlock()

	return nil, nil
}

// GetAllInstancesByNode will retrieve all the instances running on a specific compute Node.
func (ds *Datastore) GetAllInstancesByNode(nodeID string) ([]*types.Instance, error) {
	var instances []*types.Instance

	ds.nodesLock.RLock()

	n, ok := ds.nodes[nodeID]
	if ok {
		for _, val := range n.instances {
			instances = append(instances, val)
		}
	}

	ds.nodesLock.RUnlock()

	return instances, nil
}

// AddInstance will store a new instance in the datastore.
// The instance will be updated both in the cache and in the database
func (ds *Datastore) AddInstance(instance *types.Instance) error {
	// add to cache
	ds.instancesLock.Lock()

	ds.instances[instance.ID] = instance

	instanceStat := types.CiaoServerStats{
		ID:        instance.ID,
		TenantID:  instance.TenantID,
		NodeID:    instance.NodeID,
		Timestamp: time.Now(),
		Status:    instance.State,
	}

	ds.instanceLastStatLock.Lock()
	ds.instanceLastStat[instance.ID] = instanceStat
	ds.instanceLastStatLock.Unlock()

	ds.instancesLock.Unlock()

	ds.tenantsLock.Lock()

	tenant := ds.tenants[instance.TenantID]
	if tenant != nil {
		for name, val := range instance.Usage {
			for i := range tenant.Resources {
				if tenant.Resources[i].Rname == name {
					tenant.Resources[i].Usage += val
					break
				}
			}
		}

		// increment instances count
		for i := range tenant.Resources {
			if tenant.Resources[i].Rtype == 1 {
				tenant.Resources[i].Usage++
				break
			}
		}

		tenant.instances[instance.ID] = instance
	}

	ds.tenantsLock.Unlock()

	// update database asynchronously
	go ds.db.addInstance(instance)

	return nil
}

// RestartFailure logs a RestartFailure in the datastore
func (ds *Datastore) RestartFailure(instanceID string, reason payloads.RestartFailureReason) error {
	i, err := ds.GetInstance(instanceID)
	if err != nil {
		return errors.Wrapf(err, "error getting instance (%v)", instanceID)
	}

	msg := fmt.Sprintf("Restart Failure %s: %s", instanceID, reason.String())
	ds.db.logEvent(i.TenantID, string(userError), msg)

	return nil
}

// StopFailure logs a StopFailure in the datastore
func (ds *Datastore) StopFailure(instanceID string, reason payloads.StopFailureReason) error {
	i, err := ds.GetInstance(instanceID)
	if err != nil {
		return errors.Wrapf(err, "error getting instance (%v)", instanceID)
	}

	msg := fmt.Sprintf("Stop Failure %s: %s", instanceID, reason.String())

	ds.db.logEvent(i.TenantID, string(userError), msg)

	return nil
}

// StartFailure will clean up after a failure to start an instance.
// If an instance was a CNCI, this function will remove the CNCI instance
// for this tenant. If the instance was a normal tenant instance, the
// IP address will be released and the instance will be deleted from the
// datastore.
func (ds *Datastore) StartFailure(instanceID string, reason payloads.StartFailureReason) error {
	var tenantID string
	var cnci bool

	ds.tenantsLock.RLock()

	for key, t := range ds.tenants {
		if t.CNCIID == instanceID {
			cnci = true
			tenantID = key
			break
		}
	}

	ds.tenantsLock.RUnlock()

	if cnci == true {
		glog.Warning("CNCI ", instanceID, " Failed to start")

		err := ds.removeTenantCNCI(tenantID)

		msg := fmt.Sprintf("CNCI Start Failure %s: %s", instanceID, reason.String())
		ds.db.logEvent(tenantID, string(userError), msg)

		ds.cnciAddedLock.Lock()

		c, ok := ds.cnciAddedChans[tenantID]
		if ok {
			delete(ds.cnciAddedChans, tenantID)
		}

		ds.cnciAddedLock.Unlock()

		if c != nil {
			c <- false
		}

		return errors.Wrap(err, "error removing CNCI for tenant")
	}

	i, err := ds.GetInstance(instanceID)
	if err != nil {
		return errors.Wrapf(err, "error getting instance (%v)", instanceID)
	}

	switch reason {
	case payloads.FullCloud,
		payloads.FullComputeNode,
		payloads.NoComputeNodes,
		payloads.NoNetworkNodes,
		payloads.InvalidPayload,
		payloads.InvalidData,
		payloads.ImageFailure,
		payloads.NetworkFailure:

		ds.deleteInstance(instanceID)

	case payloads.LaunchFailure,
		payloads.AlreadyRunning,
		payloads.InstanceExists:
	}

	msg := fmt.Sprintf("Start Failure %s: %s", instanceID, reason.String())
	ds.db.logEvent(i.TenantID, string(userError), msg)

	return nil
}

// AttachVolumeFailure will clean up after a failure to attach a volume.
// The volume state will be changed back to available, and an error message
// will be logged.
func (ds *Datastore) AttachVolumeFailure(instanceID string, volumeID string, reason payloads.AttachVolumeFailureReason) error {
	// update the block data to reflect correct state
	data, err := ds.GetBlockDevice(volumeID)
	if err != nil {
		return errors.Wrapf(err, "error getting block device for volume (%v)", volumeID)
	}

	data.State = types.Available
	err = ds.UpdateBlockDevice(data)
	if err != nil {
		return errors.Wrapf(err, "error updating block device for volume (%v)", volumeID)
	}

	// get owner of this instance
	i, err := ds.GetInstance(instanceID)
	if err != nil {
		return errors.Wrapf(err, "error getting instance (%v)", instanceID)
	}

	msg := fmt.Sprintf("Attach Volume Failure %s to %s: %s", volumeID, instanceID, reason.String())

	ds.db.logEvent(i.TenantID, string(userError), msg)

	return nil
}

// DetachVolumeFailure will clean up after a failure to detach a volume.
// The volume state will be changed back to available, and an error message
// will be logged.
func (ds *Datastore) DetachVolumeFailure(instanceID string, volumeID string, reason payloads.DetachVolumeFailureReason) error {
	// update the block data to reflect correct state
	data, err := ds.GetBlockDevice(volumeID)
	if err != nil {
		return errors.Wrapf(err, "error getting block device for volume (%v)", volumeID)
	}

	// because controller wouldn't allow a detach if state
	// wasn't initially InUse, we can blindly set this back
	// to InUse.

	data.State = types.InUse
	err = ds.UpdateBlockDevice(data)
	if err != nil {
		return errors.Wrapf(err, "error updating block device for volume (%v)", volumeID)
	}

	// get owner of this instance
	i, err := ds.GetInstance(instanceID)
	if err != nil {
		return errors.Wrapf(err, "error getting instance (%v)", instanceID)
	}

	msg := fmt.Sprintf("Detach Volume Failure %s from %s: %s", volumeID, instanceID, reason.String())

	ds.db.logEvent(i.TenantID, string(userError), msg)
	return nil
}

func (ds *Datastore) deleteInstance(instanceID string) (string, error) {
	ds.instanceLastStatLock.Lock()
	delete(ds.instanceLastStat, instanceID)
	ds.instanceLastStatLock.Unlock()

	ds.instancesLock.Lock()
	i := ds.instances[instanceID]
	delete(ds.instances, instanceID)
	ds.instancesLock.Unlock()

	ds.tenantsLock.Lock()
	tenant := ds.tenants[i.TenantID]
	delete(tenant.instances, instanceID)
	if tenant != nil {
		for name, val := range i.Usage {
			for i := range tenant.Resources {
				if tenant.Resources[i].Rname == name {
					tenant.Resources[i].Usage -= val
					break
				}
			}
		}
		// decrement instances count
		for i := range tenant.Resources {
			if tenant.Resources[i].Rtype == 1 {
				tenant.Resources[i].Usage--
				break
			}
		}
	}
	ds.tenantsLock.Unlock()

	// we may not have received any node stats for this instance
	if i.NodeID != "" {
		ds.nodesLock.Lock()
		delete(ds.nodes[i.NodeID].instances, instanceID)
		ds.nodesLock.Unlock()
	}

	var err error
	if tmpErr := ds.db.deleteInstance(i.ID); tmpErr != nil {
		glog.Warningf("error deleting instance (%v): %v", i.ID, err)
		err = errors.Wrapf(tmpErr, "error deleting instance from database (%v)", i.ID)
	}

	if tmpErr := ds.ReleaseTenantIP(i.TenantID, i.IPAddress); tmpErr != nil {
		glog.Warningf("error releasing IP for instance (%v): %v", i.ID, err)
		if err == nil {
			err = errors.Wrapf(err, "error releasing IP for instance (%v)", i.ID)
		}
	}

	ds.updateStorageAttachments(instanceID, nil)

	return i.TenantID, err
}

// DeleteInstance removes an instance from the datastore.
func (ds *Datastore) DeleteInstance(instanceID string) error {
	tenantID, err := ds.deleteInstance(instanceID)
	if err != nil {
		return errors.Wrapf(err, "error deleting instance")
	}

	msg := fmt.Sprintf("Deleted Instance %s", instanceID)
	ds.db.logEvent(tenantID, string(userInfo), msg)

	return nil
}

// DeleteNode removes a node from the node cache.
func (ds *Datastore) DeleteNode(nodeID string) error {
	ds.nodesLock.Lock()
	delete(ds.nodes, nodeID)
	ds.nodesLock.Unlock()

	ds.nodeLastStatLock.Lock()
	delete(ds.nodeLastStat, nodeID)
	ds.nodeLastStatLock.Unlock()

	return nil
}

// HandleStats makes sure that the data from the stat payload is stored.
func (ds *Datastore) HandleStats(stat payloads.Stat) error {
	if stat.Load != -1 {
		ds.addNodeStat(stat)
	}

	return errors.Wrapf(ds.addInstanceStats(stat.Instances, stat.NodeUUID), "error updating stats")
}

// HandleTraceReport stores the provided trace data in the datastore.
func (ds *Datastore) HandleTraceReport(trace payloads.Trace) error {
	var err error
	for index := range trace.Frames {
		i := trace.Frames[index]

		if tmpErr := ds.db.addFrameStat(i); tmpErr != nil {
			if err == nil {
				err = errors.Wrapf(tmpErr, "error adding stats to database")
			}
		}
	}

	return err
}

// GetInstanceLastStats retrieves the last instances stats received for this node.
// It returns it in a format suitable for the compute API.
func (ds *Datastore) GetInstanceLastStats(nodeID string) types.CiaoServersStats {
	var serversStats types.CiaoServersStats

	ds.instanceLastStatLock.RLock()
	for _, instance := range ds.instanceLastStat {
		if instance.NodeID != nodeID {
			continue
		}
		serversStats.Servers = append(serversStats.Servers, instance)
	}
	ds.instanceLastStatLock.RUnlock()

	return serversStats
}

// GetNodeLastStats retrieves the last nodes stats received for this node.
// It returns it in a format suitable for the compute API.
func (ds *Datastore) GetNodeLastStats() types.CiaoComputeNodes {
	var computeNodes types.CiaoComputeNodes

	ds.nodeLastStatLock.RLock()
	for _, node := range ds.nodeLastStat {
		computeNodes.Nodes = append(computeNodes.Nodes, node)
	}
	ds.nodeLastStatLock.RUnlock()

	return computeNodes
}

func (ds *Datastore) addNodeStat(stat payloads.Stat) error {
	ds.nodesLock.Lock()

	n, ok := ds.nodes[stat.NodeUUID]
	if !ok {
		n = &node{}
		n.instances = make(map[string]*types.Instance)
		ds.nodes[stat.NodeUUID] = n
	}

	n.ID = stat.NodeUUID
	n.Hostname = stat.NodeHostName

	ds.nodesLock.Unlock()

	cnStat := types.CiaoComputeNode{
		ID:            stat.NodeUUID,
		Status:        stat.Status,
		Load:          stat.Load,
		MemTotal:      stat.MemTotalMB,
		MemAvailable:  stat.MemAvailableMB,
		DiskTotal:     stat.DiskTotalMB,
		DiskAvailable: stat.DiskAvailableMB,
		OnlineCPUs:    stat.CpusOnline,
	}

	ds.nodeLastStatLock.Lock()

	delete(ds.nodeLastStat, stat.NodeUUID)
	ds.nodeLastStat[stat.NodeUUID] = cnStat

	ds.nodeLastStatLock.Unlock()

	return errors.Wrap(ds.db.addNodeStat(stat), "error adding node stats to database")
}

var tenantUsagePeriodMinutes float64 = 5

func (ds *Datastore) updateTenantUsageNeeded(delta types.CiaoUsage, tenantID string) bool {
	if delta.VCPU == 0 &&
		delta.Memory == 0 &&
		delta.Disk == 0 {
		return false
	}

	return true
}

func (ds *Datastore) updateTenantUsage(delta types.CiaoUsage, tenantID string) {
	if ds.updateTenantUsageNeeded(delta, tenantID) == false {
		return
	}

	createNewUsage := true
	lastUsage := types.CiaoUsage{}

	ds.tenantUsageLock.Lock()

	tenantUsage := ds.tenantUsage[tenantID]
	if len(tenantUsage) != 0 {
		lastUsage = tenantUsage[len(tenantUsage)-1]
		// We will not create more than one entry per tenant every tenantUsagePeriodMinutes
		if time.Since(lastUsage.Timestamp).Minutes() < tenantUsagePeriodMinutes {
			createNewUsage = false
		}
	}

	newUsage := types.CiaoUsage{
		VCPU:   lastUsage.VCPU + delta.VCPU,
		Memory: lastUsage.Memory + delta.Memory,
		Disk:   lastUsage.Disk + delta.Disk,
	}

	// If we need to create a new usage entry, we timestamp it now.
	// If not we just update the last entry.
	if createNewUsage == true {
		newUsage.Timestamp = time.Now()
		ds.tenantUsage[tenantID] = append(ds.tenantUsage[tenantID], newUsage)
	} else {
		newUsage.Timestamp = lastUsage.Timestamp
		tenantUsage[len(tenantUsage)-1] = newUsage
	}

	ds.tenantUsageLock.Unlock()
}

// GetTenantUsage provides statistics on actual resource usage.
// Usage is provided between a specified time period.
func (ds *Datastore) GetTenantUsage(tenantID string, start time.Time, end time.Time) ([]types.CiaoUsage, error) {
	ds.tenantUsageLock.RLock()
	defer ds.tenantUsageLock.RUnlock()

	tenantUsage := ds.tenantUsage[tenantID]
	if tenantUsage == nil || len(tenantUsage) == 0 {
		return nil, nil
	}

	historyLength := len(tenantUsage)
	if tenantUsage[0].Timestamp.After(end) == true ||
		start.After(tenantUsage[historyLength-1].Timestamp) == true {
		return nil, nil
	}

	first := 0
	last := 0
	for _, u := range tenantUsage {
		if start.After(u.Timestamp) == true {
			first++
		}

		if end.After(u.Timestamp) == true {
			last++
		}
	}

	return tenantUsage[first:last], nil
}

func reduceToZero(v int) int {
	if v < 0 {
		return 0
	}

	return v
}

func (ds *Datastore) addInstanceStats(stats []payloads.InstanceStat, nodeID string) error {
	for index := range stats {
		stat := stats[index]

		instanceStat := types.CiaoServerStats{
			ID:        stat.InstanceUUID,
			NodeID:    nodeID,
			Timestamp: time.Now(),
			Status:    stat.State,
			VCPUUsage: reduceToZero(stat.CPUUsage),
			MemUsage:  reduceToZero(stat.MemoryUsageMB),
			DiskUsage: reduceToZero(stat.DiskUsageMB),
		}

		ds.instanceLastStatLock.Lock()

		lastInstanceStat := ds.instanceLastStat[stat.InstanceUUID]

		deltaUsage := types.CiaoUsage{
			VCPU:   instanceStat.VCPUUsage - lastInstanceStat.VCPUUsage,
			Memory: instanceStat.MemUsage - lastInstanceStat.MemUsage,
			Disk:   instanceStat.DiskUsage - lastInstanceStat.DiskUsage,
		}

		ds.updateTenantUsage(deltaUsage, lastInstanceStat.TenantID)

		instanceStat.TenantID = lastInstanceStat.TenantID

		delete(ds.instanceLastStat, stat.InstanceUUID)
		ds.instanceLastStat[stat.InstanceUUID] = instanceStat

		ds.instanceLastStatLock.Unlock()

		ds.instancesLock.Lock()
		instance, ok := ds.instances[stat.InstanceUUID]
		if ok {
			instance.State = stat.State
			instance.NodeID = nodeID
			instance.SSHIP = stat.SSHIP
			instance.SSHPort = stat.SSHPort
			ds.nodesLock.Lock()
			ds.nodes[nodeID].instances[instance.ID] = instance
			ds.nodesLock.Unlock()
		}
		ds.instancesLock.Unlock()

		ds.updateStorageAttachments(stat.InstanceUUID, stat.Volumes)
	}

	return errors.Wrapf(ds.db.addInstanceStats(stats, nodeID), "error adding instance stats to database")
}

// GetTenantCNCISummary retrieves information about a given CNCI id, or all CNCIs
// If the cnci string is the null string, then this function will retrieve all
// tenants.  If cnci is not null, it will only provide information about a specific
// cnci.
func (ds *Datastore) GetTenantCNCISummary(cnci string) ([]types.TenantCNCI, error) {
	var cncis []types.TenantCNCI
	subnetBytes := []byte{0, 0}

	ds.tenantsLock.RLock()

	for _, t := range ds.tenants {
		if cnci != "" && cnci != t.CNCIID {
			continue
		}

		cn := types.TenantCNCI{
			TenantID:   t.ID,
			IPAddress:  t.CNCIIP,
			MACAddress: t.CNCIMAC,
			InstanceID: t.CNCIID,
		}

		for _, subnet := range t.subnets {
			binary.BigEndian.PutUint16(subnetBytes, (uint16)(subnet))
			cn.Subnets = append(cn.Subnets, fmt.Sprintf("Subnet 172.%d.%d.0/8", subnetBytes[0], subnetBytes[1]))
		}

		cncis = append(cncis, cn)

		if cnci != "" && cnci == t.CNCIID {
			break
		}
	}

	ds.tenantsLock.RUnlock()

	return cncis, nil
}

// GetCNCIWorkloadID returns the UUID of the workload template
// for the CNCI workload
func (ds *Datastore) GetCNCIWorkloadID() (string, error) {
	if ds.cnciWorkload.ID == "" {
		return "", errors.New("No CNCI Workload in datastore")
	}

	return ds.cnciWorkload.ID, nil
}

// GetNodeSummary provides a summary the state and count of instances running per node.
func (ds *Datastore) GetNodeSummary() ([]*types.NodeSummary, error) {
	// TBD: write a new routine that grabs the node summary info
	// from the cache rather than do this lengthy sql query.
	return ds.db.getNodeSummary()
}

// GetBatchFrameSummary will retieve the count of traces we have for a specific label
func (ds *Datastore) GetBatchFrameSummary() ([]types.BatchFrameSummary, error) {
	// until we start caching frame stats, we have to send this
	// right through to the database.
	return ds.db.getBatchFrameSummary()
}

// GetBatchFrameStatistics will show individual trace data per instance for a batch of trace data.
// The batch is identified by the label.
func (ds *Datastore) GetBatchFrameStatistics(label string) ([]types.BatchFrameStat, error) {
	// until we start caching frame stats, we have to send this
	// right through to the database.
	return ds.db.getBatchFrameStatistics(label)
}

// GetEventLog retrieves all the log entries stored in the datastore.
func (ds *Datastore) GetEventLog() ([]*types.LogEntry, error) {
	// we don't as of yet cache any of the events that are logged.
	return ds.db.getEventLog()
}

// ClearLog will remove all the event entries from the event log
func (ds *Datastore) ClearLog() error {
	// we don't as of yet cache any of the events that are logged.
	return ds.db.clearLog()
}

// LogEvent will add a message to the persistent event log.
func (ds *Datastore) LogEvent(tenant string, msg string) {
	ds.db.logEvent(tenant, string(userInfo), msg)
}

// AddBlockDevice will store information about new BlockData into
// the datastore.
func (ds *Datastore) AddBlockDevice(device types.BlockData) error {
	ds.bdLock.Lock()
	_, update := ds.blockDevices[device.ID]
	ds.blockDevices[device.ID] = device
	ds.bdLock.Unlock()

	// update tenants cache
	ds.tenantsLock.Lock()
	devices := ds.tenants[device.TenantID].devices
	devices[device.ID] = device
	ds.tenantsLock.Unlock()

	// store persistently
	if !update {
		go ds.db.addBlockData(device)
	} else {
		go ds.db.updateBlockData(device)
	}

	return nil
}

// DeleteBlockDevice will delete a volume from the datastore.
// It also deletes it from the tenant's list of devices.
func (ds *Datastore) DeleteBlockDevice(ID string) error {
	// lock both tenants and devices maps
	var err error

	ds.bdLock.Lock()
	ds.tenantsLock.Lock()

	dev, ok := ds.blockDevices[ID]
	if ok {
		delete(ds.blockDevices, ID)
		delete(ds.tenants[dev.TenantID].devices, ID)
	}

	ds.tenantsLock.Unlock()
	ds.bdLock.Unlock()

	if ok {
		go ds.db.deleteBlockData(ID)
	} else {
		err = ErrNoBlockData
	}

	return err
}

// GetBlockDevices will return all the BlockDevices associated with a tenant.
func (ds *Datastore) GetBlockDevices(tenant string) ([]types.BlockData, error) {
	var devices []types.BlockData

	ds.tenantsLock.RLock()

	_, ok := ds.tenants[tenant]
	if !ok {
		ds.tenantsLock.RUnlock()
		return devices, ErrNoTenant
	}

	for _, value := range ds.tenants[tenant].devices {
		devices = append(devices, value)
	}

	ds.tenantsLock.RUnlock()

	return devices, nil

}

// GetBlockDevice will return information about a block device from the
// datastore.
func (ds *Datastore) GetBlockDevice(ID string) (types.BlockData, error) {
	ds.bdLock.RLock()
	data, ok := ds.blockDevices[ID]
	ds.bdLock.RUnlock()

	if !ok {
		return types.BlockData{}, ErrNoBlockData
	}
	return data, nil
}

// UpdateBlockDevice will replace existing information about a block device
// in the datastore.
func (ds *Datastore) UpdateBlockDevice(data types.BlockData) error {
	ds.bdLock.RLock()
	_, ok := ds.blockDevices[data.ID]
	ds.bdLock.RUnlock()

	if !ok {
		return ErrNoBlockData
	}

	return errors.Wrapf(ds.AddBlockDevice(data), "error updating block device (%v)", data.ID)
}

// CreateStorageAttachment will associate an instance with a block device in
// the datastore
func (ds *Datastore) CreateStorageAttachment(instanceID string, volume payloads.StorageResource) (types.StorageAttachment, error) {
	link := attachment{
		instanceID: instanceID,
		volumeID:   volume.ID,
	}

	a := types.StorageAttachment{
		InstanceID: instanceID,
		ID:         uuid.Generate().String(),
		BlockID:    volume.ID,
		Ephemeral:  volume.Ephemeral,
		Boot:       volume.Bootable,
	}

	err := ds.db.addStorageAttachment(a)
	if err != nil {
		return types.StorageAttachment{}, errors.Wrap(err, "error adding storage attachment to database")
	}

	// ensure that the volume is marked in use as we have created an attachment
	bd, err := ds.GetBlockDevice(volume.ID)
	if err != nil {
		ds.db.deleteStorageAttachment(a.ID)
		return types.StorageAttachment{}, errors.Wrapf(err, "error fetching block device (%v)", volume.ID)
	}

	bd.State = types.InUse
	err = ds.UpdateBlockDevice(bd)
	if err != nil {
		ds.db.deleteStorageAttachment(a.ID)
		return types.StorageAttachment{}, errors.Wrapf(err, "error updating block device (%v)", volume.ID)
	}

	// add it to our links map
	ds.attachLock.Lock()
	ds.attachments[a.ID] = a
	ds.instanceVolumes[link] = a.ID
	ds.attachLock.Unlock()

	return a, nil
}

// GetStorageAttachments returns a list of volumes associated with this instance.
func (ds *Datastore) GetStorageAttachments(instanceID string) []types.StorageAttachment {
	var links []types.StorageAttachment

	ds.attachLock.RLock()
	for _, a := range ds.attachments {
		if a.InstanceID == instanceID {
			links = append(links, a)
		}
	}
	ds.attachLock.RUnlock()

	return links
}

func (ds *Datastore) updateStorageAttachments(instanceID string, volumes []string) {
	m := make(map[string]bool)

	// this for handy searching.
	for _, v := range volumes {
		m[v] = true
	}

	// see if we already know about each attachment.
	ds.attachLock.Lock()

	for _, v := range volumes {
		key := attachment{
			instanceID: instanceID,
			volumeID:   v,
		}

		_, ok := ds.instanceVolumes[key]
		if !ok {
			// add the attachment
			a := types.StorageAttachment{
				InstanceID: instanceID,
				ID:         uuid.Generate().String(),
				BlockID:    v,
			}
			ds.attachments[a.ID] = a
			ds.instanceVolumes[key] = a.ID

			// not sure what to do with an error here.
			err := ds.db.addStorageAttachment(a)
			if err != nil {
				glog.Warningf("error adding storage attachment to database: %v", err)
				continue
			}

			// update the state of the volume.
			bd, err := ds.GetBlockDevice(v)
			if err != nil {
				glog.Warningf("error fetching block device (%v): %v", v, err)
				// well, maybe we should add it, it obviously
				// exists.
				continue
			}

			bd.State = types.InUse
			err = ds.UpdateBlockDevice(bd)
			if err != nil {
				glog.Warningf("error updating block device (%v): %v", v, err)
			}
		}
	}

	// finally, check to see if all the attachments we already
	// know about are in the list.
	for _, ID := range ds.instanceVolumes {
		a := ds.attachments[ID]

		if a.InstanceID == instanceID && !m[a.BlockID] {
			bd, err := ds.GetBlockDevice(a.BlockID)
			if err != nil {
				glog.Warningf("error fetching block device (%v): %v", a.BlockID, err)
				continue
			}

			// update the state of the volume.
			bd.State = types.Available
			err = ds.UpdateBlockDevice(bd)
			if err != nil {
				glog.Warningf("error updating block device (%v): %v", a.BlockID, err)
			}

			// delete the attachment.
			key := attachment{
				instanceID: a.InstanceID,
				volumeID:   a.BlockID,
			}

			delete(ds.attachments, ID)
			delete(ds.instanceVolumes, key)

			// update persistent store asynch.
			// ok for lock to be held here, but
			// not needed as the db keeps it's
			// own locks.
			go ds.db.deleteStorageAttachment(ID)
		}
	}
	ds.attachLock.Unlock()
}

func (ds *Datastore) getStorageAttachment(instanceID string, volumeID string) (types.StorageAttachment, error) {
	var a types.StorageAttachment

	key := attachment{
		instanceID: instanceID,
		volumeID:   volumeID,
	}

	ds.attachLock.RLock()
	id, ok := ds.instanceVolumes[key]
	if ok {
		a = ds.attachments[id]
	}
	ds.attachLock.RUnlock()

	if !ok {
		return a, ErrNoStorageAttachment
	}

	return a, nil
}

// DeleteStorageAttachment will delete the attachment with the associated ID
// from the datastore.
func (ds *Datastore) DeleteStorageAttachment(ID string) error {
	ds.attachLock.Lock()
	a, ok := ds.attachments[ID]
	if ok {
		key := attachment{
			instanceID: a.InstanceID,
			volumeID:   a.BlockID,
		}

		delete(ds.attachments, ID)
		delete(ds.instanceVolumes, key)
	}
	ds.attachLock.Unlock()

	if !ok {
		return ErrNoStorageAttachment
	}

	return errors.Wrapf(ds.db.deleteStorageAttachment(ID), "error deleting storage attachment (%v) from database", ID)
}

// GetVolumeAttachments will return a list of attachments associated with
// this volume ID.
func (ds *Datastore) GetVolumeAttachments(volume string) ([]types.StorageAttachment, error) {
	var attachments []types.StorageAttachment

	ds.attachLock.RLock()

	for _, a := range ds.attachments {
		if a.BlockID == volume {
			attachments = append(attachments, a)
		}
	}

	ds.attachLock.RUnlock()

	return attachments, nil
}

// GetPool will return an external IP Pool
func (ds *Datastore) GetPool(ID string) (types.Pool, error) {
	ds.poolsLock.RLock()
	p, ok := ds.pools[ID]
	ds.poolsLock.RUnlock()

	if !ok {
		return p, types.ErrPoolNotFound
	}

	return p, nil
}

// GetPools will return a list of external IP Pools
func (ds *Datastore) GetPools() ([]types.Pool, error) {
	var pools []types.Pool

	ds.poolsLock.RLock()

	for _, p := range ds.pools {
		pools = append(pools, p)
	}

	ds.poolsLock.RUnlock()

	return pools, nil
}

// lock for the map must be held by caller.
func (ds *Datastore) isDuplicateSubnet(new *net.IPNet) bool {
	for s, exists := range ds.externalSubnets {
		if exists == true {
			// this will always succeed
			_, subnet, _ := net.ParseCIDR(s)

			if subnet.Contains(new.IP) || new.Contains(subnet.IP) {
				return true
			}
		}
	}

	return false
}

// lock for the map must be held by the caller
func (ds *Datastore) isDuplicateIP(new net.IP) bool {
	// first make sure the IP isn't covered by a subnet
	for s, exists := range ds.externalSubnets {
		// this will always succeed
		_, subnet, _ := net.ParseCIDR(s)

		if exists == true {
			if subnet.Contains(new) {
				return true
			}
		}
	}

	// next make sure that the IP isn't already in a
	// different pool
	return ds.externalIPs[new.String()]
}

// AddPool will add a brand new pool to our datastore.
func (ds *Datastore) AddPool(pool types.Pool) error {
	ds.poolsLock.Lock()

	if len(pool.Subnets) > 0 {
		// check each one to make sure it's not in use.
		for _, subnet := range pool.Subnets {
			_, newSubnet, err := net.ParseCIDR(subnet.CIDR)
			if err != nil {
				ds.poolsLock.Unlock()
				return errors.Wrapf(err, "unable to parse subnet CIDR (%v)", subnet.CIDR)
			}

			if ds.isDuplicateSubnet(newSubnet) {
				ds.poolsLock.Unlock()
				return types.ErrDuplicateSubnet
			}

			// update our list of used subnets
			ds.externalSubnets[subnet.CIDR] = true
		}
	} else if len(pool.IPs) > 0 {
		var newIPs []net.IP

		// make sure valid and not duplicate
		for _, newIP := range pool.IPs {
			IP := net.ParseIP(newIP.Address)
			if IP == nil {
				ds.poolsLock.Unlock()
				return types.ErrInvalidIP
			}

			if ds.isDuplicateIP(IP) {
				ds.poolsLock.Unlock()
				return types.ErrDuplicateIP
			}

			newIPs = append(newIPs, IP)
		}

		// now that the whole list is confirmed, we can update
		for _, IP := range newIPs {
			ds.externalIPs[IP.String()] = true
		}
	}

	ds.pools[pool.ID] = pool
	err := ds.db.addPool(pool)

	ds.poolsLock.Unlock()

	if err != nil {
		// lock must not be held when calling.
		ds.DeletePool(pool.ID)
	}

	return errors.Wrap(err, "error adding pool to database")
}

// DeletePool will delete an unused pool from our datastore.
func (ds *Datastore) DeletePool(ID string) error {
	ds.poolsLock.Lock()
	defer ds.poolsLock.Unlock()

	p, ok := ds.pools[ID]
	if !ok {
		return types.ErrPoolNotFound
	}

	// make sure all ips in this pool are not used.
	if p.Free != p.TotalIPs {
		return types.ErrPoolNotEmpty
	}

	// delete from persistent store
	err := ds.db.deletePool(ID)
	if err != nil {
		return errors.Wrapf(err, "error deleting pool (%v) from database", ID)
	}

	// delete all subnets
	for _, subnet := range p.Subnets {
		delete(ds.externalSubnets, subnet.CIDR)
	}

	// delete any individual IPs
	for _, IP := range p.IPs {
		delete(ds.externalIPs, IP.Address)
	}

	// delete the whole pool
	delete(ds.pools, ID)

	return nil
}

// AddExternalSubnet will add a new subnet to an existing pool.
func (ds *Datastore) AddExternalSubnet(poolID string, subnet string) error {
	sub := types.ExternalSubnet{
		ID:   uuid.Generate().String(),
		CIDR: subnet,
	}

	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return errors.Wrapf(err, "unable to parse subnet CIDR (%v)", subnet)
	}

	ds.poolsLock.Lock()
	defer ds.poolsLock.Unlock()

	p, ok := ds.pools[poolID]
	if !ok {
		return types.ErrPoolNotFound
	}

	if ds.isDuplicateSubnet(ipNet) {
		return types.ErrDuplicateSubnet
	}

	ones, bits := ipNet.Mask.Size()

	// deduct gateway and broadcast
	newIPs := (1 << uint32(bits-ones)) - 2
	p.TotalIPs += newIPs
	p.Free += newIPs
	p.Subnets = append(p.Subnets, sub)

	err = ds.db.updatePool(p)
	if err != nil {
		return errors.Wrap(err, "error updating pool in database")
	}

	// we are committed now.
	ds.pools[poolID] = p
	ds.externalSubnets[sub.CIDR] = true

	return nil
}

// AddExternalIPs will add a list of individual IPs to an existing pool.
func (ds *Datastore) AddExternalIPs(poolID string, IPs []string) error {
	ds.poolsLock.Lock()
	defer ds.poolsLock.Unlock()

	p, ok := ds.pools[poolID]
	if !ok {
		return types.ErrPoolNotFound
	}

	// make sure valid and not duplicate
	for _, newIP := range IPs {
		IP := net.ParseIP(newIP)
		if IP == nil {
			return types.ErrInvalidIP
		}

		if ds.isDuplicateIP(IP) {
			return types.ErrDuplicateIP
		}

		ExtIP := types.ExternalIP{
			ID:      uuid.Generate().String(),
			Address: IP.String(),
		}

		p.TotalIPs++
		p.Free++
		p.IPs = append(p.IPs, ExtIP)
	}

	// update persistent store.
	err := ds.db.updatePool(p)
	if err != nil {
		return errors.Wrap(err, "error updating pool in database")
	}

	// update cache.
	for _, IP := range p.IPs {
		ds.externalIPs[IP.Address] = true
	}
	ds.pools[poolID] = p

	return nil
}

// DeleteSubnet will remove an unused subnet from an existing pool.
func (ds *Datastore) DeleteSubnet(poolID string, subnetID string) error {
	ds.poolsLock.Lock()
	defer ds.poolsLock.Unlock()

	p, ok := ds.pools[poolID]
	if !ok {
		return types.ErrPoolNotFound
	}

	for i, sub := range p.Subnets {
		if sub.ID != subnetID {
			continue
		}

		// this path will be taken only once.
		IP, ipNet, err := net.ParseCIDR(sub.CIDR)
		if err != nil {
			return errors.Wrapf(err, "unable to parse subnet CIDR (%v)", sub.CIDR)
		}

		// check each address in this subnet is not mapped.
		for IP := IP.Mask(ipNet.Mask); ipNet.Contains(IP); incrementIP(IP) {
			_, ok := ds.mappedIPs[IP.String()]
			if ok {
				return types.ErrPoolNotEmpty
			}
		}

		ones, bits := ipNet.Mask.Size()
		numIPs := (1 << uint32(bits-ones)) - 2
		p.TotalIPs -= numIPs
		p.Free -= numIPs
		p.Subnets = append(p.Subnets[:i], p.Subnets[i+1:]...)

		err = ds.db.updatePool(p)
		if err != nil {
			return errors.Wrap(err, "error updating pool in database")
		}

		delete(ds.externalSubnets, sub.CIDR)
		ds.pools[poolID] = p

		return nil
	}

	return types.ErrInvalidPoolAddress
}

// DeleteExternalIP will remove an individual IP address from a pool.
func (ds *Datastore) DeleteExternalIP(poolID string, addrID string) error {
	ds.poolsLock.Lock()
	defer ds.poolsLock.Unlock()

	p, ok := ds.pools[poolID]
	if !ok {
		return types.ErrPoolNotFound
	}

	for i, extIP := range p.IPs {
		if extIP.ID != addrID {
			continue
		}

		// this path will be taken only once.
		// check address is not mapped.
		_, ok := ds.mappedIPs[extIP.Address]
		if ok {
			return types.ErrPoolNotEmpty
		}

		p.TotalIPs--
		p.Free--
		p.IPs = append(p.IPs[:i], p.IPs[i+1:]...)

		err := ds.db.updatePool(p)
		if err != nil {
			return errors.Wrap(err, "error updating pool in database")
		}

		delete(ds.externalIPs, extIP.Address)
		ds.pools[poolID] = p

		return nil
	}

	return types.ErrInvalidPoolAddress
}

func incrementIP(IP net.IP) {
	for i := len(IP) - 1; i >= 0; i-- {
		IP[i]++
		if IP[i] > 0 {
			break
		}
	}
}

// GetMappedIPs will return a list of mapped external IPs by tenant.
func (ds *Datastore) GetMappedIPs(tenant *string) []types.MappedIP {
	var mappedIPs []types.MappedIP

	ds.poolsLock.RLock()
	defer ds.poolsLock.RUnlock()

	for _, m := range ds.mappedIPs {
		if tenant != nil {
			if m.TenantID != *tenant {
				continue
			}
		}
		mappedIPs = append(mappedIPs, m)
	}

	return mappedIPs
}

// GetMappedIP will return a MappedIP struct for the given address.
func (ds *Datastore) GetMappedIP(address string) (types.MappedIP, error) {
	ds.poolsLock.RLock()
	defer ds.poolsLock.RUnlock()

	m, ok := ds.mappedIPs[address]
	if !ok {
		return types.MappedIP{}, types.ErrAddressNotFound
	}

	return m, nil
}

// MapExternalIP will allocate an external IP to an instance from a given pool.
func (ds *Datastore) MapExternalIP(poolID string, instanceID string) (types.MappedIP, error) {
	var m types.MappedIP

	instance, err := ds.GetInstance(instanceID)
	if err != nil {
		return m, errors.Wrapf(err, "error getting instance (%v)", instanceID)
	}

	ds.poolsLock.Lock()
	defer ds.poolsLock.Unlock()

	pool, ok := ds.pools[poolID]
	if !ok {
		return m, types.ErrPoolNotFound
	}

	if pool.Free == 0 {
		return m, types.ErrPoolEmpty
	}

	// find a free IP address in any subnet.
	for _, sub := range pool.Subnets {
		IP, ipNet, err := net.ParseCIDR(sub.CIDR)
		if err != nil {
			return m, errors.Wrapf(err, "error parsing subnet CIDR (%v)", sub.CIDR)
		}

		initIP := IP.Mask(ipNet.Mask)

		// skip gateway
		incrementIP(initIP)

		// check each address in this subnet
		for IP := initIP; ipNet.Contains(IP); incrementIP(IP) {
			_, ok := ds.mappedIPs[IP.String()]
			if !ok {
				m.ID = uuid.Generate().String()
				m.ExternalIP = IP.String()
				m.InternalIP = instance.IPAddress
				m.InstanceID = instanceID
				m.TenantID = instance.TenantID
				m.PoolID = pool.ID
				m.PoolName = pool.Name

				pool.Free--

				err = ds.db.addMappedIP(m)
				if err != nil {
					return types.MappedIP{}, errors.Wrap(err, "error adding IP mapping to database")
				}
				ds.mappedIPs[IP.String()] = m

				err = ds.db.updatePool(pool)
				if err != nil {
					return types.MappedIP{}, errors.Wrap(err, "error updating pool in database")
				}

				ds.pools[poolID] = pool

				return m, nil
			}
		}
	}

	// we are still looking. Check our individual IPs
	for _, IP := range pool.IPs {
		_, ok := ds.mappedIPs[IP.Address]
		if !ok {
			m.ID = uuid.Generate().String()
			m.ExternalIP = IP.Address
			m.InternalIP = instance.IPAddress
			m.InstanceID = instanceID
			m.TenantID = instance.TenantID
			m.PoolID = pool.ID
			m.PoolName = pool.Name

			pool.Free--

			err = ds.db.addMappedIP(m)
			if err != nil {
				return types.MappedIP{}, errors.Wrap(err, "error adding IP mapping to database")
			}
			ds.mappedIPs[IP.Address] = m

			err = ds.db.updatePool(pool)
			if err != nil {
				return types.MappedIP{}, errors.Wrap(err, "error updating pool in database")
			}

			ds.pools[poolID] = pool

			return m, nil
		}
	}

	// if you got here you are out of luck. But you never should.
	glog.Warningf("Pool reports %d free addresses but none found", pool.Free)
	return m, types.ErrPoolEmpty
}

// UnMapExternalIP will stop associating a given address with an instance.
func (ds *Datastore) UnMapExternalIP(address string) error {
	ds.poolsLock.Lock()
	defer ds.poolsLock.Unlock()

	m, ok := ds.mappedIPs[address]
	if !ok {
		return types.ErrAddressNotFound
	}

	// get pool and update Free
	pool, ok := ds.pools[m.PoolID]
	if !ok {
		return types.ErrPoolNotFound
	}

	pool.Free++

	err := ds.db.deleteMappedIP(m.ID)
	if err != nil {
		return errors.Wrap(err, "error deleting IP mapping from database")
	}
	delete(ds.mappedIPs, address)

	err = ds.db.updatePool(pool)
	if err != nil {
		return errors.Wrap(err, "error updating pool in database")
	}

	ds.pools[pool.ID] = pool

	return nil
}

// GenerateCNCIWorkload is used to create a workload definition for the CNCI.
// This function should be called prior to any workload launch.
func (ds *Datastore) GenerateCNCIWorkload(vcpus int, memMB int, diskMB int, key string, password string) {
	// generate the CNCI workload.
	config := `---
#cloud-config
users:
  - name: cloud-admin
    gecos: CIAO Cloud Admin
    lock-passwd: false
    passwd: ` + password + `
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
    - ` + key + `
...
`
	cpus := payloads.RequestedResource{
		Type:      payloads.VCPUs,
		Value:     vcpus,
		Mandatory: false,
	}

	mem := payloads.RequestedResource{
		Type:      payloads.MemMB,
		Value:     memMB,
		Mandatory: false,
	}

	network := payloads.RequestedResource{
		Type:      payloads.NetworkNode,
		Value:     1,
		Mandatory: true,
	}

	storage := types.StorageResource{
		ID:         "",
		Bootable:   true,
		Ephemeral:  false,
		SourceType: types.ImageService,
		SourceID:   "4e16e743-265a-4bf2-9fd1-57ada0b28904",
	}

	wl := types.Workload{
		ID:          uuid.Generate().String(),
		Description: "CNCI",
		FWType:      string(payloads.EFI),
		VMType:      payloads.QEMU,
		Config:      config,
		Defaults:    []payloads.RequestedResource{cpus, mem, network},
		Storage:     []types.StorageResource{storage},
	}

	// for now we have a single global cnci workload.
	ds.cnciWorkload = wl
}

// GetQuotas returns the set of quotas from the database without any caching.
func (ds *Datastore) GetQuotas(tenantID string) ([]types.QuotaDetails, error) {
	return ds.db.getQuotas(tenantID)
}

// UpdateQuotas updates the quotas for a tenant in the database.
func (ds *Datastore) UpdateQuotas(tenantID string, qds []types.QuotaDetails) error {
	return ds.db.updateQuotas(tenantID, qds)
}
