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

package libsnnet

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var snTestNet string
var snTestDebug = false

func debugPrint(t *testing.T, args ...interface{}) {
	if snTestDebug {
		t.Log(args)
	}
}

//ScaleCfg is used to setup test parameters for
//testing scaling of network interface creation
//The *Short are used when running go test --short
var ScaleCfg = struct {
	MaxBridgesShort int
	MaxVnicsShort   int
	MaxBridgesLong  int
	MaxVnicsLong    int
}{2, 64, 8, 64}

func snTestInit() {
	snTestNet = os.Getenv("SNNET_ENV")
	if snTestNet == "" {
		snTestNet = "127.0.0.1/24"
	}

	debug := os.Getenv("SNNET_DEBUG")
	if debug != "" && debug != "false" {
		snTestDebug = true
	}
}

func cnTestInit() (*ComputeNode, error) {
	snTestInit()

	_, testNet, err := net.ParseCIDR(snTestNet)
	if err != nil {
		return nil, err
	}

	netConfig := &NetworkConfig{
		ManagementNet: []net.IPNet{*testNet},
		ComputeNet:    []net.IPNet{*testNet},
		Mode:          GreTunnel,
	}

	cn := &ComputeNode{
		ID:            "TestCNUUID",
		NetworkConfig: netConfig,
	}

	err = cn.Init()
	if err != nil {
		return nil, err
	}
	err = cn.ResetNetwork()
	if err != nil {
		return nil, err
	}

	err = cn.DbRebuild(nil)
	if err != nil {
		return nil, err
	}

	return cn, nil

}

//Tests the scaling of the CN VNIC Creation
//
//This tests creates a large number of VNICs across a number
//of subnets
//
//Test should pass OK
func TestCN_Scaling(t *testing.T) {

	assert := assert.New(t)
	cn, err := cnTestInit()
	require.Nil(t, err)

	//From YAML on instance init
	tenantID := "tenantuuid"
	concIP := net.IPv4(192, 168, 111, 1)

	var maxBridges, maxVnics int
	if testing.Short() {
		maxBridges = ScaleCfg.MaxBridgesShort
		maxVnics = ScaleCfg.MaxVnicsShort
	} else {
		maxBridges = ScaleCfg.MaxBridgesLong
		maxVnics = ScaleCfg.MaxVnicsLong
	}

	for s3 := 1; s3 <= maxBridges; s3++ {
		s4 := 0
		_, tenantNet, _ := net.ParseCIDR("192.168." + strconv.Itoa(s3) + "." + strconv.Itoa(s4) + "/24")
		subnetID := "suuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)

		for s4 := 2; s4 <= maxVnics; s4++ {

			vnicIP := net.IPv4(192, 168, byte(s3), byte(s4))
			mac, _ := net.ParseMAC("CA:FE:00:01:02:03")

			vnicID := "vuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			instanceID := "iuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			vnicCfg := &VnicConfig{
				VnicIP:     vnicIP,
				ConcIP:     concIP,
				VnicMAC:    mac,
				Subnet:     *tenantNet,
				SubnetKey:  s3,
				VnicID:     vnicID,
				InstanceID: instanceID,
				SubnetID:   subnetID,
				TenantID:   tenantID,
				ConcID:     "cnciuuid",
			}

			_, _, _, err := cn.CreateVnic(vnicCfg)
			assert.Nil(err)
		}
	}

	for s3 := 1; s3 <= maxBridges; s3++ {
		s4 := 0
		_, tenantNet, _ := net.ParseCIDR("192.168." + strconv.Itoa(s3) + "." + strconv.Itoa(s4) + "/24")
		subnetID := "suuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)

		for s4 := 2; s4 <= maxVnics; s4++ {

			vnicIP := net.IPv4(192, 168, byte(s3), byte(s4))
			mac, _ := net.ParseMAC("CA:FE:00:01:02:03")

			vnicID := "vuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			instanceID := "iuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			vnicCfg := &VnicConfig{
				VnicIP:     vnicIP,
				ConcIP:     concIP,
				VnicMAC:    mac,
				Subnet:     *tenantNet,
				SubnetKey:  0xF,
				VnicID:     vnicID,
				InstanceID: instanceID,
				SubnetID:   subnetID,
				TenantID:   tenantID,
				ConcID:     "cnciuuid",
			}

			_, _, err := cn.DestroyVnic(vnicCfg)
			assert.Nil(err)
		}
	}
}

