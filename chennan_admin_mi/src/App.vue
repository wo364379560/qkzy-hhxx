<template>
  <router-view />
</template>

<style>
/* Fira Code for data display */
@import url('https://fonts.googleapis.com/css2?family=Fira+Code:wght@400;500;600&display=swap');

/* ── Element Plus Theme Variables ── */
:root {
  --el-color-primary: #1677ff;
  --el-color-primary-light-3: #4096ff;
  --el-color-primary-light-5: #69b1ff;
  --el-color-primary-light-7: #91caff;
  --el-color-primary-light-8: #bae0ff;
  --el-color-primary-light-9: #e6f4ff;
  --el-color-primary-dark-2: #0958d9;
  --el-border-radius-base: 6px;
  --el-font-family: 'Inter Variable', -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Microsoft YaHei', sans-serif;
  --app-page-padding-x: 28px;
  --app-page-padding-y: 24px;
  --app-card-header-padding-x: 20px;
  --app-card-header-padding-y: 16px;
  --app-page-gap: 20px;
  --app-toolbar-gap: 12px;
  --app-touch-size: 44px;
  --app-safe-top: env(safe-area-inset-top, 0px);
  --app-safe-right: env(safe-area-inset-right, 0px);
  --app-safe-bottom: env(safe-area-inset-bottom, 0px);
  --app-safe-left: env(safe-area-inset-left, 0px);
}

*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

html, body, #app {
  min-height: 100%;
}

body {
  font-family: var(--el-font-family);
  background: #f5f7fa;
  color: #1d2433;
  -webkit-font-smoothing: antialiased;
  text-rendering: optimizeLegibility;
  overscroll-behavior-y: contain;
  overflow-x: hidden;
}

/* ── Element Plus Component Overrides ── */
.el-card {
  border: 1px solid #edf0f5 !important;
  border-radius: 8px !important;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05) !important;
}
.el-card__header {
  border-bottom: 1px solid #f0f2f7 !important;
  padding: 16px 20px !important;
  font-weight: 600;
  font-size: 14px;
  color: #1d2433;
}
.el-card__body { padding: 20px !important; }

.el-table {
  font-size: 13px;
  --el-table-padding: 7px 0;
}
.el-table th.el-table__cell {
  background: #f8faff !important;
  color: #6b7280;
  font-weight: 500;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 7px 0 !important;
}
.el-table td.el-table__cell {
  border-bottom: 1px solid #f3f4f6 !important;
  padding: 7px 0 !important;
}
.el-table--striped .el-table__body tr.el-table__row--striped td.el-table__cell {
  background: #f4f7ff !important;
}

.el-dialog { border-radius: 10px !important; overflow: hidden; }
.el-dialog__header {
  padding: 20px 24px 16px !important;
  border-bottom: 1px solid #f0f2f7;
  margin-right: 0 !important;
}
.el-dialog__title { font-weight: 600; font-size: 15px; color: #1d2433; }
.el-dialog__body { padding: 24px !important; }
.el-dialog__footer { padding: 16px 24px !important; border-top: 1px solid #f0f2f7; }

.el-button { font-weight: 500; transition: all 0.2s ease; }

/* ── Global Helpers ── */

/* Content area page layout */
.page {
  padding: var(--app-page-padding-y) var(--app-page-padding-x);
  display: flex;
  flex-direction: column;
  gap: var(--app-page-gap);
  min-height: 100%;
}

/* Card header: title + action button */
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: var(--app-card-header-padding-y) var(--app-card-header-padding-x);
  border-bottom: 1px solid #f0f2f7;
}

.page-title {
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.3;
  letter-spacing: -0.01em;
}

.page-sub {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 3px;
}

.card {
  background: #fff;
  border: 1px solid #e8ecf2;
  border-radius: 10px;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.04), 0 4px 12px rgba(15, 23, 42, 0.04);
  overflow: hidden;
}

.font-code { font-family: 'Fira Code', 'Courier New', monospace; }

