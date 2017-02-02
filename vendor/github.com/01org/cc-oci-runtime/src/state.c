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

/**
 * \file
 *
 * State-handling routines.
 */

#include <string.h>
#include <stdbool.h>

#include <glib.h>
#include <glib/gstdio.h>

#include "common.h"
#include "oci.h"
#include "util.h"
#include "state.h"
#include "runtime.h"
#include "mount.h"
#include "namespace.h"
#include "annotation.h"
#include "json.h"
#include "config.h"
#include "spec_handler.h"

#define update_subelements_and_strdup(node, data, member) \
	if (node && node->data) { \
		data->state->member = g_strdup(node->data); \
		(*(data->subelements_count))++; \
	}

struct handler_data;

static void handle_state_ociVersion_section(GNode*, struct handler_data*);
static void handle_state_id_section(GNode*, struct handler_data*);
static void handle_state_pid_section(GNode*, struct handler_data*);
static void handle_state_bundlePath_section(GNode*, struct handler_data*);
static void handle_state_commsPath_section(GNode*, struct handler_data*);
static void handle_state_processPath_section(GNode*, struct handler_data*);
static void handle_state_status_section(GNode*, struct handler_data*);
static void handle_state_created_section(GNode*, struct handler_data*);
static void handle_state_mounts_section(GNode*, struct handler_data*);
static void handle_state_namespaces_section(GNode*, struct handler_data*);
static void handle_state_console_section(GNode*, struct handler_data*);
static void handle_state_vm_section(GNode*, struct handler_data*);
static void handle_state_proxy_section(GNode*, struct handler_data*);
static void handle_state_pod_section(GNode*, struct handler_data*);
static void handle_state_annotations_section(GNode*, struct handler_data*);
static void handle_state_process_section(GNode* node, struct handler_data* data);

/*! Used to handle each section in \ref CC_OCI_STATE_FILE. */
static struct state_handler {
	/** Name of JSON element in \ref CC_OCI_STATE_FILE. */
	const char* name;

	/** Function to handle JSON element. */
	void (*handle_section)(GNode* node, struct handler_data* state);

	/** Set to zero if element is optional. */
	const size_t subelements_needed;

	/** A state handler is considered to have run successfully if
	 * this value matches \ref subelements_needed.
	 */
	size_t subelements_count;
} state_handlers[] = {
	{ "ociVersion"  , handle_state_ociVersion_section  , 1 , 0 },
	{ "id"          , handle_state_id_section          , 1 , 0 },
	{ "pid"         , handle_state_pid_section         , 1 , 0 },
	{ "bundlePath"  , handle_state_bundlePath_section  , 1 , 0 },
	{ "commsPath"   , handle_state_commsPath_section   , 1 , 0 },
	{ "processPath" , handle_state_processPath_section , 1 , 0 },
	{ "status"      , handle_state_status_section      , 1 , 0 },
	{ "created"     , handle_state_created_section     , 1 , 0 },
	{ "mounts"      , handle_state_mounts_section      , 0 , 0 },
	{ "console"     , handle_state_console_section     , 0 , 0 },
	{ "vm"          , handle_state_vm_section          , 6 , 0 },
	{ "proxy"       , handle_state_proxy_section       , 2 , 0 },
	{ "pod"         , handle_state_pod_section         , 0 , 0 },
	{ "annotations" , handle_state_annotations_section , 0 , 0 },
	{ "namespaces"  , handle_state_namespaces_section  , 0 , 0 },

	/* terminator */
	{ NULL, NULL, 0, 0 }
};

/*!
 * handler data is provided to each handler section
 * to fill up an oci_state struct and validate how many
 * subelements must have a section
 */
struct handler_data {
	struct oci_state* state;
	 /*! if subelements_count is equal to subelements_needed then all
	  * subelements were found, otherwise some subelements are
	  * missed and an error must be reported
	  */
	size_t* subelements_count;
};

/** Map of \ref oci_status values to human-readable strings. */
static struct cc_oci_map oci_status_map[] =
{
	{ OCI_STATUS_CREATED , "created" },
	{ OCI_STATUS_RUNNING , "running" },
	{ OCI_STATUS_PAUSED  , "paused"  },
	{ OCI_STATUS_STOPPED , "stopped" },
	{ OCI_STATUS_STOPPING, "stopping"},

	{ OCI_STATUS_INVALID , NULL      }
};

/**
 * Determine the human-readable string to be used to show the state
 * of the specified VM.
 *
 * \param config \ref cc_oci_config.
 *
 * \return static string representing state on success, else \c NULL.
 */
