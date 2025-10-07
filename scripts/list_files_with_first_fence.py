#!/usr/bin/env python3
from pathlib import Path
root = Path('/workspaces/srat')
search_dir = root / 'docs'

matches = []
for p in sorted(search_dir.rglob('*.md')):
    try:
        s = p.read_text(encoding='utf-8')
    except Exception:
        continue
    lines = s.splitlines()
    first_idx=None
    for i,l in enumerate(lines):
        if l.strip()!='':
            first_idx=i; break
    if first_idx is None: continue
    last_idx=None
    for i in range(len(lines)-1,-1,-1):
        if lines[i].strip()!='': last_idx=i; break
    first_line=lines[first_idx]
    last_line=lines[last_idx]
    if first_line.lstrip().startswith('```'):
        matches.append({'path':str(p.relative_to(root)),'first_idx':first_idx+1,'first_line':first_line,'last_idx':last_idx+1,'last_line':last_line})

import json
print(json.dumps(matches,indent=2))
