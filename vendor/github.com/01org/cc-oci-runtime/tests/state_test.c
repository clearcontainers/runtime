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
#include <unistd.h>
#include <sys/stat.h>
#include <fcntl.h>

#include <check.h>
#include <glib.h>
#include <glib/gstdio.h>

#include "test_common.h"
#include "../src/logging.h"
#include "../src/oci.h"
#include "../src/state.h"

#define error_free_if_set(e) if (e) { g_error_free(e); error=NULL; }

const gchar *
cc_oci_status_get (const struct cc_oci_config *config);

START_TEST(test_cc_oci_state_file_get) {
	struct cc_oci_config *config = NULL;

	config = cc_oci_config_create ();
	ck_assert (config);

	ck_assert(!cc_oci_state_file_get(config));
	g_snprintf(config->state.runtime_path, PATH_MAX, "/tmp");
	ck_assert(cc_oci_state_file_get(config));

	/* clean up */
	cc_oci_config_free (config);
} END_TEST


START_TEST(test_cc_oci_state_file_read) {
	struct oci_state *state = NULL;

	/* Needs:
	 *
	 * - ociVersion
	 * - id
	 * - pid
	 * - bundlePath
	 * - commsPath
	 * - processPath
	 * - status
	 */

	ck_assert(! cc_oci_state_file_read(NULL));

	ck_assert(! cc_oci_state_file_read("/abc/123/xyz"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-bundlePath.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-ociVersion.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-id.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-commsPath.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-processPath.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-console.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-console-path.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-console-socket.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-vm-object.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-proxy.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-proxy-ctlSocket.json"));

	ck_assert(! cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-proxy-ioSocket.json"));

	/* Annotations are optional*/
	state = cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-annotations.json");
	ck_assert (state);
	cc_oci_state_free (state);

	/* Mounts are optional */
	state = cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-no-mounts.json");
	ck_assert (state);
	cc_oci_state_free (state);

	state = cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-mounts-no-mount-destination.json");
	ck_assert (state);
	cc_oci_state_free (state);

	state = cc_oci_state_file_read(TEST_DATA_DIR
	                "/state-mounts-no-mount-directory_created.json");
	ck_assert (state);
	cc_oci_state_free (state);

	state = cc_oci_state_file_read(TEST_DATA_DIR "/state.json");
	ck_assert(state);
	ck_assert(state->oci_version);
	ck_assert(state->id);
	ck_assert(state->pid);
	ck_assert(state->bundle_path);
	ck_assert(state->comms_path);
	ck_assert(state->procsock_path);
	ck_assert(state->status);
	ck_assert(state->annotations);
	cc_oci_state_free(state);

} END_TEST

START_TEST(test_cc_oci_state_free) {
	struct oci_state *state = g_new0 (struct oci_state, 1);
	ck_assert(state);

	state->bundle_path = g_strdup("bundle_path");
	state->comms_path = g_strdup("comms_path");
	state->procsock_path = g_strdup("procsock_path");
	state->id = g_strdup("id");
	state->oci_version = g_strdup("oci_version");
	state->mounts = g_slist_append(state->mounts, g_new0(struct cc_oci_mount, 1));
	state->annotations = g_slist_append(state->annotations, g_new0(struct oci_cfg_annotation, 1));

	/* memory leaks will be detected by valgrind */
	cc_oci_state_free(state);
} END_TEST

START_TEST(test_cc_oci_state_file_create) {
	struct cc_oci_config *config = NULL;
	const gchar *timestamp = "foo";
        struct oci_cfg_annotation* a = NULL;
	struct cc_oci_mount *m = NULL;
	g_autofree gchar *tmpdir = g_dir_make_tmp (NULL, NULL);
	gboolean ret;

	config = cc_oci_config_create ();
	ck_assert (config);

	ck_assert(! cc_oci_state_file_create (NULL, NULL));

	/* Needs:
	 *
	 * - comms_path
	 * - procsock_path
	 * - optarg_bundle_path
	 * - optarg_container_id
	 * - runtime_path
	 *
	 */
	ck_assert(! cc_oci_state_file_create (config, NULL));
	ck_assert(! cc_oci_state_file_create (config, timestamp));

	config->optarg_container_id = "";

	ck_assert(! cc_oci_state_file_create (config, NULL));
	ck_assert(! cc_oci_state_file_create (config, timestamp));

	config->optarg_container_id = "foo";

	ck_assert(! cc_oci_state_file_create (config, NULL));
	ck_assert(! cc_oci_state_file_create (config, timestamp));

	config->bundle_path = g_strdup ("/tmp/bundle");

	ck_assert(! cc_oci_state_file_create (config, NULL));
	ck_assert(! cc_oci_state_file_create (config, timestamp));

	config->root_dir = g_strdup (tmpdir);
	ck_assert (config->root_dir);

	ck_assert (cc_oci_runtime_dir_setup (config));

	ck_assert(! cc_oci_state_file_create (config, NULL));
	ck_assert(! cc_oci_state_file_create (config, timestamp));

	g_snprintf(config->state.comms_path, PATH_MAX, "/tmp");
	g_snprintf(config->state.procsock_path, PATH_MAX, "/tmp");

	ck_assert(! cc_oci_state_file_create (config, NULL));

        a = g_new0(struct oci_cfg_annotation, 1);
        a->key = g_strdup("key1");
        a->value = g_strdup("val1");

        config->oci.annotations =
		g_slist_append(config->oci.annotations, a);

	/* config->oci.process not set */
	ck_assert(! cc_oci_state_file_create (config, timestamp));

	snprintf(config->oci.process.cwd, sizeof(config->oci.process.cwd),
				"%s", "/working_directory");

	config->oci.process.args = g_strsplit("/bin/echo test", " ", -1);

	/* config->vm not set */
	ck_assert(! cc_oci_state_file_create (config, timestamp));

	config->vm = g_malloc0 (sizeof(struct cc_oci_vm_cfg));
	ck_assert (config->vm);

	g_strlcpy (config->vm->hypervisor_path, "hypervisor-path",
			sizeof (config->vm->hypervisor_path));

	g_strlcpy (config->vm->image_path, "image-path",
			sizeof (config->vm->image_path));

	g_strlcpy (config->vm->kernel_path, "kernel-path",
			sizeof (config->vm->kernel_path));

	g_strlcpy (config->vm->workload_path, "workload-path",
			sizeof (config->vm->workload_path));

	config->vm->kernel_params = g_strdup ("kernel params");

	m = g_new0(struct cc_oci_mount, 1);
	g_snprintf(m->dest, sizeof(m->dest), "/tmp/tmp/tmp");
	m->directory_created = g_strdup("/tmp/tmp/");
	m->ignore_mount = true;
	config->oci.mounts = g_slist_append(config->oci.mounts, m);

	/* All required elements now set */
	ck_assert (cc_oci_state_file_create (config, timestamp));

	ret = g_file_test (config->state.state_file_path,
			G_FILE_TEST_EXISTS);
	ck_assert (ret);

	ck_assert (! g_remove (config->state.state_file_path));
	ck_assert (! g_remove (config->state.runtime_path));
	ck_assert (! g_remove (tmpdir));

	g_snprintf(config->state.runtime_path, PATH_MAX, "/abc/xyz/123");
	ck_assert(!cc_oci_state_file_create (config, timestamp));

	/* clean up */
	cc_oci_config_free (config);
} END_TEST

START_TEST(test_cc_oci_state_file_delete) {
	struct stat st;
	struct cc_oci_config *config = NULL;
	gint fd = 0;

	config = cc_oci_config_create ();
	ck_assert (config);

	g_snprintf(config->state.state_file_path, PATH_MAX,
			"/tmp/.fileXXXXXX");

	fd = g_mkstemp(config->state.state_file_path);
	ck_assert(fd != -1);
	ck_assert(close(fd) != -1);

	ck_assert(cc_oci_state_file_delete(config));

	ck_assert(stat(config->state.state_file_path, &st));

	/* clean up */
	cc_oci_config_free (config);
} END_TEST

START_TEST(test_cc_oci_state_file_exists) {
	struct cc_oci_config *config = NULL;

	config = cc_oci_config_create ();
	ck_assert (config);

	ck_assert(! cc_oci_state_file_exists(NULL));
	ck_assert(! cc_oci_state_file_exists(config));

	config->optarg_container_id = "2565";
	ck_assert(! cc_oci_state_file_exists(config));

	/* clean up */
	cc_oci_config_free (config);
} END_TEST

START_TEST(test_cc_oci_status_get) {
	const gchar *status;
	struct cc_oci_config *config = NULL;

	config = cc_oci_config_create ();
	ck_assert (config);

	status = cc_oci_status_get (NULL);
	ck_assert (! status);

	status = cc_oci_status_get (config);
	ck_assert (! g_strcmp0 (status, "created"));

	/* let's confirm our understanding of what it's doing */
	config->state.status = OCI_STATUS_CREATED;
	status = cc_oci_status_get (config);
	ck_assert (! g_strcmp0 (status, "created"));

	config->state.status = OCI_STATUS_RUNNING;
	status = cc_oci_status_get (config);
	ck_assert (! g_strcmp0 (status, "running"));

	config->state.status = OCI_STATUS_PAUSED;
	status = cc_oci_status_get (config);
	ck_assert (! g_strcmp0 (status, "paused"));

	config->state.status = OCI_STATUS_STOPPED;
	status = cc_oci_status_get (config);
	ck_assert (! g_strcmp0 (status, "stopped"));

	config->state.status = OCI_STATUS_INVALID;
	ck_assert (! cc_oci_status_get (config));

	/* clean up */
	cc_oci_config_free (config);
} END_TEST

START_TEST(test_cc_oci_status_to_str) {
	const char *str;

	str = cc_oci_status_to_str (OCI_STATUS_CREATED);
	ck_assert (! g_strcmp0 (str, "created"));

	str = cc_oci_status_to_str (OCI_STATUS_RUNNING);
	ck_assert (! g_strcmp0 (str, "running"));

	str = cc_oci_status_to_str (OCI_STATUS_PAUSED);
	ck_assert (! g_strcmp0 (str, "paused"));

	str = cc_oci_status_to_str (OCI_STATUS_STOPPED);
	ck_assert (! g_strcmp0 (str, "stopped"));

	ck_assert (! cc_oci_status_to_str (OCI_STATUS_INVALID));

} END_TEST

START_TEST(test_cc_oci_str_to_status) {

	ck_assert (cc_oci_str_to_status (NULL) == OCI_STATUS_INVALID);
	ck_assert (cc_oci_str_to_status ("") == OCI_STATUS_INVALID);
	ck_assert (cc_oci_str_to_status ("foo bar") == OCI_STATUS_INVALID);
	ck_assert (cc_oci_str_to_status ("garbage") == OCI_STATUS_INVALID);
	ck_assert (cc_oci_str_to_status ("CREATED") == OCI_STATUS_INVALID);

	ck_assert (cc_oci_str_to_status ("created") == OCI_STATUS_CREATED);
	ck_assert (cc_oci_str_to_status ("running") == OCI_STATUS_RUNNING);
	ck_assert (cc_oci_str_to_status ("paused") == OCI_STATUS_PAUSED);
	ck_assert (cc_oci_str_to_status ("stopped") == OCI_STATUS_STOPPED);
} END_TEST

Suite* make_state_suite(void) {
	Suite* s = suite_create(__FILE__);
	ADD_TEST(test_cc_oci_state_file_get, s);
	ADD_TEST(test_cc_oci_state_file_read, s);
	ADD_TEST(test_cc_oci_state_free, s);
	ADD_TEST(test_cc_oci_state_file_create, s);
	ADD_TEST(test_cc_oci_state_file_delete, s);
	ADD_TEST(test_cc_oci_state_file_exists, s);
	ADD_TEST(test_cc_oci_status_get, s);
	ADD_TEST(test_cc_oci_status_to_str, s);
	ADD_TEST(test_cc_oci_str_to_status, s);

	return s;
}

int main(void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_strdup ("state_test_debug.log");
	(void)cc_oci_log_init(&options);

	s = make_state_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
