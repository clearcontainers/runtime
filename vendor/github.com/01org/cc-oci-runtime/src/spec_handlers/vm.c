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
#include "oci.h"
#include "util.h"

static void
handle_kernel_section(GNode* root, struct cc_oci_config* config) {
	if (! (root && root->children)) {
		return;
	}
	if (g_strcmp0(root->data, "path") == 0) {
		g_autofree gchar* path = cc_oci_resolve_path(root->children->data);
		if (path) {
			if (snprintf(config->vm->kernel_path,
			    sizeof(config->vm->kernel_path),
			    "%s", path) < 0) {
				g_critical("failed to copy vm kernel path");
			}
		}
	} else if (g_strcmp0(root->data, "parameters") == 0) {
		config->vm->kernel_params = g_strdup(root->children->data);
	}
}

static void
handle_vm_section(GNode* root, struct cc_oci_config* config) {
	if (! (root && root->children)) {
		return;
	}
	if (g_strcmp0(root->data, "path") == 0) {
		g_autofree gchar* path = cc_oci_resolve_path(root->children->data);
		if (path) {
			if (snprintf(config->vm->hypervisor_path,
			    sizeof(config->vm->hypervisor_path),
			    "%s", path) < 0) {
				g_critical("failed to copy vm hypervisor path");
			}
		}

	} else if(g_strcmp0(root->data, "image") == 0) {
		g_autofree gchar* path = cc_oci_resolve_path(root->children->data);
		if (path) {
			if (snprintf(config->vm->image_path,
			    sizeof(config->vm->image_path),
			    "%s", path) < 0) {
				g_critical("failed to copy vm image path");
			}
		}
	} else if (g_strcmp0(root->data, "kernel") == 0) {
		g_node_children_foreach(root, G_TRAVERSE_ALL,
			(GNodeForeachFunc)handle_kernel_section, config);
	}
}

static bool
vm_handle_section(GNode* root, struct cc_oci_config* config) {
	gboolean ret = false;
	struct stat st;

	if (! root) {
		g_critical("root node is NULL");
		return false;
	}

	if (! config ) {
		g_critical("oci config is NULL");
		return false;
	}

	if(! config->vm) {
		config->vm = g_malloc0(sizeof(struct cc_oci_vm_cfg));
	}

	g_node_children_foreach(root, G_TRAVERSE_ALL,
		(GNodeForeachFunc)handle_vm_section, config);

	/* Needs:
	* - hypervisor_path
	* - image_path
	* - kernel_path
	* Optional:
	* - kernel_params
	*/

	if (! config->vm->hypervisor_path[0]
	    || stat (config->vm->hypervisor_path, &st) < 0) {
		g_critical("VM hypervisor path does not exist");
		goto out;
	}

	if (! config->vm->image_path[0]
	    || stat (config->vm->image_path, &st) < 0) {
		g_critical("VM image path does not exist");
		goto out;
	}

	if (! config->vm->kernel_path[0]
	    || stat (config->vm->kernel_path, &st) < 0) {
		g_critical("VM kernel path does not exist");
		goto out;
	}

	ret = true;

out:
	if (! ret) {
		g_free_if_set (config->vm->kernel_params);
		g_free (config->vm);
		config->vm = NULL;
	}

	return ret;
}

struct spec_handler vm_spec_handler = {
	.name = "vm",
	.handle_section = vm_handle_section
};
