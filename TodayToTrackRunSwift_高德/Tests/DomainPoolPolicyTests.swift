import Foundation
@main struct DomainPoolPolicyTests {
 static func main() throws {
  let six = ["https://api.prfgsu.com", "https://api.uktjwg.cn", "https://api1.prfgsu.com", "https://api1.uktjwg.cn", "https://cdnapi1.sazuts.cn", "https://cdnapi2.sazuts.cn"]
  let pool = LinLanDomainPoolDocument(version:4,ttlSeconds:600,origins:six)
  let encoded = try JSONEncoder().encode(pool)
  precondition(LinLanDomainPoolPolicy.parseDocument(encoded) == pool)
  precondition(!pool.origins.contains("https://api.ruizhihe.com"))
  for bad in ["http://api.prfgsu.com", "https://attacker.example", "https://api.prfgsu.com@attacker.example", "https://api.prfgsu.com:444", "https://api.prfgsu.com/path"] {
   let invalid=LinLanDomainPoolDocument(version:4,ttlSeconds:600,origins:six+[bad])
   let badData = try JSONEncoder().encode(invalid)
   precondition(LinLanDomainPoolPolicy.parseDocument(badData) == nil)
  }
  precondition(!LinLanDomainPoolPolicy.accepts(current:pool,next:LinLanDomainPoolDocument(version:3,ttlSeconds:600,origins:six)))
  precondition(LinLanDomainPoolPolicy.accepts(current:pool,next:pool))
  precondition(!LinLanDomainPoolPolicy.accepts(current:pool,next:LinLanDomainPoolDocument(version:4,ttlSeconds:600,origins:Array(six.reversed()))))
  let candidates=LinLanDomainPoolPolicy.recoveryCandidates(six,preferred:six[0])
  precondition(candidates.count==6 && Set(candidates)==Set(six))
  precondition(Array(candidates.prefix(3)) == [six[0],six[1],six[4]])
  precondition(LinLanDomainPoolPolicy.cacheUsable(fetchedAt:100,now:100+600+86400,ttl:600))
  precondition(!LinLanDomainPoolPolicy.cacheUsable(fetchedAt:100,now:100+600+86401,ttl:600))
  precondition(!LinLanDomainPoolPolicy.cacheUsable(fetchedAt:101,now:100,ttl:600))
  print("PASS: v4 six origins, strict trust, version, candidate reachability, bounded cache")
 }
}
