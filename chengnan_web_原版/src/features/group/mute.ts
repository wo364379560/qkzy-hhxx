/** Both SDK seconds and historical millisecond timestamps are accepted. */
export function isIndividuallyMuted(muteUntil: unknown, now = Date.now()) {
  const until = Number(muteUntil || 0);
  return Number.isFinite(until) && (until > 100_000_000_000 ? until : until * 1000) > now;
}

export function excludeMutedMembers<T extends { userID?: string; muteUntil?: number | string }>(
  candidates: T[], mutedMembers: T[], now = Date.now(),
) {
  const mutedIDs = new Set(mutedMembers.filter(item => isIndividuallyMuted(item.muteUntil, now)).map(item => item.userID));
  return candidates.filter(item => !mutedIDs.has(item.userID) && !isIndividuallyMuted(item.muteUntil, now));
}
