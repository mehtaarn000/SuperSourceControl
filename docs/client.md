# Sharing code with SSC

SSC can clone a shared repository, push the active branch, and pull remote
commits into the active branch. The server must already have that repository
and grant your token read or write access. See [server setup](server.md).

## Clone and exchange commits

Use a repository endpoint ending in `/v1/repos/<repository-name>` and provide
your token through `SSC_TOKEN`. Tokens are never saved in `.ssc` or committed.
HTTPS uses normal system/Go certificate verification; there is no insecure TLS
option. HTTP is allowed only with an explicit loopback IP for local development.
Redirects are refused so credentials cannot be forwarded to another endpoint.

```sh
export SSC_TOKEN='your-access-token'
ssc clone https://ssc.example.com:8443/v1/repos/demo project
cd project
ssc config -c authorName 'Your Name'
ssc config -c authorEmail 'you@example.com'

# Edit files, then save and share a snapshot.
ssc commit -m 'Implement the feature'
ssc push

# In another developer's checkout:
ssc pull
```

Clone requires a new destination directory; its parent directory must already
exist. It stages and verifies the repository before publishing the destination.
It downloads all advertised branches and their reachable objects. It checks out
`main`, otherwise `master`, otherwise the first branch in alphabetical order.
An empty server repository creates an empty local `main` branch.

To choose a different initial branch, put the flag before the URL:

```sh
ssc clone --branch feature/login https://ssc.example.com:8443/v1/repos/demo project
```

Only clone chooses a branch via a flag. Push and pull operate on the active local
branch and the identically named server branch. Author settings are needed to
create commits, not to clone, push, or pull.

## Connect an existing repository

```sh
ssc remote https://ssc.example.com:8443/v1/repos/demo
ssc remote       # Display the saved URL.
ssc push
```

The URL is saved in `.ssc/remote.json`; it contains no token. Use
`ssc push --remote <repository-url>` or `ssc pull --remote <repository-url>` for
a one-time override that does not change the saved URL. A write token is required
to publish changes; a read token can clone and pull.

## What push checks

Push verifies the local reachable objects, checks which objects the server
already has, and uploads only missing objects. The active branch must contain
at least one commit. It publishes the tip only after all uploads succeed.

The remote tip must be an ancestor of the local tip. SSC sends the observed
remote tip as an explicit expectation when publishing. A concurrent update gets
a conflict instead of being overwritten. SSC never retries a failed branch
update using a newer expected tip, and it does not provide force-push.

Retrying a failed push is safe: object uploads are idempotent, already-uploaded
objects are skipped, and a tip that already matches is a no-op. If the remote
has new work, pull first. If both sides have different new commits, merging is
not implemented yet; SSC stops and preserves both histories.

## What pull checks

Pull fetches an advertised tip and permits only a fast-forward of the active
branch. Equal tips, or a remote tip already contained in local history, are
no-ops. Missing remote branches and divergent histories are errors.

Before changing the checkout, SSC requires tracked files to match the current
snapshot and rejects untracked files, missing tracked files, symlinks, and special
files. Save or move that work first. The check is repeated after downloading and
staging the incoming snapshot. Root `.git` metadata is preserved; new local
snapshots also exclude `.git` and `.ssc` metadata.

Downloads are bounded and checked against object hashes and expected types.
Invalid metadata paths and case-conflicting paths are rejected. A normal failed
transfer leaves working files and branch history unchanged; verified downloaded
objects may remain cached for the next attempt. Interrupted clones can be
retried, but do not retain an object cache in the unpublished destination.

Pull stages the new snapshot and moves the previous checkout into a temporary
backup before replacing files. It publishes the branch history last. This handles
file-to-directory and directory-to-file changes and rolls back ordinary checkout
failures. It uses the same repository operation lock as commit, branch, and
revert commands. External editors do not obey this lock; avoid editing files
while a pull is applying its checkout.

## Interrupted checkout recovery

If a process is killed during checkout, or rollback itself fails, SSC retains
`.ssc/sync-pending` and a backup under `.ssc/tmp/pull-*`. Further modifying
commands refuse to run. The marker records the backup directory, branch, and
old/new top-level entries; the directory contains `backup/`, `new/`, and the
original `old-commitlog`.

Keep those files until recovery is complete. Confirm no SSC process is running,
then use the marker and remaining staged/backup entries to determine which moves
completed. Restore the old files and original branch commitlog together, or
verify that the new checkout and history are complete, before clearing the
marker and stale `.ssc/commit.lock`. Do not simply delete the marker to bypass
the check. Recovery is manual in this release, and the mechanism does not promise
power-loss durability.

## Current limits

- No automatic merge, rebase, force-push, remote branch deletion, or credential manager.
- Pull updates only the active branch; it does not create newly advertised branches.
  Clone downloads all advertised branches.
- Legacy commits remain hash-compatible, but missing parent links are history
  boundaries. Clone cannot reconstruct older commits available only in a legacy
  branch log; pull preserves existing local log entries. Operations requiring
  ancestry beyond those boundaries are rejected.
- Transfers use the server's limits: 8 MiB per object and a graph budget of
  10,000 references and 256 MiB of decompressed content. Clone applies that budget
  across all advertised branches. A graph is kept in memory during validation.
- Snapshot format still records file contents and paths, not executable bits,
  symlinks, or empty directories. There is no configurable ignore-file feature.
- Existing snapshots containing `.git` or unsafe paths cannot be transferred.
  The new metadata exclusion applies to future snapshots only.

## Tests

```sh
go test ./...
go test -race ./remote ./server
go vet ./...
```

Tests exercise actual CLI subprocesses for two developers, empty and populated
clones, multiple branches, missing-object uploads, dirty working directories,
divergent histories, interrupted checkout rollback, metadata/symlink protection,
read-only access, corrupt objects, and rejected redirects. Network tests use
local ephemeral loopback ports.
