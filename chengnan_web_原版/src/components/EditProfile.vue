<template>
  <div
    class="edit-profile-container container dialog"
    :class="[isH5 ? 'edit-profile-container-h5' : 'edit-profile-container-pc']"
    @click="closeEditProfileBox"
    @mousedown.stop
  >
    <div :class="['edit-profile-box']" @click.stop>
      <header class="title">
        <div v-if="isH5" class="title-back" @click="closeEditProfileBox">
          <Icon :file="backSVG"></Icon>
        </div>
        <div class="title-name">
          {{ TUITranslateService.t("Profile.编辑资料") }}
        </div>
      </header>
      <div class="edit-form">
        <div v-if="isH5" class="edit-form-space"></div>
        <div class="edit-form-item">
          <div class="form-label">
            {{ TUITranslateService.t("Profile.头像") }}
          </div>
          <div v-if="isH5" class="form-info" @click="showBottomPopup('avatar')">
            <Avatar
              useSkeletonAnimation
              :url="userProfile.avatar"
              size="60px"
            />
            <Icon
              class="form-info-arrow"
              :file="rightArrowIcon"
              size="14px"
            ></Icon>
          </div>
          <EditProfilePopup
            class="form-item"
            :show="isPC || currentBottomPopupShow === 'avatar'"
            :title="TUITranslateService.t('Profile.选择头像')"
            @onClose="closeBottomPopup"
            @onSubmit="submitEditProfileBox"
          >
            <input
              ref="avatarInputRef"
              class="avatar-upload-input"
              type="file"
              accept="image/*"
              @change="handleAvatarSelected"
            />
            <button
              class="avatar-upload-button"
              type="button"
              :disabled="avatarUploading"
              @click="openAvatarUploader"
            >
              {{ avatarUploading ? "上传中..." : "上传本地头像" }}
            </button>
            <button v-if="avatarUploading" class="avatar-upload-button" type="button" @click="cancelAvatarUpload">
              取消上传
            </button>
            <button v-if="pendingAvatar" class="avatar-upload-button" :disabled="avatarUploading" type="button" @click="retryAvatarSync">重试资料同步</button>
            <ul class="avatar-list">
              <li
                :class="[
                  'avatar-list-item',
                  currentEditProfile.avatar === avatar &&
                    'avatar-list-item-selected',
                ]"
                v-for="avatar in avatarList"
                :key="avatar"
                @click="changeCurrentEditProfile('avatar', avatar)"
              >
                <Avatar
                  useSkeletonAnimation
                  :url="avatar"
                  :size="isPC ? '36px' : '50px'"
                />
              </li>
            </ul>
          </EditProfilePopup>
        </div>
        <div v-if="isH5" class="edit-form-space"></div>
        <div class="edit-form-item">
          <div class="form-label">
            {{ TUITranslateService.t("Profile.昵称") }}
          </div>
          <div v-if="isH5" class="form-info" @click="showBottomPopup('nick')">
            <div class="form-info-content">{{ userProfile.nick }}</div>
            <Icon
              class="form-info-arrow"
              :file="rightArrowIcon"
              size="14px"
            ></Icon>
          </div>
          <EditProfilePopup
            class="form-item"
            :show="isPC || currentBottomPopupShow === 'nick'"
            :title="TUITranslateService.t('Profile.设置昵称')"
            @onClose="closeBottomPopup"
            @onSubmit="submitEditProfileBox"
          >
            <input
              class="form-item-input"
              type="text"
              v-model="currentEditProfile.nick"
            />
          </EditProfilePopup>
        </div>
        <div class="edit-form-item account-form-item">
          <div class="form-label">
            {{ TUITranslateService.t("Profile.账号") }}
          </div>
          <div class="form-info account-info">
            {{ serverPhone || currentWananID || userProfile.userID }}
          </div>
        </div>
        <div class="edit-form-item">
          <div class="form-label">
            {{ TUITranslateService.t("Profile.万安号") }}
          </div>
          <div
            v-if="isH5"
            class="form-info"
            @click="showBottomPopup('wananID')"
          >
            <div class="form-info-content">
              {{ currentWananID || TUITranslateService.t("Profile.暂未设置") }}
            </div>
            <Icon
              class="form-info-arrow"
              :file="rightArrowIcon"
              size="14px"
            ></Icon>
          </div>
          <EditProfilePopup
            class="form-item"
            :show="isPC || currentBottomPopupShow === 'wananID'"
            :title="TUITranslateService.t('Profile.万安号')"
            @onClose="closeBottomPopup"
            @onSubmit="submitEditProfileBox"
          >
            <input
              v-model="currentWananID"
              class="form-item-input"
              type="text"
              maxlength="20"
            />
          </EditProfilePopup>
        </div>
        <div v-if="isH5" class="edit-form-space"></div>
        <div class="edit-form-item">
          <div class="form-label">
            {{ TUITranslateService.t("Profile.个性签名") }}
          </div>
          <div
            v-if="isH5"
            class="form-info"
            @click="showBottomPopup('selfSignature')"
          >
            <div class="form-info-content">{{ userProfile.selfSignature }}</div>
            <Icon
              class="form-info-arrow"
              :file="rightArrowIcon"
              size="14px"
            ></Icon>
          </div>
          <EditProfilePopup
            class="form-item"
            :show="isPC || currentBottomPopupShow === 'selfSignature'"
            :title="TUITranslateService.t('Profile.个性签名')"
            @onClose="closeBottomPopup"
            @onSubmit="submitEditProfileBox"
          >
            <input
              class="form-item-input"
              type="text"
              v-model="currentEditProfile.selfSignature"
            />
          </EditProfilePopup>
        </div>
        <div class="edit-form-item">
          <div class="form-label">
            {{ TUITranslateService.t("Profile.性别") }}
          </div>
          <div v-if="isH5" class="form-info" @click="showBottomPopup('gender')">
            <div>
              {{
                (userProfile.gender &&
                  TUITranslateService.t(
                    `Profile.${genderLabelList[userProfile.gender]}`
                  )) ||
                ""
              }}
            </div>
            <Icon
              class="form-info-arrow"
              :file="rightArrowIcon"
              size="14px"
            ></Icon>
          </div>
          <EditProfilePopup
            class="form-item"
            :show="isPC || currentBottomPopupShow === 'gender'"
            :title="TUITranslateService.t('Profile.性别选择')"
            @onClose="closeBottomPopup"
            @onSubmit="submitEditProfileBox"
          >
            <ul class="gender-list">
              <li
                class="gender-list-li"
                v-for="(value, key) in genderLabelList"
                :key="key"
                @click="changeCurrentEditProfile('gender', key)"
              >
                <div
                  :class="[
                    'gender-list-item',
                    currentEditProfile.gender === key &&
                      'gender-list-item-selected',
                  ]"
                  @click="changeCurrentEditProfile('gender', key)"
                >
                  <input
                    v-if="isPC"
                    class="gender-list-radio"
                    type="radio"
                    name="gender"
                    :value="key"
                    :checked="currentEditProfile.gender === key"
                  />
                  {{ TUITranslateService.t(`Profile.${value}`) }}
                </div>
              </li>
            </ul>
          </EditProfilePopup>
        </div>
        <div class="edit-form-item">
          <div class="form-label">
            {{ TUITranslateService.t("Profile.出生年月") }}
          </div>
          <div
            v-if="isH5"
            class="form-info"
            @click="showBottomPopup('birthday')"
          >
            <div class="form-info-content">{{ birthdayObj.format }}</div>
            <Icon
              class="form-info-arrow"
              :file="rightArrowIcon"
              size="14px"
            ></Icon>
          </div>
          <EditProfilePopup
            class="form-item"
            :show="isPC || currentBottomPopupShow === 'birthday'"
            :title="TUITranslateService.t('Profile.请选择出生日期')"
            @onClose="closeBottomPopup"
            @onSubmit="submitEditProfileBox"
          >
            <div class="birthday-container">
              <DatePicker
                class="birthday-date-picker"
                type="single"
                rangeTableType="one"
                :startPlaceholder="
                  TUITranslateService.t('Profile.请选择出生日期')
                "
                popupPosition="top"
                :defaultSingleDate="birthdayObj.obj"
                @pick="pickBirthday"
              >
                <template #end-icon>
                  <Icon :file="calendarSVG" size="16px"></Icon>
                </template>
              </DatePicker>
            </div>
          </EditProfilePopup>
        </div>
      </div>
      <footer v-if="!isH5" class="edit-footer">
        <button class="btn-close" @click="closeEditProfileBox">
          {{ TUITranslateService.t("Profile.取消") }}
        </button>
        <button class="btn-save" @click="submitEditProfileBox">
          {{ TUITranslateService.t("Profile.保存") }}
        </button>
      </footer>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, defineEmits, onMounted, onBeforeUnmount } from "../TUIKit/adapter-vue";
