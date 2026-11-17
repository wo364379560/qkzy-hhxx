import { CHAT_API_BASE_URL, SDK_APP_ID } from "@/config/runtime";
// Store the unexpanded submission plan before the first network mutation. A
// partially uploaded parent can resume after refresh without inventing new IDs.
const databaseName = `rui-forward-plans:${encodeURIComponent(CHAT_API_BASE_URL)}:${SDK_APP_ID}`;
async function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    if (typeof indexedDB === "undefined") { reject(new Error("浏览器不支持保存转发任务，请使用支持 IndexedDB 的浏览器")); return; }
    const request = indexedDB.open(databaseName, 1);
    request.onupgradeneeded = () => request.result.createObjectStore("plans", {keyPath:"key"});
    request.onerror = () => reject(new Error("无法保存转发任务，请检查浏览器存储空间后重试"));
    request.onblocked = () => reject(new Error("任务存储被其他窗口占用，请关闭旧窗口后重试"));
    request.onsuccess = () => resolve(request.result);
  });
}
async function transact(owner: string, id: string, mode: "read" | "put" | "delete", plan?: unknown): Promise<any> {
  const db = await openDatabase();
  try {
    return await new Promise((resolve,reject) => {
      const tx = db.transaction("plans", mode === "read" ? "readonly" : "readwrite");
      const store = tx.objectStore("plans"), key = `${owner}:${id}`;
      const req = mode === "read" ? store.get(key) : mode === "put" ? store.put({key,plan}) : store.delete(key);
      let result: any;
      req.onsuccess = () => {result = mode === "read" ? req.result?.plan : undefined;};
      tx.oncomplete = () => resolve(result);
      tx.onabort = tx.onerror = () => reject(new Error("转发任务本地存储失败，尚未确认的内容会在恢复后继续提交"));
    });
  } finally { db.close(); }
}
export const saveForwardPlan = (owner: string, id: string, plan: unknown) => transact(owner,id,"put",plan);
export const loadForwardPlan = (owner: string, id: string) => transact(owner,id,"read");
export const deleteForwardPlan = (owner: string, id: string) => transact(owner,id,"delete");