//Tests the ResetNetwork API
//
//This test creates multiple VNICs belonging to multiple tenants
//It then uses the ResetNetwork API to reset the node's networking
//state to a clean state (as in reset). This test also check that
//the API can be called midway through a node's lifecycle and
//the DbRebuild API can be used to re-construct the node's
//networking state
//
//Test should pass OK
func TestCN_ResetNetwork(t *testing.T) {

	assert := assert.New(t)
	cn, err := cnTestInit()
	require.Nil(t, err)

	_, tenantNet, _ := net.ParseCIDR("192.168.1.0/24")

	//From YAML on instance init
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	vnicCfg := &VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		Subnet:     *tenantNet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	_, ssntpEvent, _, err := cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		assert.NotNil(ssntpEvent)
	}

	vnicCfg.TenantID = "tuuid2"
	vnicCfg.ConcIP = net.IPv4(192, 168, 1, 2)

	_, ssntpEvent, _, err = cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		assert.NotNil(ssntpEvent)
	}

	assert.Nil(cn.ResetNetwork())
	assert.Nil(cn.DbRebuild(nil))

	vnicCfg.TenantID = "tuuid"
	vnicCfg.ConcIP = net.IPv4(192, 168, 1, 1)

	_, ssntpEvent, _, err = cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		assert.NotNil(ssntpEvent)
	}

	vnicCfg.TenantID = "tuuid2"
	vnicCfg.ConcIP = net.IPv4(192, 168, 1, 2)

	_, ssntpEvent, _, err = cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		assert.NotNil(ssntpEvent)
	}

	assert.Nil(cn.ResetNetwork())
}

//Tests multiple VNIC's creation
//
//This tests tests if multiple VNICs belonging to multiple
//tenants can be successfully created and deleted on a given CN
//This tests also checks for the generation of the requisite
//SSNTP message that the launcher is expected to send to the
//CNCI via the scheduler
//
//Test should pass OK
func TestCN_MultiTenant(t *testing.T) {

	cn, err := cnTestInit()
	assert := assert.New(t)
	require.Nil(t, err)

	_, tenantNet, _ := net.ParseCIDR("192.168.1.0/24")

	//From YAML on instance init
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	vnicCfg := &VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		Subnet:     *tenantNet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	_, ssntpEvent, _, err := cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		assert.NotNil(ssntpEvent)
	}

	vnicCfg.TenantID = "tuuid2"
	vnicCfg.ConcIP = net.IPv4(192, 168, 1, 2)

	_, ssntpEvent, _, err = cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		assert.NotNil(ssntpEvent)
	}

	ssntpEvent, _, err = cn.DestroyVnic(vnicCfg)
	if assert.Nil(err) {
		assert.NotNil(ssntpEvent)
	}

	vnicCfg.TenantID = "tuuid"
	vnicCfg.ConcIP = net.IPv4(192, 168, 1, 1)

	ssntpEvent, _, err = cn.DestroyVnic(vnicCfg)
	if assert.Nil(err) {
		assert.NotNil(ssntpEvent)
	}
}

//Negative tests for CN API
//
//This tests for various invalid API invocations
//This test has to be greatly enhanced.
//
//Test is expected to pass
func TestCN_Negative(t *testing.T) {
	assert := assert.New(t)
	cn, err := cnTestInit()
	require.Nil(t, err)

	_, tenantNet, _ := net.ParseCIDR("192.168.1.0/24")

	//From YAML on instance init
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	vnicCfg := &VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		Subnet:     *tenantNet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		//TenantID:   "tuuid", Leaving it blank should fail
		SubnetID: "suuid",
		ConcID:   "cnciuuid",
	}

	_, _, _, err = cn.CreateVnic(vnicCfg)
	assert.NotNil(err)

	//Fix the errors
	vnicCfg.TenantID = "tuuid"

	// Try and create it again.
	var vnicName string
	vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		assert.NotNil(ssntpEvent)
		vnicName = vnic.LinkName
	}

	//Try and create a duplicate. Should work
	vnic, ssntpEvent, _, err = cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		assert.Nil(ssntpEvent)
		assert.Equal(vnicName, vnic.LinkName)
	}

	// Try and destroy
	ssntpEvent, _, err = cn.DestroyVnic(vnicCfg)
	if assert.Nil(err) {
		assert.NotNil(ssntpEvent)

	}
}

