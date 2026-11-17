#import "RuiHTTPDNSNetwork.h"
#import "RuiPublicAddress.h"
#import <AlicloudHTTPDNS/AlicloudHttpDNS.h>
#import <EMASCurl/EMASCurl.h>

static NSArray<NSString *> *RuiHTTPDNSHosts(void) {
    return @[@"api.prfgsu.com", @"api1.prfgsu.com", @"api.uktjwg.cn", @"api1.uktjwg.cn"];
}

@interface RuiHTTPDNSResolver : NSObject <EMASCurlProtocolDNSResolver>
@end

@implementation RuiHTTPDNSResolver
+ (HttpDnsService *)serviceAfterConsent {
    // Do not initialize the SDK or its network observers before existing consent.
    if (![[NSUserDefaults standardUserDefaults] boolForKey:@"linlan.prefs.privacy_accepted"]) return nil;
    static HttpDnsService *service;
    @synchronized(self) {
        if (service) return service;
        NSURL *url = [[NSBundle mainBundle] URLForResource:@"RuiHTTPDNS.private" withExtension:@"plist"];
        NSDictionary *config = url ? [NSDictionary dictionaryWithContentsOfURL:url] : nil;
        NSString *account = [config[@"accountId"] description];
        NSString *secret = config[@"secretKey"];
        if (account.integerValue <= 0 || ![secret isKindOfClass:NSString.class] || secret.length == 0) return nil;
        service = [[HttpDnsService alloc] initWithAccountID:account.integerValue secretKey:secret];
        [service setHTTPSRequestEnabled:YES];
        [service setPersistentCacheIPEnabled:YES];
        [service setReuseExpiredIPEnabled:NO];
        [service setDegradeToLocalDNSEnabled:NO];
        [service setNetworkingTimeoutInterval:2.0];
        [service setPreResolveAfterNetworkChanged:YES];
        [service setLogEnabled:NO];
        [service setPreResolveHosts:RuiHTTPDNSHosts() byIPType:HttpdnsQueryIPTypeBoth];
        return service;
    }
}

+ (NSString *)resolveDomain:(NSString *)domain {
    if (![RuiHTTPDNSHosts() containsObject:domain.lowercaseString]) return nil;
    HttpDnsService *service = [self serviceAfterConsent];
    if (!service) return nil;
    HttpdnsRequest *request = [[HttpdnsRequest alloc] initWithHost:domain queryIpType:HttpdnsQueryIPTypeAuto];
    request.resolveTimeoutInSecond = 2.0;
    // EMASCurl calls this on its worker; first-install misses wait for a signed result.
    HttpdnsResult *result = [service resolveHostSync:request];
    NSMutableOrderedSet<NSString *> *ips = [NSMutableOrderedSet orderedSet];
    NSTimeInterval now = NSDate.date.timeIntervalSince1970;
    NSArray<NSString *> *v4 = now < result.lastUpdatedTimeInterval + result.ttl ? result.ips : @[];
    NSArray<NSString *> *v6 = now < result.v6LastUpdatedTimeInterval + result.v6ttl ? result.ipv6s : @[];
    for (NSUInteger i = 0; i < MAX(v4.count, v6.count); i++) {
        if (i < v4.count && RuiPublicDNSLiteral(v4[i].UTF8String)) [ips addObject:v4[i]];
        if (i < v6.count && RuiPublicDNSLiteral(v6[i].UTF8String)) [ips addObject:v6[i]];
    }
    return ips.count ? [ips.array componentsJoinedByString:@","] : nil;
}
@end

@interface RuiAPIRedirectPolicy : NSObject <NSURLSessionTaskDelegate>
@end
@implementation RuiAPIRedirectPolicy
- (void)URLSession:(NSURLSession *)session task:(NSURLSessionTask *)task
    willPerformHTTPRedirection:(NSHTTPURLResponse *)response newRequest:(NSURLRequest *)request
    completionHandler:(void (^)(NSURLRequest * _Nullable))completionHandler {
    completionHandler(nil);
}
@end

@implementation RuiHTTPDNSNetwork
+ (void)prepareAfterConsent { [RuiHTTPDNSResolver serviceAfterConsent]; }
+ (NSURLSession *)sessionWithRequestTimeout:(NSTimeInterval)requestTimeout resourceTimeout:(NSTimeInterval)resourceTimeout {
    NSURLSessionConfiguration *config = NSURLSessionConfiguration.ephemeralSessionConfiguration;
    config.timeoutIntervalForRequest = requestTimeout;
    config.timeoutIntervalForResource = resourceTimeout;
    config.requestCachePolicy = NSURLRequestReloadIgnoringLocalCacheData;
    config.URLCache = nil;
    EMASCurlConfiguration *curl = [EMASCurlConfiguration defaultConfiguration];
    curl.dnsResolver = RuiHTTPDNSResolver.class;
    curl.domainWhiteList = RuiHTTPDNSHosts();
    curl.certificateValidationEnabled = YES;
    curl.domainNameVerificationEnabled = YES;
    curl.cacheEnabled = NO;
    curl.enableBuiltInRedirection = NO;
    curl.connectTimeoutInterval = 3.0;
    // Preserve system proxy/PAC behavior using the native session when present.
    curl.disabledWhenUsingSystemProxy = YES;
    [EMASCurlProtocol installIntoSessionConfiguration:config withConfiguration:curl];
    return [NSURLSession sessionWithConfiguration:config delegate:[RuiAPIRedirectPolicy new] delegateQueue:nil];
}
@end
