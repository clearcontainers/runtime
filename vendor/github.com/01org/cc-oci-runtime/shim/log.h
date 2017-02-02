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

#include <syslog.h>
#include <stdbool.h>

void shim_log_init(bool debug);

void shim_log(int priority, 
		const char *func,
		int line_number,
		const char *format, ...);

/*
 *   Acceptable values for priority:
 *
 *  LOG_EMERG      system is unusable -> 0
 *  LOG_ALERT      action must be taken immediately -> 1
 *  LOG_CRITICAL   critical conditions
 *  LOG_ERR        error conditions
 *  LOG_WARNING    warning conditions
 *  LOG_NOTICE     normal, but significant, condition
 *  LOG_INFO       informational message
 *  LOG_DEBUG      debug-level message -> 7
 */

#define shim_debug(...)     shim_log(LOG_DEBUG,  __func__, __LINE__,  __VA_ARGS__)
#define shim_info(...)      shim_log(LOG_INFO,  __func__, __LINE__,  __VA_ARGS__)
#define shim_warning(...)   shim_log(LOG_WARNING,  __func__, __LINE__, __VA_ARGS__)
#define shim_error(...)     shim_log(LOG_ERR,  __func__, __LINE__,  __VA_ARGS__)
#define shim_critical(...)  shim_log(LOG_CRITICAL,  __func__, __LINE__,  __VA_ARGS__)
#define shim_alert(...)     shim_log(LOG_ALERT,  __func__, __LINE__,  __VA_ARGS__)
#define shim_emerg(...)     shim_log(LOG_EMERG,  __func__, __LINE__, __VA_ARGS__)
