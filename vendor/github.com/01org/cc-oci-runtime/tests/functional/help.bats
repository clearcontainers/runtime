#!/usr/bin/env bats
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
#

#Based on runc commands
load common

function setup() {
	setup_common
}

function teardown() {
	cleanup_common
}

@test "cor -h" {
	run $COR -h
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
	[[ ${lines[1]} == *cc-oci-runtime* ]]
}

@test "cor --help" {
	run $COR --help
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
	[[ ${lines[1]} == *cc-oci-runtime* ]]
}

@test "cor help" {
	run $COR help
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
	[[ ${lines[1]} == *cc-oci-runtime* ]]
}

@test "cor exec -h" {
	run $COR exec -h
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor exec --help" {
	run $COR exec --help
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor kill -h" {
	run $COR kill -h
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor kill --help" {
	run $COR kill --help
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor list -h" {
	run $COR list -h
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor list --help" {
	run $COR list --help
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor pause -h" {
	run $COR pause -h
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor pause --help" {
	run $COR pause --help
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor resume -h" {
	run $COR resume -h
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor resume --help" {
	run $COR resume --help
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor start -h" {
	run $COR start -h
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor start --help" {
	run $COR start --help
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor state -h" {
	run $COR state -h
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor state --help" {
	run $COR state --help
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor delete -h" {
	run $COR delete -h
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor delete --help" {
	run $COR delete --help
	[ "$status" -eq 0 ]
	[[ ${lines[0]} =~ Usage:+ ]]
}

@test "cor foo -h" {
	run $COR foo -h
	[ "$status" -ne 0 ]
	[[ "${output}" == *"no such command: foo"* ]]
}

@test "cor foo --help" {
	run $COR foo --help
	[ "$status" -ne 0 ]
	[[ "${output}" == *"no such command: foo"* ]]
}
