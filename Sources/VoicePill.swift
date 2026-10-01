import SwiftUI
import AppKit
import AVFoundation
import Carbon
import Combine

final class VoiceApplication: NSApplication {
    override func reportException(_ exception: NSException) {
        let detail = "\(exception.name.rawValue): \(exception.reason ?? "")\n\(exception.callStackSymbols.joined(separator: "\n"))"
        try? detail.write(toFile: "/tmp/voice-pill-exception.txt", atomically: true, encoding: .utf8)
        super.reportException(exception)
    }
}

@main
struct VoicePillApp {
    @MainActor static func main() {
        let app = VoiceApplication.shared
        let delegate = AppDelegate()
        app.delegate = delegate
        withExtendedLifetime(delegate) { app.run() }
    }
}

enum Phase: Equatable { case idle, requesting, recording, transcribing, inserting, done, failed }

@MainActor
final class VoiceModel: NSObject, ObservableObject, AVAudioRecorderDelegate {
    @Published var phase: Phase = .idle
    @Published var seconds = 0
    @Published var levels: [CGFloat] = Array(repeating: 0, count: 19)
    private var smoothedLevel: CGFloat = 0
    @Published var transcript = ""
    @Published var message = ""
    @Published var destination = ""
    @Published var timingSummary = ""
    var transcriptionStarted = Date()
    var transcriptionElapsed: Double = 0
    @Published var targetIcon: NSImage?
    var target: NSRunningApplication?
    var targetElement: AXUIElement?
    var lastExternalApp: NSRunningApplication?
    func captureTarget() {
        let front = NSWorkspace.shared.frontmostApplication
        target = front?.processIdentifier == ProcessInfo.processInfo.processIdentifier ? lastExternalApp : front
        destination = target?.localizedName ?? "text field"
        targetIcon = target?.icon
        targetElement = nil
        if let target, AXIsProcessTrusted() {
            let app = AXUIElementCreateApplication(target.processIdentifier)
            var element: CFTypeRef?
            if AXUIElementCopyAttributeValue(app, kAXFocusedUIElementAttribute as CFString, &element) == .success,
               let element, CFGetTypeID(element) == AXUIElementGetTypeID() {
                targetElement = (element as! AXUIElement)
            }
        }
    }
    var live: LiveTranscriber?
    var liveNeedsRetry = false
    @Published var liveText = ""
    var recordingBackend: String?
    var recorder: AVAudioRecorder?
    var timer: Timer?
    var audioURL: URL?
    @Published var savedRecordings: [URL] = []
    var recordingFolder: URL {
        FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("VoicePill/Recordings", isDirectory: true)
    }
    override init() {
        super.init()
        refreshRecordings()
    }
    func refreshRecordings() {
        savedRecordings = ((try? FileManager.default.contentsOfDirectory(at: recordingFolder, includingPropertiesForKeys: [.creationDateKey])) ?? [])
            .filter { ["wav", "m4a"].contains($0.pathExtension) }
            .sorted { ((try? $0.resourceValues(forKeys: [.creationDateKey]).creationDate) ?? .distantPast) > ((try? $1.resourceValues(forKeys: [.creationDateKey]).creationDate) ?? .distantPast) }
    }
    func prepareRecording() throws -> URL {
        // Detach earlier failures; only explicit cancellation or success deletes audio.
        audioURL = nil
        try FileManager.default.createDirectory(at: recordingFolder, withIntermediateDirectories: true)
        return recordingFolder
    }
    func saveBackend(for url: URL) {
        try? (recordingBackend ?? "codex").write(to: url.appendingPathExtension("backend"), atomically: true, encoding: .utf8)
    }
    func retryRecording(_ url: URL) {
        guard !busy else { return }
        audioURL = url
        recordingBackend = (try? String(contentsOf: url.appendingPathExtension("backend"), encoding: .utf8)) ?? (url.pathExtension == "wav" ? "doubao" : "codex")
        captureTarget()
        transcribe()
    }
    func shutdown() {
        session = UUID(); timer?.invalidate(); timer = nil
        recorder?.stop(); recorder = nil
        live?.cancel(); live = nil
        process?.terminate(); process = nil
        // Keep unfinished audio on disk across quit and restart.
    }
    var process: Process?
    var started = Date()
    var session = UUID()
    var showDetails: (() -> Void)?
    var busy: Bool { [.requesting, .recording, .transcribing, .inserting].contains(phase) }
    var title: String {
        switch phase {
        case .idle: return "Speak. Let it flow."
        case .requesting: return "Microphone access"
        case .recording: return "Listening"
        case .transcribing: return "Transcribing"
        case .inserting: return "" // Delivery is deliberately invisible.
        case .done: return message
        case .failed: return transcript.isEmpty ? "Transcription failed" : "Could not paste"
        }
    }
    var usesDoubao: Bool { (recordingBackend ?? UserDefaults.standard.string(forKey: "fnBackend") ?? "codex") == "doubao" }
    var liveProvider: LiveProvider? { LiveProvider(rawValue: recordingBackend ?? UserDefaults.standard.string(forKey: "fnBackend") ?? "codex") }
    var usesLiveCaptions: Bool { liveProvider != nil }
    func binaryPath(_ name: String) -> String? {
        let candidates = [Bundle.main.resourcePath.map { $0 + "/" + name } ?? "", "/opt/homebrew/bin/" + name, "/usr/local/bin/" + name, NSHomeDirectory() + "/.cargo/bin/" + name, NSHomeDirectory() + "/.local/bin/" + name]
        return candidates.first(where: { FileManager.default.isExecutableFile(atPath: $0) })
    }
    var usesWeType: Bool { recordingBackend == "wetype" }
    func confirmUpload() -> Bool {
        let key = usesWeType ? "wetypeUploadConsent" : (usesDoubao ? "doubaoUploadConsent" : "uploadConsent")
        if UserDefaults.standard.bool(forKey: key) { return true }
        let alert = NSAlert()
        alert.messageText = usesWeType ? "Transcribe with WeType" : usesDoubao ? "Transcribe with Doubao" : "Transcribe with Codex"
        alert.informativeText = usesWeType ? "Recordings will be sent to WeChat Input through an unofficial third-party provider (dicta-asr). It registers a device and stores credentials locally. The published binary is available, but its source repository is currently unavailable. Starting a recording authorizes sending it to WeChat Input." : usesDoubao
            ? "Recordings will be sent to Doubao through FreeASR's unofficial IME protocol. FreeASR registers a device and stores credentials locally. No paid API key is configured. Availability is not guaranteed. Starting a recording authorizes sending that recording to Doubao."
            : "Recordings will be sent to ChatGPT using your local Codex sign-in. Starting a recording authorizes sending that recording to ChatGPT."
        alert.addButton(withTitle: "Agree and Continue")
        alert.addButton(withTitle: "Cancel")
        guard alert.runModal() == .alertFirstButtonReturn else { return false }
        UserDefaults.standard.set(true, forKey: key)
        return true
    }
    func toggle(backend: String? = nil) {
        if phase == .recording { stop(); return }
        guard !busy else { return }
        recordingBackend = backend ?? UserDefaults.standard.string(forKey: "fnBackend") ?? "codex"
        captureTarget()
        guard confirmUpload() else { return }
        phase = .requesting
        let token = UUID(); session = token
        Task {
            let allowed = await AVCaptureDevice.requestAccess(for: .audio)
            guard session == token else { return }
            guard allowed else { fail("Microphone access is off. Enable Voice Pill in System Settings > Privacy & Security > Microphone."); return }
            begin()
        }
    }
    func beginLive() {
        guard let provider = liveProvider, let binary = binaryPath(provider.binaryName) else { fail("Missing live transcription backend"); return }
        do {
            let folder = try prepareRecording()
            let url = folder.appendingPathComponent(UUID().uuidString + ".wav")
            audioURL = url
            saveBackend(for: url)
            let token = session
            let stream = LiveTranscriber(); live = stream; liveNeedsRetry = false
            liveText = ""; transcript = ""; message = ""; seconds = 0; started = Date()
            try stream.start(binary: binary, provider: provider, recording: url, punctuation: UserDefaults.standard.bool(forKey: "doubaoPunctuation"), update: { [weak self] text in
                guard let self, self.session == token else { return }
                self.updateCaption(text)
            }, completion: { [weak self] result in
                guard let self, self.session == token else { return }
                // A dropped network stream must not stop the microphone.
                // Keep the complete WAV and retry only when the user releases Fn.
                if self.phase == .recording {
                    self.trace("live_ended_while_recording_fallback_pending")
                    self.liveNeedsRetry = true
                    return
                }
                self.timer?.invalidate(); self.timer = nil
                self.live?.finish(); self.live = nil
                switch result {
                case .success(let text):
                    guard self.phase == .transcribing else { self.fail("Speech session ended before you released Fn. Please retry."); return }
                    self.transcriptionElapsed = Date().timeIntervalSince(self.transcriptionStarted)
                    self.finishTranscription(text)
                case .failure: self.transcribe()
                }
            })
            phase = .recording
            timer = Timer.scheduledTimer(withTimeInterval: 0.2, repeats: true) { [weak self] _ in
                Task { @MainActor in
                    guard let self, self.phase == .recording else { return }
                    self.seconds = Int(Date().timeIntervalSince(self.started))
                    // Keep recording through a stream disconnect; release retries the full WAV.
                }
            }
        } catch { live?.cancel(); live = nil; fail(error.localizedDescription) }
    }
    func begin() {
        if usesLiveCaptions { beginLive(); return }
        liveText = ""
        do {
            // Remove only an earlier recording owned by this app when starting afresh.
            let folder = try prepareRecording()
            let url = folder.appendingPathComponent(UUID().uuidString + (usesWeType ? ".wav" : ".m4a"))
            audioURL = url
            saveBackend(for: url)
            let settings: [String: Any] = usesWeType
                ? [AVFormatIDKey: kAudioFormatLinearPCM, AVSampleRateKey: 16000, AVNumberOfChannelsKey: 1, AVLinearPCMBitDepthKey: 16, AVLinearPCMIsFloatKey: false, AVLinearPCMIsBigEndianKey: false]
                : [AVFormatIDKey: kAudioFormatMPEG4AAC, AVSampleRateKey: 44100, AVNumberOfChannelsKey: 1, AVEncoderAudioQualityKey: AVAudioQuality.high.rawValue]
            let r = try AVAudioRecorder(url: url, settings: settings)
            r.delegate = self; r.isMeteringEnabled = true
            guard r.record() else { throw NSError(domain: "VoicePill", code: 1, userInfo: [NSLocalizedDescriptionKey: "Could not start recording. Check your microphone."]) }
            recorder = r; started = Date(); seconds = 0; smoothedLevel = 0; levels = Array(repeating: 0, count: 19); transcript = ""; message = ""; phase = .recording
            timer = Timer.scheduledTimer(withTimeInterval: 0.08, repeats: true) { [weak self] _ in
                Task { @MainActor in
                    guard let self, self.phase == .recording else { return }
                    self.recorder?.updateMeters()
                    let rms = self.recorder?.averagePower(forChannel: 0) ?? -70
                    let peak = self.recorder?.peakPower(forChannel: 0) ?? -70
                    // Speech at -40...-20 dB needs perceptual scaling, not linear
                    // amplitude, otherwise most microphone input looks flat.
                    let db = max(rms, peak - 8)
                    let normalized = CGFloat(max(0, min(1, (db + 52) / 38)))
                    let amplitude = pow(normalized, 0.8)
                    let response: CGFloat = amplitude > self.smoothedLevel ? 0.85 : 0.5
                    self.smoothedLevel += (amplitude - self.smoothedLevel) * response
                    self.levels.removeFirst(); self.levels.append(self.smoothedLevel)
                    self.seconds = Int(Date().timeIntervalSince(self.started))
                    if self.seconds >= 300 { self.stop() }
                }
            }
        } catch { fail(error.localizedDescription) }
    }
    func trace(_ reason: String) {
        let path = FileManager.default.temporaryDirectory.appendingPathComponent("voice-pill-events.log")
        let line = "\(Date().timeIntervalSince1970) \(session) \(reason) elapsed=\(Int(Date().timeIntervalSince(started))) phase=\(phase)\n"
        if ((try? path.resourceValues(forKeys: [.fileSizeKey]).fileSize) ?? 0) > 65536 { try? Data().write(to: path) }
        if !FileManager.default.fileExists(atPath: path.path) { FileManager.default.createFile(atPath: path.path, contents: nil) }
        if let file = try? FileHandle(forWritingTo: path) { file.seekToEndOfFile(); file.write(Data(line.utf8)); try? file.close() }
    }
    func stop() {
        trace("stop_requested")
        timer?.invalidate(); timer = nil
        recorder?.stop(); recorder = nil
        guard Date().timeIntervalSince(started) > 0.35 else { cancel(); return }
        if let live {
            phase = .transcribing; transcriptionStarted = Date(); live.finish()
            if liveNeedsRetry { session = UUID(); live.cancel(); self.live = nil; transcribe(); return }
            let token = session
            let finishTimeout = liveProvider?.finishTimeout ?? 35
            Task { @MainActor in
                try? await Task.sleep(for: .seconds(finishTimeout))
                guard session == token, phase == .transcribing, self.live != nil else { return }
                session = UUID(); self.live?.cancel(); self.live = nil
                transcribe()
            }
        } else { transcribe() }
    }
    func cancel() {
        trace("cancel_requested")
        session = UUID(); timer?.invalidate(); timer = nil
        recorder?.stop(); recorder = nil
        live?.cancel(); live = nil
        process?.terminate(); process = nil
        discardAudio(); phase = .idle
    }
    func discardAudio() {
        if let audioURL {
            try? FileManager.default.removeItem(at: audioURL)
            try? FileManager.default.removeItem(at: audioURL.appendingPathExtension("backend"))
        }
        audioURL = nil
        refreshRecordings()
    }
    func fail(_ text: String) { message = text; phase = .failed; refreshRecordings() }
    func transcribe() {
        guard let audioURL else { return }
        guard confirmUpload() else { return }
        let doubao = usesDoubao
        let wetype = usesWeType
        let name = wetype ? "wetype-asr" : doubao ? "freeasr" : "codex-asr"
        guard let binary = binaryPath(name) else { fail("Missing transcription backend: " + name); return }
        phase = .transcribing
        transcriptionStarted = Date()
        let token = UUID(); session = token
        let job = Process()
        job.executableURL = URL(fileURLWithPath: binary)
        if wetype {
            job.arguments = ["--input", audioURL.path]
        } else if doubao {
            let credentials = NSHomeDirectory() + "/Library/Application Support/VoicePill/doubao-credentials.json"
            job.arguments = ["transcribe", audioURL.path, "--format", "text", "--quiet", "--credential-path", credentials]
            if UserDefaults.standard.object(forKey: "doubaoPunctuation") as? Bool != true {
                job.arguments?.append("--no-punctuation")
            }
        } else {
            job.arguments = [audioURL.path, "--request-timeout-seconds", "90", "--connect-timeout-seconds", "15"]
        }
        var env = ProcessInfo.processInfo.environment
        env["PATH"] = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
        env["CODEX_ASR_BIN"] = binary
        env["TRANSCRIBE_AUDIO_REQUEST_TIMEOUT_SECONDS"] = "90"
        env["TRANSCRIBE_AUDIO_MAX_ATTEMPTS"] = "1"
        if wetype {
            env["XDG_CONFIG_HOME"] = NSHomeDirectory() + "/Library/Application Support/VoicePill"
        }
        job.environment = env
        // Files avoid pipe-buffer deadlocks even when an upstream process is verbose.
        let out = audioURL.appendingPathExtension("txt")
        let err = audioURL.appendingPathExtension("log")
        FileManager.default.createFile(atPath: out.path, contents: nil)
        FileManager.default.createFile(atPath: err.path, contents: nil)
        do {
            let stdout = try FileHandle(forWritingTo: out)
            let stderr = try FileHandle(forWritingTo: err)
            job.standardOutput = stdout; job.standardError = stderr
            job.terminationHandler = { [weak self] completed in
                try? stdout.close(); try? stderr.close()
                let text = (try? String(contentsOf: out, encoding: .utf8))?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                let detail = (try? String(contentsOf: err, encoding: .utf8)) ?? ""
                try? FileManager.default.removeItem(at: out); try? FileManager.default.removeItem(at: err)
                Task { @MainActor in
                    guard let self, self.session == token else { return }
                    self.process = nil
                    guard completed.terminationStatus == 0, !text.isEmpty else {
                        self.transcriptionElapsed = Date().timeIntervalSince(self.transcriptionStarted)
                        self.timingSummary = String(format: "Transcription failed after %.2f s", self.transcriptionElapsed)
                        self.fail(detail.isEmpty ? "No speech was recognized. The recording is saved for retry." : String(detail.prefix(1600))); return
                    }
                    self.transcriptionElapsed = Date().timeIntervalSince(self.transcriptionStarted)
                    self.finishTranscription(text)
                }
            }
            process = job
            try job.run()
            if wetype {
                Task { @MainActor in
                    try? await Task.sleep(for: .seconds(90))
                    guard self.session == token, job.isRunning else { return }
                    self.session = UUID(); job.terminate(); self.process = nil
                    self.fail("WeType timed out. Your recording is saved for retry.")
                }
            }
        } catch {
            process = nil
            try? FileManager.default.removeItem(at: out); try? FileManager.default.removeItem(at: err)
            fail(error.localizedDescription)
        }
    }
    func updateCaption(_ text: String) {
        liveText = text
    }
    func finishTranscription(_ text: String) {
        transcript = text
        discardAudio()
        deliver(text)
    }
    func copyTranscript() {
        NSPasteboard.general.clearContents(); NSPasteboard.general.setString(transcript, forType: .string)
        message = "Copied"; phase = .done
    }
    func deliver(_ text: String) {
        guard UserDefaults.standard.object(forKey: "autoPaste") as? Bool ?? true else {
            copyTranscript(); return
        }
        guard AXIsProcessTrusted() else {
            fail("Automatic paste requires Accessibility access. Click Allow Accessibility, enable Voice Pill in System Settings, then click Paste to Original App. Your text is saved."); return
        }
        guard let target, !target.isTerminated else {
            fail("The original app is unavailable. Start a new recording from your target text field. Your text is saved."); return
        }
        phase = .inserting
        let token = session
        let deliveryStarted = Date()
        Task { @MainActor in
            let wasActive = target.isActive
            if !wasActive { target.activate(options: []) }
            for _ in 0..<20 {
                if target.isActive { break }
                try? await Task.sleep(for: .milliseconds(50))
                guard session == token else { return }
            }
            guard session == token, target.isActive else {
                fail("Could not switch to \(destination). Focus the original text field and retry."); return
            }
            var focusChanged = !wasActive
            if let element = targetElement {
                var focused: CFTypeRef?
                _ = AXUIElementCopyAttributeValue(AXUIElementCreateApplication(target.processIdentifier), kAXFocusedUIElementAttribute as CFString, &focused)
                if focused == nil || !CFEqual(element, focused!) {
                    _ = AXUIElementSetAttributeValue(element, kAXFocusedAttribute as CFString, kCFBooleanTrue)
                    focusChanged = true
                }
            }
            if focusChanged { try? await Task.sleep(for: .milliseconds(120)) }
            guard session == token, target.isActive else { return }
            do {
                try await PasteController.paste(text, targetPID: target.processIdentifier)
            } catch {
                fail(error.localizedDescription); return
            }
            // Read back promptly; slow editors get up to 600 ms, fast ones don't wait.
            var confirmed = false
            for _ in 0..<12 {
                var value: CFTypeRef?
                let appElement = AXUIElementCreateApplication(target.processIdentifier)
                var focus: CFTypeRef?
                if AXUIElementCopyAttributeValue(appElement, kAXFocusedUIElementAttribute as CFString, &focus) == .success,
                   let focus, CFGetTypeID(focus) == AXUIElementGetTypeID() {
                    _ = AXUIElementCopyAttributeValue(focus as! AXUIElement, kAXValueAttribute as CFString, &value)
                }
                if let content = value as? String, content.contains(text) { confirmed = true; break }
                try? await Task.sleep(for: .milliseconds(50))
                guard session == token else { return }
            }
            let deliveryElapsed = Date().timeIntervalSince(deliveryStarted)
            timingSummary = String(format: "Last run: transcription %.2f s · paste %.2f s", transcriptionElapsed, deliveryElapsed)
            if confirmed {
                message = "Pasted"; phase = .done
            } else {
                fail("Paste was sent, but delivery could not be confirmed. Focus an editable text field and retry. Your text is saved.")
            }
        }
    }
    nonisolated func audioRecorderEncodeErrorDidOccur(_ recorder: AVAudioRecorder, error: Error?) {
        Task { @MainActor in self.timer?.invalidate(); self.fail(error?.localizedDescription ?? "Audio encoding failed.") }
    }
}

