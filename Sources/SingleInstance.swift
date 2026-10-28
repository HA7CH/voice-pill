import AppKit
import Darwin

// Both the original local ID and the distributed ID share one shortcut owner.
@MainActor enum SingleInstance {
    private static var lockDescriptor: Int32 = -1
    static func acquire() -> Bool {
        let identifiers = ["com.lawted.voicepill", "com.ha7ch.voicepill"]
        let currentPID = ProcessInfo.processInfo.processIdentifier
        if let existing = identifiers.flatMap({ NSRunningApplication.runningApplications(withBundleIdentifier: $0) })
            .first(where: { $0.processIdentifier != currentPID && !$0.isTerminated }) {
            existing.activate()
            return false
        }
        let folder = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("VoicePill", isDirectory: true)
        do { try FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true) }
        catch { return false }
        let descriptor = open(folder.appendingPathComponent("instance.lock").path, O_CREAT | O_RDWR, S_IRUSR | S_IWUSR)
        guard descriptor >= 0 else { return false }
        guard flock(descriptor, LOCK_EX | LOCK_NB) == 0 else { close(descriptor); return false }
        lockDescriptor = descriptor // The OS releases this lock when the process exits.
        return true
    }
}
