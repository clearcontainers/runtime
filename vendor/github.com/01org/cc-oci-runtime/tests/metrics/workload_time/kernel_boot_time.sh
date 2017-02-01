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

#  Description of the test:
#  This test measures the time kernelspace takes when a clear container
#  boots.

set -e

[ $# -ne 1 ] && ( echo >&2 "Usage: $0 <times to run>"; exit 1 )

SCRIPT_PATH=$(dirname "$(readlink -f "$0")")
source "${SCRIPT_PATH}/../common/test.common"

TEST_NAME="Kernel Boot Time"
TIMES="$1"
TMP_FILE=$(mktemp dmesglog.XXXXXXXXXX || true)

function get_kernelspace_time(){
	net=$1
	test_args="Network"
	if [ "$net" == "nonet" ]
	then
		test_args="No-network"
		run_options="--net none"
	fi
	test_result_file=$(echo "${RESULT_DIR}/${TEST_NAME}-${test_args}" | sed 's| |-|g')
	backup_old_file "$test_result_file"
	write_csv_header "$test_result_file"
	for i in $(seq 1 "$TIMES")
	do
		eval docker run "$run_options" -ti debian dmesg > "$TMP_FILE"
		test_data=$(grep "Freeing" "$TMP_FILE" | tail -1 | awk '{print $2}' | cut -d']' -f1)
		write_result_to_file "$TEST_NAME" "$test_args" "$test_data" "$test_result_file"
		rm "$TMP_FILE"
	done
}

echo "Executing test: ${TEST_NAME}"

get_kernelspace_time
get_kernelspace_time nonet

clean_docker_ps
