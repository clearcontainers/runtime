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

package parallel

import (
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/01org/ciao/clogger/gloginterface"
	"github.com/01org/ciao/networking/libsnnet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cnNetEnv string
var cnParallel = true

//Controls the number of go routines that concurrently invoke Network APIs
//This checks that the internal throttling is working
var cnMaxOutstanding = 128

var scaleCfg = struct {
	maxBridgesShort int
	maxVnicsShort   int
	maxBridgesLong  int
	maxVnicsLong    int
}{2, 8, 200, 32}

const (
	allRoles = libsnnet.TenantContainer + libsnnet.TenantVM
)

func cninit() {
	cnNetEnv = os.Getenv("SNNET_ENV")

	if cnNetEnv == "" {
		cnNetEnv = "127.0.0.1/24"
	}

	if cnParallel {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(1)
	}
}

func logTime(t *testing.T, start time.Time, fn string) {
	elapsedTime := time.Since(start)
	t.Logf("function %s took %s", fn, elapsedTime)
}

func pickRandomRole(input int) libsnnet.VnicRole {
	if input%2 == 1 {
		return libsnnet.TenantContainer
	}
	return libsnnet.TenantVM
}

func CNAPIParallel(t *testing.T, role libsnnet.VnicRole, modelCancel bool) {
	assert := assert.New(t)

	var sem = make(chan int, cnMaxOutstanding)

	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	cninit()

	_, mnet, _ := net.ParseCIDR(cnNetEnv)

	//From YAML, on agent init
	mgtNet := []net.IPNet{*mnet}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	require.Nil(t, cn.Init())
	require.Nil(t, cn.ResetNetwork())
	require.Nil(t, cn.DbRebuild(nil))

	//From YAML on instance init
	tenantID := "tenantuuid"
	concIP := net.IPv4(192, 168, 254, 1)

	var maxBridges, maxVnics int
	if testing.Short() {
		maxBridges = scaleCfg.maxBridgesShort
		maxVnics = scaleCfg.maxVnicsShort
	} else {
		maxBridges = scaleCfg.maxBridgesLong
		maxVnics = scaleCfg.maxVnicsLong
	}

	channelSize := maxBridges*maxVnics + 1
	createCh := make(chan *libsnnet.VnicConfig, channelSize)
	destroyCh := make(chan *libsnnet.VnicConfig, channelSize)
	cancelCh := make(chan chan interface{}, channelSize)

	t.Log("Priming interfaces")

	for s3 := 1; s3 <= maxBridges; s3++ {
		s4 := 0
		_, tenantNet, _ := net.ParseCIDR("192.168." + strconv.Itoa(s3) + "." + strconv.Itoa(s4) + "/24")
		subnetID := "suuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)

		for s4 := 2; s4 <= maxVnics; s4++ {

			vnicIP := net.IPv4(192, 168, byte(s3), byte(s4))
			mac, _ := net.ParseMAC("CA:FE:00:01:02:03")

			vnicID := "vuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			instanceID := "iuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)

			role := role
			if role == allRoles {
				role = pickRandomRole(s4)
			}

			vnicCfg := &libsnnet.VnicConfig{
				VnicRole:   role,
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

			if modelCancel {
				vnicCfg.CancelChan = make(chan interface{})
			}
			createCh <- vnicCfg
			destroyCh <- vnicCfg
			cancelCh <- vnicCfg.CancelChan
		}
	}

	close(createCh)
	close(destroyCh)
	close(cancelCh)

	var wg sync.WaitGroup
	wg.Add(len(createCh))

	if modelCancel {
		for c := range cancelCh {
			go func(c chan interface{}) {
				time.Sleep(100 * time.Millisecond)
				close(c)
			}(c)
		}
	}

	for vnicCfg := range createCh {
		sem <- 1
		go func(vnicCfg *libsnnet.VnicConfig) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			if !assert.NotNil(vnicCfg) {
				return
			}
			defer logTime(t, time.Now(), "Create VNIC")

			_, _, _, err := cn.CreateVnic(vnicCfg)
			if !modelCancel {
				//We expect failures only when we have cancellations
				assert.Nil(err)
			}
		}(vnicCfg)
	}

	wg.Wait()

	wg.Add(len(destroyCh))
	for vnicCfg := range destroyCh {
		sem <- 1
		go func(vnicCfg *libsnnet.VnicConfig) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			if !assert.NotNil(vnicCfg) {
				return
			}
			defer logTime(t, time.Now(), "Destroy VNIC")
			_, _, err := cn.DestroyVnic(vnicCfg)
			if !modelCancel {
				//We expect failures only when we have cancellations
				assert.Nil(err)
			}
		}(vnicCfg)
	}

	wg.Wait()
}

func TestCNContainer_Parallel(t *testing.T) {
	CNAPIParallel(t, libsnnet.TenantContainer, false)
}

func TestCNVM_Parallel(t *testing.T) {
	CNAPIParallel(t, libsnnet.TenantVM, false)
}

