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

#include <glib.h>

#include "command.h"
#include "json.h"
#include "util.h"
#include "oci.h"
#include "spec_handler.h"
#include <stdlib.h>
#include <glib.h>

extern struct start_data start_data;

static GOptionEntry options_start[] =
{
	// FIXME: runc allows this. why?
	{
		"bundle", 'b', G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &start_data.bundle,
		"path to the bundle directory",
		NULL
	},

	{NULL}
};

static gboolean
handler_start (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	struct oci_state  *state;
	gboolean           ret;

	/* not used */
	g_autofree gchar  *config_file = NULL;

	g_assert (sub);
	g_assert (config);

	if (handle_default_usage (argc, argv, sub->name,
				&ret, 1, NULL)) {
		return ret;
	}

	config->optarg_container_id = argv[0];

	ret = cc_oci_get_config_and_state (&config_file, config, &state);
	if (! ret) {
		return false;
	}

	/* Transfer certain state elements to config to allow the
	 * state file to be rewritten with full details.
	 */
	if (! cc_oci_config_update (config, state)) {
		return false;
	}

	return cc_oci_start (config, state);
}

struct subcommand command_start =
{
	.name         = "start",
	.options      = options_start,
	.handler      = handler_start,
	.description  = "run workload in a created container",
};
