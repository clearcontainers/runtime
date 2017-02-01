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

#include "spec_handler.h"
#include "util.h"
#include "../src/oci-config.h"

static struct oci_cfg_hook* current_hook = NULL;
static bool error_detected = false;

static void
save_current_hook(GSList** hooks_list) {
	if (! current_hook) {
		return;
	}

	/* Required:
	* - path
	* Optional:
	* - args
	* - env
	* - timeout
	*/
	if (! current_hook->path[0]) {
		g_critical("missing hook path");
		goto err;
	}

	*hooks_list = g_slist_append((*hooks_list), current_hook);
	current_hook = NULL;
	return;

err:
	cc_oci_hook_free(current_hook);
	error_detected = true;
	current_hook = NULL;
}

static void
handle_hook(GNode* root, GSList** list) {
	gchar* endptr = NULL;

	if ((!root) || error_detected) {
		return;
	}
	/* null separator */
	if(!root->data) {
		/* save current hook */
		save_current_hook(list);
	} else if (root->children) {
		/* create a new hook and fill it */
		if (!current_hook) {
			current_hook = g_new0 (struct oci_cfg_hook, 1);
		}

		if (g_strcmp0(root->data, "path") == 0) {
			g_snprintf(current_hook->path, sizeof(current_hook->path),
			           "%s", (char*)root->children->data);
		} else if (g_strcmp0(root->data, "args") == 0) {
			current_hook->args = node_to_strv(root);
		} else if (g_strcmp0(root->data, "env") == 0) {
			current_hook->env = node_to_strv(root);
		} else if (g_strcmp0(root->data, "timeout") == 0) {
			current_hook->timeout =
			    (gint)g_ascii_strtoll((char*)root->children->data, &endptr, 10);
			if (endptr == root->children->data) {
				g_critical("failed to convert '%s' to int",
				    (char*)root->children->data);
			}
		}
	}
}

static void
handle_hooks_section(GNode* root, struct cc_oci_config* config) {
	GSList** current_list = NULL;

	if (! (root && root->children) || error_detected) {
		return;
	}
	if (g_strcmp0(root->data, "prestart") == 0) {
		current_list = &(config->oci.hooks.prestart);
	} else if (g_strcmp0(root->data, "poststart") == 0) {
		current_list = &(config->oci.hooks.poststart);
	} else if (g_strcmp0(root->data, "poststop") == 0) {
		current_list = &config->oci.hooks.poststop;
	} else {
		g_critical("Unknown hook: %s", (char*)root->data);
		return;
	}

	g_node_children_foreach(root, G_TRAVERSE_ALL,
			(GNodeForeachFunc)handle_hook, current_list);

	/* save last hook */
	save_current_hook(current_list);

}

static bool
hooks_handle_section(GNode* root, struct cc_oci_config* config) {
	error_detected = false;

	if (! root) {
		g_critical("root node is NULL");
		return false;
	}

	if (! config ) {
		g_critical("oci config is NULL");
		return false;
	}

	g_node_children_foreach(root, G_TRAVERSE_ALL,
		(GNodeForeachFunc)handle_hooks_section, config);

	return !error_detected;
}

struct spec_handler hooks_spec_handler = {
	.name = "hooks",
	.handle_section = hooks_handle_section
};
