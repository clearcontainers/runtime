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

#include <check.h>
#include <glib.h>
#include <glib/gstdio.h>

#include "test_common.h"
#include "logging.h"
#include "oci.h"
#include "runtime.h"

#define TEST_EXPECTED_PREFIX LOCALSTATEDIR "/run/cc-oci-runtime"

START_TEST(test_cc_oci_runtime_path_get) {
	gchar *expected;
	struct cc_oci_config *config = NULL;

	config = cc_oci_config_create ();
	ck_assert (config);

	ck_assert (! cc_oci_runtime_path_get (NULL));

	/* container_id not set */
	ck_assert (! cc_oci_runtime_path_get (config));

	config->optarg_container_id = "foo";

	ck_assert (cc_oci_runtime_path_get (config));

	expected = g_strdup_printf ("%s/%s",
			TEST_EXPECTED_PREFIX,
			config->optarg_container_id);

	ck_assert (! g_strcmp0 (config->state.runtime_path, expected));
	g_free (expected);
	cc_oci_config_free (config);

} END_TEST

START_TEST(test_cc_oci_runtime_dir_setup) {
	gboolean ret;
	gchar *tmpdir = g_dir_make_tmp (NULL, NULL);
	struct cc_oci_config *config = NULL;

	config = cc_oci_config_create ();
	ck_assert (config);

	ck_assert (! cc_oci_runtime_dir_setup (NULL));

	/* container_id not set */
	ck_assert (! cc_oci_runtime_dir_setup (config));

	config->optarg_container_id = "foo";

	/* Set the runtimepath which subverts cc_oci_runtime_dir_setup()
	 * setting it.
	 */
	g_snprintf (config->state.runtime_path,
			(gulong)sizeof (config->state.runtime_path),
			"%s/%s",
			tmpdir,
			config->optarg_container_id);

	ret = g_file_test (config->state.runtime_path,
			G_FILE_TEST_EXISTS);
	ck_assert (! ret);

	ck_assert (cc_oci_runtime_dir_setup (config));

	ret = g_file_test (config->state.runtime_path,
			G_FILE_TEST_EXISTS | G_FILE_TEST_IS_DIR);
	ck_assert (ret);

	ck_assert (! g_rmdir (config->state.runtime_path));
	ck_assert (! g_rmdir (tmpdir));

	g_free (tmpdir);
	cc_oci_config_free (config);

} END_TEST

START_TEST(test_cc_oci_runtime_dir_delete) {
	gboolean ret;
	gchar *tmpdir = g_dir_make_tmp (NULL, NULL);
	struct cc_oci_config *config = NULL;

	config = cc_oci_config_create ();
	ck_assert (config);

	ck_assert (! cc_oci_runtime_dir_delete (NULL));

	/* No runtime_path set */
	ck_assert (! cc_oci_runtime_dir_delete (config));

	g_snprintf (config->state.runtime_path,
			(gulong)sizeof (config->state.runtime_path),
			"hello");

	/* runtime_path is not absolute */
	ck_assert (! cc_oci_runtime_dir_delete (config));

	g_snprintf (config->state.runtime_path,
			(gulong)sizeof (config->state.runtime_path),
			"../hello");

	/* runtime_path still not absolute */
	ck_assert (! cc_oci_runtime_dir_delete (config));

	g_snprintf (config->state.runtime_path,
			(gulong)sizeof (config->state.runtime_path),
			"%s",
			tmpdir);

	ret = g_file_test (config->state.runtime_path,
			G_FILE_TEST_EXISTS);
	ck_assert (ret);

	ck_assert (cc_oci_runtime_dir_delete (config));

	ret = g_file_test (config->state.runtime_path,
			G_FILE_TEST_EXISTS | G_FILE_TEST_IS_DIR);
	ck_assert (! ret);

	g_free (tmpdir);
	cc_oci_config_free (config);
} END_TEST

Suite* make_runtime_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST(test_cc_oci_runtime_path_get, s);
	ADD_TEST(test_cc_oci_runtime_dir_setup, s);
	ADD_TEST(test_cc_oci_runtime_dir_delete, s);

	return s;
}

int main(void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_strdup ("runtime_test_debug.log");
	(void)cc_oci_log_init(&options);

	s = make_runtime_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
