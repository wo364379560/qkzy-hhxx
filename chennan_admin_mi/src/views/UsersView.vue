<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox, type TableInstance } from "element-plus";
import { Download, MoreFilled, Plus, Refresh } from "@element-plus/icons-vue";
import {
  getUserList,
  getIMApplication,
  repairUserIM,
  getUserOwners,
  batchCreateUsers,
  generateUsers,
  banUser,
  unbanUser,
  kickUser,
  resolveUserLoginCooldown,
  deleteUser,
  disableUserTOTP,
  enableUserTOTP,
  resetUserRealname,
  getUserDeletePreview,
  transferUser,
  batchBanUsers,
  batchDeleteUsers,
  batchUnbanUsers,
  batchTransferUsers,
  batchSetUserCreateGroupPermission,
  batchSetUserGlobalSearch,
  batchSaveUserIPWhitelist,
  syncUsersPreview,
  syncUsersConfirm,
  getUserDetail,
  saveUserIPWhitelist,
  batchGetUserStatus,
  exportUsers,
  updateUser,
  type GenerateUsersParams,
  type GenerateUsersResultItem,
  type GeneratedPasswordMode,
  type UserDetail,
  type UserDeletePreview,
} from "@/api/user";
import UserEditDialog from "@/components/UserEditDialog.vue";
import { useAuthStore } from "@/stores/auth";
import type {
  AppUser,
  SyncUserPreview,
  SyncUserStatus,
  UserOwnerOption,
} from "@/types";
import { useResponsive } from "@/composables/useResponsive";

// ── 角色 ──
const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { isMobile } = useResponsive();
const isAdmin = computed(() => authStore.isAdmin);
const pageSubText = computed(() => {
  if (authStore.isSuperAdmin) return "所有 IM 注册用户";
  if (authStore.isManager) return "我归属范围内的 IM 注册用户";
  return "我名下的 IM 注册用户";
});

// ── 筛选 ──
interface FilterField {
  key: string;
  label: string;
  type: "text" | "select";
  paramKey?: string;
  options?: { value: string; label: string }[];
}
interface ActiveFilter {
  key: string;
  paramKey: string;
  label: string;
  value: string;
  display: string;
}
const activeFilters = ref<ActiveFilter[]>([]);
const filterWrapRef = ref<HTMLElement | null>(null);
const dropdownOpen = ref(false);
const selectedField = ref<FilterField | null>(null);
const filterInput = ref("");
const filterSelect = ref("");
const ownerOptions = ref<UserOwnerOption[]>([]);

const ownerFilterOptions = computed(() =>
  ownerOptions.value.map((owner) => ({
    value: String(owner.id),
    label: owner.username,
  })),
);

const ownerManagementOptions = computed(() =>
  ownerOptions.value.map((owner) => {
    const isSelf = owner.id === authStore.user?.id;
    const roleLabel =
      owner.role === "super_admin"
        ? "超级管理员"
        : owner.role === "admin"
          ? "管理员"
          : "助理";
    const statusSuffix = owner.status === "active" ? "" : "（已禁用）";
    const selfSuffix = isSelf ? "（我自己）" : "";
    return {
      value: owner.id,
      label: `${roleLabel}: ${owner.username}${selfSuffix}${statusSuffix}`,
    };
  }),
);

const defaultManagedOwnerID = computed(() => authStore.user?.id ?? 0);
const defaultManagedOwnerLabel = computed(() => {
  if (isAdmin.value) {
    return (
      ownerManagementOptions.value.find(
        (option) => option.value === defaultManagedOwnerID.value,
      )?.label ?? "当前账号"
    );
  }
  const username = authStore.user?.username || "当前助理";
  return `助理: ${username}`;
});

const selectedFieldOptions = computed(() => {
  if (!selectedField.value) return [];
  if (selectedField.value.key === "ownerName") return ownerFilterOptions.value;
  return selectedField.value.options ?? [];
});

const FILTER_FIELDS = computed<FilterField[]>(() => {
  const fields: FilterField[] = [];
  if (isAdmin.value) {
    fields.push({
      key: "ownerName",
      paramKey: "ownerID",
      label: "归属昵称",
      type: "select",
      options: ownerFilterOptions.value,
    });
  }
  fields.push(
    {
      key: "status",
      label: "账号状态",
      type: "select",
      options: [
        { value: "active", label: "正常" },
        { value: "banned", label: "已封禁" },
      ],
    },
    {
      key: "isActivated",
      label: "激活状态",
      type: "select",
      options: [
        { value: "true", label: "已激活" },
        { value: "false", label: "未激活" },
      ],
    },
    {
      key: "ipWhitelistEnabled",
      label: "白名单状态",
      type: "select",
      options: [
        { value: "true", label: "已启用" },
        { value: "false", label: "未启用" },
      ],
    },
    {
      key: "currentLoginIP",
      label: "当前登录 IP",
      type: "text",
    },
  );
  return fields;
});

function openDropdown() {
  dropdownOpen.value = true;
  selectedField.value = null;
  filterInput.value = "";
  filterSelect.value = "";
  if (isAdmin.value && ownerOptions.value.length === 0) {
    void loadUserOwners();
  }
}
function selectField(f: FilterField) {
  selectedField.value = f;
  filterInput.value = "";
  filterSelect.value = "";
}
function confirmFilter() {
  if (!selectedField.value) return;
  const f = selectedField.value;
  const raw =
    f.type === "select" ? filterSelect.value : filterInput.value.trim();
  if (!raw) return;
  const display =
    f.type === "select"
      ? (selectedFieldOptions.value.find((o) => o.value === raw)?.label ?? raw)
      : raw;
  activeFilters.value = activeFilters.value.filter((a) => a.key !== f.key);
  activeFilters.value.push({
    key: f.key,
    paramKey: f.paramKey ?? f.key,
    label: f.label,
    value: raw,
    display,
  });
  dropdownOpen.value = false;
  selectedField.value = null;
  page.value = 1;
  clearSelection();
  syncUsersRouteQuery();
}
function removeFilter(key: string) {
  activeFilters.value = activeFilters.value.filter((a) => a.key !== key);
  page.value = 1;
  clearSelection();
  syncUsersRouteQuery();
}
function clearAllFilters() {
  activeFilters.value = [];
  page.value = 1;
  clearSelection();
  syncUsersRouteQuery();
}
function onDocMousedown(e: MouseEvent) {
  if (filterWrapRef.value && !filterWrapRef.value.contains(e.target as Node))
    dropdownOpen.value = false;
}
onMounted(() => document.addEventListener("mousedown", onDocMousedown));
onUnmounted(() => document.removeEventListener("mousedown", onDocMousedown));

// ── 列表 ──
const loading = ref(false);
const loginCooldownResolvingIds = ref<Set<number>>(new Set());
const users = ref<AppUser[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(10);
const pageSizeOptions = [10, 30, 50];
const usersTableRef = ref<TableInstance>();
const selectedUsers = ref<AppUser[]>([]);
const searchKeyword = ref("");
const exportingFilteredUsers = ref(false);

// ── 编辑弹窗 ──
const showEditDialog = ref(false);
const editingUser = ref<UserDetail | null>(null);
const showWhitelistModal = ref(false);
const whitelistTargetUser = ref<UserDetail | null>(null);
const whitelistLoading = ref(false);
const whitelistSubmitting = ref(false);
const whitelistForm = reactive({
  enabled: false,
  ip: "",
  currentLoginIP: "",
});
const showBatchWhitelistModal = ref(false);
const batchWhitelistSubmitting = ref(false);
const batchWhitelistForm = reactive({
  enabled: true,
  ip: "",
});

function splitWhitelistIPs(value: string) {
  return value
    .split(/[\n\r,，;；]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function appendWhitelistIP(ip: string) {
  const trimmedIP = ip.trim();
  if (!trimmedIP) return;
  const entries = splitWhitelistIPs(whitelistForm.ip);
  if (!entries.includes(trimmedIP)) {
    entries.push(trimmedIP);
  }
  whitelistForm.ip = entries.join("\n");
}

function formatWhitelistPreview(value: string) {
  const entries = splitWhitelistIPs(value);
  if (entries.length === 0) return "";
  if (entries.length <= 2) return entries.join(" / ");
  return `${entries.slice(0, 2).join(" / ")} 等 ${entries.length} 个`;
}

async function openEditDialog(user: AppUser) {
  try {
    const res = await getUserDetail(user.id);
    editingUser.value = res.data.data;
    showEditDialog.value = true;
  } catch (e: any) {
    ElMessage.error(e.message || "获取用户详情失败");
  }
}

watch(
  () => whitelistForm.enabled,
  (enabled) => {
    if (enabled && !whitelistForm.ip.trim() && whitelistForm.currentLoginIP) {
      whitelistForm.ip = whitelistForm.currentLoginIP;
    }
  },
);

async function openWhitelistModal(user: AppUser) {
  whitelistLoading.value = true;
  showWhitelistModal.value = true;
  try {
    const res = await getUserDetail(user.id);
    const detail = res.data.data;
    whitelistTargetUser.value = detail;
    whitelistForm.enabled = Boolean(detail.ip_whitelist_enabled);
    whitelistForm.currentLoginIP = detail.current_login_ip?.trim() || "";
    whitelistForm.ip = detail.ip_whitelist_ip || detail.current_login_ip || "";
  } catch (e: any) {
    showWhitelistModal.value = false;
    whitelistTargetUser.value = null;
    ElMessage.error(e.message || "获取白名单配置失败");
  } finally {
    whitelistLoading.value = false;
  }
}

function closeWhitelistModal() {
  showWhitelistModal.value = false;
  whitelistTargetUser.value = null;
  whitelistForm.enabled = false;
  whitelistForm.ip = "";
  whitelistForm.currentLoginIP = "";
}

async function submitWhitelistModal() {
  if (!whitelistTargetUser.value) return;
  if (whitelistForm.enabled && !whitelistForm.ip.trim()) {
    ElMessage.warning("启用白名单后请填写至少 1 个 IP 地址");
    return;
  }

  whitelistSubmitting.value = true;
  try {
    const res = await saveUserIPWhitelist(whitelistTargetUser.value.id, {
      enabled: whitelistForm.enabled,
      ip: whitelistForm.ip.trim() || undefined,
    });
    const result = res.data.data;
    if (result?.forced_logout) {
      ElMessage.success("白名单已保存，用户当前登录 IP 不匹配，已被强制下线");
    } else {
      ElMessage.success("登录白名单已保存");
    }
    closeWhitelistModal();
    await fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "保存白名单失败");
  } finally {
    whitelistSubmitting.value = false;
  }
}

function openBatchWhitelistModal() {
  if (selectedUsers.value.length === 0) {
    ElMessage.warning("请先选择用户");
    return;
  }
  batchWhitelistForm.enabled = true;
  batchWhitelistForm.ip = "";
  showBatchWhitelistModal.value = true;
}

function closeBatchWhitelistModal() {
  showBatchWhitelistModal.value = false;
  batchWhitelistForm.enabled = true;
  batchWhitelistForm.ip = "";
}

async function submitBatchWhitelistModal() {
  if (selectedUsers.value.length === 0) {
    ElMessage.warning("请先选择用户");
    return;
  }
  if (batchWhitelistForm.enabled && !batchWhitelistForm.ip.trim()) {
    ElMessage.warning("启用白名单后请填写至少 1 个 IP 地址");
    return;
  }

  await ElMessageBox.confirm(
    `确认将已选 ${selectedUsers.value.length} 个用户的登录白名单设置为相同配置？该操作会覆盖这些用户原有登录白名单。`,
    "确认批量设置",
    { type: "warning" },
  );

  batchWhitelistSubmitting.value = true;
  try {
    const userIDs = selectedUsers.value.map((user) => user.id);
    const res = await batchSaveUserIPWhitelist(userIDs, {
      enabled: batchWhitelistForm.enabled,
      ip: batchWhitelistForm.ip.trim() || undefined,
    });
    const data = res.data.data;
    const failedDetails = (data.details ?? [])
      .filter((item) => !item.success)
      .slice(0, 3)
      .map((item) => `${item.nickname || item.userID}: ${item.message}`)
      .join("；");
    const forcedLogoutText =
      data.forced_logout > 0 ? `，强制下线 ${data.forced_logout} 个` : "";
    const summary = `成功 ${data.success} 个，失败 ${data.failed} 个${forcedLogoutText}`;
    ElMessage[data.failed > 0 ? "warning" : "success"](
      `批量登录白名单设置完成：${summary}${failedDetails ? `；${failedDetails}` : ""}`,
    );
    closeBatchWhitelistModal();
    clearSelection();
    await fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "批量设置登录白名单失败");
  } finally {
    batchWhitelistSubmitting.value = false;
  }
}

async function handleQuickEnableWhitelist(user: AppUser) {
  const currentLoginIP = user.currentLoginIP?.trim();
  if (!currentLoginIP) {
    ElMessage.warning("未获取到当前登录 IP，请手动设置白名单");
    await openWhitelistModal(user);
    return;
  }

  whitelistSubmitting.value = true;
  try {
    const res = await saveUserIPWhitelist(user.id, {
      enabled: true,
      ip: currentLoginIP,
    });
    const result = res.data.data;
    if (result?.forced_logout) {
      ElMessage.success("白名单已启用，存在不匹配会话，用户已被强制下线");
    } else {
      ElMessage.success("已按当前登录 IP 快捷启用白名单");
    }
    await fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "快捷启用失败");
  } finally {
    whitelistSubmitting.value = false;
  }
}

async function handleQuickDisableWhitelist(user: AppUser) {
  whitelistSubmitting.value = true;
  try {
    await saveUserIPWhitelist(user.id, { enabled: false });
    ElMessage.success("已关闭登录白名单");
    await fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "关闭白名单失败");
  } finally {
    whitelistSubmitting.value = false;
  }
}

// ── 在线状态 ──
const statusRefreshing = ref(false);

async function refreshStatus(showSuccess = true) {
  if (users.value.length === 0) return;
  statusRefreshing.value = true;
  try {
    const imUserIDs = users.value.map((u) => u.imUserID);
    const res = await batchGetUserStatus(imUserIDs);
    const statusMap = new Map(
      res.data.data?.map((s) => [s.im_user_id, s]) ?? [],
    );
    if (showSuccess) {
      await fetchUsers(false);
    }
    users.value.forEach((u) => {
      const status = statusMap.get(u.imUserID);
      u.onlineStatus = status?.status || "offline";
      u.platform = status?.platform || "";
      if (status?.last_offline_at) {
        u.lastOfflineAt = status.last_offline_at;
      }
    });
    if (showSuccess) {
      ElMessage.success("状态已刷新");
    }
  } catch (e: any) {
    if (showSuccess) {
      ElMessage.error(e.message || "刷新状态失败");
    }
  } finally {
    statusRefreshing.value = false;
  }
}

function clearSelection() {
  selectedUsers.value = [];
  usersTableRef.value?.clearSelection();
}

function isValidPhone(value: string) {
  return /^1[3-9]\d{9}$/.test(value.trim());
}

