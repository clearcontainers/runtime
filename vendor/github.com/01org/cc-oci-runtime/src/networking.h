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

#ifndef _CC_OCI_NETWORKING_H
#define _CC_OCI_NETWORKING_H

#include "netlink.h"

void cc_oci_net_interface_free (struct cc_oci_net_if_cfg *if_cfg);

void cc_oci_net_ipv4_route_free(struct cc_oci_net_ipv4_route *route);

gboolean cc_oci_network_create(const struct cc_oci_config *const config,
		      struct netlink_handle *hndl);

gchar * cc_net_get_ip_address(const gint family, const void *const sin_addr);


gboolean cc_oci_network_discover(struct cc_oci_config *const config,
			struct netlink_handle *hndl);

gchar* get_pcie_ifname(guint index);

#endif /* _CC_OCI_NETWORKING_H */