//Tests a node can serve as both CN and NN simultaneously
//
//This test checks that at least from the networking point
//of view we can create both Instance VNICs and CNCI VNICs
//on the same node. Even though the launcher does not
//support this mode today, the networking layer allows
//creation and co-existence of both type of VNICs on the
//same node and they will both work
//
//Test should pass OK
func TestCN_AndNN(t *testing.T) {
	assert := assert.New(t)
	cn, err := cnTestInit()
	require.Nil(t, err)

	_, tenantNet, _ := net.ParseCIDR("192.168.1.0/24")

	//From YAML on instance init
	cnciMac, _ := net.ParseMAC("CA:FE:CC:01:02:03")
	cnciVnicCfg := &VnicConfig{
		VnicRole:   DataCenter,
		VnicMAC:    cnciMac,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
	}

	// Create a VNIC
	var cnciVnic1Name string
	cnciVnic, err := cn.CreateCnciVnic(cnciVnicCfg)
	if assert.Nil(err) {
		//Launcher will attach to this name and send out the event
		cnciVnic1Name = cnciVnic.LinkName
	}

	var cnciVnic1DupName string
	// Try and create it again. Should return cached value
	cnciVnic, err = cn.CreateCnciVnic(cnciVnicCfg)
	if assert.Nil(err) {
		cnciVnic1DupName = cnciVnic.LinkName
	}

	assert.Equal(cnciVnic1Name, cnciVnic1DupName)

	//From YAML on instance init
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	vnicCfg := &VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		Subnet:     *tenantNet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	// Create a VNIC: Should create bridge and tunnels
	var vnic1Name, vnic1DupName string
	vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		//We expect a bridge creation event
		assert.NotNil(ssntpEvent)
		//Launcher will attach to this name and send out the event
		vnic1Name = vnic.LinkName
	}

	// Try and create it again. Should return cached value
	vnic, ssntpEvent, _, err = cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		//We do not expect a bridge creation event
		assert.Nil(ssntpEvent)
		//Launcher will attach to this name and send out the event
		vnic1DupName = vnic.LinkName
	}

	assert.Equal(vnic1Name, vnic1DupName)

	mac2, _ := net.ParseMAC("CA:FE:00:01:02:22")
	vnicCfg2 := &VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 2),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac2,
		Subnet:     *tenantNet,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	// Create a second VNIC on the same tenant subnet
	vnic, ssntpEvent, _, err = cn.CreateVnic(vnicCfg2)
	if assert.Nil(err) {
		//We do not expect a bridge creation event
		assert.Nil(ssntpEvent)
	}

	ssntpEvent, _, err = cn.DestroyVnic(vnicCfg2)
	if assert.Nil(err) {
		assert.Nil(ssntpEvent)
	}

	cnciMac2, _ := net.ParseMAC("CA:FE:CC:01:02:22")
	cnciVnicCfg2 := &VnicConfig{
		VnicRole:   DataCenter,
		VnicMAC:    cnciMac2,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid2",
	}

	// Create and destroy a second VNIC
	_, err = cn.CreateCnciVnic(cnciVnicCfg2)
	assert.Nil(err)
	assert.Nil(cn.DestroyCnciVnic(cnciVnicCfg2))

	// Destroy the first VNIC
	assert.Nil(cn.DestroyCnciVnic(cnciVnicCfg))

	// Try and destroy it again - should work
	assert.Nil(cn.DestroyCnciVnic(cnciVnicCfg))

	// Destroy the first VNIC - Deletes the bridge and tunnel
	ssntpEvent, _, err = cn.DestroyVnic(vnicCfg)
	if assert.Nil(err) {
		//We expect a bridge deletion event
		assert.NotNil(ssntpEvent)
	}

	// Try and destroy it again - should work
	ssntpEvent, _, err = cn.DestroyVnic(vnicCfg)
	if assert.Nil(err) {
		assert.Nil(ssntpEvent)
	}
}

