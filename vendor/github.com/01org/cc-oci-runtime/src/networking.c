/*
 * This file is part of cc-oci-runtime.
 *
 * Copyright (C) 2016 Intel Corporation
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; either version 2
 * of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
 *
 * This file incorporates work covered by the following copyright and
 * permission notice:
 *
 *	Copyright (c) 1983, 1993
 *		The Regents of the University of California.  All rights reserved.
 *
 *	Redistribution and use in source and binary forms, with or without
 *	modification, are permitted provided that the following conditions
 *	are met:
 *	1. Redistributions of source code must retain the above copyright
 *	notice, this list of conditions and the following disclaimer.
 *	2. Redistributions in binary form must reproduce the above copyright
 *	notice, this list of conditions and the following disclaimer in the
 *	documentation and/or other materials provided with the distribution.
 *	4. Neither the name of the University nor the names of its contributors
 *	may be used to endorse or promote products derived from this software
 *	without specific prior written permission.
 *
 *	THIS SOFTWARE IS PROVIDED BY THE REGENTS AND CONTRIBUTORS ``AS IS'' AND
 *	ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 *	IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 *	ARE DISCLAIMED.  IN NO EVENT SHALL THE REGENTS OR CONTRIBUTORS BE LIABLE
 *	FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 *	DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 *	OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 *	HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 *	LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 *	OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 *	SUCH DAMAGE.
 *
 */

/** \file
 *
 * Networking routines, used to setup networking (currently docker specific).
 *
 */

#include <string.h>
#include <stdbool.h>
#include <signal.h>
#include <fcntl.h>
#include <errno.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <linux/if_tun.h>
#include <ifaddrs.h>
#include <arpa/inet.h>
#include <net/if_arp.h>
#include <net/if.h>

#include <glib.h>
#include <glib/gprintf.h>

#include "oci.h"
#include "util.h"
#include "netlink.h"

#define TUNDEV "/dev/net/tun"

/*!
 * Free the specified \ref cc_oci_net_ipv4_cfg.
 *
 * \param ipv4_cfg \ref cc_oci_net_ipv4_cfg.
 */
static void
cc_oci_net_ipv4_free (struct cc_oci_net_ipv4_cfg *ipv4_cfg)
{
	if (!ipv4_cfg) {
		return;
	}

	g_free_if_set (ipv4_cfg->ip_address);
	g_free_if_set (ipv4_cfg->subnet_mask);
	g_free (ipv4_cfg);
}

/*!
 * Free the specified \ref cc_oci_net_ipv6_cfg.
 *
 * \param ipv6_cfg \ref cc_oci_net_ipv6_cfg.
 */
static void
cc_oci_net_ipv6_free (struct cc_oci_net_ipv6_cfg *ipv6_cfg)
{
	if (!ipv6_cfg) {
		return;
	}

	g_free_if_set (ipv6_cfg->ipv6_address);
	g_free (ipv6_cfg);
}

/*!
 * Free the specified \ref cc_oci_net_if_cfg.
 *
 * \param if_cfg \ref cc_oci_net_if_cfg
 */
void
cc_oci_net_interface_free (struct cc_oci_net_if_cfg *if_cfg)
{
	if (!if_cfg) {
		return;
	}

	g_free_if_set (if_cfg->mac_address);
	g_free_if_set (if_cfg->ifname);
	g_free_if_set (if_cfg->bridge);
	g_free_if_set (if_cfg->tap_device);

	if (if_cfg->ipv4_addrs) {
		g_slist_free_full(if_cfg->ipv4_addrs,
                (GDestroyNotify)cc_oci_net_ipv4_free);
	}

	if (if_cfg->ipv6_addrs) {
		g_slist_free_full(if_cfg->ipv6_addrs,
                (GDestroyNotify)cc_oci_net_ipv6_free);
	}

	g_free (if_cfg);
}

/*!
 * Free the specified \ref cc_oci_net_ipv4_route.
 *
 * \param route \ref cc_oci_net_ipv4_route
 */
void cc_oci_net_ipv4_route_free(struct cc_oci_net_ipv4_route *route)
{
	if (! route) {
		return;
	}

	g_free_if_set (route->dest);
	g_free_if_set (route->ifname);
	g_free_if_set (route->gateway);

	g_free(route);
}

