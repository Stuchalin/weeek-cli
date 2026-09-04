#!/bin/sh

set -eu

repo_root=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
fixture_root=$(mktemp -d)
trap 'rm -rf "$fixture_root"' EXIT HUP INT TERM

mkdir -p \
    "$fixture_root/scripts" \
    "$fixture_root/plugin/.claude-plugin" \
    "$fixture_root/plugin/.codex-plugin" \
    "$fixture_root/plugin/skills/weeek/references" \
    "$fixture_root/.claude-plugin" \
    "$fixture_root/.agents/plugins"

cp "$repo_root/scripts/check-plugin.sh" "$fixture_root/scripts/check-plugin.sh"
cp "$repo_root/plugin/.claude-plugin/plugin.json" "$fixture_root/plugin/.claude-plugin/plugin.json"
cp "$repo_root/plugin/.codex-plugin/plugin.json" "$fixture_root/plugin/.codex-plugin/plugin.json"
cp "$repo_root/plugin/skills/weeek/SKILL.md" "$fixture_root/plugin/skills/weeek/SKILL.md"
cp "$repo_root"/plugin/skills/weeek/references/*.md "$fixture_root/plugin/skills/weeek/references/"
cp "$repo_root/.claude-plugin/marketplace.json" "$fixture_root/.claude-plugin/marketplace.json"
cp "$repo_root/.agents/plugins/marketplace.json" "$fixture_root/.agents/plugins/marketplace.json"

"$fixture_root/scripts/check-plugin.sh" >/dev/null

assert_rejected() {
    expected=$1
    if output=$("$fixture_root/scripts/check-plugin.sh" 2>&1); then
        echo "check-plugin test: invalid fixture was accepted: $expected" >&2
        exit 1
    fi
    case "$output" in
        *"$expected"*) ;;
        *)
            echo "check-plugin test: expected '$expected', got: $output" >&2
            exit 1
            ;;
    esac
}

printf 'not json\n' >"$fixture_root/plugin/.codex-plugin/plugin.json"
assert_rejected "invalid JSON"
cp "$repo_root/plugin/.codex-plugin/plugin.json" "$fixture_root/plugin/.codex-plugin/plugin.json"

python3 - "$fixture_root/plugin/.codex-plugin/plugin.json" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
manifest = json.loads(path.read_text(encoding="utf-8"))
manifest["version"] = "999.0.0"
path.write_text(json.dumps(manifest), encoding="utf-8")
PY
assert_rejected "versions do not match"
cp "$repo_root/plugin/.codex-plugin/plugin.json" "$fixture_root/plugin/.codex-plugin/plugin.json"

rm "$fixture_root/plugin/skills/weeek/references/tasks.md"
assert_rejected "missing reference"

echo "check-plugin tests passed"
