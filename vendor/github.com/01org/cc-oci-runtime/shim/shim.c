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

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <string.h>
#include <sys/un.h>
#include <poll.h>
#include <assert.h>
#include <stdarg.h>
#include <unistd.h>
#include <signal.h>
#include <termios.h>
#include <sys/ioctl.h>
#include <stdint.h>
#include <inttypes.h>
#include <getopt.h>
#include <stdbool.h>
#include <limits.h>
#include <fcntl.h>
#include <termios.h>

#include "utils.h"
#include "log.h"
#include "shim.h"

/* globals */

struct pollfd poll_fds[MAX_POLL_FDS] = {{-1}};

// File descriptors are added at specific index in the poll_fds array
#define SIGNAL_FD_INDEX 0
#define PROXY_IO_INDEX 1
#define PROXY_CTL_INDEX 2
#define STDIN_INDEX 3

/* Pipe used for capturing signal occurence */
int signal_pipe_fd[2] = { -1, -1 };

static char *program_name;

struct termios *saved_term_settings;

/*!
 * Add file descriptor to the array of polled descriptors
 *
 * \param poll_fds Array of polled fds
 * \param index Index at which the fd is added
 * \param fd File descriptor to add
 * \param events Events for the fd that should be polled
 */
void add_pollfd(struct pollfd *poll_fds, nfds_t index, int fd,  short events) {
	struct pollfd pfd = { 0 };

	if ( !poll_fds || fd < 0 || index >= MAX_POLL_FDS) {
		shim_warning("Not able to add fd to poll_fds array\n");
		return;
	}

	pfd.fd = fd;
	pfd.events = events;
	poll_fds[index] = pfd;
}

/*!
 * Signal handler for the signals that should be caught and 
 * forwarded to the proxy
 *
 * \param signal_no Signal number of the signal caught
 */
static void
signal_handler(int signal_no)
{
	int savedErrno;                     /* In case we change 'errno' */
 
	savedErrno = errno;
	/* Write signal number to pipe, so that the signal can be identfied later */
	if (write(signal_pipe_fd[1], &signal_no, sizeof(signal_no)) == -1 && errno != EAGAIN) {
		return;
	}
	errno = savedErrno;
}

/*!
 * Assign signal handler for all the signals that should be
 * forwarded by the shim to the proxy.
 *
 * \param sa Signal handler
 * \return true on success, false otherwise
 */
bool
assign_all_signals(struct sigaction *sa)
{
	if (! sa) {
		return false;
	}

        for (int i = 0; shim_signal_table[i]; i++) {
                if (sigaction(shim_signal_table[i], sa, NULL) == -1) {
			shim_error("Error assigning signal handler for %d : %s\n",
				shim_signal_table[i], strerror(errno));
                        return false;
                }
        }
        return true;
}

void restore_terminal(void) {
	if ( isatty(STDIN_FILENO) && saved_term_settings) {
		if (tcsetattr (STDIN_FILENO, TCSANOW, saved_term_settings)) {
			shim_warning("Unable to restore terminal: %s\n",
					strerror(errno));
		}
		free(saved_term_settings);
		saved_term_settings = NULL;
	}
}

/*!
 * Print formatted message to stderr and exit with EXIT_FAILURE
 *
 * \param format Format that specifies how subsequent arguments are
 *  converted for output
 */
void
err_exit(const char *format, ...)
{
	va_list	args;

	if ( !format) {
		return;
	}
	fprintf(stderr, "%s: ", program_name);
	va_start(args, format);
	vfprintf(stderr, format, args);
	va_end(args);
	restore_terminal();
	exit(EXIT_FAILURE);
}

/*!
 * Construct message in the proxy ctl rpc protocol format
 * Proxy control message format:
 *
 * | length  | Reserved| Data(Request/Response   |
 * | . . . . | . . . . | . . . . . . . . . . . . |
 * 0         4         8                         length
 *
 * \param json Json Payload
 * \param[out] len Length of the message
 *
 * \return Newly allocated string on sucess, NULL on failure
 */
char*
get_proxy_ctl_msg(const char *json, size_t *len) {
	char   *proxy_ctl_msg = NULL;
	size_t  json_len;

	if (! (json && len)) {
		return NULL;
	}

	json_len = strlen(json);
	*len = json_len + PROXY_CTL_HEADER_SIZE;
	proxy_ctl_msg = calloc(*len + 1, sizeof(char));

	if (! proxy_ctl_msg) {
		abort();
	}

	set_big_endian_32((uint8_t*)proxy_ctl_msg + PROXY_CTL_HEADER_LENGTH_OFFSET, 
				(uint32_t)(json_len));
	strncpy(proxy_ctl_msg + PROXY_CTL_HEADER_SIZE, json, json_len);

	return proxy_ctl_msg;
 }

