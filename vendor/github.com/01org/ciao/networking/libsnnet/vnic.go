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
	"strings"
	"syscall"

	"github.com/vishvananda/netlink"
)

// NewVnic is used to initialize the Vnic properties
// This has to be called prior to Create() or GetDevice()
func NewVnic(id string) (*Vnic, error) {
	Vnic := &Vnic{}
	Vnic.Link = &netlink.GenericLink{}
	Vnic.GlobalID = id
	Vnic.Role = TenantVM
	return Vnic, nil
}

// NewContainerVnic is used to initialize a container Vnic properties
// This has to be called prior to Create() or GetDevice()
func NewContainerVnic(id string) (*Vnic, error) {
	Vnic := &Vnic{}
	Vnic.Link = &netlink.Veth{}
	Vnic.GlobalID = id
	Vnic.Role = TenantContainer
	return Vnic, nil
}

//InterfaceName is used to retrieve the name of the physical interface to
//which the VM or the container needs to be connected to
//Returns "" if the link is not setup
func (v *Vnic) InterfaceName() string {
	switch v.Role {
	case TenantVM:
		return v.LinkName
	case TenantContainer:
		return v.PeerName()
	default:
		return ""
	}
}

//PeerName is used to retrieve the peer name
//Returns "" if the link is not setup or if the link
//has no peer
func (v *Vnic) PeerName() string {
	if v.Role != TenantContainer {
		return v.LinkName
	}

	if strings.HasPrefix(v.LinkName, prefixVnicHost) {
		return strings.Replace(v.LinkName, prefixVnicHost, prefixVnicCont, 1)
	}

	if strings.HasPrefix(v.LinkName, prefixVnicCont) {
		return strings.Replace(v.LinkName, prefixVnicCont, prefixVnicHost, 1)
	}

	return fmt.Sprintf("%s_peer", v.LinkName)
}

// GetDevice is used to associate with an existing VNIC provided it satisfies
// the needs of a Vnic. Returns error if the VNIC does not exist
func (v *Vnic) GetDevice() error {

	if v.GlobalID == "" {
		return netError(v, "get device unnamed vnic")
	}

	link, err := netlink.LinkByAlias(v.GlobalID)
	if err != nil {
		return netError(v, "get device interface does not exist: %v", v.GlobalID)
	}

	switch v.Role {
	case TenantVM:
		vl, ok := link.(*netlink.GenericLink)
		if !ok {
			return netError(v, "get device incorrect interface type %v %v", v.GlobalID, link.Type())
		}

		// TODO: Why do both tun and tap interfaces return the type tun
		if link.Type() != "tun" {
			return netError(v, "get device incorrect interface type %v %v", v.GlobalID, link.Type())
		}

		if flags := uint(link.Attrs().Flags); (flags & syscall.IFF_TAP) == 0 {
			return netError(v, "get device incorrect interface type %v %v", v.GlobalID, link)
		}
		v.LinkName = vl.Name
		v.Link = vl
	case TenantContainer:
		vl, ok := link.(*netlink.Veth)
		if !ok {
			return netError(v, "get device incorrect interface type %v %v", v.GlobalID, link.Type())
		}
		v.LinkName = vl.Name
		v.Link = vl
	default:
		return netError(v, " invalid or unsupported VNIC type %v", v.GlobalID)
	}

	return nil
}

// GetDeviceByName is used to associate with an existing VNIC relying on its
// link name instead of its alias. Returns error if the VNIC does not exist
func (v *Vnic) GetDeviceByName(linkName string) error {

	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return netError(v, "get device interface does not exist: %v", linkName)
	}

	switch v.Role {
	case TenantVM:
		vl, ok := link.(*netlink.GenericLink)
		if !ok {
			return netError(v, "get device incorrect interface type %v %v", linkName, link.Type())
		}

		// TODO: Why do both tun and tap interfaces return the type tun
		if link.Type() != "tun" {
			return netError(v, "get device incorrect interface type %v %v", linkName, link.Type())
		}

		if flags := uint(link.Attrs().Flags); (flags & syscall.IFF_TAP) == 0 {
			return netError(v, "get device incorrect interface type %v %v", linkName, link)
		}
		v.LinkName = vl.Name
		v.Link = vl
	case TenantContainer:
		vl, ok := link.(*netlink.Veth)
		if !ok {
			return netError(v, "get device incorrect interface type %v %v", linkName, link.Type())
		}
		v.LinkName = vl.Name
		v.Link = vl
	default:
		return netError(v, " invalid or unsupported VNIC type %v", linkName)
	}

	return nil
}

func createVMVnic(v *Vnic) (link netlink.Link, err error) {

	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{Name: v.LinkName},
		Mode:      netlink.TUNTAP_MODE_TAP,
	}

	if err := netlink.LinkAdd(tap); err != nil {
		return nil, netError(v, "create link add %v %v", v.GlobalID, err)
	}

	link, err = netlink.LinkByName(v.LinkName)
	if err != nil {
		return nil, netError(v, "create link by name %v %v", v.GlobalID, err)
	}

	vl, ok := link.(*netlink.GenericLink)
	if !ok {
		return nil, netError(v, "create incorrect interface type %v %v", v.GlobalID, link.Type())
	}

	return vl, nil
}

