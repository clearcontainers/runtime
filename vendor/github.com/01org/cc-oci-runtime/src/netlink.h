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

#ifndef _CC_OCI_NETLINK_H
#define _CC_OCI_NETLINK_H

#include "oci.h"

struct netlink_handle {
	guint seq;
	struct mnl_socket *nl;
};

struct netlink_handle * netlink_init(void);

void netlink_close(struct netlink_handle *const hndl);

gboolean netlink_link_enable(struct netlink_handle *const hndl,
				const gchar *const interface, gboolean enable);

gboolean netlink_link_add_bridge(struct netlink_handle *const hndl,
				 const gchar *const name);

gboolean netlink_link_set_master(struct netlink_handle *const hndl,
				 guint dev, guint master);

gboolean netlink_link_set_addr(struct netlink_handle *const hndl,
			       const gchar *const interface, gulong size, 
			       const guchar *const hwaddr);

gboolean netlink_get_routes(struct cc_oci_config *config, 
				struct netlink_handle *const hndl,
				guchar family);

#endif /* _CC_OCI_NETLINK_H */
