#!/usr/bin/env python3
"""Ship license notices for the Go modules and bundled Opus library."""
import json, pathlib, shutil, subprocess, sys
root = pathlib.Path(__file__).resolve().parents[1]
out = pathlib.Path(sys.argv[1]) / 'Contents/Resources/ThirdPartyLicenses'
out.mkdir(parents=True, exist_ok=True)
raw = subprocess.check_output(['go', 'list', '-m', '-json', 'all'], cwd=root/'vendor/FreeASR', text=True)
decoder = json.JSONDecoder()
while raw.strip():
    module, end = decoder.raw_decode(raw.lstrip()); raw = raw.lstrip()[end:]
    if module.get('Main') or not module.get('Dir'): continue
    directory = pathlib.Path(module['Dir'])
    for name in ('LICENSE', 'LICENSE.txt', 'LICENSE.md', 'COPYING'):
        source = directory/name
        if source.is_file():
            destination = out/(module['Path'].replace('/', '_') + '-' + name)
            if destination.exists(): destination.chmod(0o644)
            shutil.copyfile(source, destination)
            break
    else: raise SystemExit('No license found for ' + module['Path'])
opus = pathlib.Path(subprocess.check_output(['brew', '--prefix', 'opus'], text=True).strip())
shutil.copy2(opus/'COPYING', out/'libopus-COPYING')
shutil.copy2(root/'THIRD_PARTY.md', out/'THIRD_PARTY.md')
metadata = json.loads(subprocess.check_output([
    'cargo', 'metadata', '--locked', '--format-version', '1', '--no-default-features',
    '--features', 'streaming', '--manifest-path', str(root/'vendor/codex-asr/Cargo.toml')
], text=True))
for package in metadata['packages']:
    directory = pathlib.Path(package['manifest_path']).parent
    for source in directory.iterdir():
        if source.is_file() and source.name.upper().startswith(('LICENSE', 'COPYING', 'NOTICE')):
            destination = out/('rust-' + package['name'] + '-' + package['version'] + '-' + source.name)
            if destination.exists(): destination.chmod(0o644)
            shutil.copyfile(source, destination)
(out/'rust-packages.json').write_text(json.dumps([
    {k: p.get(k) for k in ('name', 'version', 'license', 'repository')} for p in metadata['packages']
], indent=2) + '\n')
