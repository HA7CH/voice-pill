import Foundation

@main struct StreamPipeTests {
    static func main() throws {
        let pipe = Pipe()
        let sent = Data("{\"type\":\"partial\",\"text\":\"实时文字\"}\n".utf8)
        let start = Date()
        let writerDone = DispatchSemaphore(value: 0)
        DispatchQueue.global().async {
            try! pipe.fileHandleForWriting.write(contentsOf: sent)
            Thread.sleep(forTimeInterval: 1.5)
            try! pipe.fileHandleForWriting.close()
            writerDone.signal()
        }
        let received = try StreamPipe.readChunk(from: pipe.fileHandleForReading)
        let elapsed = Date().timeIntervalSince(start)
        precondition(received == sent, "Short UTF-8 result was lost")
        precondition(elapsed < 0.75, "Interim text was buffered until EOF")
        precondition(writerDone.wait(timeout: .now()) == .timedOut, "Writer should still be open")
        precondition(tryEOF(pipe.fileHandleForReading), "Expected EOF after writer closes")
        print(String(format: "PASS: interim delivered in %.3f s with writer open; EOF handled", elapsed))
    }
    static func tryEOF(_ handle: FileHandle) -> Bool { (try? StreamPipe.readChunk(from: handle).isEmpty) == true }
}
