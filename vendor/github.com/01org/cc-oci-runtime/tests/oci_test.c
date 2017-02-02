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

#include <stdlib.h>
#include <stdbool.h>
#include <assert.h>
#include <errno.h>
#include <sys/types.h>
#include <sys/wait.h>

#include <check.h>
#include <glib.h>
#include <glib/gstdio.h>
#include <json-glib/json-glib.h>
#include <json-glib/json-gobject.h>

#include "test_common.h"
#include "../src/logging.h"
#include "../src/runtime.h"
#include "../src/state.h"
#include "../src/oci.h"
#include "../src/util.h"

gboolean cc_oci_vm_running (const struct oci_state *state);
gboolean cc_oci_create_container_workload (struct cc_oci_config *config);
gchar* get_user_home_dir(struct cc_oci_config *config, gchar *password_path);
void set_env_home(struct cc_oci_config *config);

// TODO: add a 2nd VM state file
START_TEST(test_cc_oci_list) {
	struct cc_oci_config *config = NULL;
	struct cc_oci_config *vm1_config = NULL;
	gboolean ret;
	gchar *tmpdir;
	gchar *vm1_dir;
	gchar *outfile = NULL;
	gchar *contents;
	gchar **lines;
	gchar *expected = NULL;
	JsonParser *parser = NULL;
	JsonReader *reader = NULL;
	JsonNode *node = NULL;
	const gchar *value;

	config = cc_oci_config_create ();
	ck_assert (config);

	vm1_config = cc_oci_config_create ();
	ck_assert (vm1_config);

	ck_assert (! cc_oci_list (NULL, NULL, true));
	ck_assert (! cc_oci_list (NULL, NULL, false));

	ck_assert (! cc_oci_list (NULL, "", true));
	ck_assert (! cc_oci_list (NULL, "", false));

	ck_assert (! cc_oci_list (config, NULL, true));
	ck_assert (! cc_oci_list (config, NULL, false));

	ck_assert (! cc_oci_list (config, "", true));
	ck_assert (! cc_oci_list (config, "", false));

	ck_assert (! cc_oci_list (config, "", true));
	ck_assert (! cc_oci_list (config, "", false));

	ck_assert (! cc_oci_list (config, "invalid format", true));
	ck_assert (! cc_oci_list (config, "invalid format", false));

	tmpdir = g_dir_make_tmp (NULL, NULL);
	ck_assert (tmpdir);

	vm1_dir = g_build_path ("/", tmpdir, "vm1", NULL);
	ck_assert (vm1_dir);

	/* specify an ENOENT path */
	config->root_dir = g_build_path ("/", tmpdir, "does-not-exist", NULL);
	ck_assert (config->root_dir);

	/**************************************/
	/* test default ASCII output - no VMs */

	SAVE_OUTPUT (outfile) {
		ret = cc_oci_list (config, "table", false);
	}
	ck_assert (ret);

	/* check contents */
	ret = g_file_get_contents (outfile, &contents, NULL, NULL);
	ck_assert (ret);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	ck_assert (lines[0]);
	ck_assert (! *lines[1]);

	/* expect only the header line with fields separated by
	 * whitespace.
	 */
	ret = g_regex_match_simple ("^ID\\s*"
			"PID\\s*"
			"STATUS\\s*"
			"BUNDLE\\s*"
			"CREATED\\s*$",
			contents, 0, 0);
	ck_assert (ret);

	g_free (contents);
	g_strfreev (lines);
	ck_assert (! g_remove (outfile));
	g_free (outfile);

	/**************************************/
	/* test ASCII output - no VMs, all mode */

	SAVE_OUTPUT (outfile) {
		ret = cc_oci_list (config, "table", true);
	}
	ck_assert (ret);

	/* check contents */
	ret = g_file_get_contents (outfile, &contents, NULL, NULL);
	ck_assert (ret);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	ck_assert (lines[0]);
	ck_assert (! *lines[1]);

	/* expect only the header line with fields separated by
	 * whitespace.
	 */
	ret = g_regex_match_simple ("^ID\\s*"
			"PID\\s*"
			"STATUS\\s*"
			"BUNDLE\\s*"
			"CREATED\\s*"
			"HYPERVISOR\\s*"
			"KERNEL\\s*"
			"IMAGE\\s*$",
			contents, 0, 0);
	ck_assert (ret);

	g_free (contents);
	g_strfreev (lines);
	ck_assert (! g_remove (outfile));
	g_free (outfile);

	/*****************************/
	/* test JSON output - no VMs */

	SAVE_OUTPUT (outfile) {
		ret = cc_oci_list (config, "json", false);
	}
	ck_assert (ret);

	/* check contents */
	ret = g_file_get_contents (outfile, &contents, NULL, NULL);
	ck_assert (ret);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	ck_assert (lines[0]);

	/* the expected empty JSON document */
	ck_assert (! g_strcmp0 (lines[0], "null"));
	ck_assert (! lines[1]);

	g_free (contents);
	g_strfreev (lines);
	ck_assert (! g_remove (outfile));
	g_free (outfile);

	/*****************************/
	/* test JSON output - no VMs, all mode */

	SAVE_OUTPUT (outfile) {
		ret = cc_oci_list (config, "json", true);
	}
	ck_assert (ret);

	/* check contents */
	ret = g_file_get_contents (outfile, &contents, NULL, NULL);
	ck_assert (ret);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	ck_assert (lines[0]);

	/* the expected empty JSON document */
	ck_assert (! g_strcmp0 (lines[0], "null"));
	ck_assert (! lines[1]);

	g_free (contents);
	g_strfreev (lines);
	ck_assert (! g_remove (outfile));
	g_free (outfile);

	/*******************************/
	/* now, switch to a valid (but empty) directory */

	g_free (config->root_dir);
	config->root_dir = g_strdup (tmpdir);

	/**************************************/
	/* test default ASCII output - no VMs */

	SAVE_OUTPUT (outfile) {
		ret = cc_oci_list (config, "table", false);
	}
	ck_assert (ret);

	/* check contents */
	ret = g_file_get_contents (outfile, &contents, NULL, NULL);
	ck_assert (ret);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	ck_assert (lines[0]);
	ck_assert (! *lines[1]);

	/* expect only the header line with fields separated by
	 * whitespace.
	 */
	ret = g_regex_match_simple ("^ID\\s*"
			"PID\\s*"
			"STATUS\\s*"
			"BUNDLE\\s*"
			"CREATED\\s*$",
			contents, 0, 0);
	ck_assert (ret);

	g_free (contents);
	g_strfreev (lines);
	ck_assert (! g_remove (outfile));
	g_free (outfile);

	/*****************************/
	/* test JSON output - no VMs */

	SAVE_OUTPUT (outfile) {
		ret = cc_oci_list (config, "json", false);
	}
	ck_assert (ret);


	/* check contents */
	ret = g_file_get_contents (outfile, &contents, NULL, NULL);
	ck_assert (ret);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	ck_assert (lines[0]);

	/* the expected empty JSON document */
	ck_assert (! g_strcmp0 (lines[0], "null"));
	ck_assert (! lines[1]);

	g_free (contents);
	g_strfreev (lines);
	ck_assert (! g_remove (outfile));
	g_free (outfile);

	/**********************************************************/
	/* now, create a real state file to simulate a running VM */

	ck_assert (test_helper_create_state_file ("vm1", tmpdir,
				vm1_config));

	/************************************/
	/* test default ASCII output - 1 VM */

	SAVE_OUTPUT (outfile) {
		ret = cc_oci_list (config, "table", false);
	}
	ck_assert (ret);

	/* check contents */
	ret = g_file_get_contents (outfile, &contents, NULL, NULL);
	ck_assert (ret);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	ck_assert (lines[0]);
	ck_assert (lines[1]);
	ck_assert (! *lines[2]);

	/* expect only the header line with fields separated by
	 * whitespace.
	 */
	ret = g_regex_match_simple ("^ID\\s*"
			"PID\\s*"
			"STATUS\\s*"
			"BUNDLE\\s*"
			"CREATED\\s*$",
			lines[0], 0, 0);
	ck_assert (ret);

	expected = g_strdup_printf ("%s\\s*"
			"%d\\s*"
			"%s\\s*"
			"%s\\s*"
			"%s\\s*",
			vm1_config->optarg_container_id,
			vm1_config->state.workload_pid,
			"created",
			vm1_config->bundle_path,
			"timestamp for vm1");
	ck_assert (expected);

	ret = g_regex_match_simple (expected, lines[1], 0, 0);
	ck_assert (ret);

	g_free (contents);
	g_free (expected);
	g_strfreev (lines);
	ck_assert (! g_remove (outfile));
	g_free (outfile);

	/************************************/
	/* test default ASCII output - 1 VM, all mode */

	SAVE_OUTPUT (outfile) {
		ret = cc_oci_list (config, "table", true);
	}
	ck_assert (ret);

	/* check contents */
	ret = g_file_get_contents (outfile, &contents, NULL, NULL);
	ck_assert (ret);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	ck_assert (lines[0]);
	ck_assert (lines[1]);
	ck_assert (! *lines[2]);

	/* expect only the header line with fields separated by
	 * whitespace.
	 */
	ret = g_regex_match_simple ("^ID\\s*"
			"PID\\s*"
			"STATUS\\s*"
			"BUNDLE\\s*"
			"CREATED\\s*"
			"HYPERVISOR\\s*"
			"KERNEL\\s*"
			"IMAGE\\s*$",
			lines[0], 0, 0);
	ck_assert (ret);

	expected = g_strdup_printf ("%s\\s*"
			"%d\\s*"
			"%s\\s*"
			"%s\\s*"
			"%s\\s*"
			"%s\\s*"
			"%s\\s*"
			"%s\\s*",
			vm1_config->optarg_container_id,
			vm1_config->state.workload_pid,
			"created",
			vm1_config->bundle_path,
			"timestamp for vm1",
			vm1_config->vm->hypervisor_path,
			vm1_config->vm->kernel_path,
			vm1_config->vm->image_path);
	ck_assert (expected);

	ret = g_regex_match_simple (expected, lines[1], 0, 0);
	ck_assert (ret);

	g_free (contents);
	g_free (expected);
	g_strfreev (lines);
	ck_assert (! g_remove (outfile));
	g_free (outfile);

	/*****************************/
	/* test JSON output - 1 VM */

	SAVE_OUTPUT (outfile) {
		ret = cc_oci_list (config, "json", false);
	}
	ck_assert (ret);

	parser = json_parser_new ();
	ck_assert (parser);

	ret = json_parser_load_from_file (parser, outfile, NULL);
	ck_assert (ret);

	reader = json_reader_new (NULL);
	ck_assert (reader);

	json_reader_set_root (reader, json_parser_get_root (parser));

	/* "list" returns an array of objects */
	ck_assert (json_reader_is_array (reader));

	/* we expect a single element */
	ck_assert (json_reader_count_elements (reader) == 1);

	/* read element */
	ck_assert (json_reader_read_element (reader, 0));

	ck_assert (json_reader_read_member (reader, "id"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, vm1_config->optarg_container_id));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "status"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, "created"));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "bundle"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, vm1_config->bundle_path));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "created"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, "timestamp for vm1"));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "pid"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	ck_assert (vm1_config->state.workload_pid == json_node_get_int (node));
	json_reader_end_member (reader);

	g_object_unref (reader);
	g_object_unref (parser);

	ck_assert (! g_remove (outfile));
	g_free (outfile);

	/*****************************/
	/* test JSON output - 1 VM, all mode */

	SAVE_OUTPUT (outfile) {
		ret = cc_oci_list (config, "json", true);
	}
	ck_assert (ret);

	parser = json_parser_new ();
	ck_assert (parser);

	ret = json_parser_load_from_file (parser, outfile, NULL);
	ck_assert (ret);

	reader = json_reader_new (NULL);
	ck_assert (reader);

	json_reader_set_root (reader, json_parser_get_root (parser));

	/* "list" returns an array of objects */
	ck_assert (json_reader_is_array (reader));

	/* we expect a single element */
	ck_assert (json_reader_count_elements (reader) == 1);

	/* read element */
	ck_assert (json_reader_read_element (reader, 0));

	ck_assert (json_reader_read_member (reader, "id"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, vm1_config->optarg_container_id));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "status"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, "created"));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "bundle"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, vm1_config->bundle_path));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "created"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, "timestamp for vm1"));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "hypervisor"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, vm1_config->vm->hypervisor_path));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "kernel"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, vm1_config->vm->kernel_path));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "image"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	value = json_node_get_string (node);
	ck_assert (value);
	ck_assert (! g_strcmp0 (value, vm1_config->vm->image_path));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "pid"));
	node = json_reader_get_value (reader);
	ck_assert (node);
	ck_assert (vm1_config->state.workload_pid == json_node_get_int (node));
	json_reader_end_member (reader);

	g_object_unref (reader);
	g_object_unref (parser);

	ck_assert (! g_remove (outfile));
	g_free (outfile);
	/* clean up */
	ck_assert (! g_remove (vm1_config->state.state_file_path));
	ck_assert (! g_remove (vm1_config->state.runtime_path));

	ck_assert (! g_remove (tmpdir));
	g_free (tmpdir);
	cc_oci_config_free (vm1_config);
	cc_oci_config_free (config);
	g_free (vm1_dir);

} END_TEST

