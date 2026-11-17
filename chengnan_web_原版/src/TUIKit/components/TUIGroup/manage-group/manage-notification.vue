<template>
  <main
    v-if="!isUniFrameWork"
    class="notification"
  >
    <label v-if="canPin && !isEdit" class="notice-pin-row">
      <span>置顶公告</span>
      <input type="checkbox" role="switch" aria-label="置顶公告" :checked="pinned" :disabled="saving" @change="changePinned" />
    </label>
    <textarea
      v-if="isEdit"
      v-model="input"
      class="textarea"
      
    />
    <div
      v-if="isEdit"
      class="byte-counter"
      :class="{
        'byte-counter-warning': currentByteLength > 4000 && currentByteLength <= GROUP_NOTICE_MAX_BYTES,
        'byte-counter-error': currentByteLength > GROUP_NOTICE_MAX_BYTES,
      }"
    >
      {{ currentByteLength }}/{{ GROUP_NOTICE_MAX_BYTES }} 字节
    </div>
    <section v-else>
      <p v-if="!groupProfile.notification">
        {{ TUITranslateService.t(`TUIGroup.暂无公告`) }}
      </p>
      <article v-else>
        {{ groupProfile.notification }}
      </article>
    </section>
    <footer v-if="isAuthorNotification">
      <button
        v-if="isEdit"
        class="btn"
        @click="updateProfile"
      >
        {{ TUITranslateService.t(`TUIGroup.发布`) }}
      </button>
      <button
        v-else
        class="btn"
        @click="isEdit = !isEdit"
      >
        {{ TUITranslateService.t(`TUIGroup.编辑`) }}
      </button>
    </footer>
  </main>
  <div
    v-else
    class="edit-h5"
  >
    <main class="edit-h5-main">
      <header class="edit-h5-header">
        <aside class="left">
          <h1>{{ TUITranslateService.t(`TUIGroup.群公告`) }}</h1>
        </aside>
        <span
          class="close"
          @click="close('notification')"
        >{{
          TUITranslateService.t(`关闭`)
        }}</span>
      </header>
      <div class="notification">
        <textarea
          v-if="isEdit"
          v-model="input"
          :class="[isUniFrameWork ? 'uni-height' : '', 'textarea']"
          
        />
        <div
          v-if="isEdit"
          class="byte-counter"
          :class="{
            'byte-counter-warning': currentByteLength > 4000 && currentByteLength <= GROUP_NOTICE_MAX_BYTES,
            'byte-counter-error': currentByteLength > GROUP_NOTICE_MAX_BYTES,
          }"
        >
          {{ currentByteLength }}/{{ GROUP_NOTICE_MAX_BYTES }} 字节
        </div>
        <section
          v-else
          class="row"
        >
          <p
            v-if="!groupProfile.notification"
            class="row-p"
          >
            {{ TUITranslateService.t(`TUIGroup.暂无公告`) }}
          </p>
          <article v-else>
            {{ groupProfile.notification }}
          </article>
        </section>
        <footer
          v-if="isAuthorNotification"
          class="footer"
        >
          <button
            v-if="isEdit"
            class="btn"
            @click="updateProfile"
          >
            {{ TUITranslateService.t(`TUIGroup.发布`) }}
          </button>
          <button
            v-else
            class="btn"
            @click="isEdit = !isEdit"
          >
            {{ TUITranslateService.t(`TUIGroup.编辑`) }}
          </button>
        </footer>
      </div>
    </main>
  </div>
</template>

<script lang="ts" setup>
// @ts-nocheck
import {
  TUITranslateService,
  IGroupModel,
} from '@tencentcloud/chat-uikit-engine-lite';
import { watchEffect, ref, computed, nextTick } from '../../../adapter-vue';
import { Toast, TOAST_TYPE } from '../../common/Toast/index';
import { isUniFrameWork } from '../../../utils/env';

const props = defineProps({
  data: {
    type: Object,
    default: () => ({}),
  },
  canPin: { type: Boolean, default: false },
  pinned: { type: Boolean, default: false },
  saving: { type: Boolean, default: false },
  isAuthor: {
    type: Boolean,
    default: false,
  },
});

const groupProfile = ref<IGroupModel>({});
const input = ref('');
const isAuthorNotification = ref(false);
const isEdit = ref(false);

