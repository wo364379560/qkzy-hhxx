<template>
  <div
    :class="['about-container', 'container', isH5 && 'container-h5']"
    @click="closeAboutBox"
    @mousedown.stop
  >
    <div
      :class="['about-box', 'box', isH5 && 'box-h5']"
      @click.stop
    >
      <header class="title">
        <div
          v-if="isH5"
          class="title-back"
          @click="closeAboutBox"
        >
          <Icon :file="backSVG" />
        </div>
        <div class="title-name">隐私与安全</div>
        <button
          v-if="!isH5"
          class="title-close"
          type="button"
          @click="closeAboutBox"
        >
          关闭
        </button>
      </header>

      <main class="main">
        <section class="hero-card">
          <img
            class="hero-card-logo"
            src="../assets/image/logo.svg"
            alt=""
          >
          <div class="hero-card-info">
            <div class="hero-card-name">{{ APP_NAME }}</div>
            <div class="hero-card-version">SDK版本 {{ SDKVersion }}</div>
            <div class="hero-card-tags">
              <span class="hero-card-tag">账号安全</span>
              <span class="hero-card-tag">防盗</span>
            </div>
          </div>
        </section>

        <section class="content-grid">
          <div class="security-column">
            <article class="panel-card panel-card-primary">
              <div class="panel-card-header">
                <div class="panel-card-label">登录保护</div>
                <div
                  class="panel-card-badge"
                  :class="totpStatus.enabled ? 'panel-card-badge-safe' : ''"
                >
                  {{ totpStatus.enabled ? '已开启' : '未开启' }}
                </div>
              </div>
              <div class="panel-card-title">Google 验证</div>
              <div class="panel-card-description">
                开启后，Web 登录需要输入验证器中的 6 位动态验证码。
              </div>
              <button
                v-if="!totpStatus.enabled"
                class="panel-card-action panel-card-action-primary"
                type="button"
                :disabled="totpLoading"
                @click="startTOTPSetup"
              >
                立即开启
              </button>
            </article>

            <article class="panel-card panel-card-danger">
              <div class="panel-card-header">
                <div class="panel-card-label">账号安全</div>
                <div class="panel-card-badge">高风险</div>
              </div>
              <div class="panel-card-title">注销账号</div>
              <div class="panel-card-description">
                注销后账号将不可恢复，请谨慎操作。
              </div>
              <button
                class="panel-card-action panel-card-action-danger"
                type="button"
                @click="toggleCancellation"
              >
                继续注销
              </button>
            </article>
          </div>

          <article class="legal-panel">
            <div class="legal-panel-header">
              <div class="panel-card-label">隐私协议</div>
            </div>
            <div class="legal-list">
              <button
                v-for="item in legalLinks"
                :key="item.title"
                class="legal-list-item"
                type="button"
                @click="openExternal(item.url)"
              >
                <span class="legal-list-title">{{ item.title }}</span>
                <span class="legal-list-arrow">›</span>
              </button>
            </div>
          </article>
        </section>
      </main>

      <div
        v-if="isCancellationBoxShow"
        :class="['cancellation-container', 'container', isH5 && 'container-h5']"
        @click="toggleCancellation"
      >
        <div
          :class="['cancellation-box', 'box', isH5 && 'box-h5']"
          @click.stop
        >
          <header
            v-if="isH5"
            class="title"
          >
            <div
              class="title-back"
              @click="toggleCancellation"
            >
              <Icon :file="backSVG" />
            </div>
            <div class="title-name">注销账号</div>
          </header>
          <main class="dialog-main">
            <img
              src="../assets/image/warn.svg"
              width="60"
              height="60"
            >
            <p class="dialog-main-text">
              注销后，您将无法使用当前账号，相关数据也将删除无法找回。
            </p>
            <p>
              <span>当前账号: </span>
              <span class="dialog-main-id">{{ props.userProfile.userID }}</span>
            </p>
          </main>
          <footer class="footer-actions">
            <button
              class="btn btn-error"
              @click.stop="submitCancellation"
            >
              注销
            </button>
            <button
              class="btn btn-default"
              @click.stop="toggleCancellation"
            >
              取消
            </button>
          </footer>
        </div>
      </div>

      <div
        v-if="totpDialog.visible"
        :class="['cancellation-container', 'container', isH5 && 'container-h5']"
        @click="closeTOTPDialog"
      >
        <div
          :class="['totp-box', 'box', isH5 && 'box-h5']"
          @click.stop
        >
          <header class="totp-dialog-head">
            <div class="totp-dialog-title">{{ totpDialog.title }}</div>
            <button class="title-close" type="button" @click="closeTOTPDialog">关闭</button>
          </header>
          <main class="totp-dialog-main">
            <template v-if="totpDialog.mode === 'setup'">
              <p class="totp-dialog-desc">
                请使用 Google Authenticator 或 Microsoft Authenticator 扫描二维码，然后填写 6 位验证码完成绑定。
              </p>
              <div class="totp-qrcode">
                <QrcodeVue :value="totpSetup.totpUri" :size="180" level="H" />
              </div>
              <div class="totp-secret-box">{{ totpSetup.totpSecret }}</div>
              <label class="totp-field">
                <span>验证码</span>
                <input
                  v-model.trim="totpForm.code"
                  type="text"
                  inputmode="numeric"
                  maxlength="6"
                  placeholder="请输入 6 位验证码"
                >
              </label>
              <button class="btn btn-primary" :disabled="totpLoading" @click="confirmTOTPBind">
                {{ totpLoading ? '开启中...' : '确认开启' }}
              </button>
            </template>

          </main>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive } from 'vue';
