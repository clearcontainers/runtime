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

/** \file
 *
 * Qemu QMP routines, used to talk to a running hypervisor.
 *
 * QMP messages are single-line UTF-8-encoded JSON documents.
 * Each message is separated by \ref CC_OCI_MSG_SEPARATOR.
 *
 * See: http://wiki.qemu.org/QMP
 */

#include <string.h>
#include <stdbool.h>
#include <sys/types.h>
#include <signal.h>

#include <glib.h>
#include <glib/gprintf.h>
#include <gio/gio.h>
#include <gio/gunixsocketaddress.h>

#include "oci.h"
#include "util.h"
#include "network.h"
#include "common.h"

/** Size of buffer to use to receive network data */
#define CC_OCI_NET_BUF_SIZE 2048

/** String that separates messages returned from the hypervisor */
#define CC_OCI_MSG_SEPARATOR "\r\n"

/*! VM connection object. */
struct cc_oci_vm_conn
{
	/*! Full path to named socket. */
	gchar socket_path[PATH_MAX];

	/*! Socket address associated with \ref socket_path. */
	GSocketAddress *socket_addr;

	/*! The socket. */
	GSocket *socket;
};

/*!
 * Free the specified \c GString.
 *
 * \param msg \c GString.
 */
static void
cc_oci_net_msg_free (GString *msg)
{
	g_assert (msg);

	g_string_free (msg, true);
}

/*!
 * Free all QMP messages.
 *
 * \param msgs List of \c GString's.
 */
static void
cc_oci_net_msgs_free_all (GSList *msgs)
{
	if (! msgs) {
		return;
	}

	g_slist_free_full (msgs, (GDestroyNotify)cc_oci_net_msg_free);
}

/*!
 * Read a QMP message.
 *
 * \param socket \c GSocket to use.
 * \param expected_count Number of messages to try to receive.
 * \param[out] msgs List of received messages (which are of
 *   type \c GString).
 * \param[out] count Number of messages saved in \p msgs.
 * \return \c true on success, else \c false.
 */
static gboolean
cc_oci_qmp_msg_recv (GSocket *socket,
		gsize expected_count,
		GSList **msgs,
		gsize *count)
{
	gboolean     ret = false;
	gchar        buffer[CC_OCI_NET_BUF_SIZE];
	gsize        total = 0;
	gssize       bytes;
	gssize       msg_len;
	GError      *error = NULL;
	GString     *msg = NULL;
	GString     *received = NULL;
	gchar       *p;

	g_assert (socket);
	g_assert (expected_count);
	g_assert (msgs);
	g_assert (count);

	g_debug ("client expects %lu message%s",
			expected_count,
			expected_count == 1 ? "" : "s");

	do {
		/* reset */
		memset (buffer, '\0', CC_OCI_NET_BUF_SIZE);

		/* read a chunk */
		bytes = g_socket_receive (socket, buffer, CC_OCI_NET_BUF_SIZE,
				NULL, &error);

		if (bytes <= 0) {
			g_critical ("client failed to receive: %s",
					error->message);
			g_error_free (error);
			goto out;
		}

		/* save the chunk */
		if (! received) {
			received = g_string_new_len (buffer, bytes);
		} else {
			g_string_append_len (received, buffer, bytes);
		}

		total += (gsize)bytes;

		while (true) {
			if (! received->len) {
				/* All the data read so far has been consumed */
				break;
			}

			/* Check for end of message marker to determine if a
			 * complete message has been received yet.
			 */
			p = g_strstr_len (received->str, (gssize)total,
					CC_OCI_MSG_SEPARATOR);
			if (! p) {
				/* No complete message to operate on */
				break;
			}

			/* We now have (*atleast* !!) one complete message to handle */

			/* Calculate the length of the message */
			msg_len = p - received->str;

			/* Save the message */
			msg = g_string_new_len (received->str, msg_len);

			/* Add the the complete message to the list */
			*msgs = g_slist_append (*msgs, msg);
			(*count)++;

			g_debug ("client read message %lu '%s' (len=%lu)",
					(unsigned long int)*count,
					msg->str,
					msg->len);

			/* Remove the handled data
			 * (including the message separator).
			 */
			g_string_erase (received, 0,
					(gssize)
					((gsize)msg_len + sizeof (CC_OCI_MSG_SEPARATOR)-1));

			/* Get ready to potentially receive a new
			 * message (caller is responsible for freeing the list).
			 */
			msg = NULL;
		}

		if (*count == expected_count) {
			g_debug ("found expected number of messages (%lu)",
					(unsigned long int)expected_count);
			break;
		}
	} while (true);

	/* All complete messages should have been added to the list */
	g_assert (! msg);

	/* All received data should have been handled */
	g_assert (! received->len);

	/* clean up */
	g_string_free (received, true);

	g_debug ("client received %lu message%s "
			"(expected %lu) in %lu bytes",
			(unsigned long int)*count,
			*count == 1 ? "" : "s",
			(unsigned long int)expected_count,
			(unsigned long int)total);

	ret = *count == expected_count;

out:
	return ret;
}

