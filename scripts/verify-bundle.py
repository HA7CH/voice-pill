#!/usr/bin/env python3
"""Reject developer paths, nonportable libraries and private artifacts."""
import pathlib, subprocess, sys
app = pathlib.Path(sys.argv[1]).resolve()
for file in app.rglob('*'):
    if not file.is_file(): continue
    if file.suffix in {'.pem', '.key', '.p12', '.wav', '.m4a', '.log', '.keychain-db'} or 'credentials' in file.name or file.name == 'wetype-asr':
        raise SystemExit('Unexpected private/experimental artifact: ' + str(file))
    kind = subprocess.check_output(['file', '-b', str(file)], text=True)
    if 'Mach-O' not in kind: continue
    for line in subprocess.check_output(['otool', '-L', str(file)], text=True).splitlines()[1:]:
        dep = line.strip().split(' (')[0]
        if dep.startswith('/') and not dep.startswith(('/usr/lib/', '/System/Library/')):
            raise SystemExit('Nonportable dependency: ' + dep)
        if dep.startswith('@loader_path/') and not (file.parent/dep.removeprefix('@loader_path/')).resolve().exists():
            raise SystemExit('Missing bundled dependency: ' + dep)
subprocess.run(['codesign','--verify','--deep','--strict', str(app)], check=True)
subprocess.run([str(app/'Contents/Resources/freeasr'), 'version'], env={'PATH':'/usr/bin:/bin'}, check=True)
print('PASS: portable libraries, helper launch, signature and no private artifacts')
