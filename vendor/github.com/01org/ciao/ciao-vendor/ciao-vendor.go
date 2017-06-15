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
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
)

type repoInfo struct {
	URL     string `json:"url"`
	Version string `json:"version"`
	License string `json:"license"`
}

type packageDeps struct {
	p    string
	deps []string
}

type packageInfo struct {
	name      string
	vendored  bool
	installed bool
	CGO       bool `json:"cgo"`
	Standard  bool `json:"standard"`
}

type subPackage struct {
	name     string
	wildcard string
	docs     []string
	cgo      bool
}

type clientInfo struct {
	name string
	err  error
}

type piList []*packageInfo

func (p piList) Len() int {
	return len(p)
}

func (p piList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p piList) Less(i, j int) bool {
	return p[i].name < p[j].name
}

var repos = map[string]repoInfo{}

var listTemplate = `
{{- range .Deps -}}
{{.}}
{{end -}}
`

var directTemplate = `
{{- range .Imports -}}
{{.}}
{{end -}}
`

func getPackageDetails(name string) *packageInfo {
	packageTemplate := `{
  "standard" : {{.Standard}},
  "cgo" : {{if .CFiles}}true{{else}}false{{end}}
}`
	pi := &packageInfo{name: name}
	cmd := exec.Command("go", "list", "-f", packageTemplate, name)
	output, err := cmd.Output()
	if err != nil {
		return pi
	}

	pi.installed = true
	_ = json.Unmarshal(output, pi)

	return pi
}

func getPackageDependencies(packages []string, template string) (map[string]struct{}, error) {
	deps := make(map[string]struct{})
	args := []string{"list", "-f", template}
	args = append(args, packages...)
	var output bytes.Buffer
	cmd := exec.Command("go", args...)
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %v", err)
	}

	scanner := bufio.NewScanner(&output)
	for scanner.Scan() {
		deps[scanner.Text()] = struct{}{}
	}
	return deps, nil
}

func calcDeps(projectRoot string, packages []string) (piList, error) {
	deps, err := getPackageDependencies(packages, listTemplate)
	if err != nil {
		return nil, err
	}

	ch := make(chan *packageInfo)
	for pkg := range deps {
		go func(pkg string) {
			localDep := strings.HasPrefix(pkg, projectRoot)
			vendoredDep := strings.HasPrefix(pkg, path.Join(projectRoot, "vendor"))
			if localDep && !vendoredDep {
				ch <- nil
			} else {
				pd := getPackageDetails(pkg)
				if pd.Standard {
					ch <- nil
				} else {
					pd.vendored = vendoredDep
					ch <- pd
				}
			}
		}(pkg)
	}

	depsAr := make(piList, 0, len(deps))
	for i := 0; i < cap(depsAr); i++ {
		pd := <-ch
		if pd != nil {
			depsAr = append(depsAr, pd)
		}
	}

	sort.Sort(depsAr)
	return depsAr, nil
}

func checkWD() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("Unable to determine cwd: %v", err)
	}
	gopath, _ := os.LookupEnv("GOPATH")
	if gopath == "" {
		return "", "", fmt.Errorf("GOPATH is not set")
	}

	pths := strings.Split(gopath, ":")

	for _, p := range pths {
		if strings.HasPrefix(cwd, path.Join(p, "src")) {
			return cwd, p, nil
		}
	}

	return "", "", fmt.Errorf("ciao-vendor must be run from $GOPATH/src/path/to/project")
}