import { ref, withDefaults, defineProps, defineEmits } from '../TUIKit/adapter-vue';
import { TUIStore, StoreName } from '@tencentcloud/chat-uikit-engine-lite';
import { ElMessage } from 'element-plus';
import QrcodeVue from 'qrcode.vue';
import Icon from '../TUIKit/components/common/Icon.vue';
import backSVG from '../TUIKit/assets/icon/back.svg';
import { isH5 } from '../TUIKit/utils/env';
import { Link } from '../utils/link';
import { IUserProfile } from '../TUIKit/interface';
import router from '../router';
import { useAuthSession } from '@/features/auth/session';
import { APP_NAME } from '@/config/runtime';
import {
  bindUserTOTP,
  getUserTOTPStatus,
  setupUserTOTP,
} from '@/api';

const props = withDefaults(
  defineProps<{
    userProfile: IUserProfile;
  }>(),
  {
    userProfile: () => ({} as IUserProfile),
  },
);

const emits = defineEmits(['closeAboutBox']);
const authSession = useAuthSession();

const SDKVersion = TUIStore.getData(StoreName.APP, 'SDKVersion');
const isCancellationBoxShow = ref(false);
const totpLoading = ref(false);
const totpStatus = reactive({
  enabled: false,
  boundAt: '',
});
const totpSetup = reactive({
  tempToken: '',
  totpSecret: '',
  totpUri: '',
});
const totpForm = reactive({
  code: '',
});
const totpDialog = reactive({
  visible: false,
  mode: '' as '' | 'setup',
  title: '',
});

const legalLinks = [
  { title: '隐私政策', url: Link.privacy.url },
  { title: '用户协议', url: Link.agreement.url },
  { title: '免责声明', url: Link.statement.url },
  { title: '个人信息收集清单', url: Link.personalInfo.url },
  { title: '第三方信息共享清单', url: Link.thirdPartyInfo.url },
  { title: '未成年人使用条款', url: Link.minorTerms.url },
];

function closeAboutBox() {
  emits('closeAboutBox');
}

function openExternal(url: string) {
  window.open(url, '_blank', 'noopener,noreferrer');
}

function toggleCancellation() {
  isCancellationBoxShow.value = !isCancellationBoxShow.value;
}

async function loadTOTPStatus() {
  try {
    const status = await getUserTOTPStatus();
    totpStatus.enabled = status.enabled;
    totpStatus.boundAt = status.bound_at;
  } catch (error) {
    console.warn('load totp status failed', error);
  }
}

function openTOTPDialog(mode: typeof totpDialog.mode, title: string) {
  totpDialog.mode = mode;
  totpDialog.title = title;
  totpDialog.visible = true;
  totpForm.code = '';
}

function closeTOTPDialog() {
  totpDialog.visible = false;
}

async function startTOTPSetup() {
  totpLoading.value = true;
  try {
    const setup = await setupUserTOTP();
    totpSetup.tempToken = setup.temp_token;
    totpSetup.totpSecret = setup.totp_secret;
    totpSetup.totpUri = setup.totp_uri;
    openTOTPDialog('setup', '开启 Google 验证');
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '开启失败，请稍后重试');
  } finally {
    totpLoading.value = false;
  }
}

