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

#include <string.h>
#include <errno.h>
#include <sys/stat.h>
#include <fcntl.h>

#include <glib/gstdio.h>

#include "mount.h"
#include "common.h"
#include "namespace.h"

/** Mounts that will be ignored.
 *
 * These are standard mounts that will be created within the VM
 * automatically, hence do not need to be mounted before the VM is
 * started.
 */
static struct mntent cc_oci_mounts_to_ignore[] =
{
	{ NULL, (char *)"/proc"           , NULL, NULL, -1, -1 },
	{ NULL, (char *)"/dev"            , NULL, NULL, -1, -1 },
	{ NULL, (char *)"/dev/pts"        , NULL, NULL, -1, -1 },
	{ NULL, (char *)"/dev/shm"        , NULL, NULL, -1, -1 },
	{ NULL, (char *)"/dev/mqueue"     , NULL, NULL, -1, -1 },
	{ NULL, (char *)"/sys"            , NULL, NULL, -1, -1 },
	{ NULL, (char *)"/sys/fs/cgroup"  , NULL, NULL, -1, -1 }
};

/*!
 * Determine if the specified \ref cc_oci_mount represents a
 * mount that can be ignored.
 *
 * \param m \ref cc_oci_mount.
 *
 * \return \c true if mount can be ignored, else \c false.
 */
private gboolean
cc_oci_mount_ignore (struct cc_oci_mount *m)
{
	struct mntent  *me;
	size_t          i;
	size_t          max = CC_OCI_ARRAY_SIZE (cc_oci_mounts_to_ignore);

	if (! m) {
		return false;
	}

	for (i = 0; i < max; i++) {
		me = &cc_oci_mounts_to_ignore[i];

		if (cc_oci_found_str_mntent_match (me, m, mnt_fsname)) {
			goto ignore;
		}

		if (cc_oci_found_str_mntent_match (me, m, mnt_dir)) {
			goto ignore;
		}

		if (cc_oci_found_str_mntent_match (me, m, mnt_type)) {
			goto ignore;
		}
	}

	return false;

ignore:
	m->ignore_mount = true;
	return true;
}

/*!
 * Free the specified \ref cc_oci_mount.
 *
 * \param m \ref cc_oci_mount.
 */
void
cc_oci_mount_free (struct cc_oci_mount *m)
{
	g_assert (m);

	g_free_if_set (m->mnt.mnt_fsname);
	g_free_if_set (m->mnt.mnt_dir);
	g_free_if_set (m->mnt.mnt_type);
	g_free_if_set (m->mnt.mnt_opts);
	g_free_if_set (m->directory_created);

	g_free (m);
}

/*!
 * Free all mounts.
 *
 * \param mounts List of \ref cc_oci_mount.
 */
void
cc_oci_mounts_free_all (GSList *mounts)
{
	if (! mounts) {
		return;
	}

	g_slist_free_full (mounts, (GDestroyNotify)cc_oci_mount_free);
}

/*!
 * Mount the resource specified by \p m.
 *
 * \param m \ref cc_oci_mount.
 * \param dry_run If \c true, don't actually call \c mount(2),
 * just log what would be done.
 *
 * \return \c true on success, else \c false.
 */
private gboolean
cc_oci_perform_mount (const struct cc_oci_mount *m, gboolean dry_run)
{
	const char *fmt = "%smount%s %s of type %s "
		"onto %s with options '%s' "
		"and flags 0x%lx%s%s";

	int ret;
	struct stat st;

	if (! m) {
		return false;
	}

	g_debug (fmt, dry_run ? "Not " : "",
			"ing",
			m->mnt.mnt_fsname,
			m->mnt.mnt_type,
			m->dest,
			m->mnt.mnt_opts ? m->mnt.mnt_opts : "",
			m->flags,
			dry_run ? " (dry-run mode)" : "",
			"");

	if (dry_run) {
		return true;
	}

	if (stat (m->mnt.mnt_fsname, &st) < 0) {
		g_critical ("Unable to handle mount file:"
				"getting file status  %s (%s)",
				m->mnt.mnt_fsname, strerror (errno));
		return false;
	}

	if (S_ISREG(st.st_mode)) {
		int fd;
		fd = creat (m->dest, st.st_mode);
		if( fd < 0 ) {
			g_critical ("Unable to handle mount file:"
					"creating file %s (%s)",
					m->dest, strerror (errno));
			return false;
		}
		close(fd);

	}

	ret = mount (m->mnt.mnt_fsname,
			m->dest,
			m->mnt.mnt_type,
			m->flags,
			m->mnt.mnt_opts);
	if (ret) {
		int saved = errno;
		gchar *msg;

		msg = g_strdup_printf (": %s", strerror (saved));

		g_critical (fmt,
				"Failed to ",
				"",
				m->mnt.mnt_fsname,
				m->mnt.mnt_type,
				m->dest,
                m->mnt.mnt_opts ? m->mnt.mnt_opts : "",
				m->flags,
				"",
				msg);
		g_free (msg);
	}

	return ret == 0;
}

