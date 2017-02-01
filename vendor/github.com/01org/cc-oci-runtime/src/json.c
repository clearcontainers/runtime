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

#include <stdbool.h>
#include <stdio.h>

#include <json-glib/json-glib.h>

/**
 * Buffer which must be large enough to hold the string representation
 * of any JSON node type (\c GType).
 */
#define NODE_BUF_SIZE 64

/*!
 * Convert the specified \c JsonNode into a string.
 *
 * \param node \c JsonNode.
 * \return Newly-allocated string on success, else \c NULL.
 */
static gchar *
cc_oci_json_string (JsonNode* node) {
	gchar buffer[NODE_BUF_SIZE];
	GType valueType = json_node_get_value_type(node);

	switch (valueType) {
	case G_TYPE_STRING:
		return json_node_dup_string(node);

	case G_TYPE_DOUBLE:
	case G_TYPE_FLOAT:
		g_snprintf(buffer, NODE_BUF_SIZE, "%f", json_node_get_double(node));
		break;

	case G_TYPE_INT:
	case G_TYPE_INT64:
		g_snprintf(buffer, NODE_BUF_SIZE, "%ld", json_node_get_int(node));
		break;

	case G_TYPE_BOOLEAN:
		if (json_node_get_boolean(node)) {
			g_snprintf(buffer, NODE_BUF_SIZE, "%s", "true");
		} else {
			g_snprintf(buffer, NODE_BUF_SIZE, "%s", "false");
		}
		break;

	default:
		g_snprintf(buffer, NODE_BUF_SIZE, "%s", "Unknown type");
		break;
	}

	return g_strdup(buffer);
}

/*!
 * Recursive function that handles converging \c JsonNode's to \c
 * GNode's.
 *
 * \param root \c Root JsonNode to convert.
 * \param node \c GNode.
 * \param parsing_array \c true if handling an array, else \c false.
 */
static void
cc_oci_json_parse_aux(JsonNode* root, GNode* node, bool parsing_array) {
	guint i;

	g_assert (root);
	g_assert (node);

	if (JSON_NODE_TYPE(root) == JSON_NODE_OBJECT) {
		JsonObject *object = json_node_get_object(root);

		if (object) {
			guint j;
			guint size;
			GList* keys, *key = NULL;
			GList* values, *value = NULL;

			size = json_object_get_size(object);
			keys = json_object_get_members(object);
			values = json_object_get_values(object);
			node = g_node_append(node, g_node_new(NULL));

			for (j = 0, key = keys, value = values; j < size; j++) {
				if (key) {
					node = g_node_append(node->parent, g_node_new(g_strdup(key->data)));
				}
				if (value) {
					cc_oci_json_parse_aux(value->data, node, false);
				}

				key = g_list_next(key);
				value = g_list_next(value);
			}

			if (keys) {
				g_list_free(keys);
			}
			if (values) {
				g_list_free(values);
			}
		}
	} else if (JSON_NODE_TYPE(root) == JSON_NODE_ARRAY) {
		JsonArray* array = json_node_get_array(root);
		guint array_size = json_array_get_length (array);

		for (i = 0; i < array_size; i++) {
			JsonNode *array_element = json_array_get_element(array, i);
			cc_oci_json_parse_aux(array_element, node, true);
		}
	} else if (JSON_NODE_TYPE(root) == JSON_NODE_VALUE) {
		node = g_node_append(node, g_node_new(cc_oci_json_string(root)));

		if (parsing_array) {
			node = g_node_append(node, g_node_new(NULL));
		}
	}
}

/*!
 * Convert a JSON file into a tree of nodes.
 *
 * \param[out] node Tree representation of \p filename.
 * \param filename Absolute path to JSON file to parse.
 *
 * \return \c true on success, else \c false.
 */
bool
cc_oci_json_parse (GNode** node, const gchar* filename) {
	bool result = false;
	GError* error = NULL;
	JsonParser* parser = NULL;
	JsonNode *root = NULL;

	if ((!node) || (!filename) || (!(*filename))) {
		return false;
	}

	parser = json_parser_new();
	if (! json_parser_load_from_file(parser, filename, &error)) {
		g_debug("unable to parse '%s'", filename);
		if (error) {
			g_debug("Error parsing '%s': %s", filename, error->message);
			g_error_free(error);
		}
		goto exit;
	}

	root = json_parser_get_root (parser);
	if (! root) {
		goto exit;
	}

	*node = g_node_new(g_strdup(filename));
	cc_oci_json_parse_aux(root, *node, false);

	result = true;

exit:
	g_object_unref(parser);
	return result;
}
