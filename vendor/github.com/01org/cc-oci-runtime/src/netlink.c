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
 */

/** \file
 *
 * Networking netlink routines, used to setup networking.
 *
 */

#include <errno.h>
#include <string.h>

#include <glib.h>
#include <glib/gprintf.h>

#include <libmnl/libmnl.h>
#include <net/if.h>
#include <linux/if_link.h>
#include <linux/rtnetlink.h>

#include "netlink.h"
#include "util.h"
#include "networking.h"

/*!
 * Setup the netlink socket to use with netlink
 * transactions.
 * This handle should be used for all netlink transactions
 * for a given thread.
 *
 * \warning The handle is not thread safe.
 *
 * \return \c handle to netlink on success, else \c NULL.
 */
struct netlink_handle *
netlink_init(void) {
	struct netlink_handle *hndl = g_malloc0(sizeof(*hndl));

	hndl->nl = mnl_socket_open(NETLINK_ROUTE);
	if (hndl->nl == NULL) {
		g_critical("mnl_socket_open %s", strerror(errno));
		goto out;
	}

	if (mnl_socket_bind(hndl->nl, 0, MNL_SOCKET_AUTOPID) < 0) {
		g_critical("mnl_socket_bind %s", strerror(errno));
		netlink_close(hndl);
		goto out;
	}

	hndl->seq = (guint)time(NULL);
	return hndl;

out:
	g_free(hndl);
	return NULL;
}

/*!
 * Close the netlink connection
 *
 * \param hndl handle returned from a call to \ref netlink_init().
 */
void
netlink_close(struct netlink_handle *const hndl) {
	if (hndl == NULL) {
		return;
	}

	if (mnl_socket_close(hndl->nl) == -1) {
		g_critical("mnl_socket_close %s", strerror(errno));
	}
}

/*!
 * Execute a netlink transaction and check the result.
 *
 * This method can be used for any transaction where
 * only the success or failure of the transaction needs
 * to be known (i.e. no data is send back).
 *
 * \param hndl handle returned from a call to \ref netlink_init().
 * \param nlh pre created netlink message.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
netlink_execute(struct netlink_handle *const hndl,
		struct nlmsghdr *const nlh) {
	guint8 buf[MNL_SOCKET_BUFFER_SIZE];
	ssize_t ret = -1;
	gboolean status = false;
	guint  portid, seq;

	if ((hndl == NULL) || (nlh == NULL)) {
		g_critical("%s NULL parameter", __func__);
		return false;
	}

	if (hndl->nl == NULL) {
		g_critical("%s NULL parameter", __func__);
		return false;
	}

	nlh->nlmsg_seq = seq = hndl->seq++;
	portid = mnl_socket_get_portid(hndl->nl);

	if (mnl_socket_sendto(hndl->nl, nlh, nlh->nlmsg_len) < 0) {
		g_critical("mnl_socket_sendto %s", strerror(errno));
		goto out;
	}

	ret = mnl_socket_recvfrom(hndl->nl, buf, sizeof(buf));
	if (ret == -1) {
		g_critical("mnl_socket_recvfrom failed %s", strerror(errno));
		goto out;
	}

	while (ret > 0) {
		ret = mnl_cb_run(buf, (size_t)ret, seq, portid, NULL, NULL);
		if (ret <= 0) {
			break;
		}
		ret = mnl_socket_recvfrom(hndl->nl, buf, sizeof(buf));
	}

	if (ret == -1) {
		g_critical("netlink error");
		goto out;
	}

	status = true;
out:
	return status;
}

/*!
 * Netlink command equivalent to
 * "ip link set dev \<interface\> \<up|down\>".
 *
 * \param hndl handle returned from a call to \ref netlink_init().
 * \param interface device name to enable/disable.
 * \param enable if \c true device will enabled, else disabled.
 *
 * \return \c true on success, else \c false.
 */