START_TEST(test_cc_oci_get_bundle_path) {
	gchar *path;

	ck_assert (! cc_oci_get_bundlepath_file (NULL, NULL));
	ck_assert (! cc_oci_get_bundlepath_file ("", ""));
	ck_assert (! cc_oci_get_bundlepath_file ("", NULL));
	ck_assert (! cc_oci_get_bundlepath_file (NULL, ""));

	path = cc_oci_get_bundlepath_file ("a", "b");
	ck_assert (path);
	ck_assert (! g_strcmp0 (path, "a/b"));
	g_free (path);

	path = cc_oci_get_bundlepath_file ("/a", "b");
	ck_assert (path);
	ck_assert (! g_strcmp0 (path, "/a/b"));
	g_free (path);

	path = cc_oci_get_bundlepath_file ("/a", "/b");
	ck_assert (path);
	ck_assert (! g_strcmp0 (path, "/a/b"));
	g_free (path);

	path = cc_oci_get_bundlepath_file ("/a/", "/b");
	ck_assert (path);
	ck_assert (! g_strcmp0 (path, "/a/b"));
	g_free (path);

	path = cc_oci_get_bundlepath_file ("/a/", "/b/");
	ck_assert (path);
	ck_assert (! g_strcmp0 (path, "/a/b/"));
	g_free (path);
} END_TEST

