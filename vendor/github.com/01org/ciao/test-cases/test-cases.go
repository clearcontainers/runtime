//
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
//

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"syscall"
	"text/tabwriter"
	"text/template"
	"time"
)

// PackageInfo contains information about a package under test.
type PackageInfo struct {
	// Name is the name of the package.
	Name string `json:"name"`

	// Path is the import path of the package.
	Path string `json:"path"`

	// File contains a list of all the test files associated with a package.
	Files []string `json:"files"`

	// XFiles contains a list of all the external test files associated with
	// a package.
	XFiles []string `json:"xfiles"`
}

// TestInfo contains information about an individual executed test case.
type TestInfo struct {
	// Name is the name of the test case.
	Name string

	// Summary provides a brief one line description of the test case.
	Summary string

	// Description contains a detailed description of the test case.
	Description string

	// ExpectedResult outlines the success criteria for the test case.
	ExpectedResult string

	// Pass indicates whether the test case passed or not.
	Pass bool

	// Result provides the result of the test case.
	Result string

	// TimeTaken is a description of the time taken to run the test case.
	TimeTaken string

	logs []string
}

// PackageTests contains information about the tests that have been executed for
// an individual package
type PackageTests struct {
	// Name is the name of the package.
	Name string

	// Coverage is the amount of package code covered by the test cases.
	Coverage string

	// Tests is an array containing information about the specific test cases.
	Tests []*TestInfo
}

type testResults struct {
	result    string
	timeTaken string
	logs      []string
}

type colouredRow struct {
	ansiSeq string
	columns []string
}

const goListTemplate = `{
"name" : "{{.ImportPath}}",
"path" : "{{.Dir}}",
"files" : [ {{range $index, $elem := .TestGoFiles }}{{if $index}}, "{{$elem}}"{{else}}"{{$elem}}"{{end}}{{end}} ],
"xfiles" : [ {{range $index, $elem := .XTestGoFiles }}{{if $index}}, "{{$elem}}"{{else}}"{{$elem}}"{{end}}{{end}} ]
},
`

const htmlTemplate = `
<html>
<head>
<title>Test Cases</title>
<style type="text/css">
{{.CSS}}
</style>
</head>
<body>
{{range .Tests}}
<h1>{{.Name}}</h1>
<p><i>Coverage: {{.Coverage}}</i></p>
<table style="table-layout:fixed" border="1">
<tr><th style="width:10%">Name</th><th style="width:20%">Summary</th><th style="width:30%">Description</th><th style="width:20%">ExpectedResult</th><th style="width:10%">Result</th><th style="width:10%">Time Taken</th></tr>
{{range .Tests}}
<tr {{if .Pass}}style="color: green"{{else}}style="color: red"{{end}}><td>{{.Name}}</td><td>{{.Summary}}</td><td>{{.Description}}</td><td>{{.ExpectedResult}}</td><td>{{.Result}}</td><td>{{.TimeTaken}}</td></tr>
{{end}}
</table>
{{end}}
</body>
</html>
`

var newLineRegexp = regexp.MustCompile(`(\r\n)|[\n\r]`)

type formatType string

const (
	formatText       formatType = "text"
	formatColourText            = "colour-text"
	formatHTML                  = "html"
	formatTAP                   = "tap"
)

func (f *formatType) String() string {
	return string(*f)
}
func (f *formatType) Set(val string) error {
	v := formatType(val)
	if v != formatText && v != formatColourText && v != formatHTML && v != formatTAP {
		return fmt.Errorf("invalid format;  %s, %s, %s, %s expected",
			formatText, formatColourText, formatHTML, formatTAP)
	}
	*f = formatType(val)
	return nil
}

var resultRegexp *regexp.Regexp
var coverageRegexp *regexp.Regexp

var cssPath string
var short bool
var race bool
var tags string
var coverProfile string
var appendProfile bool
var format formatType = formatColourText

var timeout int
var verbose bool

