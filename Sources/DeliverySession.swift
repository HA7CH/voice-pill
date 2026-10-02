import Foundation

// An abandoned paste must never leave the app busy and block the next Fn press.
@MainActor
enum DeliverySession {
    enum Interruption: Equatable { case abandoned, timedOut }

    static func run(timeout: Duration = .seconds(8),
                    isCurrent: @escaping () -> Bool,
                    interrupted: @escaping (Interruption) -> Void,
                    operation: () async -> Void) async {
        let watchdog = Task { @MainActor in
            do { try await Task.sleep(for: timeout) } catch { return }
            if isCurrent() { interrupted(.timedOut) }
        }
        defer {
            watchdog.cancel()
            if isCurrent() { interrupted(.abandoned) }
        }
        await operation()
    }
}