func copyRepos(cwd, sourceRoot string, subPackages map[string][]*subPackage) error {
	errCh := make(chan error)
	for k, r := range repos {
		go func(k string, URL string) {
			packages, ok := subPackages[k]
			if !ok {
				errCh <- nil
				return
			}

			cmd1 := exec.Command("git", "archive", repos[k].Version)
			cmd1.Dir = path.Join(sourceRoot, k)
			os.MkdirAll(path.Join(cwd, "vendor", k), 0755)
			args := []string{"-xC", path.Join(cwd, "vendor", k), "--wildcards",
				"--no-wildcards-match-slash"}
			for _, a := range packages {
				if a.wildcard != "" {
					args = append(args, a.wildcard+".go")
				}
				if a.cgo {
					args = append(args, a.wildcard+".[ch]")
				}
				args = append(args, a.docs...)
			}
			args = append(args, "--exclude", "*_test.go")
			cmd2 := exec.Command("tar", args...)
			pipe, err := cmd1.StdoutPipe()
			if err != nil {
				errCh <- fmt.Errorf("Unable to retrieve pipe for git command %v: %v", args, err)
				return
			}
			defer func() {
				_ = pipe.Close()
			}()
			cmd2.Stdin = pipe
			err = cmd1.Start()
			if err != nil {
				errCh <- fmt.Errorf("Unable to start git command %v: %v", args, err)
				return
			}
			err = cmd2.Run()
			if err != nil {
				errCh <- fmt.Errorf("Unable to run tar command %v", err)
				return
			}
			errCh <- nil
		}(k, r.URL)
	}

	var err error
	for range repos {
		rcvErr := <-errCh
		if err == nil && rcvErr != nil {
			err = rcvErr
		}
	}

	return err
}

func updateNonVendoredDeps(deps piList, projectRoot string) error {
	fmt.Println("Updating non-vendored dependencies")

	goGot := make(map[string]struct{})
	for _, d := range deps {
		args := []string{"get", "-v"}

		var repoFound string
		for k := range repos {
			if strings.HasPrefix(d.name, k) {
				repoFound = k
				break
			}
		}

		if _, ok := goGot[repoFound]; !ok {
			args = append(args, "-u")
		}
		args = append(args, d.name)
		cmd := exec.Command("go", args...)
		stdout, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		err = cmd.Start()
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

		err = cmd.Wait()
		if err != nil {
			return err
		}

		goGot[repoFound] = struct{}{}
	}

	return nil
}

func getCurrentBranch(repo string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "HEAD")
	cmd.Dir = repo
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(output))
	if !scanner.Scan() {
		return "", fmt.Errorf("Unable to determine current branch of %s",
			repo)
	}
	branch := strings.TrimSpace(scanner.Text())
	const prefix = "refs/heads/"
	if !strings.HasPrefix(branch, prefix) {
		return "", fmt.Errorf("Unable to determine current branch of %s",
			repo)
	}

	return branch[len(prefix):], nil
}

func checkoutVersion(sourceRoot string) {
	for k, v := range repos {
		cmd := exec.Command("git", "checkout", v.Version)
		cmd.Dir = path.Join(sourceRoot, k)
		_ = cmd.Run()
	}
}

func checkoutMaster(sourceRoot string) {
	for k := range repos {
		cmd := exec.Command("git", "checkout", "master")
		cmd.Dir = path.Join(sourceRoot, k)
		_ = cmd.Run()
	}
}

func findDocs(dir, prefix string) ([]string, error) {
	docs := make([]string, 0, 8)
	docGlob := []string{
		"LICENSE*",
		"README*",
		"NOTICE",
		"MAINTAINERS*",
		"PATENTS*",
		"AUTHORS*",
		"CONTRIBUTORS*",
		"VERSION",
	}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && (dir != path) {
			return filepath.SkipDir
		}

		for _, pattern := range docGlob {
			match, err := filepath.Match(pattern, info.Name())
			if err != nil {
				return err
			}

			if match {
				docs = append(docs, filepath.Join(prefix, info.Name()))
				break
			}
		}
		return nil

	})
	if err != nil {
		return nil, err
	}

	return docs, nil
}

func computeSubPackages(deps piList) map[string][]*subPackage {
	subPackages := make(map[string][]*subPackage)
	for _, d := range deps {
		for k := range repos {
			if !strings.HasPrefix(d.name, k) {
				continue
			}
			packages := subPackages[k]

			pkg := d.name[len(k):]
			if pkg == "" {
				packages = append([]*subPackage{{name: k, wildcard: "*", cgo: d.CGO}}, packages...)
			} else if pkg[0] == '/' {
				packages = append(packages, &subPackage{name: d.name, wildcard: pkg[1:] + "/*", cgo: d.CGO})
			} else {
				fmt.Printf("Warning: unvendored package: %s\n", d.name)
			}
			subPackages[k] = packages
			break
		}
	}
	return subPackages
}

