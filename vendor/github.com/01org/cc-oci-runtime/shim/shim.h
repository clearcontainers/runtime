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

#include <stdio.h>

/* The shim would be handling fixed number of predefined fds.
 * This would be signal fd, stdin fd, proxy socket fd and an I/O
 * fd passed by the runtime
 */
#define MAX_POLL_FDS 4

struct cc_shim {
	char       *container_id;
	int         proxy_sock_fd;
	int         proxy_io_fd;
	uint64_t    io_seq_no;
	uint64_t    err_seq_no;
	bool        exiting;
};

/*
 * control message format
 * | ctrl id | length  | payload (length-8)      |
 * | . . . . | . . . . | . . . . . . . . . . . . |
 * 0         4         8                         length
 */
#define CONTROL_HEADER_SIZE             8
#define CONTROL_HEADER_LENGTH_OFFSET    4

/*
 * stream message format
 * | stream sequence | length  | payload (length-12)     |
 * | . . . . . . . . | . . . . | . . . . . . . . . . . . |
 * 0                 8         12                        length
 */
#define STREAM_HEADER_SIZE              12
#define STREAM_HEADER_LENGTH_OFFSET     8

#define PROXY_CTL_HEADER_SIZE           8
#define PROXY_CTL_HEADER_LENGTH_OFFSET  0

/*
 * Hyperstart is limited to sending this number of bytes to
 * a client.
 *
 * (This value can be determined by inspecting the hyperstart
 * source where hyper_event_ops->wbuf_size is set).
 */
#define HYPERSTART_MAX_RECV_BYTES       10240

