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
#include <stdio.h>
#include <sys/types.h>
#include <sys/socket.h>

#include <check.h>
#include <glib.h>
#include <glib/gstdio.h>
#include <fcntl.h>

#include "test_common.h"
#include "../src/logging.h"
#include "../src/process.h"
#include "../src/netlink.h"
#include "../src/util.h"

gboolean cc_oci_cmd_is_shell (const char *cmd);
gboolean cc_run_hook (struct oci_cfg_hook* hook,
		const gchar* state,
		gsize state_length);
gboolean cc_oci_setup_shim (struct cc_oci_config *config,
		int proxy_fd,
		int proxy_io_fd,
		int shim_flock_fd);
GSocketConnection *cc_oci_socket_connection_from_fd (int fd);
gboolean cc_oci_setup_child (struct cc_oci_config *config);
gboolean cc_oci_vm_netcfg_get (struct cc_oci_config *config,
		struct netlink_handle *hndl);
gboolean
cc_shim_launch (struct cc_oci_config *config, int *child_err_fd,
		int *shim_args_fd, int *shim_socket_fd, gboolean initial_workload);

extern GMainLoop *hook_loop;


START_TEST(test_cc_run_hook) {

	struct oci_cfg_hook *hook = NULL;
	g_autofree gchar *cmd = NULL;

	/* XXX: note that the command the tests run must read from
	 * stdin!
	 *
	 * dd was chosen since not only does it read from stdin, but
	 * also accepts arguments that can be tested, and those
	 * arguments can be repeated without generating an error.
	 */
	cmd = g_find_program_in_path ("dd");
	ck_assert (cmd);

	/*******************************/

	hook_loop = g_main_loop_new (NULL, 0);
	ck_assert (hook_loop);

	hook = g_new0 (struct oci_cfg_hook, 1);
	ck_assert (hook);

	g_strlcpy (hook->path, "dd", sizeof (hook->path));

	/* fails since full path not specified */
	ck_assert (! cc_run_hook (hook, "", 1));

	g_main_loop_unref (hook_loop);
	cc_oci_hook_free (hook);

	/*******************************/
	/* full path specified, no args */

	hook_loop = g_main_loop_new (NULL, 0);
	ck_assert (hook_loop);

	hook = g_new0 (struct oci_cfg_hook, 1);
	ck_assert (hook);

	g_strlcpy (hook->path, cmd, sizeof (hook->path));

	ck_assert (cc_run_hook (hook, "", 1));

	g_main_loop_unref (hook_loop);
	cc_oci_hook_free (hook);

	/*******************************/
	/* full path specified, cmd arg */

	hook_loop = g_main_loop_new (NULL, 0);
	ck_assert (hook_loop);

	hook = g_new0 (struct oci_cfg_hook, 1);
	ck_assert (hook);

	g_strlcpy (hook->path, cmd, sizeof (hook->path));

	hook->args = g_new0 (gchar *, 2);
	ck_assert (hook->args);
	hook->args[0] = g_strdup (cmd);

	ck_assert (cc_run_hook (hook, "", 1));

	g_main_loop_unref (hook_loop);
	cc_oci_hook_free (hook);

	/*******************************/
	/* full path, cmd arg + 1 arg */

	hook_loop = g_main_loop_new (NULL, 0);
	ck_assert (hook_loop);

	hook = g_new0 (struct oci_cfg_hook, 1);
	ck_assert (hook);

	g_strlcpy (hook->path, cmd, sizeof (hook->path));

	hook->args = g_new0 (gchar *, 3);
	ck_assert (hook->args);
	hook->args[0] = g_strdup (cmd);
	hook->args[1] = g_strdup ("bs=1");

	ck_assert (cc_run_hook (hook, "", 1));

	g_main_loop_unref (hook_loop);
	cc_oci_hook_free (hook);

	/*******************************/
	/* full path, cmd arg + 2 args */

	hook_loop = g_main_loop_new (NULL, 0);
	ck_assert (hook_loop);

	hook = g_new0 (struct oci_cfg_hook, 1);
	ck_assert (hook);

	g_strlcpy (hook->path, cmd, sizeof (hook->path));

	hook->args = g_new0 (gchar *, 4);
	ck_assert (hook->args);
	hook->args[0] = g_strdup (cmd);
	hook->args[1] = g_strdup ("bs=1");
	hook->args[2] = g_strdup ("bs=1");

	ck_assert (cc_run_hook (hook, "", 1));

	g_main_loop_unref (hook_loop);
	cc_oci_hook_free (hook);

	/*******************************/
	/* full path, cmd arg + 3 args */

	hook_loop = g_main_loop_new (NULL, 0);
	ck_assert (hook_loop);

	hook = g_new0 (struct oci_cfg_hook, 1);
	ck_assert (hook);

	g_strlcpy (hook->path, cmd, sizeof (hook->path));

	hook->args = g_new0 (gchar *, 5);
	ck_assert (hook->args);
	hook->args[0] = g_strdup (cmd);
	hook->args[1] = g_strdup ("bs=1");
	hook->args[2] = g_strdup ("bs=1");
	hook->args[3] = g_strdup ("bs=1");

	ck_assert (cc_run_hook (hook, "", 1));

	g_main_loop_unref (hook_loop);
	cc_oci_hook_free (hook);

	/*******************************/

} END_TEST

