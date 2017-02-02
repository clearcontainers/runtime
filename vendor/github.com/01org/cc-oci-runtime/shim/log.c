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
#include <stdarg.h>
#include <stdlib.h>

#include "log.h"

static bool debug;

/*!
 * Setup logging.
 *
 * \param _debug Bool for logging debug output.
 */
void shim_log_init(bool _debug)
{
	int syslog_options = (LOG_CONS | LOG_PID | LOG_PERROR | LOG_NOWAIT);

	debug = _debug;
	openlog(0, syslog_options, LOG_USER);
}

/*!
 * Log to syslog.
 *
 * \param priority Syslog priority.
 * \param func Function at call site.
 * \param line_number Call site line number.
 * \param format Format and arguments to log.
 */
void shim_log(int priority, const char *func, int line_number, const char *format, ...)
{
	va_list vargs;
	char *buf;

	if (! (format && func)) {
		return;
	}

	if (priority < LOG_EMERG || priority > LOG_DEBUG) {
		return;
	}

	if (priority == LOG_DEBUG && !debug) {
		return;
	}

	va_start(vargs, format);
	if (vasprintf(&buf, format, vargs) == -1) {
		va_end(vargs);
		return;
	}

	if (priority <=  LOG_ERR) {
		fprintf(stderr, "%s:%d:%s\n", func, line_number, buf);
	}

	syslog(priority, "%s:%d:%s", func, line_number, buf);
	va_end(vargs);
	free(buf);
}
