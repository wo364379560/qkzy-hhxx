import { reactive } from "../../../adapter-vue";

export type LargeAttachmentUploadTaskStatus =
  | "preparing"
  | "uploading"
  | "sending"
  | "failed";

export interface LargeAttachmentUploadTask {
  id: string;
  conversationID: string;
  fileName: string;
  fileSize: number;
  progress: number;
  status: LargeAttachmentUploadTaskStatus;
  createdAt: number;
  errorMessage?: string;
  cancel?: () => void;
  retry?: () => void;
}

interface CreateLargeAttachmentUploadTaskOptions {
  conversationID: string;
  file: File;
  cancel?: () => void;
  retry?: () => void;
}

const uploadTasks = reactive<LargeAttachmentUploadTask[]>([]);

function clampProgress(progress: number): number {
  if (!Number.isFinite(progress)) {
    return 0;
  }
  return Math.max(0, Math.min(1, progress));
}

function createTaskID(): string {
  return `large-attachment-${Date.now()}-${Math.random()
    .toString(16)
    .slice(2)}`;
}

export function createLargeAttachmentUploadTask(
  options: CreateLargeAttachmentUploadTaskOptions
): LargeAttachmentUploadTask {
  const task: LargeAttachmentUploadTask = {
    id: createTaskID(),
    conversationID: options.conversationID,
    fileName: options.file.name || "文件",
    fileSize: Number(options.file.size || 0),
    progress: 0,
    status: "preparing",
    createdAt: Date.now(),
    cancel: options.cancel,
    retry: options.retry,
  };
  uploadTasks.push(task);
  return task;
}

export function getLargeAttachmentUploadTasks(conversationID: string) {
  return uploadTasks.filter((task) => task.conversationID === conversationID);
}

export function updateLargeAttachmentUploadTask(
  taskID: string,
  patch: Partial<
    Pick<
      LargeAttachmentUploadTask,
      "progress" | "status" | "errorMessage" | "cancel" | "retry"
    >
  >
) {
  const task = uploadTasks.find((item) => item.id === taskID);
  if (!task) {
    return;
  }
  if (typeof patch.progress === "number") {
    task.progress = clampProgress(patch.progress);
  }
  if (patch.status) {
    task.status = patch.status;
  }
  if ("errorMessage" in patch) {
    task.errorMessage = patch.errorMessage;
  }
  if ("cancel" in patch) {
    task.cancel = patch.cancel;
  }
  if ("retry" in patch) {
    task.retry = patch.retry;
  }
}

export function failLargeAttachmentUploadTask(
  taskID: string,
  errorMessage = "上传失败"
) {
  updateLargeAttachmentUploadTask(taskID, {
    status: "failed",
    errorMessage,
  });
}

export function removeLargeAttachmentUploadTask(taskID: string) {
  const index = uploadTasks.findIndex((task) => task.id === taskID);
  if (index >= 0) {
    uploadTasks.splice(index, 1);
  }
}

export function cancelLargeAttachmentUploadTask(taskID: string) {
  const task = uploadTasks.find((item) => item.id === taskID);
  if (!task) {
    return;
  }
  if (task.status === "failed") {
    removeLargeAttachmentUploadTask(taskID);
    return;
  }
  task.cancel?.();
}

export function retryLargeAttachmentUploadTask(taskID: string) {
  const task = uploadTasks.find((item) => item.id === taskID);
  const retry = task?.retry;
  removeLargeAttachmentUploadTask(taskID);
  retry?.();
}
