#!/usr/bin/env python3
"""Write Finder presentation metadata without automating Finder preferences."""
import pathlib,sys
from ds_store import DSStore
from mac_alias import Alias
mount=pathlib.Path(sys.argv[1])
alias=Alias.for_file(str(mount/'.background/background.png')).to_bytes()
with DSStore.open(str(mount/'.DS_Store'),'w+') as store:
    store['.']['bwsp']={'ShowStatusBar':False,'ShowToolbar':False,'ShowPathbar':False,'ShowSidebar':False,'ShowTabView':False,'ContainerShowSidebar':False,'SidebarWidth':0,'WindowBounds':'{{320, 180}, {760, 460}}'}
    store['.']['icvp']={'viewOptionsVersion':1,'backgroundType':2,'backgroundColorRed':1.0,'backgroundColorGreen':1.0,'backgroundColorBlue':1.0,'scrollPositionX':0.0,'scrollPositionY':0.0,'backgroundImageAlias':alias,'iconSize':160.0,'textSize':14.0,'gridSpacing':100.0,'gridOffsetX':0.0,'gridOffsetY':0.0,'arrangeBy':'none','showIconPreview':False,'showItemInfo':False,'labelOnBottom':True}
    store['.']['vSrn']=('long',1)
    store['.']['icvl']=('type',b'icnv')
    store['Voice Pill.app']['Iloc']=(190,205)
    store['Applications']['Iloc']=(570,205)