//Tests typical sequence of NN APIs
//
//This tests exercises the standard set of APIs that
//the launcher invokes when setting up a NN and creating
//VNICs. It check for duplicate VNIC creation, duplicate
//VNIC deletion
//
//Test is expected to pass
func TestNN_Base(t *testing.T) {
	assert := assert.New(t)
	cn, err := cnTestInit()
	require.Nil(t, err)

	//From YAML on instance init
	cnciMac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	cnciVnicCfg := &VnicConfig{
		VnicRole:   DataCenter,
		VnicMAC:    cnciMac,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
	}

	// Create a VNIC
	var cnciVnic1Name string
	cnciVnic, err := cn.CreateCnciVnic(cnciVnicCfg)
	if assert.Nil(err) {
		cnciVnic1Name = cnciVnic.LinkName
	}

	var cnciVnic1DupName string
	// Try and create it again. Should return cached value
	cnciVnic, err = cn.CreateCnciVnic(cnciVnicCfg)
	if assert.Nil(err) {
		cnciVnic1DupName = cnciVnic.LinkName
	}

	assert.Equal(cnciVnic1Name, cnciVnic1DupName)

	cnciMac2, _ := net.ParseMAC("CA:FE:00:01:02:22")
	cnciVnicCfg2 := &VnicConfig{
		VnicRole:   DataCenter,
		VnicMAC:    cnciMac2,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid2",
	}

	// Create and destroy a second VNIC
	_, err = cn.CreateCnciVnic(cnciVnicCfg2)
	assert.Nil(err)
	assert.Nil(cn.DestroyCnciVnic(cnciVnicCfg2))
	assert.Nil(cn.DestroyCnciVnic(cnciVnicCfg))

	//Destroy again, it should work
	assert.Nil(cn.DestroyCnciVnic(cnciVnicCfg))
}

func validSsntpEvent(ssntpEvent *SsntpEventInfo, cfg *VnicConfig) error {

	if ssntpEvent == nil {
		return fmt.Errorf("SsntpEvent: nil")
	}

	//Note: Checking for non nil values just to ensure the caller called it with all
	//parameters setup properly.
	switch {
	case ssntpEvent.ConcID != cfg.ConcID:
	case ssntpEvent.CnciIP != cfg.ConcIP.String():
	//case ssntpEvent.CnIP != has to be set by the caller
	case ssntpEvent.Subnet != cfg.Subnet.String():
	case ssntpEvent.SubnetKey != cfg.SubnetKey:
	case ssntpEvent.SubnetID != cfg.SubnetID:
	case ssntpEvent.TenantID != cfg.TenantID:
	default:
		return nil
	}
	return fmt.Errorf("SsntpEvent: fields do not match %v != %v", ssntpEvent, cfg)
}

