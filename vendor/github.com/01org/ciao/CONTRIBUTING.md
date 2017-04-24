# Contributing to Ciao

Ciao is an open source project licensed under the [Apache v2 License] (https://opensource.org/licenses/Apache-2.0)

## Coding Style

Ciao follows the standard formatting recommendations and language idioms set out
in the [Effective Go](https://golang.org/doc/effective_go.html) guide. It's
definitely worth reading - but the relevant sections are
[formatting](https://golang.org/doc/effective_go.html#formatting)
and [names](https://golang.org/doc/effective_go.html#names).

## Certificate of Origin

In order to get a clear contribution chain of trust we use the [signed-off-by language] (https://01.org/community/signed-process)
used by the Linux kernel project.

## Patch format

Beside the signed-off-by footer, we expect each patch to comply with the following format:

```
<component>: Change summary

More detailed explanation of your changes: Why and how.
Wrap it to 72 characters.
See [here] (http://chris.beams.io/posts/git-commit/)
for some more good advices.

Signed-off-by: <contributor@foo.com>
```

For example:

```
ssntp: Implement role checking

SSNTP roles are contained within the SSNTP certificates
as key extended attributes. On both the server and client
sides we are verifying that the claimed roles through the
SSNTP connection protocol match the certificates.

Signed-off-by: Samuel Ortiz <sameo@linux.intel.com>
```

## Pull requests

We accept github pull requests.

If you want to work on github.com/01org/ciao and your fork on the same workstation you will need to use multiple GOPATHs.  Assuming this is the case

1. Open a terminal
2. mkdir -p ~/go-fork/src/github.com/01org (replacing go-fork with your preferred location)
3. export GOPATH=~/go-fork
4. cd $GOPATH/src/github.com/01org
5. git clone https://github.com/GITHUB-USERNAME/ciao.git (replace GITHUB-USERNAME with your username)
6. cd ciao
7. go install ./...

Once you've finished making your changes push them to your fork and send the PR via the github UI.  If you don't need to maintain the github.com/01org/ciao repo and your fork on the same workstation you can skip steps 2 and 3.

## Quality Controls

We request you give quality assurance some consideration by:
* Adding go unit tests for changes where it makes sense.
* Using our [test-cases](https://github.com/01org/ciao/tree/master/test-cases) package to run unit tests because a simple ```gotest ./...``` can result in multiple packages' unit tests running in parallel which may be unsafe.  Simply replace ```go test``` with ```test-cases``` in your workflow.
* Running basic cluster acceptance tests which are available in [_release/bat](https://github.com/01org/ciao/tree/master/_release/bat) and are most easily run inside a [singlevm](https://github.com/01org/ciao/tree/master/testutil/singlevm) test environment, but may also be run on a hardware cluster which has your code installed/configured/running.  In some cases it will be necessary to test on real hardware, but in many a virtual cluster is an easy and fully sufficient path to test.
* Adding BAT tests for changes where it makes sense.  The BAT tests themselves are implemented in go in the [bat package](https://github.com/01org/ciao/tree/master/bat), which is essentially a wrapper/driver around ciao-cli.  Both the [bat package](https://github.com/01org/ciao/tree/master/bat) and [BAT tests](https://github.com/01org/ciao/tree/master/_release/bat) are easy to extend.
* Enabling [Travis CI](https://travis-ci.org/01org/ciao) on your github fork of Ciao to get continuous integration feedback on your dev/test branches. We have tresholds on code coverage tracked by [coveralls](https://coveralls.io/github/01org/ciao) which you will see reported once you submit your pull request.

## Issue tracking

If you have a problem, please let us know.  IRC is a perfectly fine place
to quickly informally bring something up, if you get a response.  The
[mailing list](https://lists.clearlinux.org/mailman/listinfo/ciao-devel)
is a more durable communication channel.

If it's a bug not already documented, by all means please [open an
issue in github](https://github.com/01org/ciao/issues/new) so we all get visibility
the problem and work toward resolution.

For feature requests we're also using github issues, with the label
"enhancement".

Our github bug/enhancement backlog and work queue are tracked in a
[Ciao waffle.io kanban](https://waffle.io/01org/ciao).

## Closing issues

You can either close issues manually by adding the fixing commit SHA1 to the issue
comments or by adding the `Fixes` keyword to your commit message:

```
ssntp: test: Add Disconnection role checking tests

We check that we get the right role from the disconnection
notifier.

Fixes #121

Signed-off-by: Samuel Ortiz <sameo@linux.intel.com>
```

Github will then automatically close that issue when parsing the
[commit message](https://help.github.com/articles/closing-issues-via-commit-messages/).