// This might look a little convoluted but we can't just go get
// on all the repos in repos, using a wildcard.  This would build
// loads of stuff we're not interested in at best and at worst,
// breakage in a package we're not interested in would break
// ciao-vendor
//
// We can't just go get github.com/01org/ciao this would pull down
// the dependencies of the master version of ciao's depdendencies
// which is not what we want.  This might miss some dependencies
// which have been deleted from the master branch of ciao's
// dependencies.
//
// So we need to figure out which dependencies ciao actually has,
// pull them down, check out the version of these dependencies
// that ciao actually uses, and then recompute our dependencies.
//
// Right now it's possible for a ciao dependency to have a dependency
// that is no longer present in master.  This dependency will not be
// pulled down.  If this happens, ciao-vendor vendor will need to be
// invoked again.  We could probably fix this here.

func vendor(cwd, projectRoot, sourceRoot string) error {

	checkoutVersion(sourceRoot)
	deps, err := calcDeps(projectRoot, []string{"./..."})
	if err != nil {
		checkoutMaster(sourceRoot)
		return err
	}

	i := 0
	for ; i < len(deps); i++ {
		if !deps[i].vendored {
			break
		}
	}

	if i < len(deps) {
		checkoutMaster(sourceRoot)
		err = updateNonVendoredDeps(deps, projectRoot)
		if err != nil {
			return err
		}
		checkoutVersion(sourceRoot)

		deps, err = calcDeps(projectRoot, []string{"./..."})
		if err != nil {
			checkoutMaster(sourceRoot)
			return err
		}
	}

	subPackages := computeSubPackages(deps)

	for k := range subPackages {
		packages := subPackages[k]

		for _, p := range packages {
			dir := path.Join(sourceRoot, p.name)
			prefix := p.name[len(k):]
			if len(prefix) > 0 {
				prefix = prefix[1:]
			}
			docs, err := findDocs(dir, prefix)
			if err != nil {
				checkoutMaster(sourceRoot)
				return err
			}
			p.docs = docs
		}

		if packages[0].wildcard != "*" {
			dir := path.Join(sourceRoot, k)
			docs, err := findDocs(dir, "")
			if err != nil {
				checkoutMaster(sourceRoot)
				return err
			}
			packages = append(packages, &subPackage{name: k, docs: docs})
		}
		subPackages[k] = packages
	}
	checkoutMaster(sourceRoot)

	fmt.Println("Populating vendor folder")

	err = copyRepos(cwd, sourceRoot, subPackages)
	if err != nil {
		return err
	}

	fmt.Println("Dependencies vendored.  Run go run ciao-vendor/ciao-vendor.go check to verify all is well")
	return nil
}

func usedBy(name string, packages piList, depsMap map[string][]string) string {
	var users bytes.Buffer

	for _, p := range packages {
		if p.name == name {
			continue
		}

		deps := depsMap[p.name]
		for _, d := range deps {
			if d == name {
				users.WriteString(" ")
				users.WriteString(p.name)
				break
			}
		}
	}

	// BUG(markus): We don't report when a dependency is used by ciao if
	// it is also used by a dependency

	if users.Len() == 0 {
		return "project"
	}

	return users.String()[1:]
}

func depsByPackage(packages piList) map[string][]string {
	depsMap := make(map[string][]string)
	depsCh := make(chan packageDeps)
	for _, p := range packages {
		go func(p string) {
			var output bytes.Buffer
			cmd := exec.Command("go", "list", "-f", listTemplate, p)
			cmd.Stdout = &output
			err := cmd.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to call get list on %s : %v", p, err)
				depsCh <- packageDeps{p: p}
				return
			}
			scanner := bufio.NewScanner(&output)
			deps := make([]string, 0, 32)
			for scanner.Scan() {
				deps = append(deps, scanner.Text())
			}
			depsCh <- packageDeps{p, deps}
		}(p.name)
	}

	for range packages {
		pkgDeps := <-depsCh
		depsMap[pkgDeps.p] = pkgDeps.deps
	}

	return depsMap
}

