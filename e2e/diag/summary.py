"""Markdown of the JSON lines of term.vim, for $GITHUB_STEP_SUMMARY.

    python summary.py results.jsonl
"""
import json
import sys

print("| nvim | mode | uis | cmd | id | exited (code, ms) | jobwait 1.5s / end | lines 1.5s | lines end |")
print("|---|---|---|---|---|---|---|---|---|")
messages = []
for line in open(sys.argv[1], encoding="utf-8"):
    r = json.loads(line)
    if "messages" in r:
        messages.append((r.get("nvim", ""), r["mode"], r["messages"].strip()))
        continue
    ex = r.get("exited")
    exit_s = f"{ex['code']}, {ex['ms']}" if ex else "alive"
    lines = lambda k: "<br>".join(x.replace("|", "\\|") for x in r.get(k, [])[:5]) or "(empty)"
    print(f"| {r['nvim']} | {r['mode']} | {r['uis']} | `{' '.join(r['cmd'])}` | {r['id']} "
          f"| {exit_s} | {r.get('jobwait_1500ms')} / {r.get('jobwait_end')} "
          f"| {lines('lines_1500ms')} | {lines('lines_end')} |")
for ver, mode, m in messages:
    print(f"\n**:messages ({ver} {mode})**\n\n```\n{m or '(none)'}\n```")
