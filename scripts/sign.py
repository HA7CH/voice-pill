#!/usr/bin/env python3
import pathlib, subprocess, sys
root=pathlib.Path(__file__).resolve().parents[1]
s=root/'.signing'
app=pathlib.Path(sys.argv[1])
if not (s/'ready').exists():
    for helper in list((app/'Contents/Frameworks').glob('*.dylib')) + [app/'Contents/Resources/freeasr', app/'Contents/Resources/codex-asr', app]:
        subprocess.run(['/usr/bin/codesign','--force','--sign','-','--timestamp=none',str(helper)],check=True)
    raise SystemExit(0)
k=str(s/'voice-pill.keychain-db')
subprocess.run(['/usr/bin/security','unlock-keychain','-p',(s/'password').read_text(),k],check=True)
fingerprint = subprocess.check_output(['/usr/bin/openssl', 'x509', '-in', str(s/'cert.pem'), '-noout', '-fingerprint', '-sha1'], text=True).strip().split('=')[-1].replace(':','')
req='=designated => identifier "com.ha7ch.voicepill" and anchor H"'+fingerprint+'"' 
for helper in list((app/'Contents/Frameworks').glob('*.dylib')) + [app/'Contents/Resources/freeasr', app/'Contents/Resources/codex-asr']:
    if helper.exists():
        subprocess.run(['/usr/bin/codesign','--force','--sign','Voice Pill Local Development','--keychain',k,'--timestamp=none',str(helper)],check=True)

subprocess.run(['/usr/bin/codesign','--force','--sign','Voice Pill Local Development','--keychain',k,'--timestamp=none','--requirements',req,sys.argv[1]],check=True)
