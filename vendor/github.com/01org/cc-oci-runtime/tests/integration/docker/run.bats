#!/usr/bin/env bats
# *-*- Mode: sh; sh-basic-offset: 8; indent-tabs-mode: nil -*-*

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

#Based on docker commands

SRC="${BATS_TEST_DIRNAME}/../../lib/"

setup() {
	source $SRC/test-common.bash
	runtime_docker
}

@test "Run shell echo" {
	run $DOCKER_EXE run busybox sh -c "echo Passed"
	[ "${status}" -eq 0 ]
	[[ "${output}" == 'Passed'* ]]
}

@test "stdout using run" {
	run bash -c "$DOCKER_EXE run busybox ls /etc/resolv.conf"
	[[ "${output}" =~ "/etc/resolv.conf" ]]
	run bash -c "$DOCKER_EXE run busybox ls /etc/resolv.conf 1>/dev/null"
	[ -z "${output}" ]
}

@test "stderr using run" {
	run bash -c "$DOCKER_EXE run busybox ls /etc/foo"
	[[ "${output}" =~ "ls: /etc/foo: No such file or directory" ]]
	run bash -c "$DOCKER_EXE run busybox ls /etc/foo 2>/dev/null"
	[ -z "${output}" ]
}

@test "stdin from pipe" {
	run bash -c "echo hello | $DOCKER_EXE run -i busybox"
	echo status: "${status}"
	echo output: "${output}"
	[[ "${status}" == 127 ]]
	[[ "${output}" =~ "sh: hello: not found" ]]
}
