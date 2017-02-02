/*
 * This file is part of cc-oci-runtime.
 *
 * Copyrighth (C) 2016 Intel Corporation
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
#include <errno.h>
#include <sys/stat.h>
#include <arpa/inet.h>
#include <gio/gunixsocketaddress.h>
#include "oci.h"
#include "json.h"
#include "common.h"
#include "pod.h"
#include "proxy.h"
#include "util.h"
#include "networking.h"
#include "command.h"

extern struct start_data start_data;

struct watcher_proxy_data
{
	GMainLoop   *loop;
	GIOChannel  *channel;
	gchar       *msg_to_send;
	GString     *msg_received;
	int          socket_fd;
	/**
	 * Indicates that we expect an out-of-band file descriptor
	 * from proxy socket.
	 */
	int         *oob_fd;
};

/** Format of a proxy message */
struct proxy_msg {
	/** Number of bytes in payload. */
	guint32  length;

	/** Not used. */
	guint32  reserved;

	/** Message payload (JSON). */
	gchar   *data;
};

/**
 * Free resources associated with \p proxy.
 *
 * \param proxy \ref cc_proxy.
 *
 */
void
cc_proxy_free (struct cc_proxy *proxy) {
	if (! proxy) {
		return;
	}

	g_free_if_set (proxy->agent_ctl_socket);
	g_free_if_set (proxy->agent_tty_socket);
	g_free_if_set (proxy->vm_console_socket);

	if (proxy->socket) {
		g_object_unref (proxy->socket);
	}

	g_free (proxy);
}

/**
 * Determine if already connected to the proxy.
 *
 * \return \c true if connected, else \c false.
 */
static inline gboolean
cc_proxy_connected (struct cc_proxy *proxy)
{
	return proxy->socket ? true : false;
}

/**
 * Connect to CC_OCI_PROXY.
 *
 * \param proxy \ref cc_proxy.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_connect (struct cc_proxy *proxy)
{
	GSocketAddress *addr;
	GError *error = NULL;
	gboolean ret = false;
	const gchar *path = NULL;
	int fd = -1;
	const gchar *proxy_socket_path = NULL;

	if (! proxy) {
		return false;
	}

	if (cc_proxy_connected (proxy)) {
		g_critical ("already connected to proxy");
		return false;
	}

	proxy_socket_path = start_data.proxy_socket_path;
	if (! proxy_socket_path) {
		proxy_socket_path = CC_OCI_PROXY_SOCKET;
	}

	g_debug ("connecting to proxy %s", CC_OCI_PROXY);

	addr = g_unix_socket_address_new (proxy_socket_path);
	if (! addr) {
		g_critical ("socket path does not exist: %s",
				proxy_socket_path);
		goto out_addr;
	}

	path = g_unix_socket_address_get_path (G_UNIX_SOCKET_ADDRESS (addr));

	proxy->socket = g_socket_new (G_SOCKET_FAMILY_UNIX,
				      G_SOCKET_TYPE_STREAM,
				      G_SOCKET_PROTOCOL_DEFAULT,
				      &error);
	if (! proxy->socket) {
		g_critical ("failed to create socket for %s: %s",
				path,
				error->message);
		g_error_free (error);
		goto out_socket;
	}

	/* block on write and read */
	g_socket_set_blocking (proxy->socket, TRUE);

	fd = g_socket_get_fd (proxy->socket);

	ret = cc_oci_fd_toggle_cloexec(fd, true);
	if (! ret) {
		g_critical ("failed to set close-exec bit on proxy socket");
		goto out;
	}

	ret = g_socket_connect (proxy->socket, addr, NULL, &error);
	if (! ret) {
		g_critical ("failed to connect to proxy socket %s: %s",
				path,
				error->message);
		g_error_free (error);
		goto out_connect;
	}

	g_debug ("connected to proxy socket %s", path);

	ret = true;

out:
	return ret;
out_connect:
	g_clear_object (&proxy->socket);
out_socket:
	g_object_unref (addr);
out_addr:
	return ret;
}

/**
 * Disconnect from CC_OCI_PROXY.
 *
 * \param proxy \ref cc_proxy.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_disconnect (struct cc_proxy *proxy)
{
	GError   *error = NULL;
	gboolean   ret = false;

	if (! proxy) {
		return false;
	}

	if (! cc_proxy_connected (proxy)) {
		g_critical ("not connected to proxy");
		return ret;
	}

	g_debug ("disconnecting from proxy");

	if (! g_socket_close (proxy->socket, &error)) {
		g_critical ("failed to disconnect from proxy: %s",
		    error->message);
		g_error_free(error);
		goto out;
	}

	ret = true;

out:
	g_clear_object (&proxy->socket);

	return ret;
}

/**
 * Read a file descriptor from the proxy's socket.
 *
 * \param proxy_fd the fd of the proxy socket
 * \param fd fd read out of proxy_fd (out parameter)
 *
 * The proxy can send fds through OOB data after a successful reply of certain
 * payloads. proxy will send us a 1 byte dummy message containing 'F'
 * (OOB_FD_FLAG) for signaling OOB data.
 *
 * \return \c true on success, \c false otherwise.
 */
