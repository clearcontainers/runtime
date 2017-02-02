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
	clean_docker_ps
	runtime_docker
}

@test "modifying a container with exec" {
	$DOCKER_EXE run --name containertest -d ubuntu bash -c "sleep 50"
	$DOCKER_EXE ps -a | grep "sleep 50"
	$DOCKER_EXE exec -d containertest bash -c "echo 'hello world' > file"
	$DOCKER_EXE exec containertest bash -c "cat /file"
}

@test "exec a container with privileges" {
	$DOCKER_EXE run -d --name containertest ubuntu bash -c "sleep 30"
	$DOCKER_EXE exec -i --privileged containertest bash -c "mount -t tmpfs none /mnt"
	$DOCKER_EXE exec containertest bash -c "df -h | grep "/mnt""
}

@test "copying file from host to container using exec" {
	content="hello world"
	$DOCKER_EXE run --name containertest -d ubuntu bash -c "sleep 30"
	echo $content | $DOCKER_EXE exec -i containertest bash -c "cat > /home/file.txt"
	$DOCKER_EXE exec -i containertest bash -c "cat /home/file.txt" | grep "$content"
}

@test "stdout forwarded using exec" {
	$DOCKER_EXE run --name containertest -d ubuntu bash -c "sleep 30"
	$DOCKER_EXE exec -ti containertest ls /etc/resolv.conf 2>/dev/null | grep "/etc/resolv.conf" 
}

@test "stderr forwarded using exec" {
        $DOCKER_EXE run --name containertest -d ubuntu bash -c "sleep 30"
        if $DOCKER_EXE exec containertest ls /etc/foo >/dev/null; then false; else true; fi	
}

@test "check exit code using exec" {
	$DOCKER_EXE run --name containertest -d ubuntu bash -c "sleep 30"
	run $DOCKER_EXE exec containertest bash -c "exit 42"
	[ "$status" -eq 42 ]
}
