# yeet
 A repo-tool wrapper written in Go. `yeet` lets you quickly switch between changesets that exist across multiple repositories within your repo-tool manifest.

 ## Overview
 `yeet` simplifies the process of checking out and rebasing similarly-named remote branches across multiple repositories that are managed using the repo tool (https://gerrit.googlesource.com/git-repo/).

 Imagine your co-worker is working on *feature123*. They made changes in a few different repositories, each of which now has a branch named `feature123` where this new feature lives. Now imagine you want to check out those changes and test the new feature locally.

 **The problem**: while you can check out `feature123` on the repositories that were modified for that feature, you don't know what commit all the other repositories were sitting at when `feature123` was built. It's pretty likely that after checking out `feature123` in just a select few reposiorites while the other repos sit at whatever commit you last checked out, the entire code base is no longer in a valid state.

 **The solution**: The simplest way to fix this is to:
 - Bring **all** the repositories up to the tip of the main branch
 - Check out `feature123` on those select repositories in which this feature lives
 - Rebase `feature123` onto the tip of the main branch in those repositories

 At this point you've effectively brought your code base up to the state it would be in if `feature123` was merged into the main branch.

 This is where `yeet` comes in. `yeet` automates this process by doing all the git-fu for you.

## Usage

### Config File

`yeet` uses a YAML config file to specify some additional pieces of information, including:
- The name of your main (production) branch
- The name of the remote you would prefer to check out from if not "origin"
- The directory containing all repositories maintained by your repo tool

Before running the `take` command, ensure this config file has been filled in correctly.

### Commands

#### refresh

```
$ yeet refresh
```

Before you can use `yeet` to perform a rebase, you need a list of the repositories across which to rebase the target branch and their remote addresses. The `refresh` command collects this information via the `repo list` command and saves the information to *repolist.json*.

#### take

```
$ yeet take <targetbranch>
```

Bring all repositories up to the tip of the main branch and create a new branch `<targetbranch>` by rebasing `origin/<targetbranch>` onto the tip of main in those repos where `origin/<targetbranch>` exists