static gboolean
cc_proxy_receive_fd(int proxy_fd, int *fd)
{
	struct msghdr msg = { 0 };
	gchar iov_buffer[1] = { 0 };
	struct iovec io = { .iov_base = iov_buffer,
	                    .iov_len = sizeof(iov_buffer) };
	char ctl_buffer[CMSG_SPACE(sizeof(int))];
	struct cmsghdr *cmsg = NULL;
	ssize_t bytes_read;
	unsigned char *data;

	msg.msg_iov = &io;
	msg.msg_iovlen = 1;
	msg.msg_control = ctl_buffer;
	msg.msg_controllen = sizeof(ctl_buffer);

	while (1) {
		bytes_read = recvmsg(proxy_fd, &msg, 0);
		if (bytes_read < 0 && (errno == EAGAIN || errno == EINTR)) {
			continue;
		}
		break;
	}

	if (bytes_read == -1){
		g_critical("recvmsg failed: %s", strerror(errno));
		return false;
	}

	if (bytes_read == 0) {
		g_critical("recvmsg failed: EOF");
		return false;
	}

	if (bytes_read != 1 || iov_buffer[0] != OOB_FD_FLAG) {
		g_critical("recvmsg failed: read %zd bytes, flag: %c",
			   bytes_read, iov_buffer[0]);
		return false;
	}

	cmsg = CMSG_FIRSTHDR(&msg);
	if (!cmsg) {
		g_critical("could not read the control message");
		return false;
	}

	data = CMSG_DATA(cmsg);
	if (!data) {
		g_critical("missing out of band data");
		return false;
	}

	*fd = *((int*) data);

	g_message("received fd from proxy %d", *fd);

	return true;
}

/**
 * Read a message from proxy's socket.
 *
 * \param source GIOChannel.
 * \param condition GIOCondition.
 * \param proxy_data struct watcher_proxy_data.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_proxy_msg_read (GIOChannel *source, GIOCondition condition,
	struct watcher_proxy_data *proxy_data)
{
	ssize_t bytes_read, total_bytes_read = 0;
	size_t payload_length;
	guint8 header[MESSAGE_HEADER_LENGTH] = { 0 };
	gchar buf[LINE_MAX];

	/*
	 * Start by reading the message header and the payload length.
	 */
	bytes_read = read(proxy_data->socket_fd, header, sizeof(header));
	if (bytes_read <= 0) {
		g_critical("couldn't read header from proxy: %s",
			   strerror(errno));
		goto out;
	}

	payload_length = cc_oci_get_big_endian_32 (header);
	g_debug ("proxy msg length: %ld", payload_length);

	/*
	 * The proxy sends back very small messages usually just a:
	 *    '{"status":"success"}'
	 * The biggest response is from allocateIO that adds an ioBase field,
	 * so 1k is quite a high boundary.
	 */
	if (payload_length > 1024) {
		g_critical("received bogus payload length");
		goto out;
	}

	/*
	 * Read the payload
	 */
	while(total_bytes_read < payload_length) {
		bytes_read = read(proxy_data->socket_fd, buf,
				  payload_length - (size_t)total_bytes_read);
		if (bytes_read <= 0) {
			/* Continue if we're missing some bytes */
			if (errno == EAGAIN || errno == EINTR) {
				continue;
			}
			g_critical("lost proxy connection: %s",
				   strerror(errno));
			break;
		}

		g_string_append_len(proxy_data->msg_received, buf, bytes_read);

		total_bytes_read += bytes_read;
	}

	if (proxy_data->msg_received->len > 0) {
		g_debug("message read from proxy socket: %s",
			proxy_data->msg_received->str);
	}

out:
	g_main_loop_quit (proxy_data->loop);

	/* Unregister this watcher */
	return false;
}

/**
 * Write down a message into proxy's socket.
 *
 * \param source GIOChannel.
 * \param condition GIOCondition.
 * \param proxy_data struct watcher_proxy_data.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_proxy_msg_write (GIOChannel *source, GIOCondition condition,
	struct watcher_proxy_data *proxy_data)
{
	gsize bytes_written = 0;
	gsize len = 0;
	GIOStatus status;
	GError *error = NULL;
	struct proxy_msg msg = { 0 };

	if (condition == G_IO_HUP) {
		g_io_channel_unref(source);
		goto out;
	}

	len = (gsize)g_utf8_strlen(proxy_data->msg_to_send, -1);

	msg.length = htonl ((guint32)len);
	msg.data = proxy_data->msg_to_send;

	g_debug ("sending message (length %lu) to proxy socket",
			(unsigned long int)len);

	do {
		status = g_io_channel_write_chars(source,
				(const gchar *)&msg,
				(gssize)sizeof (msg.length) +
				sizeof (msg.reserved),
				&bytes_written,
				&error);
	} while (status == G_IO_STATUS_AGAIN);

	if (status == G_IO_STATUS_ERROR) {
		g_debug ("proxy message length write failed: %s",
				error->message);
		g_error_free (error);
		goto out;
	}

	g_debug("writing message data to proxy socket: %s",
			proxy_data->msg_to_send);

	do {
		status = g_io_channel_write_chars(source,
				proxy_data->msg_to_send,
				(gssize)len,
				&bytes_written,
				&error);
	} while (status == G_IO_STATUS_AGAIN);

	if (status == G_IO_STATUS_ERROR) {
		g_debug ("proxy write failed: %s", error->message);
		g_error_free (error);
		goto out;
	}

	do {
		status = g_io_channel_flush (source, &error);
	} while (status == G_IO_STATUS_AGAIN);

	if (status == G_IO_STATUS_ERROR) {
		g_debug ("proxy flush failed: %s",
		    error->message);
		g_error_free (error);
		goto out;
	}

	/* Now we've sent the initial negotiation message,
	 * register a handler to wait for a reply.
	 */
	g_io_add_watch(source, G_IO_IN | G_IO_HUP,
	    (GIOFunc)cc_proxy_msg_read, proxy_data);

