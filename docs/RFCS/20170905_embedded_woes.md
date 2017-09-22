- Feature Name: Embedded.go
- Status: draft
- Start Date: 2017-09-05
- Authors: Bram Gruneir
- RFC PR:
- Cockroach Issue:

# Summary

This RFC considers alternatives to the current `embedded.go` system with the ultimate goal of finding a replacement.

# Motivation

When working on anything UI related, the current system of checking in the resulting `embedded.go` forces the serialization of all changes. This causes fairly long delays in development and discourages small CLs and PRs. As the team grows, so will the pain from this, as it tends to take a long time to regenerate (and retest) it for each rebase.

The file is huge, currently sitting at 10GiB.  This in itself causes numerous problems.  It breaks github's `assign reviewers` feature (while not directly our problem, we still have to work around it). It makes our repo significantly larger.

Futhermore, having to check in CI to ensure that that embedded.go has been correctly updated can take a while.  We now have the `make pre-push` command to do this locally, but that still does require a long time.

## Current System

For each change list applied that involves the UI in any way (be it updates to the UI directly or a protobuf update that touches the UI), `embedded.go` has to be regenerated.  This takes around During a larger rebase,


# Options

Here are some potential solutions.  Please note that this list is not exhaustive and I've love to hear of more options.

## Move the UI to another repo and vendor it

This seems like a quick and easy solution at first, but there are a number of problems with it. There are timing issues, as to how often we update the vendored version.

### Advantages

* We can set the frequency of updates, to say, nightly, so that it won't interfere with normal debugging.
* There is no need to have the UI tools unless a developer is directly working on it.
* We already have different makefiles, so this should be a relatively easy change to make.

### Disadvantages

* This breaks up the codebase, and there's an issue here that the UI relies on the protos, and then cockroach relies on `embedded.go`.
* There is still a stored copy of it locally, however, it will get updated less frequently (and ideally automatically).
* By being in another repo, it may discourage developers from venturing into UI development.
* There needs to be some system to automatically update the vendored version, since manually doing so will be a massive pain.

## Vendor/Store just `embedded.go`

This approach tries to avoid the issues of splitting the repo into two projects, but might be significantly more complicated to implement.
It also could use some other system to store versions of it, S3 for example, instead of using the current vendoring system. But then we may have to introduce more tools around managing that.

### Advantages

* We can set the frequency of updates, to say, nightly, so that it won't interfere with normal debugging.

### Disadvantages

* There is still a stored copy of it locally, however, it will get updated less frequently (and ideally automatically).
* There needs to be some system to automatically update the vendored version, since manually doing so will be a massive pain.

## Generate `embedded.go` during the build

How about we just remove embedded.go from everywhere except for locally when building cockroach.  So this will spread out the pain of having to re-generate the file on a fresh build, and every time any of the files it includes gets updated, but it doesn't suffer from most of the disadvantages that the other potential solutions

### Advantages

* `embedded.go` is never stored anywhere.
* Gets rid of our current system of having to build two different systems (ui and then cockroach) for each PR.

### Disadvantages

* This will slow down all build times (including tests).  But this slower build would only happen after a rebase if the UI is not being worked on.
* A caching system will have to be added but I'm pretty sure that we could adjust the makefile accordingly.
* Every developer will require the full UI toolkit to build cockroach, regardless of what area they're working on.

## Use a dev-mode for `embedded.go`

* Instead of recompiling `embedded.go` for every build, we switch embedding programs to Matt Jibson's that has first class support for just opening the files on disk.  And we add building everything needed except for embedded.go to the main build script.  Only on release builds, would we generate embedded.go.

### Advantages

### Disadvantage

* This still relies on webpack, which is the biggest time sync.  If it can be removed, this would be the best solution.