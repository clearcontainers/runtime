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
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func linkDump(t *testing.T) error {
	out, err := exec.Command("ip", "-d", "link").CombinedOutput()

	if err != nil {
		return err
	}

	debugPrint(t, "dumping link info \n", string(out))

	return nil
}

func dockerRestart() error {
	_, err := exec.Command("service", "docker", "restart").CombinedOutput()
	if err != nil {
		_, err = exec.Command("systemctl", "restart", "docker").CombinedOutput()
		return err
	}
	return err
}

//Will be replaced by Docker API's in launcher
//docker run -it --net=<subnet.Name> --ip=<instance.IP> --mac-address=<instance.MacAddresss>
//debian ip addr show eth0 scope global
func dockerRunVerify(name string, ip net.IP, mac net.HardwareAddr, subnetID string) error {
	cmd := exec.Command("docker", "run", "--name", ip.String(), "--net="+subnetID,
		"--ip="+ip.String(), "--mac-address="+mac.String(),
		"debian", "ip", "addr", "show", "eth0", "scope", "global")
	out, err := cmd.CombinedOutput()

	if err != nil {
		return err
	}

	if !strings.Contains(string(out), ip.String()) {
		return fmt.Errorf("docker container IP not setup %s", ip.String())
	}
	if !strings.Contains(string(out), mac.String()) {
		return fmt.Errorf("docker container MAC not setup %s", mac.String())
	}
	if !strings.Contains(string(out), "mtu 1400") {
		return fmt.Errorf("docker container MTU not setup")
	}

	if err := dockerContainerInfo(name); err != nil {
		return fmt.Errorf("docker container inspect failed %s %v", name, err.Error())
	}
	return nil
}

func dockerContainerDelete(name string) error {
	_, err := exec.Command("docker", "stop", name).CombinedOutput()
	if err != nil {
		return err
	}

	_, err = exec.Command("docker", "rm", name).CombinedOutput()
	return err
}

func dockerContainerInfo(name string) error {
	_, err := exec.Command("docker", "ps", "-a").CombinedOutput()
	if err != nil {
		return err
	}

	_, err = exec.Command("docker", "inspect", name).CombinedOutput()
	return err
}

//Will be replaced by Docker API's in launcher
// docker network create -d=ciao [--ipam-driver=ciao]
// --subnet=<ContainerInfo.Subnet> --gateway=<ContainerInfo.Gate
// --opt "bridge"=<ContainerInfo.Bridge> ContainerInfo.SubnetID
//The IPAM driver needs top of the tree Docker (which needs special build)
//is not tested yet
func dockerNetCreate(subnet net.IPNet, gw net.IP, bridge string, subnetID string) error {
	cmd := exec.Command("docker", "network", "create", "-d=ciao", "--ipam-driver=ciao",
		"--subnet="+subnet.String(), "--gateway="+gw.String(),
		"--opt", "bridge="+bridge, subnetID)

	_, err := cmd.CombinedOutput()
	return err
}

//Will be replaced by Docker API's in launcher
// docker network rm ContainerInfo.SubnetID
func dockerNetDelete(subnetID string) error {
	_, err := exec.Command("docker", "network", "rm", subnetID).CombinedOutput()
	return err
}
func dockerNetList() error {
	_, err := exec.Command("docker", "network", "ls").CombinedOutput()
	return err
}

func dockerNetInfo(subnetID string) error {
	_, err := exec.Command("docker", "network", "inspect", subnetID).CombinedOutput()
	return err
}

func dockerRunTop(name string, ip net.IP, mac net.HardwareAddr, subnetID string) {
	cmd := exec.Command("docker", "run", "--name", ip.String(), "--net="+subnetID,
		"--ip="+ip.String(), "--mac-address="+mac.String(),
		"debian", "top", "-b", "-d1")
	go func() { _ = cmd.Run() }() // Ensures that the containers stays alive. Kludgy
	return
}

