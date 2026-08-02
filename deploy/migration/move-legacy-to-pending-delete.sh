#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ARCHIVE_ROOT="${1:-$REPO_ROOT/archives}"

cd "$ARCHIVE_ROOT"

PENDING_DIR="260218-legacy-pending-delete"
ZIP_PATH="260218-old-markdown-archives.zip"

legacy_files=$(find person channel -maxdepth 1 -type f -name '*.md' | sort || true)
legacy_count=$(printf "%s\n" "$legacy_files" | sed '/^$/d' | wc -l | tr -d ' ')

echo "legacy_count=$legacy_count"

if [ "$legacy_count" -gt 0 ]; then
  mkdir -p "$PENDING_DIR/person" "$PENDING_DIR/channel"

  if [ ! -f "$ZIP_PATH" ]; then
    printf "%s\n" "$legacy_files" | zip -q -@ "$ZIP_PATH"
    echo "backup_created=$ZIP_PATH"
  else
    echo "backup_exists=$ZIP_PATH"
  fi

  while IFS= read -r f; do
    [ -z "$f" ] && continue
    mkdir -p "$PENDING_DIR/$(dirname "$f")"
    mv "$f" "$PENDING_DIR/$f"
    echo "moved=$f -> $PENDING_DIR/$f"
  done <<EOF
$legacy_files
EOF
fi

pending_count=$(find "$PENDING_DIR" -type f -name '*.md' | wc -l | tr -d ' ')
echo "pending_count=$pending_count"