START_TEST(test_cc_oci_config_update) {
	struct cc_oci_config *config = NULL;
	struct oci_state *state;
	struct cc_oci_mount *m;
	struct cc_oci_vm_cfg *vm;
	state = g_malloc0 (sizeof (struct oci_state));
	ck_assert (state);

	config = cc_oci_config_create ();
	ck_assert (config);

	ck_assert (! cc_oci_config_update (NULL, NULL));
	ck_assert (! cc_oci_config_update (NULL, state));
	ck_assert (! cc_oci_config_update (config, NULL));

	ck_assert (! config->oci.mounts);
	ck_assert (! config->console);
	ck_assert (! config->vm);

	/**************************/
	/* setup the state object */

	state->console = g_strdup ("console");

	/* create mount object */
	m = g_malloc0 (sizeof (struct cc_oci_mount));
	ck_assert (m);
	m->mnt.mnt_fsname = g_strdup ("fsname");
	m->mnt.mnt_dir = g_strdup ("dir");
	m->mnt.mnt_type = g_strdup ("type");
	m->mnt.mnt_opts = g_strdup ("options");

	/* add mount object to state */
	state->mounts = g_slist_append (state->mounts, m);
	ck_assert (g_slist_length (state->mounts) == 1);

	/* create vm object */
	vm = g_malloc0 (sizeof(struct cc_oci_vm_cfg));
	ck_assert (vm);
	g_strlcpy (vm->hypervisor_path, "hypervisor_path", sizeof (vm->hypervisor_path));
	g_strlcpy (vm->image_path, "image_path", sizeof (vm->image_path));
	g_strlcpy (vm->kernel_path, "kernel_path", sizeof (vm->kernel_path));
	g_strlcpy (vm->workload_path, "workload_path", sizeof (vm->workload_path));
	vm->kernel_params = g_strdup ("kernel params");

	/* add vm object to state */
	state->vm = vm;

	/* perform the transfer */
	ck_assert (cc_oci_config_update (config, state));

	ck_assert (! state->mounts);
	ck_assert (! state->console);
	ck_assert (! state->vm);

	ck_assert (config->oci.mounts);
	ck_assert (config->console);

	ck_assert (config->vm);

	ck_assert (g_slist_length (config->oci.mounts) == 1);

	/* check mount object */
	m = (struct cc_oci_mount *)g_slist_nth_data (config->oci.mounts, 0);
	ck_assert (m);
	ck_assert (! g_strcmp0 (m->mnt.mnt_fsname, "fsname"));
	ck_assert (! g_strcmp0 (m->mnt.mnt_dir, "dir"));
	ck_assert (! g_strcmp0 (m->mnt.mnt_type, "type"));
	ck_assert (! g_strcmp0 (m->mnt.mnt_opts, "options"));

	/* check console */
	ck_assert (! g_strcmp0 (config->console, "console"));

	/* check vm object */
	ck_assert (! g_strcmp0 (config->vm->hypervisor_path, "hypervisor_path"));
	ck_assert (! g_strcmp0 (config->vm->image_path, "image_path"));
	ck_assert (! g_strcmp0 (config->vm->kernel_path, "kernel_path"));
	ck_assert (! g_strcmp0 (config->vm->workload_path, "workload_path"));
	ck_assert (! g_strcmp0 (config->vm->kernel_params, "kernel params"));

	/* clean up */
	cc_oci_config_free (config);
	cc_oci_state_free (state);

} END_TEST