/*!
 * Send "hyper" payload to cc-proxy. This will be forwarded to hyperstart.
 *
 * \param fd File descriptor to send the message to(should be proxy ctl socket fd)
 * \param Hyperstart cmd id
 * \param json Json payload
 */
void
send_proxy_hyper_message(int fd, const char *hyper_cmd, const char *json) {
	char      *proxy_payload = NULL;
	char      *proxy_command_id = "hyper";
	char      *proxy_ctl_msg = NULL;
	size_t     len = 0, offset = 0;
	ssize_t    ret;

	/* cc-proxy has the following format for "hyper" payload:
	 * {
	 *    "id": "hyper",
	 *    "data": {
	 *        "hyperName": "ping",
	 *        "data": "json payload",
	 *    }
	 * }
	*/

	if ( !json || fd < 0) {
		return;
	}


	ret = asprintf(&proxy_payload,
			"{\"id\":\"%s\",\"data\":{\"hyperName\":\"%s\",\"data\":%s}}",
			proxy_command_id, hyper_cmd, json);

	if (ret == -1) {
		abort();
	}

	proxy_ctl_msg = get_proxy_ctl_msg(proxy_payload, &len);
	free(proxy_payload);

	while (offset < len) {
		ret = write(fd, proxy_ctl_msg + offset, len-offset);
		if (ret == EINTR) {
			continue;
		}
		if (ret <= 0 ) {
			free(proxy_ctl_msg);
			shim_error("Error writing to proxy: %s\n", strerror(errno));
			return;
		}
		offset += (size_t)ret;
	}
	free(proxy_ctl_msg);
}

/*!
 * Read signals received and send message in the hyperstart protocol
 * format to the proxy ctl socket.
 *
 * \param shim \ref cc_shim
 */
void
handle_signals(struct cc_shim *shim) {
	int                sig;
	char              *buf;
	int                ret;
	char              *cmd = NULL;
	struct winsize     ws;
	static char*       cmds[] = { "winsize", "killcontainer"};

	if ( !(shim && shim->container_id) || shim->proxy_sock_fd < 0) {
		return;
	}

	while (read(signal_pipe_fd[0], &sig, sizeof(sig)) != -1) {
		shim_debug("Handling signal : %d on fd %d\n", sig, signal_pipe_fd[0]);
		if (sig == SIGWINCH ) {
			cmd = cmds[0];
			if (ioctl(STDIN_FILENO, TIOCGWINSZ, &ws) == -1) {
				shim_warning("Error getting the current window size: %s\n",
					strerror(errno));
				continue;
			}
			ret = asprintf(&buf, "{\"seq\":%"PRIu64", \"row\":%d, \"column\":%d}",
					shim->io_seq_no, ws.ws_row, ws.ws_col);
			shim_debug("handled SIGWINCH for container %s (row=%d, column=%d)\n",
				shim->container_id, ws.ws_row, ws.ws_col);

		} else {
			cmd = cmds[1];
			ret = asprintf(&buf, "{\"container\":\"%s\", \"signal\":%d}",
                                                        shim->container_id, sig);
			shim_debug("Killed container %s with signal %d\n", shim->container_id, sig);
		}
		if (ret == -1) {
			abort();
		}

		send_proxy_hyper_message(shim->proxy_sock_fd, cmd, buf);
		free(buf);
        }
}

/*!
 * Read data from stdin(with tty set in raw mode)
 * and send it to proxy I/O channel
 * Reference : https://github.com/hyperhq/runv/blob/master/hypervisor/tty.go#L448
 *
 * \param shim \ref cc_shim
 */
