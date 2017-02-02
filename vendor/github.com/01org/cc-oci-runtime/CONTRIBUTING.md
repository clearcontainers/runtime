# Contributing to cc-oci-runtime

cc-oci-runtime is an open source project licensed under the [GPL v2 License] (https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html).

## Coding Style (C)

The coding style for cc-oci-runtime is roughly K&R with function names in
column 0, and variable names aligned in declarations.

The right results can be almost achieved by doing the following.

* GNU Emacs: if you're not using auto-newline, the following should do the right thing:

```
	(defun cc-oci-runtime-c-mode-common-hook ()
	  (c-set-style "k&r")
	  (setq indent-tabs-mode t
	        c-basic-offset 8))
```

* VIM: the default works except for the case labels in switch statements.  Set the following option to fix that:

```
	setlocal cinoptions=:0
```

* Indent: can be used to reformat code in a different style:

```
	indent -kr -i8 -psl
```

## Coding Style (Go)

The usual Go style, enforced by `gofmt`, should be used. Additionally, the [Go
Code Review](https://github.com/golang/go/wiki/CodeReviewComments) document
contains a few common errors to be mindful of.


## Certificate of Origin

In order to get a clear contribution chain of trust we use the [signed-off-by language] (https://01.org/community/signed-process)
used by the Linux kernel project.

## Patch format

Beside the signed-off-by footer, we expect each patch to comply with the following format:

```
       Subsystem: Change summary (no longer than 75 characters)

       More detailed explanation of your changes: Why and how.
       Wrap it to 72 characters.
       See:
           http://chris.beams.io/posts/git-commit/
       for some more good advice, and the Linux Kernel document:
           https://git.kernel.org/cgit/linux/kernel/git/torvalds/linux.git/tree/Documentation/SubmittingPatches

       Signed-off-by: <contributor@foo.com>
```

For example:

```
       OCI: Ensure state is updated when kill fails.

       Killing a container involves a number of steps. If any of these fail,
       the container state should be reverted to it's previous value.

       Signed-off-by: James Hunt <james.o.hunt@intel.com>
```

Note, that the body of the message should not just be a continuation of the subject line, and is not used to extend the subject line beyond its length limit. They should stand alone as complete sentence and paragraphs.

It is recommended that each of your patches fixes one thing. Smaller patches are easier to review, and are thus more likely to be accepted and merged, and problems are more likely to be picked up during review.

## Pull requests

We accept [github pull requests] (https://github.com/01org/cc-oci-runtime/pulls).

Github has a basic introduction to the process [here] (https://help.github.com/articles/using-pull-requests/).

When submitting your Pull Request (PR), treat the Pull Request message the same you would a patch message, including pre-fixing the title with a subsystem name. Github by default seems to copy the message from your first patch, which many times is appropriate, but please ensure your message is accurate and complete for the whole Pull Request, as it ends up in the git log as the merge message.

Your pull request may get some feedback and comments, and require some rework. The recommended procedure for reworking is to rework your branch to a new clean state and 'force push' it to your github. GitHub understands this action, and does sensible things in the online comment history. Do not pile patches on patches to rework your branch. Any relevant information from the github comments section should be re-worked into your patch set, as the ultimate place where your patches are documented is in the git log, and not in the github comments section.

For more information on github 'force push' workflows see [here] (http://blog.adamspiers.org/2015/03/24/why-and-how-to-correctly-amend-github-pull-requests/).

It is perfectly fine for your Pull Request to contain more than one patch - use as many patches as you need to implement the Request (see the previously mentioned 'small patch' thoughts). Each Pull Request should only cover one topic - if you mix up different items in your patches or pull requests then you will most likely be asked to rework them.

## Reviews

Before your Pull Requests are merged into the main code base, they will be reviewed. Anybody can review any Pull Request and leave feedback (in fact, it is encouraged), but this project runs a rotational GateKeeper schedule for who is ultimately responsible for merging the Pull Requests into the main code base. See [here] (https://github.com/01org/cc-oci-runtime/wiki/GateKeeper-Schedule).

We use an 'acknowledge' system for people to note if they agree, or disagree, with a Pull Request. We utilise some automated systems that can spot common acknowledge patterns, which include placing any of these at the beginning of a comment line:

 - LGTM
 - lgtm
 - +1
 - Approve

## Contact

The Clear Containers community can be reached through its IRC channel and a
dedicated mailing list:

* IRC: `#clearcontainers @ freenode.net`.
* Mailing list: [Subscribe](https://lists.01.org/mailman/listinfo/cc-devel).

## Issue tracking

If you have a problem, please let us know. [IRC](#contact) is a perfectly fine place to quickly
informally bring something up, if you get a response.
The [mailing list](https://lists.01.org/mailman/listinfo/cc-devel) is a more durable
communication channel.

If it's a bug not already documented, by all means please [open an
issue in github](https://github.com/01org/cc-oci-runtime/issues/new) so we all get
visibility on the problem and work toward resolution.

## Closing issues

You can either close issues manually by adding the fixing commit SHA1 to the issue
comments or by adding the `Fixes` keyword to your commit message:

```
Fix handling of semvers with only a single pre-release field

Fixes #121

Signed-off-by: James Hunt <james.o.hunt@intel.com>
```

Github will then automatically close that issue when parsing the
[commit message](https://help.github.com/articles/closing-issues-via-commit-messages/).
