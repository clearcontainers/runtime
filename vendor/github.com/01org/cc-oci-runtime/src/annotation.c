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

#include "annotation.h"
#include "common.h"

/*!
 * Free the specified annotation.
 *
 * \param a \ref oci_cfg_annotation.
 */
private void
cc_oci_annotation_free (struct oci_cfg_annotation *a) {
	if (! a) {
		return;
	}

	g_free_if_set (a->key);
	g_free_if_set (a->value);

	g_free (a);
}

/*!
 * Free all annotations.
 *
 * \param annotations List of \ref oci_cfg_annotation.
 */
void
cc_oci_annotations_free_all (GSList *annotations) {
	if (! annotations) {
		return;
	}

	g_slist_free_full (annotations,
            (GDestroyNotify)cc_oci_annotation_free);
}

/*!
 * Convert the list of annotations to a JSON object.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c JsonObject
 */
JsonObject *
cc_oci_annotations_to_json (const struct cc_oci_config *config)
{
        JsonObject *obj = NULL;
        GSList *l;

        obj  = json_object_new ();

        for (l = config->oci.annotations; l && l->data; l = g_slist_next (l)) {
                struct oci_cfg_annotation *a = (struct oci_cfg_annotation *)l->data;

                json_object_set_string_member(obj, a->key, a->value);
        }

        return obj;
}

