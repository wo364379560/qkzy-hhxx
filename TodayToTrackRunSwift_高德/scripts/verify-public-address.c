#include "RuiPublicAddress.h"
#include <assert.h>
#include <stdio.h>

int main(void) {
    const char *allowed[] = {"8.8.8.8", "1.1.1.1", "2001:4860:4860::8888", "2606:4700:4700::1111", "64:ff9b::808:808", "::ffff:8.8.8.8"};
    const char *blocked[] = {"0.0.0.0", "127.0.0.1", "10.0.0.1", "100.64.0.1", "169.254.1.1", "172.16.0.1", "192.168.1.1", "192.0.0.1", "192.0.2.1", "198.18.0.1", "198.51.100.1", "203.0.113.1", "224.0.0.1", "255.255.255.255", "::", "::1", "fc00::1", "fe80::1", "ff02::1", "2001:db8::1", "2001::1", "::ffff:127.0.0.1", "64:ff9b::7f00:1", "edge.example", ""};
    for (unsigned i = 0; i < sizeof allowed / sizeof *allowed; i++) assert(RuiPublicDNSLiteral(allowed[i]));
    for (unsigned i = 0; i < sizeof blocked / sizeof *blocked; i++) assert(!RuiPublicDNSLiteral(blocked[i]));
    assert(!RuiPublicDNSLiteral(NULL));
    puts("PASS: actual iOS C address filter; public IPv4/IPv6/NAT64 and reserved/private negative cases");
}
