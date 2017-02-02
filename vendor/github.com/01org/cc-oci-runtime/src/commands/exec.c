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

/*
 * Minimum permitted args for exec command
 * Only one is required ( container id )
 * exec can be done using:
 * - runtime exec [options] container_id cmd args
 * - runtime exec [options] -p process.json container_id
 * where process.json is have a
 * json node like \c oci_cfg_process
 **/
#define MIN_EXEC_ARGS 1

extern struct start_data start_data;

gchar    *cwd          = NULL;
gchar   **env          = NULL;
gchar    *process_json = NULL;

/* Not supported flags */
/* clear containers does not support apparmor*/
gchar    *apparmor       = NULL;
gchar    *cap            = NULL;
/* clear containers does not support SELINUX*/
gchar    *process_label  = NULL;
gboolean  no_new_privs   = false;
gboolean *no_subreaper   = NULL;


/* ignore -pedantic to cast handle_option_console
 * and  handle_option_user, a function pointer, to a
 * void* */
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wpedantic"
static GOptionEntry options_exec[] =
{
	{
		"apparmor", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &apparmor,
		/* clear containers does not support apparmor */
		"not implemented",
		NULL
	},
	{
		"cap", 'c', G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &cap,
		"not implemented",
		NULL
	},
	{
		"console", 0, G_OPTION_FLAG_OPTIONAL_ARG,
		G_OPTION_ARG_CALLBACK, handle_option_console,
		"set pty console that will by the exec workload",
		NULL
	},
	{
		"cwd", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &cwd,
		"current working directory to run the exec workload",
		NULL
	},
	{
		"detach", 'd' , G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, &start_data.detach,
		"exec process in detach mode",
		NULL
	},
	{
		"env", 'e', G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING_ARRAY, &env,
		"in the container",
		NULL
	},
	{
		"no-new-privs", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, &no_new_privs,
		"not implemented",
		NULL
	},
	{
		"no-subreaper", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, &no_subreaper,
		"not implemented",
		NULL
	},
	{
		"pid-file", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &start_data.pid_file,
		"the file to write the process ID of the new "
		"process executed in the container",
		NULL
	},
	{
		"process", 'p' , G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &process_json,
		"specify path to process.json",
		NULL
	},
	{
		"tty", '0' , G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, &start_data.allocate_tty,
		"allocate a pseudo-TTY for the new exec process",
		NULL
	},
	{
		"user", 'u' , G_OPTION_FLAG_NONE,
		G_OPTION_ARG_CALLBACK, handle_option_user,
		"<uid>[:<gid>], UID for the process to run in format",
		NULL
	},
	{NULL}
};

static gboolean
handler_exec (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	struct oci_state  *state = NULL;
	gchar             *config_file = NULL;
	gboolean           ret = false;
	struct oci_cfg_process *process = NULL;

	g_assert (sub);
	g_assert (config);

	process = &config->oci.process;

	if (handle_default_usage (argc, argv, sub->name,
				&ret, MIN_EXEC_ARGS, "<cmd> [args]")) {
		return ret;
	}

	/* Used to allow us to find the state file */
	config->optarg_container_id = argv[0];

	/* Jump over the container name */
	argv++; argc--;

	if ( argc == 0  && ! process_json) {
		g_print ("Usage: %s <container-id> <cmd> [args]\n",
			sub->name);
		goto out;
	}

	process->user.uid = start_data.user.uid;
	process->user.gid = start_data.user.gid;
	process->env = env;
	if ((start_data.console != NULL) && (start_data.console[0])) {
		process->terminal = true;
	} else {
		process->terminal = false;
	}

	if (cwd){
		if (snprintf (process->cwd, sizeof(process->cwd),
		    "%s", cwd) < 0) {
			g_critical("failed to copy process cwd");
		}
	}

	if (argc > 0){
		/* +1 NULL */
		process->args = g_new0 (gchar *, (gsize) argc + 1 );
		for ( int i = 0; i < argc ; i++){
			process->args[i] = g_strdup(argv[i]);
		}
	}

	ret = cc_oci_get_config_and_state (&config_file, config, &state);
	if (! ret) {
		goto out;
	}

	g_free_if_set(config->console);
	config->console = g_strdup(start_data.console);

	ret = cc_oci_exec (config, state, process_json);
	if (! ret) {
		goto out;
	}

	ret = true;

out:
	g_free_if_set (config_file);
	cc_oci_state_free (state);
	g_free_if_set (apparmor);
	g_free_if_set (cap);
	g_free_if_set (cwd);
	g_free_if_set (process_json);
	g_free_if_set (process_label);
	g_free_if_set (process_json);

	return ret;
}

struct subcommand command_exec =
{
	.name        = "exec",
	.options     = options_exec,
	.handler     = handler_exec,
	.description = "execute a new task inside an existing container",
};