//Tests typical sequence of CN APIs
//
//This tests exercises the standard set of APIs that
//the launcher invokes when setting up a CN and creating
//VNICs. It check for duplicate VNIC creation, duplicate
//VNIC deletion
//
//Test is expected to pass
func TestCN_Base(t *testing.T) {
	assert := assert.New(t)
	cn, err := cnTestInit()
	require.Nil(t, err)

	_, tenantNet, _ := net.ParseCIDR("192.168.1.0/24")

	//From YAML on instance init
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	vnicCfg := &VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 100),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac,
		Subnet:     *tenantNet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	// Create a VNIC: Should create bridge and tunnels
	var vnic1Name, vnic1DupName string
	vnic, ssntpEvent, _, err := cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		//We expect a bridge creation event
		if assert.NotNil(ssntpEvent) {
			//Check the fields of the ssntpEvent
			err := validSsntpEvent(ssntpEvent, vnicCfg)
			assert.Nil(err)
			assert.Equal(ssntpEvent.Event, SsntpTunAdd)
		}
		vnic1Name = vnic.LinkName
	}

	// Try and create it again. Should return cached value
	vnic, ssntpEvent, _, err = cn.CreateVnic(vnicCfg)
	if assert.Nil(err) {
		assert.Nil(ssntpEvent)
		vnic1DupName = vnic.LinkName
	}

	assert.Equal(vnic1Name, vnic1DupName)

	mac2, _ := net.ParseMAC("CA:FE:00:01:02:22")
	vnicCfg2 := &VnicConfig{
		VnicIP:     net.IPv4(192, 168, 1, 2),
		ConcIP:     net.IPv4(192, 168, 1, 1),
		VnicMAC:    mac2,
		Subnet:     *tenantNet,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	// Create a second VNIC on the same tenant subnet
	vnic, ssntpEvent, _, err = cn.CreateVnic(vnicCfg2)
	if assert.Nil(err) {
		//No bridge creation event expected
		assert.Nil(ssntpEvent)
	}

	ssntpEvent, _, err = cn.DestroyVnic(vnicCfg2)
	if assert.Nil(err) {
		//No bridge creation event expected
		assert.Nil(ssntpEvent)
	}

	// Destroy the first VNIC - Deletes the bridge and tunnel
	ssntpEvent, _, err = cn.DestroyVnic(vnicCfg)
	if assert.Nil(err) {
		//No bridge creation event expected
		if assert.NotNil(ssntpEvent) {
			//Check the fields of the ssntpEvent
			err := validSsntpEvent(ssntpEvent, vnicCfg)
			assert.Nil(err)
			assert.Equal(ssntpEvent.Event, SsntpTunDel)
		}
	}

	// Try and destroy it again - should work
	ssntpEvent, _, err = cn.DestroyVnic(vnicCfg)
	if assert.Nil(err) {
		//No bridge deletion event expected
		assert.Nil(ssntpEvent)
	}
}

//Whitebox test the CN API
//
//This tests exercises tests the primitive operations
//that the CN API rely on. This is used to check any
//issues with the underlying netlink library or kernel
//This tests fails typically if the kernel or netlink
//implementation changes
//
//Test is expected to pass
func TestCN_Whitebox(t *testing.T) {
	assert := assert.New(t)
	var instanceMAC net.HardwareAddr
	var err error

	// Typical inputs in YAML from Controller
	tenantUUID := "tenantUuid"
	instanceUUID := "tenantUuid"
	subnetUUID := "subnetUuid"
	subnetKey := uint32(0xF)
	concUUID := "concUuid"
	//The IP corresponding to CNCI, maybe we can use DNS resolution here?
	concIP := net.IPv4(192, 168, 1, 1)
	//The IP corresponding to the VNIC that carries tenant traffic
	cnIP := net.IPv4(127, 0, 0, 1)
	instanceMAC, err = net.ParseMAC("CA:FE:00:01:02:03")
	assert.Nil(err)

	// Create the CN tenant bridge only if it does not exist
	bridgeAlias := fmt.Sprintf("br_%s_%s_%s", tenantUUID, subnetUUID, concUUID)
	bridge, _ := NewBridge(bridgeAlias)

	if assert.NotNil(bridge.GetDevice()) {
		// First instance to land, create the bridge and tunnel
		assert.Nil(bridge.Create())
		defer func() { _ = bridge.Destroy() }()

		// Create the tunnel to connect to the CNCI
		local := cnIP
		remote := concIP

		greAlias := fmt.Sprintf("gre_%s_%s_%s", tenantUUID, subnetUUID, concUUID)
		gre, _ := newGreTunEP(greAlias, local, remote, subnetKey)

		assert.Nil(gre.create())
		defer func() { _ = gre.destroy() }()

		assert.Nil(gre.attach(bridge))
		assert.Nil(gre.enable())
		assert.Nil(bridge.Enable())
	}

	// Create the VNIC for the instance
	vnicAlias := fmt.Sprintf("vnic_%s_%s_%s_%s", tenantUUID, instanceUUID, instanceMAC, concUUID)
	vnic, _ := NewVnic(vnicAlias)
	vnic.MACAddr = &instanceMAC

	assert.Nil(vnic.Create())
	defer func() { _ = vnic.Destroy() }()

	assert.Nil(vnic.Attach(bridge))
	assert.Nil(vnic.Enable())
	assert.Nil(bridge.Enable())
}