private
const gchar *
cc_oci_status_get (const struct cc_oci_config *config)
{
	if (! config) {
		return NULL;
	}

	return cc_oci_status_to_str (config->state.status);
}

/*!
 * handler for ociVersion section.
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_ociVersion_section(GNode* node, struct handler_data* data) {
	update_subelements_and_strdup(node, data, oci_version);
}

/*!
 *  handler for id section.
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_id_section(GNode* node, struct handler_data* data) {
	update_subelements_and_strdup(node, data, id);
}

/*!
 *  handler for pid section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_pid_section(GNode* node, struct handler_data* data) {
	gchar* endptr = NULL;

	if (node) {
		if (! node->data) {
			return;
		}
		data->state->pid =
			(GPid)g_ascii_strtoll((char*)node->data, &endptr, 10);
		if (endptr != node->data) {
			(*(data->subelements_count))++;
		} else {
			g_critical("failed to convert '%s' to int",
			    (char*)node->data);
		}
	}
}

/*!
 *  handler for bundlePath section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_bundlePath_section(GNode* node, struct handler_data* data) {
	update_subelements_and_strdup(node, data, bundle_path);
}

/*!
 *  handler for commsPath section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_commsPath_section(GNode* node, struct handler_data* data) {
	update_subelements_and_strdup(node, data, comms_path);
}


/*!
 *  handler for processPath section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_processPath_section (GNode* node, struct handler_data* data) {
	update_subelements_and_strdup(node, data, procsock_path);
}

/*!
 *  handler for status section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_status_section(GNode* node, struct handler_data* data) {
	if (node && node->data) {
		data->state->status = cc_oci_str_to_status ((const gchar *)node->data);
		(*(data->subelements_count))++;
	}
}

/*!
 * handler for created section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_created_section(GNode* node, struct handler_data* data) {
	update_subelements_and_strdup(node, data, create_time);
}

/*!
 * handler for mounts section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_mounts_section(GNode* node, struct handler_data* data) {
	struct cc_oci_mount* m;

	if (! (node && node->data)) {
		return;
	}
	if (! (node->children && node->children->data)) {
		g_critical("%s missing value", (char*)node->data);
		return;
	}

	if (! g_strcmp0(node->data, "destination")) {
		m = g_new0 (struct cc_oci_mount, 1);
		g_strlcpy (m->dest, (char*)node->children->data, sizeof (m->dest));
		m->ignore_mount = false;
		data->state->mounts = g_slist_append(data->state->mounts, m);
	} else if (! g_strcmp0(node->data, "directory_created")) {
		GSList *l = g_slist_last(data->state->mounts);
		if (l) {
			m = (struct cc_oci_mount*)l->data;
			m->directory_created = g_strdup((char*)node->children->data);
		}
	}
}

/*!
 * handler for namespaces section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_namespaces_section(GNode* node, struct handler_data* data) {
	struct oci_cfg_namespace* n;

	if (! (node && node->data) || ! (data && data->state)) {
		return;
	}
	if (! (node->children && node->children->data)) {
		g_critical("%s missing value", (char*)node->data);
		return;
	}

	if (! g_strcmp0(node->data, "type")) {
		n = g_new0 (struct oci_cfg_namespace, 1);
		n->type = cc_oci_str_to_ns((char*)node->children->data);
		data->state->namespaces = g_slist_append(data->state->namespaces, n);
	} else if (! g_strcmp0(node->data, "path")) {
		GSList *l = g_slist_last(data->state->namespaces);
		if (l) {
			n = (struct oci_cfg_namespace*)l->data;
			n->path = g_strdup((char*)node->children->data);
		}
	}
}

/*!
 * handler for console section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_console_section(GNode* node, struct handler_data* data) {
	if (! (node && node->data)) {
		return;
	}
	if (! (node->children && node->children->data)) {
		g_critical("%s missing value", (char*)node->data);
		return;
	}
	if (g_strcmp0(node->data, "path") == 0) {
		(*(data->subelements_count))++;
		data->state->console = g_strdup(node->children->data);
	} else {
		g_critical("unknown console option: %s", (char*)node->data);
	}
}

/*!
 * handler for vm section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_vm_section(GNode* node, struct handler_data* data) {
	struct cc_oci_vm_cfg *vm;

	if (! (node && node->data)) {
		return;
	}
	if (! (node->children && node->children->data)) {
		g_critical("%s missing value", (char*)node->data);
		return;
	}

	g_assert (data->state);

	vm = data->state->vm;

	g_assert (vm);

	if (g_strcmp0(node->data, "workload_path") == 0) {
		g_strlcpy (vm->workload_path,
				node->children->data,
				sizeof (vm->workload_path));
		(*(data->subelements_count))++;
	} else if (g_strcmp0(node->data, "hypervisor_path") == 0) {
		g_strlcpy (vm->hypervisor_path,
				node->children->data,
				sizeof (vm->hypervisor_path));
		(*(data->subelements_count))++;
	} else if (g_strcmp0(node->data, "kernel_path") == 0) {
		g_strlcpy (vm->kernel_path,
				node->children->data,
				sizeof (vm->kernel_path));
		(*(data->subelements_count))++;
	} else if (g_strcmp0(node->data, "image_path") == 0) {
		g_strlcpy (vm->image_path,
				node->children->data,
				sizeof (vm->image_path));
		(*(data->subelements_count))++;
	} else if (g_strcmp0(node->data, "kernel_params") == 0) {
		vm->kernel_params = g_strdup(node->children->data);
		(*(data->subelements_count))++;
	} else if (g_strcmp0(node->data, "pid") == 0) {
		gchar *endptr = NULL;
		vm->pid = (GPid)g_ascii_strtoll((char*)node->children->data, &endptr, 10);
		if (endptr != node->children->data) {
			(*(data->subelements_count))++;
		} else {
			g_critical("failed to convert '%s' to int",
			    (char*)node->children->data);
		}
	} else {
		g_critical("unknown console option: %s", (char*)node->data);
	}
}

/*!
 * handler for proxy section.
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_proxy_section(GNode* node, struct handler_data* data) {
	struct cc_proxy *proxy;

	if (! (node && node->data)) {
		return;
	}
	if (! (node->children && node->children->data)) {
		g_critical("%s missing value", (char*)node->data);
		return;
	}

	g_assert (data->state);

	proxy = data->state->proxy;

	g_assert (proxy);

	if (g_strcmp0(node->data, "ctlSocket") == 0) {
		proxy->agent_ctl_socket =
			g_strdup ((gchar *)node->children->data);
		(*(data->subelements_count))++;
	} else if (g_strcmp0(node->data, "ioSocket") == 0) {
		proxy->agent_tty_socket =
			g_strdup ((gchar *)node->children->data);
		(*(data->subelements_count))++;
	} else if (g_strcmp0(node->data, "consoleSocket") == 0) {
		proxy->vm_console_socket =
			g_strdup ((gchar *)node->children->data);
		(*(data->subelements_count))++;
	} else {
		g_critical("unknown proxy option: %s", (char*)node->data);
	}
}

/*!
 * handler for pod section.
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_pod_section(GNode* node, struct handler_data* data) {
	struct cc_pod *pod;

	if (! (node && node->data)) {
		return;
	}
	if (! (node->children && node->children->data)) {
		g_critical("%s missing value", (char*)node->data);
		return;
	}

	if (! (data && data->state)) {
		g_critical("Missing handler data");
		return;
	}

	if (! data->state->pod) {
		data->state->pod = g_malloc0 (sizeof(struct cc_pod));
		if (! data->state->pod) {
			g_critical("Could not allocate pod");
			return;
		}
	}

	pod = data->state->pod;

	if (g_strcmp0(node->data, "sandbox") == 0) {
		pod->sandbox = !g_strcmp0 ((gchar *)node->children->data, "true")
			? true : false;
		(*(data->subelements_count))++;
	} else if (g_strcmp0(node->data, "sandbox_name") == 0) {
		pod->sandbox_name =
			g_strdup ((gchar *)node->children->data);
		(*(data->subelements_count))++;
	} else {
		g_critical("unknown pod option: %s", (char*)node->data);
	}
}

/*!
 * handler for annotations section
 *
 * \param node \c GNode.
 * \param data \ref handler_data.
 */
