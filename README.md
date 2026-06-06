# aboftybot (Aerospike edition)

A small IRC bot backed by [Aerospike](https://aerospike.com/). This is a **playground project** — a place to practice Go, the Aerospike Go client, and GitOps-style Kubernetes deploys on my homelab k3s cluster.

It is not aiming to become a general-purpose bot framework or a Discord-style bot people adopt. It does one channel, a handful of dot-commands, and that's enough.

The original lived in a monorepo; this is a refactor with cleaner layout, updated dependencies, and Kustomize overlays for dev and prod.

## What it does

Connects to [Snoonet](https://snoonet.org/), joins a channel, and responds to a few commands. Every message in the channel increments a per-channel daily line counter stored in Aerospike.

| Command | Description |
|---------|-------------|
| `.go [user]` | Tell someone (or yourself) to leave |
| `.lines [YYYY-MM-DD]` | Line count for the channel today, or on a given date |
| `.gtfb [user]` | Random insult from the `gtfb` set, directed at a user |
| `.topl` | Top 5 busiest days in the channel |
| `.pastl [N]` | Line counts for the last N days (default 7) |

## Why this repo exists

Mostly to sharpen skills I'd use elsewhere, in a low-stakes app:

- **Go** — hand-rolled IRC parsing, env-based config, fail-fast startup, table-driven tests
- **Aerospike** — `Operate` + `AddOp` for atomic counters, secondary indexes, queries, batch gets
- **GitOps** — Kustomize base + overlays, secret generators, image tag rewrites, CI-built container images

If you're looking for a polished IRC bot library, look elsewhere. If you want a small, readable example of "bot talks to Aerospike on k8s," you're in the right place.

## Aerospike data model

Namespace: `aboftybot` (cluster manifests live in [`data/aerospike`](../../data/aerospike)).

| Set | Primary key | Bins | Used by |
|-----|-------------|------|---------|
| `line_counts` | `{channel}\|{YYYY-MM-DD}` | `channel`, `date`, `count` | `.lines`, `.topl`, `.pastl`, background counter |
| `gtfb` | (varies) | `insult` | `.gtfb` |

On startup the bot ensures a secondary index `lineCountIdx` exists on `line_counts.channel`, so `.topl` can query by channel.

**Worth knowing:** line count PKs used to be date-only (`2026-06-05`). They're now `{channel}|{date}` for per-channel isolation. See `historical/migrate_pk_line_count.go` if you ever need to reason about old data.

## Layout

```
main.go              # IRC connection loop and command dispatch
utils/               # Aerospike helpers (db, line counts, insults)
deploy/
  base/              # Shared Deployment
  overlays/dev/      # Namespace: aboftybot-dev, staging Aerospike
  overlays/prod/     # Namespace: aboftybot, prod Aerospike
historical/          # Old dumps, one-off migration scripts, dev notes (gitignored)
```

## Configuration

Required environment variables:

| Variable | Source | Purpose |
|----------|--------|---------|
| `IRC_SERVER` | ConfigMap | IRC host:port (e.g. `irc.snoonet.org:6667`) |
| `IRC_NICK` | ConfigMap | Bot nickname |
| `IRC_CHANNEL` | ConfigMap | Channel to join |
| `IRC_PASSWORD` | Secret | NickServ identify password |
| `DB_HOST` | ConfigMap | Aerospike host (in-cluster DNS) |
| `DB_USER` | Secret | Aerospike username |
| `DB_PASS` | Secret | Aerospike password |

Copy `deploy/overlays/<env>/secret.env.example` to `secret.env` and fill in real values before applying. Secret files are gitignored.

## Local development

**Run tests:**

```bash
go test ./...
```

**Build the binary:**

```bash
go build -o aboftybot .
```

**Build a container image:**

```bash
docker build -t aboftybot:dev-test .
```

For a quick IRC smoke test you'd need all env vars set and a reachable Aerospike instance. Most iteration happens against the dev overlay on the homelab cluster instead.

## Deploying

Run these from this directory (`apps/aboftybot_aerospike/`).

**Preview manifests:**

```bash
kubectl kustomize deploy/overlays/dev
kubectl kustomize deploy/overlays/prod
```

**Apply:**

```bash
kubectl apply -k deploy/overlays/dev
# or
kubectl apply -k deploy/overlays/prod
```

| Overlay | Namespace | Aerospike | Image |
|---------|-----------|-----------|-------|
| `dev` | `aboftybot-dev` | `asdb.asdb-staging.svc.cluster.local` | Local `aboftybot:dev-test` on k3s nodes |
| `prod` | `aboftybot` | `asdb.asdb-prod.svc.cluster.local` | `ghcr.io/aboft/aboftybot-go:<tag>` |

### Dev image workflow (k3s)

k3s nodes don't share a registry, so the dev overlay rewrites the image to a locally built tag. After building on your machine, import the image onto each node:

```bash
for node in johto hoenn paldea; do
  echo "==> $node"
  docker save aboftybot:dev-test | ssh "$node" \
    'sudo -n /usr/local/bin/k3s ctr images import -'
done
```

Then apply the dev overlay. Images don't replicate between nodes — every node that might schedule the pod needs the import.

## CI

GitHub Actions (`.github/workflows/ci.yaml`):

- **On every push/PR to `main`:** `go test ./...`
- **On version tags (`v*`):** build and push `ghcr.io/aboft/aboftybot-go:<tag>`

To ship prod: tag a release, bump `newTag` in `deploy/overlays/prod/kustomization.yaml`, and apply.

## Notes for future me

- This is a playground. Don't over-invest in features nobody asked for.
- `historical/` holds SQL dumps, JSON exports, and migration scripts from the pre-Aerospike era — useful reference, not part of the running app.
- `IncrementLineCount` uses `Operate` + `AddOp` instead of read-then-write to avoid races on busy channels.
- IRC parsing is hand-rolled and tested in `main_test.go` — good enough for Snoonet, not a general-purpose IRC library.
- Aerospike client is on v8; `go.mod` still lists v6 as a leftover indirect dep from tooling — safe to clean up when convenient.
