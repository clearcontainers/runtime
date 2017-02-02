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
show_version (void)
{
	g_print ("%s version: %s\n", PACKAGE_NAME, PACKAGE_VERSION);
	g_print ("spec version: %s\n", CC_OCI_SUPPORTED_SPEC_VERSION);
	g_print ("commit: %s\n", GIT_COMMIT);
}

static gboolean
handler_version (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	(void)sub;
	(void)config;

	show_version ();
	return true;
}

struct subcommand command_version =
{
	.name        = "version",
	.handler     = handler_version,
	.description = "shows the program version and OCI spec supported version",
};