static void
handle_state_annotations_section(GNode* node, struct handler_data* data)
{
        struct oci_cfg_annotation *ann = NULL;

        if (! (node && node->data)) {
                return;
        }

        if (! (node->children && node->children->data)) {
                g_critical("%s missing value", (char*)node->data);
                return;
        }

        g_assert(data->state);

        ann = g_new0 (struct oci_cfg_annotation, 1);
        ann->key = g_strdup ((gchar *)node->data);
        if (node->children->data) {
                ann->value = g_strdup ((gchar *)node->children->data);
        }

        data->state->annotations = g_slist_prepend(data->state->annotations,
                                                        ann);
}

/*!
* handler for process section usig oci spec handlers
*
* \param node \c GNode.
* \param data \ref handler_data.
*/
static void
handle_state_process_section(GNode* node, struct handler_data* data)
{
	struct cc_oci_config config =  { {0} };;

	g_assert(data->state);

	config.oci.process.args = NULL;
	config.oci.process.env  = NULL;

	data->state->process = g_new0(struct oci_cfg_process, 1);

	process_spec_handler.handle_section(node, &config);

	*data->state->process = config.oci.process;
}

/*!
 * process all sections in state.json using the right section handler
 *
 * \param node \c GNode.
 * \param state \ref oci_state.
 */