.input {
  display: block;
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  font-family: var(--el-font-family);
  color: #1e293b;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.input:focus { border-color: #1677ff; box-shadow: 0 0 0 3px rgba(22, 119, 255, 0.1); }
.input::placeholder { color: #94a3b8; }
.input[readonly] { background: #f8fafc; color: #64748b; cursor: not-allowed; }

.btn-primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  padding: 10px 16px;
  font-size: 14px;
  font-weight: 500;
  font-family: var(--el-font-family);
  color: #fff;
  background: #1677ff;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
  gap: 6px;
}
.btn-primary:hover { background: #4096ff; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.table-scroll-x {
  width: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  -webkit-overflow-scrolling: touch;
}

.table-scroll-x::-webkit-scrollbar {
  height: 8px;
}

.table-scroll-x::-webkit-scrollbar-thumb {
  background: #dbe3ef;
  border-radius: 999px;
}

.mobile-only {
  display: none !important;
}

.desktop-only {
  display: initial !important;
}

.badge-active    { display: inline-flex; align-items: center; background: #f6ffed; color: #52c41a; border: 1px solid #b7eb8f; font-size: 12px; font-weight: 500; padding: 0 9px; height: 24px; border-radius: 6px; }
.badge-banned    { display: inline-flex; align-items: center; background: #fff2f0; color: #ff4d4f; border: 1px solid #ffa39e; font-size: 12px; font-weight: 500; padding: 0 9px; height: 24px; border-radius: 6px; }
.badge-admin     { display: inline-flex; align-items: center; background: #eff6ff; color: #1677ff; border: 1px solid #bfdbfe; font-size: 12px; font-weight: 500; padding: 0 9px; height: 24px; border-radius: 6px; }
.badge-assistant { display: inline-flex; align-items: center; background: #faf5ff; color: #9333ea; border: 1px solid #e9d5ff; font-size: 12px; font-weight: 500; padding: 0 9px; height: 24px; border-radius: 6px; }
.badge-inactive  { display: inline-flex; align-items: center; background: #fafafa; color: #8c8c8c; border: 1px solid #d9d9d9; font-size: 12px; font-weight: 500; padding: 0 9px; height: 24px; border-radius: 6px; }

@keyframes spin { to { transform: rotate(360deg); } }
.animate-spin { animation: spin 1s linear infinite; }

@media (max-width: 1199px) {
  :root {
    --app-page-padding-x: 18px;
    --app-page-padding-y: 18px;
    --app-card-header-padding-x: 18px;
    --app-card-header-padding-y: 16px;
    --app-page-gap: 16px;
  }

  .card-header {
    align-items: flex-start;
    flex-wrap: wrap;
  }
}

@media (max-width: 767px) {
  :root {
    --app-page-padding-x: 12px;
    --app-page-padding-y: 12px;
    --app-card-header-padding-x: 14px;
    --app-card-header-padding-y: 14px;
    --app-page-gap: 12px;
    --app-toolbar-gap: 10px;
  }

  body {
    font-size: 14px;
  }

  .page-title {
    font-size: 15px;
  }

  .page-sub {
    font-size: 12px;
    line-height: 1.45;
  }

  .card {
    border-radius: 12px;
  }

  .card-header {
    gap: 10px;
  }

  .el-card__header,
  .el-card__body,
  .el-dialog__header,
  .el-dialog__body,
  .el-dialog__footer {
    padding-left: 14px !important;
    padding-right: 14px !important;
  }

  .el-dialog {
    width: calc(100vw - 24px) !important;
    margin-top: max(12px, var(--app-safe-top)) !important;
  }

  .table-scroll-x > .el-table {
    min-width: 680px;
  }

  .el-pagination {
    width: 100%;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 8px;
  }

  .el-button,
  .btn-primary,
  .el-input__wrapper,
  .el-select__wrapper {
    min-height: var(--app-touch-size);
  }

  .mobile-only {
    display: initial !important;
  }

  .desktop-only {
    display: none !important;
  }
}
</style>