func computeClients(packages piList) map[string]string {
	depsMap := depsByPackage(packages)
	clientMap := make(map[string]string)
	for _, p := range packages {
		clientMap[p.name] = usedBy(p.name, packages, depsMap)
	}
	return clientMap
}

func verify(deps piList, vendorRoot string) ([]string, []string, []string, []string) {
	uninstalled := make([]string, 0, 128)
	missing := make([]string, 0, 128)
	notVendored := make([]string, 0, 128)
	notUsed := make([]string, 0, 128)
	reposUsed := make(map[string]struct{})

depLoop:
	for _, d := range deps {
		if !d.installed {
			uninstalled = append(uninstalled, d.name)
		}
		for k := range repos {
			if strings.HasPrefix(d.name, k) ||
				(len(d.name) > len(vendorRoot)+1 &&
					strings.HasPrefix(d.name[len(vendorRoot)+1:], k)) {
				if !d.vendored {
					cmd := exec.Command("go", "list", path.Join(vendorRoot, d.name))
					if cmd.Run() != nil {
						notVendored = append(notVendored, d.name)
					}
				}
				reposUsed[k] = struct{}{}
				continue depLoop
			}
		}
		missing = append(missing, d.name)
	}

	for k := range repos {
		if _, ok := reposUsed[k]; !ok {
			notUsed = append(notUsed, k)
		}
	}

	return missing, uninstalled, notVendored, notUsed
}

func checkKnown(missing []string, deps piList) bool {
	if len(missing) == 0 {
		fmt.Println("All Dependencies Known: [OK]")
		return true
	}

	clientMap := computeClients(deps)

	fmt.Println("All Dependencies Known: [FAIL]")
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "Missing Package\tUsed By")
	for _, d := range missing {
		fmt.Fprintf(w, "%s\t%s\n", d, clientMap[d])
	}
	w.Flush()
	fmt.Println("")
	return false
}

func checkUninstalled(uninstalled []string) bool {
	if len(uninstalled) == 0 {
		fmt.Println("All Dependencies Installed: [OK]")
		return true
	}

	fmt.Println("All Dependencies Installed: [FAIL]")
	for _, d := range uninstalled {
		fmt.Printf("\t%s\n", d)
	}
	fmt.Println("")
	return false
}

func checkVendored(notVendored []string) bool {
	if len(notVendored) == 0 {
		fmt.Println("All Dependencies Vendored: [OK]")
		return true
	}

	fmt.Println("All Dependencies Vendored: [FAIL]")
	for _, d := range notVendored {
		fmt.Printf("\t%s\n", d)
	}
	fmt.Println("")

	return false
}

func checkNotUsed(notUsed []string) bool {
	if len(notUsed) == 0 {
		fmt.Println("All Dependencies Used: [OK]")
		return true
	}
	fmt.Println("All Dependencies Used: [FAIL]")
	for _, k := range notUsed {
		fmt.Println(k)
	}
	return false
}

func check(cwd, projectRoot string) error {
	deps, err := calcDeps(projectRoot, []string{"./..."})
	if err != nil {
		return err
	}
	vendorRoot := path.Join(projectRoot, "vendor")
	missing, uninstalled, notVendored, notUsed := verify(deps, vendorRoot)

	ok := checkKnown(missing, deps)
	ok = checkUninstalled(uninstalled) && ok
	ok = checkVendored(notVendored) && ok
	ok = checkNotUsed(notUsed) && ok

	if !ok {
		return fmt.Errorf("Dependency checks failed")
	}

	return nil
}