async function loadUserOwners() {
  if (!isAdmin.value) {
    ownerOptions.value = [];
    return;
  }
  try {
    const res = await getUserOwners();
    ownerOptions.value = res.data.data ?? [];
    restoreFiltersFromRoute();
  } catch (e: any) {
    ownerOptions.value = [];
    ElMessage.error(e.message || "获取归属人员列表失败");
  }
}

const buildParams = () => {
  const p: Record<string, any> = { page: page.value, pageSize: pageSize.value };
  if (searchKeyword.value.trim()) p.keyword = searchKeyword.value.trim();
  for (const f of activeFilters.value) p[f.paramKey] = f.value;
  return p;
};

const USERS_FILTER_QUERY_KEYS = [
  "ownerID",
  "status",
  "isActivated",
  "ipWhitelistEnabled",
  "currentLoginIP",
] as const;

function queryValue(value: unknown) {
  if (Array.isArray(value)) return String(value[0] ?? "");
  return value == null ? "" : String(value);
}

function positiveInteger(value: unknown, fallback: number) {
  const parsed = Number.parseInt(queryValue(value), 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function restoreFiltersFromRoute() {
  const fieldsByParam = new Map(
    FILTER_FIELDS.value.map((field) => [field.paramKey ?? field.key, field]),
  );
  activeFilters.value = USERS_FILTER_QUERY_KEYS.flatMap((paramKey) => {
    const value = queryValue(route.query[paramKey]);
    if (!value) return [];
    const field = fieldsByParam.get(paramKey);
    if (!field) return [];
    const display = field.type === "select"
      ? (field.options?.find((option) => option.value === value)?.label ?? value)
      : value;
    return [{
      key: field.key,
      paramKey,
      label: field.label,
      value,
      display,
    }];
  });
}

function restoreUsersStateFromRoute() {
  page.value = positiveInteger(route.query.page, 1);
  const restoredPageSize = positiveInteger(route.query.pageSize, 10);
  pageSize.value = pageSizeOptions.includes(restoredPageSize)
    ? restoredPageSize
    : 10;
  searchKeyword.value = queryValue(route.query.keyword);
  restoreFiltersFromRoute();
}

function buildUsersRouteQuery() {
  const query: Record<string, string> = {
    page: String(page.value),
    pageSize: String(pageSize.value),
  };
  const keyword = searchKeyword.value.trim();
  if (keyword) query.keyword = keyword;
  for (const filter of activeFilters.value) {
    query[filter.paramKey] = filter.value;
  }
  return query;
}

function syncUsersRouteQuery() {
  const query = buildUsersRouteQuery();
  const nextPath = router.resolve({ name: "Users", query }).fullPath;
  if (nextPath === route.fullPath) {
    void fetchUsers();
    return;
  }
  void router.replace({ name: "Users", query });
}

const buildExportParams = () => {
  const params = buildParams();
  const exportParams: Record<string, any> = {};
  if (params.keyword) exportParams.keyword = params.keyword;
  if (params.status) exportParams.status = params.status;
  if (params.currentLoginIP) exportParams.currentLoginIP = params.currentLoginIP;
  if (params.ownerID) exportParams.ownerID = Number(params.ownerID);
  if (params.isActivated === "true") exportParams.isActivated = true;
  if (params.isActivated === "false") exportParams.isActivated = false;
  if (params.ipWhitelistEnabled === "true") exportParams.ipWhitelistEnabled = true;
  if (params.ipWhitelistEnabled === "false") exportParams.ipWhitelistEnabled = false;
  return exportParams;
};

function applySearch() {
  page.value = 1;
  clearSelection();
  syncUsersRouteQuery();
}

function clearSearch() {
  if (!searchKeyword.value) return;
  searchKeyword.value = "";
  page.value = 1;
  clearSelection();
  syncUsersRouteQuery();
}

async function handleExportFilteredUsers() {
  if (total.value === 0) {
    ElMessage.warning("暂无可导出的记录");
    return;
  }

  exportingFilteredUsers.value = true;
  try {
    const blob = await exportUsers(buildExportParams());
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `users_export_${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
    ElMessage.success(`已开始导出 ${total.value} 条用户资料`);
  } catch (e: any) {
    ElMessage.error(e.message || "导出用户资料失败");
  } finally {
    exportingFilteredUsers.value = false;
  }
}

async function fetchUsers(refreshOnlineStatus = true) {
  loading.value = true;
  try {
    const res = await getUserList(buildParams());
    users.value = res.data.data?.list ?? [];
    total.value = res.data.data?.total ?? 0;
    if (refreshOnlineStatus) {
      await refreshStatus(false);
    }
  } catch (e: any) {
    ElMessage.error(e.message || "获取用户列表失败");
  } finally {
    loading.value = false;
  }
}
onMounted(() => {
  if (isAdmin.value) {
    void loadUserOwners();
  }
});

watch(
  () => route.fullPath,
  () => {
    if (route.name !== "Users") return;
    restoreUsersStateFromRoute();
    const canonicalPath = router.resolve({
      name: "Users",
      query: buildUsersRouteQuery(),
    }).fullPath;
    if (canonicalPath !== route.fullPath) {
      void router.replace(canonicalPath);
      return;
    }
    void fetchUsers();
  },
  { immediate: true },
);

function ownerLabel(row: AppUser) {
  if (row.ownerState === "unavailable") return "归属不可确认";
  if (row.ownerState === "pool") return "公海";
  return row.ownerName || "—";
}

function statusLabel(row: AppUser) {
  if (row.status === "deleted")
    return { cls: "status-deleted", text: "已注销" };
  if (row.loginCooldown) return { cls: "status-cooldown", text: "冷却中" };
  if (row.loginReason === "owner_banned")
    return { cls: "status-banned", text: "归属已禁用" };
  if (row.status === "banned") return { cls: "status-banned", text: "已封禁" };
  if (!row.isActivated) return { cls: "status-pending", text: "未激活" };
  return { cls: "status-active", text: "正常" };
}

function statusTitle(row: AppUser) {
  const label = statusLabel(row);
  if (!row.loginCooldown || label.text !== "冷却中") return label.text;
  const parts = ["账号登录错误次数过多，当前处于冷却中"];
  if (row.loginCooldownUntil) parts.push(`冷却到：${row.loginCooldownUntil}`);
  if (row.loginCooldownReason) parts.push(row.loginCooldownReason);
  return parts.join("；");
}

function cleanupStatusLabel(row: AppUser) {
  switch (row.cleanupStatus) {
    case "auto_banned":
      return { cls: "status-banned", text: "自动封禁" };
    case "manual_banned":
      return { cls: "status-banned", text: "手动封禁" };
    case "whitelisted":
      return { cls: "status-info", text: "清理白名单" };
    case "restored":
      return { cls: "status-active", text: "已恢复" };
    default:
      return { cls: "status-inactive", text: "未处置" };
  }
}

async function copyInviteCode(code: string) {
  await navigator.clipboard.writeText(code);
  ElMessage.success("已复制");
}

async function handleKick(user: AppUser) {
  await ElMessageBox.confirm(
    `确认将用户 "${user.nickname}" 踢出登录？`,
    "确认操作",
    { type: "warning" },
  );
  try {
    await kickUser(user.id);
    ElMessage.success("已踢出");
  } catch (e: any) {
    ElMessage.error(e.message || "操作失败");
  }
}

async function handleResolveLoginCooldown(user: AppUser) {
  if (loginCooldownResolvingIds.value.has(user.id)) return;
  const untilText = user.loginCooldownUntil ? `\n冷却到：${user.loginCooldownUntil}` : "";
  await ElMessageBox.confirm(
    `确认解除用户 "${user.nickname}" 的登录冷却？${untilText}`,
    "解除登录冷却",
    { type: "warning" },
  );
  loginCooldownResolvingIds.value = new Set(loginCooldownResolvingIds.value).add(user.id);
  try {
    await resolveUserLoginCooldown(user.id);
    ElMessage.success("已解除登录冷却");
    await fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "解除登录冷却失败");
  } finally {
    const next = new Set(loginCooldownResolvingIds.value);
    next.delete(user.id);
    loginCooldownResolvingIds.value = next;
  }
}
async function handleBan(user: AppUser) {
  const isBanned = user.status === "banned";
  await ElMessageBox.confirm(
    `确认${isBanned ? "解封" : "封禁"}用户 "${user.nickname}"？`,
    "确认操作",
    { type: "warning" },
  );
  try {
    if (isBanned) await unbanUser(user.id);
    else await banUser(user.id);
    ElMessage.success(isBanned ? "已解封" : "已封禁");
    fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "操作失败");
  }
}

function banActionCopy(row: AppUser) {
  if (row.loginReason === "owner_banned") {
    return {
      titleClass: "dropdown-item-default",
      title: "归属待解封",
      desc: "需先解封归属助理，当前用户才能恢复登录",
    };
  }
  if (row.status === "banned") {
    return {
      titleClass: "dropdown-item-success",
      title: "解封",
      desc: "恢复账号正常使用",
    };
  }
  return {
    titleClass: "dropdown-item-warning",
    title: "封禁",
    desc: "禁止账号继续使用",
  };
}

function isDeletedUser(row: AppUser) {
  return row.status === "deleted" || row.accountStatus === "deleted";
}

function ensureNoDeletedSelection() {
  if (selectedUsers.value.some(isDeletedUser)) {
    ElMessage.warning("已注销用户不能执行该操作，请先取消选择已注销用户");
    return false;
  }
  return true;
}

async function handleDelete(user: AppUser) {
  let preview: UserDeletePreview | null = null;
  try {
    const res = await getUserDeletePreview(user.id);
    preview = res.data.data;
  } catch (e: any) {
    ElMessage.error(e.message || "获取注销预览失败");
    return;
  }

  const impactLines = [
    preview?.operationDetail || "系统会将账号标记为已注销，撤销所有登录态并踢下线。",
    `本地业务数据：${preview?.willDeleteLocalData ? "删除" : "保留"}`,
    `腾讯 IM 账号：${preview?.willDeleteTencentIM ? "删除" : "保留"}`,
    `在线状态：${preview?.willKickOnline ? "踢下线" : "不处理"}`,
  ];

  await ElMessageBox.confirm(
    [
      `确认软注销用户 "${user.nickname}"？`,
      "",
      ...impactLines,
    ].join("<br>"),
    "危险操作",
    {
      type: "error",
      dangerouslyUseHTMLString: true,
      confirmButtonText: "确认注销",
      cancelButtonText: "取消",
    },
  );

  try {
    await deleteUser(user.id);
    ElMessage.success("已注销");
    fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "操作失败");
  }
}

async function handleToggleCreateGroupPermission(user: AppUser) {
  const nextEnabled = !user.canCreateGroup;
  await ElMessageBox.confirm(
    `确认${nextEnabled ? "允许" : "禁止"}用户 "${user.nickname}" 建群？`,
    "确认操作",
    { type: "warning" },
  );
  try {
    await batchSetUserCreateGroupPermission([user.id], nextEnabled);
    ElMessage.success(nextEnabled ? "已允许建群" : "已禁止建群");
    await fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "操作失败");
  }
}

async function handleToggleGlobalSearch(user: AppUser) {
  if (isDeletedUser(user)) return;
  const enabled = !user.globallySearchable;
  try {
    await ElMessageBox.confirm(`确认${enabled ? '开启' : '关闭'}用户“${user.nickname}”的全局搜索？`, '全局搜索', { type: 'warning' });
    const res = await batchSetUserGlobalSearch([user.id], enabled);
    if (res.data.data.failed > 0) {
      ElMessage.error('操作失败，请确认该用户在你的管理范围内');
      return;
    }
    ElMessage.success(enabled ? '已开启全局搜索' : '已关闭全局搜索');
    await fetchUsers();
  } catch (e: any) {
    if (e !== 'cancel' && e !== 'close') ElMessage.error(e.message || '操作失败');
  }
}

async function handleEnableUserTOTP(user: AppUser) {
  await ElMessageBox.confirm(
    `确认开启用户 "${user.nickname}" 的 Google 验证？开启后该用户下次 Web 登录需绑定或校验动态码，App 登录暂不受影响。`,
    "开启 Google 验证",
    {
      type: "warning",
      confirmButtonText: "确认开启",
      cancelButtonText: "取消",
    },
  );
  try {
    await enableUserTOTP(user.id);
    ElMessage.success("已开启 Google 验证");
    await fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "操作失败");
  }
}

async function handleDisableUserTOTP(user: AppUser) {
  await ElMessageBox.confirm(
    `确认关闭用户 "${user.nickname}" 的 Google 验证？关闭后会撤销该用户当前 Web 登录态。`,
    "关闭 Google 验证",
    {
      type: "warning",
      confirmButtonText: "确认关闭",
      cancelButtonText: "取消",
    },
  );
  try {
    await disableUserTOTP(user.id);
    ElMessage.success("已关闭 Google 验证");
    await fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "操作失败");
  }
}

async function handleResetUserRealname(user: AppUser) {
  await ElMessageBox.confirm(
    `确认重置用户 "${user.nickname}" 的实名认证状态？重置后用户需重新完成实名核验。`,
    "重置实名认证",
    {
      type: "warning",
      confirmButtonText: "确认重置",
      cancelButtonText: "取消",
    },
  );
  try {
    await resetUserRealname(user.id);
    ElMessage.success("已重置实名认证");
    await fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "操作失败");
  }
}

const imRepairPending = ref<number[]>([]);
async function handleIMRepair(row: AppUser) {
  if (imRepairPending.value.includes(row.id)) return;
  imRepairPending.value.push(row.id);
  try {
    const application = await getIMApplication();
    const sdkAppId = application.data.data.sdk_app_id;
    await ElMessageBox.confirm(`将用户 ${row.im_user_id} 同步到当前 IM 应用 ${sdkAppId}。请确认本应用和业务归属正确。`, '同步聊天账号');
    await repairUserIM(row.id, sdkAppId);
    ElMessage.success('腾讯账号已同步并读回确认');
  } catch (error) { if (error !== 'cancel' && error !== 'close') ElMessage.error(error instanceof Error ? error.message : '同步失败'); }
  finally { imRepairPending.value = imRepairPending.value.filter(id => id !== row.id); }
}

function handleRowAction(command: { action: string; row: AppUser }) {
  const { action, row } = command;
  switch (action) {
    case "imRepair":
      void handleIMRepair(row);
      return;
    case "conversation":
      openConversationShortcut(row);
      return;
    case "whitelist":
      openWhitelistModal(row);
      return;
    case "assign":
      openAssignModal(row);
      return;
    case "kick":
      handleKick(row);
      return;
    case "resolveLoginCooldown":
      handleResolveLoginCooldown(row);
      return;
    case "ban":
      handleBan(row);
      return;
    case "globalSearch":
      void handleToggleGlobalSearch(row);
      return;
    case "groupPermission":
      handleToggleCreateGroupPermission(row);
      return;
    case "enableTotp":
      handleEnableUserTOTP(row);
      return;
    case "disableTotp":
      handleDisableUserTOTP(row);
      return;
    case "resetRealname":
      handleResetUserRealname(row);
      return;
    case "delete":
      handleDelete(row);
      return;
    default:
      return;
  }
}

// ── 分配弹窗 ──
const showAssignModal = ref(false);
const assignTargetUser = ref<AppUser | null>(null);
const assignTargetCount = ref(0);
const assignOwnerID = ref<number>(0);
const assignSubmitting = ref(false);

function ensureOwnerOptionsLoaded() {
  if (isAdmin.value && ownerOptions.value.length === 0) {
    void loadUserOwners();
  }
}

function openAssignModal(user: AppUser) {
  assignTargetUser.value = user;
  assignTargetCount.value = 1;
  assignOwnerID.value = user.ownerID;
  showAssignModal.value = true;
  ensureOwnerOptionsLoaded();
}

function openBatchAssignModal() {
  if (selectedUsers.value.length === 0) return;
  assignTargetUser.value = null;
  assignTargetCount.value = selectedUsers.value.length;
  assignOwnerID.value = defaultManagedOwnerID.value;
  showAssignModal.value = true;
  ensureOwnerOptionsLoaded();
}

async function submitAssign() {
  if (assignOwnerID.value === 0) return;
  assignSubmitting.value = true;
  try {
    if (assignTargetUser.value) {
      await transferUser(assignTargetUser.value.id, assignOwnerID.value);
    } else {
      await batchTransferUsers(
        selectedUsers.value.map((user) => user.id),
        assignOwnerID.value,
      );
    }
    ElMessage.success("分配成功");
    showAssignModal.value = false;
    fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "分配失败");
  } finally {
    assignSubmitting.value = false;
  }
}

// ── 新建弹窗 ──
const showCreateModal = ref(false);
const createTab = ref<"manual" | "file" | "template">("manual");

function openCreateModal() {
  showCreateModal.value = true;
  createTab.value = "manual";
  resetManual();
  resetFile();
  resetTemplate();
  ensureOwnerOptionsLoaded();
}

// ── 手动录入 ──
interface ManualRow {
  phone: string;
  password: string;
  nickname: string;
  ownerID: number;
  result: { success: boolean; message: string } | null;
}

function buildManualRow(): ManualRow {
  return {
    phone: "",
    password: "",
    nickname: "",
    ownerID: defaultManagedOwnerID.value,
    result: null,
  };
}

const manualRows = ref<ManualRow[]>([buildManualRow()]);
const manualSubmitted = ref(false);
const manualSubmitting = ref(false);

function addRow() {
  manualRows.value.push(buildManualRow());
}
function removeRow(i: number) {
  if (manualRows.value.length === 1) return;
  manualRows.value.splice(i, 1);
}
function resetManual() {
  manualRows.value = [buildManualRow()];
  manualSubmitted.value = false;
}

async function submitManual() {
  const invalid = manualRows.value.some(
    (r) => !r.phone.trim() || !r.password.trim(),
  );
  if (invalid) {
    ElMessage.warning("请填写所有必填字段（手机号、初始密码）");
    return;
  }
  const invalidPhone = manualRows.value.find((r) => !isValidPhone(r.phone));
  if (invalidPhone) {
    ElMessage.warning(`手机号格式无效：${invalidPhone.phone}`);
    return;
  }

  manualSubmitting.value = true;
  manualSubmitted.value = true;
  try {
    const payload = manualRows.value.map((r) => ({
      phone: r.phone.trim(),
      password: r.password,
      nickname: r.nickname.trim() || undefined,
      owner_id: r.ownerID || undefined,
    }));
    const res = await batchCreateUsers(payload);
    const results = res.data.data ?? [];
    results.forEach((r, i) => {
      if (manualRows.value[i])
        manualRows.value[i].result = { success: r.success, message: r.message };
    });
    const ok = results.filter((r) => r.success).length;
    ElMessage.success(`创建完成：${ok} / ${results.length} 成功`);
    if (ok > 0) fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "提交失败");
  } finally {
    manualSubmitting.value = false;
  }
}

// ── 文件导入 ──
interface PreviewRow {
  phone: string;
  password: string;
  nickname: string;
  result: { success: boolean; message: string } | null;
}

const fileInputRef = ref<HTMLInputElement | null>(null);
const csvFile = ref<File | null>(null);
const previewRows = ref<PreviewRow[]>([]);
const fileSubmitted = ref(false);
const fileSubmitting = ref(false);

function resetFile() {
  csvFile.value = null;
  previewRows.value = [];
  fileSubmitted.value = false;
  if (fileInputRef.value) fileInputRef.value.value = "";
}

function triggerFileInput() {
  fileInputRef.value?.click();
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  if (!file.name.endsWith(".csv")) {
    ElMessage.error("仅支持 .csv 格式");
    return;
  }
  csvFile.value = file;
  fileSubmitted.value = false;

  const reader = new FileReader();
  reader.onload = (ev) => {
    const text = ev.target?.result as string;
    const lines = text
      .split("\n")
      .map((l) => l.trim())
      .filter(Boolean);
    const dataLines = lines.slice(1); // 跳过表头
    previewRows.value = dataLines
      .map((line) => {
        const cols = line.split(",").map((c) => c.trim().replace(/^"|"$/g, ""));
        return {
          phone: cols[0] ?? "",
          password: cols[1] ?? "",
          nickname: cols[2] ?? "",
          result: null,
        };
      })
      .filter((r) => r.phone);
    if (!previewRows.value.length) ElMessage.warning("CSV 文件没有有效数据行");
  };
  reader.readAsText(file, "UTF-8");
}

async function submitFile() {
  if (!previewRows.value.length) return;
  const invalidPhone = previewRows.value.find((r) => !isValidPhone(r.phone));
  if (invalidPhone) {
    ElMessage.warning(`手机号格式无效：${invalidPhone.phone}`);
    return;
  }
  fileSubmitting.value = true;
  fileSubmitted.value = true;
  try {
    const payload = previewRows.value.map((r) => ({
      phone: r.phone,
      password: r.password,
      nickname: r.nickname || undefined,
    }));
    const res = await batchCreateUsers(payload);
    const results = res.data.data ?? [];
    results.forEach((r, i) => {
      if (previewRows.value[i])
        previewRows.value[i].result = {
          success: r.success,
          message: r.message,
        };
    });
    const ok = results.filter((r) => r.success).length;
    ElMessage.success(`导入完成：${ok} / ${results.length} 成功`);
    if (ok > 0) fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "提交失败");
  } finally {
    fileSubmitting.value = false;
  }
}

function downloadTemplate() {
  const content =
    "手机号,初始密码,昵称(选填)\n13800000001,password123,示例用户";
  const blob = new Blob(["\uFEFF" + content], {
    type: "text/csv;charset=utf-8;",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "users_template.csv";
  a.click();
  URL.revokeObjectURL(url);
}

// ── 模板生成 ──
interface TemplatePreviewRow {
  index: number;
  phone: string;
  nickname: string;
  password: string;
}

const templateCountPresets = [10, 20, 50, 100];
const passwordModeOptions: Array<{
  value: GeneratedPasswordMode;
  label: string;
  desc: string;
}> = [
  {
    value: "random_strong",
    label: "完全随机",
    desc: "每个账号生成独立随机密码，更安全",
  },
  {
    value: "fixed",
    label: "统一固定密码",
    desc: "所有账号使用同一个初始密码，风险更高",
  },
  {
    value: "prefix_random",
    label: "固定前缀 + 随机后缀",
    desc: "统一前缀，后缀随机，兼顾可记忆与安全性",
  },
];

const templateForm = reactive({
  count: 10,
  ownerID: 0,
  realNameVerified: true,
  canCreateGroup: false,
  phonePrefix: "139",
  phoneMiddleMode: "sequence" as "sequence" | "random",
  phoneMiddleStart: 1,
  phoneSuffix: "1234",
  nicknamePrefix: "测试用户",
  nicknameStart: 1,
  nicknamePadding: 3,
  passwordMode: "random_strong" as GeneratedPasswordMode,
  fixedPassword: "",
  passwordPrefix: "Chengnan@",
  passwordRandomLength: 4,
  randomPasswordLength: 10,
});

const templateSubmitting = ref(false);
const templateSubmitted = ref(false);
const templateResultRows = ref<GenerateUsersResultItem[]>([]);
const templateSummary = reactive({
  requested: 0,
  success: 0,
  failed: 0,
});

const selectedTemplateOwnerLabel = computed(() => {
  if (!isAdmin.value) {
    return defaultManagedOwnerLabel.value;
  }
  return (
    ownerManagementOptions.value.find(
      (option) => option.value === templateForm.ownerID,
    )?.label ?? defaultManagedOwnerLabel.value
  );
});

function resetTemplate() {
  templateForm.count = 10;
  templateForm.ownerID = defaultManagedOwnerID.value;
  templateForm.realNameVerified = true;
  templateForm.canCreateGroup = false;
  templateForm.phonePrefix = "139";
  templateForm.phoneMiddleMode = "sequence";
  templateForm.phoneMiddleStart = 1;
  templateForm.phoneSuffix = "1234";
  templateForm.nicknamePrefix = "测试用户";
  templateForm.nicknameStart = 1;
  templateForm.nicknamePadding = 3;
  templateForm.passwordMode = "random_strong";
  templateForm.fixedPassword = "";
  templateForm.passwordPrefix = "Chengnan@";
  templateForm.passwordRandomLength = 4;
  templateForm.randomPasswordLength = 10;
  templateSubmitted.value = false;
  templateResultRows.value = [];
  templateSummary.requested = 0;
  templateSummary.success = 0;
  templateSummary.failed = 0;
}

function padNumber(value: number, width: number) {
  return String(value).padStart(width, "0");
}

function previewRandomDigits(seed: number, width: number) {
  const base = Math.abs((seed * 7919 + 1237) % Math.pow(10, width));
  return String(base).padStart(width, "0");
}

function previewRandomPassword(seed: number, length: number) {
  const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789";
  let result = "";
  for (let i = 0; i < length; i++) {
    result += alphabet[(seed * 17 + i * 31 + 7) % alphabet.length];
  }
  return result;
}

function previewPhone(index: number) {
  const middle =
    templateForm.phoneMiddleMode === "sequence"
      ? padNumber(templateForm.phoneMiddleStart + index, 4)
      : previewRandomDigits(index + 1, 4);
  return `${templateForm.phonePrefix}${middle}${templateForm.phoneSuffix}`;
}

function previewNickname(index: number) {
  const sequence = templateForm.nicknameStart + index;
  const sequenceText =
    templateForm.nicknamePadding > 0
      ? padNumber(sequence, templateForm.nicknamePadding)
      : String(sequence);
  return `${templateForm.nicknamePrefix}${sequenceText}`;
}

function previewPassword(index: number) {
  switch (templateForm.passwordMode) {
    case "fixed":
      return templateForm.fixedPassword || "请填写固定密码";
    case "prefix_random":
      return `${templateForm.passwordPrefix}${previewRandomPassword(index + 3, templateForm.passwordRandomLength)}`;
    case "random_strong":
    default:
      return previewRandomPassword(
        index + 5,
        templateForm.randomPasswordLength,
      );
  }
}

const templatePreviewRows = computed<TemplatePreviewRow[]>(() => {
  const count = Math.max(1, Math.min(templateForm.count || 1, 5));
  return Array.from({ length: count }, (_, index) => ({
    index: index + 1,
    phone: previewPhone(index),
    nickname: previewNickname(index),
    password: previewPassword(index),
  }));
});

function validateTemplateForm() {
  if (
    !Number.isInteger(templateForm.count) ||
    templateForm.count < 1 ||
    templateForm.count > 200
  ) {
    ElMessage.warning("生成数量需在 1~200 之间");
    return false;
  }
  if (!/^1[3-9]\d$/.test(templateForm.phonePrefix)) {
    ElMessage.warning("手机号前三位格式无效");
    return false;
  }
  if (!/^\d{4}$/.test(templateForm.phoneSuffix)) {
    ElMessage.warning("手机号后四位必须为 4 位数字");
    return false;
  }
  if (templateForm.phoneMiddleMode === "sequence") {
    if (
      !Number.isInteger(templateForm.phoneMiddleStart) ||
      templateForm.phoneMiddleStart < 0 ||
      templateForm.phoneMiddleStart > 9999
    ) {
      ElMessage.warning("手机号中间四位起始值需在 0000~9999 之间");
      return false;
    }
    if (templateForm.phoneMiddleStart + templateForm.count - 1 > 9999) {
      ElMessage.warning("手机号模板剩余可用空间不足，请调整中间四位起始值");
      return false;
    }
  }
  if (!templateForm.nicknamePrefix.trim()) {
    ElMessage.warning("昵称前缀不能为空");
    return false;
  }
  if (
    !Number.isInteger(templateForm.nicknameStart) ||
    templateForm.nicknameStart < 0
  ) {
    ElMessage.warning("昵称起始编号不能小于 0");
    return false;
  }
  if (
    !Number.isInteger(templateForm.nicknamePadding) ||
    templateForm.nicknamePadding < 0 ||
    templateForm.nicknamePadding > 8
  ) {
    ElMessage.warning("昵称补零位数需在 0~8 之间");
    return false;
  }

  if (templateForm.passwordMode === "fixed") {
    if (templateForm.fixedPassword.length < 8) {
      ElMessage.warning("固定密码至少需要 8 位");
      return false;
    }
  }
  if (templateForm.passwordMode === "prefix_random") {
    if (!templateForm.passwordPrefix.trim()) {
      ElMessage.warning("密码固定前缀不能为空");
      return false;
    }
    if (
      !Number.isInteger(templateForm.passwordRandomLength) ||
      templateForm.passwordRandomLength < 1 ||
      templateForm.passwordRandomLength > 24
    ) {
      ElMessage.warning("密码随机后缀长度需在 1~24 之间");
      return false;
    }
    if (
      templateForm.passwordPrefix.length + templateForm.passwordRandomLength <
      8
    ) {
      ElMessage.warning("密码总长度至少需要 8 位");
      return false;
    }
  }
  if (templateForm.passwordMode === "random_strong") {
    if (
      !Number.isInteger(templateForm.randomPasswordLength) ||
      templateForm.randomPasswordLength < 8 ||
      templateForm.randomPasswordLength > 32
    ) {
      ElMessage.warning("随机密码长度需在 8~32 位之间");
      return false;
    }
  }
  return true;
}

function buildGenerateUsersPayload(): GenerateUsersParams {
  return {
    count: templateForm.count,
    owner_id: isAdmin.value ? templateForm.ownerID || undefined : undefined,
    real_name_verified: templateForm.realNameVerified,
    can_create_group: templateForm.canCreateGroup,
    phone_template: {
      prefix: templateForm.phonePrefix.trim(),
      middle_mode: templateForm.phoneMiddleMode,
      middle_start: templateForm.phoneMiddleStart,
      suffix: templateForm.phoneSuffix.trim(),
    },
    nickname_template: {
      prefix: templateForm.nicknamePrefix.trim(),
      start: templateForm.nicknameStart,
      padding: templateForm.nicknamePadding,
    },
    password_template: {
      mode: templateForm.passwordMode,
      fixed_value:
        templateForm.passwordMode === "fixed"
          ? templateForm.fixedPassword
          : undefined,
      prefix:
        templateForm.passwordMode === "prefix_random"
          ? templateForm.passwordPrefix.trim()
          : undefined,
      random_length:
        templateForm.passwordMode === "prefix_random"
          ? templateForm.passwordRandomLength
          : templateForm.passwordMode === "random_strong"
            ? templateForm.randomPasswordLength
            : undefined,
    },
  };
}

async function submitTemplateGeneration() {
  if (!validateTemplateForm()) {
    return;
  }

  if (templateForm.passwordMode === "fixed") {
    try {
      await ElMessageBox.confirm(
        "当前将为全部新账号使用同一个初始密码。虽然系统会要求首次登录改密，但仍存在较高泄露风险，确认继续吗？",
        "统一固定密码提醒",
        {
          type: "warning",
          confirmButtonText: "继续生成",
          cancelButtonText: "取消",
        },
      );
    } catch (_) {
      return;
    }
  }

  templateSubmitting.value = true;
  templateSubmitted.value = true;
  try {
    const res = await generateUsers(buildGenerateUsersPayload());
    const data = res.data.data;
    templateResultRows.value = data?.items ?? [];
    templateSummary.requested = data?.requested ?? 0;
    templateSummary.success = data?.success ?? 0;
    templateSummary.failed = data?.failed ?? 0;
    if (templateSummary.failed > 0) {
      ElMessage.warning(
        `生成完成：成功 ${templateSummary.success} 个，失败 ${templateSummary.failed} 个`,
      );
    } else {
      ElMessage.success(
        `生成完成：成功 ${templateSummary.success} 个，失败 ${templateSummary.failed} 个`,
      );
    }
    if (templateSummary.success > 0) {
      void fetchUsers();
    }
  } catch (e: any) {
    ElMessage.error(e.message || "模板生成失败");
  } finally {
    templateSubmitting.value = false;
  }
}

async function copyGeneratedAccounts() {
  const successRows = templateResultRows.value.filter((row) => row.success);
  if (successRows.length === 0) {
    ElMessage.warning("暂无可复制的成功记录");
    return;
  }
  const content = [
    "手机号,初始密码,昵称,IM用户ID",
    ...successRows.map((row) =>
      [row.phone, row.password, row.nickname, row.im_user_id].join(","),
    ),
  ].join("\n");
  await navigator.clipboard.writeText(content);
  ElMessage.success(`已复制 ${successRows.length} 条账号信息`);
}

function downloadGeneratedAccounts() {
  if (templateResultRows.value.length === 0) {
    ElMessage.warning("暂无可导出的记录");
    return;
  }
  const content = [
    "序号,手机号,初始密码,昵称,IM用户ID,结果",
    ...templateResultRows.value.map((row) =>
      [
        row.index + 1,
        row.phone,
        row.password,
        row.nickname,
        row.im_user_id || "",
        row.message,
      ].join(","),
    ),
  ].join("\n");
  const blob = new Blob(["\uFEFF" + content], {
    type: "text/csv;charset=utf-8;",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `generated_users_${Date.now()}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

// ── 用户同步 ──
const showSyncModal = ref(false);
const syncLoading = ref(false);
const syncSubmitting = ref(false);
const syncPreviews = ref<SyncUserPreview[]>([]);

const syncStatusMap: Record<SyncUserStatus, { text: string; cls: string }> = {
  missing_in_im: { text: "IM 缺失", cls: "sync-status-missing" },
  nickname_changed: { text: "昵称有差异", cls: "sync-status-nickname" },
  avatar_changed: { text: "头像有差异", cls: "sync-status-avatar" },
  profile_changed: { text: "资料有差异", cls: "sync-status-changed" },
  changed: { text: "多项资料有差异", cls: "sync-status-changed" },
  synced: { text: "已同步", cls: "sync-status-synced" },
};

function syncStatusLabel(status: SyncUserStatus) {
  return syncStatusMap[status];
}

function displayValue(value: string) {
  return value || "—";
}

async function openSyncModal() {
  showSyncModal.value = true;
  syncPreviews.value = [];
  syncLoading.value = true;
  try {
    const res = await syncUsersPreview();
    syncPreviews.value = (res.data.data ?? []).map((item) => ({
      ...item,
      action:
        item.status === "missing_in_im"
          ? ("skip" as const)
          : ("update" as const),
    }));
  } catch (e: any) {
    ElMessage.error(e.message || "拉取用户差异失败");
    showSyncModal.value = false;
  } finally {
    syncLoading.value = false;
  }
}

function toggleAction(item: SyncUserPreview) {
  if (item.status === "missing_in_im") {
    item.action = "skip";
    return;
  }
  item.action = item.action === "skip" ? "update" : "skip";
}

async function submitSync() {
  const selected = syncPreviews.value.filter((p) => p.action === "update");
  if (selected.length === 0) {
    ElMessage.warning("请至少选择一个差异用户进行更新");
    return;
  }
  await ElMessageBox.confirm(
    `确认同步 ${selected.length} 个用户资料？`,
    "确认操作",
    { type: "warning" },
  );
  syncSubmitting.value = true;
  try {
    const res = await syncUsersConfirm(
      selected.map((p) => ({ imUserID: p.imUserID, action: p.action })),
    );
    const { success, failed } = res.data.data ?? { success: 0, failed: 0 };
    ElMessage.success(`同步完成：成功 ${success} 个，失败 ${failed} 个`);
    showSyncModal.value = false;
    fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "同步失败");
  } finally {
    syncSubmitting.value = false;
  }
}

async function handleBatchBan(nextBanned: boolean) {
  if (selectedUsers.value.length === 0) {
    ElMessage.warning("请先选择用户");
    return;
  }
  if (!ensureNoDeletedSelection()) return;

  const actionLabel = nextBanned ? "封禁" : "解封";
  await ElMessageBox.confirm(
    `确认对已选 ${selectedUsers.value.length} 个用户执行${actionLabel}？`,
    "确认操作",
    { type: "warning" },
  );

  try {
    const userIDs = selectedUsers.value.map((user) => user.id);
    const res = nextBanned
      ? await batchBanUsers(userIDs)
      : await batchUnbanUsers(userIDs);
    const data = res.data.data;
    const failedDetails = (data.details ?? [])
      .filter((item) => !item.success)
      .slice(0, 3)
      .map((item) => `${item.nickname || item.userID}: ${item.message}`)
      .join("；");
    const summary = `成功 ${data.success} 个，失败 ${data.failed} 个`;
    ElMessage[data.failed > 0 ? "warning" : "success"](
      `${actionLabel}完成：${summary}${failedDetails ? `；${failedDetails}` : ""}`,
    );
    fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || `${actionLabel}失败`);
  }
}

async function handleBatchDelete() {
  if (selectedUsers.value.length === 0) {
    ElMessage.warning("请先选择用户");
    return;
  }
  if (!ensureNoDeletedSelection()) return;

  await ElMessageBox.confirm(
    `确认软注销已选 ${selectedUsers.value.length} 个用户？系统会将账号标记为已注销，撤销登录态并踢下线；本地业务数据和腾讯 IM 账号不会删除。`,
    "确认批量注销",
    { type: "warning" },
  );

  try {
    const userIDs = selectedUsers.value.map((user) => user.id);
    const res = await batchDeleteUsers(userIDs);
    const data = res.data.data;
    const failedDetails = (data.details ?? [])
      .filter((item) => !item.success)
      .slice(0, 3)
      .map((item) => `${item.nickname || item.userID}: ${item.message}`)
      .join("；");
    const summary = `成功 ${data.success} 个，失败 ${data.failed} 个`;
    ElMessage[data.failed > 0 ? "warning" : "success"](
      `批量注销完成：${summary}${failedDetails ? `；${failedDetails}` : ""}`,
    );
    fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "批量注销失败");
  }
}

async function handleBatchCreateGroupPermission(enabled: boolean) {
  if (selectedUsers.value.length === 0) {
    ElMessage.warning("请先选择用户");
    return;
  }
  if (!ensureNoDeletedSelection()) return;

  const actionLabel = enabled ? "开启" : "关闭";
  await ElMessageBox.confirm(
    `确认对已选 ${selectedUsers.value.length} 个用户${actionLabel}建群权限？`,
    "确认操作",
    { type: "warning" },
  );

  try {
    const userIDs = selectedUsers.value.map((user) => user.id);
    await batchSetUserCreateGroupPermission(userIDs, enabled);
    ElMessage.success(enabled ? "已批量开启建群权限" : "已批量关闭建群权限");
    await fetchUsers();
  } catch (e: any) {
    ElMessage.error(e.message || "操作失败");
  }
}

function openConversationShortcut(user: AppUser) {
  const from = router.resolve({
    name: "Users",
    query: buildUsersRouteQuery(),
  }).fullPath;
  router.push({
    name: "Conversations",
    query: {
      mode: "user",
      userID: user.imUserID,
      nickname: user.nickname,
      phone: user.phone,
      from,
    },
  });
}

function whitelistStatusLabel(row: AppUser) {
  if (!row.ipWhitelistEnabled) {
    return { cls: "status-pending", text: "未启用" };
  }
  return { cls: "status-active", text: "已启用" };
}
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">用户管理</h1>
          <p class="page-sub">{{ pageSubText }}</p>
        </div>
        <div class="page-header-actions">
          <el-button @click="refreshStatus" :loading="statusRefreshing">
            <template #icon
              ><el-icon><Refresh /></el-icon></template
            >刷新状态
          </el-button>
          <el-button v-if="isAdmin" @click="openSyncModal">
            <template #icon
              ><el-icon><Refresh /></el-icon></template
            >同步
          </el-button>
          <el-button type="primary" @click="openCreateModal">
            <template #icon
              ><el-icon><Plus /></el-icon></template
            >新建用户
          </el-button>
        </div>
      </div>

      <!-- 筛选栏 -->
      <div class="card-toolbar">
        <div ref="filterWrapRef" class="filter-wrap">
          <div class="filter-box">
            <svg
              class="filter-icon"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            <span v-for="f in activeFilters" :key="f.key" class="filter-tag">
              <span class="filter-tag-key">{{ f.label }}:</span>{{ f.display }}
              <button
                class="filter-tag-remove"
                @click.stop="removeFilter(f.key)"
              >
                ×
              </button>
            </span>
            <input
              v-model="searchKeyword"
              class="filter-search-input"
              type="text"
              placeholder="搜索昵称 / 手机号 / IM用户ID"
              @keyup.enter="applySearch"
              @click.stop
            />
            <div class="filter-actions">
              <button
                v-if="searchKeyword"
                class="filter-inline-btn"
                @click.stop="clearSearch"
              >
                清空
              </button>
              <button
                v-if="activeFilters.length > 0"
                class="filter-inline-btn"
                @click.stop="clearAllFilters"
              >
                重置筛选
              </button>
              <button
                class="filter-inline-btn filter-inline-btn-primary"
                @click.stop="applySearch"
              >
                搜索
              </button>
              <button class="filter-inline-btn" @click.stop="openDropdown">
                筛选{{
                  activeFilters.length > 0 ? `(${activeFilters.length})` : ""
                }}
              </button>
            </div>
          </div>
          <div
            class="filter-helper"
            :class="{ 'mobile-only-helper': isMobile }"
          >
            搜索支持：昵称 / 手机号 / 信号 / IMUserID
            <span class="filter-helper-divider">|</span>
            筛选支持：{{
              isAdmin
                ? "归属昵称 / 账号状态 / 激活状态 / 白名单状态 / 当前登录 IP"
                : "账号状态 / 激活状态 / 白名单状态 / 当前登录 IP"
            }}
          </div>
          <div v-if="dropdownOpen" class="filter-dropdown">
            <template v-if="!selectedField">
              <div class="filter-dropdown-head">结构化筛选</div>
              <ul>
                <li
                  v-for="f in FILTER_FIELDS"
                  :key="f.key"
                  class="filter-field-item"
                  @click.stop="selectField(f)"
                >
                  <div class="filter-field-icon">
                    <svg
                      width="12"
                      height="12"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2a1 1 0 01-.293.707L13 13.414V19a1 1 0 01-.553.894l-4 2A1 1 0 017 21v-7.586L3.293 6.707A1 1 0 013 6V4z"
                      />
                    </svg>
                  </div>
                  <span>{{ f.label }}</span>
                  <svg
                    class="filter-chevron"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M9 5l7 7-7 7"
                    />
                  </svg>
                </li>
              </ul>
            </template>
            <template v-else>
              <div class="filter-value-head">
                <button
                  class="filter-back-btn"
                  @click.stop="selectedField = null"
                >
                  <svg
                    width="16"
                    height="16"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M15 19l-7-7 7-7"
                    />
                  </svg>
                </button>
                <span>{{ selectedField.label }}</span>
              </div>
              <div class="filter-value-body">
                <input
                  v-if="selectedField.type === 'text'"
                  v-model="filterInput"
                  class="filter-text-input"
                  :placeholder="`输入${selectedField.label}…`"
                  autofocus
                  @keyup.enter.stop="confirmFilter"
                  @click.stop
                />
                <ul v-else class="filter-option-list">
                  <li
                    v-if="selectedFieldOptions.length === 0"
                    class="filter-option filter-option-empty"
                  >
                    暂无可选项
                  </li>
                  <li
                    v-for="opt in selectedFieldOptions"
                    :key="opt.value"
                    class="filter-option"
                    :class="{ selected: filterSelect === opt.value }"
                    @click.stop="filterSelect = opt.value"
                  >
                    <div
                      class="filter-radio"
                      :class="{ active: filterSelect === opt.value }"
                    ></div>
                    <span>{{ opt.label }}</span>
                  </li>
                </ul>
                <button class="filter-confirm-btn" @click.stop="confirmFilter">
                  确认
                </button>
              </div>
            </template>
          </div>
        </div>
        <div class="users-toolbar-right">
          <el-button
            size="small"
            plain
            :loading="exportingFilteredUsers"
            @click="handleExportFilteredUsers"
          >
            <template #icon
              ><el-icon><Download /></el-icon></template
            >导出当前筛选结果
          </el-button>
          <span class="total-count">共 {{ total }} 条</span>
        </div>
      </div>

      <div v-if="selectedUsers.length > 0" class="batch-toolbar">
        <span class="batch-count">已选 {{ selectedUsers.length }} 项</span>
        <div class="batch-actions">
          <el-button
            size="small"
            plain
            @click="handleBatchCreateGroupPermission(true)"
            >批量允许建群</el-button
          >
          <el-button
            size="small"
            plain
            @click="handleBatchCreateGroupPermission(false)"
            >批量禁止建群</el-button
          >
          <el-button
            size="small"
            plain
            @click="openBatchWhitelistModal"
            >批量登录白名单</el-button
          >
          <el-button size="small" plain @click="handleBatchBan(true)"
            >批量封禁</el-button
          >
          <el-button size="small" plain @click="handleBatchBan(false)"
            >批量解封</el-button
          >
          <el-button size="small" plain @click="handleBatchDelete"
            >批量注销</el-button
          >
          <el-button
            v-if="isAdmin"
            size="small"
            plain
            @click="openBatchAssignModal"
            >批量分配</el-button
          >
          <el-button size="small" plain @click="clearSelection"
            >取消选择</el-button
          >
        </div>
      </div>

      <!-- 用户表格 -->
      <div class="table-scroll-x users-table-wrap">
        <el-table
          ref="usersTableRef"
          :data="users"
          row-key="id"
          v-loading="loading"
          stripe
          size="small"
          style="width: 100%"
          @selection-change="
            (rows: AppUser[]) => {
              selectedUsers = rows;
            }
          "
        >
          <el-table-column
            type="selection"
            width="46"
            align="center"
            reserve-selection
          />
          <el-table-column label="序号" width="60" align="center">
            <template #default="{ $index }">
              <span class="cell-id">{{
                (page - 1) * pageSize + $index + 1
              }}</span>
            </template>
          </el-table-column>
          <el-table-column label="用户" min-width="150">
            <template #default="{ row }">
              <div class="user-cell">
                <div class="user-avatar">
                  <span>{{ row.nickname?.charAt(0) }}</span>
                </div>
                <span class="cell-username">{{ row.nickname }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="IM用户ID" min-width="130">
            <template #default="{ row }"
              ><span class="cell-code cell-muted">{{
                row.imUserID
              }}</span></template
            >
          </el-table-column>
          <el-table-column v-if="!isMobile" label="信号" min-width="130">
            <template #default="{ row }">
              <span v-if="row.wananID" class="cell-code">{{
                row.wananID
              }}</span>
              <span v-else class="cell-empty">—</span>
            </template>
          </el-table-column>
          <el-table-column label="手机号" min-width="130">
            <template #default="{ row }"
              ><span class="cell-code cell-muted">{{
                row.phone
              }}</span></template
            >
          </el-table-column>
          <el-table-column
            :label="isMobile ? '归属 / 状态' : '归属'"
            :min-width="isMobile ? 168 : 120"
          >
            <template #default="{ row }">
              <div v-if="isMobile" class="owner-mobile-cell">
                <span
                  v-if="row.ownerName || row.ownerState"
                  class="owner-badge"
                  :class="
                    row.ownerRole === 'assistant'
                      ? 'owner-assistant'
                      : 'owner-admin'
                  "
                >
                  {{ ownerLabel(row) }}
                </span>
                <span v-else class="cell-empty">—</span>
                <div class="owner-mobile-statuses">
                  <span
                    class="status-badge status-badge-compact"
                    :class="statusLabel(row).cls"
                    :title="statusTitle(row)"
                  >
                    {{ statusLabel(row).text }}
                  </span>
                  <span
                    v-if="row.onlineStatus"
                    class="status-badge status-badge-compact"
                    :class="
                      row.onlineStatus === 'online'
                        ? 'status-online'
                        : 'status-offline'
                    "
                  >
                    {{ row.onlineStatus === "online" ? "在线" : "离线" }}
                  </span>
                </div>
              </div>
              <template v-else>
                <span
                  v-if="row.ownerName || row.ownerState"
                  class="owner-badge"
                  :class="
                    row.ownerRole === 'assistant'
                      ? 'owner-assistant'
                      : 'owner-admin'
                  "
                >
                  {{ ownerLabel(row) }}
                </span>
                <span v-else class="cell-empty">—</span>
              </template>
            </template>
          </el-table-column>
          <el-table-column label="建群权限" width="110" align="center">
            <template #default="{ row }">
              <el-tag :type="row.canCreateGroup ? 'success' : 'info'" size="small">
                {{ row.canCreateGroup ? "允许" : "禁止" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="邀请码" min-width="120">
            <template #default="{ row }">
              <div v-if="row.inviteCode" class="code-cell">
                <span class="cell-code invite-code">{{ row.inviteCode }}</span>
                <button
                  class="copy-btn"
                  @click="copyInviteCode(row.inviteCode)"
                >
                  <svg
                    width="13"
                    height="13"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                    />
                  </svg>
                </button>
              </div>
              <span v-else class="cell-empty">—</span>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="状态" width="96">
            <template #default="{ row }">
              <span class="status-badge" :class="statusLabel(row).cls" :title="statusTitle(row)">{{
                statusLabel(row).text
              }}</span>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="风控处置" width="116">
            <template #default="{ row }">
              <span
                class="status-badge"
                :class="cleanupStatusLabel(row).cls"
                :title="row.cleanupReason || cleanupStatusLabel(row).text"
              >
                {{ cleanupStatusLabel(row).text }}
              </span>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="Google验证" width="112">
            <template #default="{ row }">
              <span
                class="status-badge"
                :class="row.totpEnabled ? 'status-active' : 'status-inactive'"
              >
                {{ row.totpEnabled ? "已开启" : "未开启" }}
              </span>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="实名" width="96">
            <template #default="{ row }">
              <span
                class="status-badge"
                :class="row.realNameVerified ? 'status-active' : 'status-inactive'"
                :title="row.realNameVerified && row.realName ? `${row.realName} ${row.idNumberMasked || ''}` : ''"
              >
                {{ row.realNameVerified ? "已实名" : "未实名" }}
              </span>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="在线状态" width="100">
            <template #default="{ row }">
              <span
                v-if="row.onlineStatus"
                class="status-badge"
                :class="
                  row.onlineStatus === 'online'
                    ? 'status-online'
                    : 'status-offline'
                "
              >
                {{ row.onlineStatus === "online" ? "在线" : "离线" }}
              </span>
              <span v-else class="cell-empty">—</span>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="最后在线" min-width="155">
            <template #default="{ row }">
              <span v-if="row.lastOnlineAt" class="cell-code cell-muted cell-xs">
                {{ row.lastOnlineAt }}
              </span>
              <span v-else class="cell-empty">—</span>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="最后离线" min-width="155">
            <template #default="{ row }">
              <span v-if="row.lastOfflineAt" class="cell-code cell-muted cell-xs">
                {{ row.lastOfflineAt }}
              </span>
              <span v-else class="cell-empty">—</span>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="登录白名单" min-width="190">
            <template #default="{ row }">
              <div class="whitelist-cell">
                <span
                  class="status-badge"
                  :class="whitelistStatusLabel(row).cls"
                >
                  {{ whitelistStatusLabel(row).text }}
                </span>
                <span
                  v-if="row.ipWhitelistEnabled && row.ipWhitelistIP"
                  class="cell-code cell-muted cell-xs"
                  :title="row.ipWhitelistIP"
                >
                  {{ formatWhitelistPreview(row.ipWhitelistIP) }}
                </span>
                <div class="whitelist-cell-actions">
                  <button
                    class="table-link-btn"
                    @click="openWhitelistModal(row)"
                  >
                    设置
                  </button>
                  <button
                    v-if="!row.ipWhitelistEnabled && row.currentLoginIP"
                    class="table-link-btn table-link-btn-primary"
                    @click="handleQuickEnableWhitelist(row)"
                  >
                    快捷启用
                  </button>
                  <button
                    v-else-if="row.ipWhitelistEnabled"
                    class="table-link-btn"
                    @click="handleQuickDisableWhitelist(row)"
                  >
                    关闭
                  </button>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column v-if="!isMobile" label="注册时间" min-width="155">
            <template #default="{ row }"
              ><span class="cell-code cell-muted cell-xs">{{
                row.createdAt
              }}</span></template
            >
          </el-table-column>
          <el-table-column
            label="操作"
            :width="isMobile ? 124 : 150"
            align="center"
            :fixed="isMobile ? false : 'right'"
          >
            <template #default="{ row }">
              <div class="action-btns">
                <el-button
                  size="small"
                  type="primary"
                  plain
                  style="padding: 2px 8px; font-size: 12px"
                  @click="openEditDialog(row)"
                  >编辑</el-button
                >
                <el-dropdown
                  trigger="click"
                  popper-class="user-action-dropdown"
                  @command="handleRowAction"
                >
                  <el-button size="small" plain class="more-btn">
                    更多
                    <el-icon class="more-btn-icon"><MoreFilled /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item v-if="!isDeletedUser(row)" :disabled="imRepairPending.includes(row.id)" :command="{ action: 'imRepair', row }">同步聊天账号</el-dropdown-item>
                      <el-dropdown-item
                        :command="{ action: 'conversation', row }"
                      >
                        <div class="dropdown-item">
                          <div
                            class="dropdown-item-title dropdown-item-primary"
                          >
                            会话
                          </div>
                          <div class="dropdown-item-desc">
                            打开用户会话与消息
                          </div>
                        </div>
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-if="!isDeletedUser(row)"
                        :command="{ action: 'whitelist', row }"
                      >
                        <div class="dropdown-item">
                          <div class="dropdown-item-title dropdown-item-primary">
                            白名单
                          </div>
                          <div class="dropdown-item-desc">
                            查看状态并快捷设置登录白名单
                          </div>
                        </div>
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-if="isAdmin && !isDeletedUser(row)"
                        :command="{ action: 'assign', row }"
                      >
                        <div class="dropdown-item">
                          <div
                            class="dropdown-item-title dropdown-item-default"
                          >
                            分配
                          </div>
                          <div class="dropdown-item-desc">调整用户归属人</div>
                        </div>
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-if="!isDeletedUser(row)"
                        :command="{ action: 'groupPermission', row }"
                      >
                        <div class="dropdown-item">
                          <div
                            class="dropdown-item-title"
                            :class="
                              row.canCreateGroup
                                ? 'dropdown-item-warning'
                                : 'dropdown-item-success'
                            "
                          >
                            {{ row.canCreateGroup ? "禁止建群" : "允许建群" }}
                          </div>
                          <div class="dropdown-item-desc">
                            {{
                              row.canCreateGroup
                                ? "关闭当前用户建群权限"
                                : "开启当前用户建群权限"
                            }}
                          </div>
                        </div>
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-if="!isDeletedUser(row)"
                        :command="{ action: 'kick', row }"
                      >
                        <div class="dropdown-item">
                          <div
                            class="dropdown-item-title dropdown-item-default"
                          >
                            踢出
                          </div>
                          <div class="dropdown-item-desc">强制当前设备下线</div>
                        </div>
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-if="row.loginCooldown && !isDeletedUser(row)"
                        :command="{ action: 'resolveLoginCooldown', row }"
                        :disabled="loginCooldownResolvingIds.has(row.id)"
                      >
                        <div class="dropdown-item">
                          <div class="dropdown-item-title dropdown-item-success">
                            {{
                              loginCooldownResolvingIds.has(row.id)
                                ? "解除中..."
                                : "解除登录冷却"
                            }}
                          </div>
                          <div class="dropdown-item-desc">
                            清除密码错误冷却，允许用户立即重新登录
                          </div>
                        </div>
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-if="!isDeletedUser(row)"
                        :command="{ action: row.totpEnabled ? 'disableTotp' : 'enableTotp', row }"
                      >
                        <div class="dropdown-item">
                          <div
                            class="dropdown-item-title"
                            :class="row.totpEnabled ? 'dropdown-item-warning' : 'dropdown-item-success'"
                          >
                            {{ row.totpEnabled ? "关闭Google验证" : "开启Google验证" }}
                          </div>
                          <div class="dropdown-item-desc">
                            {{
                              row.totpEnabled
                                ? "关闭 Web 登录二次验证并撤销登录态"
                                : "下次 Web 登录需绑定或校验动态码，App 暂不强制"
                            }}
                          </div>
                        </div>
                      </el-dropdown-item>
                      <el-dropdown-item v-if="!isDeletedUser(row)" :command="{ action: 'globalSearch', row }">
                        <div class="dropdown-item">
                          <div class="dropdown-item-title">{{ row.globallySearchable ? '关闭全局搜索' : '开启全局搜索' }}</div>
                          <div class="dropdown-item-desc">设置该用户能否被其他服务空间精确搜索</div>
                        </div>
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-if="authStore.isSuperAdmin && row.realNameVerified && !isDeletedUser(row)"
                        :command="{ action: 'resetRealname', row }"
                      >
                        <div class="dropdown-item">
                          <div class="dropdown-item-title dropdown-item-warning">
                            重置实名
                          </div>
                          <div class="dropdown-item-desc">
                            清除实名认证状态，用户需重新核验
                          </div>
                        </div>
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-if="!isDeletedUser(row)"
                        :command="{ action: 'ban', row }"
                      >
                        <div class="dropdown-item">
                          <div
                            class="dropdown-item-title"
                            :class="banActionCopy(row).titleClass"
                          >
                            {{ banActionCopy(row).title }}
                          </div>
                          <div class="dropdown-item-desc">
                            {{ banActionCopy(row).desc }}
                          </div>
                        </div>
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-if="!isDeletedUser(row)"
                        :command="{ action: 'delete', row }"
                        divided
                      >
                        <div class="dropdown-item">
                          <div class="dropdown-item-title dropdown-item-danger">
                            注销
                          </div>
                          <div class="dropdown-item-desc">软注销该账号</div>
                        </div>
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :page-sizes="pageSizeOptions"
          :total="total"
          layout="sizes, prev, pager, next"
          background
          small
          @size-change="
            (size: number) => {
              pageSize = size;
              page = 1;
              syncUsersRouteQuery();
            }
          "
          @current-change="
            (p: number) => {
              page = p;
              syncUsersRouteQuery();
            }
          "
        />
      </div>
    </div>

    <!-- ── 分配弹窗 ── -->
    <el-dialog
      v-model="showAssignModal"
      title="分配用户"
      :width="isMobile ? '100%' : 400"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="assign-body">
        <div v-if="assignTargetUser" class="assign-user-row">
          <div class="user-avatar">
            <span>{{ assignTargetUser.nickname?.charAt(0) }}</span>
          </div>
          <div class="assign-user-info">
            <span class="assign-username">{{ assignTargetUser.nickname }}</span>
            <span class="assign-imid">{{ assignTargetUser.imUserID }}</span>
          </div>
        </div>
        <div v-else class="assign-batch-row">
          <span class="assign-batch-count"
            >将批量分配 {{ assignTargetCount }} 个用户</span
          >
        </div>
        <div class="assign-form">
          <label class="assign-label">分配给</label>
          <el-select
            v-model="assignOwnerID"
            placeholder="选择归属人"
            style="width: 100%"
          >
            <el-option
              v-for="opt in ownerManagementOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </div>
      </div>
      <template #footer>
        <el-button @click="showAssignModal = false">取消</el-button>
        <el-button
          type="primary"
          :loading="assignSubmitting"
          @click="submitAssign"
          >确认分配</el-button
        >
      </template>
    </el-dialog>

    <!-- ── 新建用户弹窗 ── -->
    <el-dialog
      v-model="showCreateModal"
      title="新建用户"
      :width="isMobile ? '100%' : 920"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
      :top="isMobile ? '0' : '6vh'"
    >
      <div class="create-tabs">
        <button
          class="create-tab"
          :class="{ active: createTab === 'manual' }"
          @click="createTab = 'manual'"
        >
          手动录入
        </button>
        <button
          class="create-tab"
          :class="{ active: createTab === 'file' }"
          @click="createTab = 'file'"
        >
          CSV 导入
        </button>
        <button
          class="create-tab"
          :class="{ active: createTab === 'template' }"
          @click="createTab = 'template'"
        >
          模板生成
        </button>
      </div>

      <!-- 手动录入 -->
      <template v-if="createTab === 'manual'">
        <div class="m-table-wrap">
          <table class="m-table">
            <thead>
              <tr>
                <th class="th-num">#</th>
                <th>手机号 <span class="req">*</span></th>
                <th>初始密码 <span class="req">*</span></th>
                <th>昵称 <span class="opt">选填</span></th>
                <th v-if="isAdmin">归属人</th>
                <th v-if="manualSubmitted">结果</th>
                <th class="th-del"></th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, i) in manualRows"
                :key="i"
                :class="{ 'row-alt': i % 2 === 1 }"
              >
                <td class="td-num font-code">{{ i + 1 }}</td>
                <td class="td-input">
                  <el-input
                    v-model="row.phone"
                    size="small"
                    placeholder="138 0000 0000"
                  />
                </td>
                <td class="td-input">
                  <el-input
                    v-model="row.password"
                    size="small"
                    type="password"
                    placeholder="至少 8 位"
                    show-password
                  />
                </td>
                <td class="td-input">
                  <el-input
                    v-model="row.nickname"
                    size="small"
                    placeholder="留空则首登设置"
                  />
                </td>
                <td v-if="isAdmin" class="td-input">
                  <el-select
                    v-model="row.ownerID"
                    size="small"
                    style="width: 120px"
                  >
                    <el-option
                      v-for="opt in ownerManagementOptions"
                      :key="opt.value"
                      :value="opt.value"
                      :label="opt.label"
                    />
                  </el-select>
                </td>
                <td v-if="manualSubmitted" class="td-result">
                  <el-tag
                    v-if="row.result"
                    :type="row.result.success ? 'success' : 'danger'"
                    size="small"
                    effect="light"
                    round
                  >
                    {{ row.result.message }}
                  </el-tag>
                </td>
                <td class="td-del">
                  <button
                    class="del-btn"
                    :disabled="manualRows.length === 1"
                    @click="removeRow(i)"
                  >
                    <svg
                      width="13"
                      height="13"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M20 12H4"
                      />
                    </svg>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
          <div class="m-table-footer">
            <button class="add-row-btn" @click="addRow">
              <svg
                width="13"
                height="13"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 4v16m8-8H4"
                />
              </svg>
              添加一行
            </button>
          </div>
        </div>
        <div class="modal-actions">
          <el-button v-if="manualSubmitted" @click="resetManual"
            >重置</el-button
          >
          <el-button @click="showCreateModal = false">取消</el-button>
          <el-button
            type="primary"
            :loading="manualSubmitting"
            @click="submitManual"
            >创建用户</el-button
          >
        </div>
      </template>

      <!-- CSV 导入 -->
      <template v-else-if="createTab === 'file'">
        <div class="file-area">
          <div class="file-toolbar">
            <el-button size="small" @click="downloadTemplate">
              <template #icon
                ><el-icon><Download /></el-icon></template
              >下载 CSV 模板
            </el-button>
          </div>

          <div v-if="!csvFile" class="dropzone" @click="triggerFileInput">
            <svg
              class="drop-icon"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.5"
                d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
              />
            </svg>
            <p class="drop-title">点击上传 CSV 文件</p>
            <p class="drop-hint">
              仅支持 .csv 格式，表头：手机号,初始密码,昵称(选填)
            </p>
            <input
              ref="fileInputRef"
              type="file"
              accept=".csv"
              class="file-hidden"
              @change="onFileChange"
            />
          </div>

          <template v-if="previewRows.length > 0">
            <div class="preview-header">
              <span class="preview-count"
                >已解析 <b>{{ previewRows.length }}</b> 条记录</span
              >
              <el-button size="small" @click="resetFile">重新上传</el-button>
            </div>
            <div class="preview-table-wrap">
              <el-table
                :data="previewRows"
                stripe
                size="small"
                style="width: 100%"
                max-height="300"
              >
                <el-table-column label="#" width="48" align="center">
                  <template #default="{ $index }"
                    ><span class="font-code cell-xs">{{
                      $index + 1
                    }}</span></template
                  >
                </el-table-column>
                <el-table-column label="手机号" min-width="130">
                  <template #default="{ row }"
                    ><span class="font-code cell-muted">{{
                      row.phone
                    }}</span></template
                  >
                </el-table-column>
                <el-table-column label="初始密码" min-width="110">
                  <template #default
                    ><span class="cell-xs">••••••••</span></template
                  >
                </el-table-column>
                <el-table-column label="昵称" min-width="120">
                  <template #default="{ row }"
                    ><span class="cell-muted">{{
                      row.nickname || "—"
                    }}</span></template
                  >
                </el-table-column>
                <el-table-column v-if="fileSubmitted" label="结果" width="110">
                  <template #default="{ row }">
                    <el-tag
                      v-if="row.result"
                      :type="row.result.success ? 'success' : 'danger'"
                      size="small"
                      effect="light"
                      round
                    >
                      {{ row.result.message }}
                    </el-tag>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </template>
        </div>
        <div class="modal-actions">
          <el-button v-if="csvFile" @click="resetFile">重新上传</el-button>
          <el-button @click="showCreateModal = false">取消</el-button>
          <el-button
            type="primary"
            :disabled="!previewRows.length"
            :loading="fileSubmitting"
            @click="submitFile"
          >
            创建用户
          </el-button>
        </div>
      </template>

      <template v-else>
        <div class="template-layout">
          <div class="template-card">
            <div class="template-card-head">
              <div>
                <div class="template-card-title">模板配置</div>
                <div class="template-card-sub">
                  按手机号、昵称、密码模板一键批量生成账号
                </div>
              </div>
              <div class="template-count-pills">
                <button
                  v-for="preset in templateCountPresets"
                  :key="preset"
                  class="template-count-pill"
                  :class="{ active: templateForm.count === preset }"
                  @click="templateForm.count = preset"
                >
                  {{ preset }}
                </button>
              </div>
            </div>

            <div class="template-section">
              <div class="template-section-title">生成设置</div>
              <div class="template-form-grid template-form-grid-top">
                <div class="template-field">
                  <label>生成数量</label>
                  <el-input-number
                    v-model="templateForm.count"
                    :min="1"
                    :max="200"
                    controls-position="right"
                    style="width: 100%"
                  />
                </div>
                <div v-if="isAdmin" class="template-field">
                  <label>归属人</label>
                  <el-select
                    v-model="templateForm.ownerID"
                    placeholder="选择归属人"
                    style="width: 100%"
                  >
                    <el-option
                      v-for="opt in ownerManagementOptions"
                      :key="opt.value"
                      :label="opt.label"
                      :value="opt.value"
                    />
                  </el-select>
                </div>
                <div class="template-field">
                  <label>归属预览</label>
                  <div class="template-static-value">
                    {{ selectedTemplateOwnerLabel }}
                  </div>
                </div>
                <div class="template-field template-toggle-field">
                  <label>自动实名</label>
                  <div class="template-switch-row">
                    <el-switch
                      v-model="templateForm.realNameVerified"
                      inline-prompt
                      active-text="开"
                      inactive-text="关"
                    />
                    <span class="template-static-value">
                      {{
                        templateForm.realNameVerified
                          ? "开启：生成后标记为已实名"
                          : "关闭：生成后保持未实名"
                      }}
                    </span>
                  </div>
                  <div class="template-tip" style="margin-top: 6px">
                    默认开启；关闭后账号需用户自行完成实名核验
                  </div>
                </div>
                <div class="template-field template-toggle-field">
                  <label>允许建群</label>
                  <div class="template-switch-row">
                    <el-switch
                      v-model="templateForm.canCreateGroup"
                      inline-prompt
                      active-text="开"
                      inactive-text="关"
                    />
                    <span class="template-static-value">
                      {{
                        templateForm.canCreateGroup
                          ? "开启：生成后允许创建群聊"
                          : "关闭：生成后禁止创建群聊"
                      }}
                    </span>
                  </div>
                  <div class="template-tip" style="margin-top: 6px">
                    默认关闭；可在用户管理中单独或批量调整
                  </div>
                </div>
              </div>
            </div>

            <div class="template-section">
              <div class="template-section-title">手机号模板</div>
              <div class="template-form-grid">
                <div class="template-field">
                  <label>前三位</label>
                  <el-input
                    v-model="templateForm.phonePrefix"
                    maxlength="3"
                    placeholder="如 139"
                  />
                </div>
                <div class="template-field">
                  <label>中间四位模式</label>
                  <el-select
                    v-model="templateForm.phoneMiddleMode"
                    style="width: 100%"
                  >
                    <el-option label="递增" value="sequence" />
                    <el-option label="随机" value="random" />
                  </el-select>
                </div>
                <div
                  class="template-field"
                  :class="{
                    'template-field-disabled':
                      templateForm.phoneMiddleMode !== 'sequence',
                  }"
                >
                  <label>中间起始值</label>
                  <el-input-number
                    v-model="templateForm.phoneMiddleStart"
                    :min="0"
                    :max="9999"
                    controls-position="right"
                    style="width: 100%"
                    :disabled="templateForm.phoneMiddleMode !== 'sequence'"
                  />
                </div>
                <div class="template-field">
                  <label>后四位</label>
                  <el-input
                    v-model="templateForm.phoneSuffix"
                    maxlength="4"
                    placeholder="如 1234"
                  />
                </div>
              </div>
              <div class="template-tip">
                格式为「前三位 + 中间四位 +
                后四位」，当前版本支持固定前三位和后四位，中间四位可选递增或随机。
              </div>
            </div>

            <div class="template-section">
              <div class="template-section-title">昵称模板</div>
              <div class="template-form-grid">
                <div class="template-field">
                  <label>固定前缀</label>
                  <el-input
                    v-model="templateForm.nicknamePrefix"
                    maxlength="24"
                    placeholder="如 测试用户"
                  />
                </div>
                <div class="template-field">
                  <label>起始编号</label>
                  <el-input-number
                    v-model="templateForm.nicknameStart"
                    :min="0"
                    :max="999999"
                    controls-position="right"
                    style="width: 100%"
                  />
                </div>
                <div class="template-field">
                  <label>补零位数</label>
                  <el-input-number
                    v-model="templateForm.nicknamePadding"
                    :min="0"
                    :max="8"
                    controls-position="right"
                    style="width: 100%"
                  />
                </div>
              </div>
            </div>

            <div class="template-section">
              <div class="template-section-title">密码模板</div>
              <div class="template-mode-list">
                <button
                  v-for="option in passwordModeOptions"
                  :key="option.value"
                  class="template-mode-card"
                  :class="{
                    active: templateForm.passwordMode === option.value,
                  }"
                  @click="templateForm.passwordMode = option.value"
                >
                  <span class="template-mode-title">{{ option.label }}</span>
                  <span class="template-mode-desc">{{ option.desc }}</span>
                </button>
              </div>
              <div class="template-form-grid template-password-grid">
                <div
                  v-if="templateForm.passwordMode === 'fixed'"
                  class="template-field template-field-span"
                >
                  <label>统一固定密码</label>
                  <el-input
                    v-model="templateForm.fixedPassword"
                    type="password"
                    show-password
                    placeholder="至少 8 位，所有账号共用"
                  />
                </div>
                <template
                  v-else-if="templateForm.passwordMode === 'prefix_random'"
                >
                  <div class="template-field">
                    <label>固定前缀</label>
                    <el-input
                      v-model="templateForm.passwordPrefix"
                      maxlength="24"
                      placeholder="如 Chengnan@"
                    />
                  </div>
                  <div class="template-field">
                    <label>随机后缀长度</label>
                    <el-input-number
                      v-model="templateForm.passwordRandomLength"
                      :min="1"
                      :max="24"
                      controls-position="right"
                      style="width: 100%"
                    />
                  </div>
                </template>
                <div v-else class="template-field">
                  <label>随机密码长度</label>
                  <el-input-number
                    v-model="templateForm.randomPasswordLength"
                    :min="8"
                    :max="32"
                    controls-position="right"
                    style="width: 100%"
                  />
                </div>
              </div>
              <div
                v-if="templateForm.passwordMode === 'fixed'"
                class="template-risk-banner"
              >
                统一固定密码适合短期批量开通，但安全性较低。系统仍会要求用户首次登录后修改密码。
              </div>
            </div>
          </div>

          <div class="template-card template-card-preview">
            <div class="template-card-head">
              <div>
                <div class="template-card-title">样例预览</div>
                <div class="template-card-sub">
                  根据当前模板即时展示前 5
                  条样例，最终创建结果以服务端去重后的数据为准
                </div>
              </div>
            </div>

            <div class="template-preview-meta">
              <span>归属：{{ selectedTemplateOwnerLabel }}</span>
              <span
                >自动实名：{{
                  templateForm.realNameVerified ? "开启" : "关闭"
                }}</span
              >
              <span
                >允许建群：{{
                  templateForm.canCreateGroup ? "开启" : "关闭"
                }}</span
              >
              <span
                >密码模式：{{
                  passwordModeOptions.find(
                    (option) => option.value === templateForm.passwordMode,
                  )?.label
                }}</span
              >
            </div>

            <div class="preview-table-wrap">
              <el-table
                :data="templatePreviewRows"
                stripe
                size="small"
                style="width: 100%"
                max-height="260"
              >
                <el-table-column label="#" width="52" align="center">
                  <template #default="{ row }"
                    ><span class="font-code cell-xs">{{
                      row.index
                    }}</span></template
                  >
                </el-table-column>
                <el-table-column label="手机号" min-width="132">
                  <template #default="{ row }"
                    ><span class="font-code cell-muted">{{
                      row.phone
                    }}</span></template
                  >
                </el-table-column>
                <el-table-column label="昵称" min-width="140">
                  <template #default="{ row }"
                    ><span class="cell-muted">{{
                      row.nickname
                    }}</span></template
                  >
                </el-table-column>
                <el-table-column label="初始密码" min-width="150">
                  <template #default="{ row }"
                    ><span class="font-code cell-muted">{{
                      row.password
                    }}</span></template
                  >
                </el-table-column>
                <el-table-column label="实名" width="72" align="center">
                  <template #default>
                    <span
                      :class="
                        templateForm.realNameVerified
                          ? 'status-active'
                          : 'status-inactive'
                      "
                      >{{
                        templateForm.realNameVerified ? "已实名" : "未实名"
                      }}</span
                    >
                  </template>
                </el-table-column>
              </el-table>
            </div>

            <div class="template-result-note">
              初始密码只会在本次生成结果里完整展示一次，关闭弹窗后无法再次查看明文。
            </div>

            <template v-if="templateSubmitted">
              <div class="template-result-head">
                <div class="template-result-summary">
                  <span>请求 {{ templateSummary.requested }} 个</span>
                  <span class="template-summary-success"
                    >成功 {{ templateSummary.success }}</span
                  >
                  <span class="template-summary-failed"
                    >失败 {{ templateSummary.failed }}</span
                  >
                </div>
                <div class="template-result-actions">
                  <el-button size="small" plain @click="copyGeneratedAccounts"
                    >复制成功记录</el-button
                  >
                  <el-button
                    size="small"
                    plain
                    @click="downloadGeneratedAccounts"
                    >导出 CSV</el-button
                  >
                </div>
              </div>

              <div class="preview-table-wrap">
                <el-table
                  :data="templateResultRows"
                  stripe
                  size="small"
                  style="width: 100%"
                  max-height="280"
                >
                  <el-table-column label="#" width="52" align="center">
                    <template #default="{ row }"
                      ><span class="font-code cell-xs">{{
                        row.index + 1
                      }}</span></template
                    >
                  </el-table-column>
                  <el-table-column label="手机号" min-width="132">
                    <template #default="{ row }"
                      ><span class="font-code cell-muted">{{
                        row.phone || "—"
                      }}</span></template
                    >
                  </el-table-column>
                  <el-table-column label="初始密码" min-width="148">
                    <template #default="{ row }"
                      ><span class="font-code cell-muted">{{
                        row.password || "—"
                      }}</span></template
                    >
                  </el-table-column>
                  <el-table-column label="昵称" min-width="140">
                    <template #default="{ row }"
                      ><span class="cell-muted">{{
                        row.nickname || "—"
                      }}</span></template
                    >
                  </el-table-column>
                  <el-table-column label="IM用户ID" min-width="132">
                    <template #default="{ row }"
                      ><span class="font-code cell-muted">{{
                        row.im_user_id || "—"
                      }}</span></template
                    >
                  </el-table-column>
                  <el-table-column label="结果" min-width="120">
                    <template #default="{ row }">
                      <el-tag
                        :type="row.success ? 'success' : 'danger'"
                        size="small"
                        effect="light"
                        round
                      >
                        {{ row.message }}
                      </el-tag>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </template>
          </div>
        </div>
        <div class="modal-actions">
          <el-button v-if="templateSubmitted" @click="resetTemplate"
            >重置</el-button
          >
          <el-button @click="showCreateModal = false">取消</el-button>
          <el-button
            type="primary"
            :loading="templateSubmitting"
            @click="submitTemplateGeneration"
          >
            立即生成
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- ── 用户同步弹窗 ── -->
    <el-dialog
      v-model="showSyncModal"
      title="同步用户资料"
      :width="isMobile ? '100%' : 980"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div v-loading="syncLoading" class="sync-body">
        <div
          v-if="!syncLoading && syncPreviews.length === 0"
          class="sync-empty"
        >
          <p>未发现需要同步的差异用户</p>
        </div>
        <template v-else>
          <div class="sync-header">
            <span class="sync-count"
              >共 {{ syncPreviews.length }} 个差异用户</span
            >
            <span class="sync-hint"
              >支持将本地资料更新为 IM
              最新资料：昵称、头像、签名、性别、生日、加好友策略</span
            >
          </div>
          <div class="sync-table-wrap">
            <table class="sync-table">
              <thead>
                <tr>
                  <th>IM用户ID</th>
                  <th>本地昵称</th>
                  <th>IM昵称</th>
                  <th>本地头像</th>
                  <th>IM头像</th>
                  <th>差异字段</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in syncPreviews" :key="item.imUserID">
                  <td>
                    <span class="cell-code">{{ item.imUserID }}</span>
                  </td>
                  <td>
                    <span class="cell-muted">{{
                      displayValue(item.localNickname)
                    }}</span>
                  </td>
                  <td>
                    <span class="cell-muted">{{
                      displayValue(item.imNickname)
                    }}</span>
                  </td>
                  <td>
                    <a
                      v-if="item.localAvatarUrl"
                      :href="item.localAvatarUrl"
                      target="_blank"
                      rel="noreferrer"
                      class="sync-link"
                      >查看</a
                    >
                    <span v-else class="cell-empty">—</span>
                  </td>
                  <td>
                    <a
                      v-if="item.imAvatarUrl"
                      :href="item.imAvatarUrl"
                      target="_blank"
                      rel="noreferrer"
                      class="sync-link"
                      >查看</a
                    >
                    <span v-else class="cell-empty">—</span>
                  </td>
                  <td>
                    <span class="cell-muted">{{
                      item.diffFields?.length ? item.diffFields.join("、") : "—"
                    }}</span>
                  </td>
                  <td>
                    <span
                      class="sync-status-badge"
                      :class="syncStatusLabel(item.status).cls"
                    >
                      {{ syncStatusLabel(item.status).text }}
                    </span>
                  </td>
                  <td>
                    <button
                      class="action-toggle"
                      :class="`action-${item.action}`"
                      :disabled="item.status === 'missing_in_im'"
                      @click="toggleAction(item)"
                    >
                      {{
                        item.status === "missing_in_im"
                          ? "不可更新"
                          : item.action === "update"
                            ? "已选更新"
                            : "加入更新"
                      }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
      </div>
      <template #footer>
        <el-button @click="showSyncModal = false">取消</el-button>
        <el-button
          type="primary"
          :loading="syncSubmitting"
          :disabled="syncLoading || syncPreviews.length === 0"
          @click="submitSync"
        >
          确认更新
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showBatchWhitelistModal"
      title="批量设置登录白名单"
      :width="isMobile ? '100%' : 560"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
      @close="closeBatchWhitelistModal"
    >
      <div class="whitelist-quick-dialog">
        <div class="whitelist-quick-summary">
          <div class="whitelist-quick-title">
            已选 {{ selectedUsers.length }} 个用户
          </div>
          <div class="whitelist-quick-sub">
            提交后会将所选用户的登录白名单设置为相同配置，并覆盖他们原有的登录白名单。每行一个 IP，也支持逗号或分号分隔多个 IP。
          </div>
        </div>

        <el-form :model="batchWhitelistForm" label-width="110px">
          <el-form-item label="启用白名单">
            <el-switch v-model="batchWhitelistForm.enabled" />
          </el-form-item>
          <el-form-item label="白名单 IP">
            <el-input
              v-model="batchWhitelistForm.ip"
              type="textarea"
              :rows="5"
              :disabled="!batchWhitelistForm.enabled"
              placeholder="每行一个 IP 地址，支持填写多个"
            />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-button @click="closeBatchWhitelistModal">取消</el-button>
        <el-button
          type="primary"
          :loading="batchWhitelistSubmitting"
          @click="submitBatchWhitelistModal"
        >
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- ── 编辑用户弹窗 ── -->
    <el-dialog
      v-model="showWhitelistModal"
      :title="
        whitelistTargetUser
          ? `登录白名单 · ${whitelistTargetUser.nickname}`
          : '登录白名单'
      "
      :width="isMobile ? '100%' : 560"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
      @close="closeWhitelistModal"
    >
      <div v-loading="whitelistLoading" class="whitelist-quick-dialog">
        <div class="whitelist-quick-summary">
          <div class="whitelist-quick-title">支持多个精确 IP</div>
          <div class="whitelist-quick-sub">
            每行一个 IP，也支持逗号或分号分隔多个 IP。保存后若当前活动登录 IP 均不匹配，系统会立即强制下线。
          </div>
        </div>

        <el-form :model="whitelistForm" label-width="110px">
          <el-form-item label="启用白名单">
            <el-switch v-model="whitelistForm.enabled" />
          </el-form-item>
          <el-form-item label="白名单 IP">
            <el-input
              v-model="whitelistForm.ip"
              type="textarea"
              :rows="4"
              :disabled="!whitelistForm.enabled"
              placeholder="每行一个 IP 地址，支持填写多个"
            />
          </el-form-item>
          <el-form-item label="当前登录 IP">
            <span class="readonly-value">{{
              whitelistForm.currentLoginIP || "未获取"
            }}</span>
          </el-form-item>
        </el-form>

        <div class="whitelist-quick-actions">
          <el-button
            v-if="whitelistForm.currentLoginIP"
            plain
            @click="appendWhitelistIP(whitelistForm.currentLoginIP)"
          >
            添加当前登录 IP
          </el-button>
          <el-button
            v-if="!whitelistForm.enabled && whitelistForm.currentLoginIP"
            plain
            type="primary"
            @click="
              whitelistForm.enabled = true;
              whitelistForm.ip = whitelistForm.currentLoginIP;
            "
          >
            快捷启用
          </el-button>
        </div>
      </div>

      <template #footer>
        <el-button @click="closeWhitelistModal">取消</el-button>
        <el-button
          type="primary"
          :loading="whitelistSubmitting"
          @click="submitWhitelistModal"
        >
          保存
        </el-button>
      </template>
    </el-dialog>

    <UserEditDialog
      v-model:visible="showEditDialog"
      :user="editingUser"
      @success="fetchUsers"
    />
  </div>
</template>

<style scoped>
.page {
  padding: 24px 28px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.card-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.toolbar-left {
  flex: 1;
  min-width: 0;
}
.filter-wrap {
  position: relative;
  flex: 1;
  max-width: 760px;
}
.filter-box {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 4px 10px;
  border: 1px solid #f0f0f0;
  border-radius: 10px;
  background: #fff;
  transition:
    border-color 0.15s,
    box-shadow 0.15s;
}
.filter-box:hover {
  border-color: rgba(22, 119, 255, 0.4);
}
.filter-box:focus-within {
  border-color: rgba(22, 119, 255, 0.6);
  box-shadow: 0 0 0 3px rgba(22, 119, 255, 0.08);
}
.filter-icon {
  width: 14px;
  height: 14px;
  color: #94a3b8;
  flex-shrink: 0;
}
.filter-tag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  background: rgba(22, 119, 255, 0.1);
  color: #1677ff;
  font-size: 11px;
  font-weight: 500;
  border-radius: 4px;
}
.filter-tag-key {
  color: rgba(22, 119, 255, 0.6);
  margin-right: 2px;
}
.filter-tag-remove {
  margin-left: 2px;
  background: none;
  border: none;
  color: rgba(22, 119, 255, 0.5);
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0;
}
.filter-tag-remove:hover {
  color: #1677ff;
}
.filter-search-input {
  flex: 1;
  min-width: 220px;
  height: 30px;
  border: none;
  outline: none;
  background: transparent;
  color: #1e293b;
  font-size: 13px;
}
.filter-search-input::placeholder {
  color: #94a3b8;
}
.filter-inline-btn {
  height: 28px;
  padding: 0 10px;
  border: 1px solid #e5e7eb;
  border-radius: 999px;
  background: #fff;
  color: #64748b;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.filter-inline-btn:hover {
  border-color: #bfdbfe;
  color: #1677ff;
}
.filter-inline-btn-primary {
  border-color: #1677ff;
  background: #1677ff;
  color: #fff;
}
.filter-inline-btn-primary:hover {
  background: #4096ff;
  border-color: #4096ff;
  color: #fff;
}
.filter-placeholder {
  color: #94a3b8;
  font-size: 13px;
  user-select: none;
}
.filter-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  width: 288px;
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 10px;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
  z-index: 50;
  overflow: hidden;
}
.filter-helper {
  margin-top: 6px;
  padding: 0 12px;
  color: #94a3b8;
  font-size: 11px;
  line-height: 1.5;
}
.filter-helper-divider {
  margin: 0 6px;
  color: #cbd5e1;
}
.filter-dropdown-head {
  font-size: 10px;
  font-weight: 600;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 12px 12px 6px;
}
.filter-field-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  cursor: pointer;
  font-size: 14px;
  color: #1e293b;
  transition: background 0.1s;
}
.filter-field-item:hover {
  background: #f8fafc;
}
.filter-field-icon {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  background: rgba(22, 119, 255, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #1677ff;
  flex-shrink: 0;
}
.filter-chevron {
  width: 14px;
  height: 14px;
  color: #d1d5db;
  margin-left: auto;
}
.filter-value-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px 8px;
  border-bottom: 1px solid #f0f0f0;
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
}
.filter-back-btn {
  background: none;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  display: flex;
}
.filter-back-btn:hover {
  color: #64748b;
}
.filter-value-body {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.filter-text-input {
  width: 100%;
  height: 32px;
  padding: 0 12px;
  font-size: 14px;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  outline: none;
}
.filter-text-input:focus {
  border-color: rgba(22, 119, 255, 0.6);
}
.filter-option-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 240px;
  overflow-y: auto;
  padding-right: 2px;
}
.filter-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  color: #1e293b;
  transition: background 0.1s;
}
.filter-option:hover {
  background: #f8fafc;
}
.filter-option.selected {
  background: rgba(22, 119, 255, 0.08);
  color: #1677ff;
}
.filter-option-empty {
  color: #94a3b8;
  cursor: default;
  justify-content: center;
}
.filter-option-empty:hover {
  background: transparent;
}
.filter-radio {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid #d1d5db;
  flex-shrink: 0;
  transition: all 0.1s;
}
.filter-radio.active {
  border-color: #1677ff;
  background: #1677ff;
}
.filter-confirm-btn {
  width: 100%;
  height: 32px;
  border-radius: 6px;
  background: #1677ff;
  color: #fff;
  font-size: 12px;
  font-weight: 500;
  border: none;
  cursor: pointer;
}
.filter-confirm-btn:hover {
  background: #4096ff;
}

