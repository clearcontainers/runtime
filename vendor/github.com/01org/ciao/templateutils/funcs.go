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

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"
)

type tableHeading struct {
	name  string
	index int
}

var sortAscMap = map[reflect.Kind]func(interface{}, interface{}) bool{
	reflect.Int: func(v1, v2 interface{}) bool {
		return v1.(int) < v2.(int)
	},
	reflect.Int8: func(v1, v2 interface{}) bool {
		return v1.(int8) < v2.(int8)
	},
	reflect.Int16: func(v1, v2 interface{}) bool {
		return v1.(int16) < v2.(int16)
	},
	reflect.Int32: func(v1, v2 interface{}) bool {
		return v1.(int32) < v2.(int32)
	},
	reflect.Int64: func(v1, v2 interface{}) bool {
		return v1.(int64) < v2.(int64)
	},
	reflect.Uint: func(v1, v2 interface{}) bool {
		return v1.(uint) < v2.(uint)
	},
	reflect.Uint8: func(v1, v2 interface{}) bool {
		return v1.(uint8) < v2.(uint8)
	},
	reflect.Uint16: func(v1, v2 interface{}) bool {
		return v1.(uint16) < v2.(uint16)
	},
	reflect.Uint32: func(v1, v2 interface{}) bool {
		return v1.(uint32) < v2.(uint32)
	},
	reflect.Uint64: func(v1, v2 interface{}) bool {
		return v1.(uint64) < v2.(uint64)
	},
	reflect.Float64: func(v1, v2 interface{}) bool {
		return v1.(float64) < v2.(float64)
	},
	reflect.Float32: func(v1, v2 interface{}) bool {
		return v1.(float32) < v2.(float32)
	},
	reflect.String: func(v1, v2 interface{}) bool {
		return v1.(string) < v2.(string)
	},
}

var sortDscMap = map[reflect.Kind]func(interface{}, interface{}) bool{
	reflect.Int: func(v1, v2 interface{}) bool {
		return v2.(int) < v1.(int)
	},
	reflect.Int8: func(v1, v2 interface{}) bool {
		return v2.(int8) < v1.(int8)
	},
	reflect.Int16: func(v1, v2 interface{}) bool {
		return v2.(int16) < v1.(int16)
	},
	reflect.Int32: func(v1, v2 interface{}) bool {
		return v2.(int32) < v1.(int32)
	},
	reflect.Int64: func(v1, v2 interface{}) bool {
		return v2.(int64) < v1.(int64)
	},
	reflect.Uint: func(v1, v2 interface{}) bool {
		return v2.(uint) < v1.(uint)
	},
	reflect.Uint8: func(v1, v2 interface{}) bool {
		return v2.(uint8) < v1.(uint8)
	},
	reflect.Uint16: func(v1, v2 interface{}) bool {
		return v2.(uint16) < v1.(uint16)
	},
	reflect.Uint32: func(v1, v2 interface{}) bool {
		return v2.(uint32) < v1.(uint32)
	},
	reflect.Uint64: func(v1, v2 interface{}) bool {
		return v2.(uint64) < v1.(uint64)
	},
	reflect.Float64: func(v1, v2 interface{}) bool {
		return v2.(float64) < v1.(float64)
	},
	reflect.Float32: func(v1, v2 interface{}) bool {
		return v2.(float32) < v1.(float32)
	},
	reflect.String: func(v1, v2 interface{}) bool {
		return v2.(string) < v1.(string)
	},
}

type valueSorter struct {
	val   reflect.Value
	field int
	less  func(v1, v2 interface{}) bool
}

func (v *valueSorter) Len() int {
	return v.val.Len()
}

func (v *valueSorter) Less(i, j int) bool {
	iVal := v.val.Index(i)
	jVal := v.val.Index(j)
	return v.less(iVal.Field(v.field).Interface(), jVal.Field(v.field).Interface())
}

func (v *valueSorter) Swap(i, j int) {
	iVal := v.val.Index(i).Interface()
	jVal := v.val.Index(j).Interface()
	v.val.Index(i).Set(reflect.ValueOf(jVal))
	v.val.Index(j).Set(reflect.ValueOf(iVal))
}

func getValue(obj interface{}) reflect.Value {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	return val
}

func fatalf(name, format string, args ...interface{}) {
	panic(template.ExecError{
		Name: name,
		Err:  fmt.Errorf(format, args...),
	})
}

