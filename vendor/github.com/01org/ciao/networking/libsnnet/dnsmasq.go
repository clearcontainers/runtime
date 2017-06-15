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
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

//Various configuration options
const (
	pidPath    = "/tmp/"
	leasePath  = "/tmp/"
	configPath = "/tmp/"
	hostsPath  = "/tmp/"
	MACPrefix  = "02:00" //Prefix for all private MAC addresses
//	CONFIG_PATH = "/etc/"
//	PID_PATH = "/var/run/"
)

//TODO: Set these up above to correct defaults

// Dnsmasq contains all the information required to spawn
// a dnsmasq process on behalf of a tenant on a concentrator
type Dnsmasq struct {
	SubnetID    string                // UUID of the Tenant Subnet to which the  dnsmasq supports
	CNCIId      string                // UUID of the CNCI instance
	TenantID    string                // UUID of the Tenant to which the CNCI belongs to
	TenantNet   net.IPNet             // The tenant subnet served by this dnsmasq, has to be /29 or larger
	ReservedIPs int                   // Reserve IP at the start of subnet
	ConcIP      net.IP                // IP Address of the CNCI
	IPMap       map[string]*DhcpEntry // Static mac to IP map, key is macaddress
	Dev         *Bridge               // The bridge on which dnsmasq will attach
	MTU         int                   // MTU that takes into account the tunnel overhead
	DomainName  string                // Domain Name to be assigned to the subnet

	// Private fields
	dhcpSize  int
	subnet    net.IP    // The DHCP addresses will be served from this subnet
	gateway   net.IPNet // The address of the bridge. Will also be default gw to the instances
	startIP   net.IP    // First address in the DHCP range Skipping ReservedIPs
	endIP     net.IP    // Last address in the DHCP range excluding broadcast
	confFile  string
	pidFile   string
	leaseFile string
	hostsFile string
}

// NewDnsmasq initializes a new dnsmasq instance and attaches it to the specified bridge
// The dnsmasq object is initialized but no operations have been executed or files created
// This is a pure in-memory operation
func newDnsmasq(id string, tenant string, subnet net.IPNet, reserved int, b *Bridge) (*Dnsmasq, error) {
	if b == nil {
		return nil, fmt.Errorf("invalid bridge")
	}

	d := &Dnsmasq{
		SubnetID:    id,
		TenantID:    tenant,
		TenantNet:   subnet,
		ReservedIPs: reserved,
		IPMap:       make(map[string]*DhcpEntry),
		Dev:         b,
	}

	if err := d.getFileConfiguration(); err != nil {
		return nil, err
	}

	if err := d.setMTU(); err != nil {
		return nil, err
	}

	if err := d.getSubnetConfiguration(); err != nil {
		return nil, err
	}

	return d, nil
}

// Start the dnsmasq service
// This creates the actual files and performs configuration
func (d *Dnsmasq) start() error {
	if err := d.createConfigFile(); err != nil {
		return fmt.Errorf("d.createConfigFile failed %v", err)
	}

	if err := d.createHostsFile(); err != nil {
		return fmt.Errorf("d.createHostsFile failed %v", err)
	}

	if err := d.Dev.AddIP(&d.gateway); err != nil {
		_ = d.Dev.DelIP(&d.gateway) //TODO: check it already has the IP
		if err = d.Dev.AddIP(&d.gateway); err != nil {
			return fmt.Errorf("d.Dev.AddIP failed %v %v", err, d.gateway.String())
		}
	}

	if err := d.launch(); err != nil {
		return fmt.Errorf("d.launch failed %v", err)
	}

	return nil
}

// Attach to an existing service
// Returns -1 and error on failure
// Returns pid of current process on success
func (d *Dnsmasq) attach() (int, error) {
	pid, err := d.getPid()

	if err != nil {
		return -1, fmt.Errorf("No pid file %v", err)
	}

	if err = syscall.Kill(pid, syscall.Signal(0)); err != nil {
		return -1, fmt.Errorf("Process does not exist or unable to attach %v", err)
	}
	return pid, nil
}

