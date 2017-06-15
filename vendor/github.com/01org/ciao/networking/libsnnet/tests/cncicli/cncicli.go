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
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/01org/ciao/clogger/gloginterface"
	"github.com/01org/ciao/networking/libsnnet"
)

func cnciInit(cnci *libsnnet.Cnci) error {
	if err := cnci.Init(); err != nil {
		return err
	}
	if err := cnci.RebuildTopology(); err != nil {
		return err
	}
	return nil
}

func reset(cnci *libsnnet.Cnci) {
	if err := cnci.Shutdown(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	os.Exit(0)
}

func create(cnci *libsnnet.Cnci, tenantSubnet *net.IPNet, subnetKey uint32, cnIP net.IP) {
	if _, err := cnci.AddRemoteSubnet(*tenantSubnet, int(subnetKey), cnIP); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func delete(cnci *libsnnet.Cnci, tenantSubnet *net.IPNet, subnetKey uint32, cnIP net.IP) {
	if err := cnci.DelRemoteSubnet(*tenantSubnet, int(subnetKey), cnIP); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func main() {

	operationIn := flag.String("operation", "create", "operation <create|delete|reset> reset clears all CNCI setup")
	cnciSubnetIn := flag.String("cnciSubnet", "", "CNCI Physical subnet on which the NN can be reached")
	tenantSubnetIn := flag.String("tenantSubnet", "192.168.8.0/21", "Tenant subnet served by this CNCI")
	cnIPIn := flag.String("cnip", "127.0.0.1", "CNCI reachable CN IP address")
	cnciIDIn := flag.String("cnciuuid", "cnciuuid", "CNCI UUID")

	flag.Parse()

	libsnnet.Logger = gloginterface.CiaoGlogLogger{}

	cnci := &libsnnet.Cnci{
		ID: *cnciIDIn,
	}
	cnci.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	if *cnciSubnetIn != "" {
		_, cnciPhyNet, err := net.ParseCIDR(*cnciSubnetIn)
		if err != nil {
			fmt.Println("Error invalid CNCI IP", *cnciSubnetIn)
			os.Exit(-1)
		}
		cnci.ManagementNet = []net.IPNet{*cnciPhyNet}
		cnci.ComputeNet = []net.IPNet{*cnciPhyNet}
	}

	err := cnciInit(cnci)
	if err != nil {
		fmt.Printf("Init failed [%v]\n", err)
		os.Exit(-1)
	}

	if *operationIn == "reset" {
		reset(cnci)
	}

	_, tenantSubnet, err := net.ParseCIDR(*tenantSubnetIn)
	if err != nil {
		fmt.Println("Error invalid tenant subnet", *tenantSubnetIn)
		os.Exit(-1)
	}
	subnetKey := binary.LittleEndian.Uint32(tenantSubnet.IP)

	cnIP := net.ParseIP(*cnIPIn)
	if cnIP == nil {
		fmt.Println("Error invalid CN IP ", *cnIPIn)
		os.Exit(-1)
	}

	switch *operationIn {
	case "create":
		create(cnci, tenantSubnet, subnetKey, cnIP)
	case "delete":
		delete(cnci, tenantSubnet, subnetKey, cnIP)
	default:
		fmt.Println("Invalid operation ", *operationIn)
	}

	os.Exit(0)
}
