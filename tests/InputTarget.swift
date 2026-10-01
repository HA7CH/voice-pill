import AppKit

final class InputTargetDelegate: NSObject, NSApplicationDelegate, NSTextViewDelegate {
    var window: NSWindow!
    var editor: NSTextView!
    func applicationDidFinishLaunching(_ notification: Notification) {
        let menu = NSMenu(), item = NSMenuItem(title: "Edit", action: nil, keyEquivalent: "")
        let edit = NSMenu(title: "Edit")
        edit.addItem(withTitle: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v")
        edit.addItem(withTitle: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c")
        edit.addItem(withTitle: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a")
        item.submenu = edit; menu.addItem(item); NSApp.mainMenu = menu
        window = NSWindow(contentRect: NSRect(x: 200, y: 250, width: 460, height: 180), styleMask: [.titled, .closable, .resizable], backing: .buffered, defer: false)
        window.title = "Voice Pill · Local input test"
        editor = NSTextView(frame: NSRect(x: 0, y: 0, width: 460, height: 180))
        editor.font = .systemFont(ofSize: 18); editor.isRichText = false; editor.delegate = self
        window.contentView = editor
        window.makeKeyAndOrderFront(nil); window.makeFirstResponder(editor)
        NSApp.activate(ignoringOtherApps: true)
        try? "Ready\n".write(toFile: "/tmp/voice-pill-fixture-ready.txt", atomically: true, encoding: .utf8)
    }
    func textDidChange(_ notification: Notification) {
        try? editor.string.write(toFile: "/tmp/voice-pill-fixture-input.txt", atomically: true, encoding: .utf8)
    }
}
@main struct InputTargetApp {
    static func main() {
        let app = NSApplication.shared, delegate = InputTargetDelegate()
        app.setActivationPolicy(.regular); app.delegate = delegate
        withExtendedLifetime(delegate) { app.run() }
    }
}
