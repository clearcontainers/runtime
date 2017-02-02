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
	volName='volume1'
}

@test "Volume - create" {
	$DOCKER_EXE volume create --name "$volName"
	$DOCKER_EXE volume ls | grep "$volName"
}

@test "Volume - inspect" {
	$DOCKER_EXE volume inspect --format '{{ .Mountpoint }}' "$volName"
}

@test "Volume - use volume in a container" {
	testFile='hello_world'
	containerPath='/attached_vol'
	run bash -c "$DOCKER_EXE run -i -v $volName:$containerPath busybox touch $containerPath/$testFile"
	[ $status -eq 0 ]
	run bash -c "$DOCKER_EXE run -i -v $volName:$containerPath busybox ls $containerPath | grep $testFile"
	$DOCKER_EXE rm $($DOCKER_EXE ps -qa)
	[ $status -eq 0 ]
}

@test "Volume - bind-mount a directory" {
	dir_path="$(pwd)/sharedSpace"
	test_file="foo"
	containerPath="/root/sharedSpace"
	mkdir "$dir_path"
	touch "$dir_path/$test_file"
	run bash -c "$DOCKER_EXE run -i -v $dir_path:$containerPath busybox ls $containerPath | grep $test_file"
	rm -r "$dir_path"
	$DOCKER_EXE rm $($DOCKER_EXE ps -qa)
	[ $status -eq 0 ]

}

@test "Volume - bind-mount a single file" {
	test_file="$(pwd)/foo"
	file_content="bar"
	echo "$file_content" >  "$test_file"
	containerPath="/root/foo"
	run bash -c "$DOCKER_EXE run -i -v $test_file:$containerPath busybox cat $containerPath | grep $file_content"
	rm "$test_file"
	$DOCKER_EXE rm $($DOCKER_EXE ps -qa)
	[ $status -eq 0 ]
}

@test "Delete the volume" {
	$DOCKER_EXE volume rm "$volName"
}
