const DEFAULT_ADMIN_API_BASE_URL = import.meta.env.DEV
  ? '/api/v1/admin' : 'https://api.ruizhihe.com/api/v1/admin';

export const APP_NAME = '睿之禾';

/** r2: production admin stays on CF; development keeps the local Vite proxy. */
export const ADMIN_API_BASE_URL =
  import.meta.env.VITE_ADMIN_API_BASE_URL?.trim() || DEFAULT_ADMIN_API_BASE_URL;
