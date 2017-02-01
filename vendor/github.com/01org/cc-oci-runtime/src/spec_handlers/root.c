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

#include <stdio.h>
#include <sys/stat.h>

#include "spec_handler.h"
#include "util.h"

static void
handle_root_section(GNode* root, struct cc_oci_config* config) {
	if (! (root && root->children)) {
		return;
	}
	if (g_strcmp0(root->data, "path") == 0) {
		g_autofree gchar *full = cc_oci_resolve_path ((char*)root->children->data);
		if (full) {
			g_snprintf (config->oci.root.path,
					sizeof(config->oci.root.path),
					"%s", full);
		}
	} else if (g_strcmp0(root->data, "readonly") == 0) {
		if (g_strcmp0(root->children->data, "true") == 0) {
			config->oci.root.read_only = true;
		} else if (g_strcmp0(root->children->data, "false") == 0) {
			config->oci.root.read_only = false;
		} else {
			g_critical("readonly unknown type");
		}
	}
}

static bool
root_handle_section(GNode* root, struct cc_oci_config* config) {
	gboolean ret = false;

	if (! root) {
		g_critical("root node is NULL");
		goto out;
	}

	if (! config ) {
		g_critical("oci config is NULL");
		goto out;
	}

	g_node_children_foreach(root, G_TRAVERSE_ALL,
		(GNodeForeachFunc)handle_root_section, config);

	/* Required:
	* - path
	* Optional:
	* - readonly
	*/
	if (! config->oci.root.path[0]) {
		g_critical("missing root path");
		goto out;
	}
	if (! g_file_test (config->oci.root.path,
				 G_FILE_TEST_IS_DIR)) {
		g_critical ("rootfs not a directory: %s",
				config->oci.root.path);
		goto out;
	}
	ret = true;

out:
	return ret;
}

struct spec_handler root_spec_handler = {
	.name = "root",
	.handle_section = root_handle_section
};