static void
handle_state_sections(GNode* node, struct oci_state* state) {
	struct state_handler* handler;
	struct handler_data data = { .state=state };

	if (! (node && node->data)) {
		return;
	}

	for (handler=state_handlers; handler->name; handler++) {
		if (g_strcmp0(handler->name, node->data) == 0) {
			data.subelements_count = &handler->subelements_count;
			g_node_children_foreach(node, G_TRAVERSE_ALL,
				(GNodeForeachFunc)handler->handle_section, &data);
			return;
		}
	}
	/* Handle "process" node using oci spec handlers */
	if (g_strcmp0(node->data, "process") == 0) {
		handle_state_process_section(node, &data);
		return;
	}

	g_critical("handler not found %s", (char*)node->data);
}

/*!
 * Update the specified config with the state file path.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_state_file_get (struct cc_oci_config *config)
{
	g_assert (config);

	if (! config->state.runtime_path[0]) {
		return false;
	}

	g_snprintf (config->state.state_file_path,
			sizeof (config->state.state_file_path),
			"%s/%s",
			config->state.runtime_path,
			CC_OCI_STATE_FILE);

	return true;
}

/*!
 * Determine if the state file exists.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_state_file_exists (struct cc_oci_config *config)
{
	if (! config) {
		return false;
	}

	if (! cc_oci_runtime_path_get (config)) {
		return false;
	}

	if (! cc_oci_state_file_get (config)) {
		return false;
	}

	return g_file_test (config->state.state_file_path,
			G_FILE_TEST_EXISTS);
}

/*!
 * Read the state file.
 *
 * \param file Full path to \ref CC_OCI_STATE_FILE state file.
 *
 * \return Newly-allocated \ref oci_state on success, else \c NULL.
 */
struct oci_state *
cc_oci_state_file_read (const char *file)
{
	GNode* node = NULL;
	struct oci_state *state = NULL;
	struct state_handler* handler;

	if (! file) {
		return NULL;
	}

	if (! cc_oci_json_parse(&node, file)) {
		g_critical("failed to parse json file: %s", file);
		return NULL;
	}

	state = g_new0 (struct oci_state, 1);
	if (state) {
#ifdef DEBUG
		cc_oci_node_dump (node);
#endif // DEBUG

		state->vm = g_malloc0 (sizeof(struct cc_oci_vm_cfg));
		if (! state->vm) {
			g_free (state);
			state = NULL;
			goto out;
		}

		state->proxy = g_malloc0 (sizeof(struct cc_proxy));
		if (! state->proxy) {
			g_free (state->vm);
			g_free (state);
			state = NULL;
			goto out;
		}

		/* reset subelements_count */
		for (handler=state_handlers; handler->name; ++handler) {
			handler->subelements_count = 0;
		}

		g_node_children_foreach(node, G_TRAVERSE_ALL,
			(GNodeForeachFunc)handle_state_sections, state);

		for (handler=state_handlers; handler->name; ++handler) {
			if (handler->subelements_count < handler->subelements_needed) {
				g_critical("failed to run handler: %s", handler->name);
				cc_oci_state_free(state);
				state = NULL;
				break;
			}
		}
	}

out:
	g_free_node(node);
	return state;
}

/*!
 * Free all resources associated with the specified \ref oci_state.
 *
 * \param state \ref oci_state.
 */