import TUIChatEngine, {
  TUITranslateService,
  TUIStore,
  StoreName,
} from "@tencentcloud/chat-uikit-engine-lite";
import dayjs from "dayjs";
import { Toast, TOAST_TYPE } from "../TUIKit/components/common/Toast/index";
import EditProfilePopup from "./EditProfilePopup.vue";
import DatePicker from "../TUIKit/components/common/DatePicker";
import Avatar from "../TUIKit/components/common/Avatar/index.vue";
import Icon from "../TUIKit/components/common/Icon.vue";
import backSVG from "../TUIKit/assets/icon/back.svg";
import rightArrowIcon from "../TUIKit/assets/icon/right-icon.svg";
import calendarSVG from "../assets/icon/calendar.svg";
import { IUserProfile } from "../TUIKit/interface";
import { isH5, isPC } from "../TUIKit/utils/env";
import { enableSampleTaskStatus } from "../TUIKit/utils/enableSampleTaskStatus";
import {
  syncCurrentUserProfile,
  uploadAndSyncAvatar,
  syncUploadedAvatar,
} from "@/features/account/profile";
import { AvatarSyncError } from '@/features/account/avatarSync';
import { getProfile } from "@/api";

type EditableProfile = Pick<
  IUserProfile,
  "avatar" | "nick" | "selfSignature" | "gender" | "birthday"
