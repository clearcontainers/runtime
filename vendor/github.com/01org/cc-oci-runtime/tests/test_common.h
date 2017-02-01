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

#include <unistd.h>
#include <assert.h>

#include <glib.h>

#include "../src/spec_handler.h"
#include "../src/runtime.h"
#include "../src/state.h"

#define ADD_TEST(test,suite) \
	TCase *tc_##test; \
	tc_##test = tcase_create(#test); \
	tcase_add_test(tc_##test, test); \
	suite_add_tcase(suite, tc_##test);

#define ADD_TEST_TIMEOUT(test,suite,timeout) \
	TCase *tc_##test; \
	tc_##test = tcase_create(#test); \
	tcase_add_test(tc_##test, test); \
	suite_add_tcase(suite, tc_##test); \
	tcase_set_timeout(tc_##test, timeout);

/**
 * Run the block following this macro with stdout and stderr redirected
 * to a temporary file.
 *
 * \param[out] _name_ptr Pointer to string.
 *
 * \note \p _name_ptr will be set to the name of the temporary file
 * containing any output from the code block. It is the callers
 * responsibility to free the string by calling \c g_free().
 * 
 * Example invocation:
 *
 * \code
 *
 *   gchar *outfile = NULL;
 *
 *   SAVE_OUTPUT (outfile) {
 *       system ("echo hello");
 *   }
 *
 *   // check contents of "outfile"
 *
 *   g_free (outfile);
 *   g_remove (outfile);
 *
 * \endcode
 *
 * \note This macro was inspired by the "TEST_DIVERT_STDOUT_FD" macro
 * from the libnih version 1.0.3 package (GPLv2), written by
 * Scott James Remnant (scott@netsplit.com).
 * See https://github.com/keybuk/libnih for further details. At the time
 * of writing, the "TEST_DIVERT_STDOUT_FD" macro could be found at:
 *
 *   https://github.com/keybuk/libnih/blob/1.0.3/nih/test_divert.h#L38
 */
#define SAVE_OUTPUT(_name_ptr) \
	for (int _count = 0, \
	     _saved_stdout = dup (STDOUT_FILENO), \
	     _saved_stderr = dup (STDERR_FILENO), \
	     _fd = g_file_open_tmp (NULL, &_name_ptr, NULL); \
	     _count < 3; _count++) \
		if (_count == 0) { \
			/* setup */ \
			fflush (NULL); \
			assert (_saved_stdout != -1); \
			assert (_saved_stderr != -1); \
			assert (_fd != -1); \
			/* divert stdout+stderr to the tmp file */ \
			assert (dup2 (_fd, STDOUT_FILENO) != -1); \
			assert (dup2 (_fd, STDERR_FILENO) != -1); \
		} else if (_count == 2) { \
			/* teardown */ \
			fflush (NULL); \
			/* switch stdout+stderr back to their \
			 * original file descriptors. \
			 */ \
			assert (dup2 (_saved_stdout, STDOUT_FILENO) != -1); \
			assert (dup2 (_saved_stderr, STDERR_FILENO) != -1); \
			close (_fd); \
			close (_saved_stdout); \
			close (_saved_stderr); \
		} else

struct spec_handler_test {
	const char* file;
	bool test_result;
};

gboolean strv_contains_regex (gchar **strv, const char *regex);
gboolean check_timestamp_format (const gchar *timestamp);
GNode *node_find_child(GNode* node, const gchar* data);
void test_spec_handler(struct spec_handler* handler,
    struct spec_handler_test* tests);
gboolean test_helper_create_state_file (const char *name,
		const char *root_dir,
		struct cc_oci_config *config);
pid_t run_qmp_vm(char **socket_path);
void create_fake_test_files(void);
void remove_fake_test_files(void);
