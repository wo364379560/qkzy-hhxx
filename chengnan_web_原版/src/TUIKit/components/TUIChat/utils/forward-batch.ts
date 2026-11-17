export type ForwardBatchResult<T, R> =
  | {
      item: T;
      status: "fulfilled";
      value: R;
    }
  | {
      item: T;
      status: "rejected";
      reason: unknown;
    };

export interface ForwardBatchProgress<T, R> {
  completed: number;
  total: number;
  result: ForwardBatchResult<T, R>;
}

export async function runForwardBatch<T, R>(
  items: T[],
  worker: (item: T, index: number) => Promise<R>,
  options: {
    concurrency?: number;
    onProgress?: (progress: ForwardBatchProgress<T, R>) => void;
  } = {}
): Promise<Array<ForwardBatchResult<T, R>>> {
  if (items.length === 0) {
    return [];
  }

  const concurrency = Math.max(
    1,
    Math.min(Math.floor(options.concurrency || 1), items.length)
  );
  const results = new Array<ForwardBatchResult<T, R>>(items.length);
  let nextIndex = 0;
  let completed = 0;

  const runWorker = async () => {
    while (nextIndex < items.length) {
      const index = nextIndex;
      nextIndex += 1;
      const item = items[index];
      let result: ForwardBatchResult<T, R>;

      try {
        result = {
          item,
          status: "fulfilled",
          value: await worker(item, index),
        };
      } catch (reason) {
        result = {
          item,
          status: "rejected",
          reason,
        };
      }

      results[index] = result;
      completed += 1;
      options.onProgress?.({
        completed,
        total: items.length,
        result,
      });
    }
  };

  await Promise.all(Array.from({ length: concurrency }, () => runWorker()));
  return results;
}

// One shared start-rate gate, including sends inside each target's message loop.
export function createForwardPacer(intervalMs = 250) {
  let tail = Promise.resolve();
  let nextAt = 0;
  return {
    wait() {
      const turn = tail.then(async () => {
        while (nextAt > Date.now()) {
          await new Promise((resolve) =>
            setTimeout(resolve, nextAt - Date.now())
          );
        }
        nextAt = Date.now() + intervalMs;
      });
      tail = turn;
      return turn;
    },
    coolDown(ms = 2000) {
      nextAt = Math.max(nextAt, Date.now() + ms);
    },
  };
}

export async function withForwardTimeout<T>(
  operation: Promise<T>,
  ms = 15000
): Promise<T> {
  let timer: ReturnType<typeof setTimeout>;
  try {
    return await Promise.race([
      operation,
      new Promise<never>((_, reject) => {
        timer = setTimeout(() => reject(new Error("forward_timeout")), ms);
      }),
    ]);
  } finally {
    clearTimeout(timer!);
  }
}