gboolean
netlink_link_enable(struct netlink_handle *const hndl,
		    const gchar *const interface, gboolean enable)  {
	guint8 buf[MNL_SOCKET_BUFFER_SIZE];
	struct nlmsghdr *nlh = NULL;
	struct ifinfomsg *ifm = NULL;
	guint change = 0, flags = 0;

	if ((hndl == NULL) || (interface == NULL)) {
		g_critical("%s NULL parameter", __func__);
		return false;
	}

	g_debug("netlink_link_enable[%d] %s", enable, interface);

	if (enable) {
		change |= IFF_UP;
		flags |= IFF_UP;
	} else {
		change |= IFF_UP;
		flags &= (guint)~IFF_UP;
	}

	nlh = mnl_nlmsg_put_header(buf);
	nlh->nlmsg_type = RTM_NEWLINK;
	nlh->nlmsg_flags = NLM_F_REQUEST | NLM_F_ACK;
	ifm = mnl_nlmsg_put_extra_header(nlh, sizeof(*ifm));
	ifm->ifi_family = AF_UNSPEC;
	ifm->ifi_change = change;
	ifm->ifi_flags = flags;

	mnl_attr_put_str(nlh, IFLA_IFNAME, interface);

	return netlink_execute(hndl, nlh);
}

/*!
 * Netlink command equivalent to
 * "ip link add name ${bridge name} type bridge".
 *
 * \param hndl handle returned from a call to \ref netlink_init().
 * \param name of the bridge to create.
 *
 * \return \c true on success, else \c false.
 */
gboolean
netlink_link_add_bridge(struct netlink_handle *const hndl,
			const gchar *const name)  {
	guint8 buf[MNL_SOCKET_BUFFER_SIZE];
	struct nlmsghdr *nlh = NULL;
	struct ifinfomsg *ifm = NULL;
	struct nlattr* link_attr = NULL;

	if ((hndl == NULL) || (name == NULL)) {
		g_critical("%s NULL parameter", __func__);
		return false;
	}

	g_debug("netlink_link_add_bridge %s", name);

	nlh = mnl_nlmsg_put_header(buf);
	nlh->nlmsg_type = RTM_NEWLINK;
	nlh->nlmsg_flags = NLM_F_REQUEST | NLM_F_CREATE | NLM_F_EXCL | NLM_F_ACK;
	ifm = mnl_nlmsg_put_extra_header(nlh, sizeof(*ifm));
	ifm->ifi_family = AF_UNSPEC;

	mnl_attr_put_str(nlh, IFLA_IFNAME, name);
	link_attr = mnl_attr_nest_start(nlh, IFLA_LINKINFO);
	mnl_attr_put_str(nlh, IFLA_INFO_KIND, "bridge");
	mnl_attr_nest_end(nlh, link_attr);

	return netlink_execute(hndl, nlh);
}

/*!
 * Netlink command equivalent to
 * "ip link set dev ${interface name} master ${bridge name}".
 *
 * \param hndl handle returned from a call to \ref netlink_init().
 * \param dev index of the device to add to the bridge.
 * \param master index of the bridge.
 *
 * \return \c true on success, else \c false.
 */
gboolean
netlink_link_set_master(struct netlink_handle *const hndl,
			guint dev, guint master)  {
	guint8 buf[MNL_SOCKET_BUFFER_SIZE];
	struct nlmsghdr *nlh = NULL;
	struct ifinfomsg *ifm = NULL;

	if (hndl == NULL) {
		g_critical("%s NULL parameter", __func__);
		return false;
	}

	g_debug("netlink_link_set_master %d %d", dev, master);

	nlh = mnl_nlmsg_put_header(buf);
	nlh->nlmsg_type = RTM_SETLINK;
	nlh->nlmsg_flags = NLM_F_REQUEST | NLM_F_ACK;
	ifm = mnl_nlmsg_put_extra_header(nlh, sizeof(*ifm));
	ifm->ifi_family = AF_UNSPEC;
	ifm->ifi_index = (gint)dev;

	mnl_attr_put_u32(nlh, IFLA_MASTER, master);

	return netlink_execute(hndl, nlh);
}

