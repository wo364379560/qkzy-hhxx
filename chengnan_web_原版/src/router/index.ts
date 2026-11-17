import { installChunkRecovery } from "./chunk-recovery";
import { createRouter, createWebHashHistory } from "vue-router";
import { useAuthSession } from "@/features/auth/session";

const authSession = useAuthSession();

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      name: "boot",
      component: () => import("../views/Boot.vue"),
    },
    {
      path: "/welcome",
      name: "welcome",
      component: () => import("../views/Welcome.vue"),
    },
    {
      path: "/login",
      name: "login",
      component: () => import("../views/Login.vue"),
    },
    {
      path: "/register",
      name: "register",
      component: () => import("../views/Register.vue"),
    },
    {
      path: "/forgot-password",
      name: "forgot-password",
      component: () => import("../views/ForgotPassword.vue"),
    },
    {
      path: "/change-password",
      name: "change-password",
      component: () => import("../views/ChangePassword.vue"),
      meta: {
        requiresAuth: true,
      },
    },
    {
      path: "/first-setup",
      name: "first-setup",
      component: () => import("../views/FirstSetup.vue"),
      meta: {
        requiresAuth: true,
      },
    },
    {
      path: "/home",
      name: "home",
      component: () => import("../views/Home.vue"),
      meta: {
        requiresAuth: true,
      },
    },
  ],
});

const guestRouteNames = new Set([
  "boot",
  "welcome",
  "login",
  "register",
  "forgot-password",
]);

const DYNAMIC_IMPORT_RELOAD_KEY = "webDynamicImportReloaded";

installChunkRecovery(router);

router.beforeEach(async (to) => {
  await authSession.ensureInitialized();

  const hasSession = authSession.isAuthenticated.value;
  const authenticatedRouteName = authSession.resolveAuthenticatedRouteName();

  if (to.name === "boot") {
    return hasSession ? { name: authenticatedRouteName } : { name: "welcome" };
  }

  if (!hasSession) {
    if (to.meta.requiresAuth) {
      return { name: "welcome" };
    }
    return true;
  }

  if (to.name === "change-password" || to.name === "first-setup") {
    return { name: "home" };
  }

  if (to.name && guestRouteNames.has(String(to.name))) {
    return { name: authenticatedRouteName };
  }

  return true;
});

export default router;
