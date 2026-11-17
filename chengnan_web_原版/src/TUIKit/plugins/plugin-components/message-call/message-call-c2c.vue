<template>
  <div
    v-if="isCallMessage && conversationType === TYPES.CONV_C2C"
    class="call"
    :class="['call-' + conversationType, message.flow === 'out' && 'call-reverse']"
    @click="callAgain"
  >
    <div :class="['icon', message.flow === 'out' && callInfo.type === 2 && 'icon-reverse']">
      <Icon :file="callInfo.icon" />
    </div>
    <span class="call-content">{{ custom }}</span>
  </div>
</template>
<script setup lang="ts">
import TUIChatEngine from '@tencentcloud/chat-uikit-engine-lite';
import { computed, ref } from '../../../adapter-vue';
import { JSONToObject } from '../../../utils/index';
import Icon from '../../../components/common/Icon.vue';
import callVideoSVG from '../../../assets/icon/call-video.svg';
import callVoiceSVG from '../../../assets/icon/call-voice.svg';
import { startDoMattxC2CCall } from '@/features/call/doMattxCallBridge';
import { DoMattxCallMediaType } from '@/features/call/doMattxCallProtocol';

const props = defineProps({
  message: {
    type: Object,
    default: () => ({}),
  },
  signalingInfo: {
    type: Object,
    default: () => ({}),
  },
  customContent: {
    type: Object,
    default: () => ({}),
  },
});
const TYPES = ref(TUIChatEngine.TYPES);
const isCallMessage = computed(() => props.signalingInfo != null);
const callInfo = computed((): { type: number; icon: string } => {
  const parsed = JSONToObject(props.signalingInfo?.data);
  const fromPayload = parsed && typeof parsed === 'object' ? Number(parsed.call_type) : 0;
  const fromText = String(props.customContent?.custom || '').includes('视频') ? 2 : 1;
  const callType = fromPayload === 1 || fromPayload === 2 ? fromPayload : fromText;
  return {
    type: callType,
    icon: callType === 2 ? callVideoSVG : callVoiceSVG,
  };
});
const conversationType = computed(() => props.message?.conversationType);
const custom = computed(() => props.customContent?.custom);

const callAgain = async () => {
  if (conversationType.value !== TUIChatEngine.TYPES.CONV_C2C) {
    return;
  }
  const userID = props.message?.flow === 'out' ? props.message?.to : props.message?.from;
  if (!userID) {
    return;
  }
  try {
    await startDoMattxC2CCall(
      userID,
      callInfo.value.type === 2 ? DoMattxCallMediaType.Video : DoMattxCallMediaType.Audio,
    );
  } catch (error) {
    console.warn('start call again failed', error);
  }
};
</script>
<style scoped lang="scss">
.call {
  display: flex;
  flex-direction: row;
  align-items: center;

  &-C2C {
    cursor: pointer;
  }

  &-GROUP {
    cursor: default;
  }

  &-content {
    padding-left: 5px;
  }

  .icon {
    width: 20px;
    height: 20px;
  }

  &-reverse {
    flex-direction: row-reverse;

    .icon-reverse {
      transform: rotate(180deg);
    }

    .call-content {
      padding-right: 5px;
      padding-left: 0;
    }
  }
}
</style>
