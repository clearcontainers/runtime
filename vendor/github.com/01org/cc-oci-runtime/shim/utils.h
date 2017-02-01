// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#pragma once

#include <stdio.h>

extern int shim_signal_table[];

bool set_fd_nonblocking(int fd);
void set_big_endian_32(uint8_t *buf, uint32_t val);
uint32_t get_big_endian_32(const uint8_t *buf);
void set_big_endian_64(uint8_t *buf, uint64_t val);
uint64_t get_big_endian_64(const uint8_t *buf);