/*!
 * Check a QMP "execute" response message.
 *
 * \param result Response from server.
 * \param bytes Size of \p result.
 * \param expect_empty \c true if the result is expected to be an
 *   empty json message, else \c false.
 *
 * \warning FIXME: no validation performed on non-empty QMP messages yet.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_oci_qmp_check_result (const char *result, gsize bytes,
		gboolean expect_empty)
{
	gboolean      ret;
	JsonParser   *parser = NULL;
	JsonReader   *reader = NULL;
	GError       *error = NULL;
	gint          count = 0;

	g_assert (result);

	parser = json_parser_new ();
	reader = json_reader_new (NULL);

	ret = json_parser_load_from_data (parser, result,
			(gssize)bytes, &error);
	if (! ret) {
		g_critical ("failed to check qmp response: %s",
				error->message);
		g_error_free (error);

		goto out;
	}

	json_reader_set_root (reader, json_parser_get_root (parser));

	ret = json_reader_read_member (reader, "return");
	if (! ret) {
		goto out;
	}

	ret = json_reader_is_object (reader);
	if (! ret) {
		goto out;
	}

	count = json_reader_count_members (reader);
	if (count && expect_empty) {
		g_critical ("expected empty object denoting success, "
				"but found %d members", count);
		goto out;
	}

	ret = true;

out:
	if (reader) {
		json_reader_end_member (reader);
		g_object_unref (reader);
	}

	if (parser) {
		g_object_unref (parser);
	}

	return ret;
}

/*!
 * Send a QMP message to the hypervisor.
 *
 * \param conn \ref cc_oci_vm_conn to use.
 * \param msg Data to send (json format).
 * \param msg_len message length.
 * \param expected_resp_count Expected number of response messages.
 * \param expect_empty \c true if the response message is expected
 *   to be an empty json message, else \c false.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_oci_qmp_msg_send (struct cc_oci_vm_conn *conn,
		const char *msg,
		gsize msg_len,
		gsize expected_resp_count,
		gboolean expect_empty)
{
	private gboolean   initialised = false;
	const  gchar      capabilities[] = "{ \"execute\": \"qmp_capabilities\" }";
	GError           *error = NULL;
	gssize            size;
	gboolean          ret = false;
	GSList           *msgs = NULL;
	GString          *recv_msg = NULL;
	gsize             msg_count = 0;

	g_assert (conn);
	g_assert (msg);

	if (! initialised) {
		/* The QMP protocol requires we query its capabilities
		 * before sending any further messages.
		 */

		g_debug ("sending required initial capabilities "
				"message (%s)", capabilities);

		size = g_socket_send (conn->socket, capabilities,
				sizeof (capabilities)-1, NULL, &error);
		if (size < 0) {
			g_critical ("failed to send json: %s", capabilities);
			if (error) {
				g_critical ("error: %s", error->message);
				g_error_free (error);
			}
			goto out;
		}

		/* Get the response */
		ret = cc_oci_qmp_msg_recv (conn->socket,
				1, &msgs, &msg_count);
		if (! ret) {
			goto out;
		}

		recv_msg = g_slist_nth_data (msgs, 0);
		if (! recv_msg) {
			ret = false;
			goto out;
		}

		/* Check it */
		ret = cc_oci_qmp_check_result (recv_msg->str,
				recv_msg->len, true);
		if (! ret) {
			goto out;
		}

		cc_oci_net_msgs_free_all (msgs);

		initialised = true;

		/* reset */
		recv_msg = NULL;
		msgs = NULL;
		msg_count = 0;
	}

	g_debug ("sending message '%s'", msg);

	size = g_socket_send (conn->socket, msg, msg_len,
			NULL, &error);
	if (size < 0) {
		g_critical ("failed to send json: %s", msg);
		if (error) {
			g_critical ("error: %s", error->message);
			g_error_free (error);
		}
		goto out;
	}

	if (! expected_resp_count) {
		ret = true;
		goto out;
	}

	/* Get the response */
	ret = cc_oci_qmp_msg_recv (conn->socket,
			expected_resp_count,
			&msgs, &msg_count);
	if (! ret) {
		goto out;
	}

	if (expected_resp_count == 1) {
		/* Expected response:
		 *
		 * - Message 1: return object.
		 */
		recv_msg = g_slist_nth_data (msgs, 0);
	} else if (expected_resp_count == 2) {
		/* Expected response:
		 *
		 * - Message 1: event object (FIXME: not currently checked).
		 * - Message 2: return object.
		 */
		recv_msg = g_slist_nth_data (msgs, 1);
	} else if (expected_resp_count == 3) {
		/* Expected response:
		 *
		 * - Message 1: return object.
		 *
		 * - Message 2: timestamp (POWERDOWN event).
		 *
		 * - Message 3: timestamp (SHUTDOWN event).
		 */
		recv_msg = g_slist_nth_data (msgs, 0);
	} else {
		g_critical ("BUG: don't know how to handle message with %lu responses", (unsigned long int)expected_resp_count);
		ret = false;
		goto out;
	}

	if (! recv_msg) {
		ret = false;
		goto out;
	}

	/* Check message */
	ret = cc_oci_qmp_check_result (recv_msg->str,
			recv_msg->len, expect_empty);
	if (! ret) {
		goto out;
	}

	ret = true;

