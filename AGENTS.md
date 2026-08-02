# Telegram Message Sync Bot

## Project Overview

Go bot that archives Telegram channel messages and cross-posts to social media (BlueSky, Mastodon, Twitter).

## Tech Stack

- **Language**: Go 1.26
- **Telegram**: `github.com/go-telegram/bot`
- **Database**: SQLite via GORM (`gorm.io/gorm`, `gorm.io/driver/sqlite`)
- **Social Media**: BlueSky (`go-atproto`), Mastodon (`go-mastodon`), Twitter (`gotwi`)
- **CLI**: Cobra (`github.com/spf13/cobra`)
- **Config**: YAML (`gopkg.in/yaml.v3`)

## Project Structure

```
├── main.go                  # Entry point
├── internal/
│   ├── Database/            # SQLite layer (Database.go, Messages.go, SyncRecords.go)
│   ├── Entity/              # Data models (Config, Message, SyncRecord)
│   └── Handler/             # Bot command handlers (Admin, Start, Version)
├── pkg/
│   ├── SocialMediaUtils/    # BlueSky, Mastodon, Twitter integrations
│   ├── FileUtils/           # File utilities
│   ├── LogUtils/            # Logging utilities
│   ├── StrUtils/            # String utilities
│   └── TgUtils/             # Telegram message formatting
├── config/
│   ├── config.yaml          # Runtime configuration
│   └── templates/           # Config templates
├── docs/
│   ├── memories/            # Architecture, design, tech-stack docs
│   └── implementation-plans/ # Implementation plan files
├── deploy/systemd/          # Systemd service files
└── scripts/migration/       # Data migration scripts
```

## Build & Run

```shell
go build -o tg main.go
./tg sync -c ./config/config.yaml
```

## Verification

- Run `go build ./...` and `go vet ./...` to verify code compiles and is correct
- Run `go test ./...` to execute tests
- After any change, confirm the build and tests pass before finishing

## Conventions

- Keep code simple and direct
- When implementing from a plan, execute tasks atomically — one at a time
- Always run verification steps before moving to the next task
- Update `docs/memories/` if architectural decisions change during implementation
- Update `docs/implementation-plans/` with execution records after completing each task
