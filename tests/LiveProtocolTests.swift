import Foundation

@main struct LiveProtocolTests {
    static func main() throws {
        var codex = LiveEventDecoder(provider: .codex)
        let data = Data("{\"type\":\"partial\",\"text\":\"你好\",\"elapsed_ms\":1500,\"revision\":1}\n".utf8)
        var updates: [String] = []
        for byte in data { updates += try codex.append(Data([byte])) }
        precondition(updates == ["你好"], "Fragmented UTF-8 or numeric fields broke captions")
        _ = try codex.append(Data("{\"type\":\"final\",\"text\":\"First sentence\",\"revision\":2}\n".utf8))
        precondition((try? codex.completed(exitStatus: 0)) == nil, "Utterance final incorrectly completed the session")
        _ = try codex.append(Data("{\"type\":\"partial\",\"text\":\"First sentence Second\"}\n{\"type\":\"result\",\"text\":\"First sentence Second sentence\"}\n".utf8))
        precondition(tryValue(codex) == "First sentence Second sentence")
        precondition((try? codex.completed(exitStatus: 1)) == nil)
        var doubao = LiveEventDecoder(provider: .doubao)
        _ = try doubao.append(Data("{\"type\":\"partial\",\"text\":\"早\"}\n{\"type\":\"final\",\"text\":\"早上好\"}\n".utf8))
        precondition(tryValue(doubao) == "早上好")
        var truncated = LiveEventDecoder(provider: .codex)
        _ = try truncated.append(Data("{\"type\":\"result\",\"text\":\"incomplete\"}".utf8))
        precondition((try? truncated.completed(exitStatus: 0)) == nil)
        print("PASS: Codex numeric events, fragmented UTF-8, whole-session completion, nonzero exit, Doubao compatibility")
    }
    static func tryValue(_ decoder: LiveEventDecoder) -> String { (try? decoder.completed(exitStatus: 0)) ?? "" }
}