func TestCNVMContainer_Parallel(t *testing.T) {
	CNAPIParallel(t, libsnnet.TenantContainer+libsnnet.TenantVM, false)
}

func TestCNVMContainer_Cancel(t *testing.T) {
	CNAPIParallel(t, libsnnet.TenantContainer+libsnnet.TenantVM, true)
}

//Docker Testing
//TODO: Place all docker utility functions in a single file
func dockerRestart(t *testing.T) error {
	_, err := exec.Command("service", "docker", "restart").CombinedOutput()
	if err != nil {
		out, err := exec.Command("systemctl", "restart", "docker").CombinedOutput()
		if err != nil {
			t.Error("docker restart", string(out), err)
		}
	}
	return err
}

//Will be replaced by Docker API's in launcher
//docker run -it --net=none debian ip addr show lo
func dockerRunNetNone(t *testing.T, name string, ip net.IP, mac net.HardwareAddr, subnetID string) error {
	assert := assert.New(t)
	defer logTime(t, time.Now(), "dockerRunNetNone")

	cmd := exec.Command("docker", "run", "--name", ip.String(), "--net=none",
		"debian", "ip", "addr", "show", "lo")
	out, err := cmd.CombinedOutput()

	assert.Nil(err, out)
	err = dockerContainerInfo(t, name)
	assert.Nil(err)

	return err
}

//Will be replaced by Docker API's in launcher
//docker run -it debian ip addr show lo
func dockerRunNetDocker(t *testing.T, name string, ip net.IP, mac net.HardwareAddr, subnetID string) error {
	assert := assert.New(t)
	defer logTime(t, time.Now(), "dockerRunNetDocker")

	cmd := exec.Command("docker", "run", "--name", ip.String(),
		"debian", "ip", "addr", "show", "lo")
	out, err := cmd.CombinedOutput()

	assert.Nil(err, string(out))
	err = dockerContainerInfo(t, name)
	assert.Nil(err)

	return err
}

//Will be replaced by Docker API's in launcher
//docker run -it --net=<subnet.Name> --ip=<instance.IP> --mac-address=<instance.MacAddresss>
//debian ip addr show eth0 scope global
func dockerRunVerify(t *testing.T, name string, ip net.IP, mac net.HardwareAddr, subnetID string) error {
	assert := assert.New(t)
	defer logTime(t, time.Now(), "dockerRunVerify")

	cmd := exec.Command("docker", "run", "--name", ip.String(), "--net="+subnetID,
		"--ip="+ip.String(), "--mac-address="+mac.String(),
		"debian", "ip", "addr", "show", "eth0", "scope", "global")
	out, err := cmd.CombinedOutput()
	assert.Nil(err)

	if !strings.Contains(string(out), ip.String()) {
		t.Error("docker container IP not setup ", ip.String())
	}
	if !strings.Contains(string(out), mac.String()) {
		t.Error("docker container MAC not setup ", mac.String())
	}
	if !strings.Contains(string(out), "mtu 1400") {
		t.Error("docker container MTU not setup ")
	}

	err = dockerContainerInfo(t, name)
	assert.Nil(err)
	return err
}

func dockerContainerDelete(t *testing.T, name string) error {
	assert := assert.New(t)
	defer logTime(t, time.Now(), "dockerContainerDelete")
	_ = exec.Command("docker", "stop", name).Run()
	out, err := exec.Command("docker", "rm", name).CombinedOutput()
	assert.Nil(err, string(out))
	return err
}

func dockerContainerInfo(t *testing.T, name string) error {
	assert := assert.New(t)
	defer logTime(t, time.Now(), "dockerContainerInfo")
	_, err := exec.Command("docker", "ps", "-a").CombinedOutput()
	assert.Nil(err)

	out, err := exec.Command("docker", "inspect", name).CombinedOutput()
	assert.Nil(err, string(out))
	return err
}

//Will be replaced by Docker API's in launcher
// docker network create -d=ciao [--ipam-driver=ciao]
// --subnet=<ContainerInfo.Subnet> --gateway=<ContainerInfo.Gate
// --opt "bridge"=<ContainerInfo.Bridge> ContainerInfo.SubnetID
//The IPAM driver needs top of the tree Docker (which needs special build)
//is not tested yet
func dockerNetCreate(t *testing.T, subnet net.IPNet, gw net.IP, bridge string, subnetID string) error {
	assert := assert.New(t)
	defer logTime(t, time.Now(), "dockerNetCreate")
	cmd := exec.Command("docker", "network", "create", "-d=ciao",
		"--subnet="+subnet.String(), "--gateway="+gw.String(),
		"--opt", "bridge="+bridge, subnetID)

	out, err := cmd.CombinedOutput()
	assert.Nil(err, string(out))
	return err
}

//Will be replaced by Docker API's in launcher
// docker network rm ContainerInfo.SubnetID
func dockerNetDelete(t *testing.T, subnetID string) error {
	assert := assert.New(t)
	defer logTime(t, time.Now(), "dockerNetDelete")
	out, err := exec.Command("docker", "network", "rm", subnetID).CombinedOutput()
	assert.Nil(err, string(out))
	return err
}

