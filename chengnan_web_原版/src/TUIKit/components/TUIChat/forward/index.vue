<template>
  <Overlay :visible="isShowForwardPanel" :useMask="false">
    <Transfer
      v-if="!forwardBatch"
      :title="TUITranslateService.t('TUIChat.转发')"
      :isSearch="true"
      :isCustomItem="false"
      :list="filteredConversationList"
      :searchPlaceholder="TUITranslateService.t('TUIChat.搜索会话')"
      :emptyText="emptyText"
      :isHiddenBackIcon="isUniFrameWork"
      :submitLoading="isForwarding"
      :submitDisabled="isForwarding"
      @search="handleSearch"
      @cancel="closeForwardPanel"
      @submit="finishSelected"
    >
      <template #header-extra>
        <div class="forward-filter-tabs">
          <button
            v-for="item in filterTabs"
            :key="item.key"
            :class="[
              'forward-filter-tab',
              currentFilter === item.key && 'forward-filter-tab-active',
            ]"
            type="button"
            :disabled="isForwarding"
            @click="setFilter(item.key)"
          >
            {{ item.label }}
          </button>
        </div>
      </template>
    </Transfer>
    <section
      v-else
      class="forward-progress"
      role="dialog"
      aria-label="转发进度"
      aria-modal="true"
    >
      <h3>{{ isForwarding ? "正在转发" : "转发结果" }}</h3>
      <p v-if="forwardBatch.serverRequestID" class="forward-progress-warning">
        {{ forwardBatch.serverNotice || "正在提交服务器转发任务，请勿重复发送" }}
      </p>
      <p v-else-if="isForwarding" class="forward-progress-warning">
        正在按配置限速发送，临时限流时会自动减速、等待后继续。可收起窗口继续聊天，请保持网页打开。
      </p>
      <p v-if="forwardBatch.lastRun?.skippedSuccess" class="forward-progress-warning">
        本轮仅补发 {{ forwardBatch.lastRun.targetIDs.length }} 个会话、{{ forwardBatch.lastRun.messageCount }} 条消息；
        已成功的 {{ forwardBatch.lastRun.skippedSuccess }} 条不会重复发送。
      </p>
      <template v-if="forwardBatch.serverRequestID">
        <p>共 {{ forwardBatch.serverState?.total || 0 }} 条 · 已发送 {{ forwardedCount }} · 失败 {{ forwardBatch.serverState?.failed || 0 }} · 待处理 {{ forwardBatch.serverState?.pending || 0 }}</p>
        <div class="forward-progress-list">
          <div v-for="row in serverRows" :key="`${row.conversation_id}:${row.index}`" class="forward-progress-row">
            <strong>{{ row.name || row.conversation_id }} · 第 {{ row.index + 1 }} 条</strong>
            <span>{{ row.preview }} — {{ serverStatusName(row.status) }}</span>
            <span v-if="row.error" class="forward-progress-error">{{ row.error }}</span>
          </div>
          <p v-if="!serverRows.length">{{ forwardBatch.serverState?.running ? "正在排队，稍后显示结果" : "本页暂无结果" }}</p>
        </div>
        <div class="forward-progress-pages">
          <button type="button" :disabled="!forwardBatch.serverState?.page_index" @click="serverPage(-1)">上一页</button>
          <span>第 {{ (forwardBatch.serverState?.page_index || 0)+1 }} 页</span>
          <button type="button" :disabled="!forwardBatch.serverState?.next_cursor" @click="serverPage(1)">下一页</button>
        </div>
      </template>
      <div v-else class="forward-progress-list">
        <div
          v-for="row in visibleForwardProgress"
          :key="row.id"
          class="forward-progress-row"
        >
          <strong>{{ row.name }}<small v-if="row.skipped"> · 已完成，本轮跳过</small></strong>
          <span
            >已发送 {{ row.success }}/{{ row.total }} 条<span v-if="row.sending"
              >，发送中</span
            ><span v-if="row.retrying">，限流等待中，将自动继续</span
            ><span v-if="row.failed">，失败 {{ row.failed }} 条</span
            ><span v-if="row.unknown">，状态未知 {{ row.unknown }} 条</span
            ><span v-if="row.pending">，未发送 {{ row.pending }} 条</span></span
          >
          <details class="forward-item-results">
            <summary>逐条结果</summary>
            <div v-for="(item, index) in row.items" :key="index">
              {{ index + 1 }}. {{ item.preview }} — {{ serverStatusName(item.status) }}
            </div>
            <button v-if="row.total > (resultItemLimit[row.id] || 50)" type="button" @click="resultItemLimit[row.id] = (resultItemLimit[row.id] || 50) + 50">再显示50条</button>
          </details>
          <span v-if="row.error" class="forward-progress-error">{{
            row.error
          }}</span>
        </div>
      </div>
      <div v-if="!forwardBatch.serverRequestID && forwardProgress.length > progressPageSize" class="forward-progress-pages">
        <button type="button" :disabled="progressPage === 0" @click="progressPage--">上一页</button>
        <span>{{ progressPage + 1 }} / {{ progressPageCount }}</span>
        <button type="button" :disabled="progressPage + 1 >= progressPageCount" @click="progressPage++">下一页</button>
      </div>
      <p v-if="hasUnknownForwardResult" class="forward-progress-warning">
        部分消息可能已送达，请先核对目标会话。无法安全重试的消息将保留为状态未知。
      </p>
      <footer>
        <button v-if="forwardBatch.serverRequestID" type="button" @click="leaveServerResults">收起并继续选择消息</button>
        <button v-if="forwardBatch.serverAccepted && forwardBatch.serverState?.running" type="button" @click="cancelServerForward">停止未发送部分</button>
        <button v-if="forwardBatch.serverRequestID && !isForwarding" type="button" @click="refreshServerForward">{{ forwardBatch.serverState?.state === "uploading" ? "继续提交" : "查询进度" }}</button>
        <button type="button" @click="hideForwardResults">
          {{ isForwarding ? "后台转发" : "收起结果" }}
        </button>
        <button
          v-if="!isForwarding"
          :disabled="
            forwardBatch.targets.some(hasSettlingForwardItems)
          "
          type="button"
          @click="closeForwardPanel"
        >
          {{ forwardBatch.serverRequestID ? "结束查看" : "结束本次转发" }}
        </button>
        <button
          v-if="canRetryForward"
          type="button"
          :disabled="isForwarding"
          @click="retryForwardBatch"
        >
          {{ forwardBatch.serverRequestID ? "仅重试失败项" : "重试未完成部分" }}
        </button>
      </footer>
    </section>
  </Overlay>
  <button
    v-if="forwardBatch && !isShowForwardPanel"
    class="forward-background-status"
    type="button"
    @click="isShowForwardPanel = true"
  >
    {{ isForwarding ? "正在转发" : "查看转发结果" }} · 已发送
    {{ forwardedCount }} 条
  </button>
  <button class="forward-tasks-button" type="button" @click="openJobs()">转发任务</button>
  <Overlay :visible="showJobs" @onOverlayClick="showJobs = false">
    <section class="forward-progress" role="dialog" aria-label="服务端转发任务">
      <h3>转发任务</h3>
      <p v-if="jobsError">{{ jobsError }}</p>
      <button v-for="job in serverJobs" :key="job.request_id" type="button" @click="selectJob(job.request_id)">
        {{ job.request_id.slice(0, 12) }} · {{ job.state }} · 已发送 {{ job.success }}/{{ job.total }}
      </button>
      <p v-if="!serverJobs.length && !jobsError">暂无服务端转发任务</p>
      <button v-if="jobsCursor" type="button" @click="openJobs(jobsCursor)">下一页</button>
      <button type="button" @click="openJobs()">刷新</button>
      <button type="button" @click="showJobs = false">关闭</button>
    </section>
  </Overlay>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash";
