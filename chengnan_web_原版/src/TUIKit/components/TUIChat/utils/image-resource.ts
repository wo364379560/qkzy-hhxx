import type { IMessageModel } from "@tencentcloud/chat-uikit-engine-lite";

type ImageInfo = {
  url?: string;
};

const getImageInfoArray = (message?: IMessageModel): ImageInfo[] => {
  const imageInfoArray = (message?.payload as any)?.imageInfoArray;
  return Array.isArray(imageInfoArray) ? imageInfoArray : [];
};

const getFirstValidUrl = (items: Array<ImageInfo | undefined>) => {
  for (const item of items) {
    const url = item?.url?.trim?.() || "";
    if (url) {
      return url;
    }
  }
  return "";
};

export const getImagePreviewUrl = (message?: IMessageModel) => {
  const imageInfoArray = getImageInfoArray(message);
  return (
    getFirstValidUrl([
      imageInfoArray[1],
      imageInfoArray[2],
      imageInfoArray[0],
    ]) ||
    ((message?.payload as any)?.url ?? "")
  );
};

export const getImageOriginalUrl = (message?: IMessageModel) => {
  const imageInfoArray = getImageInfoArray(message);
  return (
    getFirstValidUrl([
      imageInfoArray[0],
      imageInfoArray[1],
      imageInfoArray[2],
    ]) ||
    ((message?.payload as any)?.url ?? "")
  );
};
