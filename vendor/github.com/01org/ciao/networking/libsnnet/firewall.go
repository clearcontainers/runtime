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
	"os"
	"os/exec"
	"strconv"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
)

/* https://wiki.archlinux.org/index.php/iptables

                               XXXXXXXXXXXXXXXXXX
                             XXX     Network    XXX
                               XXXXXXXXXXXXXXXXXX
                                       +
                                       |
                                       v
 +-------------+              +------------------+
 |table: filter| <---+        | table: nat       |
 |chain: INPUT |     |        | chain: PREROUTING|
 +-----+-------+     |        +--------+---------+
       |             |                 |
       v             |                 v
 [local process]     |           ****************          +--------------+
       |             +---------+ Routing decision +------> |table: filter |
       v                         ****************          |chain: FORWARD|
****************                                           +------+-------+
Routing decision                                                  |
****************                                                  |
       |                                                          |
       v                        ****************                  |
+-------------+       +------>  Routing decision  <---------------+
|table: nat   |       |         ****************
|chain: OUTPUT|       |               +
+-----+-------+       |               |
      |               |               v
      v               |      +-------------------+
+--------------+      |      | table: nat        |
|table: filter | +----+      | chain: POSTROUTING|
|chain: OUTPUT |             +--------+----------+
+--------------+                      |
                                      v
                               XXXXXXXXXXXXXXXXXX
                             XXX    Network     XXX
                               XXXXXXXXXXXXXXXXXX

*/

const (
	procIPFwd = "/proc/sys/net/ipv4/ip_forward"
)

//FwAction defines firewall action to be performed
type FwAction int

const (
	//FwDisable disables
	FwDisable FwAction = iota
	//FwEnable enables
	FwEnable
)

//Firewall defines a single firewall instance
type Firewall struct {
	ExtInterfaces []string
	*iptables.IPTables
}

//InitFirewall Enables routing on the node and NAT on all
//external facing interfaces. Enable NAT right away to prevent
//tenant traffic escape
//TODO: Only enable external routing. Internal routing should
//always be enabled
func InitFirewall(devices ...string) (*Firewall, error) {

	if len(devices) == 0 {
		return nil, fmt.Errorf("initFirewall: Invalid input params")
	}

	ipt, err := iptables.New()
	if err != nil {
		return nil, fmt.Errorf("initFirewall: Unable to setup iptables %v", err)
	}

	f := &Firewall{
		IPTables: ipt,
	}

	// create CIAO Floating IPs user defined chains
	floatingIPsChains := []string{"ciao-floating-ip-pre", "ciao-floating-ip-post"}
	for _, chain := range floatingIPsChains {
		// verify it exists if not create it
		_ = ipt.NewChain("nat", chain)
	}

	// insert ciao-floating-ip-pre into PREROUTING Chain
	ok, err := ipt.Exists("nat", "PREROUTING", "-j", "ciao-floating-ip-pre")
	if err != nil {
		return nil, fmt.Errorf("Error: InitFirewall could not verify existence of chain ciao-floating-ip-pre, %v", err)
	}
	if !ok {
		err := ipt.Insert("nat", "PREROUTING", 1, "-j", "ciao-floating-ip-pre")
		if err != nil {
			return nil, fmt.Errorf("Error: InitFirewall could not create ciao-floating-ip-pre chain")
		}
	}

	// insert ciao-floating-ip-post into POSTROUTING Chain
	ok, err = ipt.Exists("nat", "POSTROUTING", "-j", "ciao-floating-ip-post")
	if err != nil {
		return nil, fmt.Errorf("Error: InitFirewall could not verify existence of chain ciao-floating-ip-post, %v", err)
	}
	if !ok {
		err := ipt.Insert("nat", "POSTROUTING", 1, "-j", "ciao-floating-ip-post")
		if err != nil {
			return nil, fmt.Errorf("Error: InitFirewall could not create ciao-floating-ip-post chain")
		}
	}

	for _, device := range devices {

		//iptables -t nat -A POSTROUTING -o $device -j MASQUERADE
		err = ipt.AppendUnique("nat", "POSTROUTING",
			"-o", device, "-j", "MASQUERADE")

		if err != nil {
			ok, err := ipt.Exists("nat", "POSTROUTING",
				"-o", device, "-j", "MASQUERADE")
			if !ok {
				return nil, fmt.Errorf("Error: InitFirewall NAT enable [%v] %v", device, err)
			}
		}

		f.ExtInterfaces = append(f.ExtInterfaces, device)
	}

	if err = Routing(FwEnable); err != nil {
		return nil, fmt.Errorf("Error: InitFirewall routing enable %v", err)
	}

	return f, nil

}

