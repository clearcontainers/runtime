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

@test "pause without container id" {
	run $COR pause
	echo $output
	[ "$status" -ne 0 ]
	[[ "${output}" == "Usage: pause <container-id>" ]]
}

@test "pause with invalid container id" {
	run $COR pause FOO
	[ "$status" -ne 0 ]
	[[ "${output}" =~ "failed to parse json file:" ]]
}

@test "resume without container id" {
	run $COR resume
	echo $output
	[ "$status" -ne 0 ]
	[[ "${output}" == "Usage: resume <container-id>" ]]
}

@test "resume with invalid container id" {
	run $COR resume FOO
	[ "$status" -ne 0 ]
	[[ "${output}" =~ "failed to parse json file:" ]]
}

@test "start then pause and resume" {
	workload_cmd "sh"

	# 'run' runs in background since it will
	# update the state file once shim ends
	cmd="$COR run --console --bundle $BUNDLE_DIR $container_id &"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "running"

	cmd="$COR pause $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "paused"

	cmd="$COR resume $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "running"

	cmd="$COR kill $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	testcontainer "$container_id" "killed"

	cmd="$COR delete $container_id"
	run_cmd "$cmd" "0" "$COR_TIMEOUT"
	verify_runtime_dirs "$container_id" "deleted"
}
