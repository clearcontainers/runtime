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

#include <string.h>
#include <stdbool.h>

#include <glib.h>
#include <json-glib/json-glib.h>
#include <json-glib/json-gobject.h>

#include "util.h"
#include "mount.h"
#include "annotation.h"
#include "namespace.h"
#include "oci.h"
#include "semver.h"
#include "oci-config.h"
#include "networking.h"
#include "proxy.h"

/*!
 * Free all resources associated with \p h hook object.
 *
 * \param h \ref oci_cfg_hook.
 */
void
cc_oci_hook_free (struct oci_cfg_hook *h) {
	if (h) {
		if (h->args) {
			g_strfreev(h->args);
		}
		if (h->env) {
			g_strfreev(h->env);
		}
		g_free(h);
	}
}

/*!
 * Perform checks on the specified config.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_config_check (const struct cc_oci_config *config)
{
	gint  ret;

	if (! config) {
		return false;
	}

	if (! config->oci.oci_version) {
		g_critical ("no OCI version specified");
		return false;
	}

	g_debug ("OCI spec versions: config=%s, runtime=%s",
			config->oci.oci_version,
			CC_OCI_SUPPORTED_SPEC_VERSION);

	ret = cc_oci_semver_cmp (CC_OCI_SUPPORTED_SPEC_VERSION,
			config->oci.oci_version);

	if (ret < 0) {
		g_critical ("cannot handle config version %s",
				config->oci.oci_version);
		return false;
	}

	return true;
}

/*!
 * Get the path of \ref CC_OCI_CONFIG_FILE.
 *
 * \param bundle_path Full path to containers bundle path.
 *
 * \return Newly-allocated path string on success, else \c NULL.
 */
gchar *
cc_oci_config_file_path (const gchar *bundle_path)
{
	if (! bundle_path) {
		return NULL;
	}

	return cc_oci_get_bundlepath_file (bundle_path,
			CC_OCI_CONFIG_FILE);
}

/*!
 * Create a new \ref cc_oci_config object.
 *
 * \return Newly-allocated \ref cc_oci_config on success,
 * else \c NULL.
 */
struct cc_oci_config *
cc_oci_config_create (void)
{
	struct cc_oci_config *config;

	config = g_malloc0 (sizeof (struct cc_oci_config));
	if (! config) {
		return NULL;
	}

	config->proxy = g_malloc0 (sizeof (struct cc_proxy));
	if (! config->proxy) {
		g_free (config);
		return NULL;
	}

	return config;
}

/*!
 * Free all resources associated with the \p config object.
 *
 * \param config \ref cc_oci_config.
 */
void
cc_oci_config_free (struct cc_oci_config *config)
{
	if (! config) {
		return;
	}

	g_free_if_set (config->oci.oci_version);
	g_free_if_set (config->oci.hostname);
	g_free_if_set (config->console);
	g_free_if_set (config->bundle_path);
	g_free_if_set (config->root_dir);
	g_free_if_set (config->pid_file);

	if (config->vm) {
		g_free_if_set (config->vm->kernel_params);
		g_free (config->vm);
	}

	if (config->pod) {
		g_free_if_set (config->pod->sandbox_name);
		g_free (config->pod);
	}

	if (config->oci.process.args) {
		g_strfreev (config->oci.process.args);
	}

	if (config->oci.process.env) {
		g_strfreev (config->oci.process.env);
	}

	cc_oci_mounts_free_all (config->oci.mounts);
	cc_oci_annotations_free_all (config->oci.annotations);

	g_free_if_set (config->oci.platform.os);
	g_free_if_set (config->oci.platform.arch);

	if (config->oci.hooks.prestart) {
		g_slist_free_full(config->oci.hooks.prestart,
                (GDestroyNotify)cc_oci_hook_free);
	}
	if (config->oci.hooks.poststart) {
		g_slist_free_full(config->oci.hooks.poststart,
                (GDestroyNotify)cc_oci_hook_free);
	}
	if (config->oci.hooks.poststop) {
		g_slist_free_full(config->oci.hooks.poststop,
                (GDestroyNotify)cc_oci_hook_free);
	}
	if (config->oci.oci_linux.namespaces) {
		g_slist_free_full(config->oci.oci_linux.namespaces,
                (GDestroyNotify)cc_oci_ns_free);
	}

	g_free_if_set (config->net.hostname);
	g_free_if_set (config->net.dns_ip1);
	g_free_if_set (config->net.dns_ip2);

	if (config->net.interfaces) {
		g_slist_free_full(config->net.interfaces,
                (GDestroyNotify)cc_oci_net_interface_free);
	}

	if (config->net.routes) {
		g_slist_free_full(config->net.routes,
                (GDestroyNotify)cc_oci_net_ipv4_route_free);
	}

	cc_proxy_free (config->proxy);

	g_free (config);
}

/*!
 * find and call the spec handler for each child of GNode
 *
 * \param [in] root \c GNode
 * \param[in,out] config \ref cc_oci_config.
 * \param spec_handlers Array of \ref spec_handler's.
 *
 * \return \c false if a spec handler fails, else \c true.
 */
gboolean
cc_oci_process_config (GNode *root, struct cc_oci_config *config,
	struct spec_handler **spec_handlers)
{
	GNode* node;

	for (node = g_node_first_child(root); node; node = g_node_next_sibling(node)) {
		if (! node->data) {
			continue;
		}

		if (node->children) {
			if (g_strcmp0 (node->data, "ociVersion") == 0) {
				config->oci.oci_version = g_strdup (node->children->data);
			}

			if (g_strcmp0 (node->data, "hostname") == 0) {
				config->oci.hostname = g_strdup (node->children->data);
			}
		}

		/* looking for right spec handler */
		for (struct spec_handler** i=spec_handlers; (*i); ++i) {
			if (g_strcmp0((*i)->name, node->data) == 0) {
				/* run spec handler */
				if (! (*i)->handle_section(node, config)) {
					g_critical("failed spec handler: %s", (*i)->name);
					return false;
				}
				break;
			}
		}
	}

	return true;
}