func dockerRunPingVerify(name string, ip net.IP, mac net.HardwareAddr, subnetID string, addr string) error {
	cmd := exec.Command("docker", "run", "--name", ip.String(), "--net="+subnetID,
		"--ip="+ip.String(), "--mac-address="+mac.String(),
		"debian", "ping", "-c", "1", "192.168.111.100")
	out, err := cmd.CombinedOutput()

	if err != nil {
		return err
	}

	if !strings.Contains(string(out), "1 packets received") {
		return fmt.Errorf("docker connectivity test failed %s", ip.String())
	}
	return nil
}

func startDockerPlugin(t *testing.T) (*DockerPlugin, error) {

	dockerPlugin := NewDockerPlugin()
	if err := dockerPlugin.Init(); err != nil {
		t.Error(err)
		return nil, err
	}

	if err := dockerPlugin.Start(); err != nil {
		t.Error(err)
		return nil, err
	}

	//Restarting docker here so the plugin will
	//be picked up without modifying the boot scripts
	if err := dockerRestart(); err != nil {
		t.Error(err)
		return nil, err
	}
	return dockerPlugin, nil
}

func stopDockerPlugin(dockerPlugin *DockerPlugin) error {
	if err := dockerPlugin.Stop(); err != nil {
		return err
	}
	if err := dockerPlugin.Close(); err != nil {
		return err
	}
	return nil
}

//Tests typical sequence of CN Container APIs
//
//This tests exercises the standard set of APIs that
//the launcher invokes when setting up a CN and creating
//Container VNICs.
//
//Test is expected to pass
func TestCNContainer_Base(t *testing.T) {
	assert := assert.New(t)

	cn, err := cnTestInit()
	require.Nil(t, err)

	dockerPlugin, err := startDockerPlugin(t)
	require.Nil(t, err)

	//From YAML on instance init
	//Two VNICs on the same tenant subnet
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	mac2, _ := net.ParseMAC("CA:FE:00:02:02:03")
	_, tnet, _ := net.ParseCIDR("192.168.111.0/24")
	tip := net.ParseIP("192.168.111.100")
	tip2 := net.ParseIP("192.168.111.102")
	cip := net.ParseIP("192.168.200.200")

	vnicCfg := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip,
		ConcIP:     cip,
		VnicMAC:    mac,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg2 := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip2,
		ConcIP:     cip,
		VnicMAC:    mac2,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	var subnetID, iface string //Used to check that they match

	// Create a VNIC: Should create bridge and tunnels
	if vnic, ssntpEvent, cInfo, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error(err)
	} else {
		// expected SSNTP Event
		if assert.NotNil(ssntpEvent) {
			assert.Equal(ssntpEvent.Event, SsntpTunAdd)
		}
		// expected Container Event
		if assert.NotNil(cInfo) {
			assert.Equal(cInfo.CNContainerEvent, ContainerNetworkAdd)
			assert.NotEqual(cInfo.SubnetID, "")
			assert.NotEqual(cInfo.Subnet.String(), "")
			assert.NotEqual(cInfo.Gateway.String(), "")
			assert.NotEqual(cInfo.Bridge, "")
		}
		assert.Nil(validSsntpEvent(ssntpEvent, vnicCfg))

		//Cache the first subnet ID we see. All subsequent should have the same
		subnetID = cInfo.SubnetID
		iface = vnic.InterfaceName()
		assert.NotEqual(iface, "")

		//Launcher will attach to this name and send out the event
		//Launcher will also create the logical docker network
		debugPrint(t, "VNIC created =", vnic.LinkName, ssntpEvent, cInfo)
		assert.Nil(linkDump(t))

		//Now kick off the docker commands
		assert.Nil(dockerNetCreate(cInfo.Subnet, cInfo.Gateway, cInfo.Bridge, cInfo.SubnetID))
		assert.Nil(dockerNetInfo(cInfo.SubnetID))
		assert.Nil(dockerRunVerify(vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID))
		assert.Nil(dockerContainerDelete(vnicCfg.VnicIP.String()))
	}

	//Duplicate VNIC creation
	if vnic, ssntpEvent, cInfo, err := cn.CreateVnic(vnicCfg); err != nil {
		t.Error(err)
	} else {
		assert.Nil(ssntpEvent, "ERROR: DUP unexpected event")
		if assert.NotNil(cInfo) {
			assert.Equal(cInfo.SubnetID, subnetID)
			assert.Equal(cInfo.CNContainerEvent, ContainerNetworkInfo)
			assert.Equal(iface, vnic.InterfaceName())
		}
	}

	//Second VNIC creation - Should succeed
	if vnic, ssntpEvent, cInfo, err := cn.CreateVnic(vnicCfg2); err != nil {
		t.Error(err)
	} else {
		assert.Nil(ssntpEvent)
		if assert.NotNil(cInfo) {
			assert.Equal(cInfo.SubnetID, subnetID)
			assert.Equal(cInfo.CNContainerEvent, ContainerNetworkInfo)
		}
		iface = vnic.InterfaceName()
		assert.NotEqual(iface, "")
		assert.Nil(dockerRunVerify(vnicCfg2.VnicIP.String(), vnicCfg2.VnicIP,
			vnicCfg2.VnicMAC, cInfo.SubnetID))
		assert.Nil(dockerContainerDelete(vnicCfg2.VnicIP.String()))
	}

	//Duplicate VNIC creation
	if vnic, ssntpEvent, cInfo, err := cn.CreateVnic(vnicCfg2); err != nil {
		t.Error(err)
	} else {
		assert.Nil(ssntpEvent)
		if assert.NotNil(cInfo) {
			assert.Equal(cInfo.SubnetID, subnetID)
			assert.Equal(cInfo.CNContainerEvent, ContainerNetworkInfo)
			assert.Equal(iface, vnic.InterfaceName())
		}
	}

	//Destroy the first one
	if ssntpEvent, cInfo, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error(err)
	} else {
		assert.Nil(ssntpEvent)
		assert.Nil(cInfo)
	}

	//Destroy it again
	if ssntpEvent, cInfo, err := cn.DestroyVnic(vnicCfg); err != nil {
		t.Error(err)
	} else {
		assert.Nil(ssntpEvent)
		assert.Nil(cInfo)
	}

	// Try and destroy - should work - cInfo should be reported
	if ssntpEvent, cInfo, err := cn.DestroyVnic(vnicCfg2); err != nil {
		t.Error(err)
	} else {
		assert.NotNil(ssntpEvent)
		if assert.NotNil(cInfo) {
			assert.Equal(cInfo.SubnetID, subnetID)
			assert.Equal(cInfo.CNContainerEvent, ContainerNetworkDel)
		}
	}

	//Has to be called after the VNIC has been deleted
	assert.Nil(dockerNetDelete(subnetID))
	assert.Nil(dockerNetList())

	//Destroy it again
	if ssntpEvent, cInfo, err := cn.DestroyVnic(vnicCfg2); err != nil {
		t.Error(err)
	} else {
		assert.Nil(ssntpEvent)
		assert.Nil(cInfo)
	}

	assert.Nil(stopDockerPlugin(dockerPlugin))
}

