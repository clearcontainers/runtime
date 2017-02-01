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
#include <json-glib/json-glib.h>
#include <json-glib/json-gobject.h>

#include "test_common.h"
#include "logging.h"
#include "oci.h"

void
cc_oci_error (const char *file,
		int line_number,
		const char *function,
		const char *fmt,
		...);

START_TEST(test_cc_oci_log_init) {
	const gchar *logfile = "logging_test_debug.log";
	gboolean ret;
	gchar *contents = NULL;
	gchar **lines = NULL;
	GError *error = NULL;
	struct cc_log_options options = { 0 };
	gchar *tmpdir = g_dir_make_tmp (NULL, NULL);
	JsonParser *parser = NULL;
	JsonReader *reader = NULL;
	const gchar *value;

	options.enable_debug = true;

	options.use_json = false;

	/* we want the logfile created below tmpdir */
	options.filename = g_build_path ("/", tmpdir, logfile, NULL);

	/* setup logging */
	ck_assert (cc_oci_log_init(&options));

	/* ensure log file doesn't exist before any logging calls made */
	ck_assert (! g_file_test (options.filename, G_FILE_TEST_EXISTS));

	/* ensure global logging disabled */
	ck_assert (! options.global_logfile);

	g_info ("G_LOG_LEVEL_INFO: %s (int=%d)", "hello world", 42);
	g_message ("G_LOG_LEVEL_MESSAGE: %s (int=%d)", "baz", 321);
	g_warning ("G_LOG_LEVEL_WARNING: %s (int=%d)", "testing", 123);
	g_critical("G_LOG_LEVEL_CRITICAL: %s (int=%d)", "another test", -1);

	g_print ("g_print data will NOT BE LOGGED");

	/* Ensure log file created */
	ck_assert (g_file_test (options.filename, G_FILE_TEST_EXISTS));

	ret = g_file_get_contents (options.filename, &contents, NULL, &error);
	ck_assert (ret);
	ck_assert (! error);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	/* XXX: note the implicit test that the logfile gets appended to */
	ck_assert (lines[0]);
	ck_assert (g_str_has_suffix (lines[0], "G_LOG_LEVEL_INFO: hello world (int=42)"));

	ck_assert (lines[1]);
	ck_assert (g_str_has_suffix (lines[1], "G_LOG_LEVEL_MESSAGE: baz (int=321)"));

	ck_assert (lines[2]);
	ck_assert (g_str_has_suffix (lines[2], "G_LOG_LEVEL_WARNING: testing (int=123)"));

	ck_assert (lines[3]);
	ck_assert (g_str_has_suffix (lines[3], "G_LOG_LEVEL_CRITICAL: another test (int=-1)"));

	/* last field (artifact of splitting on NL) */
	ck_assert (lines[4]);
	ck_assert (! g_strcmp0 (lines[4], ""));

	ck_assert (! lines[5]);

	g_free (contents);
	g_strfreev (lines);

	ck_assert (! g_remove (options.filename));

	cc_oci_error (__FILE__, __LINE__, __func__, "testing cc_oci_error");

	/************************************************************/
	/* Now, test g_debug() handling */

	/* Disable g_debug () messages */
	options.enable_debug = false;

	g_debug ("WILL NOT BE LOGGED");

	ck_assert (! g_file_test (options.filename, G_FILE_TEST_EXISTS));

	/* re-enable debug messages */
	options.enable_debug = true;

	g_debug ("G_LOG_LEVEL_DEBUG: %s (int=%d)", "!de bug, da bug!", 13);

	ret = g_file_get_contents (options.filename, &contents, NULL, &error);
	ck_assert (ret);
	ck_assert (! error);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	ck_assert (lines[0]);
	ck_assert (g_str_has_suffix (lines[0], "G_LOG_LEVEL_DEBUG: !de bug, da bug! (int=13)"));

	/* last field (artifact of splitting on NL) */
	ck_assert (lines[1]);
	ck_assert (! g_strcmp0 (lines[1], ""));

	ck_assert (! lines[2]);

	g_free (contents);
	g_strfreev (lines);

	ck_assert (! g_remove (options.filename));

	/************************************************************/
	/* Test JSON logging */

	options.use_json = true;

	g_critical ("this message is in json format");

	ret = g_file_get_contents (options.filename, &contents, NULL, &error);
	ck_assert (ret);
	ck_assert (! error);

	lines = g_strsplit (contents, "\n", -1);
	ck_assert (lines);

	parser = json_parser_new ();
	reader = json_reader_new (NULL);

	ret = json_parser_load_from_data (parser, lines[0], -1, &error);

	ck_assert (ret);
	ck_assert (! error);

	json_reader_set_root (reader, json_parser_get_root (parser));

	ck_assert (json_reader_read_member (reader, "level"));
	value = json_reader_get_string_value (reader);
	ck_assert (! g_strcmp0 (value, "critical"));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "mesg"));
	value = json_reader_get_string_value (reader);
	ck_assert (! g_strcmp0 (value, "this message is in json format"));
	json_reader_end_member (reader);

	ck_assert (json_reader_read_member (reader, "time"));
	value = json_reader_get_string_value (reader);
	ck_assert (check_timestamp_format (value));

	json_reader_end_member (reader);

	g_object_unref (reader);
	g_object_unref (parser);

	g_free (contents);
	g_strfreev (lines);

	/* use global log file */
	options.use_json = false;
	options.global_logfile = g_new0(char, PATH_MAX);
	g_snprintf(options.global_logfile, PATH_MAX, "%s", options.filename);
	ck_assert (cc_oci_log_init(&options));
	g_message("testing g_message with global log file");
	options.use_json = true;
	g_message("testing g_message with global log file");

	/************************************************************/
	/* clean up */

	ck_assert (! g_remove (options.filename));
	ck_assert (! g_remove (tmpdir));
	cc_oci_log_free (&options);
	g_free (tmpdir);

} END_TEST

Suite* make_runtime_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST(test_cc_oci_log_init, s);

	return s;
}

int main(void) {
	int number_failed;
	Suite* s;
	SRunner* sr;

	s = make_runtime_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
