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

#include <stdbool.h>
#include <string.h>
#include <errno.h>

#include <glib.h>
#include <glib/gstdio.h>

#include "oci.h"
#include "util.h"
#include "runtime.h"

/*!
 * Update the specified config with the runtime path.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_runtime_path_get (struct cc_oci_config *config)
{
	if (! config) {
		return false;
	}

	if (! config->optarg_container_id) {
		return false;
	}

	g_snprintf (config->state.runtime_path,
			(gulong)sizeof (config->state.runtime_path),
			"%s/%s",
			config->root_dir ? config->root_dir
			: CC_OCI_RUNTIME_DIR_PREFIX,
			config->optarg_container_id);

	return true;
}

/*!
 * Create the runtime path specified by \p config.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_runtime_dir_setup (struct cc_oci_config *config)
{
	g_autofree gchar *dirname = NULL;

	if (! config) {
		return false;
	}

	/* XXX: This test is expected to fail normally, but by
	 * performing the check here, the tests can subvert
	 * the path.
	 */
	if (! config->state.runtime_path[0]) {
		if (! cc_oci_runtime_path_get (config)) {
			return false;
		}
	}

	g_snprintf (config->state.comms_path,
			(gulong)sizeof (config->state.comms_path),
			"%s/%s",
			config->state.runtime_path,
			CC_OCI_HYPERVISOR_SOCKET);

	g_snprintf (config->state.procsock_path,
			(gulong)sizeof (config->state.procsock_path),
			"%s/%s",
			config->state.runtime_path,
			CC_OCI_PROCESS_SOCKET);

	dirname = g_path_get_dirname (config->state.runtime_path);
	if (! dirname) {
		return false;
	}

	if (g_mkdir_with_parents (dirname, CC_OCI_DIR_MODE)) {
		g_critical ("failed to create directory %s: %s",
				dirname, strerror (errno));
	}

	g_debug ("creating directory %s", config->state.runtime_path);

	return ! g_mkdir_with_parents (config->state.runtime_path, CC_OCI_DIR_MODE);
}

/*!
 * Recursively delete the runtime directory specified by \p config.
 *
 * \param config \ref cc_oci_config.
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_runtime_dir_delete (struct cc_oci_config *config)
{
	if (! config) {
		return false;
	}
	if (config->state.runtime_path[0] != '/') {
		return false;
	}

	return cc_oci_rm_rf (config->state.runtime_path);
}
