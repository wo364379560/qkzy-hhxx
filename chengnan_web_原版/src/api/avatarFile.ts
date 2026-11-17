export async function prepareAvatar(file: File, signal?: AbortSignal): Promise<File> {
  if (!/^image\/(jpeg|png|webp)$/i.test(file.type)) throw new Error('请选择 JPG、PNG 或 WebP 头像');
  if (file.size > 20 * 1024 * 1024) throw new Error('头像原图不能超过 20MB');
  if (signal?.aborted) throw new DOMException('上传已取消', 'AbortError');
  const bitmap = typeof createImageBitmap === 'function'
    ? await createImageBitmap(file)
    : await new Promise<HTMLImageElement>((resolve, reject) => {
      const image = new Image();
      const url = URL.createObjectURL(file);
      const cleanup = () => { URL.revokeObjectURL(url); signal?.removeEventListener('abort', abort); };
      const abort = () => { image.src = ''; cleanup(); reject(new DOMException('上传已取消', 'AbortError')); };
      image.onload = () => { cleanup(); resolve(image); };
      image.onerror = () => { cleanup(); reject(new Error('头像图片无法读取')); };
      signal?.addEventListener('abort', abort, { once: true });
      image.src = url;
    });
  try {
    if (!bitmap.width || !bitmap.height) throw new Error('头像图片无效');
    const scale = Math.min(1, 1024 / Math.max(bitmap.width, bitmap.height));
    const canvas = document.createElement('canvas');
    canvas.width = Math.max(1, Math.round(bitmap.width * scale));
    canvas.height = Math.max(1, Math.round(bitmap.height * scale));
    const context = canvas.getContext('2d');
    if (!context) throw new Error('当前浏览器无法处理头像');
    context.fillStyle = '#fff';
    context.fillRect(0, 0, canvas.width, canvas.height);
    context.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
    const blob = await new Promise<Blob>((resolve, reject) => canvas.toBlob(value => value ? resolve(value) : reject(new Error('头像压缩失败')), 'image/jpeg', .85));
    if (signal?.aborted) throw new DOMException('上传已取消', 'AbortError');
    return new File([blob], 'avatar.jpg', { type: 'image/jpeg' });
  } finally { if ('close' in bitmap) bitmap.close(); }
}