START_TEST(test_cc_oci_get_config_and_state) {
	struct cc_oci_config *config = NULL;
	struct cc_oci_config *vm1_config = NULL;
	struct oci_state *state;
	gchar *config_file = NULL;
	gchar *tmpdir;
	g_autofree gchar *path = NULL;

	config = cc_oci_config_create ();
	ck_assert (config);

	vm1_config = cc_oci_config_create ();
	ck_assert (vm1_config);

	tmpdir = g_dir_make_tmp (NULL, NULL);
	ck_assert (tmpdir);

	ck_assert (! cc_oci_get_config_and_state (NULL, NULL, NULL));
	ck_assert (! cc_oci_get_config_and_state (NULL, config, &state));
	ck_assert (! cc_oci_get_config_and_state (NULL, NULL, &state));
	ck_assert (! cc_oci_get_config_and_state (&config_file, NULL, NULL));
	ck_assert (! cc_oci_get_config_and_state (NULL, config, NULL));
	ck_assert (! cc_oci_get_config_and_state (&config_file, config, NULL));

	/* no container id */
	ck_assert (! cc_oci_get_config_and_state (&config_file, config, &state));

	/* create a VM state file */
	ck_assert (test_helper_create_state_file ("vm1", tmpdir, vm1_config));
	ck_assert (! vm1_config->oci.mounts);

	config->optarg_container_id = "vm1";
	config->root_dir = g_strdup (tmpdir);
	ck_assert (config->root_dir);

	/* load details of vm1_config into config and state */
	ck_assert (cc_oci_get_config_and_state (&config_file, config, &state));

	/* check config */
	ck_assert (! g_strcmp0 (config->state.runtime_path, vm1_config->state.runtime_path));
	ck_assert (! g_strcmp0 (config->state.state_file_path, vm1_config->state.state_file_path));
	ck_assert (! g_strcmp0 (config->state.comms_path, vm1_config->state.comms_path));
	ck_assert (! g_strcmp0 (config->bundle_path, vm1_config->bundle_path));

	ck_assert (config->state.workload_pid == vm1_config->state.workload_pid);
	ck_assert (! config->oci.mounts);

	ck_assert (config_file);
	path = g_build_path ("/", config->bundle_path, "config.json", NULL);
	ck_assert (! g_strcmp0 (path, config_file));

	ck_assert (! config->vm);

	/* check state */
	ck_assert (state);
	ck_assert (! g_strcmp0 (state->oci_version, "1.0.0-rc1"));
	ck_assert (! g_strcmp0 (state->id, config->optarg_container_id));

	ck_assert (state->pid == config->state.workload_pid);
	ck_assert (state->status == OCI_STATUS_CREATED);
	ck_assert (! g_strcmp0 (state->bundle_path, config->bundle_path));
	ck_assert (! g_strcmp0 (state->comms_path, config->state.comms_path));

	ck_assert (! g_strcmp0 (state->console, vm1_config->console));
	ck_assert (! config->console);
	ck_assert (! g_strcmp0 (state->create_time, "timestamp for vm1"));
	ck_assert (! state->mounts);

	ck_assert (state->vm);
	ck_assert (vm1_config->vm);
	ck_assert (! g_strcmp0 (state->vm->hypervisor_path, vm1_config->vm->hypervisor_path));
	ck_assert (! g_strcmp0 (state->vm->image_path, vm1_config->vm->image_path));
	ck_assert (! g_strcmp0 (state->vm->kernel_path, vm1_config->vm->kernel_path));
	ck_assert (! g_strcmp0 (state->vm->workload_path, vm1_config->vm->workload_path));

	ck_assert (state->vm->kernel_params);
	ck_assert (! g_strcmp0 (state->vm->kernel_params, vm1_config->vm->kernel_params));

	/* clean up */
	ck_assert (! g_remove (vm1_config->state.state_file_path));
	ck_assert (! g_remove (vm1_config->state.runtime_path));

	ck_assert (! g_remove (tmpdir));
	g_free (tmpdir);
	cc_oci_config_free (vm1_config);
	cc_oci_config_free (config);
	g_free (config_file);
	cc_oci_state_free (state);

} END_TEST

