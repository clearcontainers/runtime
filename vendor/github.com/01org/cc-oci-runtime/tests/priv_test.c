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

#include <check.h>
#include <glib.h>
#include <glib/gstdio.h>

#include "test_common.h"

#include "command.h"
#include "oci.h"
#include "../src/logging.h"
#include "priv.h"

START_TEST(test_cc_oci_get_priv_level) {
	int                    argc = 2;
	gchar                 *argv[] = { "IGNORED", "arg", NULL };
	struct cc_oci_config  *config = NULL;
	struct subcommand      sub;
	gchar                 *tmpdir = g_dir_make_tmp (NULL, NULL);
	gchar                 *tmpdir_enoent = g_build_path ("/", tmpdir, "foo", NULL);

	config = cc_oci_config_create ();
	ck_assert (config);

	sub.name = "help";
	ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == -1);

	sub.name = "version";
	ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == -1);

	sub.name = "list";
	argv[1] = "--help";
	ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == -1);

	sub.name = "list";
	argv[1] = "-h";
	ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == -1);

	/* set to a non-"--help" value */
	argv[1] = "foo";

	/* root_dir not specified, so root required */
	ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 1);

	config->root_dir = g_strdup ("/");

	if (access (config->root_dir, W_OK) < 0) {
		/* user cannot write to root_dir */
		ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 1);
	} else {
		/* user can write to root_dir */
		ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 0);
	}

	g_free (config->root_dir);
	config->root_dir = g_strdup (tmpdir);

	ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 0);

	/* make directory inaccessible to non-root */
	ck_assert (! g_chmod (tmpdir, 0));

	if (getuid ()) {
		ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 1);
	} else {
		/* root can write to any directory */
		ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 0);
	}

	/* make directory accessible once again */
	ck_assert (! g_chmod (tmpdir, 0755));

	g_free (config->root_dir);

	/* specify a non-existing directory */
	config->root_dir = g_strdup (tmpdir_enoent);

	if (getuid ()) {
		/* parent directory does exist so no extra privs
		 * required.
		 */
		ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 0);
	} else {
		/* root can write to any directory */
		ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 0);
	}

	/* make parent directory inaccessible to non-root again */
	ck_assert (! g_chmod (tmpdir, 0));

	if (getuid ()) {
		ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 1);
	} else {
		/* root can write to any directory */
		ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 0);
	}

	/* make parent directory accessible once again */
	ck_assert (! g_chmod (tmpdir, 0755));

	if (getuid ()) {
		ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 0);
	} else {
		/* root can write to any directory */
		ck_assert (cc_oci_get_priv_level (argc, argv, &sub, config) == 0);
	}

	/* clean up */
	ck_assert (! g_remove (tmpdir));
	g_free (tmpdir);
	g_free (tmpdir_enoent);
	cc_oci_config_free (config);

} END_TEST

Suite* make_priv_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST(test_cc_oci_get_priv_level, s);

	return s;
}

int main(void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_strdup ("priv_test_debug.log");
	(void)cc_oci_log_init(&options);

	s = make_priv_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
