# gitli Security Audit

Date: 2026-10-11
Scope: Go backend under `internal/` (read-only static review, no code modified).
Method: manual review + `grep`/`rg` of SQL construction, `exec.Command*`, `filepath.Join`,
route table, secret handling and webhook delivery.

Legend — Severity: **high** / **medium** / **low** / **info**.

---

## 1. SQL injection

**No findings.**

- No SQL statement was found built with `fmt.Sprintf` or `+` string
  concatenation that embeds user input. The `Sprintf(SELECT|INSERT|...)`
  search returned zero matches.
- Every dynamic query uses placeholders (`?`) and `st.Rebind(...)`, then passes
  values as query arguments. Representative examples:
  - `internal/service/repo/repo.go:82`, `:105`, `:120`
  - `internal/service/rbac/manage.go:53`, `:161`, `:267`
  - `internal/service/registry/registry.go:88`, `:178`, `:212`
  - `internal/service/lfs/lfs.go:113`, `:125`, `:157`
  - `internal/service/package/blob.go:87`, `:136`
- The only string-concatenation in a query is `repoColumns` (a compile-time
  constant column list) at `internal/service/repo/repo.go:105,111,120`; no user
  input is involved.
- `internal/transport/http/handlers_admin.go:26` concatenates a **hardcoded,
  allow-listed** table name into `SELECT COUNT(*) FROM <table>`; the table set is
  a fixed literal slice (`handlers_admin.go:38-41`), so this is not injectable.
- `internal/service/comment/comment.go:70` rebuilds `WHERE <column> = ?` from an
  internal `column` variable (not request input) with a bound parameter.

Conclusion: all `Rebind(` usages reviewed are parameterized. No unparameterized
query found.

---

## 2. Command injection

### 2.1 HIGH — Git argument injection via user-controlled `ref`/`base`/`head` (arbitrary file write)

- Locations:
  - `internal/service/git/git.go:77` — `s.runner.RunStream(ctx, path, "show", ref+":"+subPath)`
  - `internal/service/git/git.go:68` — `"ls-tree", "--long", "-z", target`
  - `internal/service/git/git.go:86` — `"log", fmt.Sprintf("-%d", limit), "--format="+format, ref`
  - `internal/service/git/git.go:95` — `"diff", "--no-color", base, head`
  - `internal/service/git/git.go:100` — `"blame", "--porcelain", ref, "--", subPath`
  - `internal/service/git/graph.go:31` — `"log", fmt.Sprintf("-%d", limit), "--format="+fs, ref`
  - `internal/service/git/language.go:49` — `"ls-tree", "-r", "--long", "-z", ref`
  - `internal/service/ide/ide.go:57,122,149` — `ls-tree`/`show`/`grep` with `ref`/`spec`
- Reachable from unauthenticated-to-read routes:
  - `ref` from URL path `/{owner}/{repo}/tree/{ref}` → `handlers_git.go:21`
  - `ref` from URL path `/{owner}/{repo}/blob/{ref}/*` → `handlers_git.go:41`
  - `ref` query on `/commits`, `/ide/file`, `/ide/tree`, `/ide/symbols`
    (`handlers_git.go:63`, `handlers_ide.go:21,49,70`)
  - `base`/`head` query on `/diff` → `handlers_git.go:103-104`
- The code correctly avoids a shell (args are passed separately and
  `exec.Command` is used), so **shell** injection is not possible. However, the
  value is passed as a positional git argument with no `--` end-of-options
  guard, so a value beginning with `-` is interpreted by git as an **option**.
- Empirically verified against the system git:
  - `git show --output=/tmp/pwn HEAD` creates `/tmp/pwn` (arbitrary path).
  - `git diff --no-color --output=/tmp/pwn3` creates `/tmp/pwn3`.
  - `git log --output=/tmp/pwn2 HEAD` creates `/tmp/pwn2`.
- Impact: a user with only **read** access to a repo can write a file at an
  arbitrary absolute path writable by the gitli process. `--output=<path>` and
  similar options turn this into arbitrary file write / potential RCE (e.g.
  overwriting hooks, authorized_keys, config). Requires the process user to have
  write permission at the target path, but the target is attacker-chosen.
