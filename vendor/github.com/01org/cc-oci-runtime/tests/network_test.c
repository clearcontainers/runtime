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

#include <stdbool.h>
#include <stdlib.h>

#include <check.h>
#include <glib.h>

#include "test_common.h"
#include "../src/oci.h"
#include "../src/network.h"
#include "../src/logging.h"
#include "../src/util.h"

START_TEST(test_cc_oci_vm_pause) {
	char *socket_path = NULL;
	pid_t pid = 0;
	char *diname = NULL;

	pid = run_qmp_vm (&socket_path);
	ck_assert (pid > 0);
	ck_assert (socket_path != NULL);

	ck_assert (! cc_oci_vm_pause (NULL, -1));
	ck_assert (! cc_oci_vm_pause (socket_path, -1));
	ck_assert (! cc_oci_vm_pause (NULL, pid));
	ck_assert (! cc_oci_vm_pause (NULL, 0));
	ck_assert (! cc_oci_vm_pause (socket_path, 0));
	ck_assert (! cc_oci_vm_pause ("/path/to/nothingness", pid));

	ck_assert (cc_oci_vm_pause (socket_path, pid));

	kill (pid, SIGTERM);

	diname = g_path_get_dirname (socket_path);
	cc_oci_rm_rf(diname);

	g_free(socket_path);
	g_free(diname);
} END_TEST

START_TEST(test_cc_oci_vm_resume) {
	char *socket_path = NULL;
	pid_t pid = 0;
	char *diname = NULL;

	pid = run_qmp_vm (&socket_path);
	ck_assert (pid > 0);
	ck_assert (socket_path != NULL);

	ck_assert (! cc_oci_vm_resume (NULL, -1));
	ck_assert (! cc_oci_vm_resume (socket_path, -1));
	ck_assert (! cc_oci_vm_resume (NULL, pid));
	ck_assert (! cc_oci_vm_resume (NULL, 0));
	ck_assert (! cc_oci_vm_resume (socket_path, 0));
	ck_assert (! cc_oci_vm_resume ("/path/to/nothingness", pid));

	ck_assert (cc_oci_vm_pause (socket_path, pid));
	ck_assert (cc_oci_vm_resume (socket_path, pid));

	kill (pid, SIGTERM);

	diname = g_path_get_dirname (socket_path);
	cc_oci_rm_rf(diname);

	g_free(socket_path);
	g_free(diname);
} END_TEST

Suite* make_oci_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST_TIMEOUT (test_cc_oci_vm_pause, s, 10);
	ADD_TEST_TIMEOUT (test_cc_oci_vm_resume, s, 10);

	return s;
}

int main (void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_strdup ("network_test_debug.log");
	(void)cc_oci_log_init(&options);

	s = make_oci_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
