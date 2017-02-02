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
#  This test measures the time to stop a container using docker stop
#  This measure is available for COR and runc

set -e

[ $# -ne 2 ] && ( echo >&2 "Usage: $0 <runtime> <times to run>"; exit 1 )

SCRIPT_PATH=$(dirname "$(readlink -f "$0")")
source "${SCRIPT_PATH}/../../lib/test-common.bash"

CMD='sh'
IMAGE='ubuntu'
RUNTIME="$1"
TIMES="$2"
TEST_NAME="docker stop time"
TEST_ARGS="image=${IMAGE} runtime=${RUNTIME} units=seconds"
TEST_RESULT_FILE=$(echo "${RESULT_DIR}/${TEST_NAME}-${RUNTIME}" | sed 's| |-|g')
TMP_FILE=$(mktemp workloadTime.XXXXXXXXXX || true)

function shut_down_container(){
	if [[ "$RUNTIME" != 'runc' && "$RUNTIME" != 'cor' ]]; then
        die "Runtime ${RUNTIME} is not valid"
	fi
	$DOCKER_EXE run -tid --runtime "$RUNTIME" "$IMAGE" "$CMD" > /dev/null && \
		(time -p ${DOCKER_EXE} stop "$($DOCKER_EXE ps -q)") &> "$TMP_FILE" && \
		test_data=$(grep 'real' "$TMP_FILE" | cut -f2 -d' ')
	write_result_to_file "$TEST_NAME" "$TEST_ARGS" "$test_data" "$TEST_RESULT_FILE"
	rm -f $TMP_FILE
}

echo "Executing test: ${TEST_NAME} ${TEST_ARGS}"
backup_old_file  "$TEST_RESULT_FILE"
write_csv_header "$TEST_RESULT_FILE"
for i in $(seq 1 "$TIMES"); do
	shut_down_container
done
get_average "$TEST_RESULT_FILE"
clean_docker_ps
