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

#include <string.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>
#include <errno.h>
#include <stdbool.h>

#include <glib.h>
#include <glib/gprintf.h>

#include "util.h"
#include "config.h"

/** Full path to \c rm(1) command. */
#define CC_OCI_RM_CMD "/bin/rm"

#define make_table_entry(value) \
{ value, #value }

static struct cc_oci_signal_table
{
	int          num;
	const gchar *name;
} signal_table[] = {
	make_table_entry (SIGHUP),
	make_table_entry (SIGINT),
	make_table_entry (SIGQUIT),
	make_table_entry (SIGILL),
	make_table_entry (SIGTRAP),
	make_table_entry (SIGABRT),
	make_table_entry (SIGIOT),
	make_table_entry (SIGBUS),
	make_table_entry (SIGFPE),
	make_table_entry (SIGKILL),
	make_table_entry (SIGUSR1),
	make_table_entry (SIGSEGV),
	make_table_entry (SIGUSR2),
	make_table_entry (SIGPIPE),
	make_table_entry (SIGALRM),
	make_table_entry (SIGTERM),
	make_table_entry (SIGSTKFLT),
	make_table_entry (SIGCLD),
	make_table_entry (SIGCHLD),
	make_table_entry (SIGCONT),
	make_table_entry (SIGSTOP),
	make_table_entry (SIGTSTP),
	make_table_entry (SIGTTIN),
	make_table_entry (SIGTTOU),
	make_table_entry (SIGURG),
	make_table_entry (SIGXCPU),
	make_table_entry (SIGXFSZ),
	make_table_entry (SIGVTALRM),
	make_table_entry (SIGPROF),
	make_table_entry (SIGWINCH),
	make_table_entry (SIGPOLL),
	make_table_entry (SIGIO),
	make_table_entry (SIGPWR),
	make_table_entry (SIGSYS),
	make_table_entry (SIGUNUSED),

	/* terminator */
	{ -1, NULL }
};

/*!
 *
 * \param signame Name of signal.
 *
 * \note \p signame may by a full name (such as "SIGINT"),
 * or a partial name (such as "INT").
 *
 * \return signal number, or -1 on error.
 */
int
cc_oci_get_signum (const gchar *signame)
{
	struct cc_oci_signal_table  *s;
	gchar full_name[32] = { 0 };

	if (! signame) {
		return -1;
	}

	if (! g_str_has_prefix (signame, "SIG")) {
		g_strlcpy (full_name, "SIG", sizeof (full_name));
	}

	g_strlcat (full_name, signame, sizeof (full_name));

	for (s = signal_table; s && s->name; s++) {
		if (! g_strcmp0 (full_name, s->name)) {
			return s->num;
		}
	}

	return -1;
}

/*!
 * Create an ISO-8601-formatted timestamp.
 *
 * \return Newly-allocated string.
 */
gchar *
cc_oci_get_iso8601_timestamp (void)
{
	GTimeVal   tv;
	GDateTime *dt = NULL;
	gchar     *timestamp = NULL;

	dt = g_date_time_new_now_local ();
	if (! dt) {
		return NULL;
	}

	if (g_date_time_is_daylight_savings (dt)) {
		GDateTime *dt2 = NULL;

		dt2 = g_date_time_add_hours (dt, 1);
		if (! dt2) {
			goto out;
		}

		g_date_time_unref (dt);
		dt = dt2;
	}

	if (! g_date_time_to_timeval (dt, &tv)) {
		goto out;
	}

	timestamp = g_time_val_to_iso8601 (&tv);

out:
	if (dt) {
		g_date_time_unref (dt);
	}

	return timestamp;
}

/*!
 * Create a pidfile.
 *
 * \param pidfile Full path to pidfile.
 * \param pid Process ID.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_create_pidfile (const gchar *pidfile, GPid pid)
{
	gboolean  ret = false;
	GError   *err = NULL;
	GString  *str = NULL;

	if (! pidfile || *pidfile != '/') {
		return false;
	}

	if (pid <= 0) {
		return false;
	}

	str = g_string_new("");
	if (! str) {
		return ret;
	}

	/* XXX: must NOT add a newline! */
	g_string_printf (str, "%d", pid);

	ret = g_file_set_contents (pidfile, str->str, (gssize)str->len, &err);
	if (! ret) {
		g_critical ("failed to create pidfile for pid %d: err=%s\n",
				pid, err->message);
		g_error_free (err);
		goto out;
	}

	ret = true;

	g_debug ("created pidfile %s for pid %d", pidfile, (int)pid);

out:
	if (str) {
		g_string_free(str, true);
	}

	return ret;
}

