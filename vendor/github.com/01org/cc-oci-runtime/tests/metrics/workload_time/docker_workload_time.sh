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
#  This test measures the complete workload of a container using docker.
#  From calling docker until the workload is completed and the container
#  is shutdown.

set -e

[ $# -ne 4 ] && ( echo >&2 "Usage: $0 <cmd to run> <image> <runtime> <times to run>"; exit 1 )

SCRIPT_PATH=$(dirname "$(readlink -f "$0")")
source "${SCRIPT_PATH}/../../lib/test-common.bash"

CMD="$1"
IMAGE="$2"
RUNTIME="$3"
TIMES="$4"
TMP_FILE=$(mktemp workloadTime.XXXXXXXXXX || true)
TEST_NAME="docker run time"
TEST_ARGS="image=${IMAGE} command=${CMD} runtime=${RUNTIME} units=seconds"
TEST_RESULT_FILE=$(echo "${RESULT_DIR}/${TEST_NAME}-${IMAGE}-${CMD}-${RUNTIME}" | sed 's| |-|g')

function run_workload(){
	if [[ "$RUNTIME" != 'runc' && "$RUNTIME" != 'cor' ]]; then
		die "Runtime ${RUNTIME} is not valid"
	fi
	(time -p $DOCKER_EXE run -ti --runtime "$RUNTIME" "$IMAGE" "$CMD") &> "$TMP_FILE"
	if [ $? -eq 0 ]; then
		test_data=$(grep ^real "$TMP_FILE" | cut -f2 -d' ')
		write_result_to_file "$TEST_NAME" "$TEST_ARGS" "$test_data" "$TEST_RESULT_FILE"
	fi
	rm -f $TMP_FILE
}

echo "Executing test: ${TEST_NAME} ${TEST_ARGS}"
backup_old_file "$TEST_RESULT_FILE"
write_csv_header "$TEST_RESULT_FILE"
for i in $(seq 1 "$TIMES"); do
	run_workload
done
get_average "$TEST_RESULT_FILE"
clean_docker_ps
