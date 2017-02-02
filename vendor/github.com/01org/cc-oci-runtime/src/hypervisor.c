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
#include <sys/types.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdio.h>
#include <string.h>
#include <errno.h>

#include <glib.h>
#include <uuid/uuid.h>

#include "oci.h"
#include "util.h"
#include "hypervisor.h"
#include "common.h"

/** Length of an ASCII-formatted UUID */
#define UUID_MAX 37

/* Values passed in from automake.
 *
 * XXX: They are assigned to variables to allow the tests
 * to modify the values.
 */
private gchar *sysconfdir = SYSCONFDIR;
private gchar *defaultsdir = DEFAULTSDIR;

static gchar *
cc_oci_expand_net_cmdline(struct cc_oci_config *config) {
	/* www.kernel.org/doc/Documentation/filesystems/nfs/nfsroot.txt
        * ip=<client-ip>:<server-ip>:<gw-ip>:<netmask>:<hostname>:
         * <device>:<autoconf>:<dns0-ip>:<dns1-ip>
	 */

	if (! config) {
		return NULL;
	}

	if (! config->net.hostname) {
		return NULL;
	}

	return ( g_strdup_printf("ip=::::::%s::off::",
		config->net.hostname));
}

#define QEMU_FMT_NETDEV "tap,ifname=%s,script=no,downscript=no,id=%s,vhost=on"

static gchar *
cc_oci_expand_netdev_cmdline(struct cc_oci_config *config, guint index) {
	struct cc_oci_net_if_cfg *if_cfg = NULL;

	if_cfg = (struct cc_oci_net_if_cfg *)
		g_slist_nth_data(config->net.interfaces, index);

	if (if_cfg == NULL) {
		goto out;
	}


	return g_strdup_printf(QEMU_FMT_NETDEV,
		if_cfg->tap_device,
		if_cfg->tap_device);

out:
	return g_strdup("");
}

/* "pcie.0" is the child pci bus available for device "pci-lite-host".
 * Use a pci slot available on that bus after adding an offset to take 
 * into account busy slots and the slots used earlier in our qemu options.
 */
#define QEMU_FMT_DEVICE "driver=virtio-net-pci,bus=/pci-lite-host/pcie.0,addr=%x,netdev=%s"
#define QEMU_FMT_DEVICE_MAC QEMU_FMT_DEVICE ",mac=%s"

static gchar *
cc_oci_expand_net_device_cmdline(struct cc_oci_config *config, guint index) {
	struct cc_oci_net_if_cfg *if_cfg = NULL;

	if_cfg = (struct cc_oci_net_if_cfg *)
		g_slist_nth_data(config->net.interfaces, index);

	if (if_cfg == NULL) {
		goto out;
	}

	g_debug("PCI Offset used for network: %d", PCI_OFFSET);

	if ( if_cfg->mac_address == NULL ) {
		return g_strdup_printf(QEMU_FMT_DEVICE,
			index + PCI_OFFSET,
			if_cfg->tap_device);
	} else {
		return g_strdup_printf(QEMU_FMT_DEVICE_MAC,
			index + PCI_OFFSET,
			if_cfg->tap_device,
			if_cfg->mac_address);
	}

out:
	return g_strdup("");
}

/*!
 * Append qemu options for networking
 *
 * \param config \ref cc_oci_config.
 * \param additional_args Array that will be appended
 */
static void
cc_oci_append_network_args(struct cc_oci_config *config, 
			GPtrArray *additional_args)
{
	gchar *netdev_params = NULL;
	gchar *net_device_params = NULL;

	if (! (config && additional_args)) {
		return;
	}

	if ( config->net.interfaces == NULL ) {
		g_ptr_array_add(additional_args, g_strdup("-net\nnone\n"));
	} else {
		for (guint index = 0; index < g_slist_length(config->net.interfaces); index++) {
			netdev_params = cc_oci_expand_netdev_cmdline(config, index);
			net_device_params = cc_oci_expand_net_device_cmdline(config, index);

			g_ptr_array_add(additional_args, g_strdup("-netdev"));
			g_ptr_array_add(additional_args, netdev_params);
			g_ptr_array_add(additional_args, g_strdup("-device"));
			g_ptr_array_add(additional_args, net_device_params);
		}
        }
}

