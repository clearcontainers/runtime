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
#include "events.h"
#define DEFAULT_INTERVAL 5

static gboolean run_once;
static gint interval = DEFAULT_INTERVAL;


static GOptionEntry options_events[] =
{
	{
		"stats", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, &run_once,
		"show container stats and exit", NULL
	},
	{
		"interval", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_INT, &interval,
		"set the interval to refresh stats", NULL
	},
	{NULL}
};

static gboolean
handler_events (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	gboolean               ret = false;
	struct oci_state      *state = NULL;
	gchar                 *config_file = NULL;

	if (! (sub  &&  config) ) {
		ret = false;
		goto out;
	}

	if (handle_default_usage (argc, argv, sub->name,
				&ret, -1, NULL)) {
		goto out;
	}

	if (interval <= 0) {
		g_critical ("Interval must be greater than 0");
		return false;
	}

	/* Used to allow us to find the state file */
	config->optarg_container_id = argv[0];
	ret = cc_oci_get_config_and_state (&config_file, config, &state);
	if (! ret) {
		goto out;
	}
	if (run_once) {
		/* set interva 0 to avoid show_container_stats blocking */
		interval = 0;
	}

	ret = show_container_stats(config, state, interval);

out:
	g_free_if_set (config_file);
	cc_oci_state_free (state);

	return ret;
}


struct subcommand command_events =
{
	.name    = "events",
	.options = options_events,
	.handler = handler_events,
	.description = "shows container resource usage statistics"
};