watchEffect(() => {
  groupProfile.value = props.data;
  input.value = groupProfile.value.notification;
  isAuthorNotification.value = props.isAuthor;
});

const emits = defineEmits(['update', 'close', 'pin']);
const GROUP_NOTICE_MAX_BYTES = 5000;
function changePinned(event: Event) {
  const checkbox = event.target as HTMLInputElement;
  const next = checkbox.checked;
  checkbox.checked = props.pinned;
  emits('pin', next);
}

function getUtf8ByteLength(value: string) {
  return new TextEncoder().encode(value).length;
}

const currentByteLength = computed(() => getUtf8ByteLength(input.value || ''));

const updateProfile = () => {
  const normalizedValue = input.value.trim();
  if (getUtf8ByteLength(normalizedValue) > GROUP_NOTICE_MAX_BYTES) {
    Toast({
      message: `群公告字节数超出限制，最大长度为${GROUP_NOTICE_MAX_BYTES}字节`,
      type: TOAST_TYPE.ERROR,
    });
    return;
  }
  if (normalizedValue !== (groupProfile.value.notification || '')) {
    emits('update', { key: 'notification', value: normalizedValue });
    nextTick(() => {
      input.value = '';
    });
  }
  isEdit.value = !isEdit.value;
};

const close = (tabName: string) => {
  emits('close', tabName);
};
</script>

<style lang="scss" scoped>
@import "../../../assets/styles/common";

.notice-pin-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px; }
.notice-pin-row input { appearance: none; width: 36px; height: 20px; border-radius: 12px; background: #bbb; cursor: pointer; position: relative; }
.notice-pin-row input::after { content: ""; position: absolute; width: 16px; height: 16px; background: white; border-radius: 50%; top: 2px; left: 2px; transition: transform .15s; }
.notice-pin-row input:checked { background: #1677ff; }
.notice-pin-row input:checked::after { transform: translateX(16px); }
.notice-pin-row input:disabled { opacity: .5; cursor: wait; }

.notification {
  flex: 1;
  min-height: 0;
  padding: 20px;
  display: flex;
  flex-direction: column;
  word-break: break-all;
  overflow-y: auto;
  overflow-x: hidden;

  .row {
    flex: 1;
    font-size: 14px;

    .row-p {
      text-align: center;
      padding-bottom: 20px;
    }
  }

  .textarea {
    margin-bottom: 20px;
    box-sizing: border-box;
    padding: 10px;
    border: 1px solid #e8e8e9;
    resize: none;
    font-size: 14px;
    height: 100%;
  }

  .byte-counter {
    margin-top: -12px;
    margin-bottom: 12px;
    font-size: 12px;
    color: #999;
    text-align: right;
  }

  .byte-counter-warning {
    color: #d48806;
  }

  .byte-counter-error {
    color: #cf1322;
  }

  .uni-height {
    height: 20vh;
  }

  .footer {
    display: flex;
    justify-content: flex-end;
    padding: 20px 10px;
  }
}

.btn {
  background: #3370ff;
  border: 0 solid #2f80ed;
  padding: 4px 28px;
  font-weight: 400;
  font-size: 12px;
  color: #fff;
  line-height: 24px;
  border-radius: 4px;

  &-cancel {
    background: #fff;
    border: 1px solid #ddd;
    color: #828282;
  }
}

.edit-h5 {
  position: fixed;
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: flex-end;
  z-index: 1;

  .edit-h5-main {
    background: #fff;
    flex: 1;
    padding: 18px;
    border-radius: 12px 12px 0 0;
    width: 80vw;
  }

  &-header {
    display: flex;
    align-items: center;
    justify-content: space-between;

    .close {
      font-family: PingFangSC-Regular;
      font-weight: 400;
      font-size: 18px;
      color: #3370ff;
      letter-spacing: 0;
      line-height: 27px;
    }
  }

  &-footer {
    display: flex;

    .btn {
      flex: 1;
      border: none;
      background: #147aff;
      border-radius: 5px;
      font-family: PingFangSC-Regular;
      font-weight: 400;
      font-size: 16px;
      color: #fff;
      letter-spacing: 0;
      line-height: 27px;
      padding: 8px 0;

      &:disabled {
        opacity: 0.3;
      }
    }
  }
}
</style>
