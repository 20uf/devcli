# devcli

CLI for interactive access to AWS ECS Fargate containers.

Dynamically discovers clusters, services, tasks and containers — no static configuration needed.

## Install

### From GitHub Releases

Download the latest binary from [Releases](https://github.com/20uf/devcli/releases) and place it in your `$PATH`.

### From source

```bash
go install github.com/20uf/devcli@latest
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
devcli update
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
