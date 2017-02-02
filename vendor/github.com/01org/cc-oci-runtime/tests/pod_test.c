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
#include "pod.h"
#include "oci.h"
#include "oci-config.h"

const gchar *cc_pod_container_id(struct cc_oci_config *config);
gboolean cc_pod_is_sandbox(struct cc_oci_config *config);
gboolean cc_pod_is_vm(struct cc_oci_config *config);

START_TEST(test_cc_pod_container_id) {
	struct cc_oci_config *config = NULL;

	ck_assert(!cc_pod_container_id(config));

	config = cc_oci_config_create ();
	ck_assert (config);

	config->optarg_container_id = "pod1";

	ck_assert(! g_strcmp0(cc_pod_container_id(config), "pod1"));

	config->pod = g_malloc0 (sizeof (struct cc_pod));
	ck_assert(config->pod);

	config->pod->sandbox_name = g_strdup("sandbox1");
	config->pod->sandbox = false;
	ck_assert(! g_strcmp0(cc_pod_container_id(config), "sandbox1"));

	config->pod->sandbox = true;
	ck_assert(! g_strcmp0(cc_pod_container_id(config), "pod1"));

	/* clean up */
	cc_oci_config_free (config);
} END_TEST

START_TEST(test_cc_pod_is_sandbox) {
	struct cc_oci_config *config = NULL;

	ck_assert(!cc_pod_is_sandbox(config));

	config = cc_oci_config_create ();
	ck_assert(config);
	ck_assert(!cc_pod_is_sandbox(config));

	config->pod = g_malloc0 (sizeof (struct cc_pod));
	ck_assert(config->pod);

	config->pod->sandbox = false;
	ck_assert(!cc_pod_is_sandbox(config));

	config->pod->sandbox = true;
	ck_assert(cc_pod_is_sandbox(config));

	/* clean up */
	cc_oci_config_free (config);
} END_TEST

START_TEST(test_cc_pod_is_vm) {
	struct cc_oci_config *config = NULL;

	ck_assert(cc_pod_is_vm(config));

	config = cc_oci_config_create ();
	ck_assert(config);
	ck_assert(cc_pod_is_vm(config));

	config->pod = g_malloc0 (sizeof (struct cc_pod));
	ck_assert(config->pod);

	config->pod->sandbox = false;
	ck_assert(!cc_pod_is_vm(config));

	config->pod->sandbox = true;
	ck_assert(cc_pod_is_vm(config));

	/* clean up */
	cc_oci_config_free (config);
} END_TEST

Suite* make_pod_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST (test_cc_pod_container_id, s);
	ADD_TEST (test_cc_pod_is_sandbox, s);
	ADD_TEST (test_cc_pod_is_vm, s);

	return s;
}

int main (void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_strdup ("pod_test_debug.log");
	(void)cc_oci_log_init(&options);

	s = make_pod_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