- Recommended fix: reject refs/paths beginning with `-` (or any leading dash) in
  the git service, and/or insert `--` before revision arguments where git
  supports it. Prefer validating `ref` against `git check-ref-format` /
  `rev-parse --verify` before use. Do **not** rely on the current argument
  split alone. Example guard: reject values matching `^-`, or use
  `--end-of-options` (git ≥ 2.24) where applicable.

### 2.2 LOW — `exec.Command` usages are otherwise correct

Reviewed all `exec.Command` / `exec.CommandContext` sites:

- `internal/infra/gitcmd/gitcmd.go:126` — `exec.CommandContext(ctx, r.bin, args...)`,
  no shell; env whitelist (`gitcmd.go:21-28`, `:150-162`) excludes
  `GIT_SSH_COMMAND`, `GIT_EXTERNAL_DIFF`, `GIT_CONFIG*`, etc. Good.
- `internal/app/ssh_adapter.go:62` — verb is selected from a fixed allow-list
  (`splitGitCommand`, `ssh_adapter.go:70-78`) and the repo path is validated by
  `ValidateRepoPath`. Good.
- `internal/service/git/archive.go:21` — args built by `archiveArgs`, `ref` is a
  positional argument placed **after** `--format=...`; a `ref` beginning with `-`
  is likewise interpreter as an option (same class as 2.1; archive is
  read-only + `--output` file write). Included in finding 2.1.
- `internal/service/git/smarthttp.go:40` — `service` is checked by
  `IsSmartService`; `sub` derived from it; safe.
- No use of `sh`, `-c`, or string concatenation into a command anywhere.

No separate command-injection finding beyond 2.1.

---

## 3. Path traversal

### 3.1 No findings (well validated)