void
handle_stdin(struct cc_shim *shim)
{
	ssize_t        nread;
	int            ret;
	ssize_t        len;
	static uint8_t buf[BUFSIZ+STREAM_HEADER_SIZE];

	if (! shim || shim->proxy_io_fd < 0) {
		return;
	}

	nread = read(STDIN_FILENO , buf+STREAM_HEADER_SIZE, BUFSIZ);
	if (nread < 0) {
		shim_warning("Error while reading stdin char :%s\n", strerror(errno));
		return;
	} else if (nread == 0) {
		/* EOF received on stdin, send eof to hyperstart and remove stdin fd from
		 * the polled descriptors to prevent further eof events
		 */
		poll_fds[STDIN_INDEX].fd = -1;
	}

	len = nread + STREAM_HEADER_SIZE;
	set_big_endian_64 (buf, shim->io_seq_no);
	set_big_endian_32 (buf + STREAM_HEADER_LENGTH_OFFSET, (uint32_t)len);

	// TODO: handle write in the poll loop to account for write blocking
	ret = (int)write(shim->proxy_io_fd, buf, (size_t)len);
	if (ret == -1) {
		shim_warning("Error writing from fd %d to fd %d: %s\n",
			     STDIN_FILENO, shim->proxy_io_fd, strerror(errno));
		return;
	}
}

/*!
 * Read and parse I/O message on proxy I/O fd
 *
 * \param shim \ref cc_shim
 * \param[out] seq Seqence number of the I/O stream
 * \param[out] stream_len Length of the data received
 *
 * \return newly allocated string on success, else \c NULL.
 */
char*
read_IO_message(struct cc_shim *shim, uint64_t *seq, ssize_t *stream_len) {
	char *buf = NULL;
	ssize_t need_read = STREAM_HEADER_SIZE;
	ssize_t bytes_read = 0, want, ret;
	ssize_t max_bytes = HYPERSTART_MAX_RECV_BYTES;

	if (! (shim && seq && stream_len)) {
		return NULL;
	}

	*stream_len = 0;

	buf = calloc(STREAM_HEADER_SIZE, 1);
	if (! buf ) {
		abort();
	}

	while (bytes_read < need_read) {
		want = need_read - bytes_read;
		if (want > BUFSIZ)  {
			want = BUFSIZ;
		}

		ret = read(shim->proxy_io_fd, buf+bytes_read, (size_t)want);
		if (ret == -1) {
			free(buf);
			err_exit("Error reading from proxy I/O fd: %s\n", strerror(errno));
		} else if (ret == 0) {
			/* EOF received on proxy I/O fd*/
			free(buf);
			err_exit("EOF received on proxy I/O fd\n");
		}

		bytes_read += ret;

		if (*stream_len == 0 && bytes_read >= STREAM_HEADER_SIZE) {
			*stream_len = get_big_endian_32((uint8_t*)(buf+STREAM_HEADER_LENGTH_OFFSET));

			// length is 12 when hyperstart sends eof before sending exit code
			if (*stream_len == STREAM_HEADER_SIZE) {
				break;
			}

			/* Ensure amount of data is within expected bounds */
			if (*stream_len > max_bytes) {
				shim_warning("message too big (limit is %lu, but proxy returned %lu)",
						(unsigned long int)max_bytes,
						(unsigned long int)stream_len);
				goto err;
			}

			if (*stream_len > STREAM_HEADER_SIZE) {
				need_read = *stream_len;
				buf = realloc(buf, (size_t)*stream_len);
				if (! buf) {
					abort();
				}
			}
		}
	}
	*seq = get_big_endian_64((uint8_t*)buf);
	return buf;

err:
	free(buf);
	return NULL;
}

/*!
 * Handle output on the proxy I/O fd
 *
 *\param shim \ref cc_shim
 */
void
handle_proxy_output(struct cc_shim *shim)
{
	uint64_t  seq;
	char     *buf = NULL;
	int       outfd;
	ssize_t   stream_len = 0;
	ssize_t   ret;
	ssize_t   offset;
	int       code = 0;

	if (shim == NULL) {
		return;
	}

	buf = read_IO_message(shim, &seq, &stream_len);
	if ((! buf) || (stream_len <= 0) || (stream_len > HYPERSTART_MAX_RECV_BYTES)) {
		/*TODO: is exiting here more appropriate, since this denotes
		 * error communicating with proxy or proxy has exited
		 */
		goto out;
	}

	if (seq == shim->io_seq_no) {
		outfd = STDOUT_FILENO;
	} else if (seq == shim->io_seq_no + 1) {//proxy allocates errseq 1 higher
		outfd = STDERR_FILENO;
	} else {
		shim_warning("Seq no %"PRIu64 " received from proxy does not match with\
				 shim seq %"PRIu64 "\n", seq, shim->io_seq_no);
		goto out;
	}

	if (!shim->exiting && stream_len == STREAM_HEADER_SIZE) {
		shim->exiting = true;
		goto out;
	} else if (shim->exiting && stream_len == (STREAM_HEADER_SIZE+1)) {
		code = *(buf + STREAM_HEADER_SIZE); 	// hyperstart has sent the exit status
		shim_debug("Exit status for container: %d\n", code);
		free(buf);
		restore_terminal();
		exit(code);
	}

	/* TODO: what if writing to stdout/err blocks? Add this to the poll loop
	 * to watch out for EPOLLOUT
	 */
	offset = STREAM_HEADER_SIZE;
	while (offset < stream_len) {
		ret = write(outfd, buf+offset, (size_t)(stream_len-offset));
		if (ret <= 0 ) {
			goto out;
		}
		offset += (ssize_t)ret;
	}

out:
	if (buf) {
		free (buf);
	}
}