/*!
 * Recursively delete a directory.
 *
 * \param path Full path to directory to delete.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_rm_rf (const gchar *path)
{
	gchar *cmd;
	gboolean ret = false;

	if (! path || ! *path) {
		return false;
	}

	cmd = g_strdup_printf ("%s -rf \"%s\" >/dev/null 2>&1",
			CC_OCI_RM_CMD, path);
	if (! cmd) {
		return false;
	}

	if (system (cmd)) {
		g_critical ("failed to remove directory %s", path);
	} else {
		ret = true;
	}

	g_free (cmd);

	return ret;
}

/*!
 * Convert a JSON object to a string.
 *
 * \param obj \c JsonObject.
 * \param pretty If \c true, pretty-format the string.
 * \param[out] string_len string length.
 *
 * \return Newly-allocated string on success, else \c NULL.
 */
gchar *
cc_oci_json_obj_to_string (JsonObject *obj, gboolean pretty, gsize *string_len)
{
	JsonGenerator  *generator;
	JsonNode       *root;
	gchar          *str;

	if (! obj) {
		return NULL;
	}

	root = json_node_alloc ();

	/* refs obj to allow node to be freed */
	json_node_init_object (root, obj);

	generator = json_generator_new ();
	json_generator_set_root (generator, root);

	g_object_set (generator, "pretty", pretty, NULL);

	str = json_generator_to_data (generator, string_len);

	json_node_free (root);
	g_object_unref (generator);

	return str;
}

/*!
 * Convert a JSON array to a string.
 *
 * \param array \c JsonArray.
 * \param pretty If \c true, pretty-format the string.
 *
 * \return Newly-allocated string on success, else \c NULL.
 */
gchar *
cc_oci_json_arr_to_string (JsonArray *array, gboolean pretty)
{
	JsonGenerator  *generator;
	JsonNode       *root;
	gchar          *str;
	gsize           len;

	if (! array) {
		return NULL;
	}

	root = json_node_alloc ();

	/* refs array to allow node to be freed */
	json_node_init_array (root, array);

	generator = json_generator_new ();
	json_generator_set_root (generator, root);

	g_object_set (generator, "pretty", pretty, NULL);

	str = json_generator_to_data (generator, &len);

	json_node_free (root);
	g_object_unref (generator);

	return str;
}

/*!
 * Replace a single occurences of \p from with \p to in the
 * string specified by \p str.
 *
 * If \p to is specifies as \c "", this will cause \p from to be
 * deleted from \p str.
 *
 * \param str String to modify.
 * \param from Search string.
 * \param to Replacement value.
 *
 * \note Limitation - only a single occurence is currently handled.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_replace_string (gchar **str, const char *from, const char *to)
{
	gchar  *p;
	gchar  *new = NULL;
	guint   pre;
	gchar  *post = NULL;

	g_assert (str);

	/* Although both from and to must be specified, to can be ""
	 * meaning "delete the occurence of from".
	 */
	if (! from || ! *from || ! to) {
		return true;
	}

	p = g_strstr_len (*str, -1, from);
	if (! p) {
		/* No match */
		return true;
	}

	pre = (guint)(p - *str);
	post = p + g_utf8_strlen (from, LINE_MAX);

	new = g_strdup_printf ("%.*s%s%s",
			pre ? pre : 0,
			*str,
			to,
			*post ? post : "");

	g_free (*str);
	*str = new;

	return true;
}