import { resolveForwardTargets } from "./forwardTargets";
import { createCapturedForwardSender } from "./forwardDispatch";
import { supportsServerMessages, createServerForwardBatch, runServerForward, rememberedForwardJob, forgetForwardJob, loadServerProgressPage, retryServerForward, cancelServerForwardTask } from "./serverForward";
import { readForwardSettings, listForwardJobs } from "../../../../api";
import {
  computed,
  onMounted,
  onUnmounted,
  ref,
  shallowRef,
  triggerRef,
} from "../../../adapter-vue";
import TUIChatEngine, {
  Friend,
  IConversationModel,
  IGroupModel,
  IMessageModel,
  TUIStore,
  StoreName,
  TUIChatService,
  TUITranslateService,
} from "@tencentcloud/chat-uikit-engine-lite";
import Overlay from "../../common/Overlay/index.vue";
import Transfer from "../../common/Transfer/index.vue";
import { Toast, TOAST_TYPE } from "../../../components/common/Toast";
import { isUniFrameWork } from "../../../utils/env";
import { isEnabledMessageReadReceiptGlobal } from "../utils/utils";
import OfflinePushInfoManager, {
  IOfflinePushInfoCreateParams,
} from "../offlinePushInfoManager/index";
import type { ITransferListItem } from "../../../interface";
import {
  snapshotForwardMessages,
  createForwardBatch,
  runForwardBatch,
  canRetryForwardTarget,
  getForwardPendingWork,
  summarizeForwardTarget,
  isForwardRateLimited,
  configureForwardPacingPolicy,
  hasSettlingForwardItems,
  forwardTargetError,
  type ForwardBatch,
} from "./forwardBatch";

interface IEmits {
  (e: "toggleMultipleSelectMode", visible?: boolean): void;
}

const emits = defineEmits<IEmits>();

type ForwardFilter = "all" | "group" | "friend";

interface IForwardConversationItem extends ITransferListItem {
  conversationID: string;
  conversationType: string;
  searchName?: string;
  rawConversation?: any;
}