/*!
 * Handle data on the proxy ctl socket fd
 *
 *\param shim \ref cc_shim
 */
void
handle_proxy_ctl(struct cc_shim *shim)
{
	char buf[LINE_MAX] = { 0 };
	ssize_t ret;

	if (! shim) {
		return;
	}

	ret = read(shim->proxy_sock_fd, buf, LINE_MAX-1);
	if (ret == -1) {
		err_exit("Error reading from the proxy ctl socket: %s\n", strerror(errno));
	} else if (ret == 0) {
		err_exit("EOF received on proxy ctl socket. Proxy has exited\n");
	}

	//TODO: Parse the json and log error responses explicitly
	shim_debug("Proxy response:%s\n", buf + PROXY_CTL_HEADER_SIZE);
}

/*
 * Parse number from input
 *
 * \return Long long integer on success, -1 on failure
 */
long long
parse_numeric_option(char *input) {
	char       *endptr;
	long long   num;

	if ( !input) {
		return -1;
	}

	errno = 0;
	num = strtoll(input, &endptr, 10);
	if ( errno || *endptr ) {
		return -1;
	}
	return num;
}

/*!
 * Print program usage
 */
void
print_usage(void) {
	printf("%s: Usage\n", program_name);
        printf("  -c,  --container-id   Container id\n");
        printf("  -p,  --proxy-sock-fd  File descriptor of the socket connected to cc-proxy\n");
        printf("  -o,  --proxy-io-fd    File descriptor of I/0 fd sent by the cc-proxy\n");
        printf("  -s,  --seq-no         Sequence no for stdin and stdout\n");
        printf("  -e,  --err-seq-no     Sequence no for stderr\n");
        printf("  -d,  --debug          Enable debug output\n");
        printf("  -h,  --help           Display this help message\n");
}

