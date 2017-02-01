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

# Based on docker commands

SRC="${BATS_TEST_DIRNAME}/../../lib/"
term_var="TERM=.*"
tty_dev="/dev/pts/.*"

setup() {
	source $SRC/test-common.bash
	clean_docker_ps
	runtime_docker
}

@test "TERM env variable is set when allocating a tty" {
	$DOCKER_EXE run -t ubuntu env | grep -q "$term_var"
}

@test "TERM env variable is not set when not allocating a tty" {
	run bash -c "$DOCKER_EXE run ubuntu env | grep -q $term_var"
	# Expecting RC=1 from the grep command since
	# the TERM env variable should not exist.
	[ "${status}" -eq 1 ]
}

@test "Check that pseudo tty is setup properly when allocating a tty" {
	run $DOCKER_EXE run -ti ubuntu tty
	echo "${output}" | grep "$tty_dev"
}
