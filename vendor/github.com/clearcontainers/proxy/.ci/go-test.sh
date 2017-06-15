#!/bin/bash -e

script_dir=$(cd `dirname $0`; pwd)
root_dir=`dirname $script_dir`

test_packages=('.;.,./client' './api;./api')
go_test_flags="-v -race -timeout 2s"

echo Running go test on packages "'$test_packages'" with flags "'$go_test_flags'"

function test_html_coverage
{
	test_coverage
	go tool cover -html=profile.cov
	rm -f profile.cov
}

function test_coverage
{
	echo "mode: atomic" > profile.cov

	for pkg in ${test_packages[@]}; do
		fields=(${pkg//;/ })
		pkg_name=${fields[0]}
		pkg_cover=${fields[1]}
		go test $go_test_flags -covermode=atomic -coverprofile=profile_tmp.cov -coverpkg $pkg_cover $pkg_name
		[ -f profile_tmp.cov ] && tail -n +2 profile_tmp.cov >> profile.cov;
		rm -f profile_tmp.cov
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
