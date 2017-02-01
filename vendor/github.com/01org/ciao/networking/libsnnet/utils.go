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
	"math/rand"
	"net"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
)

const (
	prefixBridge   = "sbr"
	prefixVnic     = "svn"
	prefixVnicCont = "svp"
	prefixVnicHost = "svn"
	prefixCnciVnic = "svc"
	prefixGretap   = "sgt"
)

const ifaceRetryLimit = 10

var (
	ifaceRseed rand.Source
	ifaceRsrc  *rand.Rand
)

// EqualNetSlice compare 2 network slices
func EqualNetSlice(slice1, slice2 []string) bool {
	// if a slice is nil and other isn't then are not equal
	// if length of slices are not the same, are not equal
	if (slice1 == nil && slice2 != nil) ||
		(slice2 == nil && slice1 != nil) ||
		(len(slice1) != len(slice2)) {
		return false
	}
	sortedSlice1 := make(sort.StringSlice, len(slice1))
	sortedSlice2 := make(sort.StringSlice, len(slice2))
	copy(sortedSlice1, slice1)
	copy(sortedSlice2, slice2)
	sortedSlice1.Sort()
	sortedSlice2.Sort()
	return reflect.DeepEqual(sortedSlice1, sortedSlice2)

}

func init() {
	ifaceRseed = rand.NewSource(time.Now().UnixNano())
	ifaceRsrc = rand.New(ifaceRseed)
}

func validSnPrefix(s string) bool {
	switch {
	case strings.HasPrefix(s, prefixBridge):
	case strings.HasPrefix(s, prefixVnic):
	case strings.HasPrefix(s, prefixVnicCont):
	case strings.HasPrefix(s, prefixVnicHost):
	case strings.HasPrefix(s, prefixCnciVnic):
	case strings.HasPrefix(s, prefixGretap):
	default:
		return false
	}

	return true
}

func getPrefix(device interface{}) (string, error) {

	prefix := ""

	switch d := device.(type) {
	case *Bridge:
		prefix = prefixBridge
	case *Vnic:
		switch d.Role {
		case TenantVM:
			prefix = prefixVnic
		case TenantContainer:
			prefix = prefixVnicHost
		}
	case *GreTunEP:
		prefix = prefixGretap
	case *CnciVnic:
		prefix = prefixCnciVnic
	}

	if prefix == "" {
		return prefix, fmt.Errorf("invalid device type %T %v", device, device)
	}

	return prefix, nil

}

// GenIface generates locally unique interface names based on the
// type of device passed in. It will additionally check if the
// interface name exists on the localhost based on unique
// When uniqueness is specified error will be returned
// if it is not possible to generate a locally unique name within
// a finite number of retries
func genIface(device interface{}, unique bool) (string, error) {

	prefix, err := getPrefix(device)
	if err != nil {
		return "", err
	}

	if !unique {
		iface := fmt.Sprintf("%s_%x", prefix, ifaceRsrc.Uint32())
		return iface, nil
	}

	for i := 0; i < ifaceRetryLimit; i++ {
		iface := fmt.Sprintf("%s_%x", prefix, ifaceRsrc.Uint32())
		if _, err := netlink.LinkByName(iface); err != nil {
			return iface, nil
		}
	}

	// The chances of the failure are remote
	return "", fmt.Errorf("unable to create unique interface name")
}

func validPhysicalLink(link netlink.Link) bool {
	phyDevice := true

	switch link.Type() {
	case "device":
	case "bond":
	case "vlan":
	case "macvlan":
	case "bridge":
		if strings.HasPrefix(link.Attrs().Name, "docker") ||
			strings.HasPrefix(link.Attrs().Name, "virbr") {
			phyDevice = false
		}
	default:
		phyDevice = false
	}

	if (link.Attrs().Flags & net.FlagLoopback) != 0 {
		return false
	}

	if travisCI {
		return true
	}
	return phyDevice
}