func init() {
	flag.StringVar(&cssPath, "css", "", "Full path to CSS file")
	flag.BoolVar(&short, "short", false, "If true -short is passed to go test")
	flag.BoolVar(&race, "race", false, "If true -race is passed to go test")
	flag.StringVar(&tags, "tags", "", "Build tags to pass to go test")
	flag.StringVar(&coverProfile, "coverprofile", "", "Path of coverage profile to be generated")
	flag.BoolVar(&appendProfile, "append-profile", false, "Append generated coverage profiles an existing file")
	flag.BoolVar(&verbose, "v", false, "Output package names under test if true")
	flag.IntVar(&timeout, "timeout", 0, "Time in minutes after which a package's unit tests should time out.  0 = no timeout")
	flag.Var(&format, "format", fmt.Sprintf("Specify output format. Can be '%s', '%s', '%s', or '%s'",
		formatText, formatColourText, formatHTML, formatTAP))

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] [packages]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "[packages] is a list of go packages.  The list can include the ./... wildcard.")
		fmt.Fprintln(os.Stderr, "If no packages are specified the unit tests will be run for the go package")
		fmt.Fprintln(os.Stderr, "located in the current directory.")
		fmt.Fprintln(os.Stderr, "\nSupported flags:")
		flag.PrintDefaults()
	}

	resultRegexp = regexp.MustCompile(`--- (FAIL|PASS|SKIP): ([^\s]+) \(([^\)]+)\)`)
	coverageRegexp = regexp.MustCompile(`^coverage: ([^\s]+)`)
}

func parseCommentGroup(ti *TestInfo, comment string) {
	groups := regexp.MustCompile("\n\n").Split(comment, 4)
	fields := []*string{&ti.Summary, &ti.Description, &ti.ExpectedResult}
	for i, c := range groups {
		*fields[i] = c
	}
}

func isTestingFunc(decl *ast.FuncDecl) bool {
	if !strings.HasPrefix(decl.Name.String(), "Test") {
		return false
	}

	paramList := decl.Type.Params.List
	if len(paramList) != 1 {
		return false
	}

	recType, ok := paramList[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	pt, ok := recType.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	id, ok := pt.X.(*ast.Ident)
	if !ok {
		return false
	}

	return id.Name == "testing" && pt.Sel.Name == "T"
}

func parseTestFile(filePath string) ([]*TestInfo, error) {
	tests := make([]*TestInfo, 0, 32)
	fs := token.NewFileSet()
	tr, err := parser.ParseFile(fs, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, decl := range tr.Decls {
		if decl, ok := decl.(*ast.FuncDecl); ok {
			if !isTestingFunc(decl) {
				continue
			}

			ti := &TestInfo{Name: decl.Name.String()}
			tests = append(tests, ti)

			if decl.Doc == nil {
				continue
			}

			parseCommentGroup(ti, decl.Doc.Text())
		}
	}

	return tests, nil
}

func extractTests(packages []PackageInfo) []*PackageTests {
	pts := make([]*PackageTests, 0, len(packages))
	for _, p := range packages {
		if ((len(p.Files) == 0) && (len(p.XFiles) == 0)) ||
			strings.Contains(p.Name, "/vendor/") {
			continue
		}
		packageTest := &PackageTests{
			Name: p.Name,
		}

		files := make([]string, 0, len(p.Files)+len(p.XFiles))
		files = append(files, p.Files...)
		files = append(files, p.XFiles...)
		for _, f := range files {
			filePath := path.Join(p.Path, f)
			ti, err := parseTestFile(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse %s: %s\n",
					filePath, err)
				continue
			}
			packageTest.Tests = append(packageTest.Tests, ti...)
		}
		pts = append(pts, packageTest)
	}
	return pts
}

func findTestFiles(packs []string) ([]PackageInfo, error) {
	var output bytes.Buffer
	fmt.Fprintln(&output, "[")
	listArgs := []string{"list", "-f", goListTemplate}
	listArgs = append(listArgs, packs...)
	cmd := exec.Command("go", listArgs...)
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	lastComma := bytes.LastIndex(output.Bytes(), []byte{','})
	if lastComma != -1 {
		output.Truncate(lastComma)
	}
	fmt.Fprintln(&output, "]")
	var testPackages []PackageInfo
	err = json.Unmarshal(output.Bytes(), &testPackages)
	if err != nil {
		return nil, err
	}
	return testPackages, nil
}

func dumpErrorOutput(errorOutput *bytes.Buffer) {
	fmt.Fprintln(os.Stderr, "Output from stderr")
	fmt.Fprintln(os.Stderr, "------------------")
	scanner := bufio.NewScanner(errorOutput)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(os.Stderr, line)
	}
}