/*!
 * Request to create a named tap interface
 *
 * \param tap \c tap interface name to create
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_oci_tap_create(const gchar *const tap) {
	struct ifreq ifr;
	int fd = -1;
	gboolean ret = false;

	if (tap == NULL) {
		g_critical("invalid tap interface");
		goto out;
	}

	fd = open(TUNDEV, O_RDWR);
	if (fd < 0) {
		g_critical("Failed to open [%s] [%s]", TUNDEV, strerror(errno));
		goto out;
	}

	memset(&ifr, 0, sizeof(ifr));
	ifr.ifr_flags = IFF_TAP;
	g_strlcpy(ifr.ifr_name, tap, IFNAMSIZ);

	if (ioctl(fd, TUNSETIFF, (void *) &ifr) < 0) {
		g_critical("Failed to create tap [%s] [%s]",
			tap, strerror(errno));
		goto out;
	}

	if (ioctl(fd, TUNSETPERSIST, 1) < 0) {
		g_critical("failed to TUNSETPERSIST [%s] [%s]",
			tap, strerror(errno));
		goto out;
	}

	ret = true;
out:
	if (fd != -1) {
		close(fd);
	}

	return ret;
}

/*!
 * Request to create the networking framework
 * that will be used to connect the specified
 * container network (veth) to the VM
 *
 * The container may be associated with multiple
 * networks and function has to be invoked for
 * each of those networks
 *
 * Once the OCI spec supports the creation of
 * VM compatible tap interfaces in the network
 * plugin, this setup will not be required
 *
 * \param config \ref cc_oci_config.
 * \param hndl handle returned from a call to \ref netlink_init().
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_network_create(const struct cc_oci_config *const config,
		      struct netlink_handle *const hndl) {
	struct cc_oci_net_if_cfg *if_cfg = NULL;
	guint index = 0;

	if (config == NULL) {
		return false;
	}

	for (index=0; index<g_slist_length(config->net.interfaces); index++) {
		/* Each container has its own name space. Hence we use the
		 * same mac address prefix for tap interfaces on the host
		 * side. This method scales to support upto 2^16 networks
		 */
		guint8 mac[6] = {0x02, 0x00, 0xCA, 0xFE,
				(guint8)(index >> 8), (guint8)index};
		guint tap_index, veth_index, bridge_index;
		tap_index = veth_index = bridge_index = 0;

		if_cfg = (struct cc_oci_net_if_cfg *)
			g_slist_nth_data(config->net.interfaces, index);

		if (!cc_oci_tap_create(if_cfg->tap_device)) {
			goto out;
		}

		if (!netlink_link_add_bridge(hndl, if_cfg->bridge)) {
			goto out;
		}

		if (!netlink_link_set_addr(hndl, if_cfg->ifname,
					   sizeof(mac), mac)) {
			goto out;
		}

		bridge_index = if_nametoindex(if_cfg->bridge);
		tap_index = if_nametoindex(if_cfg->tap_device);
		veth_index = if_nametoindex(if_cfg->ifname);

		if (!netlink_link_set_master(hndl, tap_index, bridge_index)) {
			goto out;
		}
		if (!netlink_link_set_master(hndl, veth_index, bridge_index)) {
			goto out;
		}
		if (!netlink_link_enable(hndl, if_cfg->tap_device, true)) {
			goto out;
		}
		if (!netlink_link_enable(hndl, if_cfg->ifname, true)) {
			goto out;
		}
		if (!netlink_link_enable(hndl, if_cfg->bridge, true)) {
			goto out;
		}
	}

	return true;
out:
	return false;
}

/*!
 * Obtain the string representation of the inet address
 *
 * \param family \c inetfamily
 * \param sin_addr \c inet address
 *
 * \return \c string containing IP address, else \c ""
 */
gchar *
cc_net_get_ip_address(const gint family, const void *const sin_addr)
{
	gchar addrBuf[INET6_ADDRSTRLEN];

	if (!sin_addr) {
		return g_strdup("");
	}

	if (!inet_ntop(family, sin_addr, addrBuf, sizeof(addrBuf))) {
		g_critical("inet_ntop() failed with errno =  %d %s\n",
			errno, strerror(errno));
		return g_strdup("");
	}

	g_debug("IP := [%s]", addrBuf);
	return g_strdup(addrBuf);
}