let selectedToForwardMessageIDList: string[] = [];
let isMergeForward = false;
let selectedMessageSnapshot: any[] = [];
let disposed = false;
let forwardOwner = "";
let viewGeneration = 0;
const showJobs = ref(false);
const serverJobs = ref<any[]>([]);
const jobsCursor = ref('');
const jobsError = ref('');
async function openJobs(cursor = '') {
  showJobs.value = true;
  jobsError.value = '';
  const owner = currentForwardUser();
  try {
    const page = await listForwardJobs(cursor);
    if (owner !== currentForwardUser() || disposed) return;
    serverJobs.value = page.items;
    jobsCursor.value = page.next_cursor;
  } catch { jobsError.value = '任务查询失败，请重试；当前任务仍保留'; }
}
function selectJob(id: string) {
  if (forwardBatch.value && !forwardBatch.value.serverRequestID && isForwarding.value) return;
  viewGeneration++;
  isForwarding.value = false;
  forwardOwner = currentForwardUser();
  forwardBatch.value = { merge:false, running:false, targets:[], serverRequestID:id, serverAccepted:true };
  showJobs.value = false;
  isShowForwardPanel.value = true;
  void refreshServerForward();
}
function leaveServerResults() {
  viewGeneration++;
  isForwarding.value = false;
  forwardBatch.value = undefined;
  isShowForwardPanel.value = false;
}

// SDK message objects must retain their identity when passed back to resendMessage.
const forwardBatch = shallowRef<ForwardBatch>();
const progressPageSize = 50;
const progressPage = ref(0);
const resultItemLimit = ref<Record<string, number>>({});
const forwardProgress = computed(() => (forwardBatch.value?.targets || []).map(target => ({target,...summarizeForwardTarget(target)})));
const progressPageCount = computed(() => Math.max(1, Math.ceil(forwardProgress.value.length / progressPageSize)));
const visibleForwardProgress = computed(() => forwardProgress.value.slice(progressPage.value * progressPageSize, (progressPage.value+1)*progressPageSize).map(({target,...summary}) => ({
  ...summary,id:target.conversation.conversationID,name:target.name,skipped:false,
  items:target.items.slice(0,resultItemLimit.value[target.conversation.conversationID] || 50).map(item => ({
    preview:item.messages.length>1 ? `[合并转发 ${item.messages.length} 条]` : String(item.messages[0]?.payload?.text || item.messages[0]?.payload?.fileName || "[非文字消息]").slice(0,80),
    status:item.status === "retrying" ? `等待自动重试（第 ${item.retryCount} 次）` : ({pending:"未发送",sending:"发送中",success:"已发送",failed:"失败",unknown:"结果待确认"} as any)[item.status],
  })),
  error:forwardTargetError(target) ? resolveForwardErrorMessage(forwardTargetError(target),target.name) : "",
})));
const forwardedCount = computed(() => forwardBatch.value?.serverRequestID ? forwardBatch.value.serverState?.success || 0 : forwardProgress.value.reduce((sum,row) => sum+row.success,0));
const canRetryForward = computed(() => {
  const batch=forwardBatch.value;
  if (batch?.serverRequestID) return (batch.serverState?.retryable_count || 0)>0;
  const targets=batch?.targets || [];
  return !targets.some(hasSettlingForwardItems) && targets.some(canRetryForwardTarget);
});
const hasUnknownForwardResult = computed(() => forwardBatch.value?.serverRequestID ? (forwardBatch.value.serverState?.unknown || 0)>0 : forwardProgress.value.some(row => row.unknown>0));
const serverRows = computed(() => forwardBatch.value?.serverState?.items || []);
const serverStatusName = (status: string) => ({pending:"未发送",preparing:"检查权限",sending:"发送中",retrying:"等待自动重试",success:"已发送",failed:"失败",blocked:"前序消息未完成",unknown:"结果待确认",cancelled:"已取消"} as any)[status] || status;
async function serverPage(direction: number) {
  const batch=forwardBatch.value,owner=forwardOwner;
  if (!batch?.serverRequestID) return;
  const active=()=> !disposed && forwardBatch.value===batch && currentForwardUser()===owner;
  try { await loadServerProgressPage(batch,active,direction); if(active())triggerRef(forwardBatch); }
  catch(error:any) { if(active())Toast({message:error.message || "进度加载失败",type:TOAST_TYPE.ERROR}); }
}

function currentForwardUser() {
  return String(TUIStore.getData(StoreName.USER, "userProfile")?.userID || "");
}

function onForwardUserUpdated() {
  if (forwardOwner && currentForwardUser() !== forwardOwner) {
    forwardBatch.value = undefined;
    selectedMessageSnapshot = [];
    forwardOwner = "";
    isShowForwardPanel.value = false;
    isForwarding.value = false;
  }
  resumeServerForward();
}

function resumeServerForward() {
  const owner = currentForwardUser();
  if (!owner || disposed || forwardBatch.value || isForwarding.value) return;
  const id = rememberedForwardJob(owner);
  if (!id) return;
  forwardOwner = owner;
  forwardBatch.value = { merge: false, running: false, targets: [], serverRequestID: id, serverAccepted: true };
  void refreshServerForward();
}

async function refreshServerForward() {
  const batch = forwardBatch.value, owner = forwardOwner;
  if (!batch?.serverRequestID || isForwarding.value) return;
  const generation = viewGeneration;
  const active = () => !disposed && generation === viewGeneration && currentForwardUser() === owner && forwardBatch.value === batch;
  if (!active()) return;
  isForwarding.value = true;
  try { await runServerForward(batch, owner, active, () => triggerRef(forwardBatch)); }
  finally { if (active()) isForwarding.value = false; }
}

