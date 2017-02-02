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

#ifndef _CC_OCI_COMMAND_H
#define _CC_OCI_COMMAND_H

#include <stdbool.h>

#include <glib.h>

#include "oci.h"
#include "util.h"

/*! A sub-command is a command provided to the application to control
 * its behaviour (think "git").
 */
struct subcommand
{
	/*! Name of sub-command (required). */
	char *name;

	/*! Array of options to pass to
	 * \c g_option_context_parse (optional).
	 */
	GOptionEntry *options;

	/*! Function that will be called to handle the sub-command
	 * (required).
	 *
	 * Since it is also passed any remaining arguments after all
	 * global options and \ref subcommand \c options have been
	 * handled, it is also the argument handler for the sub-command.
	 *
	 * \param sub \ref subcommand being called.
	 * \param config \ref cc_oci_config.
	 * \param argc Argument count for sub-command.
	 * \param argv Argument vector for sub-command.
	 *
	 * \return \c true on success, else \c false.
	 */
	gboolean (*handler) (const struct subcommand *sub,
			struct cc_oci_config *config,
			int argc, char *argv[]);

	/*! sub-command description help(required). */
	char *description;
};

/*!
 * Data used to create and start a container
 * or execute a new workload.
 *
 * This structure is used to simplify command-line parsing for various
 * similar sub-commands. Values are eventually added to a \ref
 * cc_oci_config.
 */
struct start_data {
	gchar *bundle;
	gchar *console;
	gchar *pid_file;
	gboolean detach;
	gboolean dry_run_mode;
	gboolean  allocate_tty;
	struct oci_cfg_user  user;
	/* Path to cc-shim binary */
	gchar *shim_path;
	/* Path to cc-proxy's socket */
	gchar *proxy_socket_path;
};

gboolean handle_command_toggle (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[], gboolean pause);
gboolean handle_command_stop (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[]);
gboolean handle_command_setup (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[]);
gboolean handle_default_usage (int argc, char *argv[], const char *cmd, gboolean *ret, int min_argc, const char *extra);
gboolean handle_option_console (const gchar *option_name,
		const gchar *value,
		gpointer data,
		GError **error);
gboolean handle_option_user (const gchar *option_name,
		const gchar *value,
		gpointer data,
		GError **error);

extern struct subcommand command_checkpoint;
extern struct subcommand command_create;
extern struct subcommand command_delete;
extern struct subcommand command_events;
extern struct subcommand command_exec;
extern struct subcommand command_help;
extern struct subcommand command_kill;
extern struct subcommand command_list;
extern struct subcommand command_pause;
extern struct subcommand command_ps;
extern struct subcommand command_restore;
extern struct subcommand command_resume;
extern struct subcommand command_run;
extern struct subcommand command_start;
extern struct subcommand command_state;
extern struct subcommand command_stop;
extern struct subcommand command_update;
extern struct subcommand command_version;

extern struct subcommand *subcommands[];

#endif /* _CC_OCI_COMMAND_H */
