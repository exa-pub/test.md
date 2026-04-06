#!/bin/sh
set -e

# Persist ~/.claude.json and ~/.claude/ between container rebuilds
# Data lives in /workspaces/.claude-persist/ which survives rebuilds
PERSIST_DIR="${WORKSPACE_FOLDER:?WORKSPACE_FOLDER is not set}/.devcontainer/.persist"
mkdir -p "$PERSIST_DIR/.claude"
ln -sf "$PERSIST_DIR/.claude.json" ~/.claude.json
ln -sf "$PERSIST_DIR/.claude" ~/.claude

# Claude Code CLI
curl -fsSL https://claude.ai/install.sh | bash
