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

#ifndef _CC_OCI_NAMESPACE_H
#define _CC_OCI_NAMESPACE_H

void cc_oci_ns_free (struct oci_cfg_namespace *ns);
gboolean cc_oci_ns_setup (struct cc_oci_config *config);
const char *cc_oci_ns_to_str (enum oci_namespace ns);
enum oci_namespace cc_oci_str_to_ns (const char *str);
JsonArray *
cc_oci_ns_to_json (const struct cc_oci_config *config);
gboolean cc_oci_ns_join(struct oci_cfg_namespace *ns);

#endif /* _CC_OCI_NAMESPACE_H */