out:
	/* unregister this watcher */
	return false;
}
/**
 * Callback used to monitor CTL socket creation
 *
 * \param monitor GFileMonitor.
 * \param file GFile.
 * \param other_file GFile.
 * \param event_type GFileMonitorEvent.
 * \param loop GMainLoop.
 */
static void
cc_proxy_ctl_socket_created_callback(GFileMonitor *monitor, GFile *file,
	GFile *other_file, GFileMonitorEvent event_type, GMainLoop *loop)
{
	(void)monitor;
	(void)file;
	(void)other_file;

	g_debug("CTL created event: %d", event_type);
	g_main_loop_quit(loop);
}

/**
 * Determine if the hyper command was run successfully.
 *
 * Accomplished by checking the proxy response message which is
 * of the form:
 *
 *     {"success": [true|false], "error": "an explanation" }
 *
 * \param response \c GString containing raw proxy response message.
 * \param[out] proxy_success \c true if the last proxy command was
 *   successful, else \c false.
 *
 * \return \c true if the proxy response could be checked,
 * else \c false.
 */
static gboolean
cc_proxy_hyper_check_response (const GString *response,
		gboolean *proxy_success)
{
	JsonParser  *parser = NULL;
	JsonReader  *reader = NULL;
	GError      *error = NULL;
	gboolean     ret;

	if (! (response && proxy_success)) {
		return false;
	}

	parser = json_parser_new ();
	reader = json_reader_new (NULL);

	ret = json_parser_load_from_data (parser,
			response->str,
			(gssize)response->len,
			&error);

	if (! ret) {
		g_critical ("failed to parse proxy response: %s",
				error->message);
		g_error_free (error);
		goto out;
	}

	json_reader_set_root (reader, json_parser_get_root (parser));

	ret = json_reader_read_member (reader, "success");
	if (! ret) {
		g_critical ("failed to find proxy response");
		goto out;
	}

	*proxy_success = json_reader_get_boolean_value (reader);

	json_reader_end_member (reader);

	ret = true;

out:
	if (reader) g_object_unref (reader);
	if (parser) g_object_unref (parser);

	return ret;
}

/**
 * Run any command via the \ref CC_OCI_PROXY.
 *
 * \param proxy \ref cc_proxy.
 * \param msg_to_send gchar.
 * \param msg_received GString.
 * \param oob_fd int.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_proxy_run_cmd(struct cc_proxy *proxy,
		gchar *msg_to_send,
		GString* msg_received,
		int *oob_fd)
{
	GIOChannel        *channel = NULL;
	struct watcher_proxy_data proxy_data;
	gboolean ret = false;
	gboolean hyper_result = false;

	if (! (proxy && msg_to_send && msg_received)) {
		return false;
	}

	if (! proxy->socket) {
		g_critical ("no proxy connection");
		return false;
	}

	proxy_data.loop = g_main_loop_new (NULL, false);

	proxy_data.msg_to_send = msg_to_send;

	proxy_data.oob_fd = oob_fd;

	proxy_data.socket_fd = g_socket_get_fd (proxy->socket);

	channel = g_io_channel_unix_new(proxy_data.socket_fd);
	if (! channel) {
		g_critical("failed to create I/O channel");
		goto out;
	}

	g_io_channel_set_encoding (channel, NULL, NULL);

	proxy_data.msg_received = msg_received;

	/* add a watcher for proxy's socket stdin */
	g_io_add_watch(channel, G_IO_OUT | G_IO_HUP,
	    (GIOFunc)cc_proxy_msg_write, &proxy_data);

	g_debug ("communicating with proxy");

	/* waiting for proxy response */
	g_main_loop_run(proxy_data.loop);

	if (! cc_proxy_hyper_check_response (msg_received,
			&hyper_result)) {
		g_critical ("failed to check proxy response");
		ret = false;
	} else {
		ret = hyper_result;
	}

	/*
	 * If we're asked for a fd out of the proxy and the command has
	 * succeeded, we can now read it.
	 */
	if (oob_fd && ret == true) {
		gboolean fd_received;

		fd_received = cc_proxy_receive_fd(proxy_data.socket_fd, oob_fd);
		if (!fd_received) {
			g_critical ("failed to receive fd");
			ret = false;
		}
	}

out:
	g_main_loop_unref (proxy_data.loop);
	g_free (proxy_data.msg_to_send);
	if (channel) {
		g_io_channel_unref(channel);
	}
	return ret;
}

