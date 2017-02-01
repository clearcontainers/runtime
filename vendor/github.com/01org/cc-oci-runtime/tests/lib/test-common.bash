#!/bin/bash

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

DOCKER_EXE="docker"
DOCKER_SERVICE="docker-cor"
SCRIPT_PATH=$(dirname $(readlink -f $0))
RESULT_DIR="${SCRIPT_PATH}/../results"

# Cleaning test environment
function clean_docker_ps(){
	"$DOCKER_EXE" ps -q | xargs -r "$DOCKER_EXE" kill
	"$DOCKER_EXE" ps -aq | xargs -r "$DOCKER_EXE" rm -f
}

# Restarting test environment
function start_docker_service(){
	systemctl status "$DOCKER_SERVICE" | grep 'running'
	if [ "$?" -eq 0 ]; then
		systemctl restart "$DOCKER_SERVICE"
	fi
}

# Checking that default runtime is cor
function runtime_docker(){
	default_runtime=`$DOCKER_EXE info 2>/dev/null | grep "^Default Runtime" | cut -d: -f2 | tr -d '[[:space:]]'`
	if [ "$default_runtime" != "cor" ]; then
		die "Tests need to run with COR runtime"
	fi
}

function pull_image(){
	docker images | grep -q "$1"
	if [ $? == 1 ]; then
		echo "Pull $1 image"
		docker pull "$1"
	fi
}

die(){
	msg="$*"
	echo "ERROR: $msg" >&2
	exit 1
}

function backup_old_file(){
	if [ -f "$1" ]; then
		mv "$1" "$1.old"
	fi
}

function write_csv_header(){
	test_file="$1"
	echo "TestName,TestArguments,Value,Platform,OSVersion" > "$test_file"
}

function write_result_to_file(){
	test_name="$1"
	test_args="$2"
	test_data="$3"
	test_file="$4"
	test_platform=$(grep -m1 "model name" /proc/cpuinfo | cut -d: -f2)
	os_id=$(grep "^ID" /usr/lib/os-release | cut -d= -f2)
	os_ver_id=$(grep "^VERSION_ID" /usr/lib/os-release | cut -d= -f2)
	os_ver=$(echo "$os_id-$os_ver_id")
	echo "$test_name,$test_args,$test_data,$test_platform,$os_ver" >> "$test_file"
}

function get_average(){
	test_file=$1
	count=0;
	total=0;
	values=$(awk -F, 'FNR>=2{ print $3; }' "$test_file")
	for i in $values; do
		total=$(echo $total+$i | bc )
		count=$((count + 1))
	done
	echo "Average: " >> "$test_file"
	echo "scale=2; $total / $count" | bc >> "$test_file"
}
