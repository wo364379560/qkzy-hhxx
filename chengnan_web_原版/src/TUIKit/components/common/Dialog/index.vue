<template>
  <div
    v-if="showDialog"
    class="dialog"
    :class="[!isPC ? 'dialog-h5' : '', center ? 'center' : '']"
    @click.stop.prevent="toggleView(clickType.OUTSIDE)"
  >
    <main
      class="dialog-main"
      :class="[!backgroundDialog ? 'dialog-main-back' : '']"
      @click.stop.prevent="toggleView(clickType.INSIDE)"
    >
      <header
        v-if="isHeaderShowDialog"
        :class="['dialog-main-header', headerClass]"
      >
        <h1 class="dialog-main-title">
          {{ showTitle }}
        </h1>
        <i
          class="icon icon-close"
          @click="close"
        />
      </header>
      <div
        class="dialog-main-content"
        :class="[isUniFrameWork && isH5 ? 'dialog-main-content-uniapp' : '']"
      >
        <slot />
      </div>
      <footer
        v-if="isFooterShowDialog"
        :class="['dialog-main-footer', footerClass]"
      >
        <button
          class="btn btn-cancel"
          :disabled="submitLoadingDialog"
          @click="close"
        >
          {{ TUITranslateService.t('component.取消') }}
        </button>
        <button
          class="btn btn-default"
          :disabled="submitLoadingDialog"
          @click="submit"
        >
          {{
            submitLoadingDialog
              ? '处理中...'
              : TUITranslateService.t('component.确定')
          }}
        </button>
      </footer>
    </main>
  </div>
</template>

<script lang="ts" setup>
import { ref, watchEffect } from '../../../adapter-vue';
import { TUITranslateService } from '@tencentcloud/chat-uikit-engine-lite';
import { isPC, isH5, isUniFrameWork } from '../../../utils/env';
const clickType = {
  OUTSIDE: 'outside',
  INSIDE: 'inside',
};
const props = defineProps({
  show: {
    type: Boolean,
    default: false,
  },
  isHeaderShow: {
    type: Boolean,
    default: true,
  },
  isFooterShow: {
    type: Boolean,
    default: true,
  },
  background: {
    type: Boolean,
    default: true,
  },
  title: {
    type: String,
    default: '',
  },
  center: {
    type: Boolean,
    default: false,
  },
  headerClass: {
    type: String,
    default: '',
  },
  footerClass: {
    type: String,
    default: '',
  },
  submitLoading: {
    type: Boolean,
    default: false,
  },
  autoCloseOnSubmit: {
    type: Boolean,
    default: true,
  },
});

const showDialog = ref(false);
const isHeaderShowDialog = ref(true);
const isFooterShowDialog = ref(true);
const backgroundDialog = ref(true);
const showTitle = ref('');
const submitLoadingDialog = ref(false);

watchEffect(() => {
  showDialog.value = props.show;
  showTitle.value = props.title;
  isHeaderShowDialog.value = props.isHeaderShow;
  isFooterShowDialog.value = props.isFooterShow;
  backgroundDialog.value = props.background;
  submitLoadingDialog.value = props.submitLoading;
});

const emit = defineEmits(['update:show', 'submit']);

const toggleView = (type: string) => {
  if (type === clickType.OUTSIDE) {
    close();
  }
};

const close = () => {
  if (submitLoadingDialog.value) {
    return;
  }
  showDialog.value = !showDialog.value;
  emit('update:show', showDialog.value);
};

const submit = () => {
  if (submitLoadingDialog.value) {
    return;
  }
  emit('submit');
  if (props.autoCloseOnSubmit) {
    close();
  }
};
</script>
<style lang="scss" scoped src="./style/dialog.scss"></style>