/**
 * Send the initial message to the proxy
 * which will block until it is ready. 
 *
 * \param proxy \ref cc_proxy.
 * \param container_id container id.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_proxy_cmd_hello (struct cc_proxy *proxy, const char *container_id)
{
	JsonObject        *obj = NULL;
	JsonObject        *data = NULL;
	JsonNode          *root = NULL;
	JsonGenerator     *generator = NULL;
	gchar             *msg_to_send = NULL;
	GString           *msg_received = NULL;
	gboolean           ret = false;

	/* The name of the command used to initiate communicate
	 * with the proxy.
	 */
	const gchar       *proxy_cmd = "hello";

	if (! (proxy && proxy->socket && container_id)) {
		return false;
	}

	obj = json_object_new ();
	data = json_object_new ();

	json_object_set_string_member (obj, "id", proxy_cmd);

	json_object_set_string_member (data, "containerId",
			container_id);

	json_object_set_string_member (data, "ctlSerial",
			proxy->agent_ctl_socket);

	json_object_set_string_member (data, "ioSerial",
			proxy->agent_tty_socket);

	json_object_set_string_member (data, "console",
			proxy->vm_console_socket);

	json_object_set_object_member (obj, "data", data);

	root = json_node_new (JSON_NODE_OBJECT);
	generator = json_generator_new ();
	json_node_take_object (root, obj);

	json_generator_set_root (generator, root);
	g_object_set (generator, "pretty", FALSE, NULL);

	msg_to_send = json_generator_to_data (generator, NULL);

	msg_received = g_string_new("");

	if (! msg_received ) {
		goto out;
	}

	if (! cc_proxy_run_cmd(proxy, msg_to_send, msg_received, NULL)) {
		g_critical("failed to run proxy command %s: %s",
				proxy_cmd,
				msg_received->str);
		goto out;
	}

	ret = true;

	g_debug("msg received: %s", msg_received->str);

out:
	if (msg_received) {
		g_string_free(msg_received, true);
	}
	if (obj) {
		json_object_unref (obj);
	}

	return ret;
}

/**
 * Attach current proxy connection to a
 * previous registered VM (hello command)
 *
 * \param proxy \ref cc_proxy.
 * \param container_id container id.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_attach (struct cc_proxy *proxy, const char *container_id)
{

	JsonObject        *obj = NULL;
	JsonObject        *data = NULL;
	JsonNode          *root = NULL;
	JsonGenerator     *generator = NULL;
	gchar             *msg_to_send = NULL;
	GString           *msg_received = NULL;
	gboolean           ret = false;

	/* The name of the command used to initiate communicate
	 * with the proxy.
	 */
	const gchar       *proxy_cmd = "attach";

	if (! (proxy && proxy->socket && container_id)) {
		return false;
	}

	obj = json_object_new ();
	data = json_object_new ();

	json_object_set_string_member (obj, "id", proxy_cmd);

	json_object_set_string_member (data, "containerId",
			container_id);

	json_object_set_object_member (obj, "data", data);

	root = json_node_new (JSON_NODE_OBJECT);
	generator = json_generator_new ();
	json_node_take_object (root, obj);

	json_generator_set_root (generator, root);
	g_object_set (generator, "pretty", FALSE, NULL);

	msg_to_send = json_generator_to_data (generator, NULL);

	msg_received = g_string_new("");

	if (! msg_received ) {
		goto out;
	}

	if (! cc_proxy_run_cmd(proxy, msg_to_send, msg_received, NULL)) {
		g_critical("failed to run proxy command %s: %s",
				proxy_cmd,
				msg_received->str);
		goto out;
	}

	ret = true;

	g_debug("msg received: %s", msg_received->str);

out:
	if (msg_received) {
		g_string_free(msg_received, true);
	}
	if (obj) {
		json_object_unref (obj);
	}

	return ret;
}

/**
 * Send the final message to the proxy.
 *
 * \param proxy \ref cc_proxy.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_cmd_bye (struct cc_proxy *proxy, const char *container_id)
{
	JsonObject        *obj = NULL;
	JsonObject        *data = NULL;
	JsonNode          *root = NULL;
	JsonGenerator     *generator = NULL;
	gchar             *msg_to_send = NULL;
	GString           *msg_received = NULL;
	gboolean           ret = false;

	/* The name of the proxy command used to terminate
	 * communications.
	 */
	const gchar       *proxy_cmd = "bye";

	if (! (proxy && container_id)) {
		return false;
	}

	if (! cc_proxy_connect(proxy)) {
		return ret;
	}

	obj = json_object_new ();
	data = json_object_new ();

	json_object_set_string_member (obj, "id", proxy_cmd);

	json_object_set_string_member (data, "containerId",
			container_id);

	json_object_set_object_member (obj, "data", data);

	root = json_node_new (JSON_NODE_OBJECT);
	generator = json_generator_new ();
	json_node_take_object (root, obj);

	json_generator_set_root (generator, root);
	g_object_set (generator, "pretty", FALSE, NULL);

	msg_to_send = json_generator_to_data (generator, NULL);

	msg_received = g_string_new ("");

	if (! msg_received ) {
		goto out;
	}

	if (! cc_proxy_run_cmd(proxy, msg_to_send, msg_received, NULL)) {
		g_critical ("failed to run proxy command %s: %s",
				proxy_cmd,
				msg_received->str);
		goto out;
	}

	ret = true;

	g_debug("msg received: %s", msg_received->str);

