import { TUIStore, TUIUserService, StoreName } from '@tencentcloud/chat-uikit-engine-lite';

type Listener = { ids: Set<string>; change: (rows: any[]) => void };
const leases = new Map<symbol, Listener>();
const subscribed = new Set<string>();
const versions = new Map<string, number>();
let owner = '';
let revision = 0;
let tail = Promise.resolve();
let retry: ReturnType<typeof setTimeout> | undefined;
let failures = 0;
const currentOwner = () => String(TUIStore.getData(StoreName.USER, 'userProfile')?.userID || '');
const batches = (ids: string[]) => Array.from({ length: Math.ceil(ids.length / 100) }, (_, i) => ids.slice(i * 100, i * 100 + 100));

TUIStore.watch(StoreName.USER, { userStatusList: (list: Map<string, any>) => {
  list?.forEach((_status, id) => versions.set(id, (versions.get(id) || 0) + 1));
} });

async function synchronize() {
  const epoch = revision;
  const account = currentOwner();
  if (!account) return;
  if (owner !== account) { owner = account; subscribed.clear(); versions.clear(); }
  const wanted = new Set([...leases.values()].flatMap(lease => [...lease.ids]));
  const active = () => epoch === revision && currentOwner() === account;
  for (const ids of batches([...subscribed].filter(id => !wanted.has(id)))) {
    await TUIUserService.unsubscribeUserStatus({ userIDList: ids });
    if (currentOwner() !== account) return;
    ids.forEach(id => subscribed.delete(id));
    if (!active()) return;
  }
  for (const ids of batches([...wanted].filter(id => !subscribed.has(id)))) {
    const result = await TUIUserService.subscribeUserStatus({ userIDList: ids });
    if (currentOwner() !== account) return;
    if (!result || result.code && Number(result.code) !== 0) throw new Error('在线订阅未确认');
    const rejected = new Set<string>(result?.data?.failedUserIDList || result?.data?.failedUserList?.map((row: any) => row.userID) || []);
    ids.filter(id => !rejected.has(id)).forEach(id => subscribed.add(id));
    if (!active()) return;
    if (rejected.size) throw new Error('部分在线订阅未确认');
  }
  for (const ids of batches([...wanted])) {
    const before = new Map(ids.map(id => [id, versions.get(id) || 0]));
    const result = await TUIUserService.getUserStatus({ userIDList: ids });
    if (!active()) return;
    const data = result?.data;
    const rows = (Array.isArray(data) ? data : data?.successUserStatusList || data?.successUserList || [])
      .filter((row: any) => (versions.get(row.userID) || 0) === before.get(row.userID));
    leases.forEach(lease => lease.change(rows.filter((row: any) => lease.ids.has(row.userID))));
  }
  failures = 0;
}

function schedule() {
  tail = tail.catch(() => {}).then(synchronize).catch(() => {
    // Retain unknown status, and retry only a bounded number of times.
    if (!retry && leases.size && failures++ < 2) retry = setTimeout(() => {
      retry = undefined;
      schedule();
    }, Math.min(30000, 1000 * 2 ** failures) + Math.random() * 500);
  });
  return tail;
}

export function updatePresenceLease(key: symbol, ids: string[], change: (rows: any[]) => void) {
  leases.set(key, { ids: new Set(ids.filter(Boolean)), change });
  revision++;
  return schedule();
}
export function releasePresenceLease(key: symbol) {
  leases.delete(key);
  revision++;
  return schedule();
}
function revalidate() {
  if (document.hidden || !leases.size) return;
  subscribed.clear();
  failures = 0;
  revision++;
  void schedule();
}
document.addEventListener('visibilitychange', revalidate);
window.addEventListener('online', revalidate);
