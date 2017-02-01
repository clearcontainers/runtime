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

#ifndef _CC_OCI_STATE_H
#define _CC_OCI_STATE_H

gboolean cc_oci_state_file_get (struct cc_oci_config *config);
struct oci_state *cc_oci_state_file_read (const char *file);
void cc_oci_state_free (struct oci_state *state);
gboolean cc_oci_state_file_create (struct cc_oci_config *config,
		const char *created_timestamp);
gboolean cc_oci_state_file_delete (const struct cc_oci_config *config);
gboolean cc_oci_state_file_exists (struct cc_oci_config *config);
const char *cc_oci_status_to_str (enum oci_status status);
enum oci_status cc_oci_str_to_status (const char *str);
int cc_oci_status_length (void);

#endif /* _CC_OCI_STATE_H */