func dumpColourErrorOutput(errorOutput *bytes.Buffer) {
	fmt.Fprintln(os.Stderr, "Output from stderr")
	fmt.Fprintln(os.Stderr, "------------------")
	scanner := bufio.NewScanner(errorOutput)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(os.Stderr, "%c[%dm", 0x1b, 31)
		fmt.Fprintln(os.Stderr, line)
	}
	fmt.Fprintf(os.Stderr, "%c[%dm\n", 0x1b, 0)
}

func parseTestOutput(output bytes.Buffer, results map[string]*testResults) string {
	var coverage string

	scanner := bufio.NewScanner(&output)
	key := ""
	for scanner.Scan() {
		line := scanner.Text()

		stripped := strings.TrimSpace(line)
		if strings.HasPrefix(stripped, "PASS") || strings.HasPrefix(stripped, "FAIL") ||
			strings.HasPrefix(stripped, "=== RUN") {
			key = ""
			continue
		}

		matches := resultRegexp.FindStringSubmatch(line)
		if matches != nil && len(matches) == 4 {
			key = matches[2]
			results[key] = &testResults{
				result:    matches[1],
				timeTaken: matches[3],
				logs:      make([]string, 0, 16)}
			continue
		}

		if key != "" {
			results[key].logs = append(results[key].logs, line)
		}

		if coverage == "" {
			matches := coverageRegexp.FindStringSubmatch(line)
			if matches == nil || len(matches) != 2 {
				continue
			}
			coverage = matches[1]
		}
	}
	return coverage
}

func runCommandWithTimeout(pkg string, cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		return err
	}

	errCh := make(chan error)
	go func() {
		errCh <- cmd.Wait()
	}()

	select {
	case <-time.After(time.Minute * time.Duration(timeout)):
		if verbose {
			fmt.Printf("Aborting %s\n", pkg)
		}
		syscall.Kill(-cmd.Process.Pid, syscall.SIGABRT)
		err = <-errCh
	case err = <-errCh:
	}

	return err
}

func runPackageTests(p *PackageTests, coverFile string, errorOutput *bytes.Buffer) (int, error) {
	var output bytes.Buffer

	if verbose {
		fmt.Printf("Testing %s\n", p.Name)
	}

	exitCode := 0
	results := make(map[string]*testResults)
	args := []string{"test", p.Name, "-v", "-cover"}
	if short {
		args = append(args, "-short")
	}
	if race {
		args = append(args, "-race")
	}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	if coverFile != "" {
		args = append(args, "-coverprofile", coverFile)
	}
	cmd := exec.Command("go", args...)
	cmd.Stdout = &output
	cmd.Stderr = errorOutput

	var err error
	if timeout == 0 {
		err = cmd.Run()
	} else {
		err = runCommandWithTimeout(p.Name, cmd)
	}

	coverage := parseTestOutput(output, results)

	for _, t := range p.Tests {
		res := results[t.Name]
		if res == nil {
			t.Result = "NOT RUN"
			t.TimeTaken = "N/A"
			exitCode = 1
		} else {
			t.Result = res.result
			t.Pass = (res.result == "PASS" || res.result == "SKIP")
			if !t.Pass {
				exitCode = 1
			}
			t.TimeTaken = res.timeTaken
			t.logs = res.logs
		}
	}

	if coverage != "" {
		p.Coverage = coverage
	} else {
		p.Coverage = "Unknown"
	}

	return exitCode, err
}

