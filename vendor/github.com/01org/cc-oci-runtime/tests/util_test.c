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
#include <fcntl.h>

#include "test_common.h"
#include "../src/util.h"
#include "../src/logging.h"
#include "../src/json.h"

START_TEST(test_cc_oci_rm_rf) {

	gchar *tmpdir = g_dir_make_tmp (NULL, NULL);

	ck_assert (tmpdir);

	ck_assert (! cc_oci_rm_rf (""));
	ck_assert (! cc_oci_rm_rf (NULL));

	ck_assert (cc_oci_rm_rf (tmpdir));

	g_free (tmpdir);

} END_TEST

START_TEST(test_cc_oci_replace_string) {
	gchar    *str = g_strdup ("");
	gboolean  ret;

	/* NOP tests */
	ck_assert (cc_oci_replace_string (&str, NULL, NULL));
	ck_assert (cc_oci_replace_string (&str, NULL, ""));
	ck_assert (cc_oci_replace_string (&str, "", NULL));
	ck_assert (cc_oci_replace_string (&str, "", ""));

	g_free (str);

	/* ensure if from string not found, string is not disrupted */
	str = g_strdup ("hello");
	ret = cc_oci_replace_string (&str, "@hello@", "world");
	ck_assert (ret);
	ck_assert (! g_strcmp0 (str, "hello"));
	g_free (str);

	/* replacement taking up entire string */
	str = g_strdup ("@hello@");
	ret = cc_oci_replace_string (&str, "@hello@", "world");
	ck_assert (ret);
	ck_assert (! g_strcmp0 (str, "world"));
	g_free (str);

	/* replacement at start of string */
	str = g_strdup ("@hello@foo");
	ret = cc_oci_replace_string (&str, "@hello@", "world");
	ck_assert (ret);
	ck_assert (! g_strcmp0 (str, "worldfoo"));
	g_free (str);

	/* replacement at end of string */
	str = g_strdup ("foo@hello@");
	ret = cc_oci_replace_string (&str, "@hello@", "world");
	ck_assert (ret);
	ck_assert (! g_strcmp0 (str, "fooworld"));
	g_free (str);

	/* replacement mid-string */
	str = g_strdup ("foo@hello@bar");
	ret = cc_oci_replace_string (&str, "@hello@", "world");
	ck_assert (ret);
	ck_assert (! g_strcmp0 (str, "fooworldbar"));
	g_free (str);

	/* ensure deleting the from value works */
	str = g_strdup ("foo@hello@bar");
	ret = cc_oci_replace_string (&str, "@hello@", "");
	ck_assert (ret);
	ck_assert (! g_strcmp0 (str, "foobar"));
	g_free (str);

} END_TEST

START_TEST(test_cc_oci_create_pidfile) {
	gchar *tmpdir = g_dir_make_tmp (NULL, NULL);
	gchar *tmpfile = g_build_path ("/", tmpdir, "foo.pid", NULL);
	gchar *contents;
	gboolean ret;

	ck_assert (! cc_oci_create_pidfile (NULL, 0));
	ck_assert (! cc_oci_create_pidfile ("", 0));
	ck_assert (! cc_oci_create_pidfile (NULL, -1));
	ck_assert (! cc_oci_create_pidfile ("", -1));

	ck_assert (! cc_oci_create_pidfile ("foo", 0));
	ck_assert (! cc_oci_create_pidfile ("foo", -1));

	ck_assert (! cc_oci_create_pidfile ("no-leading-slash", 123));
	ck_assert (! cc_oci_create_pidfile (tmpfile, -1));

	ck_assert (cc_oci_create_pidfile (tmpfile, 123));

	/* check contents */
	ret = g_file_get_contents (tmpfile, &contents, NULL, NULL);
	ck_assert (ret);
	ck_assert (! g_strcmp0 (contents, "123"));
	g_free (contents);

	/* Ensure existing file overwritten */
	ck_assert (cc_oci_create_pidfile (tmpfile, 456));

	/* check contents */
	ret = g_file_get_contents (tmpfile, &contents, NULL, NULL);
	ck_assert (ret);
	ck_assert (! g_strcmp0 (contents, "456"));
	g_free (contents);

	ck_assert (! g_remove (tmpfile));
	ck_assert (! g_remove (tmpdir));

	g_free (tmpfile);
	g_free (tmpdir);

} END_TEST