async function cancelServerForward() {
  const batch=forwardBatch.value,owner=forwardOwner;
  if (!batch?.serverRequestID) return;
  const active=()=>!disposed && currentForwardUser()===owner && forwardBatch.value===batch;
  try { await cancelServerForwardTask(batch,owner,active); if(active())triggerRef(forwardBatch); }
  catch(error:any) { if(active())Toast({message:error.message || "取消失败，请重试",type:TOAST_TYPE.ERROR}); }
}

const isShowForwardPanel = ref(false);
const customConversationList = ref<IForwardConversationItem[]>([]);
const currentFilter = ref<ForwardFilter>("all");
const searchKeyword = ref("");
const isForwarding = ref(false);

const filterTabs = computed(() => [
  { key: "all" as const, label: TUITranslateService.t("TUIChat.全部") },
  { key: "group" as const, label: TUITranslateService.t("TUIChat.群聊") },
  { key: "friend" as const, label: TUITranslateService.t("TUIChat.好友") },
]);

const filteredConversationList = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase();

  return customConversationList.value.filter((item) => {
    const isGroup = isGroupConversation(item);
    const matchedFilter =
      currentFilter.value === "all" ||
      (currentFilter.value === "group" && isGroup) ||
      (currentFilter.value === "friend" && !isGroup);

    if (!matchedFilter) {
      return false;
    }

    if (!keyword) {
      return true;
    }

    return (item.searchName || "").toLowerCase().includes(keyword);
  });
});

const emptyText = computed(() =>
  searchKeyword.value.trim()
    ? TUITranslateService.t("TUIChat.无匹配结果")
    : TUITranslateService.t("TUIChat.暂无会话")
);

onMounted(() => {
  TUIStore.watch(StoreName.USER, { userProfile: onForwardUserUpdated });
  TUIStore.watch(StoreName.CUSTOM, {
    singleForwardMessageID: onSingleForwardMessageIDUpdated,
    multipleForwardMessageID: onMultipleForwardMessageIDUpdated,
  });
  resumeServerForward();
});

onUnmounted(() => {
  disposed = true;
  TUIStore.unwatch(StoreName.USER, { userProfile: onForwardUserUpdated });
  TUIStore.unwatch(StoreName.CUSTOM, {
    singleForwardMessageID: onSingleForwardMessageIDUpdated,
    multipleForwardMessageID: onMultipleForwardMessageIDUpdated,
  });

  // tuistore data must be cleared when closing the forward panel
  clearStoreData();
});

function onSingleForwardMessageIDUpdated(messageID: string | undefined) {
  // Store watchers emit their initial/cleared value too. Those are not new
  // forwarding commands: they must not open a restored task or discard it.
  if (!messageID || disposed) return;
  // Consume the command even when showing an existing job. Otherwise clicking
  // Forward on the same message after hiding the result emits no store change.
  TUIStore.update(StoreName.CUSTOM, "singleForwardMessageID", undefined);
  if (
    isForwarding.value ||
    (forwardBatch.value?.serverRequestID ? forwardBatch.value.serverState?.state === "uploading" || forwardBatch.value.serverState?.success !== forwardBatch.value.serverState?.total : forwardProgress.value.some(row => row.success !== row.total))
  ) {
    if (forwardBatch.value) isShowForwardPanel.value = true;
    return;
  }
  forwardBatch.value = undefined;
  isMergeForward = false;
  selectedToForwardMessageIDList = [messageID];
  openForwardPanel();
}

function onMultipleForwardMessageIDUpdated(
  params: { isMergeForward: boolean; messageIDList: string[] } | undefined
) {
  if (!params || disposed) return;
  TUIStore.update(StoreName.CUSTOM, "multipleForwardMessageID", undefined);
  if (
    isForwarding.value ||
    (forwardBatch.value?.serverRequestID ? forwardBatch.value.serverState?.state === "uploading" || forwardBatch.value.serverState?.success !== forwardBatch.value.serverState?.total : forwardProgress.value.some(row => row.success !== row.total))
  ) {
    if (forwardBatch.value) isShowForwardPanel.value = true;
    return;
  }
  forwardBatch.value = undefined;
  isMergeForward = false;
  const {
    isMergeForward: _isMergeForward,
    messageIDList: selectedMessageIDList,
  } = params || {};
  if (selectedMessageIDList?.length > 0) {
    isMergeForward = _isMergeForward;
    selectedToForwardMessageIDList = selectedMessageIDList;
    openForwardPanel();
  } else {
    Toast({
      message: TUITranslateService.t("TUIChat.未选择消息"),
      type: TOAST_TYPE.ERROR,
    });
  }
}

function clearStoreData() {
  TUIStore.update(StoreName.CUSTOM, "singleForwardMessageID", undefined);
  TUIStore.update(StoreName.CUSTOM, "multipleForwardMessageID", undefined);
}

function hideForwardResults() {
  isShowForwardPanel.value = false;
}

