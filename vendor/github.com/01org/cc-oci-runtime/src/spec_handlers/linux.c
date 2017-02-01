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

#include "spec_handler.h"
#include "namespace.h"
#include "util.h"

static struct oci_cfg_namespace *current_ns;
static bool error_detected = false;

/*!
 * Add \ref current_ns to the specified \c GSList.
 *
 * \param ns_list \c GSList of \ref oci_cfg_namespace's.
 */
static void
save_current_ns (GSList **ns_list)
{
	if (! current_ns) {
		return;
	}

	*ns_list = g_slist_append ((*ns_list), current_ns);
	current_ns = NULL;
}

static void
handle_namespaces_section (GNode *root, struct cc_oci_config *config)
{
	if ((! root) || error_detected) {
		return;
	}

	if (! root->data) {
		save_current_ns (&config->oci.oci_linux.namespaces);
	} else if (root->children) {
		/* create and populate a new ns */
		if (!current_ns) {
			current_ns = g_new0 (struct oci_cfg_namespace, 1);
		}

		if (! g_strcmp0 (root->data, "type")) {
			const char *type = (const gchar *)root->children->data;

			if (! type) {
				g_critical ("no namespace type specified");
				goto err;
			}

			current_ns->type = cc_oci_str_to_ns (type);
			if ((! type) || (! *type) || (current_ns->type == OCI_NS_INVALID)) {
				g_critical ("invalid namespace type: %s",
						type ? type : "");
				goto err;
			}
		} else if (! g_strcmp0 (root->data, "path")) {
			const char *path = (const gchar *)root->children->data;

			/* Note that 'path' can also be "" or absent
			 * (denoting that a new namespace should be
			 * created.
			 */
			if (path && *path) {
				current_ns->path = g_strdup (path);
			}
		}
	}

	return;

err:
	cc_oci_ns_free (current_ns);
	error_detected = true;
	current_ns = NULL;
}

static void
handle_linux_section (GNode *root, struct cc_oci_config *config)
{
	if (! (root && root->children)) {
		return;
	}

	if (! g_strcmp0 (root->data, "namespaces")) {
		g_node_children_foreach(root, G_TRAVERSE_ALL,
			(GNodeForeachFunc)handle_namespaces_section,
			config);
	}
}

static bool
linux_handle_section (GNode *root, struct cc_oci_config *config)
{
	gboolean ret = false;

	error_detected = false;

	if (! root) {
		g_critical ("root node is NULL");
		goto out;
	}

	if (! config) {
		g_critical ("oci config is NULL");
		goto out;
	}

	g_node_children_foreach (root, G_TRAVERSE_ALL,
		(GNodeForeachFunc)handle_linux_section, config);

	if (! error_detected) {
		save_current_ns (&config->oci.oci_linux.namespaces);
	}

	ret = ! error_detected;

out:
	return ret;
}

struct spec_handler linux_spec_handler = {
	.name           = "linux",
	.handle_section = linux_handle_section
};
