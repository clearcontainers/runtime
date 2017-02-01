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

extern struct start_data start_data;

/* ignore -pedantic to cast handle_option_console, a function pointer, to a
 * void* */
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wpedantic"
static GOptionEntry options_create[] =
{
	{
		"bundle", 'b', G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &start_data.bundle,
		"path to the bundle directory",
		NULL
	},
	{
		"console", 0, G_OPTION_FLAG_OPTIONAL_ARG,
		G_OPTION_ARG_CALLBACK, handle_option_console,
		"set pty console that will be used in the container",
		NULL
	},
	{
		"no-pivot", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, NULL,
		"not implemented",
	       	NULL
	},
	{
		"pid-file", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &start_data.pid_file,
		"the file to write the process ID of the created "
		"container to",
	       	NULL
	},

	{NULL}
};
#pragma GCC diagnostic pop

static gboolean
handler_create (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	g_assert (sub);
	g_assert (config);

	if (! handle_command_setup (sub, config, argc, argv)) {
		return false;
	}

	return cc_oci_create (config);
}

struct subcommand command_create =
{
	.name        = "create",
	.options     = options_create,
	.handler     = handler_create,
	.description = "create and start a container, "
		       "but do not run workload",
};
