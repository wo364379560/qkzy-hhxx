import { CHAT_API_BASE_URL, SDK_APP_ID } from "@/config/runtime";
import { initForwardJob, putForwardTargets, putForwardMessages, commitForwardJob, readForwardStatus, readForwardJob, retryForwardJob, cancelForwardJob } from "../../../../api";
import { saveForwardPlan, loadForwardPlan, deleteForwardPlan } from "./forwardPlanStore";
import type { ForwardBatch } from "./forwardBatch";

const wait = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
const storageKey = (owner: string) => `rui-forward:${encodeURIComponent(CHAT_API_BASE_URL)}:${SDK_APP_ID}:${owner}`;
export function rememberedForwardJob(owner: string): string {
  try {
    const prefix = `${storageKey(owner)}:job:`;
    const keys = Object.keys(localStorage).filter(key => key.startsWith(prefix));
    return keys.sort((a,b) => Number(localStorage.getItem(b)) - Number(localStorage.getItem(a)))[0]?.slice(prefix.length) || '';
  } catch { return ''; }
}
export function forgetForwardJob(owner: string, id?: string) {
  if (!id) return;
  try { localStorage.removeItem(`${storageKey(owner)}:job:${id}`); } catch { /* disabled storage */ }
}
function remember(owner: string, id: string) { localStorage.setItem(`${storageKey(owner)}:job:${id}`, String(Date.now())); }
const requestID = () => crypto.randomUUID().replace(/-/g, "");
const invalid = () => { throw new Error("任务进度响应无效"); };
const integer = (n: unknown): n is number => Number.isSafeInteger(n) && Number(n) >= 0;
const activeStates = ["pending", "preparing", "sending", "retrying"];
const statuses = [...activeStates,"success","failed","unknown","cancelled","blocked"];

// SDK payloads and REST MsgContent use different media enums/field names.
// Source: https://web.sdk.qcloud.com/im/doc/v3/zh-cn/Message.html
// REST: https://cloud.tencent.com/document/product/269/2720
const mediaID = (value: unknown): value is string => typeof value === "string" && value.trim().length > 0 && !/[\r\n\x00]/.test(value);
const optionalString = (value: unknown) => value === undefined || typeof value === "string";
const mediaURL = (value: unknown): value is string => {
  if (typeof value !== "string" || /[\s\x00]/.test(value)) return false;
  try {
    const parsed = new URL(value);
    return ["https:", "http:"].includes(parsed.protocol) && !!parsed.hostname && !parsed.username && !parsed.password;
  } catch { return false; }
};