out:
	if (msg_received) {
		g_string_free(msg_received, true);
	}
	if (obj) {
		json_object_unref (obj);
	}

	return ret;
}

/**
 * Ask the proxy to allocate I/O stream "sequence numbers".
 *
 * \param proxy \ref cc_proxy.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_cmd_allocate_io (struct cc_proxy *proxy,
	int *proxy_io_fd,
	int *ioBase,
	bool tty)
{
	JsonObject        *obj = NULL;
	JsonObject        *data = NULL;
	JsonNode          *root = NULL;
	JsonGenerator     *generator = NULL;
	gchar             *msg_to_send = NULL;
	GString           *msg_received = NULL;
	gboolean           ret = false;
	JsonParser        *parser = NULL;
	GError            *error = NULL;
	JsonReader        *reader = NULL;

	const gchar       *proxy_cmd = "allocateIO";
	int n_streams = IO_STREAMS_NUMBER;

	if (! proxy) {
		return false;
	}

	obj = json_object_new ();
	data = json_object_new ();

	json_object_set_string_member (obj, "id", proxy_cmd);

	/* If run interactively, allocate just 1 stream since
	 * stdout and stderr are both connected to the terminal
	 */
	if (tty) {
		json_object_set_int_member (data, "nStreams", 1);
	} else {
		json_object_set_int_member (data, "nStreams",
			n_streams);
	}

	json_object_set_object_member (obj, "data", data);

	root = json_node_new (JSON_NODE_OBJECT);
	generator = json_generator_new ();
	json_node_take_object (root, obj);

	json_generator_set_root (generator, root);
	g_object_set (generator, "pretty", FALSE, NULL);

	msg_to_send = json_generator_to_data (generator, NULL);

	msg_received = g_string_new("");

	if (! msg_received ) {
		goto out;
	}

	if (! cc_proxy_run_cmd(proxy, msg_to_send, msg_received, proxy_io_fd)) {
		g_critical("failed to run proxy command %s: %s",
				proxy_cmd,
				msg_received->str);
		goto out;
	}

	g_debug("msg received: %s", msg_received->str);

	if (!ioBase) {
		ret = true;
		goto out;
	}

	/* parse message received to get ioBase */
	parser = json_parser_new();
	ret = json_parser_load_from_data(parser,
		msg_received->str,
		(gssize) msg_received->len,
		&error);

	if (! ret) {
		g_critical ("failed to parse proxy response: %s",
				error->message);
		g_error_free (error);
		goto out;
	}

	reader = json_reader_new(json_parser_get_root(parser));
	if (!reader) {
		g_critical("failed to create reader");
		goto out;
	}

	ret = json_reader_read_member (reader, "data");
	if (! ret) {
		g_critical ("failed to find proxy data");
		goto out;
	}

	ret = json_reader_read_member (reader, "ioBase");
	if (! ret) {
		g_critical ("failed to find ioBase");
		goto out;
	}

	*ioBase = (int) json_reader_get_int_value(reader);

	json_reader_end_member (reader);

	ret = true;

out:
	if (reader) {
		g_object_unref (reader);
	}
	if (parser) {
		g_object_unref (parser);
	}
	if (msg_received) {
		g_string_free(msg_received, true);
	}
	if (obj) {
		json_object_unref (obj);
	}

	return ret;
}

/**
 * Connect to \ref CC_OCI_PROXY and wait until it is ready.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_wait_until_ready (struct cc_oci_config *config)
{
	GFile             *ctl_file = NULL;
	GFileMonitor      *monitor = NULL;
	GMainLoop         *loop = NULL;
	struct stat        st;

	if (! (config && config->proxy
				&& config->proxy->agent_ctl_socket)) {
		return false;
	}

	/* Unfortunately launching the hypervisor does not guarantee that
	 * CTL and TTY exist, for this reason we MUST wait for them before
	 * writing down any message into proxy's socket
	 */
	if (stat(config->proxy->agent_ctl_socket, &st)) {
		loop = g_main_loop_new (NULL, false);

		ctl_file = g_file_new_for_path( config->proxy->agent_ctl_socket);

		monitor = g_file_monitor(ctl_file, G_FILE_MONITOR_NONE, NULL, NULL);
		if (! monitor) {
			g_critical("failed to create a file monitor for %s",
				 config->proxy->agent_ctl_socket);
			goto out;
		}

		g_signal_connect(monitor, "changed",
			G_CALLBACK(cc_proxy_ctl_socket_created_callback), loop);

		/* last chance, if CTL socket does not exist we MUST wait for it */
		if (stat(config->proxy->agent_ctl_socket, &st)) {
			g_main_loop_run(loop);
		}
	}

	if (! cc_proxy_cmd_hello (config->proxy, config->optarg_container_id)) {
		return false;
	}
