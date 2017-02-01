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

#define _GNU_SOURCE
#include <linux/sched.h>
#include <sched.h>
#include <stdbool.h>
#include <string.h>
#include <errno.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>

#include <glib.h>

#include "oci.h"
#include "namespace.h"
#include "util.h"

/** Map of \ref oci_namespace values to human-readable strings. */
static struct cc_oci_ns_map {
	/* namespace value */
	const enum oci_namespace ns;

	/* string representation for namespace */
	const gchar* name;

	/* set to true if namespace is supported by runtime */
	const gboolean supported;
} oci_ns_map[] = {
	{ OCI_NS_CGROUP  , "cgroup"  , false },
	{ OCI_NS_IPC     , "ipc"     , false },
	{ OCI_NS_MOUNT   , "mount"   , true  },
	{ OCI_NS_NET     , "network" , true  },
	{ OCI_NS_PID     , "pid"     , false },
	{ OCI_NS_USER    , "user"    , false },
	{ OCI_NS_UTS     , "uts"     , false },

	{ OCI_NS_INVALID , NULL      , false}
};

/*!
 * Free the specified \ref oci_cfg_namespace.
 *
 * \param ns \ref oci_cfg_namespace.
 */
void
cc_oci_ns_free (struct oci_cfg_namespace *ns)
{
	if (! ns) {
		return;
	}

	g_free_if_set (ns->path);
	g_free (ns);
}

/**
 * check if namespace is supported
 *
 * \param ns \ref oci_namespace.
 *
 * \return true is namespace is supported else false.
 */
static gboolean
cc_oci_ns_supported (enum oci_namespace ns)
{
	struct cc_oci_ns_map  *p;

	for (p = oci_ns_map; p && p->name; p++) {
		if (p->ns != ns) {
			continue;
		}
		return p->supported;
	}

	return false;
}

/**
 * Convert a \ref oci_namespace into a human-readable string.
 *
 * \param ns \ref oci_namespace.
 *
 * \return String representation of \ref oci_namespace on success,
 * else \c NULL.
 */
const char *
cc_oci_ns_to_str (enum oci_namespace ns)
{
	struct cc_oci_ns_map  *p;

	for (p = oci_ns_map; p && p->name; p++) {
		if (p->ns == ns) {
			return p->name;
		}
	}

	return NULL;
}

/**
 * Convert a human-readable string state into a \ref oci_namespace.
 *
 * \param str String to convert to a \ref oci_namespace.
 *
 * \return Valid \ref oci_namespace value, or \ref OCI_NS_INVALID on
 * error.
 */
enum oci_namespace
cc_oci_str_to_ns (const char *str)
{
	struct cc_oci_ns_map  *p;

	if ((! str) || (! *str)) {
		goto out;
	}

	for (p = oci_ns_map; p && p->name; p++) {
		if (! g_strcmp0 (str, p->name)) {
			return p->ns;
		}
	}

out:
	return OCI_NS_INVALID;
}

/**
 * Join a specific namespace
 *
 * \param ns \ref oci_cfg_namespace.
 *
  * \return \c true on success, else \c false.
 */
gboolean cc_oci_ns_join(struct oci_cfg_namespace *ns)
{
	int fd;
	int saved;
	const gchar *type;

	type = cc_oci_ns_to_str (ns->type);

	fd = open (ns->path, O_RDONLY);

	if (fd < 0) {
		saved = errno;
		g_critical ("failed to open %s : %s", ns->path, strerror(errno));
		goto err;
	}

	/* join an existing namespace */
	if (setns (fd, ns->type) < 0) {
		saved = errno;
		g_critical ("failed to join to %s namespace %s : %s",
			type ? type : "", ns->path, strerror(errno));
		goto err;
	}

	close (fd);

	return true;
err:
	if (fd != -1) {
		close (fd);
	}

	/* pass errno back to caller (mainly for tests) */
	errno = saved;

	return false;
}

/**
 * Setup namespaces.
 *
 * This should not strictly be required (since the runtime does not
 * implement a "traditional linux" container). However, namespaces are
 * used to pass network configuration to the runtime so the network
 * namespace must be supported.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 *
 * \todo Show the namespace path. For unshare, the strategy should be to
 * call cc_oci_resolve_path (), passing it the value of.
 * "/proc/self/ns/%s". The complication is that %s does *NOT* match the
 * namespace names chosen by OCI, hence oci_ns_map will need to be
 * extended to add a "gchar *proc_name" element (and tests updated
 * accordingly).
 *
 * \note in the case of error, check the value of errno immediately
 * after this call to determine the reason.
 */
gboolean
cc_oci_ns_setup (struct cc_oci_config *config)
{
	const gchar               *type;
	GSList                    *l;
	struct oci_cfg_namespace  *ns;

	if (! config) {
		return false;
	}

	if (! config->oci.oci_linux.namespaces) {
		g_debug ("no namespaces to setup");
		return true;
	}

	g_debug ("setting up namespaces");

	for (l = config->oci.oci_linux.namespaces;
			l && l->data;
			l = g_slist_next (l)) {
		ns = (struct oci_cfg_namespace *)l->data;

		if (ns->type == OCI_NS_INVALID) {
			continue;
		}

		type = cc_oci_ns_to_str (ns->type);

		/* network and mount namespaces are the only supported
		 * (since it's required to setup networking and mount).
		 */
		if (! cc_oci_ns_supported (ns->type)) {
			g_debug ("ignoring %s namespace request",
					type ? type : "");
			continue;
		}

		if (ns->path) {
			if (! cc_oci_ns_join (ns)) {
				return false;
			}
			g_debug ("joined %s namespace", type ? type : "");
		} else {
			/* create a new namespace */
			if (unshare (ns->type) < 0) {
				return false;
			}
			g_debug ("created %s namespace", type ? type : "");
		}
	}

	g_debug ("finished namespace setup");

	return true;
}

/*!
 * Convert the list of namespaces to a JSON array.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c JsonArray on success, else \c NULL.
 */
JsonArray *
cc_oci_ns_to_json (const struct cc_oci_config *config)
{
	JsonArray *array = NULL;
	JsonObject *ns = NULL;
	GSList *l;

	array  = json_array_new ();

	for (l = config->oci.oci_linux.namespaces; l && l->data; l = g_slist_next (l)) {
		struct oci_cfg_namespace *n = (struct oci_cfg_namespace *)l->data;

		/* DO NOT save unsupported namespaces */
		if (! cc_oci_ns_supported(n->type)) {
			continue;
		}

		ns = json_object_new ();

		json_object_set_string_member (ns, "type",
			cc_oci_ns_to_str(n->type));

		if (n->path) {
			json_object_set_string_member (ns, "path",
				n->path);
		}

		json_array_add_object_element (array, ns);
	}

	return array;
}