- Git repo paths: `fsrepo.sanitize` (`internal/infra/fsrepo/fsrepo.go:80-88`)
  strips separators and `..`, and `gitcmd.ValidateRepoPath`
  (`internal/infra/gitcmd/validate.go:12-31`) rejects `..`, NUL, `~`, `\` and
  enforces containment under the repos root. Owner/repo names are additionally
  validated by `secutil.ValidateName` (`internal/infra/secutil/secutil.go:21-35`).
- LFS: `lfs.Path` requires a 64-char lowercase hex OID via `VerifyOID`
  (`internal/service/lfs/lfs.go:43-63`), so `oid[:2]`/`oid[2:4]` cannot escape.
- Package blobs: `pkgstore` validates the SHA-256 digest
  (`internal/service/package/blob.go:148-164`) and derives the sharded path from
  it (`blob.go:75,145`); digests are recomputed from content, not user-supplied.
- OCI registry: `ValidateDigest` gates all digest paths
  (`internal/service/registry/registry.go:63,117,183,211,262`) and
  `GetBlob`→`blobs.Open` re-validates.
- Backup restore: `safeJoin` (`internal/infra/backup/backup.go:295-308`)
  rejects `..` and absolute escapes; symlink handling is out of scope of the
  tar types processed (only TypeDir/TypeReg handled, `backup.go:279-291`).
- `secutil.SafeJoin`/`EnsureWithin` (`secutil.go:38-54`) provide containment for
  callers that need it.

Note (info, not a finding): `k`-short digests are enforced to full 64 hex, so
truncated-digest path tricks are not possible.

---

## 4. Authorization gaps (HTTP route review)

Routes are registered in `internal/transport/http/server.go`. All `/api/v1`
routes pass through `AuthenticateMiddleware` (`server.go:69`) and, per-handler,
`requireUser` and/or `loadRepoAndPerm`/`resolveRepo`/`ociRepoAccess`/
`RequirePermission`/`RequireOrgPermission`.

### 4.1 Reviewed and protected

- `/api/v1/admin/*` (`server.go:74-77`) → `requireAdmin` inside handlers
  (`handlers_admin.go:12-22,34,66`). Good.
- `/api/v1/orgs` mutations (`server.go:103-115`) → `RequireOrgPermission("org","admin")`.
- `/api/v1/repos` mutations:
  - create (`server.go:119`) → `requireUser` (`handlers_repo.go:24`).
  - delete/transfer (`server.go:123,125`) → `loadRepoAndPerm(..., PermAdmin)`.
  - fork (`server.go:124`) → `requireUser` + `loadRepoAndPerm(PermRead)`.
  - branch/tag protections, collaborators (`server.go:140-142`) → `PermAdmin`.
  - rbac collaborators/branch-rules (`server.go:145-150`) → `RequirePermission("settings","write")`.
  - wiki create/update/delete (`server.go:154-156`) → `resolveRepo(PermWrite)`.
  - releases (`server.go:159-160`) → `resolveRepo(PermWrite)`.
  - lfs upload (`server.go:164`) → `loadRepoAndPerm(PermWrite)`; batch → `PermWrite`.
  - packages publish (`server.go:167`) → `loadRepoAndPerm(PermWrite)`.
  - mirrors/imports (`server.go:169,171`) → `loadRepoAndPerm(PermWrite)`.
  - pipelines create (`server.go:133`) → `loadRepoAndPerm(PermWrite)`.
  - webhooks create (`server.go:143`) → `loadRepoAndPerm(PermAdmin)`; list redacts
    the secret (`handlers_webhook.go:44-46`).
- `/api/v1/gists` (`server.go:175-180`) → `requireUser`; private gist access is
  ownership-checked (`handlers_wiki_gist_release.go:172`).
- `/api/v1/issues` and `/api/v1/pulls` mutations → `resolveRepo(PermWrite)` for
  create/state/merge; comments/reviews require only `PermRead` + `requireUser`
  (intentional: read-access users may comment/review, standard for such platforms).
- `/api/v1/packages/...` download → `loadRepoAndPerm(PermRead)`.
- OCI `/v2/...` (`server.go:44-53`) → `ociRepoAccess` with `PermRead`/`PermWrite`
  (`handlers_registry.go:22-61`).
- Git smart HTTP (`server.go:56-65`) → `gitRepoAccess` with `PermRead`/`PermWrite`
  (`handlers_githttp.go:82-136`).
- Public routes `badges`, `feed` and `/healthz`, `/metrics` are intentionally
  unauthenticated (`server.go:40-41,209-211`).

### 4.2 Observations (not exploitable as written)

- `handleListPipelineJobs` (`server.go:201`) only checks `requireUser` and does
  **not** verify the caller has access to the pipeline's repository: any
  authenticated user can list jobs for any pipeline id. Severity: **low**
  (information disclosure of CI job metadata across tenants). File:
  `internal/transport/http/handlers_ci.go:53-71`; middleware absent in
  `server.go:201`. Recommended fix: load the pipeline, resolve its repo and call
  `loadRepoAndPerm(PermRead)` before listing jobs.
- `handleListNotifications`/`handleMarkNotificationRead` correctly scope by
  `u.ID` (`handlers_webhook.go:56,75`).

No mutation route was found that performs a state change without either
`requireUser` + a permission check or an ownership check.

---

## 5. Secrets at rest

**No findings.**

- TOTP secret: encrypted with AES-256-GCM and stored with the `enc:` prefix
  (`internal/service/auth/auth.go:369-377`); read path requires the cipher and
  hex-decodes the sealed payload (`auth.go:380-396`). Recovery codes are stored
  as SHA-256 hashes (`auth.go:269-273`, `hashString` at `auth.go:398`). Used-code
  replay protection via `totp_used_codes` (`auth.go:310-322`).
- Tokens (PAT/session/API): generated with 32 random bytes and stored only as
  their hex SHA-256 hash; plaintext returned once at creation
  (`internal/infra/token/token.go:22-43`). Sessions store `token_hash`
  (`auth.go:330-331`). Verification is constant-time (`token.go:46-58`).
- Password: Argon2id via the `password` infra package; PHC-formated hashes.
- Logging: no `slog`/`log` call was found that logs a secret, token, password or
  TOTP value (grep over `internal/service` and `internal/transport` for
  secret/token/password/totp/key log fields returned nothing).
- Webhook secrets are not returned by the list endpoint
  (`handlers_webhook.go:44-46`), though they are stored in plaintext in the DB
  (by design; needed to compute HMAC). Severity: **info** — consider encrypting
  at rest, consistent with TOTP, if DB-at-rest threat is in scope.

Residual info-level note: webhook `secret` column is stored unhashed (must be
reversible for HMAC). This is expected but differs from token handling.

---

## 6. Webhook SSRF

### 6.1 MEDIUM — DNS-rebinding TOCTOU between validation and delivery

- `internal/service/webhook/webhook.go:146-168` (`validateURL`) resolves the
  host with `net.LookupIP` and rejects private/loopback/link-local addresses;
  called both on `Create` (`webhook.go:47`) and on every `Deliver`
  (`webhook.go:86`).
- Redirects are disabled (`webhook.go:37-39`, `ErrUseLastResponse`), good.
- Gap: the `http.Client` (`webhook.go:35-40`) uses the default transport with
  no custom `DialContext`/`Control`. The address check and the actual TCP dial
  are separate DNS lookups, so a hostname can resolve to a public IP at
  validation and a private IP at dial time (classic DNS rebinding). Additionally
  `validateURL` returns **nil** when DNS resolution fails
  (`webhook.go:160`: "allow unresolved; delivery will error"), so a host that
  is unresolvable at create-time is still accepted.
- Impact: SSRF against internal/loopback/metadata endpoints (e.g. cloud
  metadata `169.254.169.254`) if `isPrivate` is bypassed via rebinding.
  Severity: **medium** (attacker must control a DNS name; requires webhook
  create permission = repo admin).
- Recommended fix: resolve the host once and pin the resolved IP by dialing with
  a custom `net.Dialer.Control`/`DialContext` that re-checks the connected IP
  against the private-address predicate, or resolve + connect to the IP
  directly. Also do not silently allow unresolved hosts at create-time where a
  policy decision is being made.

### 6.2 MEDIUM — Mirror/import remote URLs are not SSRF-validated

- `internal/service/mirror/mirror.go:43-68` (`Create`) and
  `internal/service/import/import.go:57-80` (`Create`) accept an arbitrary
  `remote_url` / `source_url` with only a non-empty check.
- These URLs are later passed to git: `git fetch <url>` / `git push --mirror <url>`
  (`mirror.go:111,115`) and `git clone --bare <url>` (`import.go:120`).
- No scheme allow-list and no private-address check — unlike the webhook
  service, which at least attempts one. `file://`, `ext::`, `ssh://` to internal
  hosts, or `http(s)://` to internal services/cloud metadata are all possible.
- Impact: authenticated users with repo write access can make the server contact
  arbitrary internal endpoints (SSRF), read local repositories via `file://`,
  and potentially use `ext::`/`ssh://` transports. Severity: **medium**.
- Recommended fix: apply the same host/scheme validation as webhooks (allow only
  `http`/`https`/`git`/`ssh` as configured), reject private/metadata addresses,
  and consider disabling git's `ext::` transport
  (`GIT_PROTOCOL_FROM_USER=0` or `protocol.ext.allow=never`).

### 6.3 LOW — `git push --mirror` option handling

- `internal/service/mirror/mirror.go:115` passes the stored remote URL as a
  positional argument after `--mirror`. Since the URL is validated only for
  non-emptiness, a value beginning with `-` could be parsed as a git option
  (same class as finding 2.1). Constrain to proper URLs to eliminate this.

---

## Summary of counts

| Severity | Count | Findings |
|----------|-------|----------|
| high     | 1     | 2.1 Git argument injection / arbitrary file write via `ref`/`base`/`head` |
| medium   | 2     | 6.1 Webhook DNS-rebinding TOCTOU; 6.2 Mirror/import URL SSRF |
| low      | 2     | 4.2 Pipeline-jobs cross-tenant read; 6.3 `git push --mirror` option handling |
| info     | 1     | 5 Webhook secret stored reversibly (by design) |

Categories with **no findings**: **SQL injection (1)** and **Path traversal (3)**.

Note: the high-severity issue (2.1) is the file-write vector enabled by passing
untrusted revisions positionally to git; it is the single item that should be
fixed first. Categories 1, 3 and 5 are clean in this review. No code was
modified; no commits were made.
