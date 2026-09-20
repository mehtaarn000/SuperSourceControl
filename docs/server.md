# SSC repository server

The server stores SSC objects and shared branch tips. It authenticates bearer
tokens and applies per-repository read/write grants. It does not run code from
repositories. Client-side `clone`, `push`, and fast-forward `pull` use this API; see the
[client guide](client.md) for the developer workflow.

## Start a server

Build with Go 1.20 or newer (use a maintained Go release):

```sh
go build -o ssc .
./ssc serve --generate-token
```

This prints a random 256-bit token and the SHA-256 digest of the token's exact
text. Give the token to its intended user over a secure channel. Put only its
digest in the server configuration. Generate a different token for each user.

Create `server.json`, replacing the digest placeholders with generator output:

```json
{
  "repositories": ["demo"],
  "tokens": [
    {
      "name": "alice",
      "sha256": "REPLACE_WITH_ALICE_TOKEN_SHA256",
      "repositories": {"demo": "write"}
    },
    {
      "name": "reviewer",
      "sha256": "REPLACE_WITH_REVIEWER_TOKEN_SHA256",
      "repositories": {"demo": "read"}
    }
  ]
}
```

```sh
./ssc serve --config server.json --data /path/to/ssc-server-data
```

The default listener is `127.0.0.1:8080`. Neither `ssc init` nor a local author
configuration is required. Keep server data outside a working repository.
Repository names use lowercase ASCII letters, digits, hyphens, and underscores,
with a maximum length of 100 characters. Listed repositories are provisioned at
startup; clients cannot create or delete repositories.

For access from other machines, provide a certificate trusted by your clients:

```sh
./ssc serve --config server.json --data /path/to/ssc-server-data \
  --listen 0.0.0.0:8443 --tls-cert server.crt --tls-key server.key
```

TLS 1.2 or newer is required on non-loopback listeners. Alternatively, terminate
TLS at a reverse proxy that forwards to an explicit loopback IP. Plaintext
listeners accept only loopback IP literals, not DNS names. Stop the server with
Ctrl-C or SIGTERM; it closes connections and finishes active handlers before
releasing its storage lock.

Edit grants or remove a token and restart to change or revoke access. `write`
includes `read`; neither permission grants access to any unlisted repository.
A token cannot claim permissions through an HTTP request. Author names embedded
in commits remain self-declared and are separate from authentication.

## HTTP API

All `/v1` endpoints require `Authorization: Bearer <token>`. Do not send tokens
in URLs. The unauthenticated `GET /healthz` endpoint returns `{"status":"ok"}`.

| Method and path | Request | Success |
| --- | --- | --- |
| `PUT /v1/repos/{repo}/objects/{hash}` | Raw decompressed object body; `Content-Type: application/octet-stream`; `X-SSC-Object-Type: blob`, `tree`, or `commit` | `204`; identical retries are safe |
| `GET /v1/repos/{repo}/objects/{hash}` | None | `200` with raw object body and `X-SSC-Object-Type` |
| `HEAD /v1/repos/{repo}/objects/{hash}` | None | `200` with type and length, no body |
| `GET /v1/repos/{repo}/refs` | None | `200`, `{"refs":{"main":"<hash>"}}` |
| `GET /v1/repos/{repo}/refs/{branch}` | None | `200`, `{"name":"main","hash":"<hash>"}` |
| `PUT /v1/repos/{repo}/refs/{branch}` | `Content-Type: application/json`; explicit `old` and `new` string fields | `200` with name and new hash |

Branch names can contain `/`, for example `refs/feature/login`. A branch cannot
also be another branch's namespace, matching local SSC branch behavior.

To create a branch, send `{"old":"","new":"<commit hash>"}`. To advance it,
send `{"old":"<observed tip>","new":"<descendant commit hash>"}`. The server
compares `old` while holding the repository lock. If another writer has already
advanced the tip, it returns `409` without changing the branch. Re-read the tip
before deciding how to integrate changes; do not blindly replace `old` and retry.
There is no force-update or branch-deletion endpoint.

Upload missing blobs, trees, and commits before updating a branch. Objects may be
uploaded in any order, but the branch update checks the complete reachable graph:
all trees, blobs, and parents must exist, match their types and hashes, and fit
validation limits. Updating an existing branch also requires its old tip to be
reachable through the new commit's parent links. This rejects history rewrites
and divergent updates; it does not perform a merge.

Interrupted uploads publish no partial object. A retry with the same hash and
body is idempotent. If the connection is lost after a successful ref update,
read the ref to establish the result; repeating the original update returns
`409` because its old tip is no longer current.

Errors use `{"error":"message"}`:

- `400`: malformed data, invalid names, wrong object hash, or incomplete body.
- `401`: missing or invalid authentication.
- `403`: a read-only token attempted a write.
- `404`: object/ref absent, unknown repository, or no grant for that repository.
- `409`: stale expected tip, non-fast-forward update, or branch namespace conflict.
- `413`: upload exceeds the object/request size limit.
- `415`: unsupported media type or encoded request body.
- `422`: missing/wrong graph objects or graph validation limit exceeded.
- `503`: request concurrency limit reached or server is stopping.
- `500`: storage failure; filesystem paths and credentials are not returned.

## Storage and limits

Each repository has a separate object directory and `refs.json`. Stored object
files are zlib-compressed `type\n<body>` envelopes. They are distinct from local
`.ssc/objects` files; the wire API transfers decompressed bodies, not disk files.
The object hash follows the existing SSC format (including the historical
blob/tree and legacy-commit hashing rules). Version 2 commits include their
parent links and author metadata in their hash.

Legacy commits can be uploaded without rewriting their IDs. Their missing
parent links are a history boundary: updates requiring ancestry beyond that
boundary are rejected. Trees with unsafe paths, duplicate paths, paths under
`.ssc` or root `.git`, and file/directory conflicts are rejected. New snapshots automatically exclude `.git` and `.ssc` metadata. There is no
configurable ignore-file feature; older snapshots containing metadata are rejected.

Current limits are 8 MiB per decompressed object, 4 KiB per ref-update request,
32 concurrent authenticated repository requests, and graph validation bounded
by 10,000 object references and 256 MiB of decompressed content per update.
Uploads are intentionally uncompressed on the wire; storage is compressed.
The server has header/body/idle timeouts and validates object hashes again when
reading stored data. It does not yet have storage quotas or garbage collection;
uploaded objects that never become reachable remain on disk.

One server process owns a data directory. `.server.lock` blocks a second process,
and atomic file replacement prevents partial object/ref files from being
published. This is single-host storage, not a distributed database or a guarantee
against power-loss corruption. If a process is killed, confirm it is no longer
running before removing its stale lock; orphan `.pending-*` files are safe to
remove while the server is stopped. Back up the data directory while stopped.
The storage directory must be administrator-controlled, not writable by clients.

## Verification

```sh
go test ./...
go test -race ./server
go vet ./...
```

Tests cover the wire format, hash verification, auth/ACL failures, interrupted
uploads, expected-tip conflicts, complete graph checks, two concurrent HTTP
writers, persistence across restarts, permission changes on restart, and the
single-process storage lock. Integration tests bind ephemeral loopback ports.