int
main(int argc, char **argv)
{
	struct cc_shim shim = {
		.container_id   =  NULL,
		.proxy_sock_fd  = -1,
		.proxy_io_fd    = -1,
		.io_seq_no      =  0,
		.err_seq_no     =  0,
		.exiting        =  false,
	};
	int                ret;
	struct sigaction   sa;
	int                c;
	bool               debug = false;
	long long          val;

	program_name = argv[0];

	struct option prog_opts[] = {
		{"container-id", required_argument, 0, 'c'},
		{"proxy-sock-fd", required_argument, 0, 'p'},
		{"proxy-io-fd", required_argument, 0, 'o'},
		{"seq-no", required_argument, 0, 's'},
		{"err-seq-no", required_argument, 0, 'e'},
		{"debug", no_argument, 0, 'd'},
		{"help", no_argument, 0, 'h'},
		{ 0, 0, 0, 0},
	};

	while ((c = getopt_long(argc, argv, "c:p:o:s:e:dh", prog_opts, NULL))!= -1) {
		switch (c) {
			case 'c':
				shim.container_id = strdup(optarg);
				break;
			case 'p':
				shim.proxy_sock_fd = (int)parse_numeric_option(optarg);
				if (shim.proxy_sock_fd < 0) {
					err_exit("Invalid value for proxy socket fd\n");
				}
				break;
			case 'o':
				shim.proxy_io_fd = (int)parse_numeric_option(optarg);
				if (shim.proxy_io_fd < 0) {
					err_exit("Invalid value for proxy IO fd\n");
				}
				break;
			case 's':
				val = parse_numeric_option(optarg);
				if (val == -1) {
					err_exit("Invalid value for I/O seqence no\n");
				}
				shim.io_seq_no = (uint64_t)val;
				break;
			case 'e':
				val = parse_numeric_option(optarg);
				if (val == -1) {
					err_exit("Invalid value for error sequence no\n");
				}
				shim.err_seq_no = (uint64_t)val;
				break;
			case 'd':
				debug = true;
				break;
			case 'h':
				print_usage();
				exit(EXIT_SUCCESS);
			default:
				print_usage();
				exit(EXIT_FAILURE);
		}
	}

	if ( !shim.container_id) {
		err_exit("Missing container id\n");
	}

	if ( shim.proxy_sock_fd == -1) {
		err_exit("Missing proxy socket file descriptor\n");
	}

	if ( shim.proxy_io_fd == -1) {
		err_exit("Missing proxy I/O file descriptor\n");
	}

	if (shim.io_seq_no == 0) {
		err_exit("Missing I/O sequence number\n");
	}

	shim_log_init(debug);

	ret = fcntl(shim.proxy_sock_fd, F_GETFD);
	if (ret == -1) {
		shim_error("Invalid proxy socket connection fd : %s\n", strerror(errno));
		exit(EXIT_FAILURE);
	}

	ret = fcntl(shim.proxy_io_fd, F_GETFD);
	if (ret == -1) {
		shim_error("Invalid proxy I/O fd : %s\n", strerror(errno));
		exit(EXIT_FAILURE);
	}

	/* Using self pipe trick to handle signals in the main loop, other strategy
	 * would be to clock signals and use signalfd()/ to handle signals synchronously
	 */
	if (pipe(signal_pipe_fd) == -1) {
		err_exit("Error creating pipe\n");
	}

	// Add read end of pipe to pollfd list and make it non-bocking
	add_pollfd(poll_fds, SIGNAL_FD_INDEX, signal_pipe_fd[0], POLLIN | POLLPRI);
	if (! set_fd_nonblocking(signal_pipe_fd[0])) {
		exit(EXIT_FAILURE);
	}
	if (! set_fd_nonblocking(signal_pipe_fd[1])) {
		exit(EXIT_FAILURE);
	}

	sigemptyset(&sa.sa_mask);
	sa.sa_flags = SA_RESTART;           /* Restart interrupted reads()s */
	sa.sa_handler = signal_handler;

	// Change the default action of all signals that should be forwarded to proxy
	if (! assign_all_signals(&sa)) {
		err_exit("sigaction");
	}

	add_pollfd(poll_fds, PROXY_IO_INDEX, shim.proxy_io_fd, POLLIN | POLLPRI);

	add_pollfd(poll_fds, PROXY_CTL_INDEX, shim.proxy_sock_fd, POLLIN | POLLPRI);

	/* Add stdin only if it is attached to a terminal.
	 * If we add stdin in the non-interactive case, since stdin is closed by docker
	 * this causes continuous close events to be generated on the poll loop.
	 */
	if (isatty(STDIN_FILENO)) {
		/*
		 * Set raw mode on the slave side of the PTY. The local pty
		 * (ie. the one on the host side) is configured in raw mode to
		 * be a dumb pipe and forward data to the pty on the VM side.
		 */
		struct termios term_settings;

		tcgetattr(STDIN_FILENO, &term_settings);
		saved_term_settings = calloc(1, sizeof(struct termios));
		if ( !saved_term_settings) {
			abort();
		}
		*saved_term_settings = term_settings;
		cfmakeraw(&term_settings);
		tcsetattr(STDIN_FILENO, TCSAFLUSH, &term_settings);

		add_pollfd(poll_fds, STDIN_INDEX, STDIN_FILENO, POLLIN | POLLPRI);
	} else if (fcntl(STDIN_FILENO, F_GETFD) != -1) {
		set_fd_nonblocking(STDIN_FILENO);
		add_pollfd(poll_fds, STDIN_INDEX, STDIN_FILENO, POLLIN | POLLPRI);
	}

	ret = atexit(restore_terminal);
	if (ret) {
		shim_debug("Could not register function for atexit");
	}

	while (1) {
		ret = poll(poll_fds, MAX_POLL_FDS, -1);
		if (ret == -1 && errno != EINTR) {
			shim_error("Error in poll : %s\n", strerror(errno));
			break;
		}

		/* check if signal was received first */
		if (poll_fds[SIGNAL_FD_INDEX].revents != 0) {
			handle_signals(&shim);
		}

		//check proxy_io_fd
		if (poll_fds[PROXY_IO_INDEX].revents != 0) {
			handle_proxy_output(&shim);
		}

		// check for proxy sockfd
		if (poll_fds[PROXY_CTL_INDEX].revents != 0) {
			handle_proxy_ctl(&shim);
		}

		// check stdin fd
		if (poll_fds[STDIN_INDEX].revents != 0) {
			handle_stdin(&shim);
		}
	}

	free(shim.container_id);
	return 0;
}
