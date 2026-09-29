import Foundation

enum LiveProvider: String {
    case codex, doubao
    var sampleRate: Double { self == .codex ? 24000 : 16000 }
    var binaryName: String { self == .codex ? "codex-asr" : "freeasr" }
    var finishTimeout: Double { self == .codex ? 65 : 35 }
    var completionEvent: String { self == .codex ? "result" : "final" }
    func arguments(punctuation: Bool) -> [String] {
        if self == .codex { return ["stream", "--sample-rate", "24000"] }
        var args = ["live", "--quiet", "--credential-path", NSHomeDirectory() + "/Library/Application Support/VoicePill/doubao-credentials.json"]
        if !punctuation { args.append("--no-punctuation") }
        return args
    }
}

// Codex's `final` ends one utterance; only `result` completes the session.
// Events from either provider are full snapshots, not deltas to concatenate.
struct LiveEventDecoder {
    let provider: LiveProvider
    private var pending = Data()
    private(set) var finalText: String?
    mutating func append(_ data: Data) throws -> [String] {
        pending.append(data)
        guard pending.count <= 1_048_576 else { throw LiveStreamError("Oversized transcription event") }
        var updates: [String] = []
        while let end = pending.firstIndex(of: 10) {
            let line = Data(pending[..<end]); pending.removeSubrange(...end)
            if line.isEmpty { continue }
            guard let event = try JSONSerialization.jsonObject(with: line) as? [String: Any],
                  let type = event["type"] as? String else { throw LiveStreamError("Invalid transcription event") }
            if type == "error" { throw LiveStreamError("The transcription connection failed. Audio is saved for retry.") }
            guard ["partial", "final", "result"].contains(type), let text = event["text"] as? String else { continue }
            if type == provider.completionEvent { finalText = text }
            updates.append(text)
        }
        return updates
    }
    func completed(exitStatus: Int32) throws -> String {
        guard exitStatus == 0, pending.isEmpty, let text = finalText?.trimmingCharacters(in: .whitespacesAndNewlines), !text.isEmpty else {
            throw LiveStreamError("No complete transcript was received. Audio is saved for retry.")
        }
        return text
    }
}

struct LiveStreamError: LocalizedError {
    let errorDescription: String?
    init(_ message: String) { errorDescription = message }
}