final class PermissionStatus: ObservableObject {
    @Published var trusted = AXIsProcessTrusted()
}

struct DetailsView: View {
    @ObservedObject var model: VoiceModel
    @AppStorage("autoPaste") var autoPaste = true
    @AppStorage("fnBackend") var fnBackend = "codex"
    @AppStorage("doubaoPunctuation") var doubaoPunctuation = false
    @StateObject private var permission = PermissionStatus()
    private let permissionPoll = Timer.publish(every: 1, on: .main, in: .common).autoconnect()
    var body: some View {
        ScrollView { VStack(alignment: .leading, spacing: 20) {
            HStack {
                Image(systemName: "waveform").font(.title2)
                VStack(alignment: .leading, spacing: 3) {
                    Text("Voice Pill").font(.title2.weight(.semibold))
                    Text("Speak. Let it flow.").foregroundStyle(.secondary)
                }
                Spacer()
                Text("Hold fn").font(.system(.body, design: .monospaced)).padding(10).glassEffect()
            }
            Divider()
            Text("Place the cursor in a text field. Hold Fn for the selected provider, or Ctrl + Fn for Codex. Release to paste. A quick tap does not record. Text is pasted into the app where you started. Alternative shortcut: ⌃⌥Space.").foregroundStyle(.secondary)
            Picker("Speech provider", selection: $fnBackend) {
                Text("Codex · Live captions").tag("codex")
                Text("Doubao · Live captions").tag("doubao")
            }.disabled(model.busy)
            Text("Fn and ⌃⌥Space use the selected provider. Ctrl + Fn always uses Codex live captions.").font(.caption).foregroundStyle(.secondary)
            if fnBackend == "doubao" { Toggle("Doubao automatic punctuation", isOn: $doubaoPunctuation).disabled(model.busy) }
            Toggle("Automatically paste after transcription", isOn: $autoPaste)
            HStack {
                Button("Allow Accessibility…") {
                    let options = [kAXTrustedCheckOptionPrompt.takeUnretainedValue() as String: true] as CFDictionary
                    _ = AXIsProcessTrustedWithOptions(options)
                }
                Label(permission.trusted ? "Access granted" : "Access not granted for this version", systemImage: permission.trusted ? "checkmark.circle.fill" : "exclamationmark.circle")
                    .font(.caption).foregroundStyle(permission.trusted ? .green : .orange)
            }
            Text("In System Settings > Keyboard, set “Press 🌐 key to” to “Do Nothing” to avoid opening emoji or Dictation. If Fn does not respond after granting access, quit and reopen Voice Pill.")
                .font(.caption).foregroundStyle(.secondary)
            if AVCaptureDevice.authorizationStatus(for: .audio) != .authorized {
                Button("Set Up Microphone…") { model.toggle() }
            }
            if model.phase == .failed {
                Text(model.message).font(.callout).foregroundStyle(.red).textSelection(.enabled).fixedSize(horizontal: false, vertical: true)

            }
            if !model.savedRecordings.isEmpty {
                Text("Saved recordings").font(.headline)
                Text("Recordings stay here until transcription succeeds. Retry sends the audio again using its original provider.").font(.caption).foregroundStyle(.secondary)
                ForEach(model.savedRecordings, id: \.path) { url in
                    HStack {
                        let date = (try? url.resourceValues(forKeys: [.creationDateKey]).creationDate) ?? Date()
                        Text(date, format: .dateTime.month().day().hour().minute().second()).font(.caption)
                        Spacer()
                        Button("Retry") { model.retryRecording(url) }.disabled(model.busy)
                    }
                }
            }
            if !model.transcript.isEmpty {
                ScrollView { Text(model.transcript).frame(maxWidth: .infinity, alignment: .leading).textSelection(.enabled).padding(12) }
                    .frame(height: 140).background(.quaternary.opacity(0.3), in: RoundedRectangle(cornerRadius: 12))
                HStack {
                    Button("Paste to Original App") { model.deliver(model.transcript) }.disabled(model.busy)
                    Button("Copy Text") { model.copyTranscript() }
                }
            }
            if !model.timingSummary.isEmpty { Text(model.timingSummary).font(.caption).foregroundStyle(.secondary) }
            Button("Quit Voice Pill") { model.shutdown(); NSApp.terminate(nil) }.disabled(model.busy)
            Spacer(minLength: 0)
            Text(fnBackend == "wetype" ? "Fn: WeType experimental · recordings are sent to WeChat Input through dicta-asr. Ctrl + Fn: Codex. No paid API key configured." : fnBackend == "doubao" ? "Powered by FreeASR · unofficial Doubao IME protocol. Audio is processed by Doubao. No paid API key required. Temporary audio is deleted after success." : "Powered by codex-asr. Audio is sent to ChatGPT using your local Codex sign-in. Temporary audio is deleted after success.")
                .font(.caption).foregroundStyle(.secondary)
        }.padding(28) }.frame(width: 480, height: 610)
        .onReceive(permissionPoll) { _ in permission.trusted = AXIsProcessTrusted() }
    }
}

