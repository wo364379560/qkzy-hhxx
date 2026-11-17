<template>
  <div
    v-show="isGroupPanelVisible"
    :class="[isGroupPanelVisible && 'tui-group']"
  >
    <CreateGroup v-if="isShowCreateGroup" />
    <ManageGroup v-if="isShowManageGroup" />
    <SelectMember v-if="isShowSelectMember" />
  </div>
</template>
<script lang="ts" setup>
import {
  TUIStore,
  StoreName,
} from '@tencentcloud/chat-uikit-engine-lite';
import {
  ref,
  computed,
  defineAsyncComponent,
  onMounted,
  onUnmounted,
} from '../../adapter-vue';

const CreateGroup = defineAsyncComponent(() => import('./create-group/index.vue'));
const ManageGroup = defineAsyncComponent(() => import('./manage-group/index.vue'));
const SelectMember = defineAsyncComponent(() => import('./select-member/index.vue'));

const isShowCreateGroup = ref(false);
const isShowManageGroup = ref(false);
const isShowSelectMember = ref(false);
const isGroupPanelVisible = computed(() => {
  return isShowCreateGroup.value || isShowManageGroup.value || isShowSelectMember.value;
});

const onGroupVisibilityUpdated = {
  isShowCreateComponent: (data: any) => {
    isShowCreateGroup.value = Boolean(data);
  },
  isShowManageComponent: (data: any) => {
    isShowManageGroup.value = Boolean(data);
  },
  isShowSelectComponent: (data: any) => {
    isShowSelectMember.value = Boolean(data);
  },
};

onMounted(() => {
  TUIStore.watch(StoreName.GRP, onGroupVisibilityUpdated);
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.GRP, onGroupVisibilityUpdated);
});

</script>
<style lang="scss" scoped>
.tui-group {
  width: 100%;
  height: 100%;
}
</style>
