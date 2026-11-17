<template>
  <button class="shared-card" type="button" @click.stop="open">
    <span class="card-main"><Avatar :url="card.avatar" size="44px" /><strong>{{ card.name }}</strong><span>›</span></span>
    <span class="card-label">{{ card.kind === 'group' ? '群名片' : '个人名片' }}</span>
  </button>
  <Teleport to="body">
    <div v-if="visible" class="card-backdrop" @click.self="close" @mousedown.stop>
      <div class="card-dialog" role="dialog" aria-modal="true" aria-label="群名片" @click.stop>
        <button class="close" type="button" @click="close">关闭</button>
        <h3>{{ group?.name || card.name }}</h3>
        <p>群ID：{{ card.id }}</p><p v-if="group">{{ group.type }} · {{ group.memberCount }}位成员</p>
        <p v-if="loading">正在确认群资料和成员身份…</p>
        <p v-else-if="error" class="error">{{ error }}</p>
        <p v-else-if="pending">申请已提交，等待审批或成员状态确认</p>
        <p v-else-if="inviteOnly">此群需要已有成员邀请；话题需先加入所属社群</p>
        <p v-else-if="forbidden">该群禁止主动加入，请联系群主</p>
        <p v-else-if="group && !member">你尚未加入此群</p>
        <button v-if="group && !loading && !error && !member && !pending && !inviteOnly && !forbidden" :disabled="joining" @click="join">{{ joining ? '正在提交' : '申请加入群聊' }}</button>
        <button v-if="member" :disabled="loading" @click="openChat">打开群聊</button>
        <button :disabled="loading || joining" @click="load">刷新状态</button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, onBeforeUnmount, watch } from 'vue';
import TUIChatEngine, { TUIStore, StoreName, TUIGroupService, TUIConversationService, TUIUserService } from '@tencentcloud/chat-uikit-engine-lite';
import Avatar from '../../../common/Avatar/index.vue';
import { Toast, TOAST_TYPE } from '../../../common/Toast';
import type { SharedCard } from '../../utils/sharedCard';
import { readSharedGroupAccess, joinSharedGroupCard } from '../../utils/sharedCardAccess';

const props = defineProps<{ card: SharedCard }>();
const emit = defineEmits<{ (event: 'openContactInfo', profile: Record<string, any>): void }>();
const visible = ref(false), loading = ref(false), joining = ref(false), member = ref(false), pending = ref(false);
const group = ref<any>(null), error = ref('');
let generation = 0;
const owner = () => String(TUIStore.getData(StoreName.USER, 'userProfile')?.userID || '');
const inviteOnly = computed(() => ['Work', 'Private'].includes(group.value?.type) || props.card.id.includes('@TOPIC#'));
const forbidden = computed(() => ['DisableApply', 'ForbidJoin'].includes(group.value?.joinOption));
function close() { ++generation; visible.value = false; loading.value = false; joining.value = false; }
onBeforeUnmount(close); watch(() => props.card.id, () => { close(); group.value = null; member.value = false; pending.value = false; });
function operationError(stage: string, cause: any) {
  const code = Number(cause?.code || cause?.errorCode || 0);
  return `${stage}失败${code ? `（${code}）` : ''}：${code === 10007 ? '操作权限不足，请核对成员身份、角色和群类型' : code === 10010 ? '群不存在或已解散' : '请检查聊天连接后重试'}`;
}
async function load() {
  const epoch = ++generation, account = owner(), id = props.card.id, chat = TUIChatEngine.chat as any;
  const active = () => epoch === generation && account === owner() && id === props.card.id && chat === TUIChatEngine.chat;
  loading.value = true; error.value = ''; member.value = false;
  try {
    if (!account || !chat) throw new Error('聊天尚未就绪');
    const result = await readSharedGroupAccess(chat, id, active);
    if (!active()) return;
    group.value = result.group; member.value = result.member;
    const joined = result.member;
    if (joined) pending.value = false;
  } catch (cause) { if (active()) error.value = operationError('群资料/成员身份查询', cause); }
  finally { if (active()) loading.value = false; }
}
async function openChat() {
  if (!member.value) return;
  try { await TUIConversationService.switchConversation(`${TUIChatEngine.TYPES.CONV_GROUP}${props.card.id}`); close(); }
  catch (cause) { error.value = operationError('打开群聊', cause); }
}
async function join() {
  if (joining.value || pending.value || inviteOnly.value || forbidden.value) return;
  const account = owner(), id = props.card.id, epoch = generation;
  joining.value = true;
  try {
    const chat = TUIChatEngine.chat as any;
    const active = () => account === owner() && epoch === generation && id === props.card.id && chat === TUIChatEngine.chat;
    const result = await joinSharedGroupCard(chat, id, active, () => TUIGroupService.joinGroup({ groupID: id, applyMessage: '通过群名片申请加入' }));
    if (!active()) return;
    group.value = result.group; member.value = result.member; pending.value = result.pending;
    await (TUIGroupService as any).getGroupList?.();
    if (active() && member.value) await openChat();
  } catch (cause) { if (account === owner() && epoch === generation && id === props.card.id) error.value = operationError('申请入群', cause); }
  finally { if (account === owner() && epoch === generation && id === props.card.id) joining.value = false; }
}
async function open() {
  if (props.card.kind === 'group') { visible.value = true; await load(); if (member.value) await openChat(); return; }
  const account = owner(), id = props.card.id, epoch = ++generation;
  try {
    const result = await TUIUserService.getUserProfile({ userIDList: [id] });
    if (epoch !== generation || account !== owner() || id !== props.card.id) return;
    const profile = result?.data?.[0];
    if (!profile?.userID) throw new Error('用户资料不存在');
    emit('openContactInfo', { ...profile, profile: { nick: profile.nick, avatar: profile.avatar, selfSignature: profile.selfSignature || '' } });
  } catch (cause) { Toast({ message: operationError('个人名片查询', cause), type: TOAST_TYPE.ERROR }); }
}
</script>

<style scoped>
.shared-card{width:248px;max-width:100%;padding:0;border:1px solid #e2e8f0;border-radius:10px;background:#fff;text-align:left;color:#1e293b;cursor:pointer}.card-main{display:flex;gap:12px;align-items:center;padding:14px}.card-main strong{flex:1;overflow-wrap:anywhere}.card-label{display:block;border-top:1px solid #edf0f4;padding:8px 14px;color:#64748b;font-size:12px}.card-backdrop{position:fixed;inset:0;background:#0006;display:grid;place-items:center;z-index:10001}.card-dialog{width:min(420px,90vw);box-sizing:border-box;background:white;padding:24px;border-radius:14px;color:#1e293b;overflow-wrap:anywhere}.card-dialog button{margin:8px 8px 0 0;padding:8px 12px;cursor:pointer}.close{float:right}.error{color:#b91c1c}
</style>