out:
	if (loop) {
		g_main_loop_unref(loop);
	}
	if (ctl_file) {
		g_object_unref(ctl_file);
	}
	if (monitor) {
		g_object_unref(monitor);
	}

	return true;
}

/**
 * Run a Hyper command via the \ref CC_OCI_PROXY.
 *
 * \note Must already be connected to the proxy.
 *
 * \param config \ref cc_oci_config.
 * \param cmd Name of hyper command to run.
 * \param payload \c JsonObject to send as message data.
 *
 * \return \c true on success, else \c false.
 */
static gboolean
cc_proxy_run_hyper_cmd (struct cc_oci_config *config,
		const char *cmd, JsonObject *payload)
{
	JsonObject        *obj = NULL;
	JsonObject        *data = NULL;
	JsonNode          *root = NULL;
	JsonGenerator     *generator = NULL;
	gboolean           ret = false;
	gchar             *msg_to_send = NULL;
	GString           *msg_received = NULL;

	/* data is optional */
	if (! (config && cmd)) {
		return false;
	}

	obj = json_object_new ();
	data = json_object_new ();

	/* tell the proxy to run in pass-through mode and forward
	 * the request on to hyperstart in the VM.
	 */
	json_object_set_string_member (obj, "id", "hyper");

	/* add the hyper command name and the data to pass to the
	 * command.
	 */
	json_object_set_string_member (data, "hyperName", cmd);
	json_object_set_object_member (data, "data", payload);

	json_object_set_object_member (obj, "data", data);

	root = json_node_new (JSON_NODE_OBJECT);
	generator = json_generator_new ();
	json_node_take_object (root, obj);

	json_generator_set_root (generator, root);
	g_object_set (generator, "pretty", FALSE, NULL);

	msg_to_send = json_generator_to_data (generator, NULL);

	msg_received = g_string_new("");

	if (! msg_received ) {
		goto out;
	}

	if (! cc_proxy_run_cmd(config->proxy, msg_to_send, msg_received, NULL)) {
		g_critical("failed to run hyper cmd %s: %s",
				cmd,
				msg_received->str);
		goto out;
	}

	g_debug("msg received: %s", msg_received->str);

	ret = true;

out:
	if (msg_received) {
		g_string_free(msg_received, true);
	}
	if (obj) {
		json_object_unref (obj);
	}

	return ret;
}

/**
 * Request \ref CC_OCI_PROXY create a new POD (container group).
 *
 * \note Must already be connected to the proxy.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_hyper_pod_create (struct cc_oci_config *config)
{
	JsonObject                   *data = NULL;
	JsonArray                    *array = NULL;
	gboolean                      ret = false;
	JsonArray                    *iface_array = NULL;
	JsonObject                   *iface_data = NULL;
	struct cc_oci_net_if_cfg     *if_cfg = NULL;
	struct cc_oci_net_ipv4_cfg   *ipv4_cfg = NULL;
	gchar                       **ifnames = NULL;
	gsize                         len;
	JsonObject                   *ipaddr_obj = NULL;
	JsonArray                    *ipaddr_arr = NULL;
	JsonArray                    *routes_array = NULL;
	JsonObject                   *route_data = NULL;
	struct cc_oci_net_ipv4_route *route = NULL;

	if (! (config && config->proxy && config->net.hostname)) {
		return false;
	}

	/* json stanza for create pod (STARTPOD without containers)*/
	data = json_object_new ();

	json_object_set_string_member (data, "hostname",
		config->net.hostname);

	/* FIXME: missing routes, dns,
	 * portmappingWhiteLists, externalNetworks ?
	 */

	array = json_array_new();

	json_object_set_array_member(data, "containers", array);

	json_object_set_string_member (data, "shareDir", "rootfs");

	/* Setup interfaces */
	iface_array = json_array_new ();

	len = g_slist_length(config->net.interfaces);
	ifnames = g_new0(gchar *, len + 1);

	for (guint i = 0; i < len; i++) {
                ifnames[i] = get_pcie_ifname(i);

		if_cfg = (struct cc_oci_net_if_cfg *)
	                g_slist_nth_data(config->net.interfaces, i);

		iface_data = json_object_new ();
		json_object_set_string_member (iface_data, "device",
			ifnames[i]);
		json_object_set_string_member (iface_data, "newDeviceName",
			if_cfg->ifname);

		ipaddr_arr = json_array_new();

		for (guint j = 0; j < g_slist_length(if_cfg->ipv4_addrs); j++) {
			ipv4_cfg = (struct cc_oci_net_ipv4_cfg *)
				g_slist_nth_data(if_cfg->ipv4_addrs, j);

			ipaddr_obj = json_object_new();
			json_object_set_string_member (ipaddr_obj, "ipAddress",
				ipv4_cfg->ip_address);
			json_object_set_string_member (ipaddr_obj, "netMask",
				ipv4_cfg->subnet_mask);
			json_array_add_object_element(ipaddr_arr, ipaddr_obj);

		}
		json_object_set_array_member (iface_data, "ipAddresses",
			ipaddr_arr);
		json_array_add_object_element(iface_array, iface_data);
	}

	json_object_set_array_member(data, "interfaces", iface_array);

	routes_array = json_array_new ();

	for (guint i = 0; i < g_slist_length(config->net.routes); i++) {
		route = (struct cc_oci_net_ipv4_route *)
	                g_slist_nth_data(config->net.routes, i);

		if ( !route->dest) {
			continue;
		}
		route_data = json_object_new ();

		json_object_set_string_member (route_data, "dest",
			route->dest);

		if (route->gateway) {
			json_object_set_string_member (route_data, "gateway",
				route->gateway);
		}
		if (route->ifname) {
			json_object_set_string_member (route_data, "device",
				route->ifname);
		}
		json_array_add_object_element(routes_array, route_data);
	}

	json_object_set_array_member(data, "routes", routes_array);

	if (! cc_proxy_run_hyper_cmd (config, "startpod", data)) {
		g_critical("failed to run pod create");
		goto out;
	}

	ret = true;

