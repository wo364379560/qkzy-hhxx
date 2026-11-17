import { reactive, readonly } from 'vue';

type ReportTargetType = 'user' | 'group';

interface ReportDialogState {
  visible: boolean;
  targetId: string;
  targetType: ReportTargetType;
  targetName: string;
}

const state = reactive<ReportDialogState>({
  visible: false,
  targetId: '',
  targetType: 'user',
  targetName: '',
});

export function openReportDialog(payload: Omit<ReportDialogState, 'visible'>) {
  state.visible = true;
  state.targetId = payload.targetId;
  state.targetType = payload.targetType;
  state.targetName = payload.targetName;
}

export function closeReportDialog() {
  state.visible = false;
  state.targetId = '';
  state.targetType = 'user';
  state.targetName = '';
}

export function useReportDialog() {
  return {
    state: readonly(state),
    openReportDialog,
    closeReportDialog,
  };
}