async function confirmTOTPBind() {
  if (!totpForm.code.trim()) {
    ElMessage.warning('请输入验证码');
    return;
  }
  totpLoading.value = true;
  try {
    await bindUserTOTP(totpSetup.tempToken, totpForm.code.trim());
    await loadTOTPStatus();
    closeTOTPDialog();
    ElMessage.success('已开启 Google 验证');
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '绑定失败，请稍后重试');
  } finally {
    totpLoading.value = false;
  }
}

async function submitCancellation() {
  isCancellationBoxShow.value = false;
  try {
    await authSession.deleteCurrentAccount();
    router.replace({ name: 'login' });
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '注销失败，请稍后重试');
  }
}

onMounted(() => {
  void loadTOTPStatus();
});
</script>

<style scoped lang="scss">
@import "../styles/common";

.about-container {
  @extend .container;
}

.about-box {
  @extend .box;
  width: min(100%, 760px);
  max-height: min(88vh, 760px);
  padding: 0;
  overflow: auto;
  align-items: stretch;
}

.title {
  @include flex(row, space-between, center);
  position: sticky;
  top: 0;
  z-index: 1;
  padding: 18px 20px;
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid #eef2f7;
}

.title-name {
  flex: 1;
  text-align: center;
  font-size: 17px;
  font-weight: 600;
  color: #111827;
}

.title-back {
  width: 24px;
}

.title-close {
  border: 0;
  background: transparent;
  color: #64748b;
  font-size: 14px;
  cursor: pointer;
}

.main {
  padding: 18px 20px 22px;
}

.hero-card {
  @include flex(row, flex-start, center);
  gap: 16px;
  min-height: 112px;
  padding: 16px 20px;
  border-radius: 18px;
  border: 1px solid #edf2f7;
  background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
}

.hero-card-logo {
  width: 56px;
  flex-shrink: 0;
}

.hero-card-name {
  color: #111827;
  font-size: 24px;
  font-weight: 700;
  line-height: 1.1;
}

.hero-card-version {
  margin-top: 6px;
  color: #6b7280;
  font-size: 14px;
}

.hero-card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.hero-card-tag {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(67, 97, 238, 0.08);
  color: #3451d1;
  font-size: 12px;
  font-weight: 600;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 0.86fr);
  gap: 16px;
  margin-top: 18px;
  align-items: start;
}

.security-column {
  display: grid;
  gap: 14px;
}

.panel-card {
  padding: 18px 20px;
  border: 1px solid #edf2f7;
  border-radius: 18px;
  background: #fff;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
}

.panel-card-primary {
  min-height: 214px;
}