/*!
 * Setup required mounts.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_handle_mounts (struct cc_oci_config *config)
{
	GSList    *l;
	gboolean   ret;
	struct stat st;
	gchar* dirname_dest = NULL;
	gchar* dirname_parent_dest = NULL;
	gchar* c = NULL;
	gchar* workload_dir;

	if (! config) {
		return false;
	}

	workload_dir = cc_oci_get_workload_dir(config);
	if (! workload_dir) {
		return false;
	}

	for (l = config->oci.mounts; l && l->data; l = g_slist_next (l)) {
		struct cc_oci_mount *m = (struct cc_oci_mount *)l->data;

		if (cc_oci_mount_ignore (m)) {
			g_debug ("ignoring mount %s", m->mnt.mnt_dir);
			continue;
		}

		g_snprintf (m->dest, sizeof (m->dest),
				"%s%s",
				workload_dir, m->mnt.mnt_dir);

		if (m->mnt.mnt_fsname[0] == '/') {
			if (stat (m->mnt.mnt_fsname, &st)) {
				g_debug ("ignoring mount, %s does not exist", m->mnt.mnt_fsname);
				continue;
			}
			if (! S_ISDIR(st.st_mode)) {
				dirname_dest = g_path_get_dirname(m->dest);
			}
		}

		if (! dirname_dest) {
			dirname_dest = g_strdup(m->dest);
		}

		dirname_parent_dest = g_strdup(dirname_dest);
		if (dirname_parent_dest && ! g_file_test(dirname_parent_dest, G_FILE_TEST_IS_DIR)) {
			/* looking for first parent directory that must be created to mount dest */
			do {
				c = g_strrstr(dirname_parent_dest, "/");
				if (c) {
					*c = '\0';
				} else {
					/* no more path separators '/' */
					break;
				}
			} while(! g_file_test(dirname_parent_dest, G_FILE_TEST_IS_DIR));

			if (c) {
				/* revert last change */
				*c = '/';
				m->directory_created = dirname_parent_dest;
				dirname_parent_dest = NULL;
			}
		}

		g_free_if_set(dirname_parent_dest);

		ret = g_mkdir_with_parents (dirname_dest, CC_OCI_DIR_MODE);
		if (ret < 0) {
			g_critical ("failed to create mount directory: %s (%s)",
					m->dest, strerror (errno));
			g_free(dirname_dest);
			return false;
		}

		g_free(dirname_dest);
		dirname_dest = NULL;

		if (! cc_oci_perform_mount (m, config->dry_run_mode)) {
			return false;
		}
	}

	return true;
}

/*!
 * Unmount the mount specified by \p m.
 *
 * \param m \ref cc_oci_mount.
 *
 * \return \c true on success, else \c false.
 */
private gboolean
cc_oci_perform_unmount (const struct cc_oci_mount *m)
{
	if (! m) {
		return false;
	}

	g_debug ("unmounting %s", m->dest);

	return umount (m->dest) == 0;
}

/*!
 * Unmount all mounts.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_handle_unmounts (const struct cc_oci_config *config)
{
	GSList  *l;
	struct oci_cfg_namespace *ns;
	gboolean mountns = false;

	if (! config) {
		return false;
	}

	/**
	 * At this point qemu is not running if there is
	 * a mount namespace with a path we have join
	 * it and umount all mounted resources
	 */
	for (l = config->oci.oci_linux.namespaces;
			l && l->data;
			l = g_slist_next (l)) {
		ns = (struct oci_cfg_namespace *)l->data;

		if (ns->type == OCI_NS_MOUNT && ns->path) {
			mountns = cc_oci_ns_join (ns);
			break;
		}
	}

	/**
	 * if there is NOT a specific mount namespace return true
	 * since namespace created by unshare in \ref cc_oci_ns_setup
	 * is destroyed when qemu ends
	 */
	if (! mountns) {
		return true;
	}

	/* umount files and directories */
	for (l = config->oci.mounts; l && l->data; l = g_slist_next (l)) {
		struct cc_oci_mount *m = (struct cc_oci_mount *)l->data;

		if (m->ignore_mount) {
			/* was never mounted */
			continue;
		}

		if (! cc_oci_perform_unmount (m)) {
			g_critical("failed to umount %s", m->dest);
			return false;
		}
	}

	/* delete directories created by cc_oci_handle_mounts */
	for (l = config->oci.mounts; l && l->data; l = g_slist_next (l)) {
		struct cc_oci_mount *m = (struct cc_oci_mount *)l->data;

		if (m->ignore_mount) {
			/* was never mounted */
			continue;
		}

		if (m->directory_created) {
			if (! cc_oci_rm_rf(m->directory_created)) {
				g_critical("failed to delete %s", m->directory_created);
			}
		}
	}

	return true;
}

/*!
 * Convert the list of mounts to a JSON array.
 *
 * Note that the returned array will be empty unless any of the list of
 * mounts provided in \ref CC_OCI_CONFIG_FILE were actually mounted
 * (many are ignored as they are unecessary in the hypervisor case).
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c JsonArray on success, else \c NULL.
 */
JsonArray *
cc_oci_mounts_to_json (const struct cc_oci_config *config)
{
	JsonArray *array = NULL;
	JsonObject *mount = NULL;
	GSList *l;

	array  = json_array_new ();

	for (l = config->oci.mounts; l && l->data; l = g_slist_next (l)) {
		struct cc_oci_mount *m = (struct cc_oci_mount *)l->data;

		if (m->ignore_mount) {
			/* was never mounted */
			continue;
		}
		mount = json_object_new ();

		json_object_set_string_member (mount, "destination",
			m->dest);

		if (m->directory_created) {
			json_object_set_string_member (mount, "directory_created",
				m->directory_created);
		}

		json_array_add_object_element (array, mount);
	}

	return array;
}
