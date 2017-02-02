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

// NewGreTunEP is used to initialize the GRE tunnel properties
// This has to be called prior to Create() or GetDevice()
func newGreTunEP(id string, localIP net.IP, remoteIP net.IP, key uint32) (*GreTunEP, error) {
	gre := &GreTunEP{}
	gre.Link = &netlink.Gretap{}
	gre.GlobalID = id
	gre.LocalIP = localIP
	gre.RemoteIP = remoteIP
	gre.Key = key
	return gre, nil
}

// GetDevice associates the tunnel with an existing GRE tunnel end point
func (g *GreTunEP) getDevice() error {

	if g.GlobalID == "" {
		return netError(g, "get device unnamed gretap device")
	}

	link, err := netlink.LinkByAlias(g.GlobalID)
	if err != nil {
		return netError(g, "get device interface does not exist: %v %v", g.GlobalID, err)
	}

	gl, ok := link.(*netlink.Gretap)
	if !ok {
		return netError(g, "get device incorrect interface type %v %v", g.GlobalID, link.Type())
	}
	g.Link = gl
	g.LinkName = gl.Name
	g.LocalIP = gl.Local
	g.RemoteIP = gl.Remote
	if gl.IKey == gl.OKey {
		g.Key = gl.IKey
	} else {
		return netError(g, "get device incorrect params IKey != OKey %v %v", g.GlobalID, gl)
	}

	return nil
}

// Create instantiates a tunnel
func (g *GreTunEP) create() error {
	var err error

	if g.GlobalID == "" || g.Key == 0 {
		return netError(g, "create cannot create an unnamed gretap device")
	}

	if g.LinkName == "" {
		if g.LinkName, err = genIface(g, false); err != nil {
			return netError(g, "create geniface %v, %v", g.GlobalID, err)
		}

		if lerr, err := netlink.LinkByAlias(g.GlobalID); err == nil {
			return netError(g, "create interface exists %v, %v", g.GlobalID, lerr)
		}
	}

	attrs := netlink.NewLinkAttrs()
	attrs.Name = g.LinkName

	gretap := &netlink.Gretap{LinkAttrs: attrs,
		IKey:     g.Key,
		OKey:     g.Key,
		Local:    g.LocalIP,
		Remote:   g.RemoteIP,
		PMtuDisc: 1,
	}

	if err := netlink.LinkAdd(gretap); err != nil {
		return netError(g, "create link add %v %v", g.GlobalID, err)
	}

	link, err := netlink.LinkByName(g.LinkName)
	if err != nil {
		return netError(g, "create link by name %v %v", g.GlobalID, err)
	}

	gl, ok := link.(*netlink.Gretap)
	if !ok {
		return netError(g, "create incorrect interface type %v, %v", g.GlobalID, link.Type())
	}
	g.Link = gl

	if err := g.setAlias(g.GlobalID); err != nil {
		_ = g.destroy()
		return netError(g, "create link set alias %v %v", g.GlobalID, err)
	}

	return nil
}

// Destroy an existing Tunnel
func (g *GreTunEP) destroy() error {

	if g.Link == nil || g.Link.Index == 0 {
		return netError(g, "destroy invalid gre link: %v", g)
	}

	if err := netlink.LinkDel(g.Link); err != nil {
		return netError(g, "destroy link del %v", err)
	}

	return nil
}

// Enable the GreTunnel
func (g *GreTunEP) enable() error {

	if g.Link == nil || g.Link.Index == 0 {
		return netError(g, "enable invalid gre link: %v", g)
	}

	if err := netlink.LinkSetUp(g.Link); err != nil {
		return netError(g, "enable link enable %v", err)
	}

	return nil

}

// Disable the Tunnel
func (g *GreTunEP) disable() error {
	if g.Link == nil || g.Link.Index == 0 {
		return netError(g, "disable invalid gre link: %v", g)
	}

	if err := netlink.LinkSetDown(g.Link); err != nil {
		return netError(g, "disable link disable %v", err)
	}
	return nil
}

func (g *GreTunEP) setAlias(alias string) error {
	if g.Link == nil || g.Link.Index == 0 {
		return netError(g, "set alias invalid gre link: %v", g)
	}

	if err := netlink.LinkSetAlias(g.Link, alias); err != nil {
		return netError(g, "set alias link set alias %v %v", alias, err)
	}

	return nil
}

// Attach the GRE tunnel to a device/bridge/switch
func (g *GreTunEP) attach(dev interface{}) error {

	if g.Link == nil || g.Link.Index == 0 {
		return netError(g, "attach gre tunnel unnitialized")
	}

	br, ok := dev.(*Bridge)
	if !ok {
		return netError(g, "attach unknown device %v, %T", dev, dev)
	}

	if br.Link == nil || br.Link.Index == 0 {
		return netError(g, "attach bridge unnitialized")
	}

	err := netlink.LinkSetMaster(g.Link, br.Link)
	if err != nil {
		return netError(g, "attach link set master %v", err)
	}

	return nil
}

// Detach the GRE Tunnel from the device/bridge it is attached to
func (g *GreTunEP) detach(dev interface{}) error {
	if g.Link == nil || g.Link.Index == 0 {
		return netError(g, "detach invalid gre link: %v", g)
	}

	br, ok := dev.(*Bridge)
	if !ok {
		return netError(g, "detach incorrect device type %v, %T", dev, dev)
	}

	if br.Link == nil || br.Link.Index == 0 {
		return netError(g, "detach bridge unnitialized")
	}

	if err := netlink.LinkSetNoMaster(g.Link); err != nil {
		return netError(g, "detach link set no master %v", err)
	}

	return nil
}
