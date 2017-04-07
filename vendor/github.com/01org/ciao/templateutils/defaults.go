//
// Copyright (c) 2017 Intel Corporation
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
//

package templateutils

import "text/template"

var funcMap = template.FuncMap{
	"filter":          filterByField,
	"filterContains":  filterByContains,
	"filterHasPrefix": filterByHasPrefix,
	"filterHasSuffix": filterByHasSuffix,
	"filterFolded":    filterByFolded,
	"filterRegexp":    filterByRegexp,
	"tojson":          toJSON,
	"select":          selectField,
	"table":           table,
	"tablex":          tablex,
	"cols":            cols,
	"sort":            sortSlice,
	"rows":            rows,
	"head":            head,
	"tail":            tail,
	"describe":        describe,
}

var funcHelpSlice = []funcHelpInfo{
	{helpFilter, helpFilterIndex},
	{helpFilterContains, helpFilterContainsIndex},
	{helpFilterHasPrefix, helpFilterHasPrefixIndex},
	{helpFilterHasSuffix, helpFilterHasSuffixIndex},
	{helpFilterFolded, helpFilterFoldedIndex},
	{helpFilterRegexp, helpFilterRegexpIndex},
	{helpToJSON, helpToJSONIndex},
	{helpSelect, helpSelectIndex},
	{helpTable, helpTableIndex},
	{helpTableX, helpTableXIndex},
	{helpCols, helpColsIndex},
	{helpSort, helpSortIndex},
	{helpRows, helpRowsIndex},
	{helpHead, helpHeadIndex},
	{helpTail, helpTailIndex},
	{helpDescribe, helpDescribeIndex},
}

func getFuncMap(cfg *Config) template.FuncMap {
	if cfg == nil {
		return funcMap
	}

	return cfg.funcMap
}

func getHelpers(cfg *Config) []funcHelpInfo {
	if cfg == nil {
		return funcHelpSlice
	}

	return cfg.funcHelp
}