void
cc_oci_state_free (struct oci_state *state)
{
	if (! state) {
		return;
	}

	g_free_if_set (state->oci_version);
	g_free_if_set (state->id);
	g_free_if_set (state->bundle_path);
	g_free_if_set (state->comms_path);
	g_free_if_set (state->procsock_path);
	g_free_if_set (state->create_time);
	g_free_if_set (state->console);

	if(state->process) {
		if (state->process->args) {
			g_strfreev (state->process->args);
		}

		if (state->process->env) {
			g_strfreev (state->process->env);
		}

		g_free_if_set (state->process);
	}

	if (state->mounts) {
		cc_oci_mounts_free_all (state->mounts);
	}

	if (state->namespaces) {
		g_slist_free_full(state->namespaces,
			(GDestroyNotify)cc_oci_ns_free);
	}

	if (state->vm) {
		g_free_if_set (state->vm->kernel_params);
		g_free (state->vm);
	}

        if (state->annotations) {
                cc_oci_annotations_free_all(state->annotations);
        }

	if (state->proxy) {
		g_free_if_set(state->proxy->agent_ctl_socket);
		g_free_if_set(state->proxy->agent_tty_socket);
		g_free_if_set(state->proxy->vm_console_socket);
		g_free (state->proxy);
	}

	if (state->pod) {
		g_free_if_set (state->pod->sandbox_name);
		g_free (state->pod);
	}

	g_free (state);
}

/*!
 * Create the state file for the specified \p config.
 *
 * \param config \ref cc_oci_config.
 * \param created_timestamp ISO 8601 timestamp for when VM Was created.
 *
 * \see https://github.com/opencontainers/specs/blob/master/runtime.md
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_state_file_create (struct cc_oci_config *config,
		const char *created_timestamp)
{
	JsonObject  *obj = NULL;
	JsonObject  *console = NULL;
	JsonObject  *vm = NULL;
	JsonObject  *proxy = NULL;
	JsonObject  *annotation_obj = NULL;
	JsonArray   *mounts = NULL;
	JsonArray   *namespaces = NULL;
	JsonObject  *process = NULL;
	JsonObject  *pod = NULL;
	gchar       *str = NULL;
	gsize        str_len = 0;
	GError      *err = NULL;
	const gchar *status;
	gboolean     ret;
	gboolean     result = false;

	if (! (config && created_timestamp)) {
		return false;
	}
	if ( ! config->optarg_container_id) {
		return false;
	}
	if ( ! (*config->optarg_container_id)) {
		return false;
	}
	if ( ! config->bundle_path) {
		return false;
	}
	if ( ! config->state.runtime_path[0]) {
		return false;
	}
	if ( ! config->state.comms_path[0]) {
		return false;
	}
	if ( ! config->state.procsock_path[0]) {
		return false;
	}
	if (! config->vm) {
		return false;
	}

	/* Note that although the proxy object must be allocated, it
	 * may not have had its members set.
	 *
	 * This has to be allowed given the way cc_oci_vm_launch()
	 * works (it creates the state file with "blank" proxy values,
	 * then later recreates it with complete information).
	 */
	if (! config->proxy) {
		return false;
	}

	if (! cc_oci_state_file_get (config)) {
		return false;
	}

	obj = json_object_new ();

	/* Add minimim required elements */
	json_object_set_string_member (obj, "ociVersion",
			CC_OCI_SUPPORTED_SPEC_VERSION);

	json_object_set_string_member (obj, "id",
			config->optarg_container_id);

	json_object_set_int_member (obj, "pid",
			(unsigned)config->state.workload_pid);

	json_object_set_string_member (obj, "bundlePath",
			config->bundle_path);

	/* Add runtime-specific elements */
	json_object_set_string_member (obj, "commsPath",
			config->state.comms_path);

	json_object_set_string_member (obj, "processPath",
			config->state.procsock_path);

	status = cc_oci_status_get (config);
	if (! status) {
		goto out;
	}

	json_object_set_string_member (obj, "status", status);

	json_object_set_string_member (obj, "created", created_timestamp);

	/* Add an array of mounts to allow "delete" to unmount these
	 * resources later.
	 */
	mounts = cc_oci_mounts_to_json (config);
	if (! mounts) {
		goto out;
	}

	json_object_set_array_member (obj, "mounts", mounts);

	/* Add an array of namespaces to allow join to them and
	 * umount or clear all resources
	 */
	namespaces = cc_oci_ns_to_json(config);
	if (! namespaces) {
		goto out;
	}

	json_object_set_array_member (obj, "namespaces", namespaces);

	/* Add an process object to allow "start" command  what workload
	 * will be used
	 */
	process = cc_oci_process_to_json(&config->oci.process);
	if (! process) {
		g_critical ("failed to create state file, no process information");
		goto out;
	}

	json_object_set_object_member (obj, "process", process);

	if (config->console) {
		console = json_object_new ();
		json_object_set_string_member (console, "path",
				config->console);
	}

	json_object_set_object_member (obj, "console", console);

	/* Add an object containing hypervisor details */
	vm = json_object_new ();

	json_object_set_int_member (vm, "pid",
			(unsigned)config->vm->pid);

	json_object_set_string_member (vm, "hypervisor_path",
			config->vm->hypervisor_path);

	json_object_set_string_member (vm, "image_path",
			config->vm->image_path);

	json_object_set_string_member (vm, "kernel_path",
			config->vm->kernel_path);

	json_object_set_string_member (vm, "workload_path",
			config->vm->workload_path);

	/* this element must be set, so specify a blank string if there
	 * really are no parameters.
	 */
	json_object_set_string_member (vm, "kernel_params",
			config->vm->kernel_params
			? config->vm->kernel_params : "");

	json_object_set_object_member (obj, "vm", vm);

	/* Add an object containing proxy details */
	proxy = json_object_new ();

	json_object_set_string_member (proxy, "ctlSocket",
			config->proxy->agent_ctl_socket ?
			config->proxy->agent_ctl_socket : "");

	json_object_set_string_member (proxy, "ioSocket",
			config->proxy->agent_tty_socket ?
			config->proxy->agent_tty_socket : "");

	json_object_set_string_member (proxy, "consoleSocket",
			config->proxy->vm_console_socket ?
			config->proxy->vm_console_socket : "");

	json_object_set_object_member (obj, "proxy", proxy);

	if (config->pod != NULL) {
		/* Add an object containing CRI-O/ocid details */
		pod = json_object_new ();

		json_object_set_boolean_member (pod, "sandbox",
						config->pod->sandbox);

		json_object_set_string_member (pod, "sandbox_name",
					       config->pod->sandbox_name ?
					       config->pod->sandbox_name : "");

		json_object_set_object_member (obj, "pod", pod);
	}

	if (config->oci.annotations) {
		/* Add an object containing annotations */
		annotation_obj = cc_oci_annotations_to_json(config);

		json_object_set_object_member(obj, "annotations", annotation_obj);
	}

	/* convert JSON to string */
	str = cc_oci_json_obj_to_string (obj, true, &str_len);
	if (! str) {
		goto out;
	}

	/* Create state file */
	ret = g_file_set_contents (config->state.state_file_path,
			str, (gssize)str_len, &err);
	if (ret) {
		result = true;
	} else {
		g_critical ("failed to create state file %s: %s",
				config->state.state_file_path, err->message);
		g_error_free (err);
	}

	g_debug ("created state file %s", config->state.state_file_path);