/*!
 * Read the specified file and break it (on newline)
 * into a newly-allocated array.
 *
 * \param file Absolute path to file to read.
 * \param[out] strv Newly-allocated string array representing the file.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_file_to_strv (const char *file, gchar ***strv)
{
	gchar     *contents = NULL;
	GError    *err = NULL;
	gboolean   ret = false;
	gsize      bytes;
	guint      len;

	if (! file || ! strv) {
		return false;
	}

	if( *file != '/' ) {
		g_critical ("No absolute path : %s", file);
		return false;
	}

	ret = g_file_get_contents (file, &contents, &bytes, &err);
	if (! ret) {
		g_critical ("failed to read file %s: %s",
				file, err->message);
		g_error_free (err);
		return ret;
	}

	*strv = g_strsplit (contents, "\n", -1);
	if (! *strv) {
		ret = false;
		goto out;
	}

	len = g_strv_length (*strv);
	if (! len) {
		g_strfreev (*strv);

		ret = false;
		goto out;
	}

	if (! *((*strv)[len-1])) {
		/* annoyingly, splitting the file results in a final
		 * empty element (containing an allocated '\0').
		 * This must not be passed to the hypervisor, or weird
		 * errors will result.
		 */
		g_free ((*strv)[len-1]);
		(*strv)[len-1] = NULL;
	}

	ret = true;

out:
	g_free (contents);

	return ret;
}

char**
node_to_strv(GNode* root) {
	char** strv = NULL;
	gsize size = 0;
	int i = 0;
	GNode* child;

	size = g_node_n_children(root)+1;
	strv = g_malloc0_n(size, sizeof(gchar *));

	for (child = g_node_first_child(root); child && i<size;
			child=g_node_next_sibling(child), ++i) {
		strv[i] = g_strdup(child->data);
	}

	return strv;
}

gboolean
gnode_free(GNode* node, gpointer data) {
	if (node->data) {
		g_free(node->data);
	}
	return false;
}

/**
 * Resolve a path by converting to canonical form:
 *
 * - expands symbolic links.
 * - converts from relative to absolute.
 *
 * \param path Path to expand.
 *
 * \return Newly-allocated path string on success, else \c NULL.
 */
gchar *
cc_oci_resolve_path (const gchar *path)
{
	char tmp[PATH_MAX] = { 0 };

	memset (tmp, '\0', sizeof (tmp));

	if (! path || ! *path) {
		return NULL;
	}

	/* this will of course fail if path doesn't exist.
	 *
	 * XXX: note that we don't take advantage of realpath(3)'s
	 * ability to allocate the resolved path because that requires
	 * the user free the memory with free(3). Let's keep it glib for
	 * now.
	 */
	if (! realpath (path, tmp)) {
		int saved = errno;
		g_print ("realpath '%s' failed: %s\n",
				path, strerror (saved));
		return NULL;
	}

	g_debug ("path '%s' resolved to '%s'", path, tmp);

	return g_strdup (tmp);
}

/**
 * Set the close-exec bit on the specified file descriptor.
 *
 * \param fd File descriptor to change.
 *
 * \param set Flag to enable or disable
 *
 * \return \c true on success, else \c false.
 **/
gboolean
cc_oci_fd_toggle_cloexec (int fd, gboolean set)
{
	int flags;

	if (fd < 0) {
		return false;
	}

	flags = fcntl (fd, F_GETFD);
	if (flags < 0) {
		return false;
	}

	if (set) {
		flags |= FD_CLOEXEC;
	}else {
		flags &= ~FD_CLOEXEC;
	}

	if (fcntl (fd, F_SETFD, flags) < 0) {
		return false;
	}

	return true;
}

/**
 * Determine if networking setup should occur.
 *
 * \return \c true if networking should be enabled, else \c false.
 */
gboolean
cc_oci_enable_networking (void)
{
	/* networking can only be setup when running as root
	 * (since it requires creating network interfaces).
	 */
	gboolean enable = ! geteuid ();

	if (! enable) {
		g_debug ("networking will not be enabled "
				"(insufficient privileges)");
	}

	return enable;
}

/*!
 * Convert the value stored in buffer to little endian
 *
 * \param buf Buffer storing the big endian value
 *
 * \return \c guint32 in big endian order.
 */