//Tests connectivity between two node local Containers
//
//Tests connectivity between two node local Containers
//using ping between a long running container and
//container that does ping
//
//Test is expected to pass
func TestCNContainer_Connectivity(t *testing.T) {
	assert := assert.New(t)
	cn, err := cnTestInit()
	require.Nil(t, err)

	dockerPlugin, err := startDockerPlugin(t)
	require.Nil(t, err)

	//From YAML on instance init
	//Two VNICs on the same tenant subnet
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	mac2, _ := net.ParseMAC("CA:FE:00:02:02:03")
	_, tnet, _ := net.ParseCIDR("192.168.111.0/24")
	tip := net.ParseIP("192.168.111.100")
	tip2 := net.ParseIP("192.168.111.102")
	cip := net.ParseIP("192.168.200.200")

	vnicCfg := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip,
		ConcIP:     cip,
		VnicMAC:    mac,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg2 := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip2,
		ConcIP:     cip,
		VnicMAC:    mac2,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	_, _, cInfo, err := cn.CreateVnic(vnicCfg)
	assert.Nil(err)

	assert.Nil(dockerNetCreate(cInfo.Subnet, cInfo.Gateway, cInfo.Bridge, cInfo.SubnetID))

	//Kick off a long running container
	dockerRunTop(vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID)

	_, _, cInfo2, err := cn.CreateVnic(vnicCfg2)
	assert.Nil(err)

	assert.Nil(dockerRunPingVerify(vnicCfg2.VnicIP.String(), vnicCfg2.VnicIP,
		vnicCfg2.VnicMAC, cInfo2.SubnetID, vnicCfg.VnicIP.String()))

	//Destroy the containers
	assert.Nil(dockerContainerDelete(vnicCfg.VnicIP.String()))
	assert.Nil(dockerContainerDelete(vnicCfg2.VnicIP.String()))

	//Destroy the VNICs
	_, _, err = cn.DestroyVnic(vnicCfg)
	assert.Nil(err)
	_, _, err = cn.DestroyVnic(vnicCfg2)
	assert.Nil(err)

	//Destroy the network, has to be called after the VNIC has been deleted
	assert.Nil(dockerNetDelete(cInfo.SubnetID))
	assert.Nil(stopDockerPlugin(dockerPlugin))
}