// Stop the dnsmasq service
func (d *Dnsmasq) stop() error {
	var cumError []error

	pid, err := d.attach()

	if err != nil {
		cumError = append(cumError, fmt.Errorf("Process does not exist %v", err))
	}

	if pid != -1 {
		if err = syscall.Kill(pid, syscall.SIGKILL); err != nil { //TODO: Try TERM
			cumError = append(cumError, fmt.Errorf("Unable to kill dnsmasq %v", err))
		} else {
			if err := os.Remove(d.pidFile); err != nil {
				cumError = append(cumError, fmt.Errorf("Unable to delete file %v %v", d.pidFile, err))
			}
		}
	}

	if err = d.Dev.DelIP(&d.gateway); err != nil {
		cumError = append(cumError, fmt.Errorf("Unable to delete bridge IP %v", err))
	}

	if err = os.Remove(d.confFile); err != nil {
		cumError = append(cumError, fmt.Errorf("Unable to delete file %v %v", d.confFile, err))
	}
	if err = os.Remove(d.hostsFile); err != nil {
		cumError = append(cumError, fmt.Errorf("Unable to delete file %v %v", d.hostsFile, err))
	}
	_ = os.Remove(d.leaseFile)

	if cumError != nil {
		allErrors := ""
		for _, e := range cumError {
			allErrors = allErrors + e.Error()
		}
		return errors.New(allErrors)
	}

	return nil
}

// Restart will stop and restart a new instance of dnsmasq
func (d *Dnsmasq) restart() error {
	_ = d.stop() //Ignore any errors

	if err := d.start(); err != nil {
		return fmt.Errorf("d.Start failed %v", err)
	}
	return nil
}

// Reload is called to update the configuration of the dnsmasq
// service. It is typically called when its configuration is updated
func (d *Dnsmasq) reload() error {

	pid, err := d.attach()
	if err != nil {
		return err
	}

	if err = d.getSubnetConfiguration(); err != nil {
		return fmt.Errorf("Unable to get subnet configuration %v", err)
	}

	//Note: This file will not take effect. Update it anyway
	if err = d.createConfigFile(); err != nil {
		return fmt.Errorf("Unable to delete config file %v", err)
	}
	if err = d.createHostsFile(); err != nil {
		return fmt.Errorf("Unable to delete hosts file %v", err)
	}
	if err = syscall.Kill(pid, syscall.SIGHUP); err != nil {
		return fmt.Errorf("Unable to reload/SIGHUP dnsmasq %v", err)
	}
	return nil
}

// AddDhcpEntry adds/updates a DHCP mapping. Typically invoked when a new
// instance is added to the subnet served by this dnsmasq service.
// Reload() has to be invoked to activate this entry is the service is already
// running
func (d *Dnsmasq) addDhcpEntry(entry *DhcpEntry) error {
	d.IPMap[entry.MACAddr.String()] = entry
	return nil
}

// Populates the file specific private variables
func (d *Dnsmasq) getFileConfiguration() error {

	if d.SubnetID == "" {
		return fmt.Errorf("invalid configuration  %v", d)
	}

	d.pidFile = fmt.Sprintf("%sdnsmasq_%s.pid", pidPath, d.SubnetID)
	d.confFile = fmt.Sprintf("%sdnsmasq_%s.conf", configPath, d.SubnetID)
	d.leaseFile = fmt.Sprintf("%sdnsmasq_%s.leases", leasePath, d.SubnetID)
	d.hostsFile = fmt.Sprintf("%sdnsmasq_%s.hosts", hostsPath, d.SubnetID)

	return nil
}

// Populates the subnet specific private variables
func (d *Dnsmasq) getSubnetConfiguration() error {

	// We need at least 2 IPs to work
	// One for the bridge and one for the tenant
	ones, bits := d.TenantNet.Mask.Size()
	if bits != 32 || ones > 30 || ones == 0 {
		return fmt.Errorf("invalid subnet %s", d.TenantNet.String())
	}
	subnetSize := ^(^0 << uint32(32-ones)) + 1

	// We need at least one IP for DHCP
	// 3 are reserved for subnet, gateway, and broadcast (subnet i.e. .0 can be
	// used but is currently is not due to legacy convention)
	if d.dhcpSize = subnetSize - d.ReservedIPs - 3; d.dhcpSize <= 0 {
		return fmt.Errorf("invalid reservation %s %v", d.TenantNet.String(), d.ReservedIPs)
	}

	//No deep copy implementation in net.IP
	//Mask is the closest to a deep copy
	//TODO Implement deep copy
	d.subnet = d.TenantNet.IP.To4().Mask(d.TenantNet.Mask)
	if d.subnet == nil {
		return fmt.Errorf("invalid subnet")
	}

	d.gateway.IP = d.TenantNet.IP.To4().Mask(d.TenantNet.Mask)
	d.gateway.Mask = d.TenantNet.Mask
	d.startIP = d.TenantNet.IP.To4().Mask(d.TenantNet.Mask)
	d.endIP = d.TenantNet.IP.To4().Mask(d.TenantNet.Mask)
	//End Hack

	//Skip the network address
	d.gateway.IP[3]++

	//Designate the first IP after network, gateway and reserved range
	startU32 := binary.BigEndian.Uint32(d.startIP)
	startU32 += uint32(2 + d.ReservedIPs)
	binary.BigEndian.PutUint32(d.startIP, startU32)

	endU32 := binary.BigEndian.Uint32(d.endIP)
	endU32 += startU32 + uint32(d.dhcpSize)
	binary.BigEndian.PutUint32(d.endIP, endU32)

	//Generate all valid IPs in this subnet and pre-assign a MAC address
	for i := 0; i < d.dhcpSize; i++ {
		vIP := make(net.IP, net.IPv4len)
		binary.BigEndian.PutUint32(vIP, startU32+uint32(i))

		//last 4 bytes will directly map to the desired IP address
		macStr := fmt.Sprintf("%s:%02x:%02x:%02x:%02x", MACPrefix, vIP[0], vIP[1], vIP[2], vIP[3])
		macAddr, err := net.ParseMAC(macStr)
		if err != nil {
			return err
		}

		dhcpEntry := &DhcpEntry{
			MACAddr: macAddr,
			IPAddr:  vIP,
		}

		if err := d.addDhcpEntry(dhcpEntry); err != nil {
			return err
		}
	}

	return nil
}