//ShutdownFirewall Disables routing and NAT
//TODO: Only external routing should be disabled.
func (f *Firewall) ShutdownFirewall() error {

	if err := Routing(FwDisable); err != nil {
		return fmt.Errorf("Error: Shutdown Firewall routing disable %v", err)
	}

	for _, device := range f.ExtInterfaces {

		err := f.Delete("nat", "POSTROUTING",
			"-o", device, "-j", "MASQUERADE")

		if err != nil {
			return fmt.Errorf("Error: Shutdown Firewall NAT disable %v", err)
		}
	}

	return nil
}

//Routing enable or disables routing
//echo 0 > /proc/sys/net/ipv4/ip_forward
//echo 1 > /proc/sys/net/ipv4/ip_forward
func Routing(action FwAction) error {
	file, err := os.OpenFile(procIPFwd, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Routing: Unable to open %v %v", procIPFwd, err)
	}
	defer func() { _ = file.Close() }()

	switch action {
	case FwEnable:
		_, err = file.WriteString("1")
	case FwDisable:
		_, err = file.WriteString("0")
	}

	if err != nil {
		return fmt.Errorf("Routing failed %v %v", action, err)
	}

	return nil
}

//ExtFwding enables or disables fwding between an externally connected interface
//and a tenant bridge (hence a tenant subnet)
//Each tenant subnet created needs explicit enabling/disabling
func (f *Firewall) ExtFwding(action FwAction, extDevice string, intDevice string) error {
	switch action {
	case FwEnable:
		//iptables -A FORWARD -i $extDevice -o $intDevice
		// -m state --state RELATED,ESTABLISHED -j ACCEPT
		err := f.AppendUnique("filter", "FORWARD",
			"-i", extDevice, "-o", intDevice,
			"-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT")

		if err != nil {
			return fmt.Errorf("enable inbound fwding failed: %v", err)
		}

		//iptables -A FORWARD -i $intDevice -o $extDevice -j ACCEPT
		err = f.AppendUnique("filter", "FORWARD",
			"-i", intDevice, "-o", extDevice, "-j", "ACCEPT")
		if err != nil {
			return fmt.Errorf("enable outbound fwding failed: %v [%s] [%s]",
				err, intDevice, extDevice)
		}
	case FwDisable:
		//iptables -D FORWARD -i $extDevice -o $intDevice
		// -m state --state RELATED,ESTABLISHED -j ACCEPT
		err := f.Delete("filter", "FORWARD",
			"-i", extDevice, "-o", intDevice,
			"-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT")

		if err != nil {
			return fmt.Errorf("disable inbound fwding failed: %v", err)
		}

		//iptables -D FORWARD -i $intDevice -o $extDevice -j ACCEPT
		err = f.Delete("filter", "FORWARD",
			"-i", intDevice, "-o", extDevice, "-j", "ACCEPT")
		if err != nil {
			return fmt.Errorf("disable outbound fwding failed: %v [%s] [%s]",
				err, intDevice, extDevice)
		}
	}

	return nil
}

