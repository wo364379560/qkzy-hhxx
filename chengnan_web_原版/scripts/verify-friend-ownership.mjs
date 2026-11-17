import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import ts from 'typescript';

const source = readFileSync(new URL('../src/features/account/friendOwnership.ts', import.meta.url),'utf8');
const flush = async () => { for(let i=0;i<30;i++) await Promise.resolve(); };
const deferred = () => { let resolve; const promise = new Promise(r => { resolve=r; }); return {promise,resolve}; };
function harness() {
  const h={calls:[],profiles:0,peers:0,session:{userID:'pool',sdkAppID:7,sessionFamilyID:'s1'}};
  h.chat={getLoginUser:()=>h.session.userID};
  h.sync=async()=>({assignment:{state:'migrated',request_id:'op'}});
  h.self=async()=>({im_user_id:h.session.userID,owner_id:9,owner_revision:1,is_assigned:true});
  const exports={};
  vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS,esModuleInterop:true}}).outputText,{
    exports,crypto:{randomUUID:()=> 'stable-request'},setTimeout:fn=>{fn();},Promise,Map,WeakMap,
    require:name=>{
      if(name==='vue')return {shallowRef:value=>({value})};
      if(name==='@tencentcloud/chat-uikit-engine-lite')return {get chat(){return h.chat;}};
      if(name==='@/features/auth/session')return {useAuthSession:()=>({currentSession:{get value(){return h.session;}}})};
      if(name==='@/api')return {
        syncFriendOwnerFromMe:async(...args)=>{h.calls.push(args);return h.sync(...args);},
        getProfile:async opts=>{assert.equal(opts.fresh,true);h.profiles++;return h.self();},
        getUserBriefs:async ids=>{h.peers++;return [{im_user_id:ids[0],owner_revision:1,owner_id:9,is_assigned:true}];},
      };
      throw new Error(name);
    },
  });
  h.api=exports; return h;
}
{
  const h=harness(),gate=deferred();h.sync=()=>gate.promise;
  const a=h.api.synchronizeFriendOwner('hidden-assigned'),b=h.api.synchronizeFriendOwner('hidden-assigned');
  assert.equal(a,b,'duplicate SDK/client events must share one request');
  gate.resolve({assignment:{state:'migrated'}});assert.equal((await a).state,'migrated');
  assert.equal(h.calls.length,1);assert.equal(h.profiles,1);assert.equal(h.peers,1);
  assert.equal(h.api.friendOwnerSelfProfile.value.owner_revision,1);
  assert.equal(h.api.friendOwnerTargets.value['hidden-assigned'].owner_id,9);
}
{
  const h=harness();h.sync=async()=>{throw Error('temporary DB failure');};
  assert.equal((await h.api.synchronizeFriendOwner('peer')).state,'pending');
  assert.equal(h.calls.length,1,'a database failure must not replay the binding POST');
  assert.equal(h.profiles,1);assert.equal(h.peers,1);
}
{
  const h=harness();h.sync=async()=>({user:{im_user_id:'peer'}});
  assert.equal((await h.api.synchronizeFriendOwner('peer')).state,'pending','old HTTP 200 must not imply assignment success');
  assert.equal(h.calls.length,1);
}
{
  const h=harness();h.sync=async()=>({assignment:{state:'pending',request_id:'queued'}});
  assert.equal((await h.api.synchronizeFriendOwner('peer')).request_id,'queued');
  assert.equal(h.calls.length,1,'pending work must continue on the server');
  assert.equal(h.profiles,1);assert.equal(h.peers,1);
}
for (const change of ['account','same-user-login','sdk-instance']) {
  const h=harness(),gate=deferred();h.sync=()=>gate.promise;
  const work=h.api.synchronizeFriendOwner('peer');await flush();
  if(change==='account')h.session={...h.session,userID:'other',sessionFamilyID:'s2'};
  if(change==='same-user-login')h.session={...h.session,sessionFamilyID:'s2'};
  if(change==='sdk-instance')h.chat={getLoginUser:()=>h.session.userID};
  gate.resolve({assignment:{state:'migrated'}});
  assert.equal(await work,null);assert.equal(h.profiles,0);assert.equal(h.api.friendOwnerSelfProfile.value,null);
}
{
  const h=harness();await h.api.synchronizeFriendOwner('peer');
  h.self=async()=>({im_user_id:'pool',owner_revision:1,owner_state:'unavailable',is_assigned:false});
  await h.api.synchronizeFriendOwner('peer');
  assert.equal(h.api.friendOwnerSelfProfile.value.owner_state,'unavailable','same revision must still refresh owner validity');
}
console.log('PASS: ownership event coalescing, single POST on failure or pending, legacy compatibility, self/peer refresh, account/generation/SDK fences and owner validity.');