func dockerNetList(t *testing.T) error {
	assert := assert.New(t)
	defer logTime(t, time.Now(), "dockerNetList")
	out, err := exec.Command("docker", "network", "ls").CombinedOutput()
	assert.Nil(err, string(out))
	return err
}

func dockerNetInfo(t *testing.T, subnetID string) error {
	assert := assert.New(t)
	defer logTime(t, time.Now(), "dockerNetInfo")
	out, err := exec.Command("docker", "network", "inspect", subnetID).CombinedOutput()
	assert.Nil(err, string(out))
	return err
}

type dockerNetType int

const (
	netCiao dockerNetType = iota
	netDockerNone
	netDockerDefault
)

//Tests launch of Docker containers at scale (serially)
//
//This tests exercises attempts to launch large
//numbers of docker containers at scale to isolate
//any issues with plugin responsiveness
//
//Test is expected to pass
func DockerSerial(netType dockerNetType, t *testing.T) {
	assert := assert.New(t)
	defer logTime(t, time.Now(), "TestDockerSerial")
	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	cninit()
	_, mnet, _ := net.ParseCIDR(cnNetEnv)

	//From YAML, on agent init
	mgtNet := []net.IPNet{*mnet}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	require.Nil(t, cn.Init())
	require.Nil(t, cn.ResetNetwork())
	require.Nil(t, cn.DbRebuild(nil))

	dockerPlugin := libsnnet.NewDockerPlugin()
	require.Nil(t, dockerPlugin.Init())

	defer func() { _ = dockerPlugin.Close() }()

	require.Nil(t, dockerPlugin.Start())
	defer func() { _ = dockerPlugin.Stop() }()

	//Restarting docker here so the the plugin will
	//be picked up without modifing the boot scripts
	require.Nil(t, dockerRestart(t))

	//From YAML on instance init
	tenantID := "tenantuuid"
	concIP := net.IPv4(192, 168, 254, 1)

	var maxBridges, maxVnics int
	if testing.Short() {
		maxBridges = scaleCfg.maxBridgesShort
		maxVnics = scaleCfg.maxVnicsShort
	} else {
		maxBridges = scaleCfg.maxBridgesLong
		maxVnics = scaleCfg.maxVnicsLong
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
			role := libsnnet.TenantContainer

			vnicCfg := &libsnnet.VnicConfig{
				VnicRole:   role,
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

			// Create a VNIC: Should create bridge and tunnels
			_, _, cInfo, err := cn.CreateVnic(vnicCfg)
			if assert.Nil(err) {
				defer func() { _, _, _ = cn.DestroyVnic(vnicCfg) }()

				if cInfo.CNContainerEvent == libsnnet.ContainerNetworkAdd {
					err := dockerNetCreate(t, cInfo.Subnet, cInfo.Gateway,
						cInfo.Bridge, cInfo.SubnetID)
					if assert.Nil(err) {
						defer func() { _ = dockerNetDelete(t, cInfo.SubnetID) }()
					}
				}

				runNetworkTest(t, netType, vnicCfg.VnicIP.String(), vnicCfg.VnicIP, vnicCfg.VnicMAC, cInfo.SubnetID)
				defer func() { _ = dockerContainerDelete(t, vnicCfg.VnicIP.String()) }()
			}
		}
	}
}

func runNetworkTest(t *testing.T, netType dockerNetType, name string, ip net.IP, mac net.HardwareAddr, subnetID string) {
	assert := assert.New(t)

	switch netType {
	case netCiao:
		err := dockerRunVerify(t, name, ip, mac, subnetID)
		assert.Nil(err)
	case netDockerNone:
		err := dockerRunNetNone(t, name, ip, mac, subnetID)
		assert.Nil(err)
	case netDockerDefault:
		err := dockerRunNetDocker(t, name, ip, mac, subnetID)
		assert.Nil(err)
	}
}

//Tests launch of Docker containers at scale (serially)
//
//This tests exercises attempts to launch large
//numbers of docker containers at scale to isolate
//any issues with plugin responsiveness
//
//Test is expected to pass
func TestDockerNetCiao_Serial(t *testing.T) {
	DockerSerial(netCiao, t)
}

//Tests launch of Docker containers at scale (serially)
//
//This tests exercises attempts to launch large
//numbers of docker containers at scale to isolate
//any issues with plugin responsiveness
//This test benchmarks docker without networking
//
//Test is expected to pass
func TestDockerNetNone_Serial(t *testing.T) {
	DockerSerial(netDockerNone, t)
}

//Tests launch of Docker containers at scale (serially)
//
//This tests exercises attempts to launch large
//numbers of docker containers at scale to isolate
//any issues with plugin responsiveness
//This test benchmarks docker with default networking
//
//Test is expected to pass
func TestDockerNetDocker_Serial(t *testing.T) {
	DockerSerial(netDockerDefault, t)
}

func TestMain(m *testing.M) {
	libsnnet.Logger = gloginterface.CiaoGlogLogger{}

	os.Exit(m.Run())
}