/*!
 * Count the number of valid bits in the subnet prefix
 * Copied over from BSD
 *
 * \param val \c the subnet
 * \param size \c size of the subnet value in bytes
 *
 * \return \c prefix length if valid else \c 0
 */
static guint
prefix(guint8 *val, guint size)
{
        guint8 *addr = val;
        guint byte, bit, plen = 0;

        for (byte = 0; byte < size; byte++, plen += 8) {
                if (addr[byte] != 0xff) {
                        break;
		}
	}

	if (byte == size) {
		return (plen);
	}

	for (bit = 7; bit != 0; bit--, plen++) {
                if (!(addr[byte] & (1 << bit))) {
                        break;
		}
	}

	/* Handle errors */
        for (; bit != 0; bit--) {
                if (addr[byte] & (1 << bit)) {
                        return(0);
		}
	}
        byte++;
        for (; byte < size; byte++) {
                if (addr[byte]) {
                        return(0);
		}
	}
        return (plen);
}

/*!
 * Obtain the Subnet prefix from subnet address
 *
 * \param family \c inetfamily
 * \param sin_addr \c inet address
 *
 * \return \c string containing subnet prefix
 */
static gchar *
subnet_to_prefix(const gint family, const void *const sin_addr) {
	guint pfix = 0;
	guint8 *addr = (guint8 *)sin_addr;

	if (!addr) {
		return g_strdup_printf("%d", pfix);
	}

	switch(family) {
	case AF_INET6:
		pfix = prefix(addr, 16);
		break;
	case AF_INET:
		pfix = prefix(addr, 4);
		break;
	default:
		g_warning("invalid prefix family %d", family);
		pfix = 0;
	}

	return g_strdup_printf("%d", pfix);
}

/*!
 * Obtain the string representation of the mac address
 * of an interface
 *
 * \param ifname \c interface name
 *
 * \return \c string containing MAC address, else \c ""
 */
static gchar *
get_mac_address(const gchar *const ifname)
{
	struct ifreq ifr;
	int fd = -1;
	gchar *macaddr;
	guint8 *data;

	if (ifname == NULL) {
		g_critical("NULL interface name");
		return g_strdup("");
	}

	fd = socket(AF_INET, SOCK_DGRAM, IPPROTO_IP);
	if (fd < 0) {
		g_critical("socket() failed with errno =  %d %s\n",
			errno, strerror(errno));
		return g_strdup("");
	}

	memset(&ifr, 0, sizeof(ifr));
	g_strlcpy(ifr.ifr_name, ifname, IFNAMSIZ);

	if (ioctl(fd, SIOCGIFHWADDR, &ifr) < 0) {
		g_critical("ioctl() failed with errno =  %d %s\n",
			errno, strerror(errno));
		macaddr = g_strdup("");
		goto out;
	}

	if (ifr.ifr_hwaddr.sa_family != ARPHRD_ETHER) {
		g_critical("invalid interface  %s\n", ifname);
		macaddr = g_strdup("");
		goto out;
	}

	data = (guint8 *)ifr.ifr_hwaddr.sa_data;
	macaddr = g_strdup_printf("%.2x:%.2x:%.2x:%.2x:%.2x:%.2x",
				(guint8)data[0], (guint8)data[1],
				(guint8)data[2], (guint8)data[3],
				(guint8)data[4], (guint8)data[5]);

out:
	close(fd);
	return macaddr;
}

/*
 * Return the predicatable interface name based on pcie address
 * Reference:
 * https://www.freedesktop.org/wiki/Software/systemd/PredictableNetworkInterfaceNames/
 *
 * \param index Index into the array that was used for the qemu
 * pcie address option as well
 *
 * \return Newly allocated string
 */
gchar *
get_pcie_ifname(guint index)
{
	return g_strdup_printf("enp0s%d", index + PCI_OFFSET);
}

/*!
 * GCompareFunc for searching through the list 
 * of existing network interfaces
 *
 * \param[in] a \c a element from the list
 * \param[in] b \c a value to compare with
 *
 * \return negative value if a < b ; zero if a = b ; positive value if a > b
 */
