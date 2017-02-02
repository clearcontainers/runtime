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

# Perform static go tests.

function usage {
	echo "Usage $0 [OPTIONS] [PACKAGES]"
	echo "Perform static go checks on PACKAGES (./... by default)."
	echo
	echo "List of options:"
	echo "  -h, --help             print this help"
	echo "  -n, --no-network       do not access the network"
}

for i in "$@"; do
	case $i in
		-h|--help)
			usage
			exit 0
			;;
		-n|--no-network)
			NONETWORK=1
			shift
			;;
		*)
			args="$args $i"
			;;
	esac
done

go_packages=$args

[ -z "$go_packages" ] && {
	go_packages=$(go list ./... | grep -v cc-oci-runtime/vendor |\
		    sed -e 's#.*/cc-oci-runtime/#./#')
}

function install_package {
	url="$1"
	name=${url##*/}

	[ -n "$NONETWORK" ] && return

	echo Updating $name...
	go get -u $url
}

install_package github.com/fzipp/gocyclo
install_package github.com/client9/misspell/cmd/misspell
install_package github.com/golang/lint/golint
install_package github.com/gordonklaus/ineffassign

echo Doing go static checks on packages: $go_packages

echo "Running misspell..."
go list -f '{{.Dir}}/*.go' $go_packages |\
    xargs -I % bash -c "misspell -error %"

echo "Running go vet..."
go vet $go_packages

echo "Running gofmt..."
go list -f '{{.Dir}}' $go_packages |\
    xargs gofmt -s -l | tee /dev/tty | \
    wc -l | xargs -I % bash -c "test % -eq 0"

echo "Running cyclo..."
go list -f '{{.Dir}}' $go_packages | xargs gocyclo -over 15

echo "Running golint..."
for p in $go_packages; do golint -set_exit_status $p; done

echo "Running ineffassign..."
go list -f '{{.Dir}}' $go_packages | xargs -L 1 ineffassign

echo "All Good!"
