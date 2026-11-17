import { readonly, shallowRef } from "vue";

export function useSingleFlight() {
  const running = shallowRef(false);
  let activePromise: Promise<unknown> | null = null;

  async function run<T>(handler: () => Promise<T>): Promise<T> {
    if (activePromise) {
      return activePromise as Promise<T>;
    }

    running.value = true;
    activePromise = Promise.resolve().then(handler);
    try {
      return await (activePromise as Promise<T>);
    } finally {
      activePromise = null;
      running.value = false;
    }
  }

  return {
    running: readonly(running),
    run,
  };
}