START_TEST(test_cc_oci_file_to_strv) {
	gboolean ret;
	gchar **strv = NULL;
	gchar *tmpdir = g_dir_make_tmp (NULL, NULL);
	gchar *tmpfile = g_build_path ("/", tmpdir, "foo.txt", NULL);

	ck_assert (! cc_oci_file_to_strv (NULL, NULL));
	ck_assert (! cc_oci_file_to_strv ("", NULL));

	ck_assert (! cc_oci_file_to_strv ("foo", NULL));
	ck_assert (! cc_oci_file_to_strv ("foo", &strv));

	/* tmpfile does not exist */
	ck_assert (! cc_oci_file_to_strv (tmpfile, &strv));

	/* file doesn't contain NL */
	ret = g_file_set_contents (tmpfile, "", -1, NULL);
	ck_assert (ret);
	ck_assert (! cc_oci_file_to_strv (tmpfile, &strv));

	/* file only contains a single NL */
	ret = g_file_set_contents (tmpfile, "\n", -1, NULL);
	ck_assert (ret);
	ck_assert (cc_oci_file_to_strv (tmpfile, &strv));

	/* we expect a single field containing an empty string */
	ck_assert (strv[0]);
	ck_assert (! g_strcmp0 (strv[0], ""));

	ck_assert (! strv[1]);

	g_strfreev (strv); strv = NULL;

	/* no trailing NL */
	ret = g_file_set_contents (tmpfile, "hello\nworld", -1, NULL);
	ck_assert (ret);

	ck_assert (cc_oci_file_to_strv (tmpfile, &strv));
	ck_assert (strv);

	ck_assert (strv[0]);
	ck_assert (! g_strcmp0 (strv[0], "hello"));

	ck_assert (strv[1]);
	ck_assert (! g_strcmp0 (strv[1], "world"));

	ck_assert (! strv[2]);

	g_strfreev (strv); strv = NULL;

	/* leading NL */
	ret = g_file_set_contents (tmpfile, "\nhello", -1, NULL);
	ck_assert (ret);

	ck_assert (cc_oci_file_to_strv (tmpfile, &strv));
	ck_assert (strv);

	ck_assert (strv[0]);
	ck_assert (! g_strcmp0 (strv[0], ""));

	ck_assert (strv[1]);
	ck_assert (! g_strcmp0 (strv[1], "hello"));

	ck_assert (! strv[2]);

	g_strfreev (strv); strv = NULL;

	/* with trailing NL */
	ret = g_file_set_contents (tmpfile, "hello\nworld\n", -1, NULL);
	ck_assert (ret);

	ck_assert (cc_oci_file_to_strv (tmpfile, &strv));
	ck_assert (strv);

	ck_assert (strv[0]);
	ck_assert (! g_strcmp0 (strv[0], "hello"));

	ck_assert (strv[1]);
	ck_assert (! g_strcmp0 (strv[1], "world"));

	ck_assert (! strv[2]);

	g_strfreev (strv); strv = NULL;

	g_free (tmpfile);
	g_free (tmpdir);

} END_TEST

START_TEST(test_cc_oci_get_iso8601_timestamp) {
	gchar *t;

	t = cc_oci_get_iso8601_timestamp ();
	ck_assert (t);

	ck_assert (check_timestamp_format (t));

	g_free (t);

} END_TEST

/* FIXME: more tests required for:
 *
 * - object containing an array (of various types).
 * - object containing an object (containing various types).
 * - object containing a double.
 */