// Unsupported/incomplete records retain SDK handling. Never invent a media UUID
// from a message ID or manufacture missing size/duration/format information.
export function toForwardMessage(message: any) {
  if (!message || typeof message !== "object" || Array.isArray(message)) return null;
  // SDK payload/type expose only the first element of a composite message.
  // Keep that whole message on the SDK path instead of dropping later elements.
  const singleElement = (elements: any) => Array.isArray(elements) && elements.length === 1 && elements[0] && typeof elements[0] === "object" && !Array.isArray(elements[0]);
  try {
    if ("getElements" in message && (typeof message.getElements !== "function" || !singleElement(message.getElements()))) return null;
    if ("_elements" in message && !singleElement(message._elements)) return null;
  } catch { return null; }
  const p = message?.payload;
  if (!p || typeof p !== "object" || Array.isArray(p)) return null;
  let content: any, preview: string;
  switch (message?.type) {
    case "TIMTextElem":
      if (typeof p.text !== "string" || !p.text.trim()) return null;
      content = { Text: p.text }; preview = p.text; break;
    case "TIMFaceElem":
      if ((p.index !== undefined && !integer(p.index)) || !optionalString(p.data) || (p.index === undefined && !p.data)) return null;
      content = { Index: p.index, Data: p.data }; preview = "[表情]"; break;
    case "TIMCustomElem": {
      if (typeof p.data !== "string" || !optionalString(p.description) || !optionalString(p.extension)) return null;
      content = { Data: p.data, Desc: p.description, Ext: p.extension }; preview = "[自定义消息]";
      try {
        const card = JSON.parse(p.data);
        if (card?.businessID === "zhihuan_contact_card" && card.version === 1 && mediaID(card.userId) && typeof card.nickname === "string") preview = `[名片] ${card.nickname}`;
      } catch { /* Opaque custom data is valid and is preserved verbatim. */ }
      break;
    }
    case "TIMLocationElem":
      if (typeof p.description !== "string" || !Number.isFinite(p.latitude) || Math.abs(p.latitude) > 90 || !Number.isFinite(p.longitude) || Math.abs(p.longitude) > 180) return null;
      content = { Desc: p.description, Latitude: p.latitude, Longitude: p.longitude }; preview = `[位置] ${p.description}`; break;
    case "TIMImageElem": {
      if (!mediaID(p.uuid) || ![1, 2, 3, 4, 255].includes(p.imageFormat) || !Array.isArray(p.imageInfoArray) || !p.imageInfoArray.length || p.imageInfoArray.length > 3) return null;
      const types = new Set<number>();
      const imageInfo: any[] = [];
      for (const i of p.imageInfoArray) {
        // SDK: 0 original, 1 198px thumbnail, 2 720px large. REST: 1, 3, 2.
        if (!i || !integer(i.type) || i.type > 2 || types.has(i.type) || !integer(i.size) || !integer(i.width) || !integer(i.height) || !mediaURL(i.url)) return null;
        types.add(i.type);
        imageInfo.push({ Type: [1, 3, 2][i.type], Size: i.size, Width: i.width, Height: i.height, URL: i.url });
      }
      // REST/SDK decoders expect original, large, thumbnail order. Do not infer
      // the type from the input array position, which may itself be reordered.
      imageInfo.sort((a, b) => a.Type - b.Type);
      content = { UUID: p.uuid, ImageFormat: p.imageFormat, ImageInfoArray: imageInfo }; preview = "[图片]"; break;
    }
    case "TIMFileElem":
      if (!mediaID(p.uuid) || !mediaURL(p.fileUrl) || !integer(p.fileSize) || typeof p.fileName !== "string" || !p.fileName.trim()) return null;
      content = { UUID: p.uuid, Url: p.fileUrl, FileSize: p.fileSize, FileName: p.fileName, Download_Flag: 2 }; preview = `[文件] ${p.fileName}`; break;
    case "TIMSoundElem":
      if (!mediaID(p.uuid) || !mediaURL(p.url) || !integer(p.size) || !integer(p.second)) return null;
      content = { UUID: p.uuid, Url: p.url, Size: p.size, Second: p.second, Download_Flag: 2 }; preview = "[语音]"; break;
    case "TIMVideoFileElem": {
      const thumbURL = p.thumbUrl ?? p.snapshotUrl;
      const thumbSize = p.thumbSize ?? p.snapshotSize;
      const thumbWidth = p.thumbWidth ?? p.snapshotWidth;
      const thumbHeight = p.thumbHeight ?? p.snapshotHeight;
      const thumbFormat = p.thumbFormat ?? p.snapshotFormat;
      // thumbUUID is the actual SDK resource identity; snapshot aliases only
      // cover metadata/URL. A synthetic `${message.ID}-thumb` is not that ID.
      if (!mediaID(p.videoUUID) || !mediaID(p.thumbUUID) || !mediaURL(p.videoUrl) || !mediaURL(thumbURL) || !integer(p.videoSize) || !integer(p.videoSecond) || !integer(thumbSize) || !integer(thumbWidth) || !integer(thumbHeight) || !optionalString(p.videoFormat) || !optionalString(thumbFormat)) return null;
      content = { VideoUUID: p.videoUUID, VideoUrl: p.videoUrl, VideoSize: p.videoSize, VideoSecond: p.videoSecond, VideoFormat: p.videoFormat, VideoDownloadFlag: 2, ThumbUUID: p.thumbUUID, ThumbUrl: thumbURL, ThumbSize: thumbSize, ThumbWidth: thumbWidth, ThumbHeight: thumbHeight, ThumbFormat: thumbFormat, ThumbDownloadFlag: 2 }; preview = "[视频]"; break;
    }
    default: return null;
  }
  const body = [{ MsgType: message.type, MsgContent: content }];
  if (new TextEncoder().encode(JSON.stringify(body)).length > 10000) return null;
  return { body, preview: preview.slice(0, 120) };
}

export function supportsServerMessages(messages: any[], merge: boolean) {
  return !merge && messages.length > 0 && messages.every(message => toForwardMessage(message) !== null);
}
export function supportsServerForward(batch: ForwardBatch) {
  return !!batch.serverPlan || !!batch.serverRequestID;
}

// Keep inputs once: never materialize the targets × messages product in Web.
export function createServerForwardBatch(conversations: any[], messages: any[], getName: (c: any) => string): ForwardBatch {
  const targets = [...new Map(conversations.map(c => [c.conversationID,c])).values()].map(c => ({conversation_id:c.conversationID, name:getName(c).slice(0,128)}));
  const mapped = messages.map(toForwardMessage);
  if (!targets.length || !mapped.length || mapped.some(m => !m) || !Number.isSafeInteger(targets.length * mapped.length)) throw new Error("转发任务内容无效");
  return {targets:[],merge:false,running:false,serverRequestID:requestID(),serverPlan:{targets,messages:mapped},
    serverState:{request_id:"",total:targets.length*mapped.length,success:0,failed:0,unknown:0,pending:targets.length*mapped.length,retryable_count:0,running:false,state:"uploading",items:[],cursor:"0",page_index:0,cursors:["0"]}};
}

