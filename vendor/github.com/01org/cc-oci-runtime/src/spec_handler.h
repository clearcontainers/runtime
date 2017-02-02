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

#ifndef _CC_OCI_SPEC_HANDLER_H
#define _CC_OCI_SPEC_HANDLER_H

#include <stdbool.h>

#include <glib.h>

#include "oci.h"

/** A spec-handler is a handler for each section
 * of config.json (spec file), spec-handler is used
 * to fill up struct cc_oci_config
 */
struct spec_handler {
	/*! Name of spec-handler (required) */
	char name[LINE_MAX];

	/*! Function that will be called to handle spec sections (required) */
	bool (*handle_section)(GNode*, struct cc_oci_config*);
};

extern struct spec_handler annotations_spec_handler;
extern struct spec_handler hooks_spec_handler;
extern struct spec_handler mounts_spec_handler;
extern struct spec_handler platform_spec_handler;
extern struct spec_handler process_spec_handler;
extern struct spec_handler root_spec_handler;
extern struct spec_handler vm_spec_handler;
extern struct spec_handler linux_spec_handler;

gboolean get_spec_vm_from_cfg_file (struct cc_oci_config* config);

#endif /* _CC_OCI_SPEC_HANDLER_H */