//Tests VM and Container VNIC Interop
//
//Tests that VM and Container VNICs can co-exist
//by created VM and Container VNICs in different orders and in each case
//tests that the Network Connectivity is functional
//
//Test is expected to pass
func TestCNContainer_Interop1(t *testing.T) {
	assert := assert.New(t)

	cn, err := cnTestInit()
	require.Nil(t, err)

	dockerPlugin, err := startDockerPlugin(t)
	require.Nil(t, err)

	//From YAML on instance init
	//Two VNICs on the same tenant subnet
	_, tnet, _ := net.ParseCIDR("192.168.111.0/24")
	tip := net.ParseIP("192.168.111.100")
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	cip := net.ParseIP("192.168.200.200")
	mac2, _ := net.ParseMAC("CA:FE:00:02:02:03")
	tip2 := net.ParseIP("192.168.111.102")
	mac3, _ := net.ParseMAC("CA:FE:00:03:02:03")
	tip3 := net.ParseIP("192.168.111.103")
	mac4, _ := net.ParseMAC("CA:FE:00:04:02:03")
	tip4 := net.ParseIP("192.168.111.104")

	vnicCfg := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip,
		ConcIP:     cip,
		VnicMAC:    mac,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg2 := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip2,
		ConcIP:     cip,
		VnicMAC:    mac2,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg3 := &VnicConfig{
		VnicRole:   TenantVM,
		VnicIP:     tip3,
		ConcIP:     cip,
		VnicMAC:    mac3,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid3",
		InstanceID: "iuuid3",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg4 := &VnicConfig{
		VnicRole:   TenantVM,
		VnicIP:     tip4,
		ConcIP:     cip,
		VnicMAC:    mac4,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid4",
		InstanceID: "iuuid4",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	_, _, cInfo, err := cn.CreateVnic(vnicCfg)
	assert.Nil(err)

	err = dockerNetCreate(cInfo.Subnet, cInfo.Gateway, cInfo.Bridge, cInfo.SubnetID)
	assert.Nil(err)

	_, _, _, err = cn.CreateVnic(vnicCfg3)
	assert.Nil(err)

	//Kick off a long running container
	dockerRunTop(vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID)

	_, _, cInfo2, err := cn.CreateVnic(vnicCfg2)
	assert.Nil(err)

	_, _, _, err = cn.CreateVnic(vnicCfg4)
	assert.Nil(err)

	assert.Nil(dockerRunPingVerify(vnicCfg2.VnicIP.String(), vnicCfg2.VnicIP,
		vnicCfg2.VnicMAC, cInfo2.SubnetID, vnicCfg.VnicIP.String()))

	//Destroy the containers
	assert.Nil(dockerContainerDelete(vnicCfg.VnicIP.String()))
	assert.Nil(dockerContainerDelete(vnicCfg2.VnicIP.String()))

	_, _, err = cn.DestroyVnic(vnicCfg)
	assert.Nil(err)

	_, _, err = cn.DestroyVnic(vnicCfg2)
	assert.Nil(err)

	_, _, err = cn.DestroyVnic(vnicCfg3)
	assert.Nil(err)

	err = dockerNetDelete(cInfo.SubnetID)
	assert.Nil(err)

	_, _, err = cn.DestroyVnic(vnicCfg4)
	assert.Nil(err)

	assert.Nil(stopDockerPlugin(dockerPlugin))
}

//Tests VM and Container VNIC Interop
//
//Tests that VM and Container VNICs can co-exist
//by created VM and Container VNICs in different orders and in each case
//tests that the Network Connectivity is functional
//
//Test is expected to pass
func TestCNContainer_Interop2(t *testing.T) {
	assert := assert.New(t)

	cn, err := cnTestInit()
	require.Nil(t, err)

	dockerPlugin, err := startDockerPlugin(t)
	require.Nil(t, err)

	//From YAML on instance init
	//Two VNICs on the same tenant subnet
	_, tnet, _ := net.ParseCIDR("192.168.111.0/24")
	tip := net.ParseIP("192.168.111.100")
	mac, _ := net.ParseMAC("CA:FE:00:01:02:03")
	cip := net.ParseIP("192.168.200.200")
	mac2, _ := net.ParseMAC("CA:FE:00:02:02:03")
	tip2 := net.ParseIP("192.168.111.102")
	mac3, _ := net.ParseMAC("CA:FE:00:03:02:03")
	tip3 := net.ParseIP("192.168.111.103")
	mac4, _ := net.ParseMAC("CA:FE:00:04:02:03")
	tip4 := net.ParseIP("192.168.111.104")

	vnicCfg := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip,
		ConcIP:     cip,
		VnicMAC:    mac,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid",
		InstanceID: "iuuid",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg2 := &VnicConfig{
		VnicRole:   TenantContainer,
		VnicIP:     tip2,
		ConcIP:     cip,
		VnicMAC:    mac2,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid2",
		InstanceID: "iuuid2",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg3 := &VnicConfig{
		VnicRole:   TenantVM,
		VnicIP:     tip3,
		ConcIP:     cip,
		VnicMAC:    mac3,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid3",
		InstanceID: "iuuid3",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	vnicCfg4 := &VnicConfig{
		VnicRole:   TenantVM,
		VnicIP:     tip4,
		ConcIP:     cip,
		VnicMAC:    mac4,
		Subnet:     *tnet,
		SubnetKey:  0xF,
		VnicID:     "vuuid4",
		InstanceID: "iuuid4",
		TenantID:   "tuuid",
		SubnetID:   "suuid",
		ConcID:     "cnciuuid",
	}

	_, _, _, err = cn.CreateVnic(vnicCfg3)
	assert.Nil(err)

	_, _, cInfo, err := cn.CreateVnic(vnicCfg)
	assert.Nil(err)

	assert.Nil(dockerNetCreate(cInfo.Subnet, cInfo.Gateway, cInfo.Bridge, cInfo.SubnetID))

	//Kick off a long running container
	dockerRunTop(vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID)

	_, _, cInfo2, err := cn.CreateVnic(vnicCfg2)
	assert.Nil(err)

	assert.Nil(dockerRunPingVerify(vnicCfg2.VnicIP.String(), vnicCfg2.VnicIP,
		vnicCfg2.VnicMAC, cInfo2.SubnetID, vnicCfg.VnicIP.String()))

	//Destroy the containers
	assert.Nil(dockerContainerDelete(vnicCfg.VnicIP.String()))
	assert.Nil(dockerContainerDelete(vnicCfg2.VnicIP.String()))

	_, _, _, err = cn.CreateVnic(vnicCfg4)
	assert.Nil(err)

	//Destroy the VNICs
	_, _, err = cn.DestroyVnic(vnicCfg)
	assert.Nil(err)

	_, _, err = cn.DestroyVnic(vnicCfg2)
	assert.Nil(err)

	//Destroy the network, has to be called after the VNIC has been deleted
	assert.Nil(dockerNetDelete(cInfo.SubnetID))

	_, _, err = cn.DestroyVnic(vnicCfg4)
	assert.Nil(err)

	_, _, err = cn.DestroyVnic(vnicCfg3)
	assert.Nil(err)

	assert.Nil(stopDockerPlugin(dockerPlugin))
}
