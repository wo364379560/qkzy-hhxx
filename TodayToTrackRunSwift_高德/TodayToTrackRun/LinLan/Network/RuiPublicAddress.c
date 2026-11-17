#include "RuiPublicAddress.h"
#include <arpa/inet.h>
#include <string.h>

static bool publicIPv4(const unsigned char *b) {
    return b[0] != 0 && b[0] != 10 && b[0] != 127 && b[0] < 224 &&
        !(b[0] == 100 && b[1] >= 64 && b[1] <= 127) && !(b[0] == 169 && b[1] == 254) &&
        !(b[0] == 172 && b[1] >= 16 && b[1] <= 31) &&
        !(b[0] == 192 && (b[1] == 168 || (b[1] == 0 && (b[2] == 0 || b[2] == 2)))) &&
        !(b[0] == 198 && (b[1] == 18 || b[1] == 19 || (b[1] == 51 && b[2] == 100))) &&
        !(b[0] == 203 && b[1] == 0 && b[2] == 113);
}

bool RuiPublicDNSLiteral(const char *value) {
    if (!value) return false;
    struct in_addr v4;
    if (inet_pton(AF_INET, value, &v4) == 1) return publicIPv4((const unsigned char *)&v4);
    struct in6_addr v6;
    if (inet_pton(AF_INET6, value, &v6) != 1) return false;
    if (IN6_IS_ADDR_V4MAPPED(&v6)) return publicIPv4(&v6.s6_addr[12]);
    static const unsigned char nat64[12] = {0, 0x64, 0xff, 0x9b, 0, 0, 0, 0, 0, 0, 0, 0};
    if (memcmp(v6.s6_addr, nat64, sizeof nat64) == 0) return publicIPv4(&v6.s6_addr[12]);
    const unsigned char *b = v6.s6_addr;
    return (b[0] & 0xe0) == 0x20 &&
        !(b[0] == 0x20 && b[1] == 1 && ((b[2] == 0x0d && b[3] == 0xb8) || b[2] < 2));
}