/*!
 * Replace any special tokens found in \p args with their expanded
 * values.
 *
 * \param config \ref cc_oci_config.
 * \param[in, out] args Command-line to expand.
 *
 * \warning this is not very efficient.
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_expand_cmdline (struct cc_oci_config *config,
		gchar **args)
{
	struct stat       st;
	gchar           **arg;
	gchar            *bytes = NULL;
	gchar            *console_device = NULL;
	gchar            *workload_dir;
	gchar		 *hypervisor_console = NULL;
	g_autofree gchar *procsock_device = NULL;

	gboolean          ret = false;
	gint              count;
	uuid_t            uuid;
	/* uuid pattern */
	const char        uuid_pattern[UUID_MAX] = "00000000-0000-0000-0000-000000000000";
	char              uuid_str[UUID_MAX] = { 0 };
	gint              uuid_index = 0;

	gchar            *kernel_net_params = NULL;
	struct cc_proxy  *proxy;

	if (! (config && args)) {
		return false;
	}

	if (! config->vm) {
		g_critical ("No vm configuration");
		goto out;
	}

	if (! config->bundle_path) {
		g_critical ("No bundle path");
		goto out;
	}

	if (! config->proxy) {
		g_critical ("No proxy");
		goto out;
	}

	/* We're about to launch the hypervisor so validate paths.*/

	workload_dir = cc_oci_get_workload_dir(config);
	if (! workload_dir) {
		g_critical ("No workload");
		goto out;
	}

	if ((!config->vm->image_path[0])
		|| stat (config->vm->image_path, &st) < 0) {
		g_critical ("image file: %s does not exist",
			    config->vm->image_path);
		return false;
	}

	if (!(config->vm->kernel_path[0]
		&& g_file_test (config->vm->kernel_path, G_FILE_TEST_EXISTS))) {
		g_critical ("kernel image: %s does not exist",
			    config->vm->kernel_path);
		return false;
	}

	if (!(workload_dir[0]
		&& g_file_test (workload_dir, G_FILE_TEST_IS_DIR))) {
		g_critical ("workload directory: %s does not exist",
			    workload_dir);
		return false;
	}

	uuid_generate_random(uuid);
	for(size_t i=0; i<sizeof(uuid_t) && uuid_index < sizeof(uuid_pattern); ++i) {
		/* hex to char */
		uuid_index += g_snprintf(uuid_str+uuid_index,
		                  sizeof(uuid_pattern)-(gulong)uuid_index,
		                  "%02x", uuid[i]);

		/* copy separator '-' */
		if (uuid_pattern[uuid_index] == '-') {
			uuid_index += g_snprintf(uuid_str+uuid_index,
			                  sizeof(uuid_pattern)-(gulong)uuid_index, "-");
		}
	}

	bytes = g_strdup_printf ("%lu", (unsigned long int)st.st_size);

	hypervisor_console = g_build_path ("/", config->state.runtime_path,
			CC_OCI_CONSOLE_SOCKET, NULL);

	console_device = g_strdup_printf (
			"socket,path=%s,server,nowait,id=charconsole0,signal=off",
			hypervisor_console);

	procsock_device = g_strdup_printf ("socket,id=procsock,path=%s,server,nowait", config->state.procsock_path);

	proxy = config->proxy;

	proxy->vm_console_socket = hypervisor_console;

	proxy->agent_ctl_socket = g_build_path ("/", config->state.runtime_path,
			CC_OCI_AGENT_CTL_SOCKET, NULL);

	g_debug("guest agent ctl socket: %s", proxy->agent_ctl_socket);

	proxy->agent_tty_socket = g_build_path("/", config->state.runtime_path,
			CC_OCI_AGENT_TTY_SOCKET, NULL);

	g_debug("guest agent tty socket: %s", proxy->agent_tty_socket);

	kernel_net_params = cc_oci_expand_net_cmdline(config);

	/* Note: @NETDEV@: For multiple network we need to have a way to append
	 * args to the hypervisor command line vs substitution
	 */
	struct special_tag {
		const gchar* name;
		const gchar* value;
	} special_tags[] = {
		{ "@WORKLOAD_DIR@"      , workload_dir               },
		{ "@KERNEL@"            , config->vm->kernel_path    },
		{ "@KERNEL_PARAMS@"     , config->vm->kernel_params  },
		{ "@KERNEL_NET_PARAMS@" , kernel_net_params          },
		{ "@IMAGE@"             , config->vm->image_path     },
		{ "@SIZE@"              , bytes                      },
		{ "@COMMS_SOCKET@"      , config->state.comms_path   },
		{ "@PROCESS_SOCKET@"    , procsock_device            },
		{ "@CONSOLE_DEVICE@"    , console_device             },
		{ "@NAME@"              , g_strrstr(uuid_str, "-")+1 },
		{ "@UUID@"              , uuid_str                   },
		{ "@AGENT_CTL_SOCKET@"  , proxy->agent_ctl_socket    },
		{ "@AGENT_TTY_SOCKET@"  , proxy->agent_tty_socket    },
		{ NULL }
	};

	for (arg = args, count = 0; arg && *arg; arg++, count++) {
		if (! count) {
			/* command must be the first entry */
			if (! g_path_is_absolute (*arg)) {
				gchar *cmd = g_find_program_in_path (*arg);

				if (cmd) {
					g_free (*arg);
					*arg = cmd;
				}
			}
		}

		/* when first character is '#' line is a comment and must be ignored */
		if (**arg == '#') {
			g_strlcpy(*arg, "\0", LINE_MAX);
			continue;
		}

		/* looking for '#' */
		gchar* ptr = g_strstr_len(*arg, LINE_MAX, "#");
		while (ptr) {
			/* if '[:space:]#' then replace '#' with '\0' (EOL) */
			if (g_ascii_isspace(*(ptr-1))) {
				g_strlcpy(ptr, "\0", LINE_MAX);
				break;
			}
			ptr = g_strstr_len(ptr+1, LINE_MAX, "#");
		}

		for (struct special_tag* tag=special_tags; tag && tag->name; tag++) {
			if (! cc_oci_replace_string(arg, tag->name, tag->value)) {
				goto out;
			}
		}
	}

	ret = true;

