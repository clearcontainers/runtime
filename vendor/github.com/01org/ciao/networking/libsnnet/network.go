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
	"time"

	"github.com/vishvananda/netlink"
)

//TODO: Add more info based on object level details and caller
func netError(dev interface{}, format string, args ...interface{}) error {
	switch dev.(type) {
	case Bridge, *Bridge:
		return fmt.Errorf("bridge error: "+format, args...)
	case Vnic, *Vnic:
		return fmt.Errorf("vnic error: "+format, args...)
	case CnciVnic, *CnciVnic:
		return fmt.Errorf("cncivnic error: "+format, args...)
	case GreTunEP, *GreTunEP:
		return fmt.Errorf("gre error: "+format, args...)
	}
	return fmt.Errorf("network error: "+format, args...)
}

type networkError struct {
	msg      string
	when     time.Time
	category string
}

// FatalError indicates that the system may be in an inconsistent
// state due to the error. The caller needs to initiate some sort of recovery.
// No new workloads should be scheduled on this node until the error is
// resolved
type FatalError struct {
	networkError
}

func (e FatalError) Error() string {
	return e.msg
}

//NewFatalError is a non recoverable error
func NewFatalError(s string) FatalError {
	return FatalError{
		networkError: networkError{
			msg:      s,
			when:     time.Now(),
			category: "FATAL",
		},
	}
}

// APIError indicates that the networking call failed. However the system
// is still consistent and the networking layer has performed appropriate cleanup
type APIError struct {
	networkError
}

func (e APIError) Error() string {
	return e.msg
}

//NewAPIError is a recoverable error
func NewAPIError(s string) APIError {
	return APIError{
		networkError: networkError{
			msg:      s,
			when:     time.Now(),
			category: "API",
		},
	}
}

// NetworkMode describes the networking configuration of the data center
type NetworkMode int

const (
	// Routed means all traffic is routed with no tenant isolation except through firewall rules
	Routed NetworkMode = iota
	// GreTunnel means tenant instances interlinked using GRE tunnels. Full tenant isolation
	GreTunnel
)

// VnicRole specifies the role of the VNIC
type VnicRole int

const (
	//TenantVM role is assigned to tenant VM
	TenantVM VnicRole = iota //Attached to a VM in the tenant network
	//TenantContainer role is assigned to a tenant container
	TenantContainer //Attach to a container in the tenant network
	//DataCenter role is assigned to resources owned by the data center
	DataCenter //Attached to the data center network
)

// Network describes the configuration of the data center network.
// This is the physical configuration of the data center.
// The Management Networks carry management/control SSNTP traffic
// The Compute Network carries tenant traffic.
// In a simplistic configuration the management network and the compute networks
// may be one and the same.
type Network struct {
	ManagementNet []net.IPNet // Enumerates all possible management subnets
	ComputeNet    []net.IPNet // Enumerates all possible compute subnets
	FloatingPool  []net.IP    // Available floating IPs
	PublicGw      net.IP      // Public IP Gateway to reach the internet
	Mode          NetworkMode
}

// Attrs contains fields common to all device types
type Attrs struct {
	LinkName string // Locally unique device name
	TenantID string // UUID of the tenant the device belongs to
	// Auto generated. Combination of UUIDs and other params.
	// Typically assigned to the alias
	// It is both locally and globally unique
	// Fully qualifies the device and its role
	GlobalID string
	MACAddr  *net.HardwareAddr
}

// Bridge represents a ciao Bridge
type Bridge struct {
	Attrs
	Link *netlink.Bridge
}

// DhcpEntry is the fully qualified MAC address to IP mapping
type DhcpEntry struct {
	MACAddr  net.HardwareAddr
	IPAddr   net.IP
	Hostname string // Optional
}

//VnicAttrs represent common Vnic attributes
type VnicAttrs struct {
	Attrs
	Role       VnicRole
	InstanceID string // UUID of the instance to which it will attach
	BridgeID   string // ID of bridge it has attached to
	IPAddr     *net.IP
	MTU        int
}

// Vnic represents a ciao VNIC (typically a tap or veth interface)
type Vnic struct {
	VnicAttrs
	Link netlink.Link // TODO: Enhance netlink library to add specific tap type to libnetlink
}

// CnciVnic represents a ciao CNCI VNIC
// This is used to connect a CNCI instance to the network
// A CNCI VNIC will be directly attached to the data center network
// Currently we use MacVtap in VEPA mode. We can also use MacVtap in Bridge Mode
type CnciVnic struct {
	VnicAttrs
	Link *netlink.Macvtap
}

// GreTunEP ciao GRE Tunnel representation
// This represents one end of the tunnel
type GreTunEP struct {
	Attrs
	Link     *netlink.Gretap
	Key      uint32
	LocalIP  net.IP
	RemoteIP net.IP
	CNCIId   string // UUID of the CNCI
	CNId     string // UUID of the CN
}