START_TEST(test_cc_oci_json_obj_to_string) {
	gchar *str;
	JsonObject *obj;
	gboolean ret;
	gchar **fields;
	gchar *value;

	ck_assert (! cc_oci_json_obj_to_string (NULL, false, NULL));
	ck_assert (! cc_oci_json_obj_to_string (NULL, true, NULL));

	obj = json_object_new ();

	/* empty object, non-pretty */
	str = cc_oci_json_obj_to_string (obj, false, NULL);
	ck_assert (str);

	ret = g_regex_match_simple (
			/* pair of braces containing optional
			 * whitespace.
			 */
			"{\\s*}",
			str, 0, 0);
	ck_assert (ret);
	g_free (str);

	/* empty object, pretty */
	str = cc_oci_json_obj_to_string (obj, true, NULL);
	ck_assert (str);

	ret = g_regex_match_simple (
			/* pair of braces containing optional
			 * whitespace.
			 */
			"{\\s*}",
			str, 0, 0);
	ck_assert (ret);
	g_free (str);

	/* non-empty object, non-pretty */
	json_object_set_string_member (obj, "test-string-set", "foo");
	json_object_set_string_member (obj, "test-string-empty", "");

	json_object_set_boolean_member (obj, "test-bool-false", false);
	json_object_set_boolean_member (obj, "test-bool-true", true);

	json_object_set_int_member (obj, "test-int-max", G_MAXINT64);
	json_object_set_int_member (obj, "test-int-min", G_MININT64);


	json_object_set_null_member (obj, "test-null");

	str = cc_oci_json_obj_to_string (obj, false, NULL);
	ck_assert (str);

	fields = g_strsplit (str, ",", -1);
	ck_assert (fields);

	g_free (str);

	ck_assert (strv_contains_regex (fields, "test-string-set\\S*foo"));
	ck_assert (strv_contains_regex (fields, "test-string-empty\\S*\"\""));

	ck_assert (strv_contains_regex (fields, "test-bool-true\\S*true"));
	ck_assert (strv_contains_regex (fields, "test-bool-false\\S*false"));

	value = g_strdup_printf ("test-int-max\\S*"
			"%" G_GINT64_FORMAT,
			G_MAXINT64);
	ck_assert (value);

	ck_assert (strv_contains_regex (fields, value));
	g_free (value);

	value = g_strdup_printf ("test-int-min\\S*"
			"%" G_GINT64_FORMAT,
			G_MININT64);
	ck_assert (value);

	ck_assert (strv_contains_regex (fields, value));
	g_free (value);

	g_strfreev (fields);

	/* clean up */
	json_object_unref (obj);

} END_TEST

/* FIXME: more tests required for:
 *
 * - array containing an array (of various types).
 * - array containing an object (containing various types).
 */
START_TEST(test_cc_oci_json_arr_to_string) {
	gchar *str;
	JsonArray *array;
	gboolean ret;

	ck_assert (! cc_oci_json_arr_to_string (NULL, false));
	ck_assert (! cc_oci_json_arr_to_string (NULL, true));

	array = json_array_new ();

	/* empty array, non-pretty */
	str = cc_oci_json_arr_to_string (array, false);
	ck_assert (str);
	ck_assert (! g_strcmp0 (str, "[]"));
	g_free (str);

	/* empty array, pretty */
	str = cc_oci_json_arr_to_string (array, true);
	ck_assert (str);

	ret = g_regex_match_simple (
			/* pair of braces containing optional
			 * whitespace.
			 */
			"[\\s*]",
			str, 0, 0);
	ck_assert (ret);

	g_free (str);

	json_array_add_null_element (array);
	json_array_add_null_element (array);
	json_array_add_boolean_element (array, false);
	json_array_add_boolean_element (array, true);
	json_array_add_int_element (array, 123);
	json_array_add_double_element (array, 3.1412);
	json_array_add_string_element (array, "foo bar");

	str = cc_oci_json_arr_to_string (array, false);
	ck_assert (str);

	ret = g_regex_match_simple (
			"[\\s*"                  /* opening bracket */
			"\\s*null\\s*,"          /* null */
			"\\s*null\\s*,"          /* null */
			"\\s*false\\s*,"         /* false */
			"\\s*true\\s*,"          /* true */
			"\\s*123\\s*,"           /* int */
			"\\s*3\\.1412\\s*,"      /* double */
			"\\s*\"foo\\ bar\"\\s*," /* string */
			"\\s*]",                 /* closing bracket */
			str, 0, 0);
	ck_assert (ret);
	g_free (str);

	/* clean up */
	json_array_unref (array);

} END_TEST

START_TEST(test_cc_oci_get_signum) {
	ck_assert(cc_oci_get_signum(NULL) == -1);
	ck_assert(cc_oci_get_signum("NOSIG") == -1);
	ck_assert(cc_oci_get_signum("") == -1);
	ck_assert(cc_oci_get_signum("SIGTERM") != -1);
	ck_assert(cc_oci_get_signum("TERM") != -1);
} END_TEST

START_TEST(test_cc_oci_node_dump) {
	GNode* node;
	cc_oci_node_dump(NULL);
	cc_oci_json_parse(&node, TEST_DATA_DIR "/node.json");
	ck_assert(node);
	cc_oci_node_dump(node);
	g_free_node(node);
} END_TEST

/*
 * TODO:
 *
 * We should really explore maximum paths values:
 *
 * - absolute maximum: PATH_MAX
 * - relative maximum: _POSIX_PATH_MAX
 * - pathconf(3)
 */