//ExtPortAccess Enables/Disables port access via external device and port
//to an internal IP address and port for the specified protocol
func (f *Firewall) ExtPortAccess(action FwAction, protocol string, extDevice string,
	externalPort int, internalIP net.IP, internalPort int) error {
	ePort := strconv.Itoa(externalPort)
	iPort := strconv.Itoa(internalPort)

	var err error
	switch action {
	case FwEnable:
		//iptables -t nat -A PREROUTING
		//-i $extDevice -p $protocol --dport $extPort -j DNAT
		//--to $intIP:$intPort
		err = f.AppendUnique("nat", "PREROUTING",
			"-i", extDevice, "-p", protocol, "--dport", ePort, "-j", "DNAT",
			"--to", internalIP.String()+":"+iPort)

		if err != nil {
			ok, err2 := f.Exists("nat", "PREROUTING",
				"-i", extDevice, "-p", protocol, "--dport", ePort, "-j", "DNAT",
				"--to", internalIP.String()+":"+iPort)

			if !ok {
				err = fmt.Errorf("unable to enable ssh %v %v [%v],[%v]",
					internalIP, iPort, err, err2)
			}
		}
	case FwDisable:
		//iptables -t nat -D PREROUTING
		//-i $extDevice -p $protocol --dport $extPort -j DNAT
		//--to $intIP:$intPort
		err = f.Delete("nat", "PREROUTING",
			"-i", extDevice, "-p", protocol, "--dport", ePort, "-j", "DNAT",
			"--to", internalIP.String()+":"+iPort)

		if err != nil {
			ok, err2 := f.Exists("nat", "PREROUTING",
				"-i", extDevice, "-p", protocol, "--dport", ePort, "-j", "DNAT",
				"--to", internalIP.String()+":"+iPort)

			if ok {
				err = fmt.Errorf("unable to disable ssh %v %v [%v],[%v]",
					internalIP, iPort, err, err2)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("Unable to %v access for %v %v %v %v %v",
			action, protocol, extDevice, internalIP, externalPort, err)
	}

	return nil
}

func ipAssign(action FwAction, ip net.IP, iface string) error {

	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("Unable to detect interface %v %v", iface, err)
	}

	addr := &netlink.Addr{IPNet: &net.IPNet{
		IP:   ip.To4(),
		Mask: net.IPv4Mask(255, 255, 255, 255),
	},
	}

	switch action {
	case FwEnable:
		err := netlink.AddrAdd(link, addr)
		if err == nil {
			return nil
		}
		//Delete the address if it exists and re-add
		//This is more definitive than searching the IP list
		err = netlink.AddrDel(link, addr)
		if err != nil {
			return fmt.Errorf("Unable to assign IP to interface %s %v %v", ip, iface, err)
		}
		err = netlink.AddrAdd(link, addr)
		if err != nil {
			return fmt.Errorf("Unable to assign IP to interface %s %v %v", ip, iface, err)
		}
	case FwDisable:
		err = netlink.AddrDel(link, addr)
		if err == nil {
			return nil
		}

		//Check if someone deleted it
		addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil || len(addrs) == 0 {
			return fmt.Errorf("Unable to unassign IP from interface %s %v %v", ip, iface, err)
		}

		for _, ad := range addrs {
			if ad.Equal(*addr) {
				return fmt.Errorf("Unable to unassign IP from interface %s %v %v", ip, iface, err)
			}
		}
		return nil
	}

	return nil
}

//PublicIPAccess Enables/Disables public access to an internal IP
func (f *Firewall) PublicIPAccess(action FwAction,
	internalIP net.IP, publicIP net.IP, extInterface string) error {

	intIP := internalIP.String()
	pubIP := publicIP.String()

	switch action {
	case FwEnable:
		// assign the pubIP to the cnci agent
		err := ipAssign(FwEnable, publicIP, extInterface)
		if err != nil {
			return fmt.Errorf("Public IP Assignment failure %v", err)
		}
		return enablePublicIP(intIP, pubIP)
	case FwDisable:
		// remove the pubIP from the cnci agent
		err := ipAssign(FwDisable, publicIP, extInterface)
		if err != nil {
			return fmt.Errorf("Public IP Assignment failure %v", err)
		}

		return disablePublicIP(intIP, pubIP)
	default:
		return fmt.Errorf("Invalid parameter %v", action)
	}
}

func enablePublicIP(intIP, pubIP string) error {
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("initFirewall: Unable to setup iptables %v", err)
	}

	// insert DNAT rule of PREROUTING
	// iptables -t nat -I ciao-floating-ip-pre -d <pubIP> -j DNAT --to-destination <intIP>
	ok, err := ipt.Exists("nat", "ciao-floating-ip-pre", "-d", pubIP+"/32", "-j", "DNAT", "--to-destination", intIP)
	if err != nil {
		return fmt.Errorf("Error: InitFirewall could not verify existence of PREROUTING rule %s to %s", pubIP, intIP)
	}

	if !ok {
		err := ipt.Insert("nat", "ciao-floating-ip-pre", 1, "-d", pubIP+"/32", "-j", "DNAT", "--to-destination", intIP)
		if err != nil {
			return fmt.Errorf("Could not insert firewall PREROUTING rule %s to %s into chain ciao-floating-ip-pre", pubIP, intIP)
		}
	}

	// insert SNAT rule of POSTROUTING
	// iptables -t nat -I ciao-floating-ip-post -s <intIP> -j SNAT --to-source <pubIP>
	ok, err = ipt.Exists("nat", "ciao-floating-ip-post", "-s", intIP+"/32", "-j", "SNAT", "--to-source", pubIP)
	if err != nil {
		return fmt.Errorf("Error: InitFirewall could not verify existence of POSTROUTING rule %s to %s", intIP, pubIP)
	}

	if !ok {
		err := ipt.Insert("nat", "ciao-floating-ip-post", 1, "-s", intIP+"/32", "-j", "SNAT", "--to-source", pubIP)
		if err != nil {
			return fmt.Errorf("Could not insert firewall POSTROUTING rule %s to %s into chain ciao-floating-ip-post", intIP, pubIP)
		}
	}

	return nil
}

