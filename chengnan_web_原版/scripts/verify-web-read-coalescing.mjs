import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import ts from 'typescript';
import axios from 'axios';
const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const nativeRequire = createRequire(import.meta.url), loaded = new Map();
const pending = [];
axios.defaults.adapter = config => new Promise((resolve, reject) => pending.push({
  config, reject, finish: value => resolve({status:200,statusText:'OK',headers:{},config,data:{code:0,data:value}}),
}));
function load(file) {
  const path = resolve(root, file);
  if (loaded.has(path)) return loaded.get(path).exports;
  const module = {exports:{}}; loaded.set(path,module);
  const code = ts.transpileModule(readFileSync(path,'utf8'), {compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2020,esModuleInterop:true}}).outputText;
  const require = name => name === '@/config/runtime' ? {CHAT_API_BASE_URL:'https://api.zyxhim.com/api/v1/chat',CHAT_API_TIMEOUT_MS:15000}
    : name === '@/features/auth/device' ? {getOrCreateDeviceID:()=> 'fixture'}
    : name.startsWith('@/') ? load(`src/${name.slice(2)}.ts`)
    : name.startsWith('.') ? load(resolve(dirname(path),`${name}.ts`)) : nativeRequire(name);
  new Function('require','module','exports',code)(require,module,module.exports);
  return module.exports;
}
const api = load('src/api/index.ts');
const tick = () => new Promise(resolve => setImmediate(resolve));
for (const [read, write, url] of [
  [api.getProfile, () => api.updateProfile({nickname:'fixture'}), '/profile'],
  [api.getConversationCustomPins, () => api.pinConversationCustom({conversationID:'C2Cfixture'}), '/conversation/pins'],
  [api.getConversationCustomPins, () => api.unpinConversationCustom({conversationID:'C2Cfixture'}), '/conversation/pins'],
]) {
  api.setChatAuthToken('fixture-a', 'account-a'); pending.length=0;
  const first=read(), second=read(); await tick();
  assert.equal(pending.length,1,'concurrent component reads must share one HTTP request');
  assert.equal(pending[0].config.url,url);
  pending[0].finish({version:1});
  assert.deepEqual(await first,await second);
  const next=read(); await tick();
  assert.equal(pending.length,2,'completed profile/permission reads must not be cached');
  pending[1].finish({version:2}); await next;
  const stale=read(); await tick();
  const mutation=write(); await tick(); pending[3].finish({version:3}); await mutation;
  const fresh=read(); await tick(); assert.equal(pending.length,5,'write must detach old read');
  pending[4].finish({version:4}); pending[2].finish({version:2});
  assert.deepEqual(await fresh,{version:4}); await stale;
  const failed=read(); const failure=assert.rejects(failed); await tick(); pending[5].reject(Error('fixture failure')); await failure;
  const retry=read(); await tick(); pending[6].finish({version:5}); await retry;
  const accountA=read(); const staleAccount=assert.rejects(accountA, error => error.code === 'SESSION_CHANGED'); await tick();
  api.setChatAuthToken('fixture-b', 'account-b'); const accountB=read(); await tick();
  assert.equal(pending.length,9,'new account must never join old account request');
  assert.equal(pending[8].config.headers.Authorization,'Bearer fixture-b');
  pending[8].finish({account:'b'}); pending[7].finish({account:'a'});
  assert.deepEqual(await accountB,{account:'b'}); await staleAccount;
}
console.log('PASS: concurrent profile/pins reads coalesce; no completed permission cache; writes and account switches invalidate; failures retry.');