out:
	if (data) {
		json_object_unref (data);
	}

	if (ifnames) {
		g_strfreev(ifnames);
	}

	return ret;
}

/**
 * Prepare an hyperstart newcontainer command using
 * the initial worload from \ref cc_oci_config and
 * then request \ref CC_OCI_PROXY to send it.
 *
 * \param config \ref cc_oci_config.
 * \param container_id container ID
 * \param rootfs container rootfs path
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_run_hyper_new_container (struct cc_oci_config *config,
				  const char *container_id,
				  const char *rootfs, const char *image)
{
	JsonObject *newcontainer_payload= NULL;
	JsonObject *process = NULL;
	JsonArray *args= NULL;
	JsonArray *envs= NULL;

	/* json stanza for NEWCONTAINER*/
	/*
	   {
	   "id": "container_id",
	   "rootfs": "??",
	   "image": "??",
	   "process": {
	   "terminal": true,
	   "stdio": 1,
	   "stderr": 2,
	   "args": [
	   "config.process.args0",
	   "config.process.args..",
	   ],
	   "envs": [
	   {
	   "env": "config.process.env_var_name",
	   "value": "var_value"
	   }
	   ],
	   "workdir": "config.process.cwd"
	   },
	   "restartPolicy": "never", ??
	   "initialize": false ??
	   }
	 * */

	if (! config) {
		return false;
	}

	newcontainer_payload = json_object_new ();
	process  = json_object_new ();
	args     = json_array_new ();
	envs     = json_array_new ();

	json_object_set_string_member (newcontainer_payload, "id",
				container_id);
	json_object_set_string_member (newcontainer_payload, "rootfs", rootfs);

	json_object_set_string_member (newcontainer_payload, "image", image);
	/*json_object_set_string_member (newcontainer_payload, "image",
	  config->optarg_container_id);
	  */

	/* newcontainer.process */
	json_object_set_boolean_member(process, "terminal",
			config->oci.process.terminal);

	json_object_set_int_member (process, "stdio",
			config->oci.process.stdio_stream);
	json_object_set_int_member (process, "stderr",
			config->oci.process.stderr_stream);

	/* initial workload from config */
	for (gchar** p = config->oci.process.args; p && *p; p++) {
		json_array_add_string_element (args, *p);
	}

	set_env_home(config);
	for (gchar** p = config->oci.process.env; p && *p; p++) {
		JsonObject *env_var = json_object_new ();
		g_autofree char *var = g_strdup(*p);
		/* Split config.process.env to get key values (KEY=VALUE) */
		char *e = g_strstr_len (var, -1, "=");
		if (! e ){
			g_critical("failed to split enviroment variable value");
			json_object_unref (newcontainer_payload);
			return false;
		}
		*e = '\0';
		e++;
		json_object_set_string_member (env_var, "value", e);
		json_object_set_string_member (env_var, "env", var);
		json_array_add_object_element (envs, env_var);
	}

	json_object_set_string_member (process, "workdir",
			config->oci.process.cwd);

	// FIXME match with config or find a good default
	json_object_set_string_member (newcontainer_payload,
			"restartPolicy", "never");

	// FIXME match with config or find a good default
	json_object_set_boolean_member (newcontainer_payload,
			"initialize", false);

	json_object_set_array_member (process, "args", args);
	json_object_set_array_member (process, "envs", envs);
	json_object_set_object_member (newcontainer_payload,
			"process", process);

	if (! cc_proxy_run_hyper_cmd (config, "newcontainer",
				newcontainer_payload)) {
		g_critical("failed to run new container");
		json_object_unref (newcontainer_payload);
		return false;
	}

	return true;
}