out:
	if (msgs) {
		cc_oci_net_msgs_free_all (msgs);
	}

	return ret;
}

/*!
 * Send a QMP pause message to the hypervisor.
 *
 * \param conn \ref cc_oci_vm_conn to use.
 * \param pid \c GPid of hypervisor process.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_oci_qmp_pause (struct cc_oci_vm_conn *conn, GPid pid)
{
	const char pause_msg[] = "{ \"execute\": \"stop\" }";
	g_assert (conn);
	g_assert (pid);

	return cc_oci_qmp_msg_send (conn, pause_msg,
			sizeof(pause_msg)-1, 2, false);
}

/*!
 * Send a QMP resume message to the hypervisor.
 *
 * \param conn \ref cc_oci_vm_conn to use.
 * \param pid \c GPid of hypervisor process.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_oci_qmp_resume (struct cc_oci_vm_conn *conn, GPid pid)
{
	const char resume_msg[] = "{ \"execute\": \"cont\" }";
	g_assert (conn);
	g_assert (pid);

	return cc_oci_qmp_msg_send (conn, resume_msg,
			sizeof(resume_msg)-1, 2, false);
}

/*!
 * Read the expected QMP welcome message.
 *
 * \param socket \c GSocket to use.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_oci_qmp_check_welcome (GSocket *socket)
{
	GError      *error = NULL;
	JsonParser  *parser = NULL;
	JsonReader  *reader = NULL;
	GSList      *msgs = NULL;
	gsize        msg_count = 0;
	gboolean     ret;
	GString     *msg = NULL;

	g_assert (socket);

	ret = cc_oci_qmp_msg_recv (socket, 1, &msgs, &msg_count);
	if (! ret) {
		goto out;
	}

	msg = g_slist_nth_data (msgs, 0);
	g_assert (msg);

	parser = json_parser_new ();
	reader = json_reader_new (NULL);

	ret = json_parser_load_from_data (parser, msg->str, (gssize)msg->len, &error);
	if (! ret) {
		g_critical ("failed to parse json: %s", error->message);
		g_error_free (error);
		goto out;
	}

	json_reader_set_root (reader, json_parser_get_root (parser));

	/* FIXME: perform more checks on the data received */
	ret = json_reader_read_member (reader, "QMP");
	if (! ret) {
		g_critical ("unexpected json data");
		json_reader_end_member (reader);
		goto out;
	}

	json_reader_end_member (reader);

