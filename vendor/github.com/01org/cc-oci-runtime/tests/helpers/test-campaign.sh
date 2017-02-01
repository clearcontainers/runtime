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

# This script builds cc-oci-runtime with your local changes and
# then executes the test suites contained in this repository.
# A log directory called `test_logs` in this path is created with all logs from the
# tests executed. A brief summary called `test_summary.log` can be found in this path.

# Run this script from the root directory of this repository
# E.g.
# [foo@bar cc-oci-runtime]$ ./tests/helpers/test-campaign.sh


SCRIPT_PATH=$(dirname "$(readlink -f "$0")")
SCRIPT_NAME=${0##*/}

[ -z "$GOPATH" ] && echo >&2 "${SCRIPT_NAME}: Please set GOPATH Env variable first." && exit 1

# Default values:
QEMU_PATH="/usr/bin/qemu-lite-system-x86_64"
IMAGE_PATH="/usr/share/clear-containers/clear-containers.img"
KERNEL_PATH="/usr/share/clear-containers/vmlinux.container"
BUNDLE_PATH="/var/lib/oci/bundle"
LOG_DIR="${SCRIPT_PATH}/test_logs"

function usage()
{
	cat << EOT
	Usage: $SCRIPT_NAME [options]
	This script builds and tests your cc-oci-runtime local changes.

	Options:
		-b <bundle-path>           : Full path to the bundle directory
		-h                         : Show this information
		-i <image-path>            : Full path to the clear-containers image
		-k <kernel-path>           : Full path to the clear-containers kernel
		-l <logs directory>        : Path of desired logs location.
		-q <qemu-path>             : Full path to qemu-lite

	Notes:
	If you do not specify an option, the script will work with default values:
	Default values:
		bundle-path            : $BUNDLE_PATH
		image-path             : $IMAGE_PATH
		kernel-path            : $KERNEL_PATH
		qemu-path              : $QEMU_PATH
		logs directory         : $LOG_DIR
EOT
}

while getopts "b:hi:k:l:q:" opt; do
	case $opt in
		b)
			BUNDLE_PATH="${OPTARG}"
			;;
		h)
			usage
			exit 0
			;;
		i)
			IMAGE_PATH="${OPTARG}"
			;;
		k)
			KERNEL_PATH="${OPTARG}"
			;;
		l)
			LOG_DIR="${OPTARG}"
			;;
		q)
			QEMU_PATH="${OPTARG}"
			;;
		\?)
			exit 1
	esac
done

shift "$((OPTIND - 1))"

AUTOGEN_LOG_FILE="${LOG_DIR}/autogen.log"
MAKE_LOG_FILE="${LOG_DIR}/make.log"
MAKE_INSTALL_LOG_FILE="${LOG_DIR}/make_install.log"
UNIT_TESTS_LOG_FILE="${LOG_DIR}/unit_tests.log"
FUNCTIONAL_TESTS_LOG_FILE="${LOG_DIR}/functional_tests.log"
VALGRIND_LOG_FILE="${LOG_DIR}/valgrind_tests.log"
CPPCHECK_LOG_FILE="${LOG_DIR}/cppcheck.log"
COVERAGE_LOG_FILE="${LOG_DIR}/code_coverage.log"
GO_TESTS_LOG_FILE="${LOG_DIR}/proxy_tests.log"
DOCKER_TESTS_LOG_FILE="${LOG_DIR}/docker-integration-tests.html"
SUMMARY_LOG_FILE="${LOG_DIR}/test_summary.log"

# Set return codes to 1
AUTOGEN_RC=1
MAKE_RC=1
MAKE_INSTALL_RC=1
UNIT_TESTS_RC=1
FUNCTIONAL_TESTS_RC=1
VALGRIND_TESTS_RC=1
CPPCHECK_RC=1
COVERAGE_RC=1
GO_TEST_RC=1
DOCKER_TESTS_RC=1

function run_autogen(){
	echo "autogen execution"
	./autogen.sh \
		--with-qemu-path="$QEMU_PATH" \
		--with-cc-image="$IMAGE_PATH" \
		--with-cc-kernel="$KERNEL_PATH" \
		--with-tests-bundle-path="$BUNDLE_PATH" \
		--enable-debug \
		--disable-silent-rules \
		--enable-cppcheck \
		--enable-valgrind \
		--disable-valgrind-helgrind \
		--disable-valgrind-drd \
		--enable-functional-tests \
		--disable-docker-tests \
		--enable-code-coverage \
		2>&1 | tee "$AUTOGEN_LOG_FILE"
	( exit "${PIPESTATUS[0]}" ) && AUTOGEN_RC=0
}

function run_make(){
	echo "make execution"
	make 2>&1 | tee "$MAKE_LOG_FILE"
	( exit "${PIPESTATUS[0]}" ) && MAKE_RC=0
}

function run_make_install(){
	echo "make execution"
	sudo make install 2>&1 | tee "$MAKE_INSTALL_LOG_FILE"
	( exit "${PIPESTATUS[0]}" ) && MAKE_INSTALL_RC=0
}

function run_test(){
	test_name=$1
	test_log=$2
	echo "Running test: $test_name"
	make "$test_name" 2>&1 | tee "$test_log"
	( exit "${PIPESTATUS[0]}" )
}

