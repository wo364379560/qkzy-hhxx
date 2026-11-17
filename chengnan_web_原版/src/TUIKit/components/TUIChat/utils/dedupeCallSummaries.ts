interface Message { conversationID?: string; time?: number; from?: string }
interface Call { key: string; duration: number; caller: string }

// Only call summaries enter the index. Time/duration buckets bound the search
// to neighbours, even for long histories containing thousands of calls.
export function dedupeCallSummaries<T extends Message>(list: T[], describe: (message: T) => Call | undefined): T[] {
  const result: T[] = [];
  const records: Array<{ call: Call; time: number; bucket: string } | undefined> = [];
  const buckets = new Map<string, Set<number>>();
  for (const message of list) {
    const call = describe(message);
    const time = Number(message.time || 0);
    if (!call?.key || !Number.isFinite(time) || !Number.isFinite(call.duration)) {
      result.push(message);
      records.push(undefined);
      continue;
    }
    const timeBin = Math.floor(time / 5);
    const durationBin = Math.floor(Math.max(0, call.duration) / 3);
    const prefix = JSON.stringify([message.conversationID, call.key]);
    const bucket = `${prefix}:${timeBin}:${durationBin}`;
    let match = Infinity;
    for (let t = timeBin - 1; t <= timeBin + 1; t++) {
      for (let d = durationBin - 1; d <= durationBin + 1; d++) {
        for (const index of buckets.get(`${prefix}:${t}:${d}`) || []) {
          const previous = records[index]!;
          if (Math.abs(previous.time - time) <= 5
            && (!(previous.call.duration > 0 || call.duration > 0)
              || Math.abs(previous.call.duration - call.duration) <= 3)) {
            match = Math.min(match, index);
          }
        }
      }
    }
    if (match !== Infinity) {
      const previous = records[match]!;
      const previousFrom = String(result[match].from || '').trim();
      const nextFrom = String(message.from || '').trim();
      const previousIsCaller = Boolean(previous.call.caller && previous.call.caller === previousFrom);
      const nextIsCaller = Boolean(call.caller && call.caller === nextFrom);
      const preferNext = previousIsCaller !== nextIsCaller ? nextIsCaller
        : Boolean(previous.call.caller && !call.caller && nextFrom === previous.call.caller);
      if (!preferNext) continue;
      buckets.get(previous.bucket)?.delete(match);
      result[match] = message;
      records[match] = { call, time, bucket };
    } else {
      match = result.length;
      result.push(message);
      records.push({ call, time, bucket });
    }
    if (!buckets.has(bucket)) buckets.set(bucket, new Set());
    buckets.get(bucket)!.add(match);
  }
  return result;
}
