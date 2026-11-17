export const CHAT_ATTACHMENT_MARKER_PREFIX = "CN_ATTACHMENT_V1:";

export interface ChatAttachmentInfo {
  fileID: string;
  name: string;
  size: number;
  isVideo: boolean;
  contentType: string;
}

export function formatChatAttachmentSize(size: number): string {
  if (!Number.isFinite(size) || size <= 0) {
    return "";
  }
  const kb = 1024;
  const mb = kb * 1024;
  const gb = mb * 1024;
  if (size >= gb) {
    return `${(size / gb).toFixed(size >= 10 * gb ? 0 : 1)}GB`;
  }
  if (size >= mb) {
    return `${(size / mb).toFixed(size >= 10 * mb ? 0 : 1)}MB`;
  }
  return `${(size / kb).toFixed(size >= 10 * kb ? 0 : 1)}KB`;
}

export function buildChatAttachmentText(
  attachment: ChatAttachmentInfo
): string {
  const title = attachment.isVideo
    ? "视频文件已上传，点击卡片预览或下载"
    : "文件已上传，点击卡片预览或下载";
  const sizeText = formatChatAttachmentSize(attachment.size);
  const summary = sizeText
    ? `${attachment.name} (${sizeText})`
    : attachment.name;
  const payload = {
    file_id: attachment.fileID,
    name: attachment.name,
    size: attachment.size,
    is_video: attachment.isVideo,
    content_type: attachment.contentType || "application/octet-stream",
  };
  return `${title}\n${summary}\n${CHAT_ATTACHMENT_MARKER_PREFIX}${JSON.stringify(
    payload
  )}`;
}

export function parseChatAttachmentText(
  input: unknown
): ChatAttachmentInfo | null {
  const text = extractPlainText(input);
  if (!text.includes(CHAT_ATTACHMENT_MARKER_PREFIX)) {
    return null;
  }
  const lines = text.split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed.startsWith(CHAT_ATTACHMENT_MARKER_PREFIX)) {
      continue;
    }
    const raw = trimmed.slice(CHAT_ATTACHMENT_MARKER_PREFIX.length).trim();
    if (!raw) {
      return null;
    }
    try {
      const parsed = JSON.parse(raw) as Record<string, unknown>;
      const fileID = String(parsed.file_id || "").trim();
      const name = String(parsed.name || "").trim();
      if (!fileID || !name) {
        return null;
      }
      const sizeValue = Number(parsed.size || 0);
      return {
        fileID,
        name,
        size: Number.isFinite(sizeValue) ? sizeValue : 0,
        isVideo: parsed.is_video === true,
        contentType: String(parsed.content_type || "application/octet-stream"),
      };
    } catch (_) {
      return null;
    }
  }
  return null;
}

export function parseChatAttachmentCustomPayload(
  input: unknown
): ChatAttachmentInfo | null {
  const textAttachment = parseChatAttachmentText(input);
  if (textAttachment) {
    return textAttachment;
  }

  const root = normalizeRecord(input);
  if (!root) {
    return null;
  }

  const candidates: Record<string, unknown>[] = [root];
  for (const key of ["attachment", "file", "payload", "data", "customData"]) {
    const nested = normalizeRecord(root[key]);
    if (nested) {
      candidates.push(nested);
    }
  }

  for (const candidate of candidates) {
    const attachment = attachmentFromRecord(candidate);
    if (attachment) {
      return attachment;
    }
  }
  return null;
}

function attachmentFromRecord(
  record: Record<string, unknown>
): ChatAttachmentInfo | null {
  const fileID = stringValue(record, ["file_id", "fileID", "fileId"]);
  const name = stringValue(record, ["name", "file_name", "fileName"]);
  if (!fileID || !name) {
    return null;
  }
  const contentType = stringValue(
    record,
    ["content_type", "contentType", "mime_type", "mimeType"],
    "application/octet-stream"
  );
  return {
    fileID,
    name,
    size: numberValue(record, ["size", "file_size", "fileSize"]),
    isVideo:
      booleanValue(record, ["is_video", "isVideo"]) ||
      contentType.toLowerCase().startsWith("video/"),
    contentType,
  };
}

function normalizeRecord(input: unknown): Record<string, unknown> | null {
  if (input && typeof input === "object" && !Array.isArray(input)) {
    return input as Record<string, unknown>;
  }
  if (typeof input === "string" && input.trim()) {
    try {
      const parsed = JSON.parse(input) as unknown;
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        return parsed as Record<string, unknown>;
      }
    } catch (_) {
      return null;
    }
  }
  return null;
}

function stringValue(
  record: Record<string, unknown>,
  keys: string[],
  fallback = ""
): string {
  for (const key of keys) {
    const value = record[key];
    if (value === undefined || value === null) {
      continue;
    }
    const text = String(value).trim();
    if (text) {
      return text;
    }
  }
  return fallback;
}

function numberValue(record: Record<string, unknown>, keys: string[]): number {
  for (const key of keys) {
    const value = Number(record[key] || 0);
    if (Number.isFinite(value) && value > 0) {
      return value;
    }
  }
  return 0;
}

function booleanValue(
  record: Record<string, unknown>,
  keys: string[]
): boolean {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "boolean") {
      return value;
    }
    if (typeof value === "number") {
      return value !== 0;
    }
    const text = String(value || "")
      .trim()
      .toLowerCase();
    if (text === "true" || text === "1") {
      return true;
    }
  }
  return false;
}

function extractPlainText(input: unknown): string {
  if (typeof input === "string") {
    return input;
  }
  if (Array.isArray(input)) {
    return input
      .map((item) => {
        if (typeof item === "string") {
          return item;
        }
        if (item && typeof item === "object" && "text" in item) {
          return String((item as { text?: unknown }).text || "");
        }
        return "";
      })
      .join("");
  }
  if (input && typeof input === "object" && "text" in input) {
    return extractPlainText((input as { text?: unknown }).text);
  }
  return "";
}