START_TEST(test_cc_oci_resolve_path) {
	gchar *tmpdir;
	gchar *file;
	gchar *slink;
	gchar *path;
	const char *d = NULL;

	tmpdir = g_dir_make_tmp (NULL, NULL);
	ck_assert (tmpdir);

	file = g_build_path ("/", tmpdir, "foo", NULL);
	ck_assert (file);

	slink = g_build_path ("/", tmpdir, "symlink", NULL);
	ck_assert (slink);

	/* create a symlink */
	ck_assert (! symlink (file, slink));

	ck_assert (! cc_oci_resolve_path (NULL));
	ck_assert (! cc_oci_resolve_path (""));
	ck_assert (! cc_oci_resolve_path ("not a path"));
	ck_assert (! cc_oci_resolve_path ("/does/not/exist"));

	/*******************************/
	/* check a known valid path */

	d = g_get_tmp_dir ();
	ck_assert (d);

	path = cc_oci_resolve_path (d);
	ck_assert (path);
	ck_assert (! g_strcmp0 (path, d));
	g_free (path);

	/*******************************/
	/* file doesn't exist */

	path = cc_oci_resolve_path (file);
	ck_assert (! path);

	/*******************************/
	/* create valid file */

	ck_assert (g_file_set_contents (file, "", -1, NULL));

	path = cc_oci_resolve_path (file);
	ck_assert (path);
	ck_assert (! g_strcmp0 (path, file));
	g_free (path);

	/*******************************/
	/* check a broken symlink */

	/* break the symlink by removing the file it points to */
	ck_assert (! g_remove (file));

	path = cc_oci_resolve_path (slink);
	ck_assert (! path);

	/*******************************/
	/* check a valid symlink */

	/* re create the file */
	ck_assert (g_file_set_contents (file, "", -1, NULL));

	path = cc_oci_resolve_path (slink);
	ck_assert (path);

	ck_assert (! g_strcmp0 (path, file));
	g_free (path);

	/* clean up */
	ck_assert (! g_remove (file));
	ck_assert (! g_remove (slink));
	ck_assert (! g_remove (tmpdir));
	g_free (tmpdir);
	g_free (file);
	g_free (slink);

} END_TEST

START_TEST(test_cc_oci_enable_networking) {
	gboolean expected = ! geteuid ();
	ck_assert (cc_oci_enable_networking () == expected);

} END_TEST

START_TEST(test_dup_over_stdio) {
	int fd = -1;

	/* save stdin fd  */
	int saved_stdin = dup(STDIN_FILENO);

	/* fd pointer is NULL */
	ck_assert(! dup_over_stdio(NULL));

	/* fd number is -1 */
	fd = -1;
	ck_assert(! dup_over_stdio(&fd));


	/* dup not used but open STDIN fd */
	fd = STDIN_FILENO ;
	ck_assert(dup_over_stdio(&fd));
	ck_assert_int_gt(fd,2);
	ck_assert( fcntl(STDIN_FILENO, F_GETFD) == -1 );
	/* now the fd is higher than 2 , the fd number will not change*/
	int saved_fd_no = fd;
	ck_assert(dup_over_stdio(&fd));
	ck_assert_int_eq(fd,saved_fd_no);
	close(fd);
	/* fd closed*/
	ck_assert(!dup_over_stdio(&fd));

	/* restore stdin fd */
	dup2(saved_stdin, STDIN_FILENO);
	close(saved_stdin);
} END_TEST

Suite* make_util_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST(test_cc_oci_rm_rf, s);
	ADD_TEST(test_cc_oci_replace_string, s);
	ADD_TEST(test_cc_oci_create_pidfile, s);
	ADD_TEST(test_cc_oci_file_to_strv, s);
	ADD_TEST(test_cc_oci_get_iso8601_timestamp, s);
	ADD_TEST(test_cc_oci_json_obj_to_string, s);
	ADD_TEST(test_cc_oci_json_arr_to_string, s);
	ADD_TEST(test_cc_oci_get_signum, s);
	ADD_TEST(test_cc_oci_node_dump, s);
	ADD_TEST(test_cc_oci_resolve_path, s);
	ADD_TEST(test_cc_oci_enable_networking, s);
	ADD_TEST(test_dup_over_stdio, s);

	return s;
}

int main(void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_strdup ("util_test_debug.log");
	(void)cc_oci_log_init(&options);

	s = make_util_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
