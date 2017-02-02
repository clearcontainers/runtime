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
	#Start use Clear Containers
	check_ccontainers
	#Default timeout for cor commands
	COR_TIMEOUT=5
	container_id="tests_id"
}

function teardown() {
	cleanup_common
}

@test "kill without container id" {
	run $COR kill
	[ "$status" -ne 0 ]
	[[ "${output}" == "Usage: kill <container-id> [<signal>]" ]]
}

@test "kill with invalid container id" {
	run $COR kill FOO
	[ "$status" -ne 0 ]
	[[ "${output}" =~ "failed to parse json file:" ]]
}

@test "start then kill (implicit signal)" {
	workload_cmd "sh"

	cmd="$COR create --console --bundle $BUNDLE_DIR $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "created"

	# 'start' runs in background since it will
	# update the state file once shim ends
	cmd="$COR start $container_id &"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "running"

	cmd="$COR kill $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "killed"

	cmd="$COR delete $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	verify_runtime_dirs "$container_id" "deleted"
}

@test "start then kill (short symbolic signal)" {
	workload_cmd "sh"

	cmd="$COR create  --console --bundle $BUNDLE_DIR $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "created"

	# 'start' runs in background since it will
	# update the state file once shim ends
	cmd="$COR start $container_id &"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "running"

	# specify invalid signal name
	cmd="$COR kill $container_id FOOBAR"
	run_cmd "$cmd" "1" "$COR_TIMEOUT"

	cmd="$COR kill $container_id TERM"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "killed"

	cmd="$COR delete $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	verify_runtime_dirs "$container_id" "deleted"
}

@test "start then kill (full symbolic signal)" {
	workload_cmd "sh"

	cmd="$COR create  --console --bundle $BUNDLE_DIR $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "created"

	# 'start' runs in background since it will
	# update the state file once shim ends
	cmd="$COR start $container_id &"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "running"

	# specify invalid signal name
	cmd="$COR kill $container_id SIGFOOBAR"
	run_cmd "$cmd" "1" "$COR_TIMEOUT"

	cmd="$COR kill $container_id SIGTERM"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "killed"

	cmd="$COR delete $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	verify_runtime_dirs "$container_id" "deleted"
}

@test "start then kill (numeric signal)" {
	workload_cmd "sh"

	cmd="$COR create  --console --bundle $BUNDLE_DIR $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "created"

	# 'start' runs in background since it will
	# update the state file once shim ends
	cmd="$COR start $container_id &"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "running"

	# specify invalid signal number
	cmd="$COR kill $container_id 123456"
	run_cmd "$cmd" "1" "$COR_TIMEOUT"

	cmd="$COR kill $container_id 15"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "killed"

	cmd="$COR delete $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	verify_runtime_dirs "$container_id" "deleted"
}
