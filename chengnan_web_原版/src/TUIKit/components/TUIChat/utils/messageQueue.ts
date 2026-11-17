export type QueueStatus =
  | "pending"
  | "sending"
  | "success"
  | "failed"
  | "unknown";
export interface MessageQueueJob {
  lane: "text" | "attachment";
  isActive: () => boolean;
  run: () => Promise<"success" | "failed" | "unknown">;
  status?: QueueStatus;
  label?: string;
}

/** Two independently bounded lanes: an upload cannot stall subsequent text.
 * A slow operation stays in its lane until the SDK settles. The deadline only
 * changes feedback; it does not authorize retry or release network capacity.
 */
export function createMessageQueue(
  onChange: () => void = () => {},
  capacity = 20,
  slowMs = 15000,
  retainFailures = false
) {
  const waiting: MessageQueueJob[] = [];
  const running = new Set<MessageQueueJob>();
  const history: MessageQueueJob[] = [];
  const lanes = new Set<string>();
  const listeners = new Set([onChange]);
  const notify = () => listeners.forEach((listener) => listener());
  function prune() {
    for (let i = waiting.length - 1; i >= 0; i--) {
      if (!waiting[i].isActive()) waiting.splice(i, 1);
    }
    for (let i = history.length - 1; i >= 0; i--) {
      if (!history[i].isActive()) history.splice(i, 1);
    }
  }
  const hasCapacity = (count = 1) => {
    prune();
    const reserved = retainFailures
      ? history.filter((job) => job.isActive() && job.status !== "success")
          .length
      : waiting.length + running.size;
    return reserved + count <= capacity;
  };
  async function drain(lane: "text" | "attachment") {
    if (lanes.has(lane)) return;
    lanes.add(lane);
    try {
      while (true) {
        prune();
        const index = waiting.findIndex((job) => job.lane === lane);
        if (index < 0) break;
        const job = waiting.splice(index, 1)[0];
        if (!job.isActive()) continue;
        job.status = "sending";
        running.add(job);
        notify();
        const timer = setTimeout(() => {
          if (job.isActive()) {
            job.status = "unknown";
            notify();
          }
        }, slowMs);
        try {
          const status = await job.run();
          if (job.isActive()) job.status = status;
        } catch {
          if (job.isActive()) job.status = "unknown";
        } finally {
          clearTimeout(timer);
          running.delete(job);
          notify();
        }
      }
    } finally {
      lanes.delete(lane);
      notify();
    }
  }
  return {
    subscribe(listener: () => void) {
      listeners.add(listener);
      listener();
      return () => {
        listeners.delete(listener);
      };
    },
    hasCapacity,
    retry(job: MessageQueueJob) {
      if (
        job.status !== "failed" ||
        !history.includes(job) ||
        !job.isActive() ||
        running.has(job) ||
        waiting.includes(job) ||
        !hasCapacity(retainFailures ? 0 : 1)
      )
        return false;
      job.status = "pending";
      waiting.push(job);
      notify();
      void Promise.resolve().then(() => {
        void drain(job.lane);
      });
      return true;
    },
    dismiss(job: MessageQueueJob) {
      if (running.has(job) || waiting.includes(job) || !history.includes(job))
        return false;
      history.splice(history.indexOf(job), 1);
      notify();
      return true;
    },
    get jobs() {
      prune();
      return [...history];
    },
    get pendingCount() {
      prune();
      return waiting.length + running.size;
    },
    enqueue(jobs: MessageQueueJob[]) {
      if (!hasCapacity(jobs.length) || jobs.some((job) => !job.isActive()))
        return false;
      for (const job of jobs) {
        job.status = "pending";
        waiting.push(job);
        history.push(job);
      }
      // Display history is bounded independently from admitted network work.
      while (history.length > capacity * 2) {
        const index = history.findIndex(
          (job) =>
            !waiting.includes(job) &&
            !running.has(job) &&
            (!retainFailures || job.status === "success")
        );
        if (index < 0) break;
        history.splice(index, 1);
      }
      notify();
      void Promise.resolve().then(() => {
        void drain("text");
        void drain("attachment");
      });
      return true;
    },
  };
}

// Components are replaced by multi-select/contact navigation. Keep admission,
// physical in-flight lanes and result history in one per-window queue.
let sharedMessageQueue: ReturnType<typeof createMessageQueue> | undefined;
export function getSharedMessageQueue() {
  if (!sharedMessageQueue) {
    sharedMessageQueue = createMessageQueue(() => {}, 20, 15000, true);
    if (typeof window !== "undefined")
      window.addEventListener("beforeunload", (event) => {
        if (
          sharedMessageQueue?.pendingCount ||
          sharedMessageQueue?.jobs.some((job) => job.status !== "success")
        ) {
          event.preventDefault();
          event.returnValue = "";
        }
      });
  }
  return sharedMessageQueue;
}