func generateHTMLReport(tests []*PackageTests) error {
	var css string
	if cssPath != "" {
		cssBytes, err := ioutil.ReadFile(cssPath)
		if err != nil {
			log.Printf("Unable to read css file %s : %v",
				cssPath, err)
		} else {
			css = string(cssBytes)
		}
	}

	tmpl, err := template.New("tests").Parse(htmlTemplate)
	if err != nil {
		log.Fatalf("Unable to parse html template: %s\n", err)
	}

	return tmpl.Execute(os.Stdout, &struct {
		Tests []*PackageTests
		CSS   string
	}{
		tests,
		css,
	})
}

func findCommonPrefix(tests []*PackageTests) string {
	for j := range tests {
		pkgName := tests[j].Name
		index := strings.LastIndex(pkgName, "/")
		if index == -1 {
			return ""
		}
		pkgRoot := pkgName[:index+1]

		var i int
		for i = 0; i < len(tests); i++ {
			if i == j {
				continue
			}
			if !strings.HasPrefix(tests[i].Name, pkgRoot) {
				break
			}
		}

		if i == len(tests) {
			return pkgRoot
		}
	}

	return ""
}

func dumpFailedTestOutput(prefix string, tests []*PackageTests, colourOn, colourOff string) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Logs for failed tests")
	fmt.Fprintln(os.Stderr, "---------------------")
	for _, p := range tests {
		for _, t := range p.Tests {
			if !t.Pass && len(t.logs) > 0 {
				fmt.Fprintf(os.Stderr, "%s", colourOff)
				pkgName := p.Name[len(prefix):]
				fmt.Fprintf(os.Stderr, "Logs for %s.%s\n", pkgName, t.Name)
				for _, s := range t.logs {
					fmt.Fprintf(os.Stderr, "%s%s\n", colourOn, s)
				}
			}
		}
	}
}

func generateColourTextReport(tests []*PackageTests, exitCode int) {
	colourOn := fmt.Sprintf("%c[%dm", 0x1b, 31)
	colourOff := fmt.Sprintf("%c[%dm", 0x1b, 0)
	prefix := findCommonPrefix(tests)
	table := make([]colouredRow, 0, 128)
	table = append(table, colouredRow{
		"",
		[]string{"Package", "Test Case", "Time Taken", "Result"},
	})
	colWidth := []int{0, 0, 0, 0}
	for i := range colWidth {
		colWidth[i] = len(table[0].columns[i])
	}

	coloured := false
	for _, p := range tests {
		pkgName := p.Name[len(prefix):]
		for _, t := range p.Tests {
			row := colouredRow{}
			if !t.Pass {
				row.ansiSeq = colourOn
				coloured = true
			} else if t.Pass && coloured {
				coloured = false
				row.ansiSeq = colourOff
			}
			row.columns = []string{pkgName, t.Name, t.TimeTaken, t.Result}
			for i := range colWidth {
				if colWidth[i] < len(row.columns[i]) {
					colWidth[i] = len(row.columns[i])
				}
			}
			table = append(table, row)
		}
	}

	for _, row := range table {
		fmt.Printf("%s", row.ansiSeq)
		for i, col := range row.columns {
			fmt.Printf(col)
			fmt.Printf("%s", strings.Repeat(" ", colWidth[i]-len(col)))
			fmt.Printf(" ")
		}
		fmt.Println("")
	}

	if coloured {
		fmt.Printf("%c[%dm\n", 0x1b, 0)
	}

	if exitCode != 0 {
		dumpFailedTestOutput(prefix, tests, colourOn, colourOff)
		fmt.Println(colourOff)
	}
}

func generateTextReport(tests []*PackageTests, exitCode int) {
	prefix := findCommonPrefix(tests)
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, ' ', 0)
	fmt.Fprintln(w, "Package\tTest Case\tTime Taken\tResult\t")
	for _, p := range tests {
		pkgName := p.Name[len(prefix):]
		for _, t := range p.Tests {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", pkgName,
				t.Name, t.TimeTaken, t.Result)
		}
	}
	_ = w.Flush()
	if exitCode != 0 {
		dumpFailedTestOutput(prefix, tests, "", "")
		fmt.Println()
	}
}

