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
#include "spec_handler.h"
#include "json.h"
#include "config.h"
#include "state.h"
#include "oci-config.h"

#include <errno.h>
#include <glib/gstdio.h>

extern struct start_data start_data;

struct subcommand *subcommands[] =
{
	&command_checkpoint,
	&command_create,
	&command_delete,
	&command_events,
	&command_exec,
	&command_help,
	&command_kill,
	&command_list,
	&command_pause,
	&command_ps,
	&command_restore,
	&command_resume,
	&command_run,
	&command_start,
	&command_state,
	&command_stop,
	&command_update,
	&command_version,

	/* terminator */
	NULL
};

/**
 * List of spec handlers used to process config on stop
 */
static struct spec_handler *stop_spec_handlers[] = {
	&hooks_spec_handler,

	/* terminator */
	NULL
};

/*!
 * Handle commands to toggle the state of the Hypervisor
 * (between paused and running).
 *
 * \param sub \ref subcommand.
 * \param config \ref cc_oci_config.
 * \param argc Argument count.
 * \param argv Argument vector.
 * \param pause If \c true, pause the VM, else resume it.
 *
 * \return \c true on success, else \c false.
 */
gboolean
handle_command_toggle (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[], gboolean pause)
{
	struct oci_state      *state = NULL;
	gchar                 *config_file = NULL;
	gboolean               ret;
	const gchar           *action;

	g_assert (sub);
	g_assert (config);

	action = pause ? "pause" : "resume";

	if (handle_default_usage (argc, argv, sub->name,
				&ret, 1, NULL)) {
		return ret;
	}

	/* Used to allow us to find the state file */
	config->optarg_container_id = argv[0];

	ret = cc_oci_get_config_and_state (&config_file, config, &state);
	if (! ret) {
		goto out;
	}

	/* Transfer certain state elements to config to allow the state *
	 * file to be rewritten with full details.
	 */
	if (! cc_oci_config_update (config, state)) {
		goto out;
	}

	ret = cc_oci_toggle (config, state, pause);
	if (! ret) {
		goto out;
	}

	ret = true;

	g_print ("%sd container %s\n", action,
			config->optarg_container_id);

out:
	g_free_if_set (config_file);
	cc_oci_state_free (state);

	if (! ret) {
		g_critical ("failed to %s container %s\n",
				action, config->optarg_container_id);
	}

	return ret;
}

/*!
 * Handle commands to stop the Hypervisor cleanly.
 *
 * \param sub \ref subcommand.
 * \param config \ref cc_oci_config.
 * \param argc Argument count.
 * \param argv Argument vector.
 *
 * \return \c true on success, else \c false.
 */
gboolean
handle_command_stop (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	struct oci_state  *state = NULL;
	gchar             *config_file = NULL;
	gboolean           ret;
	GNode*             root = NULL;

	g_assert (sub);
	g_assert (config);

	if (handle_default_usage (argc, argv, sub->name,
				&ret, 1, NULL)) {
		return ret;
	}

	/* Used to allow us to find the state file */
	config->optarg_container_id = argv[0];

	/* FIXME: deal with containerd calling "delete" twice */
	if (! cc_oci_state_file_exists (config)) {
		g_warning ("state file does not exist for container %s",
				config->optarg_container_id);

		/* don't make this fatal to keep containerd happy */
		ret = true;
		goto out;
	}

	ret = cc_oci_get_config_and_state (&config_file, config, &state);
	if (! ret) {
		goto out;
	}

	/* convert json file to GNode */
	if (! cc_oci_json_parse(&root, config_file)) {
		goto out;
	}

#ifdef DEBUG
	/* show json file converted to GNode */
	cc_oci_node_dump(root);
#endif /*DEBUG*/

	if (! cc_oci_process_config(root, config, stop_spec_handlers)) {
		g_critical ("failed to process config");
		goto out;
	}

	g_free_node(root);

	ret = cc_oci_stop (config, state);
	if (! ret) {
		goto out;
	}

	ret = true;

	g_print ("stopped container %s\n", config->optarg_container_id);

out:
	g_free_if_set (config_file);
	cc_oci_state_free (state);

	if (! ret) {
		g_critical ("failed to stop container %s\n",
				config->optarg_container_id);
	}

	return ret;
}

/*!
 * Handle commands to setup the environment as a precursor to
 * creating the state file.
 *
 * \param sub \ref subcommand.
 * \param config \ref cc_oci_config.
 * \param argc Argument count.
 * \param argv Argument vector.
 *
 * \return \c true on success, else \c false.
 */
