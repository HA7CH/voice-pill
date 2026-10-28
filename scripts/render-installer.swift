import AppKit
let width=760,height=460
let rep=NSBitmapImageRep(bitmapDataPlanes:nil,pixelsWide:width,pixelsHigh:height,bitsPerSample:8,samplesPerPixel:4,hasAlpha:true,isPlanar:false,colorSpaceName:.deviceRGB,bytesPerRow:0,bitsPerPixel:0)!
rep.size=NSSize(width:width,height:height)
NSGraphicsContext.saveGraphicsState();NSGraphicsContext.current=NSGraphicsContext(bitmapImageRep:rep)
NSGraphicsContext.current!.cgContext.scaleBy(x:1,y:1)
NSColor(srgbRed:0.975,green:0.975,blue:0.98,alpha:1).setFill();NSRect(x:0,y:0,width:width,height:height).fill()
func text(_ s:String,y:CGFloat,size:CGFloat,weight:NSFont.Weight,color:NSColor){
 let attrs:[NSAttributedString.Key:Any]=[.font:NSFont.systemFont(ofSize:size,weight:weight),.foregroundColor:color]
 let rect=(s as NSString).size(withAttributes:attrs)
 (s as NSString).draw(at:NSPoint(x:(CGFloat(width)-rect.width)/2,y:y),withAttributes:attrs)
}
text("Voice Pill",y:378,size:26,weight:.semibold,color:.black)
text("Drag Voice Pill into Applications",y:60,size:15,weight:.regular,color:NSColor(white:0.35,alpha:1))
let arrow=NSBezierPath();arrow.move(to:NSPoint(x:342,y:255));arrow.line(to:NSPoint(x:418,y:255));arrow.move(to:NSPoint(x:403,y:270));arrow.line(to:NSPoint(x:418,y:255));arrow.line(to:NSPoint(x:403,y:240));arrow.lineWidth=3;arrow.lineCapStyle = .round;arrow.lineJoinStyle = .round
NSColor(white:0.5,alpha:1).setStroke();arrow.stroke()
NSGraphicsContext.restoreGraphicsState()
try rep.representation(using:.png,properties:[:])!.write(to:URL(fileURLWithPath:CommandLine.arguments[1]))
