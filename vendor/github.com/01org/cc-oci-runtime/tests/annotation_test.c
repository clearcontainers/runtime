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
#include "../src/oci.h"
#include "../src/logging.h"

void cc_oci_annotation_free (struct oci_cfg_annotation *a);
void cc_oci_annotations_free_all (GSList *annotations);

START_TEST(test_cc_oci_annotation_free) {
	struct oci_cfg_annotation* a;

	cc_oci_annotation_free(NULL);

	/* memory leaks will be detected by valgrind */
	a = g_new0(struct oci_cfg_annotation, 1);
	cc_oci_annotation_free(a);

	a = g_new0(struct oci_cfg_annotation, 1);
	a->key = g_strdup("test");
	cc_oci_annotation_free(a);

	a = g_new0(struct oci_cfg_annotation, 1);
	a->key = g_strdup("test");
	a->value = g_strdup("test");
	cc_oci_annotation_free(a);
} END_TEST

START_TEST(test_cc_oci_annotations_free_all) {
	GSList* list = NULL;
	struct oci_cfg_annotation* a;

	cc_oci_annotations_free_all(NULL);

	list = g_slist_append(list, NULL);

	/* memory leaks will be detected by valgrind */
	a = g_new0(struct oci_cfg_annotation, 1);
	list = g_slist_append(list, a);

	a = g_new0(struct oci_cfg_annotation, 1);
	a->key = g_strdup("test");
	a->value = g_strdup("test");
	list = g_slist_append(list, a);

	cc_oci_annotations_free_all(list);

} END_TEST

Suite* make_annotation_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST(test_cc_oci_annotation_free, s);
	ADD_TEST(test_cc_oci_annotations_free_all, s);

	return s;
}

int main (void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_strdup ("annotation_test_debug.log");
	(void)cc_oci_log_init(&options);

	s = make_annotation_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
