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

static gboolean
handler_resume (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	g_assert (sub);
	g_assert (config);

	return handle_command_toggle (sub, config, argc, argv, false);
}

struct subcommand command_resume =
{
	.name        = "resume",
	.handler     = handler_resume,
	.description = "resume a previously paused container",
};
