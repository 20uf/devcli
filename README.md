# devcli

> One-liner access to AWS ECS containers & GitHub Actions workflows from your terminal.

---

## What does it do?

`devcli` is a **developer productivity tool** that simplifies:

- 🔌 **Container access** — Interactive SSH to ECS Fargate containers without memorizing cluster/service/task names
- 🚀 **Workflow deployment** — Trigger GitHub Actions workflows with typed inputs (choice, boolean, string)
- 📊 **Status tracking** — Monitor workflow runs in real-time with live logs

No static configuration. Discovers resources dynamically from AWS/GitHub.

## Why?

**Before devcli:**
```bash
# Get cluster → get service → get tasks → pick container → connect
aws ecs list-clusters | jq '.clusterArns[0]' | xargs \
  aws ecs list-services --cluster | jq '.serviceArns[0]' | xargs \
  aws ecs list-tasks --cluster --service-name | jq '.taskArns[0]' | xargs \
  aws ecs describe-tasks --cluster --tasks | jq '.[0].containers[0].name'

# Then open SSH session manually
aws ssm-documents describe-document --name AWS-StartInteractiveCommand
```

**With devcli:**
```bash
devcli connect
# ✨ Guided through cluster → service → task → container → shell
```

---

## Quick start

### Install (one-liner)

**Stable release:**
```bash
curl -fsSL https://raw.githubusercontent.com/20uf/devcli/main/install.sh | sh
```

**Pre-release (alpha/beta):**
```bash
curl -fsSL https://raw.githubusercontent.com/20uf/devcli/main/install.sh | sh -s -- --pre-release
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for pre-release testing tips.

### Commands

#### Connect to a container

```bash
# Interactive (guided)
devcli connect

# Non-interactive (flags)
devcli connect --cluster prod --service api --container php --profile sso-prod
```

#### Deploy workflows

```bash
# Interactive (select workflow, inputs)
devcli deploy

# Non-interactive (all flags)
devcli deploy --workflow deploy.yml --branch main --input environment=prod
```

#### Monitor deployments

```bash
devcli status
# View tracked runs, stream logs, dismiss from dashboard
```

#### Version management

```bash
devcli update           # Update to latest stable
devcli update --pre-release  # Update to latest pre-release
devcli version          # Show current version
```

---

## Requirements

- AWS CLI v2 with [Session Manager plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html)
- AWS SSO configured (`aws sso login --profile <profile>`)
- ECS Exec enabled on target services
- GitHub CLI (`gh`) for workflow deployment

---

## Documentation

- **[Features & User Stories](documentation/US-INDEX.md)** — What devcli can do (quick index)
- **[All Phases Summary](documentation/ALL-PHASES-SUMMARY.md)** — Technical architecture & implementation details
- **[Installation](CONTRIBUTING.md#installation)** — Detailed setup guide
- **[Contributing](CONTRIBUTING.md)** — How to add features, report bugs

---

## Getting help

### Found a bug? 🐛

Create an issue: [github.com/20uf/devcli/issues](https://github.com/20uf/devcli/issues)

Include:
- devcli version: `devcli version`
- OS & shell: `uname -s && echo $SHELL`
- Error message & steps to reproduce

### Have a feature idea? ✨

1. Check [documentation/US-INDEX.md](documentation/US-INDEX.md) — your idea might already exist
2. Open discussion: [github.com/20uf/devcli/discussions](https://github.com/20uf/devcli/discussions)
3. If approved, create a User Story (US) — see [CONTRIBUTING.md](CONTRIBUTING.md)

### Want to contribute?

See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup
- Feature implementation workflow (TDD)
- Code review process

---

## Build from source

```bash
git clone git@github.com:20uf/devcli.git
cd devcli
make build
```

---

## Contributors

Thanks to everyone who made devcli better:

- **Thomas Talbot** — Core development
- **Contributors** — See [GitHub](https://github.com/20uf/devcli/graphs/contributors)

Want to join? See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## License

MIT License — see LICENSE file

---

**Status:** Pre-release (v0.10.x) — Use with feedback welcome! 🚀
