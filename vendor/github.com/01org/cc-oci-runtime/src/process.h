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

#ifndef _CC_OCI_PROCESS_H
#define _CC_OCI_PROCESS_H

gboolean cc_oci_vm_launch (struct cc_oci_config *config);

gboolean cc_run_hooks(GSList* hooks, const gchar* state_file_path,
                       gboolean stop_on_failure);

gboolean cc_oci_vm_connect (struct cc_oci_config *config);

gboolean cc_shim_launch (struct cc_oci_config *config,
			int *child_err_fd,
			int *shim_args_fd,
			int *shim_socket_fd,
			gboolean initial_workload);

GSocketConnection *cc_oci_socket_connection_from_fd (int fd);

#endif /* _CC_OCI_PROCESS_H */