function closeForwardPanel(): void {
  if (
    isForwarding.value ||
    forwardBatch.value?.targets.some(hasSettlingForwardItems)
  )
    return;
  if (
    (forwardBatch.value?.serverRequestID ? forwardBatch.value.serverState?.state === "uploading" || forwardBatch.value.serverState?.success !== forwardBatch.value.serverState?.total : forwardProgress.value.some(row => row.success !== row.total)) &&
    !window.confirm(
      forwardBatch.value?.serverRequestID
        ? "关闭结果不会取消服务器任务；需要停止请先点击‘停止未发送部分’。请先核对未完成或结果未知的消息，确定结束查看？"
        : "仍有未完成或结果待确认的消息。结束后不会自动补发，请先核对目标会话。确定结束本次转发？"
    )
  )
    return;
  // tuistore data must be cleared when closing the forward panel
  clearStoreData();
  const hadBatch = !!forwardBatch.value;
  if (forwardBatch.value?.serverRequestID) forgetForwardJob(forwardOwner, forwardBatch.value.serverRequestID);
  forwardBatch.value = undefined;
  selectedMessageSnapshot = [];
  if (hadBatch) emits("toggleMultipleSelectMode", false);
  currentFilter.value = "all";
  searchKeyword.value = "";
  isShowForwardPanel.value = false;
}

function openForwardPanel(): void {
  if (isForwarding.value) {
    return;
  }
  try {
    selectedMessageSnapshot = snapshotForwardMessages(
      getStableForwardMessageQueue(selectedToForwardMessageIDList).map(
        (message) => message.ID
      ),
      (id) => TUIStore.getMessageModel(id)
    );
  } catch (error: any) {
    selectedMessageSnapshot = [];
    Toast({ message: error.message, type: TOAST_TYPE.ERROR });
    clearStoreData();
    return;
  }
  forwardOwner = currentForwardUser();
  currentFilter.value = "all";
  searchKeyword.value = "";
  getTransforRenderDataList();
  isShowForwardPanel.value = true;
}

function handleSearch(value: string): void {
  searchKeyword.value = value;
}

function setFilter(filter: ForwardFilter): void {
  currentFilter.value = filter;
}

async function finishSelected(
  selectedConversationItemList: IForwardConversationItem[]
): Promise<void> {
  if (
    isForwarding.value ||
    forwardBatch.value ||
    !selectedConversationItemList?.length
  ) {
    return;
  }

  const owner = forwardOwner;
  const generation = viewGeneration;
  isForwarding.value = true;
  try {
    const isActive = () =>
      !disposed && generation === viewGeneration &&
      !!owner &&
      currentForwardUser() === owner &&
      forwardOwner === owner;
    if (!isActive()) return;
    const shouldMerge = isMergeForward;
    const messages = [...selectedMessageSnapshot];
    if (!messages.length)
      throw new Error("部分原消息已失效，请返回聊天记录重新选择");
    const targets = [
      ...new Map(
        selectedConversationItemList.map((item) => [item.conversationID, item])
      ).values(),
    ];
    progressPage.value = 0;
    resultItemLimit.value = {};
    const selectedConversationList = await resolveSelectedConversationList(
      targets
    );
    if (!isActive()) return;
    const getName = (conversation: any) => String(conversation.groupProfile?.name || getForwardConversationName(conversation) || "").trim();
    if (!selectedConversationList.some((target: any) => target.resolutionError) && supportsServerMessages(messages,shouldMerge)) {
      forwardBatch.value = createServerForwardBatch(selectedConversationList,messages,getName);
      await executeForwardBatch(forwardBatch.value,isActive);
      return;
    }
    forwardBatch.value = createForwardBatch(
      selectedConversationList,
      messages,
      shouldMerge,
      (conversation) =>
        String(
          conversation.groupProfile?.name ||
            getForwardConversationName(conversation) ||
            ""
        ).trim()
    );
    await executeForwardBatch(forwardBatch.value, isActive);
  } catch (error: any) {
    if (!disposed && currentForwardUser() === owner && forwardOwner === owner)
      Toast({
        message: error?.message || "转发失败，请重试",
        type: TOAST_TYPE.ERROR,
      });
  } finally {
    if (!disposed && generation === viewGeneration && currentForwardUser() === owner && forwardOwner === owner) isForwarding.value = false;
  }
}

async function retryForwardBatch() {
  const batch = forwardBatch.value;
  if (!batch || isForwarding.value || !canRetryForward.value) return;
  if (!batch.serverRequestID && !getForwardPendingWork(batch).length) return;
  const owner = forwardOwner;
  const isActive = () =>
    !disposed &&
    forwardBatch.value === batch &&
    !!owner &&
    currentForwardUser() === owner &&
    forwardOwner === owner;
  if (!isActive()) return;
  isForwarding.value = true;
  try {
    if (batch.serverRequestID) await retryServerForward(batch,owner,isActive);
    await executeForwardBatch(batch, isActive);
  } catch(error:any) {
    if(isActive())Toast({message:error.message || "重试失败，请查询原任务后重试",type:TOAST_TYPE.ERROR});
  } finally {
    if (isActive()) isForwarding.value = false;
  }
}