function run_docker_tests(){
	echo 'docker tests verification:'
	pushd "${SCRIPT_PATH}/../integration/docker"
	sudo prove -m -Q --formatter=TAP::Formatter::HTML ./*.bats >> "$DOCKER_TESTS_LOG_FILE" \
	&& DOCKER_TESTS_RC=0
	total_tests=$(awk '/^[0-9]* tests/ {print $1}' "$DOCKER_TESTS_LOG_FILE")
	passed=$(awk '/^[0-9]* ok/ {print $1}' "$DOCKER_TESTS_LOG_FILE")
	failed=$(awk '/^[0-9]* failed/ {print $1}' "$DOCKER_TESTS_LOG_FILE")
	skipped=$(awk '/^[0-9]* skipped/ {print $1}' "$DOCKER_TESTS_LOG_FILE")
	passed=$((passed-skipped))
	echo -e "\nSummary for Docker Tests:" | tee -a "$SUMMARY_LOG_FILE"
	echo "Total Tests:  ${total_tests}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Passed Tests: ${passed}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Failed Tests: ${failed}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Skipped Tests: ${skipped}" | tee -a "$SUMMARY_LOG_FILE"
	popd
}

# Valgrind and unit tests
function check_tests(){
	log_file=$1
	test_name="${log_file/'.log'/}"
	total_tests=$(awk '/# TOTAL/ {print $3}' "$log_file")
	passed=$(awk '/# PASS/ {print $3}' "$log_file")
	failed=$(awk '/# FAIL/ {print $3}' "$log_file")
	skipped=$(awk '/# SKIP/ {print $3}' "$log_file")
	echo -e "\nSummary for ${test_name}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Total Tests:  ${total_tests}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Passed Tests: ${passed}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Failed Tests: ${failed}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Skipped Tests: ${skipped}" | tee -a "$SUMMARY_LOG_FILE"
}

# bats functional tests
function check_functional_tests(){
	log_file=$1
	test_name="${log_file/'.log'/}"
	total_tests=$(awk -F "." '/^1../ {print $3}' "$log_file")
	passed=$(grep -c "^ok" "$log_file")
	failed=$(grep -c "^not ok" "$log_file")
	skipped=$(grep -c "# skip" "$log_file")
	passed=$((passed-skipped))
	echo -e "\nSummary for ${test_name}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Total Tests:  ${total_tests}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Passed Tests: ${passed}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Failed Tests: ${failed}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Skipped Tests: ${skipped}" | tee -a "$SUMMARY_LOG_FILE"
}

function get_coverage(){
	log_file=$1
	lines_coverage=$(grep -A2 "Overall coverage" "$log_file" | awk '/lines/ {print $2}')
	functions_coverage=$(grep -A2 "Overall coverage" "$log_file" | awk '/functions/ {print $2}')
	echo -e "\nCoverage Report: " | tee -a "$SUMMARY_LOG_FILE"
	echo "Lines Coverage: ${lines_coverage}" | tee -a "$SUMMARY_LOG_FILE"
	echo "Functions Coverage: ${functions_coverage}" | tee -a "$SUMMARY_LOG_FILE"
}

if [ -d "$LOG_DIR" ]; then
	rm -rf "$LOG_DIR"
fi

mkdir "$LOG_DIR"

# Execute Tests
run_autogen
run_make
run_make_install
run_test "check-TESTS" "$UNIT_TESTS_LOG_FILE" && UNIT_TESTS_RC=0
check_tests "$UNIT_TESTS_LOG_FILE"
run_test "functional-tests" "$FUNCTIONAL_TESTS_LOG_FILE" && FUNCTIONAL_TESTS_RC=0
check_functional_tests "$FUNCTIONAL_TESTS_LOG_FILE"
run_test "check-valgrind" "$VALGRIND_LOG_FILE" && VALGRIND_TESTS_RC=0
check_tests "$VALGRIND_LOG_FILE"
run_test "cppcheck" "$CPPCHECK_LOG_FILE" && CPPCHECK_RC=0
run_test "code-coverage-capture" "$COVERAGE_LOG_FILE" && COVERAGE_RC=0
get_coverage "$COVERAGE_LOG_FILE"
run_test "check-proxy" "$GO_TESTS_LOG_FILE" && GO_TEST_RC=0
run_docker_tests

# Print Return Codes
echo -e "\nReturn Codes of executed checks:" | tee -a "$SUMMARY_LOG_FILE"
echo "autogen return code: $AUTOGEN_RC" | tee -a "$SUMMARY_LOG_FILE"
echo "make return code: $MAKE_RC" | tee -a "$SUMMARY_LOG_FILE"
echo "make install return code: $MAKE_INSTALL_RC" | tee -a "$SUMMARY_LOG_FILE"
echo "unit tests return code: $UNIT_TESTS_RC" | tee -a "$SUMMARY_LOG_FILE"
echo "functional tests return code: $FUNCTIONAL_TESTS_RC" | tee -a "$SUMMARY_LOG_FILE"
echo "valgrind tests return code: $VALGRIND_TESTS_RC" | tee -a "$SUMMARY_LOG_FILE"
echo "cppcheck return code: $CPPCHECK_RC" | tee -a "$SUMMARY_LOG_FILE"
echo "coverage return code: $COVERAGE_RC" | tee -a "$SUMMARY_LOG_FILE"
echo "check-proxy return code: $GO_TEST_RC" | tee -a "$SUMMARY_LOG_FILE"
echo "docker tests return code: $DOCKER_TESTS_RC" | tee -a "$SUMMARY_LOG_FILE"
