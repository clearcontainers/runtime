#!/usr/bin/env bats
# *-*- Mode: sh; sh-basic-offset: 8; indent-tabs-mode: nil -*-*

#  This file is part of cc-oci-runtime.
#
#  Copyright (C) 2017 Intel Corporation
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

SRC="${BATS_TEST_DIRNAME}/../../lib/"
IMG_NAME="ccbuildtests"


setup() {
	source $SRC/test-common.bash
	runtime_docker
}

teardown () {
	echo "teardown:"
	$DOCKER_EXE rmi $IMG_NAME
}

@test "docker build env vars" {
	var_value="test_env_vars"

	run $DOCKER_EXE build -t "${IMG_NAME}" - <<EOF
	FROM busybox
	ENV VAR "${var_value}"
	RUN sh -c 'env'
EOF
	echo output: "${output}"
	[ "${status}" -eq  0 ]
	echo "${output}" | grep "VAR=${var_value}"

	run $DOCKER_EXE run --rm -ti "${IMG_NAME}" sh -c 'env'
	echo output: "${output}"
	[ "${status}" -eq  0 ]
	echo "${output}" | grep "VAR=${var_value}"
}
