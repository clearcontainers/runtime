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

set -e

source $(dirname "$0")/ci-common.sh

[ "$SEMAPHORE_THREAD_RESULT" = "passed" ] && exit 0

printf "=== Build failed ===\n"

cd "$ci_build_dir"

for f in test-suite.log $(ls *_test*.log)
do
    printf "\n=== Log file: '$f' ===\n\n"
    cat "$f"
done
