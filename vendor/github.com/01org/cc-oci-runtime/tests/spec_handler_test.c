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

#include "test_common.h"
#include "../src/logging.h"
#include "../src/oci.h"
#include "../src/spec_handler.h"


START_TEST(test_get_spec_vm_from_cfg_file) {
	struct cc_oci_config *config = NULL;

	ck_assert (! get_spec_vm_from_cfg_file (config));

	config = cc_oci_config_create ();
	create_fake_test_files();

	ck_assert (get_spec_vm_from_cfg_file (config));
	ck_assert (get_spec_vm_from_cfg_file (config));

	remove_fake_test_files();
	cc_oci_config_free (config);
} END_TEST

Suite* make_state_suite(void) {
	Suite* s = suite_create(__FILE__);
	ADD_TEST(test_get_spec_vm_from_cfg_file, s);

	return s;
}

int main(void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_strdup ("spec_handler_debug.log");
	(void)cc_oci_log_init(&options);

	s = make_state_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}

