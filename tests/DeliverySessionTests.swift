import Foundation

@main struct DeliverySessionTests {
    @MainActor static func main() async {
        var pending = true
        var failures: [DeliverySession.Interruption] = []
        // Focus changed during a suspended paste; the operation returns early.
        await DeliverySession.run(isCurrent: { pending }, interrupted: {
            failures.append($0); pending = false
        }, operation: { return })
        precondition(!pending && failures == [.abandoned])

        pending = true; failures = []
        await DeliverySession.run(isCurrent: { pending }, interrupted: {
            failures.append($0); pending = false
        }, operation: { pending = false })
        precondition(failures.isEmpty, "Successful delivery must not become a failure")

        pending = true; failures = []
        await DeliverySession.run(timeout: .milliseconds(20), isCurrent: { pending }, interrupted: {
            failures.append($0); pending = false
        }, operation: { try? await Task.sleep(for: .milliseconds(80)) })
        precondition(!pending && failures == [.timedOut], "A stalled paste must unlock once")

        var generation = 1
        failures = []
        await DeliverySession.run(timeout: .milliseconds(20), isCurrent: { generation == 1 }, interrupted: {
            failures.append($0)
        }, operation: {
            generation = 2 // Another session owns the state now.
            try? await Task.sleep(for: .milliseconds(40))
        })
        precondition(generation == 2 && failures.isEmpty, "Old operations must not fail a new session")
        print("PASS: abandoned paste unlocks; successful paste preserved; timeout unlocks once; stale sessions ignored")
    }
}
