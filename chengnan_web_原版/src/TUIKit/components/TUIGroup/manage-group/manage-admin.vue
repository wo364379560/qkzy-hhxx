<template>
  <div class="admin-main">
    <div class="admin-manage">
      <div class="admin-manage-header">
        {{ TUITranslateService.t(`TUIGroup.群管理员`) }}
      </div>
      <ul class="admin-manage-list">
        <li
          v-for="(item, index) in memberAdmin.admin"
          :key="index"
          class="admin-manage-list-item"
        >
          <div class="item-main">
            <img
              class="item-main-avatar"
              :src="
                item.avatar ||
                  'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
              "
              onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
            >
          </div>
          <div class="item-name">
            {{ item.nick || item.userID }}
          </div>
        </li>
        <li
          v-if="canManageAdminsValue"
          class="admin-manage-list-item"
        >
          <div
            :class="[
              'item-main',
              'admin-action-button',
              isActionLoading && 'admin-action-button-disabled',
              isAddAdminOpening && 'admin-action-button-loading',
            ]"
            @click="addAdmin"
          >
            <Icon
              :file="plusSVG"
              width="16px"
              height="16px"
            />
          </div>
        </li>
        <li
          v-if="canManageAdminsValue"
          class="admin-manage-list-item"
        >
          <div
            v-if="memberAdmin.admin.length > 0"
            :class="[
              'item-main',
              'admin-action-button',
              isActionLoading && 'admin-action-button-disabled',
              isRemoveAdminOpening && 'admin-action-button-loading',
            ]"
            @click="removeAdmin"
          >
            <Icon
              :file="minusSVG"
              width="16px"
              height="16px"
            />
          </div>
        </li>
      </ul>
    </div>
    <div class="admin-profile-access admin-profile-access--member-profile">
      <div class="admin-profile-access-main">
        <div class="admin-profile-access-title-row">
          <div class="admin-profile-access-title">
            {{ TUITranslateService.t(`TUIGroup.允许群成员查看彼此资料`) }}
          </div>
          <span
            class="admin-profile-access-status"
            :class="memberProfileAccessEnabledValue && 'admin-profile-access-status--open'"
          >
            {{
              TUITranslateService.t(
                memberProfileAccessEnabledValue
                  ? `TUIGroup.已开启群成员资料查看`
                  : `TUIGroup.已关闭群成员资料查看`
              )
            }}
          </span>
        </div>
      </div>
      <div @click="guardGroupSettingAccess">
        <Slider
          class="admin-profile-access-switch"
          :open="memberProfileAccessEnabledValue"
          :disabled="!canManageGroupSettingsValue"
          @change="setMemberProfileAccess"
        />
      </div>
    </div>
    <div class="admin-profile-access">
      <div>
        <div class="admin-profile-access-title">
          {{ TUITranslateService.t(`TUIGroup.启用群语音`) }}
        </div>
        <div class="admin-profile-access-content">
          {{
            TUITranslateService.t(
              `TUIGroup.默认关闭；开启后，群成员可在群聊中发起群语音。`
            )
          }}
        </div>
      </div>
      <div @click="guardGroupCallsAccess">
        <Slider
          :open="groupVoiceCallEnabledValue"
          :disabled="!canManageGroupCallsValue"
          @change="setGroupVoiceCallAccess"
        />
      </div>
    </div>
    <div class="admin-profile-access">
      <div>
        <div class="admin-profile-access-title">
          {{ TUITranslateService.t(`TUIGroup.启用群视频`) }}
        </div>
        <div class="admin-profile-access-content">
          {{
            TUITranslateService.t(
              `TUIGroup.默认关闭；开启后，群成员可在群聊中发起群视频。`
            )
          }}
        </div>
      </div>
      <div @click="guardGroupCallsAccess">
        <Slider
          :open="groupVideoCallEnabledValue"
          :disabled="!canManageGroupCallsValue"
          @change="setGroupVideoCallAccess"
        />
      </div>
    </div>
    <div
      v-if="isAdminSetMuteTime"
      class="admin-mute-all"
    >
      <div>
        <div class="admin-mute-all-title">
          {{ TUITranslateService.t(`TUIGroup.全员禁言`) }}
        </div>
        <div class="admin-mute-all-content">
          {{
            TUITranslateService.t(
              `TUIGroup.全员禁言开启后，只允许群主和管理员发言。`
            )
          }}
        </div>
      </div>
      <div @click="guardGroupSettingAccess">
        <Slider
          :open="Boolean(currentGroupAdmin?.muteAllMembers)"
          :disabled="!canManageGroupSettingsValue"
          @change="setAllMuteTime"
        />
      </div>
    </div>
    <div
      v-if="canManageSlowModeValue"
      class="admin-slow-mode"
    >
      <div class="admin-slow-mode-main">
        <div class="admin-slow-mode-title">
          发言频率限制
        </div>
        <div class="admin-slow-mode-content">
          开启后，普通成员需要间隔指定秒数才能再次发言。
        </div>
        <div
          v-if="slowModeEnabledValue"
          class="admin-slow-mode-control"
        >
          <input
            v-model.number="slowModeSecondsValue"
            class="admin-slow-mode-input"
            type="number"
            min="5"
            max="3600"
            step="1"
            :disabled="slowModeLoadingValue"
            @change="submitSlowMode(true)"
            @keyup.enter="submitSlowMode(true)"
          >
          <span class="admin-slow-mode-unit">
            秒
          </span>
        </div>
      </div>
      <Slider
        :open="slowModeEnabledValue"
        :disabled="slowModeLoadingValue"
        @change="setSlowModeEnabled"
      />
    </div>
    <div
      v-if="isAdminSetMuteTime"
      class="admin-mute"
    >
      <div class="admin-mute-header">
        {{ TUITranslateService.t(`TUIGroup.单独禁言人员`) }}
      </div>
      <ul class="admin-mute-list">
        <li
          v-for="(item, index) in memberAdmin.muteMember"
          :key="index"
          class="admin-mute-list-item"
        >
          <div class="item-main">
            <img
              class="item-main-avatar"
              :src="
                item.avatar ||
                  'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
              "
              onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
            >
          </div>
          <div class="item-name">
            {{ item.nick || item.userID }}
          </div>
        </li>
        <li class="admin-mute-list-item">
          <div
            :class="[
              'item-main',
              'admin-action-button',
              isActionLoading && 'admin-action-button-disabled',
              isAddMuteOpening && 'admin-action-button-loading',
            ]"
            @click="addMute"
          >
            <Icon
              :file="plusSVG"
              width="16px"
              height="16px"
            />
          </div>
        </li>
        <li class="admin-mute-list-item">
          <div
            v-if="memberAdmin.muteMember.length > 0"
            :class="[
              'item-main',
              'admin-action-button',
              isActionLoading && 'admin-action-button-disabled',
              isRemoveMuteOpening && 'admin-action-button-loading',
            ]"
            @click="removeMute"
          >
            <Icon
              :file="minusSVG"
              width="16px"
              height="16px"
            />
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>

