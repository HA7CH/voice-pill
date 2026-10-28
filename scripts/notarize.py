#!/usr/bin/env python3
"""Submit an artifact using a preconfigured keychain profile; stop on rejection."""
import json,os,pathlib,subprocess,sys
artifact=pathlib.Path(sys.argv[1]).resolve()
profile=os.environ.get('VOICEPILL_NOTARY_PROFILE')
if not profile:raise SystemExit('Set VOICEPILL_NOTARY_PROFILE to an existing notarytool keychain profile.')
submit=artifact
if artifact.suffix=='.app':
 submit=artifact.parent/'VoicePill-notary-upload.zip'
 subprocess.run(['ditto','-c','-k','--sequesterRsrc','--keepParent',str(artifact),str(submit)],check=True)
result=subprocess.run(['xcrun','notarytool','submit',str(submit),'--keychain-profile',profile,'--wait','--output-format','json'],capture_output=True,text=True)
try:status=json.loads(result.stdout)
except json.JSONDecodeError:raise SystemExit(result.stderr.strip() or 'No notarization response')
print(json.dumps(status,indent=2))
if result.returncode or status.get('status')!='Accepted':
 raise SystemExit('Apple did not accept this artifact; do not distribute it as notarized.')
subprocess.run(['xcrun','stapler','staple',str(artifact)],check=True)
subprocess.run(['xcrun','stapler','validate',str(artifact)],check=True)