.clear-btn {
  font-size: 12px;
  color: #94a3b8;
  background: none;
  border: none;
  cursor: pointer;
  white-space: nowrap;
}
.clear-btn:hover {
  color: #64748b;
}
.users-toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
}
.total-count {
  font-size: 12px;
  color: #94a3b8;
  margin-left: auto;
  white-space: nowrap;
}

.users-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}
.user-avatar {
  width: 28px;
  height: 28px;
  border-radius: 7px;
  background: rgba(22, 119, 255, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #1677ff;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
}
.cell-id {
  font-size: 13px;
  color: #94a3b8;
}
.cell-username {
  font-size: 13px;
  font-weight: 500;
  color: #1677ff;
  line-height: 1.3;
}
.cell-code {
  font-family: "Fira Code", "Courier New", monospace;
  font-size: 13px;
  color: #1e293b;
}
.cell-muted {
  font-size: 12px;
  color: #64748b;
}
.cell-xs {
  font-size: 11px;
  color: #94a3b8;
}
.cell-empty {
  color: #d1d5db;
}
.owner-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  font-weight: 500;
  padding: 0 8px;
  height: 22px;
  border-radius: 5px;
}
.owner-admin {
  background: #f0f5ff;
  color: #2f54eb;
}
.owner-assistant {
  background: #f6ffed;
  color: #389e0d;
}
.owner-id {
  font-family: "Fira Code", "Courier New", monospace;
  font-size: 11px;
  opacity: 0.75;
}
.owner-mobile-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
}
.owner-mobile-statuses {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.code-cell {
  display: flex;
  align-items: center;
  gap: 5px;
}
.invite-code {
  font-size: 12px;
  color: #531dab;
  letter-spacing: 0.03em;
}
.copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  background: none;
  color: #d1d5db;
  cursor: pointer;
  transition: color 0.15s;
  padding: 0;
  flex-shrink: 0;
}
.copy-btn:hover {
  color: #1677ff;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  font-size: 12px;
  font-weight: 500;
  padding: 0 9px;
  height: 24px;
  border-radius: 6px;
  border: 1px solid transparent;
}
.status-active {
  color: #52c41a;
  border-color: #b7eb8f;
  background: #f6ffed;
}
.status-banned {
  color: #ff4d4f;
  border-color: #ffa39e;
  background: #fff2f0;
}
.status-deleted {
  color: #8c8c8c;
  border-color: #d9d9d9;
  background: #fafafa;
}
.status-inactive {
  color: #8c8c8c;
  border-color: #d9d9d9;
  background: #fafafa;
}
.status-pending {
  color: #d46b08;
  border-color: #ffd591;
  background: #fff7e6;
}
.status-cooldown {
  color: #b45309;
  border-color: #fcd34d;
  background: #fffbeb;
}
.status-info {
  color: #2563eb;
  border-color: #bfdbfe;
  background: #eff6ff;
}
.status-badge-compact {
  height: 20px;
  padding: 0 7px;
  font-size: 11px;
  border-radius: 999px;
}