export function applyServerStatus(batch: ForwardBatch, status: any) {
  if (status?.state === "completed") status = {...status,state:"done"};
  if (!status || status.request_id !== batch.serverRequestID || !integer(status.total) || !integer(status.success) || status.success > status.total || typeof status.running !== "boolean" || !["uploading","queued","done","cancelled"].includes(status.state)) invalid();
  for (const key of ["failed","unknown","pending","retryable_count","uploaded_targets","uploaded_messages","target_count","message_count"]) if (status[key] !== undefined && !integer(status[key])) invalid();
  const prior = batch.serverState || {items:[],cursor:"0",page_index:0,cursors:["0"]};
  batch.serverState = {...prior, ...status, items:prior.items};
  batch.serverAccepted = status.state !== "uploading";
}

export function applyServerProgress(batch: ForwardBatch, page: any) {
  if (!page || page.request_id !== batch.serverRequestID || page.page_size !== 200 || !Array.isArray(page.items) || page.items.length > 200 || !integer(page.total) || typeof page.running !== "boolean") invalid();
  const keys = new Set<string>();
  for (const row of page.items) {
    if (typeof row.conversation_id !== "string" || !/^(GROUP.+|C2C.+)$/.test(row.conversation_id) || !integer(row.index) || !integer(row.attempts) || !statuses.includes(row.status) || (integer(batch.serverState?.message_count) && batch.serverState.message_count > 0 && row.index >= batch.serverState.message_count)) invalid();
    const key = JSON.stringify([row.conversation_id,row.index]);
    if (keys.has(key)) invalid();
    keys.add(key);
  }
  if (page.next_cursor && (typeof page.next_cursor !== "string" || !/^[1-9][0-9]{0,19}$/.test(page.next_cursor))) invalid();
  batch.serverState = {...batch.serverState,items:page.items,next_cursor:page.next_cursor || ""};
}

const pageVersions = new WeakMap<ForwardBatch, number>();
export async function loadServerProgressPage(batch: ForwardBatch, active: () => boolean, direction = 0) {
  if (!active() || !batch.serverRequestID) return;
  const state = batch.serverState || {items:[],cursor:"0",page_index:0,cursors:["0"]};
  const cursors = [...state.cursors], pageIndex = state.page_index + direction;
  if (pageIndex < 0 || direction > 0 && !state.next_cursor) return;
  if (direction > 0) cursors[pageIndex] = state.next_cursor;
  const cursor = cursors[pageIndex] || "0";
  const version = (pageVersions.get(batch) || 0)+1;
  pageVersions.set(batch,version);
  const page = await readForwardJob(batch.serverRequestID,cursor);
  if (!active() || pageVersions.get(batch)!==version) return;
  applyServerProgress(batch,page);
  batch.serverState = {...batch.serverState,cursor,page_index:pageIndex,cursors};
}

async function digestPlan(plan: any) {
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(JSON.stringify({targets:plan.targets,messages:plan.messages})));
  return Array.from(new Uint8Array(digest), b => b.toString(16).padStart(2,"0")).join("");
}

// Every retried mutation keeps the same parent/chunk/retry identity. Never switch
// to SDK dispatch when a response or a partially accepted submission is lost.
async function idempotentCall(operation: () => Promise<any>, active: () => boolean) {
  for (let attempt=0; active(); attempt++) {
    try { return await operation(); }
    catch (error: any) {
      if ([400,401,403,404,409,413].includes(error?.status) || attempt >= 5) throw error;
      await wait(Math.min(10000,500*2**attempt));
    }
  }
  return null;
}

