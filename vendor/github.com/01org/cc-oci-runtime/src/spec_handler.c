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

#include "spec_handler.h"
#include "util.h"
#include "json.h"
#include "common.h"


/*!
 * If the virtual machine attribute ("vm") in config is NULL,
 * this function will create create it using the json from
 * SYSCONFDIR/CC_OCI_VM_CONFIG
 * or fallback default DEFAULTSDIR/CC_OCI_VM_CONFIG
 *
 * \param[in,out] config cc_oci_config struct
 *
 * \return \c true if can get vm spec data, else \c false.
 */
gboolean
get_spec_vm_from_cfg_file (struct cc_oci_config* config)
{
	bool result= true;
	GNode* vm_config = NULL;
	GNode* vm_node= NULL;
	gchar* sys_json_file = NULL;

	if (! config) {
		return false;
	}

	if (config->vm) {
		/* If vm spec data exist, do nothing */
		goto out;
	}
#ifdef UNIT_TESTING
	sys_json_file = g_strdup (TEST_DATA_DIR"/vm.json");
#else
	sys_json_file = g_build_path ("/", SYSCONFDIR,
		CC_OCI_VM_CONFIG, NULL);
	if (! g_file_test (sys_json_file, G_FILE_TEST_EXISTS)) {
		g_free_if_set (sys_json_file);
		sys_json_file = g_build_path ("/", DEFAULTSDIR,
		CC_OCI_VM_CONFIG, NULL);
	}
#endif // UNIT_TESTING
	g_debug ("Reading VM configuration from %s",
		sys_json_file);
	if (! cc_oci_json_parse(&vm_config, sys_json_file)) {
		result = false;
		goto out;
	}

	vm_node = g_node_first_child(vm_config);
	while (vm_node) {
		if (g_strcmp0(vm_node->data, vm_spec_handler.name) == 0) {
			break;
		}
		vm_node = g_node_next_sibling(vm_node);
	}
	vm_spec_handler.handle_section(vm_node, config);
	if (! config->vm) {
		g_critical ("VM json node not found");
		result = false;
	}
out:
	g_free_if_set (sys_json_file);
	g_free_node (vm_config);
	return result;
}
