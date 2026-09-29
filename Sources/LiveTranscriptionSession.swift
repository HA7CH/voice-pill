import Foundation

// Child-process transport shared by microphone capture and synthetic fixtures.
// Network failure completes this session, not the recording that owns it.
final class LiveTranscriptionSession {
    private let job = Process()
    private let input = Pipe()
    private let output = Pipe()
    private let writes = DispatchQueue(label: "voicepill.live.audio")
    private let lock = NSLock()
    private var pendingBytes = 0
    private var accepting = true
    private var completed = false
    private var completion: ((Result<String, Error>) -> Void)?

    func start(binary: String, provider: LiveProvider, log: URL, punctuation: Bool,
               update: @escaping (String) -> Void,
               completion: @escaping (Result<String, Error>) -> Void) throws {
        signal(SIGPIPE, SIG_IGN)
        self.completion = completion
        job.executableURL = URL(fileURLWithPath: binary)
        job.arguments = provider.arguments(punctuation: punctuation)
        job.standardInput = input; job.standardOutput = output
        FileManager.default.createFile(atPath: log.path, contents: nil)
        let stderr = try FileHandle(forWritingTo: log)
        job.standardError = stderr
        do { try job.run() } catch { try? stderr.close(); throw error }
        try? input.fileHandleForReading.close(); try? output.fileHandleForWriting.close()
        DispatchQueue.global(qos: .userInitiated).async { [self] in
            defer {
                try? output.fileHandleForReading.close()
                try? stderr.close()
                try? FileManager.default.removeItem(at: log)
            }
            var decoder = LiveEventDecoder(provider: provider)
            do {
                while true {
                    let chunk = try StreamPipe.readChunk(from: output.fileHandleForReading)
                    if chunk.isEmpty { break }
                    for text in try decoder.append(chunk) {
                        DispatchQueue.main.async { update(text) }
                    }
                }
                job.waitUntilExit()
                try? stderr.close()
                do { complete(.success(try decoder.completed(exitStatus: job.terminationStatus))) }
                catch {
                    let detail = (try? String(contentsOf: log, encoding: .utf8))?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                    complete(.failure(detail.isEmpty ? error : LiveStreamError(String(detail.suffix(1600)))))
                }
            } catch {
                if job.isRunning { job.terminate() }
                complete(.failure(error))
            }
        }
    }

    func append(_ data: Data) {
        lock.lock()
        guard accepting else { lock.unlock(); return }
        // Bound queued audio if the network child stops reading.
        guard pendingBytes + data.count <= 262_144 else {
            accepting = false; lock.unlock()
            if job.isRunning { job.terminate() }
            complete(.failure(LiveStreamError("Live transcription stalled. Recording continues for retry.")))
            return
        }
        pendingBytes += data.count; lock.unlock()
        writes.async { [self] in
            defer { lock.lock(); pendingBytes -= data.count; lock.unlock() }
            do { try input.fileHandleForWriting.write(contentsOf: data) }
            catch { complete(.failure(error)) }
        }
    }

    func finish() {
        lock.lock(); accepting = false; lock.unlock()
        writes.async { [self] in try? input.fileHandleForWriting.close() }
    }

    func cancel() {
        lock.lock(); accepting = false; completed = true; completion = nil; lock.unlock()
        if job.isRunning { job.terminate() }
        finish()
    }

    private func complete(_ result: Result<String, Error>) {
        lock.lock()
        guard !completed else { lock.unlock(); return }
        completed = true; accepting = false
        let callback = completion; completion = nil; lock.unlock()
        finish()
        DispatchQueue.main.async { callback?(result) }
    }
}
