import Foundation

@main struct LiveSessionTests {
    static func main() throws {
        if CommandLine.arguments.count == 5 && CommandLine.arguments[1] == "--live" {
            let provider = LiveProvider(rawValue: CommandLine.arguments[4])!
            try run(binary: CommandLine.arguments[2], audio: Data(contentsOf: URL(fileURLWithPath: CommandLine.arguments[3])), provider: provider, expectSuccess: true)
            return
        }
        let folder = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: folder) }
        for provider in [LiveProvider.codex, .doubao] {
            let helper = folder.appendingPathComponent(provider.rawValue)
            let script = "#!/bin/bash\nprintf '%s\\n' '{\"type\":\"partial\",\"text\":\"live text\",\"elapsed_ms\":1}'\ncat >/dev/null\nprintf '%s\\n' '{\"type\":\"\(provider.completionEvent)\",\"text\":\"complete text\"}'\n"
            try script.write(to: helper, atomically: true, encoding: .utf8)
            try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: helper.path)
            try run(binary: helper.path, audio: Data(repeating: 0, count: Int(provider.sampleRate) * 2), provider: provider, expectSuccess: true)
        }
        let broken = folder.appendingPathComponent("broken")
        try "#!/bin/bash\nprintf '%s\\n' '{\"type\":\"final\",\"text\":\"partial utterance\"}'\nexit 1\n".write(to: broken, atomically: true, encoding: .utf8)
        try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: broken.path)
        try run(binary: broken.path, audio: Data(repeating: 0, count: 48000), provider: .codex, expectSuccess: false)
    }
    static func run(binary: String, audio: Data, provider: LiveProvider, expectSuccess: Bool) throws {
        let stream = LiveTranscriptionSession()
        let log = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString + ".log")
        var done = false; var released = false; var partialBeforeRelease = false; var updates = 0
        let start = Date()
        try stream.start(binary: binary, provider: provider, log: log, punctuation: false, update: { text in
            updates += 1
            if !released { partialBeforeRelease = true }
            if updates == 1 { print(String(format: "First caption %.2f s: %@", Date().timeIntervalSince(start), text)) }
        }, completion: { result in
            switch result {
            case .success(let text):
                precondition(expectSuccess && released && partialBeforeRelease && !text.isEmpty)
                print("PASS \(provider.rawValue): \(updates) updates, complete result after release: \(text)")
            case .failure(let error):
                if expectSuccess { fatalError(error.localizedDescription) }
                print("PASS: disconnected session reported failure without waiting for input EOF")
            }
            done = true
        })
        DispatchQueue.global().async {
            let chunk = Int(provider.sampleRate / 10) * 2
            for offset in stride(from: 0, to: audio.count, by: chunk) {
                stream.append(audio.subdata(in: offset..<min(offset + chunk, audio.count)))
                Thread.sleep(forTimeInterval: 0.1)
            }
            DispatchQueue.main.async { released = true; stream.finish() }
        }
        while !done && Date().timeIntervalSince(start) < 85 { RunLoop.current.run(until: Date().addingTimeInterval(0.02)) }
        stream.cancel()
        precondition(done, "Transcription timed out")
    }
}
