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

#include "command.h"
#include "state.h"
#include "runtime.h"

static gboolean
handler_state (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	gchar* contents;
	gsize length;
	GError* error = NULL;
	gboolean ret;

	g_assert (sub);
	g_assert (config);

	if (handle_default_usage (argc, argv, sub->name,
				&ret, 1, NULL)) {
		return ret;
	}

	/* Used to allow us to find the state file */
	config->optarg_container_id = argv[0];

	g_debug ("state container_id='%s'", config->optarg_container_id);

	if (! cc_oci_runtime_path_get (config)) {
		return false;
	}

	if (! cc_oci_state_file_get (config)) {
		return false;
	}

	ret = g_file_get_contents (config->state.state_file_path,
			&contents,
			&length,
			&error);

	if (! ret) {
		g_critical ("failed to read state file %s: %s",
				config->state.state_file_path,
				error->message);
		g_error_free (error);
		return false;
	}

	g_print("%s\n", contents);
	g_free (contents);

	return true;
}

struct subcommand command_state =
{
	.name        = "state",
	.handler     = handler_state,
	.description = "shows the state of a container",
};
