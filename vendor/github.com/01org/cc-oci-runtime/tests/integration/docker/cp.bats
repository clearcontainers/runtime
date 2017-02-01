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

#Based on docker commands

SRC="${BATS_TEST_DIRNAME}/../../lib/"

setup() {
	source $SRC/test-common.bash
	clean_docker_ps
	runtime_docker
}

@test "Check mounted files at /etc after a docker cp" {
	content="test"
	testfile=$(mktemp --tmpdir="$BATS_TMPDIR" --suffix=-cor-test)
	$DOCKER_EXE run --name containertest -tid ubuntu bash
	echo $content > $testfile
	$DOCKER_EXE cp $testfile containertest:/root/
	$DOCKER_EXE exec -i containertest bash -c "ls /root/$(basename $testfile)"
	$DOCKER_EXE exec -i containertest bash -c "[ -s /etc/resolv.conf ]"
	rm -f $testfile
}
