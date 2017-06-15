//
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
//

package main

import (
	"sync"
	"time"

	"github.com/01org/ciao/database"
	"github.com/01org/ciao/payloads"
	"github.com/pkg/errors"
)

// cnciDatabase
type cnciDatabase struct {
	database.DbProvider //Database used to persist the CNCI state
	SubnetMap
	PublicIPMap
}

const (
	tableSubnetMap   = "SubnetMap"
	tablePublicIPMap = "PublicIPMap"
)

//dbCfg controls plugin data base attributes
//these may be overridden by the caller if needed
var dbCfg = struct {
	Name    string
	DataDir string
	DbFile  string
	Timeout time.Duration
}{
	Name:    "ciao cnci agent",
	DataDir: "/var/lib/ciao/networking",
	DbFile:  "docker_plugin.db",
	Timeout: 1 * time.Second,
}

//SubnetMap maintains the list of active subnets across all compute nodes
//that are handled by this CNCI
type SubnetMap struct {
	sync.Mutex
	m map[string]*payloads.TenantAddedEvent //index: Agent IP + Subnet UUID
}

//NewTable creates a new map
func (d *SubnetMap) NewTable() {
	d.m = make(map[string]*payloads.TenantAddedEvent)
}

//Name provides the name of the map
func (d *SubnetMap) Name() string {
	return tableSubnetMap
}

//NewElement allocates and returns an subnet value
func (d *SubnetMap) NewElement() interface{} {
	return &payloads.TenantAddedEvent{}
}

//Add adds a value to the map with the specified key
func (d *SubnetMap) Add(k string, v interface{}) error {
	val, ok := v.(*payloads.TenantAddedEvent)
	if !ok {
		return errors.Errorf("Invalid value type %t", v)
	}
	d.m[k] = val
	return nil
}

//PublicIPMap maintains the list of active Public IP handled by this CNCI
type PublicIPMap struct {
	sync.Mutex
	m map[string]*payloads.PublicIPCommand //index: PublicIP
}

//NewTable creates a new map
func (d *PublicIPMap) NewTable() {
	d.m = make(map[string]*payloads.PublicIPCommand)
}

//Name provides the name of the map
func (d *PublicIPMap) Name() string {
	return tablePublicIPMap
}

//NewElement allocates and returns an subnet value
func (d *PublicIPMap) NewElement() interface{} {
	return &payloads.PublicIPCommand{}
}

//Add adds a value to the map with the specified key
func (d *PublicIPMap) Add(k string, v interface{}) error {
	val, ok := v.(*payloads.PublicIPCommand)
	if !ok {
		return errors.Errorf("Invalid value type %t", v)
	}
	d.m[k] = val
	return nil
}

func dbInit() (*cnciDatabase, error) {
	db := &cnciDatabase{}
	db.DbProvider = database.NewBoltDBProvider()
	db.SubnetMap.m = make(map[string]*payloads.TenantAddedEvent)
	db.PublicIPMap.m = make(map[string]*payloads.PublicIPCommand)

	if err := db.DbInit(dbCfg.DataDir, dbCfg.DbFile); err != nil {
		return nil, errors.Wrapf(err, "db init: %v, %v", dbCfg.DataDir, dbCfg.DbFile)
	}
	if err := db.DbTableRebuild(&db.SubnetMap); err != nil {
		return nil, errors.Wrapf(err, "subnetMap")
	}
	if err := db.DbTableRebuild(&db.PublicIPMap); err != nil {
		return nil, errors.Wrapf(err, "publicIPMap")
	}
	return db, nil
}

//Will implement a simple database API to persist state across
//restarts of the host/VM
//The data base is updated to reflect desired state vs current
//state. This ensures that a restart of the CNCI will result
//in the desired state vs what is present on the current
//CNCI instance. Hence if a particular network command fails
//assuming it passes consistency checks, then on the restart of
//the CNCI the command may succeed.
func dbProcessCommand(db *cnciDatabase, cmd interface{}) error {

	switch netCmd := cmd.(type) {

	case *payloads.EventTenantAdded:

		c := &netCmd.TenantAdded

		db.SubnetMap.Lock()
		defer db.SubnetMap.Unlock()

		key := c.AgentUUID + c.TenantSubnet
		db.SubnetMap.m[key] = c

		if err := db.DbAdd(tableSubnetMap, key, db.SubnetMap.m[key]); err != nil {
			return errors.Wrapf(err, "add tenant to db: %v", c)
		}

	case *payloads.EventTenantRemoved:

		c := &netCmd.TenantRemoved

		db.SubnetMap.Lock()
		defer db.SubnetMap.Unlock()

		key := c.AgentUUID + c.TenantSubnet
		delete(db.SubnetMap.m, key)

		if err := db.DbDelete(tableSubnetMap, key); err != nil {
			return errors.Wrapf(err, "delete tenant from db: %v", c)
		}

	case *payloads.CommandAssignPublicIP:

		c := &netCmd.AssignIP
		db.PublicIPMap.Lock()
		defer db.PublicIPMap.Unlock()

		key := c.PublicIP
		db.PublicIPMap.m[key] = c

		if err := db.DbAdd(tablePublicIPMap, key, db.PublicIPMap.m[key]); err != nil {
			return errors.Wrapf(err, "add Public IP to db: %v", c)
		}

	case *payloads.CommandReleasePublicIP:

		c := &netCmd.ReleaseIP

		db.PublicIPMap.Lock()
		defer db.PublicIPMap.Unlock()

		key := c.PublicIP
		delete(db.PublicIPMap.m, key)

		if err := db.DbDelete(tablePublicIPMap, key); err != nil {
			return errors.Wrapf(err, "delete Public IP from db: %v", c)
		}

	default:
		return errors.Errorf("unknown command: %v", netCmd)

	}

	return nil
}