out:
	g_free_if_set (bytes);
	g_free_if_set (console_device);
	g_free_if_set (kernel_net_params);

	return ret;
}

/*!
 * Determine the full path to the \ref CC_OCI_HYPERVISOR_CMDLINE_FILE
 * file.
 * Priority order to get file path : bundle dir, sysconfdir , defaultsdir
 *
 * \param config \ref cc_oci_config.
 *
 * \return Newly-allocated string on success, else \c NULL.
 */
private gchar *
cc_oci_vm_args_file_path (const struct cc_oci_config *config)
{
	gchar *args_file = NULL;

	if (! config) {
		return NULL;
	}

	if (! config->bundle_path) {
		return NULL;
	}

	args_file = cc_oci_get_bundlepath_file (config->bundle_path,
			CC_OCI_HYPERVISOR_CMDLINE_FILE);
	if (! args_file) {
		return NULL;
	}

	if (g_file_test (args_file, G_FILE_TEST_EXISTS)) {
		goto out;
	}

	g_free_if_set (args_file);

	/* Try sysconfdir if bundle file does not exist */
	args_file = g_build_path ("/", sysconfdir,
			CC_OCI_HYPERVISOR_CMDLINE_FILE, NULL);

	if (g_file_test (args_file, G_FILE_TEST_EXISTS)) {
		goto out;
	}

	g_free_if_set (args_file);

	/* Finally, try stateless dir */
	args_file = g_build_path ("/", defaultsdir,
			CC_OCI_HYPERVISOR_CMDLINE_FILE, NULL);

	if (g_file_test (args_file, G_FILE_TEST_EXISTS)) {
		goto out;
	}

	g_free_if_set (args_file);

	/* no file found, so give up */
	args_file = NULL;

out:
	g_debug ("using %s", args_file);
	return args_file;
}

/*!
 * Generate the unexpanded list of hypervisor arguments to use.
 *
 * \param config \ref cc_oci_config.
 * \param[out] args Command-line to expand.
 * \param hypervisor_extra_args Additional args to be appended
 *
 * \return \c true on success, else \c false.
 */
gboolean
cc_oci_vm_args_get (struct cc_oci_config *config,
		gchar ***args,
		GPtrArray *hypervisor_extra_args)
{
	gboolean  ret;
	gchar    *args_file = NULL;
	guint     line_count = 0;
	gchar   **arg;
	gchar   **new_args;
	guint       extra_args_len = 0;

	if (! (config && args)) {
		return false;
	}

	args_file = cc_oci_vm_args_file_path (config);
	if (! args_file) {
		g_critical("File %s not found",
				CC_OCI_HYPERVISOR_CMDLINE_FILE);
	}

	ret = cc_oci_file_to_strv (args_file, args);
	if (! ret) {
		goto out;
	}

	ret = cc_oci_expand_cmdline (config, *args);
	if (! ret) {
		goto out;
	}

	/* count non-empty lines */
	for (arg = *args; arg && *arg; arg++) {
		if (**arg != '\0') {
			line_count++;
		}
	}

	if (hypervisor_extra_args) {
		extra_args_len = hypervisor_extra_args->len;
	}

	new_args = g_malloc0(sizeof(gchar*) * (line_count + extra_args_len + 1));

	/* copy non-empty lines */
	for (arg = *args, line_count = 0; arg && *arg; arg++) {
		/* *do not* add empty lines */
		if (**arg != '\0') {
			/* container fails if arg contains spaces */
			g_strstrip(*arg);
			new_args[line_count] = *arg;
			line_count++;
		} else {
			/* free empty lines */
			g_free(*arg);
		}
	}

	/*  append additional args array */
	for (int i = 0; i < extra_args_len; i++) {
		const gchar* arg = g_ptr_array_index(hypervisor_extra_args, i);
		if (arg != '\0') {
			new_args[line_count++] = g_strstrip(g_strdup(arg));
		}
	}

	/* only free pointer to gchar* */
	g_free(*args);

	/* copy new args */
	*args = new_args;

	ret = true;
out:
	g_free_if_set (args_file);
	return ret;
}

/*!
 * Populate array that will be appended to hypervisor command line.
 *
 * \param config \ref cc_oci_config.
 * \param additional_args Array to append
 */
void
cc_oci_populate_extra_args(struct cc_oci_config *config ,
		GPtrArray *additional_args)
{
	if (! (config && additional_args)) {
		return;
	}

	/* Add args to be appended here.*/
	//g_ptr_array_add(additional_args, g_strdup("-device testdevice"));

	cc_oci_append_network_args(config, additional_args);

	return;
}
