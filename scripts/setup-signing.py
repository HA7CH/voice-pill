#!/usr/bin/env python3
"""Create an isolated local code-signing identity; never alter system trust."""
import pathlib, subprocess, secrets, os
root = pathlib.Path(__file__).resolve().parents[1]
s = root / '.signing'
s.mkdir(mode=0o700, exist_ok=True)
def run(*args): subprocess.run(args, check=True, stdout=subprocess.DEVNULL)
if (s / 'ready').exists(): raise SystemExit('Signing identity already exists.')
os.umask(0o077)
password = secrets.token_hex(24)
(s / 'password').write_text(password)
(s / 'cert.conf').write_text('''[req]
distinguished_name=dn
x509_extensions=ext
prompt=no
[dn]
CN=Voice Pill Local Development
[ext]
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=critical,codeSigning
subjectKeyIdentifier=hash
''')
run('/usr/bin/openssl','req','-new','-newkey','rsa:2048','-nodes','-x509','-days','3650','-config',str(s/'cert.conf'),'-keyout',str(s/'key.pem'),'-out',str(s/'cert.pem'))
run('/usr/bin/openssl','pkcs12','-export','-inkey',str(s/'key.pem'),'-in',str(s/'cert.pem'),'-out',str(s/'identity.p12'),'-passout','file:'+str(s/'password'))
k=str(s/'voice-pill.keychain-db')
run('/usr/bin/security','create-keychain','-p',password,k)
run('/usr/bin/security','unlock-keychain','-p',password,k)
run('/usr/bin/security','import',str(s/'identity.p12'),'-k',k,'-P',password,'-T','/usr/bin/codesign')
run('/usr/bin/security','set-key-partition-list','-S','apple-tool:,apple:','-s','-k',password,k)
(s/'ready').write_text('Local identity created; not added to system trust.\n')
print('Created isolated local signing identity.')