gint static
compare_interface(gconstpointer a,
		  gconstpointer b) {
	const struct cc_oci_net_if_cfg *if_cfg = a;
	const gchar *ifname = b;

	return g_strcmp0(ifname, if_cfg->ifname);
}

/*!
 * Obtain the network configuration of the container
 * Currently done by by scanned the namespace
 * Ideally the OCI spec should be modified such that
 * these parameters are sent to the runtime
 *
 * \param[in,out] config \ref cc_oci_config.
 * \param hndl handle returned from a call to \ref netlink_init().
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_network_discover(struct cc_oci_config *const config,
			struct netlink_handle *hndl)
{
	struct ifaddrs *ifa = NULL;
	struct ifaddrs *ifaddrs = NULL;
	struct cc_oci_net_if_cfg *if_cfg = NULL;
	struct cc_oci_net_cfg *net = NULL;
	gint family;
	gchar *ifname;

	if (!config) {
		return false;
	}

	net = &(config->net);

	if (getifaddrs(&ifaddrs) == -1) {
		g_critical("getifaddrs() failed with errno =  %d %s\n",
			errno, strerror(errno));
		return false;
	}

	g_debug("Discovering container interfaces");

	/* For now add the interfaces with a valid IPv4 address */
	for (ifa = ifaddrs; ifa != NULL; ifa = ifa->ifa_next) {
		GSList *elem;

		if (!ifa->ifa_addr) {
			continue;
		}

		g_debug("Interface := [%s]", ifa->ifa_name);

		family = ifa->ifa_addr->sa_family;
		if (!((family == AF_INET) || (family == AF_INET6)))  {
			continue;
		}

		if (!g_strcmp0(ifa->ifa_name, "lo")) {
			continue;
		}

		ifname = ifa->ifa_name;
		elem = g_slist_find_custom(net->interfaces,
					   ifname,
					   compare_interface);

		if (elem == NULL) {
			if_cfg = g_malloc0(sizeof(*if_cfg));
			if_cfg->ifname = g_strdup(ifname);
			if_cfg->mac_address = get_mac_address(ifname);
			if_cfg->tap_device = g_strdup_printf("c%s", ifname);
			if_cfg->bridge = g_strdup_printf("b%s", ifname);
			net->interfaces = g_slist_append(net->interfaces, if_cfg);
		} else {
			if_cfg = (struct cc_oci_net_if_cfg  *) elem->data;
		}

		if (family == AF_INET) {
			struct cc_oci_net_ipv4_cfg *ipv4_cfg;
			ipv4_cfg = g_malloc0(sizeof(*ipv4_cfg));

			if_cfg->ipv4_addrs = g_slist_append(
				if_cfg->ipv4_addrs, ipv4_cfg);

			ipv4_cfg->ip_address = cc_net_get_ip_address(family,
				&((struct sockaddr_in *)ifa->ifa_addr)->sin_addr);
			ipv4_cfg->subnet_mask = cc_net_get_ip_address(family,
				&((struct sockaddr_in *)ifa->ifa_netmask)->sin_addr);


		} else if (family == AF_INET6) {
			struct cc_oci_net_ipv6_cfg *ipv6_cfg;
			ipv6_cfg = g_malloc0(sizeof(*ipv6_cfg));
			if_cfg->ipv6_addrs = g_slist_append(
				if_cfg->ipv6_addrs, ipv6_cfg);

			ipv6_cfg->ipv6_address = cc_net_get_ip_address(family,
				&((struct sockaddr_in6 *)ifa->ifa_addr)->sin6_addr);
			ipv6_cfg->ipv6_prefix = subnet_to_prefix(family,
				&((struct sockaddr_in6 *)ifa->ifa_netmask)->sin6_addr);
		}
	}

	freeifaddrs(ifaddrs);

	if (config->oci.hostname){
		net->hostname = g_strdup(config->oci.hostname);
	} else {
		net->hostname = g_strdup("");
	}

	netlink_get_routes(config, hndl, AF_INET);

	/* TODO: Need to see if this needed, does resolv.conf handle this */
	net->dns_ip1 = g_strdup("");
	net->dns_ip2 = g_strdup("");

	g_debug("[%d] networks discovered", g_slist_length(net->interfaces));

	return true;
}
