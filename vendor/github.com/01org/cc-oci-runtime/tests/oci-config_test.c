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

#include "test_common.h"
#include "oci.h"
#include "logging.h"
#include "oci-config.h"

START_TEST(test_cc_oci_config_check) {
	struct cc_oci_config *config = NULL;

	config = cc_oci_config_create ();
	ck_assert (config);

	ck_assert (! cc_oci_config_check (NULL));
	ck_assert (! cc_oci_config_check (config));

	config->oci.oci_version = g_strdup ("0.0.1");
	g_strlcpy (config->oci.process.cwd, "/foo", sizeof (config->oci.process.cwd));
	config->oci.platform.os = g_strdup ("linux");
	config->oci.platform.arch = g_strdup ("amd64");

	ck_assert (cc_oci_config_check (config));

	/* now, invalidate various elements */

	/* spec version higher than supported version */
	g_free (config->oci.oci_version);
	config->oci.oci_version = g_strdup ("9999.9999.9999");
	ck_assert (! cc_oci_config_check (config));

	g_free (config->oci.oci_version);
	config->oci.oci_version = g_strdup ("0.0.1");
	ck_assert (cc_oci_config_check (config));

	/* clean up */
	cc_oci_config_free (config);

} END_TEST

START_TEST(test_cc_oci_config_file_path) {
	gchar *p;

	ck_assert (! cc_oci_config_file_path (NULL));

	p = cc_oci_config_file_path ("/tmp");
	ck_assert (p);

	ck_assert (! g_strcmp0 (p, "/tmp/config.json"));

	g_free (p);

} END_TEST

Suite* make_runtime_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST(test_cc_oci_config_check, s);
	ADD_TEST(test_cc_oci_config_file_path, s);

	return s;
}

int main(void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;

	/* use underscore rather than dash as that's
	 * what automake uses.
	 */
	options.filename = g_strdup ("oci_configtest_debug.log");

	(void)cc_oci_log_init(&options);

	s = make_runtime_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
