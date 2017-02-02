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
#include "mount.h"

static struct cc_oci_mount* current_mount = NULL;
static bool error_detected = false;

/** Map of mount flags. */
static struct cc_oci_mnt_flag_map {
	const char    *name;  /*!< Human-readable name */
	unsigned long  value; /*!< Numeric mount flag value */
} mnt_flag_map[] = {
	{"bind"        , MS_BIND},
	{"dirsync"     , MS_DIRSYNC},
	{"mandlock"    , MS_MANDLOCK},
	{"move"        , MS_MOVE},
	{"noatime"     , MS_NOATIME},
	{"nodev"       , MS_NODEV},
	{"nodiratime"  , MS_NODIRATIME},
	{"noexec"      , MS_NOEXEC},
	{"nosuid"      , MS_NOSUID},
	{"ro"          , MS_RDONLY},
	{"relatime"    , MS_RELATIME},
	{"remount"     , MS_REMOUNT},
	{"silent"      , MS_SILENT},
	{"strictatime" , MS_STRICTATIME},
	{"sync"        , MS_SYNCHRONOUS},
	{"rbind"       , MS_BIND | MS_REC},
	{"rprivate"    , MS_PRIVATE | MS_REC},
	{"private"     , MS_PRIVATE},
	{"rslave"      , MS_SLAVE | MS_REC},
	{"slave"       , MS_SLAVE},
	{"rshared"     , MS_SHARED | MS_REC},
	{"shared"      , MS_SHARED},

	{NULL          , 0}
};


/*!
 * \param flag String flag name.
 *
 * \return \c mount flag value that \p flag represents,
 * or \c 0 on failure (invalid/unrecognised flag name).
 */
static unsigned long int
mount_get_flag_value (const gchar *flag)
{
	struct cc_oci_mnt_flag_map* m;

	for (m = mnt_flag_map; m->name; m++) {
		if (! g_strcmp0 (m->name, flag)) {
			return m->value;
		}
	}

	return 0;
}

/*!
 * function to handle mount-options section
 *
 * \param root contains a mount flag.
 * \param[out] mount_flags mount flags separated by ','.
 *
 */
static void
handle_options_section(GNode* root, GString* mount_flags) {
	unsigned long int flag;

	flag = mount_get_flag_value(root->data);
	if (flag) {
		/* The option is in fact a mount flag, so record
		 * the mount flag, but don't save the symbolic
		 * name as we don't need it as the flag value
		 * overrides it.
		 */
		current_mount->flags |= flag;
	} else {
		g_string_append_printf(mount_flags, "%s,", (char*)root->data);
	}
}


static void
save_current_mount(GSList** mount_list) {
	if (! current_mount) {
		return;
	}

	/* Required:
	* - destination
	* - type
	* - source
	* Optional:
	* - options
	*/
	if (! current_mount->mnt.mnt_dir) {
		g_critical("missing mount destination path");
		goto err;
	}

	if (! current_mount->mnt.mnt_type) {
		g_critical("missing mount type");
		goto err;
	}

	if (! current_mount->mnt.mnt_fsname) {
		g_critical("missing mount source path");
		goto err;
	}

	*mount_list = g_slist_append((*mount_list), current_mount);
	current_mount = NULL;
	return;
err:
	cc_oci_mount_free(current_mount);
	error_detected = true;
	current_mount = NULL;
}

/*!
 * function to handle mount section
 *
 * \param root contains mount section.
 * \param config \ref cc_oci_config..
 */
static void
handle_mounts_section(GNode* root, struct cc_oci_config* config) {
	GString* mount_flags = NULL;

	if ((!root) || error_detected) {
		return;
	}
	/* null separator */
	if (!root->data) {
		save_current_mount(&config->oci.mounts);
	} else if (root->children) {
		/* create a new mount and fill it */
		if (!current_mount) {
			current_mount = g_new0 (struct cc_oci_mount, 1);
		}

		if (g_strcmp0(root->data, "destination") == 0) {
			current_mount->mnt.mnt_dir = g_strdup((gchar*)root->children->data);
		} else if (g_strcmp0(root->data, "type") == 0) {
			current_mount->mnt.mnt_type = g_strdup((gchar*)root->children->data);
		} else if (g_strcmp0(root->data, "source") == 0) {
			current_mount->mnt.mnt_fsname = g_strdup((gchar*)root->children->data);
		} else if (g_strcmp0(root->data, "options") == 0) {
			mount_flags = g_string_new("");

			/* fill mount_flags or current_mount->flags */
			g_node_children_foreach(root, G_TRAVERSE_ALL,
				(GNodeForeachFunc)handle_options_section, mount_flags);

			/* remove last ',' */
			if (mount_flags->len) {
				g_string_truncate(mount_flags, mount_flags->len-1);
				current_mount->mnt.mnt_opts = g_strdup(mount_flags->str);
			}

			g_string_free(mount_flags, true);
		}
	}
}

static bool
mounts_handle_section(GNode* root, struct cc_oci_config* config) {
	error_detected = false;

	if (! root) {
		g_critical("root node is NULL");
		return false;
	}

	if (! config) {
		g_critical("oci config is NULL");
		return false;
	}

	/* Mounts have already been loaded (via "create") */
	if (config->oci.mounts) {
		return true;
	}

	g_node_children_foreach(root, G_TRAVERSE_ALL,
		(GNodeForeachFunc)handle_mounts_section, config);

	if (! error_detected) {
		save_current_mount(&config->oci.mounts);
	}

	return !error_detected;
}

struct spec_handler mounts_spec_handler = {
	.name = "mounts",
	.handle_section = mounts_handle_section
};
