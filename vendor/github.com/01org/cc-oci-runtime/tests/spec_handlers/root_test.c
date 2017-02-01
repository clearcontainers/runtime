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
#include <glib/gstdio.h>

#include "../test_common.h"
#include "../../src/logging.h"


extern struct spec_handler root_spec_handler;

/*
* root json needs:
* - path
* root json optional:
* - readonly
*/
static struct spec_handler_test tests[] = {
	{ TEST_DATA_DIR "/root-no-path.json"         , false },
	{ TEST_DATA_DIR "/root-no-readonly.json"     , true  },
	{ TEST_DATA_DIR "/root-readonly-false.json"  , true  },
	{ TEST_DATA_DIR "/root-path-enoent.json"     , false },
	{ TEST_DATA_DIR "/root-path-invalid.json"    , false },
	{ TEST_DATA_DIR "/root-path-wrong-type.json" , false },
	{ TEST_DATA_DIR "/root.json"                 , true  },
	{ NULL, false },
};


START_TEST(test_root_handle_section) {
	gchar *tmpdir = g_dir_make_tmp (NULL, NULL);

	/* XXX: the upper-case value must match that in the test json
	 * files!
	 */
	gchar *root_dir = g_build_path ("/", tmpdir, "ROOTFS", NULL);

	/* Used to test that the spec handler detects the file type is
	 * incorrect (since it should be a directory)
	 */
	g_autofree gchar *root_file = g_build_path ("/", tmpdir,
            "ROOTFILE", NULL);

	/* create fake rootfs directory */
	ck_assert (! g_mkdir (root_dir, 0x750));

	ck_assert (g_file_set_contents (root_file, "", -1, NULL));

	/* move to parent directory (as the runtime does, to allow it to
	 * find the rootfs below it).
	 */
	ck_assert (! g_chdir (tmpdir));

	test_spec_handler(&root_spec_handler, tests);

	/* clean up */
	ck_assert (! g_remove (root_file));
	ck_assert (! g_remove (root_dir));
	ck_assert (! g_remove (tmpdir));

	g_free (root_dir);
	g_free (tmpdir);

} END_TEST

Suite* make_root_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST(test_root_handle_section, s);

	return s;
}

int main(void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };
	gchar *cwd = g_get_current_dir();

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_build_path ("/",
            cwd,
            "root_test_debug.log", NULL);
	(void)cc_oci_log_init(&options);

	s = make_root_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	g_free (cwd);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
