# Deploying Consensus to Fly.io

Consensus is a single Go binary that, on boot, runs its migrations and seeds all
content (lessons, interview questions, checker challenges) — **idempotently**. So a
deploy is just: ship the image, point it at a Postgres, set a couple of secrets.

> **Why the image is ~1 GB and ships the Go toolchain:** the in-app code checker runs
> `go test` on the server. The runtime image therefore includes Go (see `Dockerfile`),
> and the VM is sized to **1 GB** (`fly.toml`) so compiling a test binary doesn't OOM.
> Because the checker executes code on the server, **always set `APP_USER` /
> `APP_PASSWORD`** so the app is gated behind HTTP Basic Auth — don't run it as a
> wide-open public URL.

## 1. Prerequisites (one time)

```bash
# Install the Fly CLI
brew install flyctl            # macOS;  or: curl -L https://fly.io/install.sh | sh
fly auth signup                # or: fly auth login
```

## 2. Create the app

From the repo root (`/Users/gabriel/developer/distributed_engineer`):

```bash
# Reserve the app name (fly.toml already has app="consensus"; pick a unique name).
fly apps create consensus      # if "consensus" is taken, choose another and edit fly.toml's `app`
```

## 3. Create and attach Postgres (sets DATABASE_URL automatically)

```bash
fly postgres create --name consensus-db --region iad   # match fly.toml primary_region
fly postgres attach consensus-db --app consensus        # injects DATABASE_URL as a secret
```

## 4. Set secrets

```bash
# REQUIRED — gates the app (the checker runs code on the server):
fly secrets set APP_USER="you" APP_PASSWORD="a-long-random-password" --app consensus

# OPTIONAL — turns on the "Get a Claude review" button in the checker:
fly secrets set ANTHROPIC_API_KEY="sk-ant-..." --app consensus
```

## 5. Deploy

```bash
fly deploy --app consensus     # builds the Dockerfile remotely and ships it
```

On first boot the logs will show: `running migrations` → `connected to postgres` →
`seeding curriculum` → `synced lessons` → `synced code challenges` →
`synced interview questions` → `basic auth enabled` → `listening`.

## 6. Use it

```bash
fly open --app consensus       # opens https://consensus.fly.dev
fly logs  --app consensus      # tail logs
```

Log in with the `APP_USER` / `APP_PASSWORD` you set. Everything works from anywhere:
Today, Roadmap, Resources, Quiz, Checker, all 70 lessons with Interview prep, and the
in-app `go test` checker.

## Notes & costs

- **Scale-to-zero:** `fly.toml` sets `min_machines_running = 0`, so the app sleeps when
  idle and wakes on the next request (a few seconds of cold start). Cheap for personal
  use. Set it to `1` if you want it always warm.
- **First checker run:** the image pre-warms the Go build cache, so checks run in
  ~1–2 s instead of a ~15 s cold compile.
- **Schema/content changes** redeploy safely — migrations and the seeders are
  idempotent and never touch your progress data.
- **Backups:** `fly postgres` includes daily snapshots; see `fly postgres list`.

## Re-deploying after changes

```bash
fly deploy --app consensus
```

That's it — no git push required; `fly deploy` builds from your local source.
