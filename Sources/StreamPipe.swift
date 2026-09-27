import Foundation
import Darwin

// FileHandle.read(upToCount:) can wait for the requested byte count on a pipe.
// POSIX read returns currently available bytes, so short ASR events appear now.
enum StreamPipe {
    static func readChunk(from handle: FileHandle) throws -> Data {
        var bytes = [UInt8](repeating: 0, count: 4096)
        while true {
            let count = Darwin.read(handle.fileDescriptor, &bytes, bytes.count)
            if count >= 0 { return Data(bytes.prefix(count)) }
            if errno == EINTR { continue }
            throw NSError(domain: NSPOSIXErrorDomain, code: Int(errno))
        }
    }
}