/*!
 * Netlink command equivalent to
 * "ip link set dev ${interface name} address ${address}".
 *
 * \param hndl handle returned from a call to \ref netlink_init().
 * \param interface name of the device.
 * \param size size of the address in bytes.
 * \param hwaddr link layer address of the device.
 *
 * \return \c true on success, else \c false.
 */
gboolean
netlink_link_set_addr(struct netlink_handle *const hndl,
		      const gchar *const interface, gulong size,
		      const guint8 *const hwaddr)  {
	guint8 buf[MNL_SOCKET_BUFFER_SIZE];
	struct nlmsghdr *nlh = NULL;
	struct ifinfomsg *ifm = NULL;

	if ((hndl == NULL) || (interface == NULL) || (hwaddr == NULL)) {
		g_critical("%s NULL parameter", __func__);
		return false;
	}

	if (!(size > 0)) {
		g_critical("%s size: invalid parameter", __func__);
		return false;
	}

	g_debug("netlink_link_set_addr %s", interface);
	if (size == 6) {
		g_debug("macaddr %.2x:%.2x:%.2x:%.2x:%.2x:%.2x",
				hwaddr[0], hwaddr[1], hwaddr[2],
				hwaddr[3], hwaddr[4], hwaddr[5]);
	}

	nlh = mnl_nlmsg_put_header(buf);
	nlh->nlmsg_type = RTM_SETLINK;
	nlh->nlmsg_flags = NLM_F_REQUEST | NLM_F_ACK;
	ifm = mnl_nlmsg_put_extra_header(nlh, sizeof(*ifm));
	ifm->ifi_family = AF_UNSPEC;

	mnl_attr_put_str(nlh, IFLA_IFNAME, interface);
	mnl_attr_put(nlh, IFLA_ADDRESS, size, hwaddr);

	return netlink_execute(hndl, nlh);
}

/*!
 * Callback handler that parses the netlink message
 * and populate the fields obtained from the message.
 *
 * \param attr the netlink attribute to parse.
 * \param data [in, out] table of parsed netlink attributes.
 *
 * \return \c MNL_CB_OK on success, \c MNL_CB_ERROR on error.
 */
static gint
data_ipv4_attr_cb(const struct nlattr *attr, void *data)
{
	const struct nlattr **tb = data;
	gint type = 0;

	if ((attr == NULL) || (data == NULL)) {
		g_critical("%s NULL parameter", __func__);
		return false;
	}

	type = mnl_attr_get_type(attr);

	/* skip unsupported attribute in user-space */
	if (mnl_attr_type_valid(attr, RTA_MAX) < 0) {
		return MNL_CB_OK;
	}

	switch(type) {
	case RTA_TABLE:
	case RTA_DST:
	case RTA_SRC:
	case RTA_OIF:
	case RTA_FLOW:
	case RTA_PREFSRC:
	case RTA_GATEWAY:
	case RTA_PRIORITY:
		if (mnl_attr_validate(attr, MNL_TYPE_U32) < 0) {
			g_critical("mnl_attr_validate %s", strerror(errno));
			return MNL_CB_ERROR;
		}
		break;
	case RTA_METRICS:
		if (mnl_attr_validate(attr, MNL_TYPE_NESTED) < 0) {
			g_critical("mnl_attr_validate %s", strerror(errno));
			return MNL_CB_ERROR;
		}
		break;
	}
	tb[type] = attr;
	return MNL_CB_OK;
}


/*!
 * Callback handler that scans the route table to
 * detect routes and add them.
 *
 * \param nlh netlink response buffer.
 * \param data [in, out] is the data sent to the callback handler
 *   from caller data points to cc_oci_config object to which routes
 *   are added
 *
 * \return \c MNL_CB_OK on success.
 */