START_TEST(test_cc_oci_vm_running) {
	struct oci_state state = {0};

	ck_assert (! cc_oci_vm_running (NULL));

	/* no vm */
	ck_assert (! cc_oci_vm_running (&state));

	state.vm = g_malloc0(sizeof(struct cc_oci_vm_cfg));
	ck_assert(state.vm);

	/* no pid for vm */
	ck_assert (! cc_oci_vm_running (&state));

	/* our pid provided as hypervisor pid*/
	state.vm->pid = getpid ();
	ck_assert (cc_oci_vm_running (&state));

	/* invalid pid (we hope: this is potential an unreliable test).
	 */
	state.vm->pid = (pid_t)INT_MAX;
	ck_assert (! cc_oci_vm_running (&state));

	g_free(state.vm);

} END_TEST

START_TEST(test_get_user_home_dir) {
	struct cc_oci_config *config = NULL;
	gchar *user_home;

	config = cc_oci_config_create ();
	ck_assert (config);

	config->oci.process.env = g_new0 (gchar *, 4);
	config->oci.process.env[0] = g_strdup ("foo=bar");
	config->oci.process.env[1] = g_strdup ("hello=world");
	config->oci.process.env[2] = g_strdup ("a=b");

	config->oci.process.user.uid = 0;
	user_home = get_user_home_dir(config, TEST_DATA_DIR "/passwd");
	ck_assert (! g_strcmp0 (user_home, "/root"));
	g_free(user_home);

	config->oci.process.user.uid = 1;
	user_home = get_user_home_dir(config, TEST_DATA_DIR "/passwd");
	ck_assert (! g_strcmp0 (user_home, "/usr/sbin"));
	g_free(user_home);

	cc_oci_config_free (config);

} END_TEST