<script lang="ts" setup>
import {
  TUITranslateService,
  IGroupModel,
} from '@tencentcloud/chat-uikit-engine-lite';
import { watchEffect, ref, computed } from '../../../adapter-vue';
import Slider from '../../common/Slider/index.vue';
import Icon from '../../common/Icon.vue';
import plusSVG from '../../../assets/icon/plus.svg';
import minusSVG from '../../../assets/icon/minus.svg';
import { IGroupMember } from '../../../interface';

const props = defineProps({
  member: {
    type: Object,
    default: () => {},
  },
  isSetMuteTime: {
    type: Boolean,
    default: () => false,
  },
  currentGroup: {
    type: Object,
    default: () => {},
  },
  memberProfileAccessEnabled: {
    type: Boolean,
    default: true,
  },
  groupVoiceCallEnabled: {
    type: Boolean,
    default: false,
  },
  groupVideoCallEnabled: {
    type: Boolean,
    default: false,
  },
  canManageAdmins: {
    type: Boolean,
    default: false,
  },
  canManageGroupCalls: {
    type: Boolean,
    default: false,
  },
  canManageGroupSettings: {
    type: Boolean,
    default: false,
  },
  canManageSlowMode: {
    type: Boolean,
    default: false,
  },
  slowModeEnabled: {
    type: Boolean,
    default: false,
  },
  slowModeSeconds: {
    type: Number,
    default: 30,
  },
  slowModeLoading: {
    type: Boolean,
    default: false,
  },
  actionLoading: {
    type: Boolean,
    default: () => false,
  },
  actionLoadingType: {
    type: String,
    default: () => "",
  },
});

