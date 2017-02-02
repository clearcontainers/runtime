#!/usr/bin/env bats
#  This file is part of cc-oci-runtime.
#
#  Copyright (C) 2016 Intel Corporation
#
#  This program is free software; you can redistribute it and/or
#  modify it under the terms of the GNU General Public License
#  as published by the Free Software Foundation; either version 2
#  of the License, or (at your option) any later version.
#
#  This program is distributed in the hope that it will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#  GNU General Public License for more details.
#
#  You should have received a copy of the GNU General Public License
#  along with this program; if not, write to the Free Software
#  Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
#

load common

function setup() {
	setup_common
	check_ccontainers
	COR_TIMEOUT=5
	container_id="tests_id"
}

function teardown() {
	cleanup_common
}

# Run and verify state output
# @param <container-id>
# @param <status> container state to verify (running, created ...)
function check_state() {
	container_id_state="$1"
	status="$2"
	#COR must store resolved paths to cc components
	cc_kernel_path=$(readlink -e "${CONTAINER_KERNEL}")
	cc_image_path=$(readlink -e "${CONTAINERS_IMG}")
	bundle_path=$(readlink -e "${BUNDLE_DIR}")
	hypervisor_path=$(readlink -e "${HYPERVISOR_PATH}")

	state_cmd="$COR state $container_id_state"
	local output=$(run_cmd "$state_cmd" "0" "$COR_TIMEOUT")
	[ -n "$(echo "$output" | grep -o "\"id\" : \"${container_id_state}\"")" ]
	[ -n "$(echo "$output" | grep -o "\"bundlePath\" : \"${bundle_path}\"")" ]
	[ -n "$(echo "$output" | grep -o "\"status\" : \"${status}\"")" ]
	[ -n "$(echo "$output" | grep -o "\"hypervisor_path\" : \"${hypervisor_path}\"")" ]
	[ -n "$(echo "$output" | grep -o "\"image_path\" : \"${cc_image_path}\"")" ]
	[ -n "$(echo "$output" | grep -o "\"kernel_path\" : \"${cc_kernel_path}\"")" ]
}

@test "start and kill state" {
	workload_cmd "sh"

	cmd="$COR create  --console --bundle $BUNDLE_DIR $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "created"
	check_state "$container_id" "created"

	# 'start' runs in background since it will
	# update the state file once shim ends
	cmd="$COR start $container_id &"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "running"
	check_state "$container_id" "running"

	cmd="$COR kill $container_id 15"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "killed"
	check_state "$container_id" "stopped"

	cmd="$COR delete $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	verify_runtime_dirs "$container_id" "deleted"
}

@test "state not existing container" {

	state_cmd="$COR state $container_id"
	run_cmd "$state_cmd" "1" "$COR_TIMEOUT"
}