.action-btns {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  padding: 2px 0;
}

.more-btn {
  padding: 2px 8px !important;
  font-size: 12px !important;
}

.more-btn-icon {
  margin-left: 4px;
  font-size: 12px;
}

.dropdown-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.dropdown-item-title {
  font-size: 12px;
  font-weight: 600;
  line-height: 1.3;
}

.dropdown-item-desc {
  font-size: 11px;
  line-height: 1.35;
  color: #94a3b8;
}

.dropdown-item-primary {
  color: #1677ff;
}
.dropdown-item-default {
  color: #475569;
}
.dropdown-item-warning {
  color: #d46b08;
}
.dropdown-item-success {
  color: #389e0d;
}
.dropdown-item-danger {
  color: #ff4d4f;
}

.whitelist-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.whitelist-cell-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.table-link-btn {
  padding: 0;
  border: none;
  background: transparent;
  font-size: 12px;
  line-height: 1.4;
  color: #64748b;
  cursor: pointer;
}

.table-link-btn-primary {
  color: #2563eb;
  font-weight: 600;
}

.whitelist-quick-dialog {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.whitelist-quick-summary {
  padding: 14px 16px;
  border-radius: 14px;
  background: linear-gradient(135deg, #f6faff 0%, #eef4ff 100%);
  border: 1px solid #dbe7ff;
}

.whitelist-quick-title {
  font-size: 14px;
  font-weight: 700;
  color: #123257;
}

.whitelist-quick-sub {
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.6;
  color: #5b6b7f;
}

.whitelist-quick-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.readonly-value {
  font-family: "Fira Code", "Courier New", monospace;
  font-size: 13px;
  color: #475569;
  word-break: break-all;
}

:deep(.user-action-dropdown .el-dropdown-menu__item) {
  height: auto;
  min-height: 0;
  padding: 8px 12px;
  line-height: 1.35;
  align-items: flex-start;
}

:deep(.user-action-dropdown .el-dropdown-menu__item.is-divided) {
  margin-top: 4px;
}
.pagination-bar {
  display: flex;
  justify-content: flex-end;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
}

/* 弹窗 tabs */
.create-tabs {
  display: flex;
  gap: 2px;
  padding: 0 0 16px;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 16px;
}
.create-tab {
  padding: 6px 18px;
  font-size: 13px;
  font-weight: 500;
  color: #64748b;
  background: none;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}
.create-tab:hover {
  background: #f8fafc;
  color: #1e293b;
}
.create-tab.active {
  background: rgba(22, 119, 255, 0.1);
  color: #1677ff;
}

/* 手动录入表格 */
.m-table-wrap {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
}
.m-table {
  width: 100%;
  font-size: 13px;
  border-collapse: collapse;
}
.m-table thead {
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
}
.m-table th {
  text-align: left;
  padding: 9px 12px;
  font-size: 11px;
  font-weight: 600;
  color: #8c8c8c;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  white-space: nowrap;
}
.m-table tr {
  border-bottom: 1px solid #f0f0f0;
}
.m-table tr:last-child {
  border-bottom: none;
}
.row-alt {
  background: #fafafa;
}
.th-num {
  width: 36px;
}
.th-del {
  width: 36px;
}
.td-num {
  padding: 7px 12px;
  font-size: 11px;
  color: #94a3b8;
}
.td-input {
  padding: 7px 8px;
}
.td-result {
  padding: 7px 12px;
}
.td-del {
  padding: 7px 8px;
}
.req {
  color: #ff4d4f;
  margin-left: 2px;
}
.opt {
  font-size: 10px;
  font-weight: 400;
  color: #94a3b8;
  margin-left: 3px;
  text-transform: none;
  letter-spacing: 0;
}
.m-table-footer {
  padding: 8px 12px;
  border-top: 1px solid #f0f0f0;
  background: #fafafa;
}
.add-row-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 13px;
  color: #64748b;
  background: none;
  border: none;
  cursor: pointer;
  transition: color 0.1s;
}
.add-row-btn:hover {
  color: #1677ff;
}
.del-btn {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 5px;
  border: none;
  background: none;
  cursor: pointer;
  color: #cbd5e1;
  transition: all 0.1s;
}
.del-btn:hover {
  background: #fef2f2;
  color: #ef4444;
}
.del-btn:disabled {
  opacity: 0.25;
  cursor: not-allowed;
}

/* 文件导入 */
.file-area {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.file-toolbar {
  display: flex;
}
.dropzone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
  border: 2px dashed #f0f0f0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}
.dropzone:hover {
  border-color: #1677ff;
  background: #f0f7ff;
}
.drop-icon {
  width: 44px;
  height: 44px;
  color: #cbd5e1;
  margin-bottom: 12px;
}
.drop-title {
  font-size: 14px;
  font-weight: 500;
  color: #475569;
}
.drop-hint {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 4px;
}
.file-hidden {
  display: none;
}
.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.preview-count {
  font-size: 13px;
  color: #64748b;
}
.preview-count b {
  font-weight: 600;
  color: #1e293b;
}
.preview-table-wrap {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow-x: auto;
  overflow-y: hidden;
}

/* 模板生成 */
.template-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr);
  gap: 16px;
}
.template-card {
  border: 1px solid #f0f0f0;
  border-radius: 14px;
  background: #fff;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.template-card-preview {
  background: linear-gradient(180deg, #fcfdff 0%, #ffffff 100%);
}
.template-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.template-card-title {
  font-size: 15px;
  font-weight: 700;
  color: #1e293b;
}
.template-card-sub {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.45;
  color: #94a3b8;
}
.template-count-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.template-count-pill {
  min-width: 44px;
  height: 30px;
  padding: 0 10px;
  border: 1px solid #dbeafe;
  border-radius: 999px;
  background: #eff6ff;
  color: #2563eb;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}
.template-count-pill:hover,
.template-count-pill.active {
  border-color: #2563eb;
  background: #2563eb;
  color: #fff;
}
.template-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 14px;
  border-top: 1px solid #f8fafc;
}
.template-section:first-of-type {
  padding-top: 0;
  border-top: none;
}
.template-section-title {
  font-size: 13px;
  font-weight: 700;
  color: #334155;
}
.template-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.template-form-grid-top {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.template-password-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.template-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.template-field label {
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
}
.template-field-span {
  grid-column: 1 / -1;
}
.template-field-disabled {
  opacity: 0.55;
}
.template-static-value {
  min-height: 32px;
  display: flex;
  align-items: center;
  padding: 0 12px;
  border-radius: 8px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  font-size: 13px;
  color: #334155;
}
.template-switch-row {
  min-height: 32px;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.template-switch-row .template-static-value {
  flex: 1;
  min-width: 160px;
}
.template-toggle-field .template-switch-row {
  align-items: flex-start;
  flex-direction: column;
  gap: 8px;
}
.template-toggle-field .template-switch-row .template-static-value {
  width: 100%;
  min-width: 0;
  min-height: 36px;
  padding: 8px 10px;
  line-height: 1.35;
  white-space: normal;
  word-break: break-word;
}
.template-tip {
  font-size: 12px;
  line-height: 1.45;
  color: #94a3b8;
}
.template-mode-list {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.template-mode-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  padding: 12px;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  background: #fff;
  cursor: pointer;
  transition: all 0.15s ease;
}
.template-mode-card:hover {
  border-color: #93c5fd;
  background: #f8fbff;
}
.template-mode-card.active {
  border-color: #2563eb;
  background: rgba(37, 99, 235, 0.08);
  box-shadow: inset 0 0 0 1px rgba(37, 99, 235, 0.1);
}
.template-mode-title {
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
}
.template-mode-desc {
  font-size: 12px;
  line-height: 1.45;
  color: #64748b;
  text-align: left;
}
.template-risk-banner {
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid #fed7aa;
  background: #fff7ed;
  color: #c2410c;
  font-size: 12px;
  line-height: 1.5;
}
.template-preview-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.template-preview-meta span {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  background: #f8fafc;
  color: #475569;
  font-size: 12px;
  font-weight: 500;
}
.template-result-note {
  margin-top: -4px;
  font-size: 12px;
  line-height: 1.5;
  color: #94a3b8;
}
.template-result-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.template-result-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12px;
  font-weight: 600;
  color: #475569;
}
.template-summary-success {
  color: #16a34a;
}
.template-summary-failed {
  color: #dc2626;
}
.template-result-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 分配弹窗 */
.assign-body {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.assign-user-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid #f0f0f0;
}
.assign-user-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.assign-username {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
}
.assign-imid {
  font-family: "Fira Code", monospace;
  font-size: 11px;
  color: #94a3b8;
}
.assign-form {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.assign-label {
  font-size: 13px;
  font-weight: 500;
  color: #475569;
}

/* 弹窗底部 */
.modal-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.batch-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafcff;
}

.batch-count {
  font-size: 12px;
  font-weight: 500;
  color: #475569;
}

.batch-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.assign-batch-row {
  padding: 12px 14px;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid #f0f0f0;
}

.assign-batch-count {
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
}

/* 用户同步弹窗 */
.sync-body {
  min-height: 200px;
}
.sync-empty {
  text-align: center;
  padding: 60px 20px;
  color: #94a3b8;
  font-size: 14px;
}
.sync-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  gap: 12px;
}
.sync-count {
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
}
.sync-hint {
  font-size: 12px;
  color: #94a3b8;
}
.sync-table-wrap {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
  max-height: 420px;
  overflow-y: auto;
}
.sync-table {
  width: 100%;
  font-size: 13px;
  border-collapse: collapse;
}
.sync-table thead {
  background: #fafafa;
  position: sticky;
  top: 0;
  z-index: 1;
}
.sync-table th {
  text-align: left;
  padding: 10px 12px;
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
  border-bottom: 1px solid #f0f0f0;
  white-space: nowrap;
}
.sync-table td {
  padding: 10px 12px;
  border-bottom: 1px solid #f0f0f0;
  vertical-align: middle;
}
.sync-table tbody tr:last-child td {
  border-bottom: none;
}
.sync-table tbody tr:hover {
  background: #f8fafc;
}
.sync-link {
  font-size: 12px;
  color: #1677ff;
  text-decoration: none;
}
.sync-link:hover {
  text-decoration: underline;
}
.sync-status-badge {
  display: inline-flex;
  align-items: center;
  font-size: 11px;
  font-weight: 500;
  padding: 0 8px;
  min-height: 22px;
  border-radius: 999px;
  border: 1px solid transparent;
}
.sync-status-missing {
  color: #ff4d4f;
  border-color: #ffa39e;
  background: #fff2f0;
}
.sync-status-nickname {
  color: #1677ff;
  border-color: #91caff;
  background: #e6f4ff;
}
.sync-status-avatar {
  color: #722ed1;
  border-color: #d3adf7;
  background: #f9f0ff;
}
.sync-status-changed {
  color: #d46b08;
  border-color: #ffd591;
  background: #fff7e6;
}
.sync-status-synced {
  color: #52c41a;
  border-color: #b7eb8f;
  background: #f6ffed;
}
.action-toggle {
  font-size: 12px;
  font-weight: 500;
  padding: 3px 10px;
  border-radius: 5px;
  border: 1px solid;
  cursor: pointer;
  transition: all 0.15s;
}
.action-toggle:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.action-skip {
  background: #fafafa;
  color: #94a3b8;
  border-color: #e2e8f0;
}
.action-skip:hover:not(:disabled) {
  background: #f1f5f9;
  color: #64748b;
}
.action-update {
  background: #fff7e6;
  color: #d46b08;
  border-color: #ffd591;
}

@media (max-width: 960px) {
  .page-header-actions {
    width: 100%;
    justify-content: stretch;
  }

  .page-header-actions :deep(.el-button) {
    flex: 1 1 160px;
  }

  .card-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-wrap {
    max-width: none;
  }

  .filter-search-input {
    min-width: 140px;
  }

  .filter-helper {
    padding: 0 4px;
  }

  .filter-helper-divider {
    display: none;
  }

  .users-toolbar-right {
    width: 100%;
    justify-content: space-between;
    margin-left: 0;
  }

  .total-count {
    margin-left: 0;
  }

  .template-layout {
    grid-template-columns: 1fr;
  }

  .template-card-head,
  .template-result-head {
    flex-direction: column;
    align-items: stretch;
  }

  .template-count-pills,
  .template-result-actions {
    width: 100%;
  }

  .template-form-grid,
  .template-form-grid-top,
  .template-password-grid,
  .template-mode-list {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 767px) {
  .page {
    padding: 0;
  }

  .page-header-actions :deep(.el-button) {
    flex: 1 1 calc(50% - 4px);
    min-width: 0;
  }

  .filter-box {
    gap: 8px;
    padding: 8px 10px;
  }

  .filter-search-input {
    min-width: 100%;
    order: 2;
  }

  .filter-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    width: 100%;
    order: 3;
  }

  .filter-inline-btn {
    flex: 1 1 calc(50% - 4px);
    min-height: 36px;
  }

  .filter-helper.mobile-only-helper {
    padding: 0 2px;
    line-height: 1.45;
  }

  .filter-dropdown {
    width: min(100%, 320px);
  }

  .batch-toolbar,
  .batch-actions,
  .modal-actions,
  .preview-header,
  .template-result-head,
  .sync-header {
    flex-direction: column;
    align-items: stretch;
  }

  .batch-actions,
  .modal-actions,
  .template-result-actions {
    width: 100%;
  }

  .batch-actions :deep(.el-button),
  .modal-actions :deep(.el-button),
  .template-result-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }

  .action-btns {
    flex-direction: column;
  }

  .more-btn {
    width: 100%;
  }

  .m-table-wrap,
  .sync-table-wrap {
    overflow-x: auto;
  }

  .m-table,
  .sync-table {
    min-width: 680px;
  }

  .sync-table-wrap {
    max-height: none;
  }
}
.action-update:hover:not(:disabled) {
  background: #ffe7ba;
}
.status-online {
  color: #52c41a;
  border-color: #b7eb8f;
  background: #f6ffed;
}
.status-offline {
  color: #94a3b8;
  border-color: #e2e8f0;
  background: #f8fafc;
}
</style>