gboolean
handle_command_setup (const struct subcommand *sub,
		struct cc_oci_config *config,
		int argc, char *argv[])
{
	gboolean  ret;

	g_assert (sub);
	g_assert (config);

	if (handle_default_usage (argc, argv, sub->name,
				&ret, 1, NULL)) {
		return ret;
	}

	config->optarg_container_id = argv[0];

	if (start_data.bundle) {
		/* Running in "runc mode" where --bundle has already
		 * been specified.
		 */
		if (argc != 1) {
			g_critical ("Usage: %s --bundle "
					"<bundle-path> <container-id>",
					sub->name);
			return false;
		}

		config->bundle_path = cc_oci_resolve_path (start_data.bundle);
		g_free (start_data.bundle);
		start_data.bundle = NULL;
	} else {
		/* Running in strict OCI-mode */
		if (argc != 2) {
			g_critical ("Usage: %s <container-id> "
					"<bundle-path>", sub->name);
			return false;
		}

		config->bundle_path = cc_oci_resolve_path (argv[1]);
	}

	config->console = start_data.console;
	config->pid_file = start_data.pid_file;
	config->dry_run_mode = start_data.dry_run_mode;
	config->detached_mode = start_data.detach;

	return true;
}

/*!
 * Determine if specified arguments are a request to display usage.
 *
 * A successful return from this function denotes that the arguments
 * have been handled.
 *
 * \param argc Argument count.
 * \param argv Argument vector.
 * \param cmd Name of sub-command.
 * \param ret Return code that should be applied if arguments have been
 *   handled.
 * \param min_argc Minimum permitted value of \p argc. Since \ref
 *   subcommand's are handed a modified \c argc, this value is
 *   usually \c 1 to denote the container id.
 * \param extra String showing extra arguments (additional to the
 *   container id) that the user should provide, or \c NULL.
 *
 * \return \c true on success, else \c false.
 */
gboolean
handle_default_usage (int argc, char *argv[],
		const char *cmd, gboolean *ret,
		int min_argc, const char *extra)
{
	g_assert (cmd);
	g_assert (ret);

	gboolean  help = false;

	if (argc && ((!g_strcmp0 (argv[0], "--help")) ||
		     (!g_strcmp0 (argv[0], "-h")))) {
		help = true;
	}
				
	if (help || (!argc) || (argc < min_argc)) {
		g_print ("Usage: %s <container-id>%s%s\n",
				cmd,
				extra ? " " : "",
				extra ? extra : "");

		if (help) {
			*ret = argc == 1;
		} else {
			*ret = argc >= min_argc;
		}

		/* arguments have been handled */
		return true;
	}

	return false;
}

/**
 * Handle parsing of --console which may not be provided with an
 * argument.
 *
 * \param option_name Full option name ("--console").
 * \param value Value of console option.
 * \param data \ref start_data.
 * \param error Unused.
 *
 * \return \c true if option \p option_name was parsed successfully,
 * else \c false.
 */
gboolean
handle_option_console (const gchar *option_name,
		const gchar *value,
		gpointer data,
		GError **error)
{
	struct start_data *start_data;

	g_assert (data);

	start_data = (struct start_data *)data;

	if (value) {
		start_data->console = g_strdup (value);
	}

	/* option handled */
	return true;
}

/**
 * Handle parsing of --user from exec command
 *
 * \param option_name Full option name ("--user").
 * \param value Value of user option in format <uid>[:<gid>]
 * \param data \ref start_data.
 * \param error Unused.
 *
 * \return \c true if option \p option_name was parsed successfully,
 * else \c false.
 */
gboolean
handle_option_user (const gchar *option_name,
		const gchar *value,
		gpointer data,
		GError **error)
{
	struct start_data *start_data;
	gboolean  ret = false;
	gchar   **ids = NULL;
	gchar    *struid = NULL;
	gchar    *strgid = NULL;
	gchar    *endptr;
	gint64    uid = 0;
	gint64    gid = 0;

	if (! (data && value)) {
		return false;
	}

	start_data = (struct start_data *)data;

	ids = g_strsplit (value, ":", 2);

	/* uid is *NOT* optional */
	struid = *ids;
	if (! struid) {
		g_critical("missing uid in option user");
		goto out;
	}

	uid = g_ascii_strtoll (struid, &endptr, 10);
	if (endptr == struid) {
		g_critical("failed to convert '%s' to int", struid);
		goto out;
	}

	/* gid is optional */
	if ((ids+1) != NULL && *(ids+1) != NULL) {
		/* copy gid */
		strgid = *(ids+1);
		gid = g_ascii_strtoll (strgid, &endptr,	10);
		if (endptr == strgid) {
			g_critical("failed to convert '%s' to int", strgid);
			goto out;
		}
	}

	if (uid < 0 || gid < 0){
		g_critical("negative uid value");
		goto out;
	}

	start_data->user.uid = (uid_t)uid;
	start_data->user.gid = (uid_t)gid;

	ret = true;
out:
	g_strfreev(ids);

	return ret;
}
