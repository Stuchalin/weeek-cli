# weeek-cli

Work in progress: a zero-dependency Go CLI for the public Weeek API, designed
for predictable use by AI agents.

## Release process

Before tagging a release, update the version in both plugin manifests and the
CLI compatibility version in `plugin/skills/weeek/SKILL.md`, then run
`make check-plugin`. Keep these three values identical.
