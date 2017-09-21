#!/bin/bash
#
# Copyright (c) 2017 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Description: This script will collect together all information about the
#   environment and display to stdout. Redirect to a file and attach to a bug
#   report to help developers diagnose problems.

script_name=${0##*/}
runtime=$(which "cc-runtime")

die()
{
	local msg="$*"
	echo >&2 "ERROR: $script_name: $msg"
	exit 1
}

msg()
{
	local msg="$*"
	echo "$msg"
}

heading()
{
	local name="$*"
	echo -e "\n# $name\n"
}

have_cmd()
{
	local cmd="$1"

	command -v "$cmd" &>/dev/null
	local ret=$?

	if [ $ret -eq 0 ]; then
		msg "Have \`$cmd\`"
	else
		msg "No \`$cmd\`"
	fi

	[ $ret -eq 0 ]
}

show_quoted_text()
{
	local text="$*"

	echo "\`\`\`"
	echo "$text"
	echo "\`\`\`"
}

run_cmd_and_show_quoted_output()
{
	local cmd="$*"

	msg "Output of \`$cmd\`:"
	output=$(eval "$cmd" 2>&1)
	show_quoted_text "$output"
}

show_runtime_configs()
{
	local configs config

	heading "Runtime config files"
	
	configs=$($runtime --cc-show-default-config-paths)
	if [ $? -ne 0 ]; then
		version=$($runtime --version|tr '\n' ' ')
		die "failed to check config files - runtime is probably too old ($version)"
	fi

	msg "Runtime default config files"
	show_quoted_text "$configs"

	# add in the standard defaults for good measure "just in case"
	configs+=" /etc/clear-containers/configuration.toml"
	configs+=" /usr/share/defaults/clear-containers/configuration.toml"

	# create a unique list of config files
	configs=$(echo $configs|tr ' ' '\n'|sort -u)

	msg "Runtime config file contents"

	for config in $configs; do
		if [ -e "$config" ]; then
			run_cmd_and_show_quoted_output "cat \"$config\""
		else
			msg "Config file \`$config\` not found"
		fi
	done
}

show_runtime_log_details()
{
	heading "Runtime logs"

	local global_log=$($runtime cc-env | egrep "\<GlobalLogPath\>"|cut -d= -f2-|sed 's/^ *//g'|tr -d '"')
	if [ -n "$global_log" ]; then
		if [ -e "$global_log" ]; then
			local pattern="("
			pattern+="\<warn\>"
			pattern+="|\<error\>"
			pattern+="|\<fail\>"
			pattern+="|\<fatal\>"
			pattern+="|\<impossible\>"
			pattern+="|\<missing\>"
			pattern+="|\<does.*not.*exist\>"
			pattern+="|\<not.*found\>"
			pattern+="|\<no.*such.*file\>"
			pattern+="|\<cannot\>"
			pattern+=")"

			# Note: output limited to most recent issues
			local problems=$(egrep -i "$pattern" "$global_log"|tail -50)

			if [ -n "$problems" ]; then
				msg "Recent problems found in global log \`$global_log\`:"
				show_quoted_text "$problems"
			else
				msg "No recent problems found in global log \`$global_log\`:"
			fi
		else
			msg "Global log \`$global_log\` does not exist"
		fi

	else
		msg "Global log not enabled"
	fi
}

show_package_versions()
{

	heading "Packages"

	local pattern="("

	# core components
	pattern+="cc-proxy"
	pattern+="|cc-runtime"
	pattern+="|cc-shim"

	# assets
	pattern+="|clear-containers-image"
	pattern+="|linux-container"

	# optimised hypervisor
	pattern+="|qemu-lite"

	# default distro hypervisor
	pattern+="|qemu-system-x86"

	# CC 2.x runtime. This shouldn't be installed but let's check anyway
	pattern+="|cc-oci-runtime"

	pattern+=")"

	if have_cmd "dpkg"; then
		run_cmd_and_show_quoted_output "dpkg -l|egrep \"$pattern\""
	fi

	if have_cmd "rpm"; then
		run_cmd_and_show_quoted_output "rpm -qa|egrep \"$pattern\""
	fi
}

show_container_mgr_details()
{
	heading "Container manager details"

	if have_cmd "docker"; then
		run_cmd_and_show_quoted_output "docker info"
	fi

	if have_cmd "kubectl"; then
		run_cmd_and_show_quoted_output "kubectl config view"
	fi
}

show_meta()
{
	heading "Meta"

	date=$(date '+%Y-%m-%d.%H:%M:%S.%N')
	msg "Running \`$script_name\` at \`$date\`"
}

main()
{
	[ $(id -u) -eq 0 ] || die "Need to run as root"

	show_meta

	msg "Runtime is \`$runtime\`"

	cmd="cc-env"
	heading "\`$cmd\`"
	run_cmd_and_show_quoted_output "$runtime $cmd"

	show_runtime_configs
	show_runtime_log_details
	show_container_mgr_details
	show_package_versions
}

main