out:
	if (obj) {
		json_object_unref (obj);
	}
	g_free_if_set (str);

	return result;
}

/*!
 * Delete the state file for the specified \p config.
 *
 * \param config \ref cc_oci_config.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_state_file_delete (const struct cc_oci_config *config)
{
	g_assert (config);
	g_assert (config->state.state_file_path[0]);

	g_debug ("deleting state file %s", config->state.state_file_path);

	return g_unlink (config->state.state_file_path) == 0;
}

/**
 * Convert a \ref oci_status into a human-readable string.
 *
 * \param status \ref oci_status.
 *
 * \return String representation of \ref oci_status on success,
 * else \c NULL.
 */
const char *
cc_oci_status_to_str (enum oci_status status)
{
	struct cc_oci_map *p;

	for (p = oci_status_map; p && p->name; p++) {
		if (p->num == status) {
			return p->name;
		}
	}

	return NULL;
}

/**
 * Calculate length of longest status value.
 *
 * \return Length in bytes.
 */
int
cc_oci_status_length (void)
{
	struct cc_oci_map  *p;
	int                  max = 0;

	for (p = oci_status_map; p && p->name; p++) {
		int len = (int)g_utf8_strlen (p->name, LINE_MAX);

		max = CC_OCI_MAX (len, max);
	}

	return max;
}

/**
 * Convert a human-readable string state into a \ref oci_status_map.
 *
 * \param str String to convert to a \ref oci_status_map.
 *
 * \return Valid \ref oci_status value, or \ref OCI_STATUS_INVALID on
 * error.
 */
enum oci_status
cc_oci_str_to_status (const char *str)
{
	struct cc_oci_map  *p;

	for (p = oci_status_map; p && p->name; p++) {
		if (! g_strcmp0 (str, p->name)) {
			return p->num;
		}
	}

	return OCI_STATUS_INVALID;
}
