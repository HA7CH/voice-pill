#!/usr/bin/env python3
"""Sign every Mach-O from the inside out; never fall back to ad-hoc signing."""
import os,pathlib,subprocess,sys
app=pathlib.Path(sys.argv[1]).resolve()
identity=os.environ.get('VOICEPILL_DEVELOPER_ID')
if not identity:raise SystemExit('VOICEPILL_DEVELOPER_ID must name a Developer ID Application identity.')
base=['codesign','--force','--sign',identity,'--options','runtime','--timestamp']
if os.environ.get('VOICEPILL_SIGNING_KEYCHAIN'):base+=['--keychain',os.environ['VOICEPILL_SIGNING_KEYCHAIN']]
main=app/'Contents/MacOS/VoicePill'
for p in sorted(app.rglob('*')):
 if p.is_file() and not p.is_symlink() and p!=main and 'Mach-O' in subprocess.check_output(['file','-b',str(p)],text=True):
  subprocess.run(base+[str(p)],check=True)
entitlements=pathlib.Path(__file__).resolve().parents[1]/'Resources/Release.entitlements'
subprocess.run(base+['--entitlements',str(entitlements),str(app)],check=True)
subprocess.run(['codesign','--verify','--deep','--strict',str(app)],check=True)
meta=subprocess.run(['codesign','-dv','--verbose=4',str(app)],capture_output=True,text=True,check=True).stderr
if 'Authority=Developer ID Application:' not in meta or 'runtime' not in meta or 'Timestamp=' not in meta:
 raise SystemExit('Release requires Developer ID, hardened runtime, and secure timestamp.')
print('PASS: Developer ID signature, hardened runtime, secure timestamp')