func packages(cwd, projectRoot string) error {
	uninstalledDeps := false
	plist, err := calcDeps(projectRoot, []string{"./..."})
	if err != nil {
		return err
	}

	vendorRoot := path.Join(projectRoot, "vendor")
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "Package\tStatus\tRepo\tVersion\tLicense")
	for _, d := range plist {
		fmt.Fprintf(w, "%s\t", d.name)
		r := ""
		for k := range repos {
			if strings.HasPrefix(d.name, k) ||
				(len(d.name) > len(vendorRoot)+1 &&
					strings.HasPrefix(d.name[len(vendorRoot)+1:], k)) {
				r = k
				break
			}
		}

		if d.vendored {
			fmt.Fprintf(w, "Vendored\t")
		} else if d.installed {
			fmt.Fprintf(w, "GOPATH\t")
		} else {
			fmt.Fprintf(w, "Missing\t")
			uninstalledDeps = true
		}

		if repos[r].URL != "" {
			fmt.Fprintf(w, "%s\t", r)
			if d.vendored {
				fmt.Fprintf(w, "%s\t", repos[r].Version)
			} else {
				fmt.Fprintf(w, "master\t")
			}
			fmt.Fprintf(w, "%s", repos[r].License)
		} else {
			fmt.Fprintf(w, "Unknown\tUnknown\tUnknown")
		}
		fmt.Fprintln(w)
	}
	w.Flush()

	if uninstalledDeps {
		fmt.Println("")
		return fmt.Errorf("Some dependencies are not installed.  Unable to provide complete dependency list")
	}

	return nil
}

func deps(projectRoot string) error {
	deps, err := calcDeps(projectRoot, []string{"./..."})
	if err != nil {
		return err
	}
	vendorRoot := path.Join(projectRoot, "vendor")
	missing, uninstalled, notVendored, notUsed := verify(deps, vendorRoot)
	if len(missing) != 0 || len(uninstalled) != 0 || len(notVendored) != 0 || len(notUsed) != 0 {
		return fmt.Errorf("Dependencies out of sync.  Please run go ciao-vendor/ciao-vendor.go check")
	}

	keys := make([]string, 0, len(repos))
	for k := range repos {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "Package Root\tRepo\tVersion\tLicense")

	for _, k := range keys {
		r := repos[k]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", k, r.URL, r.Version, r.License)
	}
	w.Flush()

	return nil
}

func uses(pkg string, projectRoot string, direct bool) error {
	deps, err := calcDeps(projectRoot, []string{"./..."})
	if err != nil {
		return err
	}

	var output bytes.Buffer
	cmd := exec.Command("go", "list", "./...")
	cmd.Stdout = &output
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("go list failed: %v", err)
	}

	scanner := bufio.NewScanner(&output)
	vendorPrefix := path.Join(projectRoot, "vendor")
	for scanner.Scan() {
		d := scanner.Text()
		if !strings.HasPrefix(d, vendorPrefix) {
			deps = append(deps, &packageInfo{name: d})
		}
	}

	var template string
	if direct {
		template = directTemplate
	} else {
		template = listTemplate
	}

	clientCh := make(chan clientInfo)
	for _, d := range deps {
		go func(name string) {
			ci := clientInfo{}
			pd, err := getPackageDependencies([]string{name}, template)
			if err == nil {
				if _, ok := pd[pkg]; ok {
					ci.name = name
				}
			} else {
				ci.err = err
			}
			clientCh <- ci
		}(d.name)
	}

	clients := make([]string, 0, len(deps))
	for range deps {
		clientInfo := <-clientCh
		if clientInfo.err != nil {
			return err
		}
		if clientInfo.name != "" {
			clients = append(clients, clientInfo.name)
		}
	}

	sort.Strings(clients)
	for _, client := range clients {
		fmt.Println(client)
	}

	return nil
}

