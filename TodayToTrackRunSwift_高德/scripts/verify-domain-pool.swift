import Foundation

@main struct VerifyDomainPool {
    @MainActor static func main() async throws {
        func body(_ version: Int, _ origin: String = "https://api1.prfgsu.com") -> Data {
            Data("{\"code\":0,\"data\":{\"version\":\(version),\"ttl_seconds\":3600,\"api_origins\":[\"\(origin)\"]}}".utf8)
        }
        let current = LinLanDomainPoolPolicy.parseBootstrap(body(8))!
        precondition(LinLanDomainPoolPolicy.parseBootstrap(body(0)) != nil)
        precondition(!LinLanDomainPoolPolicy.accepts(current: current, next: LinLanDomainPoolPolicy.parseBootstrap(body(7))!))
        precondition(!LinLanDomainPoolPolicy.accepts(current: current, next: LinLanDomainPoolPolicy.parseBootstrap(body(8, "https://api.uktjwg.cn"))!))
        precondition(LinLanDomainPoolPolicy.accepts(current: current, next: LinLanDomainPoolPolicy.parseBootstrap(body(9))!))
        precondition(LinLanDomainPoolPolicy.accepts(current: current, next: current))
        let encoded = try JSONEncoder().encode(current)
        precondition(LinLanDomainPoolPolicy.parseDocument(encoded) == current)
        for origin in ["http://api.prfgsu.com", "https://api.prfgsu.com/api/v1", "https://u:p@api.prfgsu.com", "https://api.prfgsu.com?x=1", "https://127.0.0.1", "https://[::1]", "https://api.prfgsu.com:8443", "https://untrusted.example"] {
            precondition(LinLanDomainPoolPolicy.parseBootstrap(body(1, origin)) == nil, origin)
        }
        precondition(LinLanDomainPoolPolicy.parseBootstrap(Data("<html>200 OK</html>".utf8)) == nil)
        precondition(LinLanDomainPoolPolicy.parseBootstrap(body(-1)) == nil)
        precondition(LinLanDomainPoolPolicy.preferMember("https://api.prfgsu.com", in: current.origins) == current.origins)
        precondition(LinLanDomainPoolPolicy.healthyPing(Data("{\"code\":0,\"data\":{\"status\":\"ok\"}}".utf8)))
        for text in ["<html>OK</html>", "{}", "{\"code\":403,\"data\":{\"status\":\"ok\"}}", "{\"code\":0,\"data\":{\"status\":\"failed\"}}"] {
            precondition(!LinLanDomainPoolPolicy.healthyPing(Data(text.utf8)))
        }
        precondition(LinLanDomainPoolPolicy.bootstrapOrigins.count == 5)
        precondition(LinLanDomainPoolPolicy.bootstrapOrigins.contains("https://api.ruizhihe.com"))
        precondition(LinLanDomainPoolPolicy.seeds == ["https://api.ruizhihe.com", "https://api.prfgsu.com", "https://api.uktjwg.cn"])
        precondition(LinLanDomainPoolPolicy.boundedCandidates(["https://api.prfgsu.com", "https://api1.prfgsu.com", "https://api.uktjwg.cn"], preferred: "https://api.prfgsu.com") == ["https://api.prfgsu.com", "https://api.uktjwg.cn"])
        precondition(LinLanDomainPoolPolicy.attemptTimeout(method: "GET", mayRetry: true, remaining: 30, requested: 60) == 15)
        precondition(LinLanDomainPoolPolicy.attemptTimeout(method: "GET", mayRetry: false, remaining: 12, requested: 60) == 12)
        precondition(LinLanDomainPoolPolicy.attemptTimeout(method: "POST", mayRetry: true, remaining: 30, requested: 60) == 30)
        for status in [502, 503, 504] { precondition(LinLanDomainPoolPolicy.isGatewayStatus(status)) }
        for status in [200, 400, 401, 403, 408, 429, 500] { precondition(!LinLanDomainPoolPolicy.isGatewayStatus(status)) }
        var request = URLRequest(url: URL(string: "https://api.ruizhihe.com/api/v1/chat/profile?q=exact%20value")!)
        request.httpMethod = "GET"
        request.setValue("Bearer same-session", forHTTPHeaderField: "Authorization")
        var probes = 0
        var future = 0
        let next = try await LinLanDomainPoolPolicy.recoveryRequest(request, allowReadRetry: true, deadline: 1000, now: { 100 },
            currentBaseURL: { "https://api.prfgsu.com/api/v1/chat" },
            recover: { failed, budget in precondition(failed == "https://api.ruizhihe.com" && budget == 8); probes += 1 },
            prepareFuture: { _ in future += 1 }, sessionIsCurrent: { true })
        precondition(next?.url?.absoluteString == "https://api.prfgsu.com/api/v1/chat/profile?q=exact%20value")
        precondition(next?.value(forHTTPHeaderField: "Authorization") == "Bearer same-session")
        precondition(probes == 1 && future == 0)
        for method in ["POST", "PUT", "DELETE"] {
            var write = request; write.httpMethod = method; write.httpBody = Data("stable-request-id".utf8)
            let result = try await LinLanDomainPoolPolicy.recoveryRequest(write, allowReadRetry: true, deadline: 1000, now: { 100 },
                currentBaseURL: { "https://api.prfgsu.com/api/v1/chat" }, recover: { _, _ in preconditionFailure("A write cannot replay") },
                prepareFuture: { _ in future += 1 }, sessionIsCurrent: { true })
            precondition(result == nil)
        }
        precondition(future == 3)
        let exhausted = try await LinLanDomainPoolPolicy.recoveryRequest(request, allowReadRetry: false, deadline: 1000, now: { 100 },
            currentBaseURL: { "https://api.prfgsu.com/api/v1/chat" }, recover: { _, _ in preconditionFailure("Read recovery exhausted") },
            prepareFuture: { _ in future += 1 }, sessionIsCurrent: { true })
        precondition(exhausted == nil && future == 4)
        var accountStillCurrent = true
        do {
            _ = try await LinLanDomainPoolPolicy.recoveryRequest(request, allowReadRetry: true, deadline: 1000, now: { 100 },
                currentBaseURL: { "https://api.prfgsu.com/api/v1/chat" }, recover: { _, _ in accountStillCurrent = false },
                prepareFuture: { _ in }, sessionIsCurrent: { accountStillCurrent })
            preconditionFailure("Account changed during recovery")
        } catch is CancellationError { }
        print("PASS: real Swift gateway recovery; GET-only retry, write future-route preparation, account fence and shared deadline")
        print("PASS: real Swift domain-pool policy, version/cache/health/origin validation")
    }
}
