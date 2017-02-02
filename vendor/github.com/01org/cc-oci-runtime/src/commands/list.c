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

static char *format;
static gboolean show_all;

static GOptionEntry options_list[] =
{
	{
		"all", 'a', G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, &show_all,
		"display all output", NULL
	},
	{
		"format", 'f', G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &format,
		"change output format", NULL
	},

	{NULL}
};

static gboolean
handler_list (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	gboolean ret;

	g_assert (sub);
	g_assert (config);

	ret = cc_oci_list (config, format ? format : "table", show_all);

	g_free_if_set (format);

	return ret;
}

struct subcommand command_list =
{
	.name        = "list",
	.options     = options_list,
	.handler     = handler_list,
	.description = "list all container details",
};