>;

const emits = defineEmits(["closeEditProfileBox"]);
// config
const avatarListBaseUrl =
  "https://im.sdk.qcloud.com/download/tuikit-resource/avatar/avatar_";
const avatarList = ["1.png", "2.png", "3.png", "4.png", "5.png", "6.png"].map(
  (url: string) => avatarListBaseUrl + url
);
const genderLabelList: {
  [propsName: string]: string;
} = {
  [TUIChatEngine.TYPES.GENDER_MALE]: "男",
  [TUIChatEngine.TYPES.GENDER_FEMALE]: "女",
  [TUIChatEngine.TYPES.GENDER_UNKNOWN]: "不显示",
};
const userProfile = ref<Partial<IUserProfile>>({});
// current edit value
const currentEditProfile = ref<EditableProfile>({
  avatar: userProfile.value?.avatar || "",
  nick: userProfile.value?.nick || "",
  selfSignature: userProfile.value?.selfSignature || "",
  gender: userProfile.value?.gender || TUIChatEngine.TYPES.GENDER_UNKNOWN,
  birthday: userProfile.value?.birthday || 0,
});
const birthdayObj = ref<{ obj: any; format: string; value: number }>({
  obj: null,
  format: "",
  value: 0,
});
const savedWananID = ref("");
const currentWananID = ref("");
const serverPhone = ref("");
const currentBottomPopupShow = ref<string>("");
const avatarInputRef = ref<HTMLInputElement | null>(null);
const avatarUploading = ref(false);
const pendingAvatar = ref<{ owner: string; url: string } | null>(null);
let avatarAbort: AbortController | null = null;
function cancelAvatarUpload() { avatarAbort?.abort(); }
onBeforeUnmount(cancelAvatarUpload);

TUIStore.watch(StoreName.USER, {
  userProfile: (userProfileData: IUserProfile) => {
    userProfile.value = userProfileData;
    const { avatar, nick, selfSignature, gender, birthday } = userProfileData;
    currentEditProfile.value = {
      avatar,
      nick,
      selfSignature,
      gender,
      birthday,
    };
    birthdayObj.value = generateBirthdayObj(userProfileData.birthday);
  },
});

onMounted(() => {
  void loadServerProfile();
});

function generateBirthdayObj(YYYYMMDD: any) {
  let birthdayDayjsObj: any = null;
  let birthdayFormat = "";
  if (YYYYMMDD && typeof YYYYMMDD === "number") {
    birthdayDayjsObj = dayjs(YYYYMMDD.toString(), "YYYYMMDD");
    birthdayFormat = birthdayDayjsObj.format("YYYY/MM/DD");
  }
  return {
    obj: birthdayDayjsObj,
    format: birthdayFormat,
    value: userProfile.value?.birthday || 0,
  };
}

function showBottomPopup(key: string) {
  currentBottomPopupShow.value = key;
}
function closeBottomPopup() {
  currentBottomPopupShow.value = "";
}

function pickBirthday(date: any) {
  currentEditProfile.value.birthday = parseInt(date.format("YYYYMMDD"), 10);
}

