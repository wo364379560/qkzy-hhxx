package security

const defaultAuthVersion = 1

// NormalizeAuthVersion returns the persisted auth version or the default one.
func NormalizeAuthVersion(version int) int {
	if version > 0 {
		return version
	}
	return defaultAuthVersion
}

// NextAuthVersion returns the next auth version for invalidating old JWTs.
func NextAuthVersion(version int) int {
	return NormalizeAuthVersion(version) + 1
}

// MatchesAuthVersion reports whether a token version still matches the account.
func MatchesAuthVersion(tokenVersion, currentVersion int) bool {
	return NormalizeAuthVersion(tokenVersion) == NormalizeAuthVersion(currentVersion)
}
