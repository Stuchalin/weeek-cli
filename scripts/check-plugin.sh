#!/bin/sh

set -eu

repo_root=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)

python3 - "$repo_root" <<'PY'
import json
import os
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
claude_manifest_path = root / "plugin/.claude-plugin/plugin.json"
codex_manifest_path = root / "plugin/.codex-plugin/plugin.json"
claude_marketplace_path = root / ".claude-plugin/marketplace.json"
codex_marketplace_path = root / ".agents/plugins/marketplace.json"
skill_path = root / "plugin/skills/weeek/SKILL.md"
reference_paths = [
    root / "plugin/skills/weeek/references/tasks.md",
    root / "plugin/skills/weeek/references/boards.md",
    root / "plugin/skills/weeek/references/projects.md",
    root / "plugin/skills/weeek/references/workspace.md",
]


def load_json(path):
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise SystemExit(f"check-plugin: invalid JSON in {path.relative_to(root)}: {error}")


claude_manifest = load_json(claude_manifest_path)
codex_manifest = load_json(codex_manifest_path)
claude_marketplace = load_json(claude_marketplace_path)
codex_marketplace = load_json(codex_marketplace_path)

for label, manifest in (
    ("Claude", claude_manifest),
    ("Codex", codex_manifest),
):
    for field in ("name", "version", "description", "skills"):
        if not isinstance(manifest.get(field), str) or not manifest[field].strip():
            raise SystemExit(f"check-plugin: {label} manifest is missing {field}")
    if manifest["skills"] != "./skills/":
        raise SystemExit(f"check-plugin: {label} manifest skills must be ./skills/")

if claude_manifest["name"] != "weeek" or codex_manifest["name"] != "weeek":
    raise SystemExit("check-plugin: plugin manifest names must be weeek")

version = claude_manifest["version"]
if codex_manifest["version"] != version:
    raise SystemExit("check-plugin: plugin manifest versions do not match")
if re.fullmatch(r"(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)", version) is None:
    raise SystemExit(f"check-plugin: plugin version is not semver: {version}")
expected_version = os.environ.get("WEEEK_PLUGIN_VERSION")
if expected_version and version != expected_version:
    raise SystemExit(
        f"check-plugin: plugin version {version} does not match release version {expected_version}"
    )

try:
    skill = skill_path.read_text(encoding="utf-8")
except OSError as error:
    raise SystemExit(f"check-plugin: cannot read SKILL.md: {error}")

frontmatter = re.match(r"\A---\n(.*?)\n---(?:\n|\Z)", skill, re.DOTALL)
if frontmatter is None:
    raise SystemExit("check-plugin: SKILL.md has no YAML frontmatter")
for field in ("name", "description"):
    if re.search(rf"(?m)^{field}:\s*\S.*$", frontmatter.group(1)) is None:
        raise SystemExit(f"check-plugin: SKILL.md frontmatter is missing {field}")
if re.search(rf"(?<![0-9.]){re.escape(version)}(?![0-9.])", skill) is None:
    raise SystemExit("check-plugin: plugin version is not mentioned in SKILL.md")

for path in reference_paths:
    if not path.is_file():
        raise SystemExit(f"check-plugin: missing reference {path.relative_to(root)}")
    if path.name not in skill:
        raise SystemExit(f"check-plugin: SKILL.md does not link {path.name}")

try:
    claude_plugin = claude_marketplace["plugins"][0]
except (KeyError, IndexError, TypeError):
    raise SystemExit("check-plugin: Claude marketplace plugin entry is incomplete")
if not isinstance(claude_marketplace.get("owner", {}).get("name"), str):
    raise SystemExit("check-plugin: Claude marketplace owner is missing")
if claude_plugin.get("name") != "weeek":
    raise SystemExit("check-plugin: Claude marketplace plugin name must be weeek")
if claude_plugin.get("source") != "./plugin":
    raise SystemExit("check-plugin: Claude marketplace source must be ./plugin")
if claude_plugin.get("version") != version:
    raise SystemExit("check-plugin: Claude marketplace version does not match manifests")

try:
    codex_plugin = codex_marketplace["plugins"][0]
    codex_source = codex_plugin["source"]
    codex_policy = codex_plugin["policy"]
except (KeyError, IndexError, TypeError):
    raise SystemExit("check-plugin: Codex marketplace plugin entry is incomplete")
if codex_source.get("path") != "./plugin":
    raise SystemExit("check-plugin: Codex marketplace source path must be ./plugin")
if codex_plugin.get("name") != "weeek" or codex_source.get("source") != "local":
    raise SystemExit("check-plugin: Codex marketplace plugin source is invalid")
if codex_policy.get("installation") not in {
    "NOT_AVAILABLE", "AVAILABLE", "INSTALLED_BY_DEFAULT"
}:
    raise SystemExit("check-plugin: invalid Codex installation policy")
if codex_policy.get("authentication") not in {"ON_INSTALL", "ON_USE"}:
    raise SystemExit("check-plugin: invalid Codex authentication policy")
if not isinstance(codex_plugin.get("category"), str) or not codex_plugin["category"]:
    raise SystemExit("check-plugin: Codex marketplace category is missing")

print(f"plugin metadata is valid (version {version})")
PY
