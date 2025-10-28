#!/usr/bin/env python3
"""
Scan docs/ for markdown files whose first non-empty line is a code fence
and whose last non-empty line is a closing fence (only backticks).
Print a JSON list of objects: {"path":..., "first_line":..., "last_line":..., "first_idx":..., "last_idx":...}
"""
import os, json
from pathlib import Path

root = Path(__file__).parent.parent
search_dir = root / 'docs'

candidates = []
for p in sorted(search_dir.rglob('*.md')):
    try:
        lines = p.read_text(encoding='utf-8').splitlines()
    except Exception as e:
        # skip binary or unreadable files
        continue
    # find first non-empty
    first_idx = None
    for i,l in enumerate(lines):
        if l.strip() != '':
            first_idx = i
            break
    if first_idx is None:
        continue
    last_idx = None
    for i in range(len(lines)-1, -1, -1):
        if lines[i].strip() != '':
            last_idx = i
            break
    if last_idx is None:
        continue
    first_line = lines[first_idx].rstrip('\n')
    last_line = lines[last_idx].rstrip('\n')
    # check if first_line starts with backticks and optional language
    if first_line.lstrip().startswith('```') and last_line.strip().startswith('```'):
        # ensure last line is only backticks and optional spaces (no trailing text)
        if last_line.strip().count('`') >= 3 and set(last_line.strip()) <= set('` '):
            candidates.append({
                'path': str(p.relative_to(root)),
                'first_line': first_line,
                'first_idx': first_idx+1,
                'last_line': last_line,
                'last_idx': last_idx+1,
            })

print(json.dumps(candidates, indent=2))
