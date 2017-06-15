#!/bin/bash

# To run in the github.com/dlespiau/covertool/examples/calc directory

function fatal() {
    echo $1
    exit 1
}

go install github.com/dlespiau/covertool

echo "• Build the coverage-instrumented version of calc"
go test -o calc -covermode count &> /dev/null

echo
echo "• Run the unit tests"
go test -covermode count -coverprofile unit-tests.cov

# Run calc with some combination of arguments to reach code paths not tested
# by unit tests.
# One would also check calc returns a proper error message and exit status but
# I've omitted those for simplicity.

echo
echo "• Cover the sub() function"
result="$(./calc -test.coverprofile=sub.cov sub 1 2)"
[[ $result != "-1" ]] && fatal "expected -1 got $result"
echo "• Result: `covertool report sub.cov`"

echo
echo "• Cover the error path taken when not enough arguments are provided"
./calc -test.coverprofile=error1.cov foo
echo "• Result: `covertool report error1.cov`"

echo
echo "• Cover the error path taken when providing an unknown operation"
./calc -test.coverprofile=error2.cov mul 3 4
echo "• Result: `covertool report error2.cov`"

# time to merge all coverage profiles

echo
echo "• Merge all coverage profiles and report the total coverage"
covertool merge -o all.cov unit-tests.cov sub.cov error1.cov error2.cov
covertool report all.cov