func newValueSorter(obj interface{}, field string, ascending bool) *valueSorter {
	val := reflect.ValueOf(obj)
	typ := reflect.TypeOf(obj)
	sTyp := typ.Elem()

	var index int
	var fTyp reflect.StructField
	for index = 0; index < sTyp.NumField(); index++ {
		fTyp = sTyp.Field(index)
		if fTyp.Name == field {
			break
		}
	}
	if index == sTyp.NumField() {
		fatalf("sort", "%s is not a valid field name", field)
	}
	fKind := fTyp.Type.Kind()

	var lessFn func(interface{}, interface{}) bool
	if ascending {
		lessFn = sortAscMap[fKind]
	} else {
		lessFn = sortDscMap[fKind]
	}
	if lessFn == nil {
		var stringer *fmt.Stringer
		if !fTyp.Type.Implements(reflect.TypeOf(stringer).Elem()) {
			fatalf("sort", "cannot sort fields of type %s", fKind)
		}
		lessFn = func(v1, v2 interface{}) bool {
			return v1.(fmt.Stringer).String() < v2.(fmt.Stringer).String()
		}
	}
	return &valueSorter{
		val:   val,
		field: index,
		less:  lessFn,
	}
}

func findField(fieldPath []string, v reflect.Value) reflect.Value {
	f := v
	for _, seg := range fieldPath {
		f = f.FieldByName(seg)
		if f.Kind() == reflect.Ptr {
			f = reflect.Indirect(f)
		}
	}
	return f
}

func filterField(fnName string, obj interface{}, field, val string, cmp func(string, string) bool) interface{} {
	defer func() {
		err := recover()
		if err != nil {
			fatalf(fnName, "Invalid use of filter: %v", err)
		}
	}()

	list := getValue(obj)
	filtered := reflect.MakeSlice(list.Type(), 0, list.Len())

	fieldPath := strings.Split(field, ".")

	for i := 0; i < list.Len(); i++ {
		v := list.Index(i)
		if v.Kind() == reflect.Ptr {
			v = reflect.Indirect(v)
		}

		f := findField(fieldPath, v)

		strVal := fmt.Sprintf("%v", f.Interface())
		if cmp(strVal, val) {
			filtered = reflect.Append(filtered, list.Index(i))
		}
	}

	return filtered.Interface()
}

func filterByField(obj interface{}, field, val string) interface{} {
	return filterField("filter", obj, field, val, func(a, b string) bool {
		return a == b
	})
}

func filterByContains(obj interface{}, field, val string) interface{} {
	return filterField("filterContains", obj, field, val, strings.Contains)
}

func filterByFolded(obj interface{}, field, val string) interface{} {
	return filterField("filterFolded", obj, field, val, strings.EqualFold)
}

func filterByHasPrefix(obj interface{}, field, val string) interface{} {
	return filterField("filterHasPrefix", obj, field, val, strings.HasPrefix)
}

func filterByHasSuffix(obj interface{}, field, val string) interface{} {
	return filterField("filterHasSuffix", obj, field, val, strings.HasSuffix)
}

func filterByRegexp(obj interface{}, field, val string) interface{} {
	return filterField("filterRegexp", obj, field, val, func(a, b string) bool {
		matched, err := regexp.MatchString(b, a)
		if err != nil {
			fatalf("filter", "Invalid regexp: %v", err)
		}
		return matched
	})
}

func selectField(obj interface{}, field string) string {
	defer func() {
		err := recover()
		if err != nil {
			fatalf("select", "Invalid use of select: %v", err)
		}
	}()

	var b bytes.Buffer
	list := getValue(obj)

	fieldPath := strings.Split(field, ".")

	for i := 0; i < list.Len(); i++ {
		v := list.Index(i)
		if v.Kind() == reflect.Ptr {
			v = reflect.Indirect(v)
		}

		f := findField(fieldPath, v)

		fmt.Fprintf(&b, "%v\n", f.Interface())
	}

	return string(b.Bytes())
}

func toJSON(obj interface{}) string {
	b, err := json.MarshalIndent(obj, "", "\t")
	if err != nil {
		return ""
	}
	return string(b)
}

func assertCollectionOfStructs(fnName string, v reflect.Value) {
	typ := v.Type()
	kind := typ.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		fatalf(fnName, "slice or an array of structs expected")
	}
	styp := typ.Elem()
	if styp.Kind() != reflect.Struct {
		fatalf(fnName, "slice or an array of structs expected")
	}
}

func getTableHeadings(fnName string, v reflect.Value) []tableHeading {
	assertCollectionOfStructs(fnName, v)

	typ := v.Type()
	styp := typ.Elem()

	var headings []tableHeading
	for i := 0; i < styp.NumField(); i++ {
		field := styp.Field(i)
		if field.PkgPath != "" || ignoreKind(field.Type.Kind()) {
			continue
		}
		headings = append(headings, tableHeading{name: field.Name, index: i})
	}

	if len(headings) == 0 {
		fatalf(fnName, "structures must contain at least one exported non-channel field")
	}
	return headings
}

func createTable(v reflect.Value, minWidth, tabWidth, padding int, headings []tableHeading) string {
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, minWidth, tabWidth, padding, ' ', 0)
	for _, h := range headings {
		fmt.Fprintf(w, "%s\t", h.name)
	}
	fmt.Fprintln(w)

	for i := 0; i < v.Len(); i++ {
		el := v.Index(i)
		for _, h := range headings {
			fmt.Fprintf(w, "%v\t", el.Field(h.index).Interface())
		}
		fmt.Fprintln(w)
	}
	_ = w.Flush()

	return b.String()
}

