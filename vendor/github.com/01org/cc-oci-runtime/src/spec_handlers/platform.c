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

static void
handle_platform_section(GNode* root, struct cc_oci_config* config) {
	if (! (root && root->children)) {
		return;
	}
	if (g_strcmp0(root->data, "os") == 0) {
		config->oci.platform.os = g_strdup(root->children->data);
	} else if(g_strcmp0(root->data, "arch") == 0) {
		config->oci.platform.arch = g_strdup(root->children->data);
	}
}

static bool
platform_handle_section(GNode* root, struct cc_oci_config* config) {
	gboolean  ret = false;

	if (! root) {
		g_critical("root node is NULL");
		return false;
	}

	if (! config ) {
		g_critical("oci config is NULL");
		return false;
	}

	g_node_children_foreach(root, G_TRAVERSE_ALL,
		(GNodeForeachFunc)handle_platform_section, config);

	if (! config->oci.platform.os) {
		g_critical ("no os found");
		return false;
	}

	ret = g_strcmp0 (config->oci.platform.os,
			CC_OCI_EXPECTED_PLATFORM);
	if (ret) {
		g_critical ("unexpected os: got '%s', expected '%s'",
				config->oci.platform.os,
				CC_OCI_EXPECTED_PLATFORM);
		return false;
	}

	if (! config->oci.platform.arch) {
		g_critical ("no architecture found");
		return false;
	}

	ret = g_strcmp0 (config->oci.platform.arch,
			CC_OCI_EXPECTED_ARCHITECTURE);
	if (ret) {
		g_critical ("unexpected architecture: "
				"got '%s', expected '%s'",
				config->oci.platform.arch,
				CC_OCI_EXPECTED_ARCHITECTURE);
		return false;
	}

	return true;
}

struct spec_handler platform_spec_handler = {
	.name = "platform",
	.handle_section = platform_handle_section
};
