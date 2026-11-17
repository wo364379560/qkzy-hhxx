import TUIChatEngine, {
  TUIChatService,
  TUITranslateService,
  IConversationModel,
  SendMessageParams,
} from "@tencentcloud/chat-uikit-engine-lite";
import {
  getChatAttachmentUploadSettings,
  isUploadCanceledError,
  uploadChatAttachment,
} from "@/api";
import { Toast, TOAST_TYPE } from "../../common/Toast/index";
import { isEnabledMessageReadReceiptGlobal } from "./utils";
import { isVideoFile, validateAttachmentFileSize } from "./fileLimits";
import { buildChatAttachmentText } from "./chatAttachmentMessage";
import OfflinePushInfoManager, {
  IOfflinePushInfoCreateParams,
} from "../offlinePushInfoManager/index";
import {
  createLargeAttachmentUploadTask,
  failLargeAttachmentUploadTask,
  removeLargeAttachmentUploadTask,
  updateLargeAttachmentUploadTask,
} from "./largeAttachmentUploadTasks";

function clampProgress(progress: number): number {
  if (!Number.isFinite(progress)) {
    return 0;
  }
  return Math.max(0, Math.min(1, progress));
}

function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return "文件上传失败，请稍后重试";
}

export function buildLargeVideoText(file: File, fileID: string): string {
  return buildLargeAttachmentText(file, fileID, true);
}

export function buildLargeAttachmentText(
  file: File,
  fileID: string,
  isVideo = false,
  contentType?: string,
  fileName?: string
): string {
  const fallbackName = isVideo ? TUITranslateService.t("TUIChat.视频") : "文件";
  const resolvedFileName = fileName || file?.name || fallbackName;
  return buildChatAttachmentText({
    fileID,
    name: resolvedFileName,
    size: Number(file?.size || 0),
    isVideo,
    contentType: contentType || file?.type || "application/octet-stream",
  });
}

export async function uploadAndSendLargeVideoMessage(
  currentConversation: IConversationModel,
  file: File
): Promise<void> {
  await uploadAndSendLargeAttachmentMessage(currentConversation, file, {
    isVideo: true,
  });
}

export function showLargeVideoSendError(): void {
  showLargeAttachmentSendError();
}

export async function uploadAndSendLargeAttachmentMessage(
  currentConversation: IConversationModel,
  file: File,
  uploadOptions: { isVideo?: boolean; isActive?: () => boolean } = {}
): Promise<void> {
  const conversationID = String(
    (currentConversation as any)?.conversationID || ""
  );
  const owner = String(TUIChatEngine.chat?.getLoginUser() || '');
  const isActive = () => !!owner && String(TUIChatEngine.chat?.getLoginUser()) === owner && (uploadOptions.isActive?.() ?? true);
  if (!isActive()) throw Object.assign(new Error('账号已切换，附件未发送'), { notDispatched: true });
  if (!conversationID) {
    Toast({
      message: "会话信息不完整，无法上传文件",
      type: TOAST_TYPE.ERROR,
    });
    return;
  }

  const settings = await getChatAttachmentUploadSettings();
  if (!settings.enabled) {
    Toast({
      message: "文件上传功能已关闭",
      type: TOAST_TYPE.ERROR,
    });
    return;
  }
  if (!validateAttachmentFileSize(file, settings.max_size_mb || 300)) {
    return;
  }

  const abortController =
    typeof AbortController !== "undefined" ? new AbortController() : undefined;
  const task = createLargeAttachmentUploadTask({
    conversationID,
    file,
    cancel: abortController ? () => abortController.abort() : undefined,
    retry: () => {
      uploadAndSendLargeAttachmentMessage(
        currentConversation,
        file,
        uploadOptions
      ).catch(() => {
        showLargeAttachmentSendError();
      });
    },
  });

  try {
    updateLargeAttachmentUploadTask(task.id, {
      status: "uploading",
      progress: 0,
    });
    const forceVideo = uploadOptions.isVideo === true;
    const result = await uploadChatAttachment(file, conversationID, {
      isVideo: forceVideo,
      signal: abortController?.signal,
      onProgress: (progress) => {
        updateLargeAttachmentUploadTask(task.id, {
          status: "uploading",
          progress: clampProgress(progress) * 0.96,
        });
      },
    });
    updateLargeAttachmentUploadTask(task.id, {
      status: "sending",
      progress: 1,
      cancel: undefined,
    });
    const isVideo = forceVideo || result.is_video || isVideoFile(file);
    const payload = {
      text: buildLargeAttachmentText(
        file,
        result.file_id,
        isVideo,
        result.content_type,
        result.file_name
      ),
    };
    const options: SendMessageParams = {
      to:
        currentConversation?.groupProfile?.groupID ||
        currentConversation?.userProfile?.userID,
      conversationType: currentConversation?.type as any,
      payload,
      needReadReceipt: isEnabledMessageReadReceiptGlobal(),
    };
    const offlinePushInfoCreateParams: IOfflinePushInfoCreateParams = {
      conversation: currentConversation,
      payload,
      messageType: TUIChatEngine.TYPES.MSG_TEXT,
    };

    if (!isActive()) throw Object.assign(new Error('账号已切换，附件未发送'), { notDispatched: true });
    await TUIChatService.sendTextMessage(options, {
      offlinePushInfo: OfflinePushInfoManager.create(
        offlinePushInfoCreateParams
      ),
    });
    removeLargeAttachmentUploadTask(task.id);
  } catch (error) {
    if (isUploadCanceledError(error)) {
      removeLargeAttachmentUploadTask(task.id);
      throw Object.assign(error as Error, { notDispatched: true });
    }
    failLargeAttachmentUploadTask(task.id, resolveErrorMessage(error));
    throw error;
  }
}

export function showLargeAttachmentSendError(): void {
  Toast({
    message: "文件上传失败，请稍后重试",
    type: TOAST_TYPE.ERROR,
  });
}
