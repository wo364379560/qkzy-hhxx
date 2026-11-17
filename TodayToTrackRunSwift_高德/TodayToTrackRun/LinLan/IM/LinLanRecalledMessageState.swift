import Foundation

/// Stores recall evidence, never message contents or authentication data.
final class LinLanRecalledMessageState {
    enum Kind: String, Codable { case unknown, own, moderator }
    struct Scope: Equatable { let key: String; let generation: UInt64 }
    struct Intent: Equatable { let scope: Scope; let id: String; let serial: UInt64 }
    private struct Evidence: Codable {
        var id: String
        var kind: Kind
        var actor: String
        var sender: String
        var group: String
    }
    private struct Pending { let intent: Intent; let evidence: Evidence }
    private let lock = NSRecursiveLock()
    private let read: (String) -> Data?
    private let write: (String, Data) -> Void
    private let limit: Int
    private var active: Scope?
    private var generation: UInt64 = 0
    private var serial: UInt64 = 0
    private var records: [String: Evidence] = [:]
    private var order: [String] = []
    private var pending: [String: Pending] = [:]

    init(read: @escaping (String) -> Data? = { _ in nil }, write: @escaping (String, Data) -> Void = { _, _ in }, limit: Int = 2048) {
        self.read = read; self.write = write; self.limit = max(1, limit)
    }
    private func locked<T>(_ body: () throws -> T) rethrows -> T {
        lock.lock(); defer { lock.unlock() }; return try body()
    }
    @discardableResult func activate(_ key: String) -> Scope { locked {
        if let active, active.key == key { return active }
        generation &+= 1
        let scope = Scope(key: key, generation: generation)
        active = scope; records.removeAll(); order.removeAll(); pending.removeAll()
        if let raw = read(key), let values = try? JSONDecoder().decode([Evidence].self, from: raw) {
            for value in values.suffix(limit) where !value.id.isEmpty {
                order.removeAll { $0 == value.id }; order.append(value.id); records[value.id] = value
            }
        }
        return scope
    } }
    func current() -> Scope? { locked { active } }
    func isCurrent(_ scope: Scope?) -> Bool { locked { scope != nil && active == scope } }
    func deactivate() { locked { generation &+= 1; active = nil; records.removeAll(); order.removeAll(); pending.removeAll() } }

    func begin(_ scope: Scope, id: String, kind: Kind, sender: String, group: String, actor: String) throws -> Intent { try locked {
        guard active == scope else { throw CancellationError() }
        guard !id.isEmpty, pending[id] == nil else { throw RecallError.inProgress }
        serial &+= 1
        let intent = Intent(scope: scope, id: id, serial: serial)
        pending[id] = Pending(intent: intent, evidence: Evidence(id: id, kind: kind, actor: actor, sender: sender, group: group))
        return intent
    } }
    func cancel(_ intent: Intent) { locked {
        if active == intent.scope, pending[intent.id]?.intent == intent { pending.removeValue(forKey: intent.id) }
    } }
    @discardableResult func confirm(_ intent: Intent) -> Bool { locked {
        guard active == intent.scope, let request = pending[intent.id], request.intent == intent else { return false }
        pending.removeValue(forKey: intent.id)
        remember(intent.scope, request.evidence)
        return true
    } }
    @discardableResult func receive(_ scope: Scope, id: String, actor: String = "", sender: String = "", group: String = "") -> Bool { locked {
        guard active == scope, !id.isEmpty else { return false }
        let request = pending[id]?.evidence
        let actualSender = sender.isEmpty ? request?.sender ?? "" : sender
        let actualActor = actor.isEmpty ? request?.actor ?? "" : actor
        let kind: Kind = !actualSender.isEmpty && !actualActor.isEmpty
            ? (actualSender == actualActor ? .own : .moderator) : request?.kind ?? .unknown
        remember(scope, Evidence(id: id, kind: kind, actor: actualActor, sender: actualSender, group: group.isEmpty ? request?.group ?? "" : group))
        return true
    } }
    func isRevoked(_ scope: Scope?, id: String, sdkRevoked: Bool) -> Bool { locked {
        sdkRevoked || (scope != nil && active == scope && records[id] != nil)
    } }
    func shouldHide(_ scope: Scope?, id: String, group: String, sender: String, actor: String, sdkRevoked: Bool) -> Bool { locked {
        guard isRevoked(scope, id: id, sdkRevoked: sdkRevoked) else { return false }
        let evidence = scope != nil && active == scope ? records[id] ?? pending[id]?.evidence : nil
        guard !(group.isEmpty ? evidence?.group ?? "" : group).isEmpty else { return false }
        if evidence?.kind == .moderator { return true }
        if evidence?.kind == .own { return false }
        let actualSender = sender.isEmpty ? evidence?.sender ?? "" : sender
        let actualActor = evidence?.actor.isEmpty == false ? evidence!.actor : actor
        // An absent operator is not evidence that the sender recalled the message.
        return actualSender.isEmpty || actualActor.isEmpty || actualSender != actualActor
    } }
    private func remember(_ scope: Scope, _ incoming: Evidence) {
        let previous = records[incoming.id]
        var value = incoming
        if previous?.kind == .moderator || incoming.kind == .moderator { value.kind = .moderator }
        else if incoming.kind == .unknown { value.kind = previous?.kind ?? .unknown }
        if value.actor.isEmpty { value.actor = previous?.actor ?? "" }
        if value.sender.isEmpty { value.sender = previous?.sender ?? "" }
        if value.group.isEmpty { value.group = previous?.group ?? "" }
        records[value.id] = value; order.removeAll { $0 == value.id }; order.append(value.id)
        while order.count > limit { records.removeValue(forKey: order.removeFirst()) }
        if let raw = try? JSONEncoder().encode(order.compactMap { records[$0] }) { write(scope.key, raw) }
    }
    enum RecallError: LocalizedError {
        case inProgress
        var errorDescription: String? { "消息ID无效或正在撤回，请稍候" }
    }
}