func updates(sourceRoot, projectRoot string) error {
	deps, err := calcDeps(projectRoot, []string{"./..."})
	if err != nil {
		return err
	}

	vendorRoot := path.Join(projectRoot, "vendor") + "/"

	for _, d := range deps {
		if strings.HasPrefix(d.name, vendorRoot) {
			d.name = d.name[len(vendorRoot):]
		}
	}

	err = updateNonVendoredDeps(deps, projectRoot)
	if err != nil {
		return err
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "Package\tStatus\t")

	keys := make([]string, 0, len(repos))

	for k := range repos {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		v := repos[k]
		var output bytes.Buffer
		cmd := exec.Command("git", "log", "--oneline", fmt.Sprintf("%s..HEAD", v.Version))
		cmd.Stdout = &output
		cmd.Dir = path.Join(sourceRoot, k)
		err = cmd.Run()
		if err != nil {
			fmt.Fprintf(w, "%s\tUnknown: %v\t\n", k, err)
			continue
		}
		scanner := bufio.NewScanner(&output)
		count := 0
		for scanner.Scan() {
			count++
		}
		if count != 0 {
			fmt.Fprintf(w, "%s\t%d commits behind HEAD\t\n", k, count)
		} else {
			fmt.Fprintf(w, "%s\tUp to date\t\n", k)
		}
	}
	w.Flush()

	return nil
}

func test(sudo bool, sourceRoot, projectRoot, pkg, version string, goTestFlags []string) error {
	fmt.Printf("Go getting %s\n", pkg)
	cmd := exec.Command("go", "get", "-t", "-u", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to go get %s", pkg)
	}

	branch, err := getCurrentBranch(path.Join(sourceRoot, pkg))
	if err != nil {
		return fmt.Errorf("Unable to determine current branch of %s: %v", pkg, err)
	}
	cmd = exec.Command("git", "checkout", version)
	cmd.Dir = path.Join(sourceRoot, pkg)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to checkout version %s of %s: %v",
			version, pkg, err)
	}

	var args []string
	var command string
	if sudo {
		command = "sudo"
		args = []string{"-E", "go"}
	} else {
		command = "go"
	}
	args = append(args, "test")
	args = append(args, goTestFlags...)
	args = append(args, pkg)
	cmd = exec.Command(command, args...)
	cmd.Dir = path.Join(sourceRoot, pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if sudo {
		cmd.Stdin = os.Stdin
	}
	err = cmd.Run()

	cmd = exec.Command("git", "checkout", branch)
	cmd.Dir = path.Join(sourceRoot, pkg)
	_ = cmd.Run()

	return err
}

func revendor(cwd, sourceRoot, projectRoot, repo, version string) error {
	ri, ok := repos[repo]
	if !ok {
		return fmt.Errorf("%s is not a vendored repository", repo)
	}
	fmt.Printf("Go getting %s\n", repo)
	cmd := exec.Command("go", "get", "-v", "-u", "-d", repo+"/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to go get %s", repo)
	}

	ri.Version = version
	repos[repo] = ri
	err = writeRepos(cwd)
	if err != nil {
		return err
	}

	vendoredDir := path.Join(cwd, "vendor", repo)
	err = os.RemoveAll(vendoredDir)
	if err != nil {
		return fmt.Errorf("Unable to remove vendored directory %s : %v",
			vendoredDir, err)
	}

	return vendor(cwd, projectRoot, sourceRoot)
}

func vendorNew(cwd, sourceRoot, projectRoot, repo string, ri repoInfo) error {
	_, ok := repos[repo]
	if ok {
		return fmt.Errorf("%s is already vendored", repo)
	}

	repos[repo] = ri
	if err := writeRepos(cwd); err != nil {
		return err
	}

	return vendor(cwd, projectRoot, sourceRoot)
}

func unvendor(cwd, sourceRoot, projectRoot, repo string) error {
	_, ok := repos[repo]
	if !ok {
		return fmt.Errorf("%s is not vendored", repo)
	}

	delete(repos, repo)
	if err := writeRepos(cwd); err != nil {
		return err
	}

	vendoredDir := path.Join(cwd, "vendor", repo)
	err := os.RemoveAll(vendoredDir)
	if err != nil {
		return fmt.Errorf("Unable to remove vendored directory %s : %v",
			vendoredDir, err)
	}

	fmt.Printf("%s unvendored.  Run go run ciao-vendor/ciao-vendor.go check to verify all is well\n", repo)

	return nil
}

