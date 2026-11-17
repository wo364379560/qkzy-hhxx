export const ESTIMATED_MESSAGE_HEIGHT = 96;
export const MAX_RENDERED_MESSAGES = 100;
export const MESSAGE_OVERSCAN = 10;

export function messageOffsets(ids: string[], heights: Map<string, number>): number[] {
  const offsets = [0];
  for (const id of ids) offsets.push(offsets[offsets.length - 1] + (heights.get(id) || ESTIMATED_MESSAGE_HEIGHT));
  return offsets;
}

export function messageIndexAt(offsets: number[], position: number): number {
  let low = 0;
  let high = Math.max(0, offsets.length - 2);
  while (low < high) {
    const middle = Math.ceil((low + high) / 2);
    if (offsets[middle] <= position) low = middle;
    else high = middle - 1;
  }
  return low;
}

export function messageWindow(offsets: number[], top: number, height: number) {
  const count = offsets.length - 1;
  if (!count) return { start: 0, end: 0, top: 0, bottom: 0 };
  const first = messageIndexAt(offsets, Math.max(0, top));
  const last = messageIndexAt(offsets, Math.max(0, top) + height);
  const start = Math.max(0, first - MESSAGE_OVERSCAN);
  const end = Math.min(count, start + MAX_RENDERED_MESSAGES, last + MESSAGE_OVERSCAN + 1);
  return { start, end, top: offsets[start], bottom: offsets[count] - offsets[end] };
}