func createCoverFile() (*os.File, error) {
	var f *os.File
	var err error
	if appendProfile {
		f, err = os.OpenFile(coverProfile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("Unable to open %s for appending: %v",
				coverProfile, err)
		}
	} else {
		f, err = os.OpenFile(coverProfile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("Unable to create coverage file %s: %v",
				coverProfile, err)
		}
		_, err = f.WriteString("mode: set\n")
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("Unable to write mode string to coverage file %s: %v",
				coverProfile, err)
		}
	}

	return f, nil
}

func appendCoverageData(f *os.File, coverFile string) error {
	cover, err := ioutil.ReadFile(coverFile)
	if err != nil {
		return fmt.Errorf("Unable to read coverage file %s: %v", coverFile, err)
	}

	index := bytes.Index(cover, []byte{'\n'})
	if index != -1 {
		cover = cover[index+1:]
	}

	_, err = f.Write(cover)
	if err != nil {
		return fmt.Errorf("Unable to append coverage data to %s: %v", coverFile, err)
	}

	return nil
}

func runTests(tests []*PackageTests) (int, *bytes.Buffer, error) {
	var errorOutput bytes.Buffer
	exitCode := 0
	if coverProfile != "" {
		coverDir, err := ioutil.TempDir("", "cover-profiles")
		if err != nil {
			err = fmt.Errorf("Unable to create temporary directory for coverage profiles: %v", err)
			return 1, &errorOutput, err

		}
		defer func() { _ = os.RemoveAll(coverDir) }()

		f, err := createCoverFile()
		if err != nil {
			return 1, &errorOutput, err
		}
		defer func() { _ = f.Close() }()

		for i, p := range tests {
			coverFile := path.Join(coverDir, fmt.Sprintf("%d", i))
			ec, err := runPackageTests(p, coverFile, &errorOutput)
			exitCode |= ec
			if err != nil {
				continue
			}
			err = appendCoverageData(f, coverFile)
			if err != nil {
				return 1, &errorOutput, err
			}
		}
	} else {
		for _, p := range tests {
			ec, _ := runPackageTests(p, "", &errorOutput)
			exitCode |= ec
		}
	}

	return exitCode, &errorOutput, nil
}

func generateTAPOutput(tests []*PackageTests) {
	i := 0
	prefix := findCommonPrefix(tests)
	for _, p := range tests {
		pkgName := p.Name[len(prefix):]
		fmt.Printf("# Tests for %s\n", pkgName)
		for _, t := range p.Tests {
			if t.Result == "PASS" {
				fmt.Printf("ok ")
			} else {
				fmt.Printf("not ok ")
			}
			testName := strings.TrimSpace(t.Summary)
			if testName == "" {
				testName = t.Name
			} else {
				testName = newLineRegexp.ReplaceAllString(testName, " ")
			}
			fmt.Printf("%d - %s\n", i+1, testName)
			i++
		}
	}
	if i > 0 {
		fmt.Printf("1..%d\n", i)
	}
}

func main() {

	flag.Parse()

	packages, err := findTestFiles(flag.Args())
	if err != nil {
		log.Fatalf("Unable to discover test files: %s", err)
	}

	tests := extractTests(packages)
	exitCode, errorOutput, err := runTests(tests)
	if err != nil {
		log.Fatal(err)
	}

	switch format {
	case formatText:
		generateTextReport(tests, exitCode)
		if exitCode != 0 {
			dumpErrorOutput(errorOutput)
		}
	case formatColourText:
		generateColourTextReport(tests, exitCode)
		if exitCode != 0 {
			dumpColourErrorOutput(errorOutput)
		}
	case formatTAP:
		generateTAPOutput(tests)
	case formatHTML:
		err := generateHTMLReport(tests)
		if err != nil {
			log.Fatalf("Unable to generate report: %s\n", err)
		}
	}

	os.Exit(exitCode)
}
