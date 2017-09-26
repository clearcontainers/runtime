#!/bin/bash
#
# Copyright (c) 2017 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

verbose=0
script_name=${0##*/}

usage()
{
    cat <<EOT
Usage: $script_name [options]

Description: Find running containers that are not using the latest assets
             (image and kernel).

Options:

  -h : Show this help.
  -v : Show containers as they are being checked.

Exit codes:

    0 on success.
    1 on error.
    2 if any containers are found to be "stale"
      (not using the latest assets).

EOT
}

die()
{
    msg="$*"
    echo >&2 "ERROR: $msg"
    exit 1
}

while getopts "hv" opt
do
    case "$opt" in
        h)
            usage
            exit 0
            ;;
        v)
            verbose=1
            ;;
    esac
done

runtime_name="cc-runtime"
runtime=$(command -v "$runtime_name" 2>/dev/null || :)
[ -z "$runtime" ] && die "cannot find runtime $runtime_name"

# Determine the newest asset versions
latest_kernel=$( $runtime cc-env|grep -A 1 '\[Kernel\]'|egrep "\<Path\>"|cut -d= -f2-|awk '{print $1}'|tr -d '"')
latest_image=$(  $runtime cc-env|grep -A 2 '\[Image\]' |egrep "\<Path\>"|cut -d= -f2-|awk '{print $1}'|tr -d '"')

# Check containers
$runtime list --cc-all|\
    tail -n +2|\
    awk \
        -v latest_image="$latest_image" \
        -v latest_kernel="$latest_kernel" \
        -v verbose="$verbose" \
        '$3 == "running" {
    container = $1

    if (verbose == 1) {
        printf("Checking container '%s'\n", container)
    }

    # XXX: See https://github.com/clearcontainers/runtime/issues/294
    if (NF == 8) {
        kernel_field = $7
        image_field = $8
    } else if (NF == 9) {
        kernel_field = $8
        image_field = $9
    } else {
        printf("ERROR: unexpected number of fields: %d\n", NF)
        exit(1)
    }

    if (kernel_field != latest_kernel) {
        printf("WARNING: Container %s not using newest kernel (newest %s, using %s)\n",
            container, latest_kernel, kernel_field)
            exitCode = 2
    } else if (image_field != latest_image) {
        printf("WARNING: Container %s not using newest image (newest %s, using %s)\n",
            container, latest_image, image_field)
            exitCode = 2
    }
} END {
    exit(exitCode)
}'
