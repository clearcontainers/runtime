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

#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <stdbool.h>
#include <string.h>
#include <assert.h>

#include "util.h"
#include "logging.h"
#include "command.h"
#include "oci-config.h"
#include "priv.h"

#define KVM_PATH "/dev/kvm"

/* globals */
static char *program_name;

/** Logging options */
static struct cc_log_options cc_log_options;

static gchar *format;
static gchar *criu;
static gboolean show_version;
static gboolean show_help;
static gboolean systemd_cgroup;

/** Path to create state under */
static gchar *root_dir;

struct start_data start_data;

/** Global options (available to all sub-commands) */
static GOptionEntry options_global[] =
{
	{
		"criu", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &criu,
		"not implemented",
		NULL
	},
	{
		"debug", 'd', G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, &cc_log_options.enable_debug,
		"enable debug output",
		NULL
	},
	{
		"global-log", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING,
		&cc_log_options.global_logfile,
		"enable global logging",
		NULL
	},
	{
		"hypervisor-log-dir", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING,
		&cc_log_options.hypervisor_log_dir,
		"specify directory path to output hypervisor log",
		NULL
	},
	{
		"log", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING,
		&cc_log_options.filename,
		"specify path to output log file",
		NULL
	},
	{
		"log-format", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &format,
		"specify format of logfile",
		NULL
	},
	{
		"root", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &root_dir,
		"directory to use for runtime state files",
		NULL
	},
	{
		"systemd-cgroup", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, &systemd_cgroup,
		"not implemented",
		NULL
	},
	{
		"version", 'v', G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, &show_version,
		"display version details",
		NULL
	},
	{
		"help", 'h', G_OPTION_FLAG_NONE,
		G_OPTION_ARG_NONE, &show_help,
		"Show help options",
		NULL
	},
	{
		"shim-path", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &start_data.shim_path,
		"specify path to cc-shim binary",
		NULL
	},
	{
		"proxy-socket-path", 0, G_OPTION_FLAG_NONE,
		G_OPTION_ARG_STRING, &start_data.proxy_socket_path,
		"specify path to cc-proxy's socket",
		NULL
	},
	/* terminator */
	{NULL}
};

/*!
 * Lookup a subcommand by name.
 *
 * \param cmd Name of sub-command to search for.
 *
 * \return \ref subcommand on success, else \c NULL.
 */
static struct subcommand *
get_subcmd (const char *cmd)
{
	struct subcommand **sub;

	for (sub = subcommands; (*sub) && (*sub)->name; sub++) {
		if (! g_strcmp0 (cmd, (*sub)->name)) {
			return *sub;
		}
	}

	return NULL;
}

/*!
 * Handle all sub-commands (and their respective arguments and options).
 *
 * \param argc Argument count.
 * \param argv Argument vector.
 * \param sub Sub-command.
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
handle_sub_commands (int argc, char *argv[],
		struct subcommand *sub,
		struct cc_oci_config *config)
{
	gboolean  ret = false;

	g_assert (argc);
	g_assert (argv);
	g_assert (sub);
	g_assert (config);

	if (sub->options) {
		GOptionContext  *context = NULL;
		GOptionGroup    *group = NULL;
		GError          *error = NULL;

		context = g_option_context_new (sub->name);
		if (! context) {
			g_critical ("failed to create sub-commmand option context");
			return false;
		}

		/* Create a new unnamed option group.
		 *
		 * This allow user data to be passed to
		 * G_OPTION_ARG_CALLBACK options (and used by functions
		 * like handle_option_console ()).
		 */
		group = g_option_group_new (
				NULL,        /* name */
				NULL,        /* description */
				NULL,        /* help_description */
				&start_data, /* user_data */
				NULL);       /* user_data destroy function */
		if (! group) {
			g_critical ("failed to create sub-command option group");
			return false;
		}

		/* Add the options to the group */
		g_option_group_add_entries (group, sub->options);

		/* Associate the group with the context.
		 *
		 * Since the context now owns the group, it is not necessary to
		 * free the group explicitly.
		 */
		g_option_context_set_main_group (context, group);

		/* parse sub-command options */
		ret = g_option_context_parse (context, &argc, &argv,
				&error);
		if (! ret) {
			g_option_context_free (context);

			g_critical ("%s: %s: %s\n",
					program_name,
					sub->name,
					error->message);
			g_error_free (error);

			goto out;
		}

		g_option_context_free (context);
	}

	/* Remove the sub-command name from the options */
	argc--;
	argv++;

	ret = sub->handler (sub, config, argc, argv);
	if (! ret) {
		goto out;
	}

	ret = true;

out:
	return ret;
}

