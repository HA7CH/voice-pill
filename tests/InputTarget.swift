import AppKit
final class Delegate: NSObject, NSApplicationDelegate {
 var window: NSWindow!
 func applicationDidFinishLaunching(_ n: Notification) {
  let menu = NSMenu()
  let editItem = NSMenuItem(title: "Edit", action: nil, keyEquivalent: "")
  let edit = NSMenu(title: "Edit")
  edit.addItem(withTitle: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v")
  edit.addItem(withTitle: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c")
  edit.addItem(withTitle: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a")
  editItem.submenu = edit; menu.addItem(editItem); NSApp.mainMenu = menu
  window = NSWindow(contentRect: NSRect(x: 200, y: 250, width: 460, height: 180), styleMask: [.titled,.closable], backing: .buffered, defer: false)
  window.title = "Voice Pill · 本地输入验收"
  let editor = NSTextView(frame: NSRect(x: 0,y: 0,width: 460,height: 180))
  editor.font = .systemFont(ofSize: 18); editor.isRichText = false
  window.contentView = editor; window.makeKeyAndOrderFront(nil); window.makeFirstResponder(editor)
  NSApp.setActivationPolicy(.regular); NSApp.activate(ignoringOtherApps: true)
 }
}
let app = NSApplication.shared
let delegate = Delegate(); app.delegate = delegate; app.run()