async function submitPlan(batch: ForwardBatch, owner: string, active: () => boolean, change: () => void) {
  const id = batch.serverRequestID!;
  let plan = batch.serverPlan;
  if (plan) {
    plan.content_hash ||= await digestPlan(plan);
    if (!active()) return;
    await saveForwardPlan(owner,id,plan);
    if (!active()) return;
    remember(owner,id);
  } else {
    plan = await loadForwardPlan(owner,id).catch(() => undefined);
    if (!active()) return;
    if (!plan) {
      // A committed task never depends on browser source data. If initial POST
      // was never received, the durable local plan instead resumes the same ID.
      const status = await readForwardStatus(id);
      if (!active()) return;
      applyServerStatus(batch,status);
      if (status.state !== "uploading") return;
      throw new Error("此任务尚未提交完整，请在原浏览器恢复提交；未提交的内容不会发送");
    }
  }
  const status = await idempotentCall(() => initForwardJob({request_id:id,target_count:plan.targets.length,message_count:plan.messages.length,content_hash:plan.content_hash}),active);
  if (!active()) return;
  applyServerStatus(batch,status);
  if (status.state === "uploading") {
    let targets = status.uploaded_targets, messages = status.uploaded_messages;
    if (!integer(targets) || !integer(messages) || targets > plan.targets.length || messages > plan.messages.length) invalid();
    while (targets < plan.targets.length && active()) {
      const part = plan.targets.slice(targets,targets+200), offset = targets;
      await idempotentCall(() => putForwardTargets(id,{offset,targets:part}),active);
      if (!active()) return;
      targets += part.length;
      batch.serverNotice = `正在提交目标 ${targets}/${plan.targets.length}；提交完成前请保持网页打开`;
      change();
    }
    while (messages < plan.messages.length && active()) {
      const part = plan.messages.slice(messages,messages+100), offset=messages;
      await idempotentCall(() => putForwardMessages(id,{offset,messages:part}),active);
      if (!active()) return;
      messages += part.length;
      batch.serverNotice = `正在提交消息 ${messages}/${plan.messages.length}；提交完成前请保持网页打开`;
      change();
    }
    if (!active()) return;
    const committed = await idempotentCall(() => commitForwardJob(id),active);
    if (!active()) return;
    applyServerStatus(batch,committed);
    if (committed.state === "uploading") throw new Error("任务还未提交完整，请点击继续提交");
  }
  batch.serverPlan = undefined;
  // Deletion failure must not turn an accepted job into a new submission.
  await deleteForwardPlan(owner,id).catch(() => {});
}

export async function runServerForward(batch: ForwardBatch, owner: string, active: () => boolean, change: () => void) {
  if (batch.running || !active() || !batch.serverRequestID) return;
  batch.running = true;
  try {
    try { await submitPlan(batch,owner,active,change); }
    catch (error: any) {
      if (!active()) return;
      batch.serverNotice = error?.status===404 ? "服务端尚未更新父任务接口，请先更新服务端，再继续提交原任务" : `${error?.message || "提交状态暂时无法确认"}；请继续查询或提交原任务，不要重新转发`;
      return;
    }
    const retryKey = `${storageKey(owner)}:retry:${batch.serverRequestID}`;
    if (localStorage.getItem(retryKey)) {
      try { await retryServerForward(batch,owner,active); }
      catch { if (active()) { batch.serverNotice = "上次补发请求的结果仍待确认，请稍后查询或重试原任务"; return; } }
    }
    while (active()) {
      try {
        const status = await readForwardStatus(batch.serverRequestID);
        if (!active()) return;
        applyServerStatus(batch,status);
        await loadServerProgressPage(batch,active);
        if (!active()) return;
        batch.serverNotice = status.state === "uploading" ? "任务尚未提交完整，请继续提交；未提交的内容不会发送" : "任务已交给服务器，关闭网页后仍会继续发送；这里只显示当前页结果";
        change();
        if (!status.running) return;
        await wait(1500);
      } catch (error: any) {
        if (!active()) return;
        batch.serverNotice = [401,403,404].includes(error?.status) ? "暂时无法查询原任务，请确认登录和服务端版本后再查询；不要重新转发" : "进度查询暂时失败，已接受的服务器任务继续执行，正在恢复查询";
        change();
        if ([401,403,404].includes(error?.status)) return;
        await wait(3000);
      }
    }
  } finally { batch.running=false; if (active()) change(); }
}

export async function retryServerForward(batch: ForwardBatch, owner: string, active: () => boolean) {
  if (!batch.serverRequestID || !active()) return;
  const key = `${storageKey(owner)}:retry:${batch.serverRequestID}`;
  const id = localStorage.getItem(key) || requestID();
  localStorage.setItem(key,id); // Preserve across a lost acknowledgement/refresh.
  const status = await idempotentCall(() => retryForwardJob(batch.serverRequestID!,id),active);
  if (!active()) return;
  applyServerStatus(batch,status);
  localStorage.removeItem(key);
}

export async function cancelServerForwardTask(batch: ForwardBatch, owner: string, active: () => boolean) {
  if (!batch.serverRequestID || !active()) return;
  const status = await idempotentCall(() => cancelForwardJob(batch.serverRequestID!),active);
  if (!active()) return;
  applyServerStatus(batch,status);
  batch.serverPlan=undefined;
  await deleteForwardPlan(owner,batch.serverRequestID).catch(() => {});
  await loadServerProgressPage(batch,active);
}