func (d *Dnsmasq) createHostsFile() error {
	file, err := os.Create(d.hostsFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	for _, e := range d.IPMap {
		s := fmt.Sprintf("%s,%s", e.MACAddr, e.IPAddr)
		if e.Hostname != "" {
			s = fmt.Sprintf("%s,%s", s, e.Hostname)
		}
		s = fmt.Sprintf("%s,id:*\n", s)
		if _, err := file.WriteString(s); err != nil {
			return err
		}
	}

	return file.Sync()
}

func (d *Dnsmasq) createConfigFile() error {
	params := make([]string, 20)

	if d.Dev == nil {
		return fmt.Errorf("bridge nil")
	}

	if d.Dev.LinkName == "" {
		return fmt.Errorf("bridge uninitialized")
	}

	params = append(params, fmt.Sprintf("pid-file=%s\n", d.pidFile))
	params = append(params, fmt.Sprintf("dhcp-leasefile=%s\n", d.leaseFile))
	params = append(params, fmt.Sprintf("dhcp-hostsfile=%s\n", d.hostsFile))
	//params = append(params, "strict-order\n")
	//params = append(params, "expand-hosts\n")
	if d.DomainName != "" {
		params = append(params, "domain=%s\n", d.DomainName)
	}
	params = append(params, "domain-needed\n")
	params = append(params, "bogus-priv\n")
	params = append(params, "bind-interfaces\n")
	params = append(params, fmt.Sprintf("interface=%s\n", d.Dev.LinkName))
	params = append(params, "except-interface=lo\n")
	params = append(params, "dhcp-no-override\n")
	params = append(params, "dhcp-ignore=tag!known\n")
	params = append(params, fmt.Sprintf("listen-address=%s\n", d.gateway.IP.String()))
	params = append(params, fmt.Sprintf("dhcp-range=%s,static\n", d.subnet.String()))
	params = append(params, fmt.Sprintf("dhcp-lease-max=%d\n", d.dhcpSize))
	params = append(params, fmt.Sprintf("dhcp-option-force=26,%d\n", d.MTU))
	//params = append(params, "log-dhcp\n")

	file, err := os.Create(d.confFile)
	if err != nil {
		return fmt.Errorf("Unable to create file %v %v", d.confFile, err)
	}
	defer func() { _ = file.Close() }()

	for _, s := range params {
		if _, err := file.WriteString(s); err != nil {
			return err
		}
	}

	return file.Sync()
}

func (d *Dnsmasq) launch() error {
	prog := "dnsmasq"
	args := fmt.Sprintf("--conf-file=%s", d.confFile)

	cmd := exec.Command(prog, args)
	_, err := cmd.Output()

	return err
}

func (d *Dnsmasq) getPid() (int, error) {

	pidbytes, err := ioutil.ReadFile(d.pidFile)
	if err != nil {
		return -1, err
	}

	//TODO: Check against the kernel.pid_max
	pidStr := strings.Trim(string(pidbytes), "\n")
	pid, err := strconv.ParseUint(pidStr, 10, 32)
	if err != nil {
		return -1, err
	}

	return int(pid), nil
}

func (d *Dnsmasq) setMTU() error {
	// TODO: Setup MTU based on tunnel type
	d.MTU = 1400
	return nil
}
