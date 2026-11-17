export interface RefreshRequestOptions {
  force?: boolean;
  ensureAfterInflight?: boolean;
}

interface RefreshState {
  inFlight: Promise<boolean> | null;
  lastFinishedAt?: number;
  activeRun?: symbol;
}

interface RefreshCoordinatorOptions {
  minIntervalMs: number;
  now?: () => number;
}

export function createRefreshCoordinator({
  minIntervalMs,
  now = Date.now,
}: RefreshCoordinatorOptions) {
  const states = new Map<string, RefreshState>();

  function getState(key: string) {
    let state = states.get(key);
    if (!state) {
      state = { inFlight: null };
      states.set(key, state);
    }
    return state;
  }

  function startRefresh(
    state: RefreshState,
    task: () => Promise<void>
  ): Promise<boolean> {
    const runToken = Symbol('refresh-run');
    const refreshPromise = (async () => {
      try {
        await task();
        return true;
      } finally {
        state.lastFinishedAt = now();
        if (state.activeRun === runToken) {
          state.inFlight = null;
          state.activeRun = undefined;
        }
      }
    })();
    state.inFlight = refreshPromise;
    state.activeRun = runToken;
    return refreshPromise;
  }

  async function run(
    key: string,
    task: () => Promise<void>,
    options: RefreshRequestOptions = {}
  ) {
    const state = getState(key);
    if (state.inFlight) {
      await state.inFlight;
      if (!options.ensureAfterInflight) {
        return false;
      }
      if (state.inFlight) {
        await state.inFlight;
        return false;
      }
      return startRefresh(state, task);
    }

    const isFresh =
      state.lastFinishedAt !== undefined &&
      now() - state.lastFinishedAt < minIntervalMs;
    if (!options.force && isFresh) {
      return false;
    }
    return startRefresh(state, task);
  }

  function clear() {
    states.clear();
  }

  return { run, clear };
}