async function executeForwardBatch(
  batch: ForwardBatch,
  isActive: () => boolean
) {
  if (batch.serverRequestID) {
    await runServerForward(batch, forwardOwner, isActive, () => triggerRef(forwardBatch));
    if (isActive() && batch.serverState?.state === "done" && batch.serverState.success === batch.serverState.total) {
      clearStoreData();
      selectedMessageSnapshot = [];
      emits("toggleMultipleSelectMode", false);
    }
    return;
  }
  // Forward using the captured SDK; the sender coalesces personal mute checks.
  // Friend/blacklist/global mute enforcement remains with IM.
  const sender = createCapturedForwardSender(
    TUIChatService.getEngine().chat,
    (messages, mode) => TUIChatService.updateMessageList(messages, mode),
    isActive,
    forwardOwner
  );
  // Server publishes the project's configured SDK command budgets. Older
  // servers retain conservative defaults; never derive a bucket from client IP.
  try {
    const settings = await readForwardSettings();
    if (!isActive()) return;
    configureForwardPacingPolicy({sendIntervalMs:settings.sdk_send_interval_ms,prepareIntervalMs:settings.sdk_prepare_interval_ms,concurrency:settings.sdk_concurrency});
  } catch { if (!isActive()) return; }
  await runForwardBatch(batch, {
    isActive,
    prepare: sender.prepare,
    onChange: () => {
      if (isActive()) triggerRef(forwardBatch);
    },
    resend: (message, item) => sender.resend(message, item),
    send: (target, item, merge) => {
      const offlinePushInfoCreateParams: IOfflinePushInfoCreateParams = {
        conversation: target.conversation,
        payload: item.messages[0]?.payload,
        messageType: merge
          ? TUIChatEngine.TYPES.MSG_MERGER
          : item.messages[0]?.type || TUIChatEngine.TYPES.MSG_TEXT,
      };
      if (!item.sendOptions) {
        item.sendOptions = cloneDeep({
          needMerge: merge,
          offlinePushInfo: OfflinePushInfoManager.create(
            offlinePushInfoCreateParams
          ),
          params: { needReadReceipt: isEnabledMessageReadReceiptGlobal() },
        });
      }
      return sender.send(target, item, merge);
    },
  });
  if (!isActive()) return;
  const results = batch.targets.map((target) => {
    const summary = summarizeForwardTarget(target);
    return {
      status: summary.success === summary.total ? "fulfilled" : "rejected",
      reason: forwardTargetError(target),
      ...summary,
    };
  });
  showForwardResultToast(
    results,
    batch.targets.map((target) => target.name)
  );
  if (results.every((result) => result.status === "fulfilled")) {
    clearStoreData();
    selectedMessageSnapshot = [];
    emits("toggleMultipleSelectMode", false);
  }
}

function resolveForwardErrorMessage(
  error: {
    code?: number;
    message?: string;
  },
  targetName = ""
) {
  if (
    Number(error.code) === 10010 ||
    /group.*(dissolved|not exist)/i.test(String(error.message || ""))
  ) {
    return TUITranslateService.t("TUIChat.该群聊已被解散");
  }
  const message = String(error.message || "");
  if (Number(error.code) === 2024 || /"code"\s*:\s*2024\b/.test(message)) {
    return "聊天服务尚未登录，请待连接恢复后重试";
  }
  if (
    Number(error.code) === 10017 ||
    /"code"\s*:\s*10017\b/.test(message) ||
    /you\s+have\s+been\s+forbidden\s+to\s+speak/i.test(message)
  ) {
    return targetName
      ? `您在群「${targetName}」中已被禁言，无法转发`
      : "您在目标群中已被禁言，无法转发";
  }
  if (error.code === 80001) {
    return TUITranslateService.t("TUIChat.内容包含敏感词汇");
  }
  if (
    [2800, 2801, 2805, 60008].includes(Number(error.code)) ||
    /60008|timeout|timed out/i.test(message)
  ) {
    return "网络波动，发送状态待确认，请先核对目标会话";
  }
  if (isForwardRateLimited(error)) {
    return "持续限流，已自动减速重试；未完成部分已保留，稍后可继续，成功消息不会重复发送";
  }
  return error.message || TUITranslateService.t("TUIChat.发送失败");
}

