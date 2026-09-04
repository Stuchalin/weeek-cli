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

reset_fixture() {
    cp "$repo_root/scripts/check-plugin.sh" "$fixture_root/scripts/check-plugin.sh"
    cp "$repo_root/plugin/.claude-plugin/plugin.json" "$fixture_root/plugin/.claude-plugin/plugin.json"
    cp "$repo_root/plugin/.codex-plugin/plugin.json" "$fixture_root/plugin/.codex-plugin/plugin.json"
    cp "$repo_root/plugin/skills/weeek/SKILL.md" "$fixture_root/plugin/skills/weeek/SKILL.md"
    cp "$repo_root"/plugin/skills/weeek/references/*.md "$fixture_root/plugin/skills/weeek/references/"
    cp "$repo_root/.claude-plugin/marketplace.json" "$fixture_root/.claude-plugin/marketplace.json"
    cp "$repo_root/.agents/plugins/marketplace.json" "$fixture_root/.agents/plugins/marketplace.json"
}

set_json_value() {
    path=$1
    json_path=$2
    value=$3
    python3 - "$fixture_root/$path" "$json_path" "$value" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
data = json.loads(path.read_text(encoding="utf-8"))
parts = sys.argv[2].split("/")
target = data
for part in parts[:-1]:
    target = target[int(part)] if isinstance(target, list) else target[part]
last = parts[-1]
if isinstance(target, list):
    target[int(last)] = json.loads(sys.argv[3])
else:
    target[last] = json.loads(sys.argv[3])
path.write_text(json.dumps(data), encoding="utf-8")
PY
}

replace_text() {
    path=$1
    old=$2
    new=$3
    python3 - "$fixture_root/$path" "$old" "$new" <<'PY'
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
text = path.read_text(encoding="utf-8")
if sys.argv[2] not in text:
    raise SystemExit(f"missing fixture text: {sys.argv[2]}")
path.write_text(text.replace(sys.argv[2], sys.argv[3], 1), encoding="utf-8")
PY
}

reset_fixture

"$fixture_root/scripts/check-plugin.sh" >/dev/null

if output=$(WEEEK_PLUGIN_VERSION=999.0.0 "$fixture_root/scripts/check-plugin.sh" 2>&1); then
    echo "check-plugin test: mismatched release version was accepted" >&2
    exit 1
fi
case "$output" in
    *"does not match release version 999.0.0"*) ;;
    *)
        echo "check-plugin test: expected release version mismatch, got: $output" >&2
        exit 1
        ;;
esac

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
    reset_fixture
}

printf 'not json\n' >"$fixture_root/plugin/.codex-plugin/plugin.json"
assert_rejected "invalid JSON"

set_json_value "plugin/.codex-plugin/plugin.json" "description" '""'
assert_rejected "missing description"

set_json_value "plugin/.codex-plugin/plugin.json" "skills" '"./other-skills/"'
assert_rejected "skills must be ./skills/"

set_json_value "plugin/.codex-plugin/plugin.json" "name" '"other"'
assert_rejected "manifest names must be weeek"

set_json_value "plugin/.codex-plugin/plugin.json" "version" '"999.0.0"'
assert_rejected "versions do not match"

set_json_value "plugin/.claude-plugin/plugin.json" "version" '"invalid"'
set_json_value "plugin/.codex-plugin/plugin.json" "version" '"invalid"'
set_json_value ".claude-plugin/marketplace.json" "plugins/0/version" '"invalid"'
assert_rejected "version is not semver"

replace_text "plugin/skills/weeek/SKILL.md" "---" "not-frontmatter"
assert_rejected "has no YAML frontmatter"

replace_text "plugin/skills/weeek/SKILL.md" "0.0.0" "version-not-mentioned"
assert_rejected "plugin version is not mentioned"

rm "$fixture_root/plugin/skills/weeek/references/tasks.md"
assert_rejected "missing reference"

replace_text "plugin/skills/weeek/SKILL.md" "tasks.md" "task-reference.md"
replace_text "plugin/skills/weeek/SKILL.md" "tasks.md" "task-reference.md"
assert_rejected "does not link tasks.md"

set_json_value ".claude-plugin/marketplace.json" "owner/name" 'null'
assert_rejected "Claude marketplace owner is missing"

set_json_value ".claude-plugin/marketplace.json" "plugins/0/source" '"./other"'
assert_rejected "Claude marketplace source must be ./plugin"

set_json_value ".claude-plugin/marketplace.json" "plugins/0/version" '"999.0.0"'
assert_rejected "Claude marketplace version does not match manifests"

set_json_value ".agents/plugins/marketplace.json" "plugins/0/source/path" '"./other"'
assert_rejected "Codex marketplace source path must be ./plugin"

set_json_value ".agents/plugins/marketplace.json" "plugins/0/policy/installation" '"BROKEN"'
assert_rejected "invalid Codex installation policy"

set_json_value ".agents/plugins/marketplace.json" "plugins/0/category" '""'
assert_rejected "Codex marketplace category is missing"

echo "check-plugin tests passed"
