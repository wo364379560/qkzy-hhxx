<template>
  <div
    v-if="state.visible"
    class="do-mattx-call-mask"
  >
    <div
      class="do-mattx-call-panel"
      :class="{
        'do-mattx-call-panel-video': state.mediaType === DoMattxCallMediaType.Video,
        'do-mattx-call-panel-audio': state.mediaType === DoMattxCallMediaType.Audio,
      }"
    >
      <div
        v-if="state.mediaType === DoMattxCallMediaType.Audio"
        class="do-mattx-call-audio-bg"
      >
        <img
          v-if="state.peerAvatar"
          :src="state.peerAvatar"
          alt=""
        />
      </div>

      <div
        v-if="state.mediaType === DoMattxCallMediaType.Video"
        class="do-mattx-call-video-area"
      >
        <div
          :id="view.remoteVideoDomID"
          class="do-mattx-call-remote-video"
        >
          <div
            v-if="!state.remoteVideoAvailable"
            class="do-mattx-call-video-placeholder"
          >
            <span>等待对方视频</span>
          </div>
        </div>
        <div class="do-mattx-call-local-video-wrap">
          <div
            :id="view.localVideoDomID"
            class="do-mattx-call-local-video"
          />
          <div
            v-if="state.cameraMuted"
            class="do-mattx-call-local-video-placeholder"
          >
            摄像头已关
          </div>
        </div>
      </div>

      <div class="do-mattx-call-topbar">
        <div class="do-mattx-call-title">{{ state.peerName || state.peerUserID }}</div>
        <div class="do-mattx-call-status">{{ statusText }}</div>
      </div>

      <div class="do-mattx-call-user">
        <img
          v-if="state.peerAvatar"
          class="do-mattx-call-avatar"
          :src="state.peerAvatar"
          alt=""
        />
        <div
          v-else
          class="do-mattx-call-avatar do-mattx-call-avatar-placeholder"
        />
        <div class="do-mattx-call-name">{{ state.peerName || state.peerUserID }}</div>
        <div class="do-mattx-call-user-status">{{ statusText }}</div>
      </div>

      <div class="do-mattx-call-actions">
        <template v-if="state.phase === 'incoming'">
          <button
            type="button"
            class="do-mattx-call-button do-mattx-call-button-danger"
            @click="reject"
          >
            <span class="do-mattx-call-button-icon">
              <img
                :src="hangupIcon"
                alt=""
              />
            </span>
            <span class="do-mattx-call-button-text">拒绝</span>
          </button>
          <button
            type="button"
            class="do-mattx-call-button do-mattx-call-button-primary"
            @click="accept"
          >
            <span class="do-mattx-call-button-icon">
              <img
                :src="acceptIcon"
                alt=""
              />
            </span>
            <span class="do-mattx-call-button-text">接听</span>
          </button>
        </template>
        <template v-else>
          <button
            v-if="state.phase === 'connected'"
            type="button"
            class="do-mattx-call-button do-mattx-call-button-muted"
            :class="{ 'do-mattx-call-button-dark': state.micMuted }"
            @click="toggleMic"
          >
            <span class="do-mattx-call-button-icon">
              <img
                :src="state.micMuted ? microphoneCloseIcon : microphoneOpenIcon"
                alt=""
              />
            </span>
            <span class="do-mattx-call-button-text">{{ state.micMuted ? "开麦" : "静音" }}</span>
          </button>
          <button
            v-if="state.phase === 'connected' && state.mediaType === DoMattxCallMediaType.Video"
            type="button"
            class="do-mattx-call-button do-mattx-call-button-muted"
            :class="{ 'do-mattx-call-button-dark': !state.cameraMuted }"
            @click="toggleCamera"
          >
            <span class="do-mattx-call-button-icon">
              <img
                :src="state.cameraMuted ? cameraOpenIcon : cameraCloseIcon"
                alt=""
              />
            </span>
            <span class="do-mattx-call-button-text">{{ state.cameraMuted ? "开摄像头" : "关摄像头" }}</span>
          </button>
          <button
            type="button"
            class="do-mattx-call-button do-mattx-call-button-danger"
            @click="hangup"
          >
            <span class="do-mattx-call-button-icon">
              <img
                :src="hangupIcon"
                alt=""
              />
            </span>
            <span class="do-mattx-call-button-text">{{ state.phase === "outgoing" ? "取消" : "挂断" }}</span>
          </button>
        </template>
      </div>
      <div
        v-if="state.error"
        class="do-mattx-call-error"
      >
        {{ state.error }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onUpdated } from "vue";
import acceptIcon from "@trtc/calls-uikit-vue/src/Components/assets/button/accept.svg";
import cameraCloseIcon from "@trtc/calls-uikit-vue/src/Components/assets/button/camera-close.svg";
import cameraOpenIcon from "@trtc/calls-uikit-vue/src/Components/assets/button/camera-open.svg";
import hangupIcon from "@trtc/calls-uikit-vue/src/Components/assets/button/hangup.svg";
import microphoneCloseIcon from "@trtc/calls-uikit-vue/src/Components/assets/button/microphone-close.svg";
import microphoneOpenIcon from "@trtc/calls-uikit-vue/src/Components/assets/button/microphone-open.svg";
import {
  acceptDoMattxCall,
  doMattxCallState as state,
  doMattxCallView as view,
  hangupDoMattxCall,
  refreshDoMattxVideoViews,
  rejectDoMattxCall,
  toggleDoMattxCameraMuted,
  toggleDoMattxMicMuted,
} from "./doMattxCallBridge";
import { DoMattxCallMediaType } from "./doMattxCallProtocol";