START_TEST(test_set_env_home) {
	gboolean ret;
	g_autofree gchar *tmpdir = NULL;
	g_autofree gchar *passwd_path = NULL;
	g_autofree gchar *tmp_etc_dir = NULL;
	struct cc_oci_config *config = NULL;
	gchar *pw_contents = "sync:x:4:65534:sync:/bin:/bin/sync\ntestuser:x:12:12:testuser:/home/testuser:/bin/bash";

	config = cc_oci_config_create ();
	ck_assert (config);

	tmpdir = g_dir_make_tmp (NULL, NULL);
	ck_assert (tmpdir);
	tmp_etc_dir = g_build_path ("/", tmpdir, "etc", NULL);
	ck_assert (! g_mkdir (tmp_etc_dir, 0750));

	g_strlcpy (config->oci.root.path,
			tmpdir,
			sizeof (config->oci.root.path));

	passwd_path = g_strdup_printf("%s/%s", config->oci.root.path, "etc/passwd");
	ret = g_file_set_contents (passwd_path, pw_contents, -1, NULL);
        ck_assert (ret);

	config->oci.process.env = g_new0 (gchar *, 4);
	config->oci.process.env[0] = g_strdup ("foo=bar");
	config->oci.process.env[1] = g_strdup ("hello=world");
	config->oci.process.env[2] = g_strdup ("a=b");

	config->oci.process.user.uid = 12;
	set_env_home(config);

	ck_assert (g_strv_contains ((const gchar * const *)config->oci.process.env, "HOME=/home/testuser"));
	ck_assert (g_strv_contains ((const gchar * const *)config->oci.process.env, "foo=bar"));
	ck_assert (g_strv_contains ((const gchar * const *)config->oci.process.env, "hello=world"));
	ck_assert (g_strv_contains ((const gchar * const *)config->oci.process.env, "a=b"));

	/* Check that HOME env var is not changed if already present in the config */
	config->oci.process.user.uid = 4;
	set_env_home(config);

	ck_assert (g_strv_contains ((const gchar * const *)config->oci.process.env, "HOME=/home/testuser"));
	cc_oci_config_free (config);

	/* recreate */
	config = cc_oci_config_create ();
	ck_assert (config);

	/* Check if default is set if home dir could not be retrieved */
	g_strlcpy (config->oci.root.path,
                        tmpdir,
                        sizeof (config->oci.root.path));

	config->oci.process.env = g_new0 (gchar *, 2);
	config->oci.process.env[0] = g_strdup ("foo=bar");
	config->oci.process.user.uid = 100;
	set_env_home(config);

	ck_assert (g_strv_contains ((const gchar * const *)config->oci.process.env, "HOME=/"));
	ck_assert (g_strv_contains ((const gchar * const *)config->oci.process.env, "foo=bar"));

	/* clean up */
	ck_assert (! g_remove (passwd_path));
	ck_assert (! g_remove (tmp_etc_dir));
	ck_assert (! g_remove (tmpdir));
	cc_oci_config_free (config);

} END_TEST


