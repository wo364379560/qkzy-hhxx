#import <Foundation/Foundation.h>

NS_ASSUME_NONNULL_BEGIN

@interface RuiHTTPDNSNetwork : NSObject
+ (void)prepareAfterConsent;
+ (NSURLSession *)sessionWithRequestTimeout:(NSTimeInterval)requestTimeout
                           resourceTimeout:(NSTimeInterval)resourceTimeout
    NS_SWIFT_NAME(session(requestTimeout:resourceTimeout:));
@end

NS_ASSUME_NONNULL_END
