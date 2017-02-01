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

/*! \file */

#include <errno.h>

#include "priv.h"

/*!
 * Determine the privilege level required to execute the sub-command.
 *
 * This is awkward given:
 *
 * - Some sub-commands do not require root.
 * - Some sub-commands require root to:
 *   - create directories below \ref CC_OCI_RUNTIME_DIR_PREFIX
 *     (or config->root_dir).
 *   - call mount(2).
 *   - read files created by other commands run as root.
 * - Some sub-commands that would normally require root don't require it
 *   if passed "--help".
 *
 * \note warning: This function can not be totally reliable since at the
 * time it is called, \ref CC_OCI_CONFIG_FILE has not been parsed so it
 * cannot know if any mounts will actually need to be performed (some
 * are ignored).
 *
 * \param argc Argument count.
 * \param argv Argument vector.
 * \param sub Sub-command.
 * \param config \ref cc_oci_config.
 *
 * \return \c 1 if higher privileges are required,
 *         \c 0 if higher privileges not required,
 *         \c -1 if no potentially privileged setup should be performed.
 */
gint
cc_oci_get_priv_level (int argc,
		char *argv[],
		struct subcommand *sub,
		struct cc_oci_config *config)
{
	g_assert (sub);
	g_assert (config);

	if (! (g_strcmp0 (sub->name, "help")
			&& g_strcmp0 (sub->name, "version"))) {
		/* no privs requires to display metadata sub-command */
		return -1;
	}

	if (argc > 1) {
		if (! (g_strcmp0 (argv[1], "--help")
				&& g_strcmp0 (argv[1], "-h"))) {
			/* no privs requires to display metadata for sub-command */
			return -1;
		}
	}

	if (config->root_dir) {
		if (! access (config->root_dir, W_OK)) {
			/* alternative root exists and is writable */
			return 0;
		} else if (errno == ENOENT) {
			gboolean  ret;
			gchar    *dir = NULL;

			dir = g_path_get_dirname (config->root_dir);

			ret = access (dir, W_OK);

			g_free (dir);

			if (ret == 0) {
				/* parent directory exists and is
				 * writable so the user will be able to
				 * create the new root directory.
				 */
				return 0;
			} else {
				/* likely to need root */
				return 1;
			}
		}
	} else {
		/* the default root is CC_OCI_RUNTIME_DIR_PREFIX, which
		 * requires root.
		 */
		return 1;
	}

	/* best to be cautious */
	return 1;
}
