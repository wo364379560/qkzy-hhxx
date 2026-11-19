// Standalone macOS integration test using the real actor and URLProtocol only.
import Foundation
nonisolated enum RuiHTTPDNSNetwork {
 static func session(requestTimeout: TimeInterval, resourceTimeout: TimeInterval) -> URLSession { .shared }
 static func prepareAfterConsent() {}
}
nonisolated final class TestClock: @unchecked Sendable {
 private let lock = NSLock(); private var value: TimeInterval = 100_000
 func now() -> TimeInterval { lock.lock(); defer { lock.unlock() }; return value }
 func advance(_ seconds: TimeInterval) { lock.lock(); value += seconds; lock.unlock() }
}
final class ProbeProtocol: URLProtocol, @unchecked Sendable {
 static let lock = NSLock()
 nonisolated(unsafe) static var requests: [URLRequest] = []
 nonisolated(unsafe) static var bootstrapDelay: TimeInterval = 0
 nonisolated(unsafe) static var version = 4
 nonisolated(unsafe) static var healthyHosts: Set<String> = ["cdnapi2.sazuts.cn"]
 nonisolated(unsafe) static var onBootstrap: (@Sendable () -> Void)?
 static let six = ["https://api.prfgsu.com", "https://api.uktjwg.cn", "https://api1.prfgsu.com", "https://api1.uktjwg.cn", "https://cdnapi1.sazuts.cn", "https://cdnapi2.sazuts.cn"]
 private let finishLock = NSRecursiveLock()
 private var stopped = false
 override class func canInit(with request: URLRequest) -> Bool { true }
 override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
 override func startLoading() {
  Self.lock.lock()
  Self.requests.append(request)
  let delay = Self.bootstrapDelay; let version = Self.version; let hosts = Self.healthyHosts; let started = Self.onBootstrap
  Self.lock.unlock()
  let bootstrap = request.url!.path == "/api/v1/public/bootstrap/domain-pool"
  if bootstrap { started?() }
  let success = bootstrap || (request.url!.path == "/api/v1/chat/ping" && hosts.contains(request.url!.host!))
  let body: [String: Any] = bootstrap ? ["code":0,"data":["version":version,"ttl_seconds":600,"api_origins":Self.six]] : ["code":0,"data":["status":"ok"]]
  let data = try! JSONSerialization.data(withJSONObject: body)
  let response = HTTPURLResponse(url: request.url!, statusCode: success ? 200 : 502, httpVersion:nil, headerFields:nil)!
  let finish: @Sendable () -> Void = {
   self.finishLock.lock(); defer { self.finishLock.unlock() }
   guard !self.stopped else { return }
   self.client?.urlProtocol(self, didReceive: response, cacheStoragePolicy:.notAllowed)
   self.client?.urlProtocol(self, didLoad:data)
   self.client?.urlProtocolDidFinishLoading(self)
  }
  if bootstrap && delay > 0 { DispatchQueue.global().asyncAfter(deadline:.now()+delay, execute:finish) } else { finish() }
 }
 override func stopLoading() { finishLock.lock(); stopped = true; finishLock.unlock() }
 static func snapshot() -> [URLRequest] { lock.lock();defer{lock.unlock()};return requests }
 static func reset(delay: TimeInterval = 0, version: Int = 4, healthy: Set<String> = ["cdnapi2.sazuts.cn"], onBootstrap: (@Sendable () -> Void)? = nil) {
  lock.lock();defer{lock.unlock()};requests=[];bootstrapDelay=delay;Self.version=version;healthyHosts=healthy;Self.onBootstrap=onBootstrap
 }
}
@main struct DomainPoolPreflightTests {
 static func main() async throws {
  let suite = "rui-domain-test-" + UUID().uuidString
  let defaults = UserDefaults(suiteName:suite)!
  defer { defaults.removePersistentDomain(forName:suite) }
  defaults.set(true,forKey:"linlan.prefs.privacy_accepted")
  let config=URLSessionConfiguration.ephemeral;config.protocolClasses=[ProbeProtocol.self]
  let session=URLSession(configuration:config)
  defer { session.invalidateAndCancel() }
  func resetCache() {
   defaults.removeObject(forKey:"linlan.domain_pool.validated_document.v1")
   defaults.removeObject(forKey:"linlan.domain_pool.last_good_origin.v5")
   defaults.removeObject(forKey:"linlan.domain_pool.validated_document_fetched.v1")
  }
  for late in [false,true] {
   resetCache()
   let started = AsyncStream<Void>.makeStream(bufferingPolicy: .bufferingNewest(1))
   ProbeProtocol.reset(delay: late ? 0.08 : 0, onBootstrap: { started.continuation.yield(()) })
   let pool=LinLanDomainPool(defaults:defaults,session:session)
   if late { await pool.ensureReady(); var it = started.stream.makeAsyncIterator(); _ = await it.next() }
   let deadline=ProcessInfo.processInfo.systemUptime+5
   async let first=pool.prepareForLogin(deadline:deadline)
   async let second=pool.prepareForLogin(deadline:deadline)
   let (a,b)=await (first,second)
   precondition(a == "https://cdnapi2.sazuts.cn/api/v1/chat" && a == b, "cold/late pool must return verified URL")
   let requests=ProbeProtocol.snapshot()
   precondition(requests.filter{$0.url?.path==LinLanAppConstants.bootstrapPath}.count==1, "refresh must stay singleflight")
   precondition(requests.allSatisfy{$0.httpMethod=="GET" && $0.httpBody==nil && $0.value(forHTTPHeaderField:"Authorization")==nil})
   let before=requests.count
   let ready=await pool.prepareForLogin(deadline:deadline)
   precondition(ready==a && ProbeProtocol.snapshot().count==before, "healthy route must not wait or probe again")
   ProbeProtocol.reset(healthy: ["cdnapi1.sazuts.cn"])
   await pool.failoverIfNeeded(failedOrigin: "https://cdnapi2.sazuts.cn")
   let noStaleHealth=await pool.prepareForLogin(deadline:ProcessInfo.processInfo.systemUptime+5)
   precondition(noStaleHealth != a, "failed route must lose its healthy flag even when all backups are cooling")
   await pool.markAvailable(origin: "https://cdnapi1.sazuts.cn")
   await pool.failoverIfNeeded(failedOrigin: "https://cdnapi2.sazuts.cn")
   let afterFailover=await pool.prepareForLogin(deadline:ProcessInfo.processInfo.systemUptime+5)
   precondition(afterFailover=="https://cdnapi1.sazuts.cn/api/v1/chat")
   precondition(a=="https://cdnapi2.sazuts.cn/api/v1/chat", "returned URL snapshot must not follow later mutable routing")
  }
  resetCache()
  let started = AsyncStream<Void>.makeStream(bufferingPolicy: .bufferingNewest(1))
  ProbeProtocol.reset(delay:0.2,onBootstrap:{started.continuation.yield(())})
  let bounded=LinLanDomainPool(defaults:defaults,session:session)
  let long = Task { await bounded.prepareForLogin(deadline:ProcessInfo.processInfo.systemUptime+2) }
  var it=started.stream.makeAsyncIterator(); _ = await it.next()
  let began=ProcessInfo.processInfo.systemUptime
  let short=await bounded.prepareForLogin(deadline:began+0.02)
  precondition(short==nil && ProcessInfo.processInfo.systemUptime-began<0.15, "joining caller must keep its own deadline")
  let longResult=await long.value
  precondition(longResult != nil, "short caller cannot cancel the shared preparation")
  precondition(ProbeProtocol.snapshot().filter{$0.url?.path==LinLanAppConstants.bootstrapPath}.count==1)

  resetCache()
  let clock=TestClock()
  let document=LinLanDomainPoolDocument(version:4,ttlSeconds:600,origins:ProbeProtocol.six)
  defaults.set(try JSONEncoder().encode(document),forKey:"linlan.domain_pool.validated_document.v1")
  defaults.set(clock.now(),forKey:"linlan.domain_pool.validated_document_fetched.v1")
  defaults.set("https://cdnapi2.sazuts.cn",forKey:"linlan.domain_pool.last_good_origin.v5")
  let expiring=LinLanDomainPool(defaults:defaults,session:session,wallNow:{clock.now()})
  await expiring.markAvailable(origin:"https://cdnapi2.sazuts.cn")
  clock.advance(600+86_401)
  let fallback=await expiring.currentBaseURL()
  precondition(!fallback.contains("sazuts.cn"),"running process must retire cache beyond TTL+24h")
  ProbeProtocol.reset(version:3)
  let downgrade=await expiring.prepareForLogin(deadline:ProcessInfo.processInfo.systemUptime+2)
  precondition(downgrade==nil,"expired cached version watermark must still reject downgrade")
  clock.advance(31)
  ProbeProtocol.reset()
  let restored=await expiring.prepareForLogin(deadline:ProcessInfo.processInfo.systemUptime+2)
  precondition(restored=="https://cdnapi2.sazuts.cn/api/v1/chat")
  defaults.set(false,forKey:"linlan.prefs.privacy_accepted")
  let denied=await expiring.prepareForLogin(deadline:ProcessInfo.processInfo.systemUptime+2)
  precondition(denied==nil,"privacy consent remains required")
  print("PASS: cold/late pool, URL snapshot, concurrent singleflight, per-caller deadline, anonymous probes, healthy immediate, failed-route invalidation, runtime stale expiry, version watermark, consent")
 }
}
