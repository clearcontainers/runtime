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

/* Sandbox rootfs */
#define CC_POD_SANDBOX_ROOTFS "workloads"

/* CRI-O/ocid namespaces */
#define CC_POD_OCID_NAMESPACE "ocid/"
#define CC_POD_OCID_NAMESPACE_SIZE 5

#define CC_POD_OCID_CONTAINER_TYPE "ocid/container_type"
#define CC_POD_OCID_SANDBOX        "sandbox"
#define CC_POD_OCID_CONTAINER      "container"

#define CC_POD_OCID_SANDBOX_NAME "ocid/sandbox_name"

#include <errno.h>
#include <string.h>
#include <sys/mount.h>

#include <glib.h>
#include <gio/gunixconnection.h>

#include "pod.h"
#include "process.h"
#include "proxy.h"
#include "state.h"

/**
 * Creates a mount point structure for a
 * pod container rootfs.
 *
 * \param config \ref cc_oci_config.
 *
 * \return cc_oci_mount on success, and a \c NULL on failure.
 */
static gboolean
add_container_mount(struct cc_oci_config *config) {
	struct cc_oci_mount *m;

	if (! (config && config->pod)) {
		return false;
	}

	m = g_malloc0 (sizeof (struct cc_oci_mount));
	if (! m) {
		goto error;
	}

	m->flags = MS_BIND;

	/* Destination */
	m->mnt.mnt_dir = g_malloc0(PATH_MAX);
	if (! m->mnt.mnt_dir) {
		goto error;
	}

	g_snprintf(m->mnt.mnt_dir, PATH_MAX, "/%s/rootfs",
		   config->optarg_container_id);

	/* Source */
	m->mnt.mnt_fsname = g_strdup(config->oci.root.path);
	if (! m->mnt.mnt_fsname) {
		goto error;
	}

	/* Type */
	m->mnt.mnt_type = g_strdup("bind");
	if (! m->mnt.mnt_type) {
		goto error;
	}

	/* Add our pod container mount to the list of all mount points */
	config->oci.mounts = g_slist_append(config->oci.mounts, m);

	return true;

error:
	if (m) {
		g_free_if_set(m->mnt.mnt_dir);
		g_free_if_set(m->mnt.mnt_fsname);
		g_free_if_set(m->mnt.mnt_type);
		g_free_if_set(m);
	}

	return false;
}


/**
 * Handle pod related OCI annotations.
 * This routine will build the config->pod structure
 * based on the pod related OCI annotations.
 *
 * \param OCI config \ref cc_oci_config.
 * \param OCI annotation \ref oci_state.
 *
 * \return 0 on success, and a negative \c errno on failure.
 */
int
cc_pod_handle_annotations(struct cc_oci_config *config, struct oci_cfg_annotation *annotation)
{
	if (! (config && annotation)) {
		return -EINVAL;
	}

	if (! (annotation->key && annotation->value)) {
		return -EINVAL;
	}

	/* We only handle CRI-O/ocid annotations for now */
	if (strncmp(annotation->key, CC_POD_OCID_NAMESPACE,
		    CC_POD_OCID_NAMESPACE_SIZE) != 0) {
		return 0;
	}

	if (! config->pod) {
		config->pod = g_malloc0 (sizeof (struct cc_pod));
		if (! config->pod) {
			return -ENOMEM;
		}
	}

	if (g_strcmp0(annotation->key, CC_POD_OCID_CONTAINER_TYPE) == 0) {
		if (g_strcmp0(annotation->value, CC_POD_OCID_SANDBOX) == 0) {
			config->pod->sandbox = true;
			config->pod->sandbox_name = g_strdup(config->optarg_container_id);

			if (! add_container_mount(config)) {
				return -ENOMEM;
			}

			g_snprintf (config->pod->sandbox_workloads,
				    sizeof (config->pod->sandbox_workloads),
				    "%s/%s/%s",
				    CC_OCI_RUNTIME_DIR_PREFIX,
				    config->optarg_container_id,
				    CC_POD_SANDBOX_ROOTFS);
		} else if (g_strcmp0(annotation->value, CC_POD_OCID_CONTAINER) == 0) {
			config->pod->sandbox = false;
		}
	} else if (g_strcmp0(annotation->key, CC_POD_OCID_SANDBOX_NAME) == 0) {
		if (config->pod->sandbox_name) {
			g_free(config->pod->sandbox_name);
		}
		config->pod->sandbox_name = g_strdup(annotation->value);

		g_snprintf (config->pod->sandbox_workloads,
			    sizeof (config->pod->sandbox_workloads),
			    "%s/%s/%s",
			    CC_OCI_RUNTIME_DIR_PREFIX,
			    config->pod->sandbox_name,
			    CC_POD_SANDBOX_ROOTFS);

		if (! add_container_mount(config)) {
			return -ENOMEM;
		}
	}

	return 0;
}

/**
 * Free resources associated with \p CRI-O/ocid
 *
 * \param ocid \ref cc_ocid.
 *
 */
void
cc_pod_free (struct cc_pod *pod) {
	if (! pod) {
		return;
	}

	g_free_if_set (pod->sandbox_name);

	g_free (pod);
}