const statusText = computed(() => {
  const mediaLabel = view.getMediaLabel(state.mediaType);
  switch (state.phase) {
    case "incoming":
      return `${mediaLabel}来电`;
    case "outgoing":
      return `正在发起${mediaLabel}`;
    case "connecting":
      return "正在连接";
    case "connected":
      return `${mediaLabel}中`;
    default:
      return "";
  }
});

function accept() {
  acceptDoMattxCall().catch((error) => console.warn("accept call failed", error));
}

function reject() {
  rejectDoMattxCall().catch((error) => console.warn("reject call failed", error));
}

function hangup() {
  hangupDoMattxCall().catch((error) => console.warn("hangup call failed", error));
}

function toggleMic() {
  toggleDoMattxMicMuted().catch((error) => console.warn("toggle mic failed", error));
}

function toggleCamera() {
  toggleDoMattxCameraMuted().catch((error) => console.warn("toggle camera failed", error));
}

onUpdated(() => {
  refreshDoMattxVideoViews();
});
</script>

<style scoped lang="scss">
.do-mattx-call-mask {
  position: fixed;
  inset: 0;
  z-index: 9999;
  pointer-events: none;
}

.do-mattx-call-panel {
  position: fixed;
  left: calc(50% - 25rem);
  top: calc(50% - 18rem);
  width: 50rem;
  height: 36rem;
  overflow: hidden;
  border-radius: 16px;
  background: #22262e;
  box-shadow:
    rgba(0, 0, 0, 0.16) 0 3px 6px,
    rgba(0, 0, 0, 0.23) 0 3px 6px;
  pointer-events: auto;
  user-select: none;
}

.do-mattx-call-audio-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
  background: #22262ed9;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    filter: blur(24px);
    opacity: 0.34;
    transform: scale(1.12);
  }

  &::after {
    position: absolute;
    inset: 0;
    content: "";
    background: #22262ed9;
  }
}

.do-mattx-call-topbar {
  position: absolute;
  top: 24px;
  left: 28px;
  right: 28px;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  color: #fff;
}

.do-mattx-call-title {
  max-width: 70%;
  overflow: hidden;
  font-size: 18px;
  font-weight: 600;
  line-height: 25px;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.do-mattx-call-status {
  font-size: 14px;
  line-height: 20px;
  color: #d5e0f2;
}

.do-mattx-call-user {
  position: absolute;
  top: 45%;
  left: 50%;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 180px;
  transform: translate(-50%, -50%);
}

.do-mattx-call-panel-video .do-mattx-call-user {
  display: none;
}

.do-mattx-call-avatar {
  width: 96px;
  height: 96px;
  border-radius: 12px;
  object-fit: cover;
  background: #e8f0ff;
}

.do-mattx-call-avatar-placeholder {
  background: linear-gradient(135deg, #2b76ff, #74a7ff);
}

.do-mattx-call-name {
  max-width: 260px;
  margin-top: 16px;
  overflow: hidden;
  font-size: 20px;
  font-weight: 600;
  color: #fff;
  line-height: 28px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.do-mattx-call-user-status {
  margin-top: 8px;
  font-size: 14px;
  color: #d5e0f2;
}

.do-mattx-call-video-area {
  position: absolute;
  inset: 0;
  overflow: hidden;
  background: #22262e;
}

.do-mattx-call-remote-video {
  width: 100%;
  height: 100%;
  background: #22262ed9;
}

.do-mattx-call-video-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d5e0f2;
  font-size: 14px;
}

.do-mattx-call-local-video-wrap {
  position: absolute;
  right: 16px;
  top: 16px;
  width: 160px;
  height: 96px;
  overflow: hidden;
  border: 1px solid rgb(255 255 255 / 40%);
  border-radius: 8px;
  background: #1f2937;
}

.do-mattx-call-local-video {
  width: 100%;
  height: 100%;
}

.do-mattx-call-local-video-placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #1f2937;
  color: #d5e0f2;
  font-size: 12px;
}

.do-mattx-call-actions {
  position: absolute;
  bottom: 6%;
  left: 50%;
  z-index: 3;
  display: flex;
  justify-content: center;
  gap: 72px;
  min-width: 280px;
  transform: translateX(-50%);
}

.do-mattx-call-button {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  border: 0;
  padding: 0;
  background: transparent;
  cursor: pointer;
}

.do-mattx-call-button-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: 50%;

  img {
    width: 26px;
    height: 26px;
  }
}

.do-mattx-call-button-primary {
  .do-mattx-call-button-icon {
    background: #51c271;
  }
}

.do-mattx-call-button-danger {
  .do-mattx-call-button-icon {
    background: #ed4651;
  }
}

.do-mattx-call-button-muted {
  .do-mattx-call-button-icon {
    background: #ffffff;
  }
}

.do-mattx-call-button-dark {
  .do-mattx-call-button-icon {
    border: 1px solid rgb(255 255 255 / 28%);
    background: rgb(255 255 255 / 16%);
  }
}

.do-mattx-call-button-text {
  font-size: 12px;
  line-height: 17px;
  color: #d5e0f2;
}

.do-mattx-call-error {
  position: absolute;
  left: 24px;
  right: 24px;
  bottom: 118px;
  z-index: 4;
  font-size: 14px;
  color: #ff9aa2;
  text-align: center;
}

@media (max-width: 768px) {
  .do-mattx-call-panel {
    inset: 0;
    width: 100vw;
    height: 100vh;
    border-radius: 0;
  }

  .do-mattx-call-actions {
    gap: 56px;
  }
}
</style>
