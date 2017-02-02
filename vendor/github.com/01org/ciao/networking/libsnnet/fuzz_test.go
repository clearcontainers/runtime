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
	"testing"

	"github.com/google/gofuzz"
)

//Fuzz Test the CN APIs
//
//
//Fuzz Test the CN APIs
//
//
//Test should not crash
func TestCN_Fuzz(t *testing.T) {
	cn, err := cnTestInit()
	if err != nil {
		t.Fatal("ERROR: Init failed", err)
	}

	f := fuzz.New()

	for i := 0; i < 1000; i++ {
		vnic := VnicConfig{}
		vnic.CancelChan = make(chan interface{})
		f.Fuzz(&vnic.ConcID)
		f.Fuzz(&vnic.ConcID)
		f.Fuzz(&vnic.VnicRole)
		f.Fuzz(&vnic.VnicIP)
		f.Fuzz(&vnic.ConcIP)
		f.Fuzz(&vnic.VnicMAC)
		f.Fuzz(&vnic.MTU)
		f.Fuzz(&vnic.SubnetKey)
		f.Fuzz(&vnic.Subnet.IP)
		f.Fuzz(&vnic.Subnet.Mask)
		f.Fuzz(&vnic.VnicID)
		f.Fuzz(&vnic.InstanceID)
		f.Fuzz(&vnic.TenantID)
		f.Fuzz(&vnic.SubnetID)
		f.Fuzz(&vnic.ConcID)
		_, _, _, _ = cn.CreateVnic(&vnic)
		_, _, _ = cn.DestroyVnic(&vnic)
		_, _ = cn.CreateCnciVnic(&vnic)
		_ = cn.DestroyCnciVnic(&vnic)
	}
}

//Fuzz Test the CNCI APIs
//
//
//Fuzz Test the CNCI APIs
//
//
//Test should not crash
func TestCNCI_Fuzz(t *testing.T) {
	cnci, err := cnciTestInit()
	if err != nil {
		t.Fatal("ERROR: Init failed", err)
	}
	defer func() {
		_ = cnci.Shutdown()
	}()

	f := fuzz.New()

	for i := 0; i < 100; i++ {
		var subnet net.IPNet
		var subnetKey int
		var cnIP net.IP
		f.Fuzz(&subnet.IP)
		f.Fuzz(&subnet.Mask)
		f.Fuzz(&subnetKey)
		f.Fuzz(&cnIP)
		_, _ = cnci.AddRemoteSubnet(subnet, subnetKey, cnIP)
		_ = cnci.DelRemoteSubnet(subnet, subnetKey, cnIP)
	}
}

//Fuzz Test the Lower Level Network APIs
//
//
//Fuzz Test the Lower Level Network APIs
//
//
//Test should not crash
func TestNwPrimitives_Fuzz(t *testing.T) {

	f := fuzz.New()

	vnic := Vnic{}
	_ = vnic.Create()
	_ = vnic.Enable()
	_ = vnic.Disable()
	_ = vnic.GetDevice()
	_ = vnic.Destroy()
	for i := 0; i < 100; i++ {
		vnic := Vnic{}
		f.Fuzz(&vnic.Role)
		f.Fuzz(&vnic.GlobalID)
		f.Fuzz(&vnic.TenantID)
		f.Fuzz(&vnic.InstanceID)
		f.Fuzz(&vnic.BridgeID)
		_ = vnic.Create()
		_ = vnic.Enable()
		_ = vnic.Disable()
		_ = vnic.GetDevice()
		_ = vnic.Destroy()
	}

	bridge := Bridge{}
	_ = bridge.Create()
	_ = bridge.Enable()
	_ = bridge.Disable()
	_ = bridge.GetDevice()
	_ = bridge.Destroy()
	for i := 0; i < 100; i++ {
		bridge := Bridge{}
		f.Fuzz(&bridge.GlobalID)
		f.Fuzz(&bridge.TenantID)
		_ = bridge.Create()
		_ = bridge.Enable()
		_ = bridge.Disable()
		_ = bridge.GetDevice()
		_ = bridge.Destroy()
	}

	gre := &GreTunEP{}
	_ = gre.create()
	_ = gre.enable()
	_ = gre.disable()
	_ = gre.getDevice()
	_ = gre.destroy()
	for i := 0; i < 100; i++ {
		f.Fuzz(&gre.GlobalID)
		f.Fuzz(&gre.LocalIP)
		f.Fuzz(&gre.RemoteIP)
		f.Fuzz(&gre.Key)
		gre, _ = newGreTunEP(gre.GlobalID, gre.LocalIP, gre.RemoteIP, gre.Key)
		_ = gre.create()
		_ = gre.enable()
		_ = gre.disable()
		_ = gre.getDevice()
		_ = gre.destroy()
	}

	for i := 0; i < 100; i++ {
		vnic, _ := newCnciVnic("xyz")
		f.Fuzz(&vnic.Role)
		f.Fuzz(&vnic.GlobalID)
		f.Fuzz(&vnic.InstanceID)
		f.Fuzz(&vnic.BridgeID)
		_ = vnic.create()
		_ = vnic.enable()
		_ = vnic.disable()
		_ = vnic.getDevice()
		_ = vnic.destroy()
	}

	for i := 0; i < 100; i++ {
		vnic, _ := NewVnic("xyz")
		f.Fuzz(&vnic.Role)
		f.Fuzz(&vnic.GlobalID)
		f.Fuzz(&vnic.TenantID)
		f.Fuzz(&vnic.InstanceID)
		f.Fuzz(&vnic.BridgeID)
		_ = vnic.Create()
		_ = vnic.Enable()
		_ = vnic.Disable()
		_ = vnic.GetDevice()
		_ = vnic.Destroy()
	}
}
