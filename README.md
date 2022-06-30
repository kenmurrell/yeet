# yeet
 A repo-tool wrapper written in Go. `yeet` lets you quickly switch between changesets that exist across multiple repositories within your repo tool manifest.

 ## Overview
 `yeet` simplifies the process of checking out and rebasing similarly-named remote branches across multiple repositories.

 Imagine your co-worker is working on *feature123*. They had to make changes in a few different repositories, each of which now have a branch `feature123` where this new feature lives. Now imagine you want to check out those changes and test the new feature locally.

 The problem: while you can check out `feature123` on the repos that were modified for that feature, you don't know what commit all the other repos were sitting at when `feature123` was built. It's pretty likely that after checking out `feature123` in just a select few reposiorites while the other repos sit at whatever commit you last checked out, the entire code base is no longer is a valid state.

 The solution: The simplest way to fix this is to:
 - Bring **all** the repositories up to the tip of the main branch
 - Check out `feature123` on those select repositories in which this feature lives
 - Rebase `feature123` onto the tip of the main branch in those repositories

 At this point you've effectively brought your code base up to the state it would be if `feature123` was merged into the main branch.

 This is where `yeet` comes in. `yeet` automates this process by doing all the git-fu for you.

## Usage

### refresh

```
$ yeet refresh
```

Before you can use `yeet` to perform a rebase, you need a list of the repositories across which to rebase the target branch and their remote addresses. The `refresh` command collects this information via the `repo list` command and saves the information to *repolist.json*.

### rebase

```
$ yeet rebase <targetbranch>
```

Bring all repositories up to the tip of the main branch and create a new branch `<targetbranch>` by rebasing `origin/<targetbranch>` onto the tip of main in those repos where `origin/<targetbranch>` exists