START_TEST(test_cc_oci_kill) {
	struct cc_oci_config *config_new = NULL;
	struct cc_oci_config *config = NULL;

	/* used to create a state file */
	struct cc_oci_config *config_tmp = NULL;

	struct oci_state *state = NULL;
	struct oci_state *state_new = NULL;
	gboolean ret;
	g_autofree gchar *tmpdir = NULL;
	g_autofree gchar *config_file = NULL;
	g_autofree gchar *config_file_new = NULL;
	int status = 0;
	GSpawnFlags flags =
		(G_SPAWN_SEARCH_PATH |
		G_SPAWN_STDOUT_TO_DEV_NULL |
		G_SPAWN_STDERR_TO_DEV_NULL |
		G_SPAWN_DO_NOT_REAP_CHILD);

	config = cc_oci_config_create ();
	ck_assert (config);

	config->oci.process.args = g_strsplit("sleep 999", " ", -1);
	snprintf(config->oci.process.cwd, sizeof(config->oci.process.cwd),
				"%s", "/working_directory");

	config_tmp = cc_oci_config_create ();
	ck_assert (config_tmp);

	config_new = cc_oci_config_create ();
	ck_assert (config_new);

	ck_assert (! cc_oci_kill (NULL, NULL, 0));

	tmpdir = g_dir_make_tmp (NULL, NULL);
	ck_assert (tmpdir);

	config_tmp->optarg_container_id = "foo";

	config->root_dir = g_strdup (tmpdir);
	ck_assert (config->root_dir);

	/* start a fake process */
	ret = g_spawn_async (NULL, /* wd */
			config->oci.process.args,
			NULL, /* env */
			flags,
			NULL, /* child setup */
			NULL, /* data */
			&config_tmp->state.workload_pid,
			NULL); /* error */
	ck_assert (ret);

	config_tmp->state.status = OCI_STATUS_RUNNING;

	ret = test_helper_create_state_file (config_tmp->optarg_container_id,
				tmpdir,
				config_tmp);
	ck_assert (ret);

	config->optarg_container_id = config_tmp->optarg_container_id;

	ck_assert (cc_oci_get_config_and_state (&config_file,
				config, &state));

	ck_assert (cc_oci_config_update (config, state));

	ck_assert (state->pid == config_tmp->state.workload_pid);

	ck_assert (cc_oci_kill (config, state, SIGTERM));
	(void)waitpid (state->pid, &status, 0);

	ck_assert (kill (config->state.workload_pid, 0) < 0);
	ck_assert (errno == ESRCH);

	ck_assert (WIFSIGNALED (status));
	ck_assert (WTERMSIG (status) == SIGTERM);

	config_new->optarg_container_id = config->optarg_container_id;
	config_new->root_dir = g_strdup (tmpdir);
	ck_assert (config_new->root_dir);

	ck_assert (cc_oci_get_config_and_state (&config_file_new,
				config_new, &state_new));

	ck_assert (state_new->status == OCI_STATUS_STOPPED);

	/* clean up */
	ck_assert (! g_remove (config_tmp->state.state_file_path));
	ck_assert (! g_remove (config_tmp->state.runtime_path));

	cc_oci_state_free (state);
	cc_oci_state_free (state_new);
	cc_oci_config_free (config_tmp);
	cc_oci_config_free (config);
	cc_oci_config_free (config_new);

	ck_assert (! g_remove (tmpdir));

} END_TEST