func runCommand(cwd, sourceRoot string, args []string) error {
	var err error

	projectRoot := cwd[len(sourceRoot)+1:]
	switch args[1] {
	case "check":
		err = check(cwd, projectRoot)
	case "vendor":
		err = vendor(cwd, projectRoot, sourceRoot)
	case "deps":
		err = deps(projectRoot)
	case "packages":
		err = packages(cwd, projectRoot)
	case "uses":
		fs := flag.NewFlagSet("uses", flag.ExitOnError)
		direct := false
		fs.BoolVar(&direct, "d", false, "output direct dependencies only")

		if err := fs.Parse(args[2:]); err != nil {
			return err
		}

		if len(fs.Args()) == 0 {
			return fmt.Errorf("Missing package for uses command")
		}

		err = uses(fs.Args()[0], projectRoot, direct)
	case "updates":
		err = updates(sourceRoot, projectRoot)
	case "test":
		fs := flag.NewFlagSet("test", flag.ExitOnError)
		sudo := false
		fs.BoolVar(&sudo, "s", false, "run tests with sudo")

		if err := fs.Parse(args[2:]); err != nil {
			return err
		}

		args = fs.Args()
		err = test(sudo, sourceRoot, projectRoot, args[0], args[1], args[2:])
	case "revendor":
		err = revendor(cwd, sourceRoot, projectRoot, args[2], args[3])
	case "vendornew":
		ri := repoInfo{URL: args[5], Version: args[3], License: args[4]}
		err = vendorNew(cwd, sourceRoot, projectRoot, args[2], ri)
	case "unvendor":
		err = unvendor(cwd, sourceRoot, projectRoot, args[2])
	}

	return err
}

func readRepos(projectRoot string) error {
	packageFile := path.Join(projectRoot, "packages.json")
	d, err := ioutil.ReadFile(packageFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("Unable to read %s : %v", packageFile, err)
		}
		return nil
	}

	err = json.Unmarshal(d, &repos)
	if err != nil {
		return fmt.Errorf("Unable to unmarshall %s : %v", packageFile, err)
	}

	return nil
}

func writeRepos(projectRoot string) error {
	packageFile := path.Join(projectRoot, "packages.json")

	d, err := json.MarshalIndent(&repos, "", "\t")
	if err != nil {
		return fmt.Errorf("Unable to marhsall %s : %v", packageFile, err)
	}

	err = ioutil.WriteFile(packageFile, d, 0755)
	if err != nil {
		return fmt.Errorf("Unable to write %s : %v", packageFile, err)
	}

	return nil
}

func checkTwoArgs(args []string) bool {
	return (len(args) == 2 &&
		(args[1] == "vendor" || args[1] == "check" || args[1] == "deps" ||
			args[1] == "packages" || args[1] == "updates"))
}

func checkArgs(args []string) bool {
	return checkTwoArgs(args) ||
		(len(args) == 3 && (args[1] == "unvendor")) ||
		(len(args) >= 3 && (args[1] == "uses")) ||
		(len(args) == 4 && (args[1] == "revendor")) ||
		(len(args) == 6 && (args[1] == "vendornew")) ||
		(len(args) >= 4 && (args[1] == "test"))
}

func main() {
	if !checkArgs(os.Args) {
		fmt.Fprintln(os.Stderr, "Usage: ciao-vendor vendor|check|deps|packages|updates")
		fmt.Fprintln(os.Stderr, "Usage: ciao-vendor uses [-d] package")
		fmt.Fprintln(os.Stderr, "Usage: ciao-vendor test package version [go-test flags]")
		fmt.Fprintln(os.Stderr, "Usage: ciao-vendor unvendor package")
		fmt.Fprintln(os.Stderr, "Usage: ciao-vendor revendor package version")
		fmt.Fprintln(os.Stderr, "Usage: ciao-vendor vendornew package version license URL")
		os.Exit(1)
	}

	cwd, goPath, err := checkWD()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = readRepos(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	sourceRoot := path.Join(goPath, "src")
	if len(cwd) < len(sourceRoot)+1 {
		fmt.Fprintln(os.Stderr, "Could not determine project root")
		os.Exit(1)
	}
	err = runCommand(cwd, sourceRoot, os.Args)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
