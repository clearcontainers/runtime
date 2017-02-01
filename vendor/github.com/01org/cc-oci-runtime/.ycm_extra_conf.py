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
#  Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
#  MA  02110-1301, USA.
#

import os
import subprocess
import ycm_core

# FIXME: should generate this list as it comes from configure.ac
pkgs = [
    'check',
    'gio-unix-2.0',
    'glib-2.0',
    'json-glib-1.0',
    'uuid',
]

# Generic.
flags = [
    '-Wall',
    '-Wextra',
    '-Werror',
    '-pthread',
    '-DUSE_CLANG_COMPLETER',
]

# Tell YCM where to find local headers.
#
# XXX: Spaces after flags are *NOT* allowed!!
flags += [
    '-I.',
    '-Isrc/',
    '-Itests/',
]

# Add includes for dependent packages.
for pkg in pkgs:
    includes = subprocess.check_output(
        ['pkg-config', '--cflags', pkg],
        universal_newlines=True
    )
    includes = includes.strip().split(' ')
    flags += includes


# YCM calls this function for each file to determine which compiler
# flags to use.
#
# (We treat all files identically).
def FlagsForFile(filename):
    return {'flags': flags, 'do_cache': True}