function changeCurrentEditProfile(key: keyof EditableProfile, value: any) {
  if (Object.prototype.hasOwnProperty.call(currentEditProfile.value, key)) {
    (currentEditProfile.value as Record<string, any>)[key] = value;
  }
}

function closeEditProfileBox() {
  cancelAvatarUpload();
  emits("closeEditProfileBox");
}

function openAvatarUploader() {
  avatarInputRef.value?.click();
}

async function handleAvatarSelected(event: Event) {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) {
    return;
  }
  const currentUserID = userProfile.value.userID || "";
  if (!currentUserID) {
    Toast({
      message: TUITranslateService.t("Profile.修改个人资料失败"),
      type: TOAST_TYPE.ERROR,
    });
    target.value = "";
    return;
  }
  avatarUploading.value = true;
  pendingAvatar.value = null;
  const controller = new AbortController();
  avatarAbort = controller;
  try {
    const avatarURL = await uploadAndSyncAvatar(currentUserID, file, controller.signal);
    if (controller.signal.aborted) return;
    currentEditProfile.value.avatar = avatarURL;
    Toast({
      message: TUITranslateService.t("Profile.修改个人资料成功"),
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error: any) {
    if (controller.signal.aborted) return;
    if (error instanceof AvatarSyncError && userProfile.value.userID === currentUserID) pendingAvatar.value = { owner: currentUserID, url: error.avatarURL };
    Toast({
      message:
        TUITranslateService.t("Profile.修改个人资料失败") + error?.message,
      type: TOAST_TYPE.ERROR,
    });
  } finally {
    if (avatarAbort === controller) avatarAbort = null;
    avatarUploading.value = false;
    target.value = "";
  }
}

async function retryAvatarSync() {
  const pending = pendingAvatar.value;
  if (!pending || pending.owner !== userProfile.value.userID) { pendingAvatar.value = null; return; }
  avatarUploading.value = true;
  try {
    await syncUploadedAvatar(pending.owner, pending.url);
    currentEditProfile.value.avatar = pending.url;
    pendingAvatar.value = null;
    Toast({ message: '头像资料已同步', type: TOAST_TYPE.SUCCESS });
  } catch (error: any) { Toast({ message: error.message, type: TOAST_TYPE.ERROR }); }
  finally { avatarUploading.value = false; }
}

async function loadServerProfile() {
  try {
    const profile = await getProfile();
    serverPhone.value = profile.phone || "";
    savedWananID.value = profile.wanan_id || "";
    currentWananID.value = profile.wanan_id || "";
  } catch (error) {
    console.warn("获取服务端资料失败", error);
  }
}

function submitEditProfileBox() {
  const isNickModified =
    currentEditProfile.value.nick !== userProfile.value.nick;
  const normalizedWananID = currentWananID.value.trim().toLowerCase();
  const profileOptions: Record<string, unknown> = {};
  Object.keys(currentEditProfile.value).forEach((key) => {
    const value = (currentEditProfile.value as Record<string, unknown>)[key];
    if (value !== null && value !== undefined && value !== "") {
      profileOptions[key] = value;
    }
  });
  if (normalizedWananID && normalizedWananID !== savedWananID.value) {
    profileOptions.wananID = normalizedWananID;
  }
  if (!Object.keys(profileOptions).length) {
    isPC && closeEditProfileBox();
    return;
  }
  syncCurrentUserProfile(profileOptions as any)
    .then(() => {
      isNickModified && enableSampleTaskStatus("modifyNickName");
      if (normalizedWananID) {
        savedWananID.value = normalizedWananID;
      }
      Toast({
        message: TUITranslateService.t("Profile.修改个人资料成功"),
        type: TOAST_TYPE.SUCCESS,
      });
      isPC && closeEditProfileBox();
    })
    .catch((error: Error) => {
      Toast({
        message:
          TUITranslateService.t("Profile.修改个人资料失败") + error?.message,
        type: TOAST_TYPE.ERROR,
      });
      isPC && closeEditProfileBox();
    });
}
</script>
<style lang="scss" scoped>
@import "../styles/common.scss";
.edit-profile-container {
  @extend .container;
  font-size: 14px;
  .edit-profile-box {
    @extend .box;
    align-items: stretch;
    .edit-form {
      flex: 1;
      @include flex(column, flex-start, stretch);
      .edit-form-item {
        @include flex(row, flex-start, center);
        min-height: 54px;
        padding: 10px;
        .form-label {
          box-sizing: border-box;
          width: 70px;
          margin-right: 20px;
          color: #333;
        }
        .form-item {
          @include flex(row, flex-start, stretch);
          .avatar-list {
            @include flex(row, space-between, stretch);
            .avatar-list-item {
              margin: 10px;
              border: 1px solid transparent;
            }
            .avatar-list-item:first-child {
              margin-left: 0px;
            }
            .avatar-list-item-selected {
              border: 1px solid #006eff;
              color: #006eff;
              border-radius: 5px;
            }
          }
          .avatar-upload-input {
            display: none;
          }
          .avatar-upload-button {
            margin-bottom: 10px;
            border: 0;
            border-radius: 6px;
            padding: 8px 12px;
            color: #ffffff;
            background: #4361ee;
            cursor: pointer;
          }
          .form-item-input {
            flex: 1;
            padding: 6px 10px;
            border: 1px solid rgba(131, 137, 153, 0.4);
            border-radius: 2px;
            line-height: 20px;
            color: #596174;
          }
          .gender-list {
            @include flex(row, space-between, center);
            .gender-list-li {
              @include flex(row);
              margin-right: 20px;
              .gender-list-item {
                @include flex(row);
                flex: 1;
                font-size: 14px;
                .gender-list-item-radio {
                  margin: 0px 2px;
                }
              }
            }
          }
          .birthday-container {
            @include flex(row, space-between, stretch);
            flex: 1;
            padding: 6px 10px;
            border: 1px solid rgba(131, 137, 153, 0.4);
            .birthday-date-picker {
              @include flex(row);
              flex: 1;
              height: 20px;
              :deep(.tui-date-picker-input) {
                @include flex(row, flex-start);
                flex: 1;
              }
              :deep(.tui-date-picker-input-start) {
                padding: 2px 0px;
                text-align: start;
                font-size: 14px;
                &::placeholder {
                  text-align: start;
                }
              }
              :deep(.tui-date-picker-dialog-container-one) {
                left: -220px;
              }
            }
          }
        }
        &.account-form-item {
          cursor: text;
          -webkit-user-select: text;
          user-select: text;
          .form-label,
          .account-info {
            cursor: text;
            -webkit-user-select: text;
            user-select: text;
          }
        }
      }
    }
    .edit-footer {
      @include flex(row, flex-end);
      .btn-close {
        @include btn-normal;
        margin-right: 10px;
      }
      .btn-save {
        @include btn-default;
      }
      .btn-close,
      .btn-save {
        width: 90px;
        height: 30px;
        padding: 2px;
      }
    }
  }
}
.edit-profile-container-pc {
  .edit-profile-box {
    width: 495px;
    min-height: 526px;
    border-radius: 10px;
    padding: 20px;
    .edit-form {
      .edit-form-item {
        .form-item {
          flex: 1;
        }
      }
    }
  }
  .title {
    justify-content: flex-start;
    padding: 0px;
  }
}
.edit-profile-container-h5 {
  @extend .edit-profile-container;
  .edit-profile-box {
    @extend .box-h5;
    width: 100%;
    height: 100%;
    background-color: #efefef;
    .title {
      background-color: #ffffff;
    }
    .edit-form {
      .edit-form-item {
        background-color: #ffffff;
        .form-label {
          font-size: 16px;
          color: #000000;
          width: 80px;
        }
        .form-info {
          @include flex(row, flex-end, stretch);
          .form-info-content {
            @include single-line-ellipsis(auto);
          }
          flex: 1;
          overflow: hidden;
          font-size: 16px;
          color: rgba(151, 151, 151, 1);
          .form-info-arrow {
            margin-left: 2px;
          }
        }
        .form-item {
          flex: none;
          .avatar-list,
          .form-item-input,
          .gender-list,
          .birthday-container {
            margin: 0 20px;
          }
          .avatar-list {
            .avatar-list-item {
              margin: 10px 0px;
            }
          }
          .gender-list {
            .gender-list-li {
              margin: 0px;
              .gender-list-item {
                border: 1px solid rgba(221, 221, 221, 1);
                font-size: 16px;
                border-radius: 5px;
                width: 120px;
                height: 40px;
                &-selected {
                  border: 1px solid #006eff;
                  color: #006eff;
                  border-radius: 5px;
                }
              }
            }
          }
          .form-item-input {
            padding: 14px 10px;
            font-size: 16px;
            border: 0px;
            background-color: rgba(248, 248, 248, 1);
          }
        }
      }
      .edit-form-space {
        box-sizing: border-box;
        height: 10px;
        padding: 0px;
        background-color: transparent;
      }
    }
  }
}
</style>
