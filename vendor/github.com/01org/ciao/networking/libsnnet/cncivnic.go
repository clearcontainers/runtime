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
	"net"

	"github.com/vishvananda/netlink"
)

// NewCnciVnic is used to initialize the CnciVnic properties
// This has to be called prior to Create() or GetDevice()
func newCnciVnic(id string) (*CnciVnic, error) {
	CnciVnic := &CnciVnic{}
	CnciVnic.Link = &netlink.Macvtap{}
	CnciVnic.Link.Mode = netlink.MACVLAN_MODE_BRIDGE
	CnciVnic.Link.TxQLen = 500
	CnciVnic.GlobalID = id
	return CnciVnic, nil
}

// GetDevice is used to associate with an existing CnciVnic provided it satisfies
// the needs of a CnciVnic. Returns error if the CnciVnic does not exist
func (v *CnciVnic) getDevice() error {

	if v.GlobalID == "" {
		return netError(v, "getdevice unnamed cnci vnic")
	}

	link, err := netlink.LinkByAlias(v.GlobalID)
	if err != nil {
		return netError(v, "getdevice interface does not exist: %v", v.GlobalID)
	}

	vl, ok := link.(*netlink.Macvtap)
	if !ok {
		return netError(v, "getdevice incorrect interface type %v %v", v.GlobalID, link.Type())
	}

	if link.Type() != "macvtap" {
		return netError(v, "getdevice incorrect interface type %v %v", v.GlobalID, link.Type())
	}

	v.LinkName = vl.Name
	v.Link = vl
	return nil
}

// Create instantiates new vnic
func (v *CnciVnic) create() error {
	var err error

	if v.GlobalID == "" {
		return netError(v, "create cannot create an unnamed cnci vnic")
	}

	if v.LinkName == "" {
		if v.LinkName, err = genIface(v, true); err != nil {
			return netError(v, "create geniface %v %v", v.GlobalID, err)
		}

		if _, err := netlink.LinkByAlias(v.GlobalID); err == nil {
			return netError(v, "create interface exists %v %v", v.GlobalID, err)
		}
	}

	v.Link.Name = v.LinkName
	if v.Link.ParentIndex == 0 {
		return netError(v, "create parent index not set %v %v", v.GlobalID, v.Link)
	}

	if err := netlink.LinkAdd(v.Link); err != nil {
		return netError(v, "create netlink.LinkAdd %v %v", v.GlobalID, err)
	}

	link, err := netlink.LinkByName(v.LinkName)
	if err != nil {
		return netError(v, "create netlink.LinkAdd %v %v", v.GlobalID, err)
	}

	vl, ok := link.(*netlink.Macvtap)
	if !ok {
		return netError(v, "create incorrect interface type %v %v", v.GlobalID, link.Type())
	}

	v.Link = vl

	if err := v.setAlias(v.GlobalID); err != nil {
		err1 := v.destroy()
		return netError(v, "create set alias [%v] [%v] [%v]", v.GlobalID, err, err1)
	}

	if v.MACAddr != nil {
		if err := v.setHardwareAddr(*v.MACAddr); err != nil {
			err1 := v.destroy()
			return netError(v, "create set hardware addr [%v] [%v] [%v] [%v]",
				v.MACAddr.String(), v.GlobalID, err, err1)
		}
	}

	return nil
}

// Destroy a vnic
func (v *CnciVnic) destroy() error {

	if v.Link == nil {
		return netError(v, "destroy invalid Link: %v", v)
	}

	if err := netlink.LinkDel(v.Link); err != nil {
		return netError(v, "destroy link del %v", err)
	}

	return nil

}

// Enable the vnic
func (v *CnciVnic) enable() error {

	if v.Link == nil {
		return netError(v, "enable invalid link: %v", v)
	}

	if err := netlink.LinkSetUp(v.Link); err != nil {
		return netError(v, "enable link up %v", err)
	}

	return nil

}

// Disable the vnic
func (v *CnciVnic) disable() error {

	if v.Link == nil {
		return netError(v, "disable invalid link: %v", v)
	}

	if err := netlink.LinkSetDown(v.Link); err != nil {
		return netError(v, "disable link down %v", err)
	}

	return nil
}

func (v *CnciVnic) setAlias(alias string) error {

	if v.Link == nil {
		return netError(v, "set alias vnic unnitialized")
	}

	if err := netlink.LinkSetAlias(v.Link, alias); err != nil {
		return netError(v, "set alias link set alias %v %v", alias, err)
	}

	return nil
}

func (v *CnciVnic) setHardwareAddr(hwaddr net.HardwareAddr) error {

	if v.Link == nil {
		return netError(v, "set hw addr vnic unnitialized")
	}

	if err := netlink.LinkSetHardwareAddr(v.Link, hwaddr); err != nil {
		return netError(v, "set hwaddr %v %v", hwaddr.String(), err)
	}

	return nil
}
