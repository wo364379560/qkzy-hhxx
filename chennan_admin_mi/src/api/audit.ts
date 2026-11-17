import request from "./request";
import type {
  ApiResponse,
  AdminAccessAudit,
  ChatQuietHoursHitLog,
  AdminLoginAudit,
  PageResult,
  SMSSendLog,
  UserOperationLog,
} from "@/types";

export type AdminAuditCleanupScope = "access" | "login" | "all";

export interface AccessAuditQuery {
  page?: number;
  pageSize?: number;
  keyword?: string;
  role?: string;
  accessType?: string;
  method?: string;
  targetType?: string;
  actionKey?: string;
  targetKeyword?: string;
}

export interface LoginAuditQuery {
  page?: number;
  pageSize?: number;
  keyword?: string;
  role?: string;
  stage?: string;
  result?: string;
}

export interface QuietHoursHitLogQuery {
  page?: number;
  pageSize?: number;
  keyword?: string;
  action?: string;
  source?: string;
}

export interface SMSSendLogQuery {
  page?: number;
  pageSize?: number;
  keyword?: string;
  scene?: string;
  status?: string;
}

export interface UserOperationLogQuery {
  page?: number;
  pageSize?: number;
  keyword?: string;
  operationType?: string;
  status?: string;
  phone?: string;
  imUserId?: string;
  ownerId?: number;
  ipAddress?: string;
  startTime?: string;
  endTime?: string;
}

export interface AuditCleanupPayload {
  scope: AdminAuditCleanupScope;
  before_time: string;
  keyword?: string;
  role?: string;
  accessType?: string;
  method?: string;
  targetType?: string;
  actionKey?: string;
  targetKeyword?: string;
  stage?: string;
  result?: string;
}

export interface AuditCleanupPreview {
  scope: AdminAuditCleanupScope;
  beforeTime: string;
  accessCount: number;
  loginCount: number;
  totalCount: number;
}

export interface AuditCleanupResult {
  scope: AdminAuditCleanupScope;
  beforeTime: string;
  accessDeleted: number;
  loginDeleted: number;
  totalDeleted: number;
}

export function getAdminAccessAudits(params: AccessAuditQuery = {}) {
  return request.get<ApiResponse<PageResult<AdminAccessAudit>>>(
    "/audits/access",
    { params },
  );
}

export function exportAdminAccessAudits(params: AccessAuditQuery = {}) {
  return request.get("/audits/access/export", {
    params,
    responseType: "blob",
  });
}

export function getAdminLoginAudits(params: LoginAuditQuery = {}) {
  return request.get<ApiResponse<PageResult<AdminLoginAudit>>>(
    "/audits/logins",
    { params },
  );
}

export function getChatQuietHoursHitLogs(params: QuietHoursHitLogQuery = {}) {
  return request.get<ApiResponse<PageResult<ChatQuietHoursHitLog>>>(
    "/audits/quiet-hours",
    { params },
  );
}

export function getSMSSendLogs(params: SMSSendLogQuery = {}) {
  return request.get<ApiResponse<PageResult<SMSSendLog>>>(
    "/audits/sms-sends",
    { params },
  );
}

export function getUserOperationLogs(params: UserOperationLogQuery = {}) {
  return request.get<ApiResponse<PageResult<UserOperationLog>>>(
    "/audits/user-operations",
    { params },
  );
}

export function previewAdminAuditCleanup(data: AuditCleanupPayload) {
  return request.post<ApiResponse<AuditCleanupPreview>>(
    "/audits/cleanup/preview",
    data,
  );
}

export function cleanupAdminAudits(data: AuditCleanupPayload) {
  return request.post<ApiResponse<AuditCleanupResult>>("/audits/cleanup", data);
}
