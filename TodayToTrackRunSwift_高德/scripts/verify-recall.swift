import Foundation

@main struct VerifyRecall {
    static func check(_ value: @autoclosure () -> Bool, _ message: String) { precondition(value(), message) }
    static func main() throws {
        var disk: [String: Data] = [:]
        func store() -> LinLanRecalledMessageState { LinLanRecalledMessageState(read: { disk[$0] }, write: { disk[$0] = $1 }) }
        func hidden(_ store: LinLanRecalledMessageState, _ scope: LinLanRecalledMessageState.Scope?, id: String = "m", sender: String = "member", actor: String = "", sdk: Bool = false, group: String = "g") -> Bool {
            store.shouldHide(scope, id: id, group: group, sender: sender, actor: actor, sdkRevoked: sdk)
        }
        let first = store(); let a = first.activate("app:a")
        check(hidden(first, a, sdk: true), "Missing operator must not create a group recall tip")
        check(!hidden(first, a, actor: "member", sdk: true), "Explicit self recall retains its tip")
        check(hidden(first, a, actor: "admin", sdk: true), "Moderator recall stays silent")
        check(!hidden(first, a, sdk: true, group: ""), "Private recall behavior is unchanged")
        let failed = try first.begin(a, id: "failed", kind: .moderator, sender: "member", group: "g", actor: "a")
        check(!hidden(first, a, id: "failed"), "Pending requests cannot hide messages")
        first.cancel(failed); check(!first.isRevoked(a, id: "failed", sdkRevoked: false), "Failure cannot fabricate recall success")
        let moderator = try first.begin(a, id: "m", kind: .moderator, sender: "member", group: "g", actor: "a")
        check(first.confirm(moderator), "REST acknowledgement is accepted")
        check(hidden(first, a, sender: "", group: ""), "Known group survives missing SDK conversation metadata")
        first.receive(a, id: "m", actor: "member", sender: "member", group: "g")
        check(hidden(first, a), "Late metadata cannot undo confirmed moderation")
        let own = try first.begin(a, id: "own", kind: .own, sender: "a", group: "g", actor: "a")
        first.receive(a, id: "own"); check(first.confirm(own), "SDK acknowledgement is accepted")
        check(!hidden(first, a, id: "own", sender: "a"), "Legacy callback retains known self recall")
        first.receive(a, id: "remote", actor: "member")
        check(!hidden(first, a, id: "remote"), "Another device's self recall is distinguished by operator")
        first.receive(a, id: "unknown")
        check(hidden(first, a, id: "unknown"), "Legacy callback hides stale SDK objects")
        let restart = store(); let b = restart.activate("app:b")
        check(!hidden(restart, b), "No cross-account state")
        let restored = restart.activate("app:a")
        check(hidden(restart, restored), "Cold restart restores moderation")
        check(!hidden(restart, restored, id: "own", sender: "a"), "Cold restart retains self recall hint")
        check(!restart.receive(b, id: "late"), "Late previous-account callback is rejected")
        let stale = try restart.begin(restored, id: "late-ack", kind: .moderator, sender: "member", group: "g", actor: "a")
        restart.deactivate(); let next = restart.activate("app:a")
        check(!restart.confirm(stale), "Late previous-session acknowledgement is rejected")
        check(!hidden(restart, next, id: "late-ack"), "Late acknowledgement cannot remove a message")
        var raw = Data("invalid".utf8)
        let bounded = LinLanRecalledMessageState(read: { _ in raw }, write: { _, data in raw = data }, limit: 2)
        let s = bounded.activate("app:a")
        for i in 0..<3 { bounded.receive(s, id: "m\(i)") }
        check(!bounded.isRevoked(s, id: "m0", sdkRevoked: false), "Storage is bounded")
        check(bounded.isRevoked(s, id: "m2", sdkRevoked: false), "Latest recall evidence survives")
        print("PASS: real Swift recall state: missing metadata, moderator/own/private rules, REST failure, delayed cache, cold restart, account/session fences and bounded persistence")
    }
}
