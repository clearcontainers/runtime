#!/bin/bash -e

script_dir=$(cd `dirname $0`; pwd)
root_dir=`dirname $script_dir`

test_packages="."
go_test_flags="-v -race -timeout 5s"

# Run a command as either root or non-root (current user).
#
# If the first argument is "root", run using sudo, else run as normal.
# All arguments after the first will be treated as the command to run.
function run_as_user
{
	user="$1"
	shift
	cmd=$*

	if [ "$user" = root ]
	then
		# use a shell to ensure PATH is correct.
		sudo -E PATH="$PATH" sh -c "$cmd"
	else
		$cmd
	fi
}

function test_html_coverage
{
	test_coverage
	go tool cover -html=profile.cov -o coverage.html
	rm -f profile.cov
}

function test_coverage
{
	cov_file="profile.cov"
	tmp_cov_file="profile_tmp.cov"

	echo "mode: atomic" > "$cov_file"

	for pkg in $test_packages; do

		# Run the unit-tests *twice* (since some must run as root and
		# others must run as non-root), combining the resulting test
		# coverage files.
		for user in non-root root; do
			printf "INFO: Running 'go test' as %s user on packages '%s' with flags '%s'\n" "$user" "$test_packages" "$go_test_flags"

			run_as_user "$user" go test $go_test_flags -covermode=atomic -coverprofile="$tmp_cov_file" $pkg
			if [ -f "${tmp_cov_file}" ]; then
				run_as_user "$user" chmod 644 "$tmp_cov_file"
				tail -n +2 "$tmp_cov_file" >> "$cov_file"
				run_as_user "$user" rm -f "$tmp_cov_file"
			fi
		done
	done
}

function test_local
{
	go test $go_test_flags $test_packages
}

if [ "$1" = "html-coverage" ]; then
	test_html_coverage
elif [ "$CI" = "true" ]; then
	test_coverage
else
	test_local
fi