out:
	if (reader) {
		g_object_unref (reader);
	}
	if (parser) {
		g_object_unref (parser);
	}
	if (msgs) {
		cc_oci_net_msgs_free_all (msgs);
	}

	g_debug ("handled qmp welcome");

	return true;
}

static void
cc_oci_vm_conn_free (struct cc_oci_vm_conn *conn)
{
	g_assert (conn);

	g_object_unref (conn->socket_addr);
	g_object_unref (conn->socket);
	g_free (conn);
}

/*!
 * Create a new \ref cc_oci_vm_conn and connect to hypervisor to
 * perform initial welcome negotiation.
 *
 * \param socket_path Full path to named socket.
 * \param pid Process ID of running hypervisor.
 *
 * \return \ref cc_oci_vm_conn on success, else \c NULL.
 */
static struct cc_oci_vm_conn *
cc_oci_vm_conn_new (const gchar *socket_path, GPid pid)
{
	struct cc_oci_vm_conn  *conn = NULL;
	GError                  *error = NULL;
	gboolean                 ret = false;

	if (! (socket_path && pid > 0)) {
		return NULL;
	}

	if (! g_file_test (socket_path, G_FILE_TEST_EXISTS)) {
		g_critical ("socket path does not exist: %s", socket_path);
		goto err;
	}

	conn = g_new0 (struct cc_oci_vm_conn, 1);
	if (! conn) {
		return NULL;
	}

	g_strlcpy (conn->socket_path, socket_path,
			sizeof (conn->socket_path));

	conn->socket_addr = g_unix_socket_address_new (socket_path);
	if (! conn->socket_addr) {
		g_critical ("failed to create a new socket addres: %s", socket_path);
		goto err;
	}

	conn->socket = g_socket_new (G_SOCKET_FAMILY_UNIX,
			G_SOCKET_TYPE_STREAM,
			G_SOCKET_PROTOCOL_DEFAULT, &error);

	if (! conn->socket) {
		g_critical ("failed to create socket: %s",
				error->message);
		g_error_free (error);
		goto err;
	}

	ret = g_socket_connect (conn->socket, conn->socket_addr,
			NULL, &error);
	if (! ret) {
		g_critical ("failed to connect to hypervisor control socket %s: %s",
				socket_path,
				error->message);
		g_error_free (error);
		goto err;
	}

	g_debug ("connected to socket path %s", socket_path);

	ret = cc_oci_qmp_check_welcome (conn->socket);
	if (! ret) {
		goto err;
	}

	return conn;

err:
	if (conn) {
		cc_oci_vm_conn_free (conn);
	}

	return NULL;
}

/*!
 * Request the running hypervisor pause.
 *
 * \param socket_path Path to \ref CC_OCI_HYPERVISOR_SOCKET.
 * \param pid \c GPid of hypervisor process.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_vm_pause (const gchar *socket_path, GPid pid)
{
	gboolean                 ret = false;
	struct cc_oci_vm_conn  *conn = NULL;

	if (! (socket_path != NULL && pid > 0)) {
		return false;
	}

	conn = cc_oci_vm_conn_new (socket_path, pid);
	if (! conn) {
		goto out;
	}

	ret = cc_oci_qmp_pause (conn, pid);
	if (! ret) {
		goto out;
	}

out:
	if (conn) {
		cc_oci_vm_conn_free (conn);
	}

	return ret;
}

/*!
 * Request the running hypervisor unpause.
 *
 * \param socket_path Path to \ref CC_OCI_HYPERVISOR_SOCKET.
 * \param pid \c GPid of hypervisor process.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_vm_resume (const gchar *socket_path, GPid pid)
{
	gboolean                 ret = false;
	struct cc_oci_vm_conn  *conn = NULL;

	if (! (socket_path != NULL && pid > 0)) {
		return false;
	}

	conn = cc_oci_vm_conn_new (socket_path, pid);
	if (! conn) {
		goto out;
	}

	ret = cc_oci_qmp_resume (conn, pid);
	if (! ret) {
		goto out;
	}

out:
	if (conn) {
		cc_oci_vm_conn_free (conn);
	}

	return ret;
}
