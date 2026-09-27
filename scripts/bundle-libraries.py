#!/usr/bin/env python3
"""Copy non-system dylibs into the app and replace absolute load paths."""
import pathlib, shutil, subprocess, sys
app = pathlib.Path(sys.argv[1]).resolve()
frameworks = app / 'Contents/Frameworks'
frameworks.mkdir(exist_ok=True)
seen = set()
def run(*args): return subprocess.check_output(args, text=True)
def bundle(binary):
    if binary in seen: return
    seen.add(binary)
    for line in run('otool', '-L', str(binary)).splitlines()[1:]:
        dependency = line.strip().split(' (')[0]
        if not dependency.startswith('/') or dependency.startswith(('/usr/lib/', '/System/Library/')): continue
        original = pathlib.Path(dependency)
        target = frameworks / original.name
        if target == binary: continue
        if not target.exists():
            shutil.copy2(original, target)
            target.chmod(0o755)
            subprocess.run(['install_name_tool', '-id', '@rpath/' + target.name, str(target)], check=True)
            bundle(target)
        relative = '@loader_path/' + ('' if binary.parent == frameworks else '../Frameworks/') + target.name
        subprocess.run(['install_name_tool', '-change', dependency, relative, str(binary)], check=True)
bundle(app / 'Contents/Resources/freeasr')