const isAdminSetMuteTime = ref(false);
const memberAdmin = ref({
  admin: [] as Array<IGroupMember>,
  member: [] as Array<IGroupMember>,
  muteMember: [] as Array<IGroupMember>,
});
const currentGroupAdmin = ref<IGroupModel | null>(null);
const memberProfileAccessEnabledValue = ref(true);
const groupVoiceCallEnabledValue = ref(false);
const groupVideoCallEnabledValue = ref(false);
const canManageAdminsValue = ref(false);
const canManageGroupCallsValue = ref(false);
const canManageGroupSettingsValue = ref(false);
const canManageSlowModeValue = ref(false);
const slowModeEnabledValue = ref(false);
const slowModeSecondsValue = ref(30);
const slowModeLoadingValue = ref(false);
const isActionLoading = computed(() => Boolean(props.actionLoading));
const isAddAdminOpening = computed(
  () => Boolean(props.actionLoading) && props.actionLoadingType === 'addAdmin'
);
const isRemoveAdminOpening = computed(
  () => Boolean(props.actionLoading) && props.actionLoadingType === 'removeAdmin'
);
const isAddMuteOpening = computed(
  () => Boolean(props.actionLoading) && props.actionLoadingType === 'addMute'
);
const isRemoveMuteOpening = computed(
  () => Boolean(props.actionLoading) && props.actionLoadingType === 'removeMute'
);

watchEffect(() => {
  memberAdmin.value = props.member as {
    admin: Array<IGroupMember>;
    member: Array<IGroupMember>;
    muteMember: Array<IGroupMember>;
  };
  isAdminSetMuteTime.value = props.isSetMuteTime;
  currentGroupAdmin.value = (props.currentGroup as IGroupModel) || null;
  memberProfileAccessEnabledValue.value = props.memberProfileAccessEnabled;
  groupVoiceCallEnabledValue.value = props.groupVoiceCallEnabled;
  groupVideoCallEnabledValue.value = props.groupVideoCallEnabled;
  canManageAdminsValue.value = props.canManageAdmins;
  canManageGroupCallsValue.value = props.canManageGroupCalls;
  canManageGroupSettingsValue.value = props.canManageGroupSettings;
  canManageSlowModeValue.value = props.canManageSlowMode;
  slowModeEnabledValue.value = props.slowModeEnabled;
  slowModeSecondsValue.value = Number(props.slowModeSeconds || 30);
  slowModeLoadingValue.value = props.slowModeLoading;
});

const emits = defineEmits([
  'addAdmin',
  'removeAdmin',
  'setMemberProfileAccess',
  'setGroupVoiceCallAccess',
  'setGroupVideoCallAccess',
  'setSlowMode',
  'setAllMuteTime',
  'addMute',
  'removeMute',
  'blockedAction',
  'close',
]);

const addAdmin = () => {
  if (isActionLoading.value) {
    return;
  }
  emits('addAdmin');
};

const removeAdmin = () => {
  if (isActionLoading.value) {
    return;
  }
  emits('removeAdmin');
};

const setMemberProfileAccess = (value: boolean) => {
  emits('setMemberProfileAccess', value);
};

const setGroupVoiceCallAccess = (value: boolean) => {
  emits('setGroupVoiceCallAccess', value);
};

const setGroupVideoCallAccess = (value: boolean) => {
  emits('setGroupVideoCallAccess', value);
};

const guardGroupSettingAccess = () => {
  if (!canManageGroupSettingsValue.value) {
    emits('blockedAction');
  }
};

const guardGroupCallsAccess = () => {
  if (!canManageGroupCallsValue.value) {
    emits('blockedAction');
  }
};

const normalizeSlowModeSeconds = (value: number) => {
  const seconds = Number(value || 30);
  if (!Number.isFinite(seconds)) {
    return 30;
  }
  return Math.min(Math.max(Math.round(seconds), 5), 3600);
};

const submitSlowMode = (enabled: boolean) => {
  const seconds = normalizeSlowModeSeconds(slowModeSecondsValue.value);
  slowModeSecondsValue.value = seconds;
  emits('setSlowMode', { enabled, seconds });
};

const setSlowModeEnabled = (value: boolean) => {
  slowModeEnabledValue.value = value;
  submitSlowMode(value);
};

const setAllMuteTime = (value: boolean) => {
  emits('setAllMuteTime', value);
};

const addMute = () => {
  if (isActionLoading.value) {
    return;
  }
  emits('addMute');
};

const removeMute = () => {
  if (isActionLoading.value) {
    return;
  }
  emits('removeMute');
};
</script>

<style lang="scss" scoped>
@import "../../../assets/styles/common";

