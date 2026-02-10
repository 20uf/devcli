# devcli

CLI for interactive access to AWS ECS Fargate containers.

Dynamically discovers clusters, services, tasks and containers — no static configuration needed.

## Install

### One-liner (recommended)

```bash
# Stable release
curl -fsSL https://raw.githubusercontent.com/20uf/devcli/main/install.sh | sh -s

# Pre-release (alpha, beta, rc)
curl -fsSL https://raw.githubusercontent.com/20uf/devcli/main/install.sh | sh -s -- --pre-release
```

### Manual download

Download the binary for your platform from [Releases](https://github.com/20uf/devcli/releases) and place it in your `$PATH`.

### From source

```bash
git clone git@github.com:20uf/devcli.git
cd devcli
make build
sudo mv devcli /usr/local/bin/
```

## Usage

### Connect to a container (interactive)

```bash
devcli connect
```

This will guide you through:
1. Select an ECS cluster
2. Select a service
3. Auto-select a running task
4. Auto-select the target container (defaults to `php` if present)
5. Open an interactive shell

### Connect with flags (non-interactive)

```bash
devcli connect --cluster my-cluster --service my-service --container php
```

### AWS profile and region

```bash
devcli connect --profile my-sso-profile --region eu-west-1
```

### Available flags

| Flag | Description |
|---|---|
| `--cluster` | ECS cluster name (skip selection) |
| `--service` | ECS service name (skip selection) |
| `--container` | Container name (skip selection) |
| `--shell` | Shell command (default: `su -s /bin/sh www-data`) |
| `--profile` | AWS profile |
| `--region` | AWS region |

### Update

```bash
# Update to latest stable
devcli update

# Update to latest pre-release
devcli update --pre-release
```

### Version

```bash
devcli version
```

## Prerequisites

- AWS CLI v2 with [Session Manager plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html)
- AWS SSO configured (`aws sso login --profile <profile>`)
- ECS Exec enabled on target services

## Build

```bash
make build
```

## Release

Releases are managed with [GoReleaser](https://goreleaser.com/). Tag a version to trigger a release:

```bash
git tag v1.0.0
git push --tags
```

## Roadmap

- [ ] **Connect BDD** — Se connecter à une base de données dans un cluster (même UX que connect)
- [ ] **Consulter SSM** — Naviguer et lire les paramètres AWS SSM Parameter Store (lecture seule)
- [ ] **CodePipeline** — Déclencher un pipeline CodePipeline et suivre son exécution en temps réel
- [ ] **GitHub Actions status** — Suivre l'état d'avancement d'un job GitHub Actions en temps réel
- [ ] **Deploy workflow inputs** — Détecter automatiquement les inputs d'un workflow et proposer un formulaire interactif (ex: Target Environment)
- [ ] **Mode verbose (`--verbose`)** — Afficher toutes les commandes exécutées, appels API et réponses AWS/GitHub pour le debug
- [ ] **Stream logs GitHub Actions** — Streamer les logs d'exécution d'un workflow run en temps réel dans le terminal

## Contributors

A special thanks to all contributors:

- **Thomas Talbot** — [thomas.talbot@ioni.tech](mailto:thomas.talbot@ioni.tech)
