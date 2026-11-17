export class AvatarSyncError extends Error {
  constructor(readonly avatarURL: string, cause: any) {
    super(`图片已上传，资料同步未完成，请重试同步（无需重传）${cause?.code ? `，错误码 ${cause.code}` : ''}`);
    this.name = 'AvatarSyncError';
  }
}

export function assertAvatarActive(active: () => boolean) {
  if (!active()) throw new DOMException('账号或群聊已切换', 'AbortError');
}

export function assertProfileConfirmed(result: any) {
  if (result?.code !== undefined && Number(result.code) !== 0) throw new Error(`聊天资料同步未确认（${result.code}）`);
}
