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
#include "../src/semver.h"

START_TEST(test_cc_oci_semver_cmp_compat) {
	ck_assert (cc_oci_semver_cmp ("1.9.0", "1.10.0") == 0);
	ck_assert (cc_oci_semver_cmp ("1.9.7", "1.10.5") == 0);
	ck_assert (cc_oci_semver_cmp ("1.99.7", "1.0.99") == 0);
	ck_assert (cc_oci_semver_cmp ("1.10.0", "1.9.0") == 0);
	ck_assert (cc_oci_semver_cmp ("0.99.99-rc2", "1.0.0-rc1") < 0);
	ck_assert (cc_oci_semver_cmp ("1.99.0", "2.1.0") < 0);
	ck_assert (cc_oci_semver_cmp ("2.1.7", "1.30.9") > 0);
	ck_assert (cc_oci_semver_cmp ("2.0.0", "0.0.1") > 0);
} END_TEST

START_TEST(test_cc_oci_string_is_numeric) {
	ck_assert (! cc_oci_string_is_numeric(NULL));
	ck_assert (! cc_oci_string_is_numeric("abc"));
	ck_assert (! cc_oci_string_is_numeric("1e5"));
	ck_assert (! cc_oci_string_is_numeric("#@$!"));
	ck_assert (cc_oci_string_is_numeric("5289"));
} END_TEST

Suite* make_semver_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST(test_cc_oci_semver_cmp_compat, s);
	ADD_TEST(test_cc_oci_string_is_numeric, s);

	return s;
}

int main (void) {
	int number_failed;
	Suite* s;
	SRunner* sr;

	s = make_semver_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
