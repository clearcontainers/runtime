
[![Build Status](https://travis-ci.org/dlespiau/covertool.svg?branch=master)](https://travis-ci.org/dlespiau/covertool)

# What is this?

This repository contains support packages and tools to produce
and use coverage-instrumented Go programs.

The full story can be read in this [blog post](
http://damien.lespiau.name/2017/05/building-and-using-coverage.html).

Package [cover](https://github.com/dlespiau/covertool/tree/master/pkg/cover)
can be used to build instrumented programs.

Package [exit](https://github.com/dlespiau/covertool/tree/master/pkg/exit)
is an atexit implementation.

The covertool utility can merge profiles produced by different runs of the
same binary and display the resulting code coverage:

```
$ go install github.com/dlespiau/covertool
$ covertool merge -o all.go unit-tests.cov usecase1.cov usecase2.cov error1.cov error2.cov ...
$ covertool report all.go
coverage: 92.9% of statements
```

Finally, the `example/calc` directory contains a fully working example:

```
$ cd $GOPATH/src/github.com/dlespiau/covertool/examples/calc
$ ./run-tests.sh 
• Build the coverage-instrumented version of calc

• Run the unit tests
PASS
coverage: 7.1% of statements
ok  	github.com/dlespiau/covertool/examples/calc	0.003s

• Cover the sub() function
• Result: coverage: 57.1% of statements

• Cover the error path taken when not enough arguments are provided
expected 3 arguments, got 1
• Result: coverage: 21.4% of statements

• Cover the error path taken when providing an unknown operation
unknown operation: mul
• Result: coverage: 50.0% of statements

• Merge all coverage profiles and report the total coverage
coverage: 92.9% of statements
```
