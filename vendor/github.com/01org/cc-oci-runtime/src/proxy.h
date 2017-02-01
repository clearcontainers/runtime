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

#ifndef _CC_OCI_PROXY_H
#define _CC_OCI_PROXY_H

#include <stdbool.h>
#include <gio/gio.h>
#include "oci.h"

/* allocate 2 streams, stdio and stderr */
#define IO_STREAMS_NUMBER 2

/*
 * 4 bytes for the message length.
 * 4 bytes for the message flags.
 */
#define HEADER_MESSAGE_LENGTH 4
#define HEADER_MESSAGE_FLAGS  4
#define MESSAGE_HEADER_LENGTH (HEADER_MESSAGE_LENGTH+HEADER_MESSAGE_FLAGS)

/*
 * As we can not send OOB data through a stream socket
 * without sending actual data, the proxy will signal
 * OOB data by sending a single byte message: 'F'.
 */
#define OOB_FD_FLAG 'F'

gboolean cc_proxy_connect (struct cc_proxy *proxy);
gboolean cc_proxy_disconnect (struct cc_proxy *proxy);
gboolean cc_proxy_attach (struct cc_proxy *proxy, const char *container_id);
gboolean cc_proxy_wait_until_ready (struct cc_oci_config *config);
gboolean cc_proxy_hyper_pod_create (struct cc_oci_config *config);
gboolean cc_proxy_cmd_bye (struct cc_proxy *proxy, const char *container_id);
gboolean cc_proxy_cmd_allocate_io (struct cc_proxy *proxy, int *proxy_io_fd,
		int *ioBase, bool tty);
gboolean
cc_proxy_hyper_kill_container (struct cc_oci_config *config, int signum);
gboolean cc_proxy_hyper_destroy_pod (struct cc_oci_config *config);
gboolean cc_proxy_run_hyper_new_container (struct cc_oci_config *config,
					const char *container_id,
					const char *rootfs, const char *image);
gboolean cc_proxy_hyper_new_pod_container(struct cc_oci_config *config,
					const char *container_id, const char *pod_id,
					const char *rootfs, const char *image);
gboolean cc_proxy_hyper_new_container (struct cc_oci_config *config);
void cc_proxy_free (struct cc_proxy *proxy);
gboolean cc_proxy_attach (struct cc_proxy *proxy, const char *container_id);
gboolean cc_proxy_hyper_exec_command (struct cc_oci_config *config);
#endif /* _CC_OCI_PROXY_H */
