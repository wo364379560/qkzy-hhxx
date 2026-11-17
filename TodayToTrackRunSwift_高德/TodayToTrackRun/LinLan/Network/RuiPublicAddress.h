#pragma once
#include <stdbool.h>

// Pure address validation shared by the iOS transport and the host-side test.
bool RuiPublicDNSLiteral(const char *value);
