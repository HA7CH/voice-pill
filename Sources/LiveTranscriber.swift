import Foundation
import AVFoundation

// Audio conversion stays on the audio callback; pipe I/O runs off that callback.
final class LiveTranscriber {
    private let engine = AVAudioEngine()
    private let job = Process()
    private let input = Pipe()
    private let output = Pipe()
    private let writes = DispatchQueue(label: "voicepill.live.audio")
    private var tapped = false
    private var errorURL: URL?
    private var errorHandle: FileHandle?

    func start(binary: String, recording: URL, punctuation: Bool,
               update: @escaping (String) -> Void,
               completion: @escaping (Result<String, Error>) -> Void) throws {
        signal(SIGPIPE, SIG_IGN)
        let node = engine.inputNode
        let source = node.outputFormat(forBus: 0)
        guard source.sampleRate > 0,
              let format = AVAudioFormat(commonFormat: .pcmFormatInt16, sampleRate: 16000, channels: 1, interleaved: true),
              let converter = AVAudioConverter(from: source, to: format) else {
            throw NSError(domain: "VoicePill", code: 1, userInfo: [NSLocalizedDescriptionKey: "Microphone format unavailable"])
        }
        let file = try AVAudioFile(forWriting: recording, settings: format.settings, commonFormat: .pcmFormatInt16, interleaved: true)
        job.executableURL = URL(fileURLWithPath: binary)
        job.arguments = ["live", "--quiet", "--credential-path", NSHomeDirectory()+"/Library/Application Support/VoicePill/doubao-credentials.json"]
        if !punctuation { job.arguments!.append("--no-punctuation") }
        job.standardInput = input; job.standardOutput = output
        let log = recording.appendingPathExtension("log")
        errorURL = log
        FileManager.default.createFile(atPath: log.path, contents: nil)
        let stderr = try FileHandle(forWritingTo: log); errorHandle = stderr
        job.standardError = stderr
        try job.run()
        // Close parent copies, so EOF reliably follows child exit / mic stop.
        try? input.fileHandleForReading.close(); try? output.fileHandleForWriting.close()
        DispatchQueue.global(qos: .userInitiated).async { [self] in
            var pending = Data(); var final: String?
            do {
                while true {
                    let chunk = try StreamPipe.readChunk(from: output.fileHandleForReading)
                    if chunk.isEmpty { break }
                    pending.append(chunk)
                    while let newline = pending.firstIndex(of: 10) {
                        let line = pending[..<newline]; pending.removeSubrange(...newline)
                        if let event = try JSONSerialization.jsonObject(with: line) as? [String: String], let text = event["text"] {
                            if event["type"] == "final" { final = text }
                            DispatchQueue.main.async { update(text) }
                        }
                    }
                }
                job.waitUntilExit()
                try? stderr.close()
                let detail = (try? String(contentsOf: log, encoding: .utf8)) ?? ""
                try? FileManager.default.removeItem(at: log)
                guard job.terminationStatus == 0, let text = final?.trimmingCharacters(in: .whitespacesAndNewlines), !text.isEmpty else {
                    throw NSError(domain: "VoicePill", code: 2, userInfo: [NSLocalizedDescriptionKey: detail.isEmpty ? "No speech recognized" : String(detail.suffix(1600))])
                }
                DispatchQueue.main.async { completion(.success(text)) }
            } catch {
                DispatchQueue.main.async { completion(.failure(error)) }
            }
            try? output.fileHandleForReading.close()
        }
        node.installTap(onBus: 0, bufferSize: 2048, format: source) { [self] buffer, _ in
            let capacity = AVAudioFrameCount(ceil(Double(buffer.frameLength)*16000/source.sampleRate)+64)
            guard let pcm = AVAudioPCMBuffer(pcmFormat: format, frameCapacity: capacity) else { return }
            var supplied = false
            var conversionError: NSError?
            converter.convert(to: pcm, error: &conversionError) { _, status in
                if supplied { status.pointee = .noDataNow; return nil }
                supplied = true; status.pointee = .haveData; return buffer
            }
            guard conversionError == nil, pcm.frameLength > 0, let samples = pcm.int16ChannelData else { return }
            let data = Data(bytes: samples[0], count: Int(pcm.frameLength)*2)
            // Preserve a WAV for the existing explicit retry action.
            try? file.write(from: pcm)
            writes.async { [self] in try? input.fileHandleForWriting.write(contentsOf: data) }
        }
        tapped = true
        do { try engine.start() } catch { cancel(); throw error }
    }

    func finish() {
        engine.stop()
        if tapped { engine.inputNode.removeTap(onBus: 0); tapped = false }
        writes.async { [self] in try? input.fileHandleForWriting.close() }
    }
    func cancel() {
        if job.isRunning { job.terminate() }
        finish()
    }
}