/**
 * Create a container within a pod.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_pod_container_create (struct cc_oci_config *config)
{
	gboolean           ret = false;
	ssize_t            bytes;
	char               buffer[2] = { '\0' };
	g_autofree gchar  *timestamp = NULL;
	int                shim_err_fd = -1;
	int                shim_args_fd = -1;
	int                shim_socket_fd = -1;
	int                proxy_fd = -1;
	int                proxy_io_fd = -1;
	int                ioBase = -1;
	GSocketConnection *shim_socket_connection = NULL;
	GError            *error = NULL;

	if (! (config && config->pod && config->proxy)) {
		return false;
	}

	timestamp = cc_oci_get_iso8601_timestamp ();
	if (! timestamp) {
		goto out;
	}

	config->state.status = OCI_STATUS_CREATED;

	/* Connect and attach to the proxy first */
	if (! cc_proxy_connect (config->proxy)) {
		goto out;
	}

	if (! cc_proxy_attach (config->proxy, config->pod->sandbox_name)) {
		goto out;
	}

	/* Launch the shim child before the state file is created.
	 *
	 * Required since the state file must contain the workloads pid,
	 * and for our purposes the workload pid is the pid of the shim.
	 *
	 * The child blocks waiting for a write to shim_args_fd.
	 */
	if (! cc_shim_launch (config, &shim_err_fd, &shim_args_fd, &shim_socket_fd, true)) {
		goto out;
	}

	/* Create the pid file. */
	if (config->pid_file) {
		ret = cc_oci_create_pidfile (config->pid_file,
				config->state.workload_pid);
		if (! ret) {
			goto out;
		}
	}

	proxy_fd = g_socket_get_fd (config->proxy->socket);
	if (proxy_fd < 0) {
		g_critical ("invalid proxy fd: %d", proxy_fd);
		goto out;
	}

	bytes = write (shim_args_fd, &proxy_fd, sizeof (proxy_fd));
	if (bytes < 0) {
		g_critical ("failed to send proxy fd to shim child: %s",
			strerror (errno));
		goto out;
	}

	if (! cc_proxy_cmd_allocate_io(config->proxy,
				&proxy_io_fd, &ioBase,
				config->oci.process.terminal)) {
		goto out;
	}

	bytes = write (shim_args_fd, &ioBase, sizeof (ioBase));
	if (bytes < 0) {
		g_critical ("failed to send proxy ioBase to shim child: %s",
			strerror (errno));
		goto out;
	}

	/* send proxy IO fd to cc-shim child */
	shim_socket_connection = cc_oci_socket_connection_from_fd(shim_socket_fd);
	if (! shim_socket_connection) {
		g_critical("failed to create a socket connection to send proxy IO fd");
		goto out;
	}

	ret = g_unix_connection_send_fd (G_UNIX_CONNECTION (shim_socket_connection),
		proxy_io_fd, NULL, &error);

	if (! ret) {
		g_critical("failed to send proxy IO fd");
		goto out;
	}

	/* save ioBase */
	config->oci.process.stdio_stream = ioBase;
	if ( config->oci.process.terminal) {
		/* For tty, pass stderr seq as 0, so that stdout and
		 * and stderr are redirected to the terminal
		 */
		config->oci.process.stderr_stream = 0;
	} else {
		config->oci.process.stderr_stream = ioBase + 1;
	}

	close (shim_args_fd);
	shim_args_fd = -1;

	g_debug ("checking shim setup (blocking)");

	bytes = read (shim_err_fd,
			buffer,
			sizeof (buffer));
	if (bytes > 0) {
		g_critical ("shim setup failed");
		ret = false;
		goto out;
	}

	/* Create the state file now that all information is
	 * available.
	 */
	g_debug ("Creating state file for the pod container");

	ret = cc_oci_state_file_create (config, timestamp);
	if (! ret) {
		g_critical ("failed to create state file");
		goto out;
	}

	/* We can now disconnect from the proxy (but the shim
	 * remains connected).
	 */
	ret = cc_proxy_disconnect (config->proxy);

out:
	if (shim_err_fd != -1) close (shim_err_fd);
	if (shim_args_fd != -1) close (shim_args_fd);
	if (shim_socket_fd != -1) close (shim_socket_fd);

	return ret;
}

/**
 * Start a container within a pod.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_pod_container_start (struct cc_oci_config *config)
{
	const gchar *pod_id;

	if (! (config && config->pod && ! config->pod->sandbox)) {
		return false;
	}

	pod_id = cc_pod_container_id(config);
	if (! pod_id) {
		return false;
	}

	g_debug("Attaching to pod %s", pod_id);

	return cc_proxy_hyper_new_pod_container(config,
						config->optarg_container_id,
						pod_id,
						"rootfs", config->optarg_container_id);
}

/**
 * Returns the pod container ID for any container.
 * For a pod or for a standalone container, it
 * simply returns config->optarg_container_id.
 * For a container running within a pod, this will
 * return the pod container ID.
 *
 * \param config \ref cc_oci_config.
 *
 * \return the pod container ID on success, else \c NULL.
 */
const gchar *
cc_pod_container_id(struct cc_oci_config *config)
{
	if (! config) {
		return NULL;
	}

	/* This is a container running within a pod */
	if (config->pod && ! config->pod->sandbox) {
		return config->pod->sandbox_name;
	}

	return config->optarg_container_id;
}

/**
 * cc_pod_sandbox tells if a container is a pod
 * sandbox or not.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true if the container is a pod sanbox, \c false otherwise
 */
gboolean
cc_pod_is_sandbox(struct cc_oci_config *config)
{
	if (config && config->pod && config->pod->sandbox) {
		return true;
	}

	return false;
}

/**
 * cc_pod_sandbox tells if a container is a virtual machine
 * or an intra pod container.
 * This is equivalent to being a sandbox or not except for
 * the standalone case where a container is a VM.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true if the container is a virtual machine, \c false otherwise
 */
gboolean
cc_pod_is_vm(struct cc_oci_config *config)
{
	if (config && config->pod && ! config->pod->sandbox) {
		return false;
	}

	return true;
}