function showForwardResultToast(
  results: Array<{
    status: string;
    reason?: any;
    success?: number;
    total?: number;
    unknown?: number;
    pending?: number;
  }>,
  targetNames: string[] = []
): void {
  const failures = results
    .map((result, index) => ({ ...result, targetName: targetNames[index] }))
    .filter((result) => result.status === "rejected");
  const successCount = results.length - failures.length;
  const sentCount = results.reduce(
    (sum, result) => sum + (result.success || 0),
    0
  );
  const unknownCount = results.reduce(
    (sum, result) => sum + (result.unknown || 0),
    0
  );
  const pendingCount = results.reduce(
    (sum, result) => sum + (result.pending || 0),
    0
  );
  if (failures.length === 0) {
    Toast({
      message: `已成功转发到 ${successCount} 个会话`,
      type: TOAST_TYPE.SUCCESS,
    });
    return;
  }

  const reasonCounts = new Map<string, number>();
  failures.forEach((failure) => {
    const reason = resolveForwardErrorMessage(
      failure.reason || {},
      failure.targetName
    );
    reasonCounts.set(reason, (reasonCounts.get(reason) || 0) + 1);
  });
  const reasons = Array.from(reasonCounts.entries())
    .map(([reason, count]) => (count > 1 ? `${reason}（${count} 个）` : reason))
    .join("；");
  Toast({
    message: `转发结果：完整成功 ${successCount} 个会话，未完成 ${failures.length} 个会话。已发送 ${sentCount} 条，状态未知 ${unknownCount} 条，未发送 ${pendingCount} 条。${reasons}`,
    type:
      sentCount > 0 || successCount > 0 || unknownCount > 0
        ? TOAST_TYPE.WARNING
        : TOAST_TYPE.ERROR,
  });
}

async function resolveSelectedConversationList(selected: IForwardConversationItem[]) {
  return resolveForwardTargets(selected, item => ensureForwardConversationModel(
    item.rawConversation || TUIStore.getConversationModel(item.conversationID) || { conversationID:item.conversationID, type:item.conversationType }, item));
}

function ensureForwardConversationModel(
  conversation: any,
  item?: IForwardConversationItem
): IConversationModel | undefined {
  const conversationID = String(
    conversation?.conversationID || item?.conversationID || ""
  );
  if (!conversationID) {
    return undefined;
  }

  if (
    typeof conversation.getShowName === "function" &&
    typeof conversation.getAvatar === "function"
  ) {
    return conversation as IConversationModel;
  }

  return {
    ...conversation,
    conversationID,
    type: conversation?.type || item?.conversationType,
    getShowName: () => getForwardConversationName(conversation, item),
    getAvatar: () => getForwardConversationAvatar(conversation, item),
  } as IConversationModel;
}

function getForwardConversationName(
  conversation: any,
  item?: IForwardConversationItem
) {
  if (typeof conversation?.getShowName === "function") {
    return conversation.getShowName();
  }
  return (
    item?.nick ||
    conversation?.name ||
    conversation?.userProfile?.nick ||
    conversation?.userProfile?.userID ||
    conversation?.groupProfile?.name ||
    conversation?.groupProfile?.groupID ||
    conversation?.conversationID ||
    ""
  );
}

function getForwardConversationAvatar(
  conversation: any,
  item?: IForwardConversationItem
) {
  if (typeof conversation?.getAvatar === "function") {
    return conversation.getAvatar();
  }
  return (
    item?.avatar ||
    conversation?.avatar ||
    conversation?.userProfile?.avatar ||
    conversation?.groupProfile?.avatar ||
    ""
  );
}

function getTransforRenderDataList() {
  const conversationList = (TUIStore.getData(
    StoreName.CONV,
    "conversationList"
  ) || []) as IConversationModel[];
  const friendList = (TUIStore.getData(StoreName.FRIEND, "friendList") ||
    []) as Friend[];
  const groupList = (TUIStore.getData(StoreName.GRP, "groupList") ||
    []) as IGroupModel[];
  const conversationMap = new Map<string, IForwardConversationItem>();

  conversationList.forEach((conversation: any) => {
    const conversationID = String(conversation?.conversationID || "");
    if (!conversationID || conversationID.includes("@TIM#SYSTEM")) {
      return;
    }
    conversationMap.set(conversationID, {
      userID: conversationID,
      conversationID,
      nick: conversation.getShowName?.() || conversation.name || conversationID,
      avatar: conversation.getAvatar?.() || conversation.avatar || "",
      conversationType: conversation.type,
      searchName: getConversationSearchName(conversation),
      rawConversation: conversation,
    });
  });

  friendList.forEach((friend: Friend) => {
    const userID = String(friend?.userID || "");
    if (!userID) {
      return;
    }
    const conversationID = `${TUIChatEngine.TYPES.CONV_C2C}${userID}`;
    const currentItem = conversationMap.get(conversationID);
    const friendSearchName = normalizeSearchName(
      friend.remark || friend.profile?.nick
    );
    conversationMap.set(conversationID, {
      ...currentItem,
      userID: conversationID,
      conversationID,
      nick: friendSearchName || currentItem?.nick || friend.userID,
      avatar: friend.profile?.avatar || currentItem?.avatar || "",
      conversationType: TUIChatEngine.TYPES.CONV_C2C,
      searchName: friendSearchName || currentItem?.searchName || "",
      rawConversation: currentItem?.rawConversation,
    });
  });

  groupList.forEach((group: IGroupModel) => {
    const groupID = String(group?.groupID || "");
    if (!groupID) {
      return;
    }
    const conversationID = `${TUIChatEngine.TYPES.CONV_GROUP}${groupID}`;
    const currentItem = conversationMap.get(conversationID);
    const groupSearchName = normalizeSearchName(group.name);
    conversationMap.set(conversationID, {
      ...currentItem,
      userID: conversationID,
      conversationID,
      nick: groupSearchName || currentItem?.nick || group.groupID,
      avatar: group.avatar || currentItem?.avatar || "",
      conversationType: TUIChatEngine.TYPES.CONV_GROUP,
      searchName: groupSearchName || currentItem?.searchName || "",
      rawConversation: currentItem?.rawConversation,
    });
  });

  customConversationList.value = Array.from(conversationMap.values());
}

