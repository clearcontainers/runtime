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
	"errors"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"os"
	"strings"
	"time"
)

const repoOwner = "01org"
const repo = "ciao"

var rcSHA = flag.String("head", "", "The sha of the release candidate")

func getAuthenticatedClient() (*github.Client, error) {
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		return nil, errors.New("You must set GITHUB_TOKEN env var")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)

	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	return client, nil
}

func getReleaseCandidateTime(client *github.Client) (time.Time, error) {
	rc := time.Time{}

	headCommit, _, _ := client.Repositories.GetCommit(repoOwner, repo, *rcSHA)
	if headCommit == nil {
		return rc, errors.New("The SHA was not valid")
	}
	rc = *headCommit.Commit.Committer.Date

	return rc, nil
}

func getLastReleaseTime(client *github.Client) (time.Time, error) {
	var lastRelease time.Time

	release, _, _ := client.Repositories.GetLatestRelease(repoOwner, repo)

	var tags []*github.RepositoryTag
	var tagsOpts github.ListOptions

	for {
		t, resp, err := client.Repositories.ListTags(repoOwner, repo, &tagsOpts)
		if err != nil {
			glog.Fatal(err)
		}

		tags = append(tags, t...)

		if resp.NextPage == 0 {
			break
		}

		tagsOpts.Page = resp.NextPage
	}

	for _, t := range tags {
		if *t.Name == *release.TagName {
			c, _, _ := client.Repositories.GetCommit(repoOwner, repo, *t.Commit.SHA)
			if c == nil {
				return lastRelease, errors.New("The SHA was not valid")
			}
			lastRelease = *c.Commit.Committer.Date
		}
	}

	return lastRelease, nil
}

func getAllPullRequests(client *github.Client) (map[int]github.PullRequest, error) {
	var prs []*github.PullRequest

	prOpts := github.PullRequestListOptions{
		State: "closed",
		Base:  "master",
	}

	for {
		pr, resp, err := client.PullRequests.List(repoOwner, repo, &prOpts)
		if err != nil {
			return nil, err
		}

		prs = append(prs, pr...)

		if resp.NextPage == 0 {
			break
		}

		prOpts.Page = resp.NextPage
	}

	prmap := make(map[int]github.PullRequest)
	for _, pr := range prs {
		prmap[*pr.Number] = *pr
	}

	return prmap, nil
}

func getIssueEvents(client *github.Client, lastRelease time.Time, rc time.Time) (map[string][]*github.IssueEvent, error) {
	prmap, err := getAllPullRequests(client)
	if err != nil {
		return nil, err
	}

	var events []*github.IssueEvent
	var eventOpts github.ListOptions

	for {
		e, resp, err := client.Issues.ListRepositoryEvents(repoOwner, repo, &eventOpts)
		if err != nil {
			return nil, err
		}

		events = append(events, e...)

		if resp.NextPage == 0 {
			break
		}

		eventOpts.Page = resp.NextPage
	}

	eventsmap := make(map[string][]*github.IssueEvent)

	for _, e := range events {
		key := *e.Event

		if key == "merged" || key == "closed" {
			num := *e.Issue.Number

			if e.Issue.PullRequestLinks != nil {
				_, ok := prmap[num]
				if !ok {
					continue
				}
			}

			if lastRelease.IsZero() {
				eventsmap[key] = append(eventsmap[key], e)
			} else {
				if lastRelease.Before(*e.CreatedAt) {
					if e.CreatedAt.Before(rc) {
						eventsmap[key] = append(eventsmap[key], e)
					}
				}
			}
		}
	}

	return eventsmap, err
}

func getCommits(client *github.Client, since time.Time, until time.Time) ([]*github.RepositoryCommit, error) {
	var commits []*github.RepositoryCommit

	copts := github.CommitsListOptions{
		Since: since,
		Until: until,
	}

	for {
		c, resp, err := client.Repositories.ListCommits(repoOwner, repo, &copts)
		if err != nil {
			return nil, err
		}

		commits = append(commits, c...)
		if resp.NextPage == 0 {
			break
		}
		if resp != nil {
			copts.ListOptions.Page = resp.NextPage
		}
	}

	return commits, nil
}

func generateReleaseNotes() error {
	client, err := getAuthenticatedClient()
	if err != nil {
		return err
	}

	rc, err := getReleaseCandidateTime(client)
	if err != nil {
		return err
	}

	lastRelease, err := getLastReleaseTime(client)
	if err != nil {
		return err
	}

	eventsmap, err := getIssueEvents(client, lastRelease, rc)
	if err != nil {
		return err
	}

	commits, err := getCommits(client, lastRelease, rc)
	if err != nil {
		return err
	}

	f, err := os.Create("release.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "Changes since last release\n\n")

	for key, list := range eventsmap {
		fmt.Fprintf(f, "---%s---\n", key)
		for _, e := range list {
			i := *e.Issue
			fmt.Fprintf(f, "\tIssue/PR #%d: %s\n", *i.Number, *i.Title)
			fmt.Fprintf(f, "\tURL: %s\n\n", *i.HTMLURL)
		}
		fmt.Fprintf(f, "\n")
	}

	if len(commits) > 0 {
		fmt.Fprintln(f, "---Full Change Log---")
	}

	for _, c := range commits {
		lines := strings.Split(*c.Commit.Message, "\n")
		fmt.Fprintf(f, "\t%s\n", lines[0])
	}

	return nil
}

func main() {
	flag.Parse()

	err := generateReleaseNotes()
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}
}