/**
 * Request \ref CC_OCI_PROXY to start a new container
 * within a pod, using intial worload from \ref cc_oci_config
 *
 * \param config \ref cc_oci_config.
 * \param container_id container ID
 * \param pod_id pod container ID
 * \param rootfs container rootfs path
 * \param image container image name
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_hyper_new_pod_container(struct cc_oci_config *config,
				 const char *container_id, const char *pod_id,
				 const char *rootfs, const char *image)
{
	gboolean ret = false;

	if (! (config && config->proxy)) {
		goto out;
	}

	if (! cc_proxy_connect (config->proxy)) {
		goto out;
	}
	if (! cc_proxy_attach (config->proxy, pod_id)) {
		goto out;
	}

	if (config->oci.process.stdio_stream < 0  ||
			config->oci.process.stderr_stream < 0 ) {
		g_critical("invalid io stream number");
		goto out;
	}

	if (! cc_proxy_run_hyper_new_container (config,
						container_id,
						rootfs, image)) {
		goto out;
	}

	ret = true;
out:
	if (config && config->proxy) {
		cc_proxy_disconnect (config->proxy);
	}

	return ret;
}
/**
 * Request \ref CC_OCI_PROXY to start a new standalone
 * container (e.g. a Docker one) using intial worload
 * from \ref cc_oci_config.
 *
 * \param config
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_hyper_new_container (struct cc_oci_config *config)
{
	return cc_proxy_hyper_new_pod_container(config,
						config->optarg_container_id,
						config->optarg_container_id,
						"", "");
}

/**
 * Request \ref CC_OCI_PROXY to kill a container
 *
 * \param config \ref cc_oci_config.
 * \param signum signal number
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_hyper_kill_container (struct cc_oci_config *config, int signum)
{
	JsonObject *killcontainer_payload;
	char       *signum_str = NULL;
	gboolean    ret = false;
	const gchar *container_id;

	if (! (config && config->proxy)) {
		return false;
	}

	container_id = cc_pod_container_id(config);
	if (! container_id) {
		return false;
	}

	if (! cc_proxy_connect (config->proxy)) {
		return false;
	}
	if (! cc_proxy_attach (config->proxy, container_id)) {
		return false;
	}

	signum_str = g_strdup_printf("%d", signum);

	killcontainer_payload = json_object_new ();

	json_object_set_string_member (killcontainer_payload, "container",
		config->optarg_container_id);
	json_object_set_string_member (killcontainer_payload, "signal",
		signum_str);

	if (! cc_proxy_run_hyper_cmd (config, "killcontainer", killcontainer_payload)) {
		g_critical("failed to run cmd killcontainer");
		goto out;
	}

	ret = true;
out:
	g_free_if_set(signum_str);

	json_object_unref (killcontainer_payload);

	cc_proxy_disconnect (config->proxy);

	return ret;
}

/**
 * Request \ref CC_OCI_PROXY to destroy the POD
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_hyper_destroy_pod (struct cc_oci_config *config)
{
	JsonObject *destroypod_payload;
	gboolean ret = false;

	if (! (config && config->proxy)) {
		return false;
	}
	if (! cc_proxy_connect (config->proxy)) {
		return false;
	}
	if (! cc_proxy_attach (config->proxy, config->optarg_container_id)) {
		return false;
	}

	destroypod_payload = json_object_new ();

	if (! cc_proxy_run_hyper_cmd (config, "destroypod", destroypod_payload)) {
		g_critical("failed to run cmd destroypod");
		goto out;
	}

	ret = true;
out:
	json_object_unref (destroypod_payload);

	cc_proxy_disconnect (config->proxy);

	return ret;
}

/**
 * Request \ref CC_OCI_PROXY to execute a workload in a container.
 *
 * \param config \ref cc_oci_config.
 * \param process \ref oci_cfg_process.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_proxy_hyper_exec_command (struct cc_oci_config *config)
{
	JsonObject *payload= NULL;
	JsonObject *process_node = NULL;
	JsonArray *args= NULL;
	JsonArray *envs= NULL;
	gboolean ret = false;
	struct oci_cfg_process *process = &config->oci.process;

	if (! (config && config->proxy)) {
		goto out;
	}

	if (process->stdio_stream < 0  ||
			process->stderr_stream < 0 ) {
		g_critical("invalid io stream number");
		goto out;
	}

	payload = json_object_new ();
	process_node  = json_object_new ();
	args     = json_array_new ();
	envs     = json_array_new ();

	json_object_set_string_member (payload, "container",
			config->optarg_container_id);

	/* execcmd.process */
	json_object_set_boolean_member(process_node, "terminal",
			config->oci.process.terminal);

	json_object_set_int_member (process_node, "stdio",
			process->stdio_stream);
	json_object_set_int_member (process_node, "stderr",
			process->stderr_stream);

	for (gchar** p = process->args; p && *p; p++) {
		json_array_add_string_element (args, *p);
	}

	set_env_home(config);
	for (gchar** p = process->env; p && *p; p++) {
		JsonObject *env_var = json_object_new ();
		g_autofree char *var = g_strdup(*p);

		char *e = g_strstr_len (var, -1, "=");
		if (! e ){
			g_critical("failed to split enviroment variable value");
			goto out;
		}
		*e = '\0';
		e++;
		json_object_set_string_member (env_var, "value", e);
		json_object_set_string_member (env_var, "env", var);
		json_array_add_object_element (envs, env_var);
	}

	if (process->cwd[0]) {
		json_object_set_string_member (process_node, "workdir", process->cwd);
	}
	json_object_set_array_member (process_node, "args", args);
	json_object_set_array_member (process_node, "envs", envs);
	json_object_set_object_member (payload,
			"process", process_node);

	if (! cc_proxy_run_hyper_cmd (config, "execcmd", payload)) {
		g_critical("failed to run execcmd");
		goto out;
	}

	ret = true;
out:
	if (payload) {
		json_object_unref (payload);
	}
	return ret;
}