func disablePublicIP(intIP, pubIP string) error {
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("initFirewall: Unable to setup iptables %v", err)
	}

	// delete DNAT PREROUTING rule
	// iptables -t nat -D ciao-floating-ip-pre -d <pubIP> -j DNAT --to-destination <intIP>
	ok, err := ipt.Exists("nat", "ciao-floating-ip-pre", "-d", pubIP+"/32", "-j", "DNAT", "--to-destination", intIP)
	if err != nil {
		return fmt.Errorf("Error: InitFirewall could not verify existence of PREROUTING rule %s to %s", pubIP, intIP)
	}

	if ok {
		err := ipt.Delete("nat", "ciao-floating-ip-pre", "-d", pubIP+"/32", "-j", "DNAT", "--to-destination", intIP)
		if err != nil {
			return fmt.Errorf("Could not delete firewall PREROUTING rule %s to %s into chain ciao-floating-ip-pre", pubIP, intIP)
		}
	}

	// delete SNAT POSTROUTING rule
	// iptables -t nat -D ciao-floating-ip-post -s <intIP> -j SNAT --to-source <pubIP>
	ok, err = ipt.Exists("nat", "ciao-floating-ip-post", "-s", intIP+"/32", "-j", "SNAT", "--to-source", pubIP)
	if err != nil {
		return fmt.Errorf("Error: InitFirewall could not verify existence of POSTROUTING rule %s to %s", intIP, pubIP)
	}

	if ok {
		err := ipt.Delete("nat", "ciao-floating-ip-post", "-s", intIP+"/32", "-j", "SNAT", "--to-source", pubIP)
		if err != nil {
			return fmt.Errorf("Could not delete firewall POSTROUTING rule %s to %s into chain ciao-floating-ip-post", intIP, pubIP)
		}
	}

	return nil
}

//DumpIPTables provides a utility routine that returns
//the current state of the iptables
func DumpIPTables() string {
	table, err := exec.Command("iptables-save").CombinedOutput()
	if err != nil {
		return fmt.Sprintf("unable to iptables-save %v", err)
	}
	return fmt.Sprintf("iptables-save =[%s]", string(table))
}

//DebugSSHPortForIP provides a utility routine that returns
//the ssh port on the tenant CNCI that can be used to reach
//a tenant instance with a given IP address
func DebugSSHPortForIP(ip net.IP) (int, error) {
	const natOffset = 33000

	extPort := int(natOffset) + (int(ip[2]) << 8) + int(ip[3])
	if extPort >= int(65535) {
		return -1, fmt.Errorf("invalid IP %s", ip)
	}

	return extPort, nil
}