function normalizeSearchName(value: unknown): string {
  return String(value || "").trim();
}

function getConversationSearchName(conversation: any): string {
  if (conversation?.type === TUIChatEngine.TYPES.CONV_GROUP) {
    return normalizeSearchName(conversation?.groupProfile?.name);
  }

  if (conversation?.type === TUIChatEngine.TYPES.CONV_C2C) {
    return normalizeSearchName(conversation?.userProfile?.nick);
  }

  return "";
}

function isGroupConversation(item: IForwardConversationItem): boolean {
  if (item.conversationType === TUIChatEngine.TYPES.CONV_GROUP) {
    return true;
  }
  if (item.conversationType === TUIChatEngine.TYPES.CONV_C2C) {
    return false;
  }
  return item.conversationID.startsWith(TUIChatEngine.TYPES.CONV_GROUP);
}
function getStableForwardMessageQueue(
  messageIDList: string[]
): IMessageModel[] {
  const currentMessageList = (TUIStore.getData(StoreName.CHAT, "messageList") ||
    []) as IMessageModel[];
  const visibleOrderMap = new Map(
    currentMessageList.map((message, index) => [message.ID, index])
  );
  const selectedOrderMap = new Map(
    messageIDList.map((messageID, index) => [messageID, index])
  );

  const models = [...new Set(messageIDList)].map((messageID) =>
    TUIStore.getMessageModel(messageID)
  );
  if (
    models.some(
      (message) => !message?.ID || message.isRevoked || message.isDeleted
    )
  )
    throw new Error("部分原消息已失效，请重新选择");
  return (models as IMessageModel[]).sort((a, b) => {
    const visibleOrderDiff =
      getMessageVisibleOrder(a, visibleOrderMap) -
      getMessageVisibleOrder(b, visibleOrderMap);
    if (visibleOrderDiff !== 0) {
      return visibleOrderDiff;
    }

    const sequenceDiff =
      normalizeMessageSequence(a) - normalizeMessageSequence(b);
    if (sequenceDiff !== 0) {
      return sequenceDiff;
    }

    if (a.time !== b.time) {
      return a.time - b.time;
    }

    return (
      (selectedOrderMap.get(a.ID) ?? Number.MAX_SAFE_INTEGER) -
      (selectedOrderMap.get(b.ID) ?? Number.MAX_SAFE_INTEGER)
    );
  });
}

function getMessageVisibleOrder(
  message: IMessageModel,
  visibleOrderMap: Map<string, number>
) {
  return visibleOrderMap.get(message.ID) ?? Number.MAX_SAFE_INTEGER;
}

function normalizeMessageSequence(message: IMessageModel) {
  const sequence = Number(message.sequence || 0);
  return Number.isFinite(sequence) ? sequence : 0;
}
</script>

<style lang="scss" scoped>
.forward-background-status {
  position: fixed;
  top: 18px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 1000;
  border: 1px solid #dce7ff;
  border-radius: 6px;
  padding: 10px 14px;
  background: #f4f8ff;
  color: #245ac4;
  cursor: pointer;
}
.forward-progress {
  width: min(600px, calc(100vw - 32px));
  max-height: calc(100vh - 32px);
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  padding: 20px;
  background: #fff;
  border: 1px solid #ddd;
  border-radius: 8px;
  box-shadow: 0 8px 24px #0002;
  color: #333;
  font-size: 14px;
  h3 {
    margin: 0 0 16px;
    font-size: 18px;
  }
  footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding-top: 16px;
  }
  button {
    border: 1px solid #ddd;
    border-radius: 4px;
    background: #fff;
    padding: 8px 12px;
    cursor: pointer;
  }
  button:disabled {
    opacity: 0.5;
    cursor: default;
  }
}
.forward-progress-list {
  overflow-y: auto;
  min-height: 0;
}
.forward-progress-pages {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding-top: 12px;
}
.forward-progress-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 0;
  border-bottom: 1px solid #eee;
  overflow-wrap: anywhere;
}
.forward-progress-error {
  color: #b42318;
}
.forward-progress-warning {
  color: #8a5200;
  margin: 12px 0 0;
}
.forward-filter-tabs {
  display: flex;
  gap: 12px;
  padding-bottom: 12px;
}

.forward-filter-tab {
  border: none;
  border-radius: 16px;
  padding: 8px 16px;
  font-size: 14px;
  line-height: 20px;
  color: #111827;
  background: #f3f4f6;
  cursor: pointer;
}

.forward-filter-tab-active {
  color: #ffffff;
  background: #1d4ed8;
  font-weight: 600;
}
</style>

<style scoped>.forward-tasks-button{position:absolute;right:16px;top:60px;z-index:20;border:0;background:#f0f5ff;color:#235795;padding:6px 10px;border-radius:6px}</style>