START_TEST(test_cc_oci_process_to_json) {
	struct oci_cfg_process *process = NULL;
	JsonObject* process_obj = NULL;\

	ck_assert (! cc_oci_process_to_json(process));

	process = calloc(1, sizeof(*process));

	ck_assert (! cc_oci_process_to_json(process));

	process->args = calloc(3, sizeof(*process->args));
	process->args[0] = "_hello";
	process->args[1] = "_world";

	ck_assert (! cc_oci_process_to_json(process));

	process->cwd[0] = '/';

	process_obj = cc_oci_process_to_json(process);
	ck_assert (process_obj);
	json_object_unref(process_obj);

	process->env = calloc(3, sizeof(*process->env));
	process->env[0] = "var1=beer";
	process->env[1] = "var1=wine";

	process_obj = cc_oci_process_to_json(process);
	ck_assert (process_obj);
	json_object_unref(process_obj);

	process->terminal = false;

	process_obj = cc_oci_process_to_json(process);
	ck_assert (process_obj);
	json_object_unref(process_obj);

	process->terminal = false;

	process_obj = cc_oci_process_to_json(process);
	ck_assert (process_obj);
	json_object_unref(process_obj);

	process->user.additionalGids = 0;
	process->user.gid = 0;
	process->user.uid = 0;

	process_obj = cc_oci_process_to_json(process);
	ck_assert (process_obj);
	json_object_unref(process_obj);

	process->stdio_stream = 0;
	process->stderr_stream = 1;

	process_obj = cc_oci_process_to_json(process);
	ck_assert (process_obj);

	ck_assert (json_object_has_member(process_obj, "cwd"));
	ck_assert (json_object_has_member(process_obj, "terminal"));
	ck_assert (json_object_has_member(process_obj, "user"));
	ck_assert (json_object_has_member(process_obj, "args"));
	ck_assert (json_object_has_member(process_obj, "env"));
	ck_assert (json_object_has_member(process_obj, "stdio_stream"));
	ck_assert (json_object_has_member(process_obj, "stderr_stream"));

	json_object_unref(process_obj);

	free(process->args);
	free(process->env);
	free(process);
} END_TEST

START_TEST(test_cc_oci_exec) {
	struct cc_oci_config config = { { 0 } };
	struct oci_state state;
	char  *process_json = NULL;
	state.pod = NULL;
	ck_assert(! cc_oci_exec(NULL, NULL, NULL));
	ck_assert(! cc_oci_exec(&config, &state, NULL));
	ck_assert(! cc_oci_exec(&config, &state, process_json));
} END_TEST

START_TEST(test_cc_oci_toggle) {
	struct cc_oci_config* config = cc_oci_config_create();
	struct oci_state state = { 0 };

	ck_assert (! cc_oci_toggle (NULL, NULL, false));
	ck_assert (! cc_oci_toggle (config, NULL, false));
	ck_assert (! cc_oci_toggle (NULL, &state, false));

	state.status = OCI_STATUS_RUNNING;
	ck_assert (cc_oci_toggle (config, &state, false));

	state.status = OCI_STATUS_RUNNING;
	ck_assert (! cc_oci_toggle (config, &state, true));
	cc_oci_config_free (config);
} END_TEST

Suite* make_oci_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST (test_cc_oci_list, s);
	ADD_TEST (test_cc_oci_get_bundle_path, s);
	ADD_TEST (test_cc_oci_config_update, s);
	ADD_TEST (test_cc_oci_get_config_and_state, s);
	ADD_TEST (test_cc_oci_vm_running, s);
	ADD_TEST (test_cc_oci_kill, s);
	ADD_TEST (test_get_user_home_dir, s);
	ADD_TEST (test_set_env_home, s);
	ADD_TEST (test_cc_oci_process_to_json, s);
	ADD_TEST (test_cc_oci_exec, s);
	ADD_TEST (test_cc_oci_toggle, s);

	return s;
}

int main (void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_strdup ("oci_test_debug.log");
	(void)cc_oci_log_init(&options);

	s = make_oci_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