START_TEST(test_cc_oci_setup_shim) {
	struct cc_oci_config config = { { 0 } };
	char tmpf1[] = "/tmp/.tmpXXXXXX";
	char tmpf2[] = "/tmp/.tmpXXXXXX";
	char tmpf3[] = "/tmp/.tmpXXXXXX";
	int tmpf1_fd = -1;
	int tmpf2_fd = -1;
	int flock_fd = -1;

	ck_assert (! cc_oci_setup_shim (NULL, -1, -1, -1));

	tmpf1_fd = g_mkstemp (tmpf1);
	ck_assert (tmpf1_fd >= 0);

	ck_assert (! cc_oci_setup_shim (NULL, tmpf1_fd, -1, -1));

	tmpf2_fd = g_mkstemp (tmpf2);
	ck_assert (tmpf2_fd >= 0);

	ck_assert (! cc_oci_setup_shim (NULL, tmpf1_fd, tmpf2_fd, -1));

	flock_fd = g_mkstemp (tmpf3);
	ck_assert (flock_fd >= 0);

	ck_assert (! cc_oci_setup_shim (NULL, tmpf1_fd, tmpf2_fd, flock_fd));

	config.oci.process.terminal = false;
	ck_assert (cc_oci_setup_shim (&config, tmpf1_fd, tmpf2_fd, flock_fd));

	config.oci.process.terminal = true;
	config.console = g_strdup("/dev/ptmx");
	ck_assert (! cc_oci_setup_shim (&config, tmpf1_fd, tmpf2_fd, flock_fd));

	g_free (config.console);

	if (tmpf1_fd > -1) { close (tmpf1_fd); }
	if (tmpf2_fd > -1) { close (tmpf2_fd); }
	if (flock_fd > -1) { close (flock_fd); }

	cc_oci_rm_rf (tmpf1);
	cc_oci_rm_rf (tmpf2);
	cc_oci_rm_rf (tmpf3);

} END_TEST

START_TEST(test_socket_connection_from_fd) {
	int sockets[2] = { -1, -1 };
	GSocketConnection *conn = NULL;
	ck_assert (! cc_oci_socket_connection_from_fd (-1));

	ck_assert (socketpair(PF_UNIX, SOCK_STREAM, 0, sockets) == 0);
	close (sockets[0]);

	conn = cc_oci_socket_connection_from_fd (sockets[1]);
	ck_assert (conn);
	g_object_unref (conn);
	close(sockets[1]);
} END_TEST

START_TEST(test_cc_oci_setup_child) {
	struct cc_oci_config config = { { 0 } };
	ck_assert (! cc_oci_setup_child (NULL));
	ck_assert (cc_oci_setup_child (&config));

	config.detached_mode = true;
	ck_assert (cc_oci_setup_child (&config));
} END_TEST

Suite* make_process_suite(void) {
	Suite* s = suite_create(__FILE__);

	ADD_TEST(test_cc_run_hook, s);
	ADD_TEST(test_cc_oci_setup_shim, s);
	ADD_TEST(test_socket_connection_from_fd, s);
	ADD_TEST(test_cc_oci_setup_child, s);

	return s;
}

int main(void) {
	int number_failed;
	Suite* s;
	SRunner* sr;
	struct cc_log_options options = { 0 };

	options.enable_debug = true;
	options.use_json = false;
	options.filename = g_strdup ("process_test_debug.log");
	(void)cc_oci_log_init(&options);

	s = make_process_suite();
	sr = srunner_create(s);

	srunner_run_all(sr, CK_VERBOSE);
	number_failed = srunner_ntests_failed(sr);
	srunner_free(sr);

	cc_oci_log_free (&options);

	return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}