.admin {
  width: 100%;
  overflow: hidden;

  &-header {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    padding: 10px;

    &-left {
      font-family: "PingFang SC", sans-serif;
      font-size: 18px;
      font-weight: 500;
      line-height: 50px;
      letter-spacing: 0;
      text-align: left;
    }

    &-close {
      font-family: "PingFang SC", sans-serif;
      font-size: 16px;
      font-weight: 400;
      line-height: 48px;
      letter-spacing: 0;
      text-align: left;
      color: #3370ff;
    }
  }

  &-main {
    flex: 1;
    min-height: 0;
    width: 100%;
    overflow-y: auto;
    overflow-x: hidden;

    .admin-manage {
      border-bottom: 10px solid #f4f5f9;
    }

    .admin-manage,
    .admin-mute {
      padding: 10px;
      width: calc(100% - 20px);
      overflow: hidden;

      &-header {
        padding-left: 10px;
        font-family: "PingFang SC", sans-serif;
        font-size: 14px;
        font-weight: 400;
        line-height: 20px;
        letter-spacing: 0;
        text-align: left;
      }

      &-list {
        display: flex;
        width: 100%;
        overflow: hidden;
        flex-wrap: wrap;

        &-item {
          flex: 0 0 36px;
          display: flex;
          flex-direction: column;
          padding: 10px;

          .item-main {
            width: 36px;
            height: 36px;
            border-radius: 4px;
            font-size: 12px;
            display: flex;
            justify-content: center;
            align-items: center;
            background: #f4f5f9;
            color: #000;

            &.admin-action-button {
              cursor: pointer;
              position: relative;
              transition: background-color 0.15s ease, opacity 0.15s ease;

              &:hover {
                background: #eef4ff;
              }
            }

            &.admin-action-button-disabled {
              cursor: not-allowed;
              opacity: 0.55;
              pointer-events: none;
            }

            &.admin-action-button-loading {
              color: transparent;

              :deep(svg),
              :deep(img) {
                opacity: 0;
              }

              &::after {
                content: "";
                position: absolute;
                width: 14px;
                height: 14px;
                border: 2px solid rgba(51, 112, 255, 0.25);
                border-top-color: #3370ff;
                border-radius: 50%;
                animation: admin-action-spin 0.8s linear infinite;
              }
            }

            &-avatar {
              width: 36px;
              height: 36px;
              overflow: hidden;
              border-radius: 4px;
            }
          }

          .item-name {
            text-align: center;
            max-width: 36px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }
      }
    }

    .admin-mute-all,
    .admin-slow-mode,
    .admin-profile-access {
      margin: 0 10px;
      padding: 20px 0;
      border-bottom: 1px solid #e8e8e9;
      display: flex;
      flex-direction: row;
      justify-content: space-between;
      align-items: center;
      gap: 12px;

      &-main {
        flex: 1;
        min-width: 0;
      }

      &-title-row {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 8px;
        padding-left: 10px;
      }

      &-title {
        padding-left: 10px;
        font-family: "PingFang SC", sans-serif;
        font-size: 14px;
        font-weight: 400;
        line-height: 20px;
        letter-spacing: 0;
        text-align: left;
      }

      &-status {
        flex: 0 0 auto;
        height: 24px;
        padding: 0 8px;
        border-radius: 12px;
        background: #f2f3f5;
        color: #7a8494;
        font-family: "PingFang SC", sans-serif;
        font-size: 12px;
        font-weight: 500;
        line-height: 24px;

        &--open {
          background: #e8f1ff;
          color: #006eff;
        }
      }

      &-content {
        color: #999;
        padding-left: 10px;
        font-family: "PingFang SC", sans-serif;
        font-size: 12px;
        font-weight: 400;
        line-height: 17px;
        letter-spacing: 0;
        text-align: left;
      }

      &--member-profile {
        padding: 18px 0;

        .admin-profile-access-title {
          padding-left: 0;
          color: #111827;
          font-size: 16px;
          font-weight: 600;
          line-height: 24px;
        }
      }
    }

    .admin-slow-mode {
      &-main {
        flex: 1;
        min-width: 0;
      }

      &-title {
        padding-left: 10px;
        font-family: "PingFang SC", sans-serif;
        font-size: 14px;
        font-weight: 400;
        line-height: 20px;
        letter-spacing: 0;
        text-align: left;
      }

      &-content {
        color: #999;
        padding-left: 10px;
        font-family: "PingFang SC", sans-serif;
        font-size: 12px;
        font-weight: 400;
        line-height: 17px;
        letter-spacing: 0;
        text-align: left;
      }

      &-control {
        display: flex;
        align-items: center;
        gap: 6px;
        padding-left: 10px;
        padding-top: 10px;
      }

      &-input {
        width: 88px;
        height: 32px;
        box-sizing: border-box;
        padding: 0 8px;
        border: 1px solid #d9dce3;
        border-radius: 4px;
        color: #111827;
        font-size: 14px;
        outline: none;

        &:focus {
          border-color: #3370ff;
        }
      }

      &-unit {
        color: #666;
        font-size: 13px;
      }
    }
  }
}

@keyframes admin-action-spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}
</style>
