import Foundation
import AVFoundation

// Microphone capture and its WAV outlive a failed network session.
final class LiveTranscriber {
    private let engine = AVAudioEngine()
    private let session = LiveTranscriptionSession()
    private var tapped = false
    private var recordingFile: AVAudioFile?

    func start(binary: String, provider: LiveProvider, recording: URL, punctuation: Bool,
               update: @escaping (String) -> Void,
               completion: @escaping (Result<String, Error>) -> Void) throws {
        let node = engine.inputNode
        let source = node.outputFormat(forBus: 0)
        guard source.sampleRate > 0,
              let format = AVAudioFormat(commonFormat: .pcmFormatInt16, sampleRate: provider.sampleRate, channels: 1, interleaved: true),
              let converter = AVAudioConverter(from: source, to: format) else {
            throw LiveStreamError("Microphone format unavailable")
        }
        recordingFile = try AVAudioFile(forWriting: recording, settings: format.settings, commonFormat: .pcmFormatInt16, interleaved: true)
        try session.start(binary: binary, provider: provider, log: recording.appendingPathExtension("log"), punctuation: punctuation, update: update, completion: completion)
        node.installTap(onBus: 0, bufferSize: 2048, format: source) { [self] buffer, _ in
            let capacity = AVAudioFrameCount(ceil(Double(buffer.frameLength) * provider.sampleRate / source.sampleRate) + 64)
            guard let pcm = AVAudioPCMBuffer(pcmFormat: format, frameCapacity: capacity) else { return }
            var supplied = false
            var conversionError: NSError?
            converter.convert(to: pcm, error: &conversionError) { _, status in
                if supplied { status.pointee = .noDataNow; return nil }
                supplied = true; status.pointee = .haveData; return buffer
            }
            guard conversionError == nil, pcm.frameLength > 0, let samples = pcm.int16ChannelData else { return }
            // Save every frame even after the stream fails, for full-file retry.
            try? recordingFile?.write(from: pcm)
            session.append(Data(bytes: samples[0], count: Int(pcm.frameLength) * 2))
        }
        tapped = true
        do { try engine.start() } catch { cancel(); throw error }
    }

    func finish() {
        engine.stop()
        if tapped { engine.inputNode.removeTap(onBus: 0); tapped = false }
        recordingFile = nil // Finalize the WAV before file-based retry reads it.
        session.finish()
    }
    func cancel() { finish(); session.cancel() }
}