func table(obj interface{}) string {
	val := getValue(obj)
	return createTable(val, 8, 8, 1, getTableHeadings("table", val))
}

func tablex(obj interface{}, minWidth, tabWidth, padding int, userHeadings ...string) string {
	val := getValue(obj)
	headings := getTableHeadings("tablex", val)
	if len(headings) < len(userHeadings) {
		fatalf("tablex", "Too many headings specified.  Max permitted %d got %d",
			len(headings), len(userHeadings))
	}
	for i := range userHeadings {
		headings[i].name = userHeadings[i]
	}
	return createTable(val, minWidth, tabWidth, padding, headings)
}

func cols(obj interface{}, fields ...string) interface{} {
	val := getValue(obj)
	assertCollectionOfStructs("cols", val)
	if len(fields) == 0 {
		fatalf("cols", "at least one column name must be specified")
	}

	var newFields []reflect.StructField
	var indicies []int
	styp := val.Type().Elem()
	for i := 0; i < styp.NumField(); i++ {
		field := styp.Field(i)
		if field.PkgPath != "" || ignoreKind(field.Type.Kind()) {
			continue
		}

		var j int
		for j = 0; j < len(fields); j++ {
			if fields[j] == field.Name {
				break
			}
		}
		if j == len(fields) {
			continue
		}

		indicies = append(indicies, i)
		newFields = append(newFields, field)
	}

	if len(indicies) != len(fields) {
		fatalf("cols", "not all column names are valid")
	}

	newStyp := reflect.StructOf(newFields)
	newVal := reflect.MakeSlice(reflect.SliceOf(newStyp), val.Len(), val.Len())
	for i := 0; i < val.Len(); i++ {
		sval := val.Index(i)
		newSval := reflect.New(newStyp).Elem()
		for j, origIndex := range indicies {
			newSval.Field(j).Set(sval.Field(origIndex))
		}
		newVal.Index(i).Set(newSval)
	}

	return newVal.Interface()
}

func sortSlice(obj interface{}, field string, direction ...string) interface{} {
	ascending := true
	if len(direction) > 1 {
		fatalf("sort", "Too many parameters passed to sort")
	} else if len(direction) == 1 {
		if direction[0] == "dsc" {
			ascending = false
		} else if direction[0] != "asc" {
			fatalf("sort", "direction parameter must be \"asc\" or \"dsc\"")
		}
	}

	val := getValue(obj)
	assertCollectionOfStructs("sort", val)

	copy := reflect.MakeSlice(reflect.SliceOf(val.Type().Elem()), 0, val.Len())
	for i := 0; i < val.Len(); i++ {
		copy = reflect.Append(copy, val.Index(i))
	}

	newobj := copy.Interface()
	vs := newValueSorter(newobj, field, ascending)
	sort.Sort(vs)
	return newobj
}

func rows(obj interface{}, rows ...int) interface{} {
	val := getValue(obj)
	typ := val.Type()
	kind := typ.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		fatalf("rows", "slice or an array of expected")
	}

	if len(rows) == 0 {
		fatalf("rows", "at least one row index must be specified")
	}

	copy := reflect.MakeSlice(reflect.SliceOf(val.Type().Elem()), 0, len(rows))
	for _, row := range rows {
		if row < val.Len() {
			copy = reflect.Append(copy, val.Index(row))
		}
	}

	return copy.Interface()
}

func assertSliceAndRetrieveCount(fnName string, obj interface{}, count ...int) (reflect.Value, int) {
	val := getValue(obj)
	typ := val.Type()
	kind := typ.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		fatalf(fnName, "slice or an array of expected")
	}

	rows := 1
	if len(count) == 1 {
		rows = count[0]
	} else if len(count) > 1 {
		fatalf(fnName, "accepts a maximum of two arguments expected")
	}

	return val, rows
}

func head(obj interface{}, count ...int) interface{} {
	val, rows := assertSliceAndRetrieveCount("head", obj, count...)
	copy := reflect.MakeSlice(reflect.SliceOf(val.Type().Elem()), 0, rows)
	for i := 0; i < rows && i < val.Len(); i++ {
		copy = reflect.Append(copy, val.Index(i))
	}

	return copy.Interface()
}

func tail(obj interface{}, count ...int) interface{} {
	val, rows := assertSliceAndRetrieveCount("tail", obj, count...)
	copy := reflect.MakeSlice(reflect.SliceOf(val.Type().Elem()), 0, rows)
	start := val.Len() - rows
	if start < 0 {
		start = 0
	}
	for i := start; i < val.Len(); i++ {
		copy = reflect.Append(copy, val.Index(i))
	}

	return copy.Interface()
}

func describe(obj interface{}) string {
	var buf bytes.Buffer
	generateIndentedUsage(&buf, obj)
	return buf.String()
}
