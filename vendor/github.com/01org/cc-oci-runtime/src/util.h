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

#ifndef _CC_OCI_UTIL_H
#define _CC_OCI_UTIL_H

#include <stdio.h>

#include <glib.h>
#include <json-glib/json-glib.h>
#include <json-glib/json-gobject.h>

#include "config.h"

/** Calculate size of array specified by \a x. */
#define CC_OCI_ARRAY_SIZE(x) (sizeof(x) / sizeof((x)[0]))

#define CC_OCI_MAX(a, b) ((a) > (b) ? (a) : (b))

/** Call \c g_free() if \a ptr is set */
#define g_free_if_set(ptr) \
	if ((ptr)) { g_free ((ptr)); ptr=NULL; }

#define g_free_node(node) \
	if ((node)) { \
		g_node_traverse(node, G_POST_ORDER, G_TRAVERSE_ALL, \
			-1, (GNodeTraverseFunc)gnode_free, NULL); \
		g_node_destroy(node); \
		node=NULL; }

#ifdef DEBUG
	void cc_oci_node_dump(GNode* node);
#else
	#define cc_oci_node_dump(x)
#endif /* DEBUG */

/**
 * Find the JSON object with the object name specified.
 *
 * \param reader JsonReader.
 * \param file filename \p reader is associated with.
 * \param object string name of object to search for.
 *
 * \return \c true on success, else \c false.
 */
#define cc_oci_get_object(reader, file, object) \
__extension__ ({ \
	int _ret; \
	_ret = json_reader_read_member (reader, object); \
	if (! _ret) { \
		g_critical ("object missing in file %s: %s", \
		file, object); \
	}; \
	_ret; \
})

gchar *cc_oci_get_iso8601_timestamp (void);
gboolean cc_oci_setup_console (const char *console);
gboolean cc_oci_create_pidfile (const gchar *pidfile, GPid pid);
gboolean cc_oci_rm_rf (const gchar *path);
gchar *cc_oci_json_obj_to_string (JsonObject *obj, gboolean pretty,
		gsize *string_len);
gchar *cc_oci_json_arr_to_string (JsonArray *array, gboolean pretty);
gboolean cc_oci_replace_string (gchar **str, const char *from,
		const char *to);
gboolean cc_oci_file_to_strv (const char *file, gchar ***strv);
char** node_to_strv(GNode* root);
gboolean gnode_free(GNode* node, gpointer data);
int cc_oci_get_signum (const gchar *signame);
gchar *cc_oci_resolve_path (const gchar *path);
gboolean cc_oci_fd_toggle_cloexec (int fd, gboolean set);
gboolean cc_oci_enable_networking (void);
guint32 cc_oci_get_big_endian_32(const guint8 *buf);
gboolean cc_oci_handle_signals (void);
gboolean dup_over_stdio(int *fdp);

#endif /* _CC_OCI_UTIL_H */
