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

#include <glib.h>
#include <json-glib/json-glib.h>
#include <stdbool.h>
#include "oci.h"
#include "util.h"

/** used by watcher_destroyed_vm() */
struct watcher_vm_data
{
	GMainLoop              *loop;
	struct cc_oci_config *config;
	struct oci_state *state;
	gboolean result;
};


/**
 * Stops show_container_stats main loop when the 
 * container VM is destroyed verifying 
 * if \ref CC_OCI_PROCESS_SOCKET is removed.
 *
 * \param monitor \c GFileMonitor.
 * \param file \c GFile.
 * \param other_file \c GFile.
 * \param event_type \c GFileMonitorEvent.
 * \param data \ref watcher_destroyed_vm.
 */
static void
watcher_destroyed_vm (GFileMonitor            *monitor,
		GFile                        *file,
		GFile                        *other_file,
		GFileMonitorEvent             event_type,
		struct watcher_vm_data *data)
{
	g_autofree gchar  *path = NULL;
	g_autofree gchar  *name = NULL;

	g_assert (data);

	if (event_type != G_FILE_MONITOR_EVENT_DELETED) {
		return;
	}

	path = g_file_get_path (file);
	if (! path) {
		return;
	}

	name = g_path_get_basename (path);

	if (g_strcmp0 (CC_OCI_PROCESS_SOCKET, name)) {
		return;
	}

	/* file was removed, remove monitor */
	g_object_unref (monitor);

	g_main_loop_quit (data->loop);
}

/*!
 * Get container stats (cpu, memory, etc) in json format.
 * \param config \ref cc_oci_config.
 * \param state \ref oci_state.
 *
 * \return json string on success, else NULL
 */
static gchar*
get_container_stats(struct cc_oci_config *config,
	struct oci_state *state)
{
	JsonObject  *root = NULL;
	JsonObject  *data = NULL;
	JsonObject  *resources = NULL;
	gchar       *stats_str = NULL;
	gsize        str_len = 0;


	if(config->state.status != OCI_STATUS_RUNNING){
		goto out;
	}

	root = json_object_new ();
	data = json_object_new ();
	resources = json_object_new ();

	/* Get CPU stats*/
	//FIXME: Implment cpu usage
	json_object_set_object_member (resources, "cpu_stats", json_object_new());

	/* Get Memory stats*/
	//FIXME: Implment memory usage
	json_object_set_object_member (resources, "memory_stats", json_object_new());

	/* Add resoruces node to data node */
	/* 
	   FIXME:
	   We dont use cgroups but, let use  CgroupStats node name
	   to make docker happy
        */
	json_object_set_object_member (data, "CgroupStats", resources);

	/* Add root elements */
	json_object_set_string_member (root, "type", "stats");
	json_object_set_string_member (root, "id", config->optarg_container_id);
	json_object_set_string_member (root, "id", config->optarg_container_id);
	json_object_set_object_member (root, "data", data);
	stats_str = cc_oci_json_obj_to_string (root, false, &str_len);

out:
	return stats_str;
}

static gboolean
show_interval_stats(struct watcher_vm_data *data)
{
	gchar       *stats_str = NULL;
	stats_str = get_container_stats(data->config, data->state);
	if (!stats_str){
		return false;
	}
	g_print("%s", stats_str);
	return true;
}

/*!
 * Show container stats.
 * If interval param is \c 0 will show the stats once and exit;
 * otherwise the function will block and show stats after
 * "interval" seconds.
 * \param config \ref cc_oci_config.
 * \param state \ref oci_state.
 * \param state \ref interval to show.
 * \param interval seconds to pause between displaying statistics.
 *
 * \return \c true on success, else \c false.
 */
gboolean
show_container_stats(struct cc_oci_config *config,
	struct oci_state *state, int interval)
{

	gchar       *stats_str = NULL;
	gboolean     result = false;

	GError        *error = NULL;
	GFile         *file = NULL;
	GFileMonitor  *monitor = NULL;
	struct watcher_vm_data  data = {0};

	if (interval) {
		data.loop = g_main_loop_new (NULL, 0);
		data.config = config;
		data.state = state;
		if (! data.loop) {
			g_critical ("cannot create main loop");
			goto out;
		}

		file = g_file_new_for_path (config->state.runtime_path);
		if (! file) {
			g_main_loop_unref (data.loop);
			goto out;
		}
		monitor = g_file_monitor_directory (file,
				G_FILE_MONITOR_WATCH_MOVES,
				NULL, &error);
		if (! monitor) {
			g_critical ("failed to monitor %s: %s",
					g_file_get_path (file),
					error->message);
			g_error_free (error);
			g_object_unref (file);
			g_main_loop_unref (data.loop);

			goto out;
		}

		g_signal_connect (monitor, "changed",
				G_CALLBACK (watcher_destroyed_vm),
				&data);

		g_timeout_add_seconds ((guint) interval,
				       (GSourceFunc) show_interval_stats,
				       (gpointer) &data);

		/* Monitor when vm is destroyed */
		g_main_loop_run (data.loop);
	}else {
		stats_str = get_container_stats(config, state);
		if (!stats_str){
			goto out;
		}
		g_print("%s", stats_str);
	}

	result = true;
out:
	g_free_if_set(stats_str);
	return result;
}
