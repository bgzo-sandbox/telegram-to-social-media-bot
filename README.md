# Telegram message sync bot

![](https://raw.githack.com/bGZo/assets/dev/2025/202503011548103.png)


This is a telegram bot for archiving message from bot and sync to social media.

Of course, you can use it as a simple telegram bot for syncing message from telegram.

## Why

Telegram have a great API and contents than other social media. There's less bot, business and ads.

So I spend a lot of time on it. The messges saved in `SavedMessage` is not enough. You should manage what you read. And post what you think to more social media.

That's what this bot do.

## How 

- Golang
- Gorm
- Sqlite
- Telebot
- Social media API
- Vibe code

## Roadmap

- [x] Messages archive
  - [x] Rich text from Telegram
  - [x] Media download
- [x] Notification
- [x] Database supported(sqlite)
- [ ] Sync social media (beta)
  - [x] Twitter
  - [x] Mastodon
  - [x] BlueSky
  - [ ] Instagram
  - [ ] Facebook
  - [ ] Thread
  - [ ] Reddit
  - [ ] Douban
  - [ ] Okjike
  - [ ] Weibo
  - [ ] Douyin
  - [ ] Bilibili
  - [ ] Xiaohongshu
  - [ ] Coolapk
  - [ ] Zhihu
  - [ ] V2Ex

## Quick start

```shell
# install dependencies
go mod tidy

# build the result
go build -o tg main.go

# give the right to run. 
chmod +x ./tg

# run bot
./tg sync -c ./config/config.yaml
```

### Pipeline execution mode (optional)

Set in `config/config.yaml`:

```yaml
pipeline:
  executionMode: serial # serial | async_experimental
```

- `serial`: default stable mode
- `async_experimental`: experimental mode (should keep equivalent behavior currently)

## Archive migration (DB -> local archives)

Use unified CLI commands to rebuild/archive Markdown files from SQLite (`archives/archive.db`) into local archives directories.

### 1) Backfill all messages from DB

```shell
./tg migrate backfill -c ./config/config.yaml
```

What it does:

- Reads all messages from DB;
- Generates/ensures `source_id/message_id.md` in `archives/person` or `archives/channel`;
- Skips existing files;
- Verifies key consistency after backfill.

### 2) Move legacy root markdown files to pending-delete directory

The executed shell operation is now persisted in:

- `scripts/migration/move-legacy-to-pending-delete.sh`

Run it:

```shell
bash ./scripts/migration/move-legacy-to-pending-delete.sh
```

What it does:

- Scans only legacy root files: `archives/person/*.md` and `archives/channel/*.md`;
- Creates backup zip first (if missing): `archives/260218-old-markdown-archives.zip`;
- Moves legacy files to: `archives/260218-legacy-pending-delete`;
- Does **not** touch new-format files under `source_id/message_id.md` directories.

### 3) Move legacy root markdown files by CLI (optional alternative)

```shell
./tg migrate move-legacy -c ./config/config.yaml
```

### 4) Quick verification

```shell
find ./archives/person -maxdepth 1 -type f -name '*.md' | wc -l
find ./archives/channel -maxdepth 1 -type f -name '*.md' | wc -l
find ./archives/260218-legacy-pending-delete -type f -name '*.md' | wc -l
```

### Optional: run in background using nohup

```shell
nohup ./tg sync -c ./config/config.yaml > bot.log 2>&1 &

# kill background
pkill -f tg
```

### Optional: run in background using nohup

Rename `tg-sync.service.bak` to `tg-sync.service`, then fill token.

```shell
cp deploy/systemd/tg-sync.service  ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user start tg-sync.service
systemctl --user enable tg-sync.service
```

Add following config:

```shell
[Unit]
Description=tg message sync bot for %i.
After=network.target

[Service]
Type=simple
User=%i
Restart=on-abort
Environment=http_proxy=192.168.31.20:10800
Environment=https_proxy=192.168.31.20:10800
ExecStart=/home/bgzo/workspaces/telegram-message-sync/tg sync -c /home/bgzo/workspaces/telegram-message-sync/config/config.yaml

[Install]
# WantedBy=multi-user.target
WantedBy=graphical-session.target
```

Then restart systemd and enable `tg@username`

```shell
systemctl daemon-reload
systenctl start tg@bgzo
systenctl enable tg@bgzo
```

## ALternatives

- https://github.com/leaperone/MultiPost-Extension

