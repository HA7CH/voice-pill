import AppKit
import Foundation

let output = URL(fileURLWithPath: CommandLine.arguments[1])
try FileManager.default.createDirectory(at: output, withIntermediateDirectories: true)
func color(_ r: CGFloat, _ g: CGFloat, _ b: CGFloat, _ a: CGFloat = 1) -> NSColor { NSColor(srgbRed:r/255,green:g/255,blue:b/255,alpha:a) }
func drawIcon(_ pixels: Int) throws {
    let rep = NSBitmapImageRep(bitmapDataPlanes:nil,pixelsWide:pixels,pixelsHigh:pixels,bitsPerSample:8,samplesPerPixel:4,hasAlpha:true,isPlanar:false,colorSpaceName:.deviceRGB,bytesPerRow:0,bitsPerPixel:0)!
    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep:rep)
    let c = NSGraphicsContext.current!.cgContext
    c.scaleBy(x:CGFloat(pixels)/1024,y:CGFloat(pixels)/1024)
    let tile = NSBezierPath(roundedRect:NSRect(x:96,y:96,width:832,height:832),xRadius:190,yRadius:190)
    NSGraphicsContext.saveGraphicsState()
    let shadow = NSShadow();shadow.shadowColor = NSColor.black.withAlphaComponent(0.18);shadow.shadowBlurRadius=24;shadow.shadowOffset=NSSize(width:0,height:-12);shadow.set()
    color(26,26,28).setFill();tile.fill()
    NSGraphicsContext.restoreGraphicsState()
    NSGradient(starting:color(13,13,15),ending:color(57,57,61))!.draw(in:tile,angle:90)
    NSGraphicsContext.saveGraphicsState();tile.addClip()
    let sheen = NSBezierPath(ovalIn:NSRect(x:-80,y:620,width:1200,height:600))
    NSColor.white.withAlphaComponent(0.035).setFill();sheen.fill()
    NSGraphicsContext.restoreGraphicsState()
    let pill = NSBezierPath(roundedRect:NSRect(x:210,y:334,width:604,height:356),xRadius:178,yRadius:178)
    NSGraphicsContext.saveGraphicsState()
    let ps=NSShadow();ps.shadowColor=color(0,0,0,0.36);ps.shadowBlurRadius=25;ps.shadowOffset=NSSize(width:0,height:-16);ps.set()
    color(255,255,255,0.16).setFill();pill.fill();NSGraphicsContext.restoreGraphicsState()
    NSGradient(starting:color(255,255,255,0.07),ending:color(255,255,255,0.27))!.draw(in:pill,angle:90)
    NSColor.white.withAlphaComponent(0.42).setStroke();pill.lineWidth=3;pill.stroke()
    let heights:[CGFloat] = [74,148,218,148,74]
    for (i,h) in heights.enumerated() {
        let bar = NSBezierPath(roundedRect:NSRect(x:362+CGFloat(i)*66,y:512-h/2,width:36,height:h),xRadius:18,yRadius:18)
        NSColor.white.setFill();bar.fill()
    }
    NSGraphicsContext.restoreGraphicsState()
    let filename = pixels == 1024 ? "AppIcon.png" : "preview-\(pixels).png"
    try rep.representation(using:.png,properties:[:])!.write(to:output.appendingPathComponent(filename))
}
try drawIcon(1024)
try drawIcon(64)
