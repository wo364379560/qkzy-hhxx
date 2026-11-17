import { Toast, TOAST_TYPE } from '../../common/Toast/index';

export const DEFAULT_MAX_ATTACHMENT_FILE_SIZE_MB = 300;
export const TENCENT_NATIVE_FILE_SIZE_MB = 100;
export const TENCENT_NATIVE_FILE_SIZE_BYTES = TENCENT_NATIVE_FILE_SIZE_MB * 1024 * 1024;
export const MAX_VIDEO_FILE_SIZE_MB = DEFAULT_MAX_ATTACHMENT_FILE_SIZE_MB;
export const MAX_VIDEO_FILE_SIZE_BYTES = MAX_VIDEO_FILE_SIZE_MB * 1024 * 1024;
export const TENCENT_NATIVE_VIDEO_FILE_SIZE_MB = TENCENT_NATIVE_FILE_SIZE_MB;
export const TENCENT_NATIVE_VIDEO_FILE_SIZE_BYTES = TENCENT_NATIVE_FILE_SIZE_BYTES;

const VIDEO_FILE_EXTENSION_REGEXP = /\.(mp4|mov|m4v|avi|mkv|webm|3gp|3gpp|flv|wmv)$/i;

function toArray(files: any): any[] {
  if (!files) {
    return [];
  }

  if (Array.isArray(files)) {
    return files;
  }

  if (typeof files.length === 'number') {
    return Array.from(files);
  }

  return [files];
}

export function collectSelectedFiles(input: any): any[] {
  if (!input) {
    return [];
  }

  const files: any[] = [];

  if (input.files) {
    files.push(...toArray(input.files));
  } else {
    files.push(...toArray(input));
  }

  if (input.file) {
    files.push(input.file);
  }
  if (input.tempFile) {
    files.push(input.tempFile);
  }
  if (input.tempFiles) {
    files.push(...toArray(input.tempFiles));
  }

  return files.filter(Boolean);
}

export function isVideoFile(file: any, forceVideo = false): boolean {
  if (forceVideo) {
    return true;
  }

  const type = String(file?.type || file?.fileType || '').toLowerCase();
  const name = String(file?.name || file?.fileName || file?.path || file?.tempFilePath || '');

  return type.startsWith('video/') || VIDEO_FILE_EXTENSION_REGEXP.test(name);
}

export function getSelectedVideoFile(input: any): File | undefined {
  const selectedFiles = collectSelectedFiles(input);
  const forceVideo = input?.dataset?.type === 'video';
  return selectedFiles.find(file => isVideoFile(file, forceVideo)) as File | undefined;
}

export function getSelectedAttachmentFile(input: any): File | undefined {
  const selectedFiles = collectSelectedFiles(input);
  return selectedFiles[0] as File | undefined;
}

export function shouldUseLargeAttachmentUpload(input: any): boolean {
  const selectedFile = getSelectedAttachmentFile(input);
  return Number(selectedFile?.size || 0) > TENCENT_NATIVE_FILE_SIZE_BYTES;
}

export function validateAttachmentFileSize(file: File, maxSizeMB = DEFAULT_MAX_ATTACHMENT_FILE_SIZE_MB): boolean {
  if (Number(file?.size || 0) <= maxSizeMB * 1024 * 1024) {
    return true;
  }

  Toast({
    message: `文件大小不能超过${maxSizeMB}MB`,
    type: TOAST_TYPE.ERROR,
  });
  return false;
}

export function shouldUseLargeVideoUpload(input: any): boolean {
  return shouldUseLargeAttachmentUpload(input);
}

export function validateVideoFileSize(input: any): boolean {
  const selectedFiles = collectSelectedFiles(input);
  const forceVideo = input?.dataset?.type === 'video';
  const selectedVideo = selectedFiles.find(file => isVideoFile(file, forceVideo)) as File | undefined;

  return selectedVideo ? validateAttachmentFileSize(selectedVideo) : true;
}
