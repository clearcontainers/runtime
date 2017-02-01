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

static void
usage (void)
{
	struct subcommand **sub;
	g_print ("Usage:\n");
	g_print ("%s [global options] [command] [command options]\n",
			PACKAGE_NAME);
	g_print ("\n");
	g_print ("Supported commands:\n");
	for (sub = subcommands; (*sub) && (*sub)->name; sub++) {
		if ((*sub)->description ){
			g_print ("\t%-15s %8s\n", (*sub)->name,(*sub)->description);
		}
	}

}

static gboolean
handler_help (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	(void)sub;
	(void)config;

	usage ();
	return true;
}

struct subcommand command_help =
{
	.name        = "help",
	.handler     = handler_help,
	.description = "show this help",
};