static gint
process_ipv4_routes(const struct nlmsghdr *nlh, void *data)
{
	struct nlattr *tb[RTA_MAX+1] = {0};
	struct rtmsg *rm = NULL;
	gint ret;
	struct cc_oci_net_ipv4_route *route = NULL;
	struct cc_oci_net_cfg *net = NULL;
	uint32_t table;

	struct cc_oci_config *config = data;

	if ((nlh == NULL) || (data == NULL)) {
		g_critical("%s NULL parameter", __func__);
		return false;
	}

	rm = mnl_nlmsg_get_payload(nlh);

	if (rm->rtm_family != AF_INET) {
		g_debug("unexpected family %d", rm->rtm_family);
		return MNL_CB_OK;
	}

	ret = mnl_attr_parse(nlh, sizeof(*rm), data_ipv4_attr_cb, tb);
	if (ret != MNL_CB_OK) {
		return ret;
	}

	net = &(config->net);

	if (!tb[RTA_TABLE]) {
		g_debug("route table not set");
		return MNL_CB_OK;
	}

	table = mnl_attr_get_u32(tb[RTA_TABLE]);
	g_debug("table=%d", table);

	// Add routes from the main routing table. Ignore routes
	// from the local route table.
	if (table != RT_TABLE_MAIN) {
		return MNL_CB_OK;
	}

	route = g_malloc0(sizeof(*route));
	net->routes = g_slist_append(net->routes, route);

	if (tb[RTA_DST]) {
		struct in_addr *addr = mnl_attr_get_payload(tb[RTA_DST]);
		route->dest = cc_net_get_ip_address(AF_INET, addr);
		g_debug("destination : %s", route->dest);
	}

	if (rm->rtm_src_len == 0 && rm->rtm_dst_len == 0) {
		// hyperstart expects the string "default" or "any" for default route
		route->dest = g_strdup("default");
	}

	if (tb[RTA_GATEWAY]) {
		struct in_addr *addr = mnl_attr_get_payload(tb[RTA_GATEWAY]);
		route->gateway = cc_net_get_ip_address(AF_INET, addr);
		g_debug("gateway : %s", route->gateway);
	}

	if (tb[RTA_OIF]) {
		uint ifindex = mnl_attr_get_u32(tb[RTA_OIF]);
		char ifname[IF_NAMESIZE];
		if (if_indextoname(ifindex, ifname)) {
			route->ifname = g_strdup(ifname);
			g_debug("ifname=%s", ifname);
		}
	}
	return MNL_CB_OK;
}

/*!
 * Scans the route table for the specified inet family and adds
 * the routes to the config object.
 *
 * \param config \ref cc_oci_config.
 * \param hndl handle returned from a call to \ref netlink_init().
 * \param family INET family.
 *
 * \return true on success, false otherwise.
 */
gboolean
netlink_get_routes(struct cc_oci_config *config,
		struct netlink_handle *const hndl,
		guchar family)
{
	guint8 buf[MNL_SOCKET_BUFFER_SIZE];
	struct nlmsghdr *nlh = NULL;
	struct rtmsg *rtm = NULL;
	glong ret;
	guint seq, portid;

	if ( ! (hndl && config)) {
		g_critical("%s NULL parameter", __func__);
		return false;
	}

	g_debug("netlink_get_default_gw");

	nlh = mnl_nlmsg_put_header(buf);
	nlh->nlmsg_type = RTM_GETROUTE;
	nlh->nlmsg_flags = NLM_F_REQUEST | NLM_F_DUMP;
	nlh->nlmsg_seq = seq = (guint)time(NULL);
	rtm = mnl_nlmsg_put_extra_header(nlh, sizeof(struct rtmsg));

	rtm->rtm_family = family;

	portid = mnl_socket_get_portid(hndl->nl);

	if (mnl_socket_sendto(hndl->nl, nlh, nlh->nlmsg_len) < 0) {
		g_critical("mnl_socket_sendto %s", strerror(errno));
		return false;
	}

	ret = mnl_socket_recvfrom(hndl->nl, buf, sizeof(buf));
	while (ret > 0) {
		ret = mnl_cb_run(buf, (size_t)ret, seq, portid,
				 process_ipv4_routes, config);

		if (ret <= MNL_CB_STOP) {
			break;
		}
		ret = mnl_socket_recvfrom(hndl->nl, buf, sizeof(buf));
	}
	if (ret == -1) {
		g_critical("mnl_socket_recvfrom %s", strerror(errno));
		return false;
	}
	return true;
}
