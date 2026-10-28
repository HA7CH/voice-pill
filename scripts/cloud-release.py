#!/usr/bin/env python3
"""Use Xcode's logged-in account for cloud Developer ID signing and notarization.

Requires cloud-managed Developer ID permission. No legacy keychain password,
private-key export, or app-specific password is read by this script.
"""
import argparse,datetime,pathlib,plistlib,shutil,subprocess
parser=argparse.ArgumentParser();parser.add_argument('--team-id',required=True);parser.add_argument('--export-notarized',action='store_true');args=parser.parse_args()
root=pathlib.Path(__file__).resolve().parents[1]
archive=root/'build/VoicePill.xcarchive'
output=root/'build/notarized-export'
def run(cmd):subprocess.run([str(x) for x in cmd],cwd=root,check=True)
if args.export_notarized:
 run(['xcodebuild','-exportNotarizedApp','-archivePath',archive,'-exportPath',output])
 app=output/'Voice Pill.app'
 run(['xcrun','stapler','validate',app]);run(['codesign','--verify','--deep','--strict',app]);run(['spctl','--assess','--type','execute','--verbose=4',app])
 # Only promote the successfully accepted app into the packaging input.
 shutil.rmtree(root/'build/Voice Pill.app')
 shutil.copytree(app,root/'build/Voice Pill.app')
 run(['bash','scripts/package.sh'])
else:
 source=root/'build/Voice Pill.app';app=archive/'Products/Applications/Voice Pill.app'
 if archive.exists():raise SystemExit('Archive exists. Use --export-notarized to resume, or preserve it before starting a new submission.')
 app.parent.mkdir(parents=True);shutil.copytree(source,app)
 main=app/'Contents/MacOS/VoicePill'
 for p in sorted(app.rglob('*')):
  if p.is_file() and not p.is_symlink() and p!=main and 'Mach-O' in subprocess.check_output(['file','-b',str(p)],text=True):run(['codesign','--force','--sign','-','--options','runtime',p])
 run(['codesign','--force','--sign','-','--options','runtime','--entitlements',root/'Resources/Release.entitlements',app])
 info=plistlib.loads((app/'Contents/Info.plist').read_bytes())
 (archive/'Info.plist').write_bytes(plistlib.dumps({'ArchiveVersion':2,'CreationDate':datetime.datetime.now(datetime.timezone.utc),'Name':'Voice Pill','SchemeName':'Voice Pill','ApplicationProperties':{'ApplicationPath':'Applications/Voice Pill.app','CFBundleIdentifier':info['CFBundleIdentifier'],'CFBundleShortVersionString':info['CFBundleShortVersionString'],'CFBundleVersion':info['CFBundleVersion'],'SigningIdentity':'-','Team':args.team_id,'Architectures':['arm64']}}))
 opts=root/'build/DeveloperIDUpload.plist';opts.write_bytes(plistlib.dumps({'method':'developer-id','teamID':args.team_id,'signingStyle':'automatic','destination':'upload','manageAppVersionAndBuildNumber':False}))
 run(['xcodebuild','-exportArchive','-archivePath',archive,'-exportPath',root/'build/notary-submission','-exportOptionsPlist',opts,'-allowProvisioningUpdates'])
 print('Submission uploaded. Run this script with --export-notarized after Apple accepts it.')
