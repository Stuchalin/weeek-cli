#!/usr/bin/env bash
# claude-review.sh - ralphex external review via claude on glm-5.3 (1M context).
#
# interface (see ralphex docs "Custom external review"): receives the path to a
# prompt file as $1, writes findings to stdout ("file:line - description", or
# exactly "NO ISSUES FOUND" when clean - the prompt file carries the full
# instructions, claude just follows them).
#
# model is pinned here, independent of global configs; [1m] selects the 1M
# context window. stderr (incl. claude's unrecognized-model notice) is dropped
# by ralphex's custom executor - findings must go to stdout.
exec claude -p --model "glm-5.3[1m]" < "$1"