final class FloatingPanel: NSPanel {
    override var canBecomeKey: Bool { false }
    override var canBecomeMain: Bool { false }
}

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    let model = VoiceModel()
    var hud: HUD!
    private var hudSignature = ""
    var details: NSWindow?
    var hotKey: EventHotKeyRef?
    var fnHeld = false
    var fnRecording = false
    var fnCodex = false
    var fnDelay: Task<Void, Never>?
    var globalMonitor: Any?
    var localMonitor: Any?
    var phaseObserver: AnyCancellable?
    var hideTask: Task<Void, Never>?
    var appObserver: NSObjectProtocol?

    func handleKey(_ event: NSEvent) {
        if event.type == .keyDown {
            if event.keyCode == 53 && model.phase == .recording { model.cancel(); fnRecording = false; return }
            // Fn + arrows, function keys, and other shortcuts are not dictation.
            if fnHeld {
                fnDelay?.cancel()
                // Once recording starts, only Escape or Fn release may end it.
                // Incidental key presses must not discard captured speech.
            }
            return
        }
        if fnHeld && !fnRecording { fnCodex = event.modifierFlags.contains(.control) }
        guard event.keyCode == 63 else { return }
        let held = event.modifierFlags.contains(.function)
        guard held != fnHeld else { return }
        fnHeld = held
        if held {
            guard !model.busy,
                  event.modifierFlags.intersection([.command, .option, .shift]).isEmpty else { return }
            fnDelay?.cancel()
            fnCodex = event.modifierFlags.contains(.control)
            fnDelay = Task { @MainActor [weak self] in
                try? await Task.sleep(for: .milliseconds(180))
                guard !Task.isCancelled, let self, self.fnHeld else { return }
                self.fnRecording = true
                self.showPill()
                self.model.toggle(backend: self.fnCodex ? "codex" : (UserDefaults.standard.string(forKey: "fnBackend") ?? "codex"))
                if !self.fnHeld && self.model.phase == .requesting { self.model.cancel() }
            }
        } else {
            fnDelay?.cancel()
            guard fnRecording else { return }
            fnRecording = false
            model.trace("fn_release")
            if model.phase == .recording { model.stop() }
            else if model.phase == .requesting { model.cancel() }
        }
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)
        model.lastExternalApp = NSWorkspace.shared.frontmostApplication
        appObserver = NSWorkspace.shared.notificationCenter.addObserver(forName: NSWorkspace.didActivateApplicationNotification, object: nil, queue: .main) { [weak self] notification in
            guard let app = notification.userInfo?[NSWorkspace.applicationUserInfoKey] as? NSRunningApplication,
                  app.processIdentifier != ProcessInfo.processInfo.processIdentifier else { return }
            Task { @MainActor [weak self] in self?.model.lastExternalApp = app }
        }
        hud = HUD()
        if CommandLine.arguments.contains("--preview-recording") {
            if CommandLine.arguments.contains("--preview-codex") { model.recordingBackend = "codex" }
            model.seconds = 0
            model.liveText = CommandLine.arguments.contains("--preview-expansion") ? "" : "Live words appear here"
            if CommandLine.arguments.contains("--preview-expansion") {
                Task { @MainActor in
                    try? await Task.sleep(for: .seconds(2))
                    for phrase in ["Live words", "Live words appear", "Live words appear here", "Live words appear here and move smoothly as you speak"] {
                        model.liveText = phrase
                        try? await Task.sleep(for: .milliseconds(500))
                    }
                }
            }
            model.levels = [0.1, 0.3, 0.7, 1, 0.5, 0.2, 0.4, 0.9, 0.6, 0.2, 0.5, 0.8, 0.3]
            model.phase = .recording
        }
        phaseObserver = model.objectWillChange.sink { [weak self] _ in
            DispatchQueue.main.async { self?.updateHUD() }
        }
        globalMonitor = NSEvent.addGlobalMonitorForEvents(matching: [.flagsChanged, .keyDown]) { [weak self] event in
            self?.handleKey(event)
        }
        localMonitor = NSEvent.addLocalMonitorForEvents(matching: [.flagsChanged, .keyDown]) { [weak self] event in
            self?.handleKey(event); return event
        }
        model.showDetails = { [weak self] in self?.openDetails() }
        var eventType = EventTypeSpec(eventClass: OSType(kEventClassKeyboard), eventKind: UInt32(kEventHotKeyPressed))
        InstallEventHandler(GetApplicationEventTarget(), { _, _, pointer in
            guard let pointer else { return noErr }
            let delegate = Unmanaged<AppDelegate>.fromOpaque(pointer).takeUnretainedValue()
            Task { @MainActor in delegate.model.toggle() }
            return noErr
        }, 1, &eventType, Unmanaged.passUnretained(self).toOpaque(), nil)
        let result = RegisterEventHotKey(UInt32(kVK_Space), UInt32(controlKey | optionKey), EventHotKeyID(signature: 0x5650494C, id: 1), GetApplicationEventTarget(), 0, &hotKey)
        if result != noErr { model.fail("Could not register the shortcut. It may be used by another app. Use Fn to start recording.") }
        NotificationCenter.default.addObserver(self, selector: #selector(position), name: NSApplication.didChangeScreenParametersNotification, object: nil)
        if !UserDefaults.standard.bool(forKey: model.usesDoubao ? "doubaoUploadConsent" : "uploadConsent") || CommandLine.arguments.contains("--show-settings") { openDetails() }
        if CommandLine.arguments.contains("--preview-pill") {
            model.captureTarget()
            showPill()
        }
        if CommandLine.arguments.contains("--verify-input-test") { runInputTest() }
        let diagnostic = "AX trusted: \(AXIsProcessTrusted())\nautoPaste: \(UserDefaults.standard.object(forKey: "autoPaste") as? Bool ?? true)\n"
        try? diagnostic.write(toFile: "/tmp/voice-pill-status.txt", atomically: true, encoding: .utf8)
    }
    func runInputTest() {
        Task { @MainActor in
            var report = "AX trusted: \(AXIsProcessTrusted())\n"
            defer { try? report.write(toFile: "/tmp/voice-pill-input-test.txt", atomically: true, encoding: .utf8) }
            try? await Task.sleep(for: .milliseconds(800))
            let inputBundle = CommandLine.arguments.first(where: { $0.hasPrefix("--verify-input-bundle=") })?.replacingOccurrences(of: "--verify-input-bundle=", with: "") ?? "com.lawted.voicepill.inputtest"
            guard let app = NSRunningApplication.runningApplications(withBundleIdentifier: inputBundle).first else { report += "FAIL: test fixture not running\n"; return }
            app.activate(options: [])
            try? await Task.sleep(for: .milliseconds(500))
            model.captureTarget()
            guard model.target?.processIdentifier == app.processIdentifier else { report += "UNVERIFIED: test fixture did not become the focused target\n"; return }
            let element = model.targetElement
            let phrases = ["Hello,我是Lotte。", "The chicks will hatch tomorrow."]
            for (index, text) in phrases.enumerated() {
                if index == 1 { model.targetElement = nil }
                model.updateCaption(text)
                report += "caption \(index): \(model.liveText == text ? "PASS" : "FAIL")\n"
                model.phase = .transcribing
                model.finishTranscription(text)
                for _ in 0..<40 {
                    try? await Task.sleep(for: .milliseconds(100))
                    if !model.busy { break }
                }
                try? await Task.sleep(for: .milliseconds(200))
                var value: CFTypeRef?
                if let element { _ = AXUIElementCopyAttributeValue(element, kAXValueAttribute as CFString, &value) }
                let received = (value as? String ?? "").contains(text) && model.transcript == text
                report += "test \(index): \(received ? "PASS" : "FAIL") path=\(model.message) \(model.timingSummary)\n"
            }
            try? report.write(toFile: "/tmp/voice-pill-input-test.txt", atomically: true, encoding: .utf8)
        }
    }
    @objc func position() { hud.reposition() }
    @objc func toggle() { showPill(); model.toggle() }
    private func updateHUD(force: Bool = false) {
        let signature = "\(model.phase)|\(model.recordingBackend ?? "codex")|\(model.liveText)"
        if force || signature != hudSignature {
            hudSignature = signature
            let content: PillContent = model.usesLiveCaptions ? .transcript : .waveform
            if HUDTuning.shared.content != content { HUDTuning.shared.content = content }
            hud.setContext(app: model.lastExternalApp, icon: model.targetIcon)
            switch model.phase {
            case .recording: hud.show(.listening, partial: model.usesLiveCaptions ? model.liveText : "")
            case .requesting: hud.show(.info, message: "Preparing microphone")
            case .transcribing: hud.show(.thinking)
            default:
                if force && model.phase == .idle { hud.show(.info, message: "Hold fn to speak") }
                else { hud.hide() }
            }
        }
        if model.phase == .recording { hud.setNormalizedLevel(model.levels.last ?? 0) }
    }
    @objc func showPill() { updateHUD(force: true) }
    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        // LaunchServices ignores new --args when an instance already exists.
        // Reopening the app must explicitly reveal the pill.
        if !model.busy { model.captureTarget() }
        openDetails()
        return true
    }
    @objc func openDetails() {
        if details == nil {
            let window = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 480, height: 610), styleMask: [.titled, .closable], backing: .buffered, defer: false)
            window.title = "Voice Pill"; window.isReleasedWhenClosed = false
            let settingsHost = NSHostingView(rootView: DetailsView(model: model))
            settingsHost.sizingOptions = []
            settingsHost.frame = NSRect(x: 0, y: 0, width: 480, height: 610)
            window.contentView = settingsHost; window.center(); details = window
        }
        NSApp.activate(ignoringOtherApps: true); details?.makeKeyAndOrderFront(nil)
    }
    @objc func quit() { model.shutdown(); NSApp.terminate(nil) }
    func applicationWillTerminate(_ notification: Notification) { model.shutdown() }
}