.panel-card-danger {
  background: linear-gradient(180deg, #ffffff 0%, #fff7f7 100%);
}

.panel-card-header {
  @include flex(row, space-between, center);
}

.panel-card-label {
  color: #64748b;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.panel-card-badge {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(220, 38, 38, 0.1);
  color: #dc2626;
  font-size: 12px;
  font-weight: 600;
}

.panel-card-badge-safe {
  background: rgba(22, 163, 74, 0.1);
  color: #16a34a;
}

.panel-card-title {
  margin-top: 16px;
  color: #111827;
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}

.panel-card-description {
  margin-top: 10px;
  color: #6b7280;
  font-size: 14px;
  line-height: 1.7;
}

.panel-card-action {
  width: 100%;
  min-height: 44px;
  margin-top: 18px;
  border-radius: 12px;
  border: 0;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
}

.panel-card-action-danger {
  color: #fff;
  background: #dc2626;
}

.panel-card-action-primary {
  color: #fff;
  background: #4361ee;
}

.panel-card-action-default {
  color: #3451d1;
  background: #eef2ff;
}

.panel-card-action-danger-light {
  color: #dc2626;
  background: #fee2e2;
}

.panel-card-action:disabled {
  cursor: not-allowed;
  opacity: 0.65;
}

.legal-panel {
  padding: 18px 20px;
  border: 1px solid #edf2f7;
  border-radius: 18px;
  background: #fff;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
}

.legal-panel-header {
  @include flex(row, space-between, center);
}

.legal-list {
  display: grid;
  gap: 2px;
  margin-top: 16px;
}

.legal-list-item {
  @include flex(row, space-between, center);
  width: 100%;
  min-height: 52px;
  border: 0;
  border-bottom: 1px solid #eef2f7;
  background: #fff;
  padding: 0;
  text-align: left;
  cursor: pointer;
}

.legal-list-item:last-child {
  border-bottom: 0;
}

.legal-list-title {
  min-width: 0;
  color: #111827;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
  white-space: normal;
  word-break: keep-all;
}

.legal-list-arrow {
  margin-left: 12px;
  color: #9ca3af;
  font-size: 18px;
  flex-shrink: 0;
}

.cancellation-container {
  @extend .container;
}

.cancellation-box {
  @extend .box;
  width: min(100%, 420px);
  padding: 24px;
}

.totp-box {
  @extend .box;
  width: min(92vw, 520px);
  max-height: min(86vh, 680px);
  box-sizing: border-box;
  padding: 0;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  justify-content: flex-start;
  overflow: hidden;
}

.totp-dialog-head {
  @include flex(row, space-between, center);
  flex: 0 0 auto;
  min-height: 64px;
  padding: 16px 20px;
  border-bottom: 1px solid #eef2f7;
}

.totp-dialog-title {
  min-width: 0;
  color: #111827;
  font-size: 17px;
  font-weight: 700;
  line-height: 1.4;
  word-break: break-word;
}

.totp-dialog-main {
  flex: 1 1 auto;
  width: 100%;
  min-height: 0;
  box-sizing: border-box;
  padding: 18px 22px 22px;
  overflow-y: auto;
  overflow-x: hidden;
}

.totp-dialog-desc {
  width: 100%;
  box-sizing: border-box;
  margin: 0 0 14px;
  color: #64748b;
  font-size: 14px;
  line-height: 1.6;
  white-space: normal;
  word-break: normal;
  overflow-wrap: break-word;
}

.totp-secret-box {
  margin-top: 14px;
  padding: 12px;
  border-radius: 12px;
  border: 1px solid #dbe2ee;
  background: #f8fafc;
  color: #111827;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: normal;
  word-break: break-all;
  overflow-wrap: anywhere;
}

.totp-qrcode {
  display: flex;
  justify-content: center;
  box-sizing: border-box;
  width: 100%;
  padding: 12px;
  border-radius: 16px;
  border: 1px solid #eef2f7;
  background: #ffffff;
}

.totp-qrcode canvas,
.totp-qrcode svg {
  max-width: 100%;
  height: auto;
}

.totp-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 16px;
  color: #334155;
  font-size: 14px;
  font-weight: 600;
}

.totp-field input {
  height: 42px;
  box-sizing: border-box;
  width: 100%;
  padding: 0 12px;
  border: 1px solid #dbe2ee;
  border-radius: 12px;
  font-size: 14px;
}

.btn-primary {
  width: 100%;
  box-sizing: border-box;
  margin-top: 16px;
  border: 0;
  color: #ffffff;
  background: #4361ee;
}

.dialog-main {
  @include flex(column, center, center);
}

.dialog-main-text {
  margin: 18px 0 12px;
  text-align: center;
  color: #475569;
  line-height: 1.7;
}

.dialog-main-id {
  color: #111827;
  font-weight: 600;
}

.footer-actions {
  @include flex(row, stretch, center);
  gap: 10px;
  margin-top: 18px;
}

.btn {
  flex: 1;
  height: 44px;
  border-radius: 12px;
}

.btn-default {
  @include btn-default;
}

.btn-error {
  @include btn-error;
}

.container-h5 {
  .about-box {
    width: 100%;
    height: 100%;
    max-height: none;
    border-radius: 0;
  }

  .main {
    padding: 16px;
  }

  .content-grid {
    grid-template-columns: 1fr;
  }

  .hero-card {
    min-height: 104px;
  }

  .totp-box {
    width: 100%;
    height: 100%;
    max-height: none;
    border-radius: 0;
  }

  .totp-dialog-main {
    padding: 16px;
  }

  .legal-list-title {
    word-break: break-word;
  }
}

@media (max-width: 760px) {
  .about-box {
    width: min(100%, 560px);
  }

  .content-grid {
    grid-template-columns: 1fr;
  }

  .totp-box {
    width: min(94vw, 520px);
  }
}
</style>