/*!
 * Setup logging.
 *
 * \param options \ref cc_log_options.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
setup_logging (struct cc_log_options *options)
{
	if (! options) {
		return false;
	}

	return cc_oci_log_init (options);
}

/*!
 * Handle all arguments and options (global and sub-commands).
 *
 * \param argc Argument count.
 * \param argv Argument vector.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
handle_arguments (int argc, char **argv)
{
	gboolean               ret = false;
	gint                   priv_level;
	struct subcommand     *sub = NULL;
	GOptionContext        *context;
	GError                *error = NULL;
	const char            *cmd;
	struct cc_oci_config  *config = NULL;

	program_name = argv[0];
	context = g_option_context_new ("- OCI runtime for Clear Containers");
	if (! context) {
		g_critical ("failed to create option context");
		return false;
	}

	config = cc_oci_config_create ();
	if (! config) {
		g_critical ("failed to create config object");
		goto out;
	}

	/* ensure parsing stops at first non-argument and
	 * non-global-option parameter
	 * (to allow sub-commands to work).
	 */
	g_option_context_set_strict_posix (context, true);

	g_option_context_add_main_entries (context, options_global, NULL);

	/* Turn off automatic help generation */
	g_option_context_set_help_enabled(context, FALSE);

	/* parse global options */
	ret = g_option_context_parse (context, &argc, &argv, &error);

	if (! ret) {
		g_critical ("%s: %s\n", program_name, error->message);
		g_error_free (error);
		goto out;
	}

	if (show_help) {
		gchar *help;

		/* Display the help info for global options here  and fall
		 * through to display the additional help output for subcommands
		 * with the help command below
		 */
		help = g_option_context_get_help (context, FALSE, NULL);
					g_print ("%s", help);
					g_free (help);
	}

	if (format && ! g_strcmp0 (format, "json")) {
		cc_log_options.use_json = true;
		g_free (format);
	}

	if (show_version) {
		sub = get_subcmd ("version");
		ret = sub->handler (sub, NULL, 0, NULL);
		goto out;
	}

	/* g_option_context_parse() has now consumed all the global options,
	 * but has still left argv[0] as the program name, so update the
	 * args to get rid of it.
	 */
	argc--;
	argv++;

	if (! argc) {
		sub = get_subcmd ("help");
		(void)sub->handler (sub, NULL, argc, argv);
		ret = true;
		goto out;
	}

	if (root_dir) {
		config->root_dir = g_strdup (root_dir);
	}

	cmd = argv[0];

	/* Find the options for the specific sub-command */
	sub = get_subcmd (cmd);
	if (! sub) {
		g_print ("no such command: %s\n", cmd);
		g_info ("no such command: %s", cmd);
		ret = false;
		goto out;
	}

	priv_level = cc_oci_get_priv_level (argc, argv, sub, config);
	if (priv_level == 1 && getuid ()) {
		g_critical ("must run as root");
		ret = false;
		goto out;
	}

	if (priv_level >= 0 &&
			! setup_logging (&cc_log_options)) {
		/* Send message to stderr as in case logging is
		 * completely broken due to failed setup.
		 */
		fprintf (stderr, "failed to setup logging\n");
		g_critical ("failed to setup logging\n");
		ret = false;
		goto out;
	}

	if (! g_file_test(KVM_PATH, G_FILE_TEST_EXISTS)) {
		fprintf (stderr, "This system does not support virtualization\n");
		g_critical ("This system does not support virtualization\n");
		g_critical("%s does not exist", KVM_PATH);
		ret = false;
		goto out;
	}

	if (cc_log_options.enable_debug) {
		/* Record how runtime was invoked in log */
		gchar *str = g_strjoinv (" ", argv);
		g_debug ("called as: %s %s", program_name, str);
		g_free (str);
	}

	/* Now, deal with the sub-commands
	 * (and their corresponding options)
	 */
	ret = handle_sub_commands (argc, argv, sub, config);

	if (! ret) {
		goto out;
	}

out:
	if (context) {
		g_option_context_free (context);
	}

	if (config) {
		cc_oci_config_free (config);
	}

	return ret;
}

/**
 * Handle global setup.
 */
static gboolean
setup (void)
{
	return cc_oci_handle_signals ();
}

/**
 * Handle cleanup.
 *
 * \param options \ref cc_log_options.
 */
static void
cleanup (struct cc_log_options *options)
{
	g_assert (options);

	cc_oci_log_free (options);
	g_free_if_set (criu);
	g_free_if_set (root_dir);
	g_free_if_set (start_data.shim_path);
	g_free_if_set (start_data.proxy_socket_path);
}

/** Entry point. */
int
main (int argc, char **argv)
{
	gboolean ret;

	ret = setup ();
	if (! ret) {
		goto out;
	}

	ret = handle_arguments (argc, argv);

	cleanup (&cc_log_options);

out:
	exit (ret ? EXIT_SUCCESS : EXIT_FAILURE);
}
