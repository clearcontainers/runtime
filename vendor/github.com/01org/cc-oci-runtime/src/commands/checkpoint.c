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

static gboolean
handler_checkpoint (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	gboolean  ret = true;

	g_assert (sub);
	g_assert (config);

	if (handle_default_usage (argc, argv, sub->name,
				&ret, 1, NULL)) {
		return ret;
	}

	if (! cc_oci_state_file_exists(config)) {
		g_warning ("state file does not exist for container %s",
				config->optarg_container_id);
		ret = false;
	}

	return ret;
}

struct subcommand command_checkpoint =
{
	.name    = "checkpoint",
	.handler = handler_checkpoint,
	.description = "checkpoint a running container",
};
