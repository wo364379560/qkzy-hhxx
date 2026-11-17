import DOMPurify from 'dompurify';
import { isUrl } from './type-check';

const editorAllowedTags = ['br', 'span', 'img'];
const editorAllowedAttrs = ['class', 'id', 'src', 'alt', 'title', 'width', 'height', 'contenteditable'];

export function sanitizeUnsafeHtml(content: unknown): string {
  return DOMPurify.sanitize(String(content ?? ''), {
    ADD_ATTR: ['target'],
  });
}

export function sanitizeEditorHtml(content: unknown): string {
  return DOMPurify.sanitize(String(content ?? ''), {
    ALLOWED_TAGS: editorAllowedTags,
    ALLOWED_ATTR: editorAllowedAttrs,
  });
}

export function normalizeExternalUrl(url: unknown): string | undefined {
  if (typeof url !== 'string') {
    return undefined;
  }
  const trimmed = url.trim();
  if (!trimmed) {
    return undefined;
  }
  return isUrl(trimmed) ? trimmed : undefined;
}