guint32 cc_oci_get_big_endian_32(const guint8 *buf) {
	return (guint32)(buf[0] << 24 | buf[1] << 16 | buf[2] << 8 | buf[3]);
}

/**
 * Perform global signal handling setup.
 */
gboolean
cc_oci_handle_signals (void)
{
	struct sigaction old_act = { {0} };
	struct sigaction new_act = { {0} };

	new_act.sa_handler = SIG_IGN;
	sigemptyset (&new_act.sa_mask);
	new_act.sa_flags = 0;

	/* Ignore SIGPIPE to avoid being terminated unceremoniously
	 * when a pipe/socket write(2) fails.
	 *
	 * This could happen if the other end of the pipe dies,
	 * but also if it is no longer running (for example a very
	 * short-running hook that doesn't expect to be passed state
	 * on its stdin).
	 */
	if (sigaction (SIGPIPE, &new_act, &old_act) < 0) {
		g_critical ("failed to ignore SIGPIPE: %s",
				strerror (errno));
		return false;
	}

	return true;
}

/**
 * dup a fd until its value is higher than stdio standard fds
 * (higher than 2), on success the original fd value is closed and
 * can not be used anymore.
 *
 * \param[in,out] fd File descriptor to change.
 *
 * \return \c true on success, else \c false.
 **/
gboolean
dup_over_stdio(int *fdp){
	int tmp_fds[3] = {-1, -1, -1};
	/* fd to dup */
	int dup_fd  = -1;
	gboolean ret = false;

	if ( ! fdp  || *fdp < 0 ) {
		return false;
	}

	if (fcntl(*fdp, F_GETFD) == -1 ){
		return false;
	}

	if ( *fdp > 2 ) {
		/* fd is already higher than 3 */
		return true;
	}
	dup_fd = *fdp;

	/* Dup until dup_fd is higher than 3 */
	for (int i = 0 ; i < CC_OCI_ARRAY_SIZE(tmp_fds) && dup_fd < 3 ; i++) {
		/* Save old fd to close it later */
		/* if we close it now will get the same fd next iteration */
		tmp_fds[i] = dup_fd;

		dup_fd = dup(tmp_fds[i]);
		if (dup_fd < 0) {
			g_critical("dup failed: %s", strerror(errno));
			break;
		}
	}

	if ( dup_fd < 3){
		/* error, we could not get a fd higher than 3
	 	 * lets close last dup fd, only if the fd is
	 	 * not the received as parameter.
	 	 * */
		if ( dup_fd != *fdp && dup_fd > -1){
			close(dup_fd);
			dup_fd = -1;
		}
	} else  {
		/* success */
		g_debug("fd moved from %d to  %d", *fdp, dup_fd);
		*fdp = dup_fd;
		dup_fd = -1;
		ret = true;
	}

	for (int i = 0 ; i < CC_OCI_ARRAY_SIZE(tmp_fds) ; i++) {
		/* Dont close the original fd on error */
		if( !ret && tmp_fds[i] == *fdp)
		{
			g_debug("failed to dup %d, not closing it", *fdp);
			continue;
		}
		if ( tmp_fds[i] > -1 ) {
			g_debug("closing tmp fd %d", tmp_fds[i]);
			if (close ( tmp_fds[i] )  == -1 ) {
				g_critical("failed to close tmp fd: %s", strerror(errno));
				ret = false;
			}
		}
	}
	return ret;
}

#ifdef DEBUG
static gboolean
cc_oci_node_dump_aux(GNode* node, gpointer data) {
	gchar indent[LINE_MAX] = { 0 };
	guint i;
	for (i = 0; i < g_node_depth(node); i++) {
		g_strlcat(indent, "    ", LINE_MAX);
	}
	g_message("%s[%d]:%s", indent, g_node_depth(node), node ? (char*)node->data : "(null)");
	return false;
}

void
cc_oci_node_dump(GNode* node) {
	if (!node) {
		return;
	}
	g_message("debug: " "======== Dumping GNode: ========");
	g_node_traverse(node, G_PRE_ORDER, G_TRAVERSE_ALL, -1, cc_oci_node_dump_aux, NULL);
}
#endif /*DEBUG*/
