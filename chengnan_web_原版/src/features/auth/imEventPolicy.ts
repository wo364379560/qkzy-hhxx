/** Values are the installed Tencent lite-chat TYPES.KICKED_OUT_* constants. */
export function imKickAction(reason: string): 'renew-signature' | 'reauthenticate' | 'reconnect' {
  if (reason === 'userSigExpired') return 'renew-signature';
  if (['multipleAccount', 'multipleDevice', 'REST_API_Kick'].includes(reason)) return 'reauthenticate';
  return 'reconnect';
}