func createContainerVnic(v *Vnic) (link netlink.Link, err error) {
	//We create only the host side veth, the container side is setup by the kernel
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: v.LinkName,
		},
		PeerName: v.PeerName(),
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return nil, netError(v, "create link add %v %v", v.GlobalID, err)
	}

	link, err = netlink.LinkByName(v.LinkName)
	if err != nil {
		return nil, netError(v, "create link by name %v %v", v.GlobalID, err)
	}
	vl, ok := link.(*netlink.Veth)
	if !ok {
		return nil, netError(v, "create incorrect interface type %v %v", v.GlobalID, link.Type())
	}

	return vl, nil
}

// Create instantiates new VNIC
func (v *Vnic) Create() error {
	var err error

	if v.GlobalID == "" {
		return netError(v, "create cannot create an unnamed vnic")
	}

	if v.LinkName == "" {
		if v.LinkName, err = genIface(v, true); err != nil {
			return netError(v, "create geniface %v %v", v.GlobalID, err)
		}

		if _, err := netlink.LinkByAlias(v.GlobalID); err == nil {
			return netError(v, "create interface exists %v", v.GlobalID)
		}
	}

	switch v.Role {
	case TenantVM:
		link, err := createVMVnic(v)
		if err != nil {
			return netError(v, "createVMVnic %v", err)
		}

		v.Link = link
	case TenantContainer:
		//We create only the host side veth, the container side is setup by the kernel
		link, err := createContainerVnic(v)
		if err != nil {
			return err
		}

		v.Link = link
	default:
		return netError(v, "invalid vnic role specified")
	}

	if err := v.setAlias(v.GlobalID); err != nil {
		_ = v.Destroy()
		return netError(v, "create set alias %v %v", v.GlobalID, err)
	}

	return nil
}

// Destroy a VNIC
func (v *Vnic) Destroy() error {

	if v.Link == nil || v.Link.Attrs().Index == 0 {
		return netError(v, "destroy unnitialized")
	}

	if err := netlink.LinkDel(v.Link); err != nil {
		return netError(v, "destroy link [%v] del [%v]", v.LinkName, err)
	}

	return nil

}

// Attach the VNIC to a bridge or a switch. Will return error if the VNIC
// incapable of binding to the specified device
func (v *Vnic) Attach(dev interface{}) error {

	if v.Link == nil || v.Link.Attrs().Index == 0 {
		return netError(v, "attach unnitialized")
	}

	br, ok := dev.(*Bridge)
	if !ok {
		return netError(v, "attach device %v, %T", dev, dev)
	}

	if br.Link == nil || br.Link.Index == 0 {
		return netError(v, "attach bridge unnitialized")
	}

	if err := netlink.LinkSetMaster(v.Link, br.Link); err != nil {
		return netError(v, "attach set master %v", err)
	}

	return nil
}

// Detach the VNIC from the device it is attached to
func (v *Vnic) Detach(dev interface{}) error {

	if v.Link == nil || v.Link.Attrs().Index == 0 {
		return netError(v, "detach unnitialized")
	}

	br, ok := dev.(*Bridge)

	if !ok {
		return netError(v, "detach unknown device %v, %T", dev, dev)
	}

	if br.Link == nil {
		return netError(v, "detach bridge unnitialized")
	}

	if err := netlink.LinkSetNoMaster(v.Link); err != nil {
		return netError(v, "detach set no master %v", err)
	}

	return nil
}

// Enable the VNIC
func (v *Vnic) Enable() error {

	if v.Link == nil || v.Link.Attrs().Index == 0 {
		return netError(v, "enable unnitialized")
	}

	if err := netlink.LinkSetUp(v.Link); err != nil {
		return netError(v, "enable link set set up %v", err)
	}

	return nil

}

// Disable the VNIC
func (v *Vnic) Disable() error {

	if v.Link == nil || v.Link.Attrs().Index == 0 {
		return netError(v, "disable unnitialized")
	}

	if err := netlink.LinkSetDown(v.Link); err != nil {
		return netError(v, "disable link set down %v", err)
	}

	return nil
}

//SetMTU of the interface
func (v *Vnic) SetMTU(mtu int) error {

	if v.Link == nil || v.Link.Attrs().Index == 0 {
		return netError(v, "disable unnitialized")
	}

	switch v.Role {
	case TenantVM:
		/* Set by DHCP. */
	case TenantContainer:
		/* Need to set the MTU of both ends */
		if err := netlink.LinkSetMTU(v.Link, mtu); err != nil {
			return netError(v, "link set mtu %v", err)
		}
		peerVeth := &netlink.Veth{
			LinkAttrs: netlink.LinkAttrs{
				Name: v.PeerName(),
			},
			PeerName: v.LinkName,
		}
		if err := netlink.LinkSetMTU(peerVeth, mtu); err != nil {
			return netError(v, "link set peer mtu %v", err)
		}
	}

	return nil
}

//SetHardwareAddr of the interface
func (v *Vnic) SetHardwareAddr(addr net.HardwareAddr) error {

	if v.Link == nil || v.Link.Attrs().Index == 0 {
		return netError(v, "disable unnitialized")
	}

	switch v.Role {
	case TenantVM:
		/* Set by QEMU. */
	case TenantContainer:
		/* Need to set the MAC on the container side */
		if err := netlink.LinkSetHardwareAddr(v.Link, addr); err != nil {
			return netError(v, "link set hardware addr %v", err)
		}
	}

	return nil
}

func (v *Vnic) setAlias(alias string) error {

	if v.Link == nil || v.Link.Attrs().Index == 0 {
		return netError(v, "set alias unnitialized")
	}

	if err := netlink.LinkSetAlias(v.Link, alias); err != nil {
		return netError(v, "link set alias %v %v", alias, err)
	}

	return nil
}
