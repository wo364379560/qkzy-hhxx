import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';
import { callLifecycleMethods } from './fixtures/installed-call-sources.mjs';
const source=readFileSync(new URL('../src/TUIKit/components/TUIGroup/manage-group/index.vue',import.meta.url),'utf8').match(/<script[^>]*>([\s\S]*?)<\/script>/)[1];
const tree=ts.createSourceFile('group.ts',source,ts.ScriptTarget.Latest,true);
const extract=name=>tree.statements.find(n=>n.getText(tree).startsWith(name)).getText(tree);
const compile=s=>ts.transpileModule(s,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS}}).outputText;
function loadCallRuntime(ctx) {
 const native=ctx.require;
 ctx.require=name=>name==='@/features/auth/recovery'?(ctx.recovery || {withTimeout:p=>p}):name==='@trtc/call-engine-lite-js'?(ctx.wasmModule || {TUICallEngine:{once:(_event,ready)=>ctx.wasmGate?ctx.wasmGate.promise.then(ready):ready()}}):name==='vue'?{reactive:x=>x,nextTick:async()=>{}}:name==='@/features/call/callSignalInbox'?{createCallSignalInbox:chat=>({chat,dispose(){},suspend(){},activate(){}})}:native(name);
 vm.runInNewContext(compile(readFileSync(new URL('../src/bootstrap/callRuntime.ts',import.meta.url),'utf8')),ctx);
}
const deferred=()=>{let resolve,reject;const promise=new Promise((a,b)=>{resolve=a;reject=b});return{promise,resolve,reject}};
function fixture(){
 const gate=deferred(); const ctx={exports:{},groupContextRevision:1,groupDisposed:false,currentGroupID:{value:'group-a'},
 groupMemberList:{value:[]},member:{value:{admin:[],member:[]}},mergedAdminUserIDSet:{value:new Set()},adminLoadError:{value:''},
 getGroupAdmins:()=>gate.promise,sortGroupMembers:x=>x,toGroupMemberFromAdmin:a=>({userID:a.user_id}),
 enrichGroupMembersWithFriendRemark:x=>x,rebuildMemberBuckets(){},syncGroupMemberOnlineStatus(){},mask:{value:false}};
 vm.runInNewContext(compile(extract('const mergeBackendGroupAdmins')+'\n'+extract('const applyCurrentGroupMemberList')+'\nexports.merge=mergeBackendGroupAdmins;exports.apply=applyCurrentGroupMemberList;'),ctx);
 return{ctx,gate};
}
test('A-B-A navigation and unmount reject old administrator results',async()=>{
 for(const unmount of [false,true]){
  const{ctx,gate}=fixture();const pending=ctx.exports.merge('group-a');
  if(unmount)ctx.groupDisposed=true;else ctx.groupContextRevision+=2;
  gate.resolve({admins:[{user_id:'stale'}]});await pending;
  assert.equal(ctx.member.value.admin.length,0);assert.equal(ctx.mergedAdminUserIDSet.value.size,0);
 }
});
test('member display does not wait for failed/hung administrator request',async()=>{
 const{ctx,gate}=fixture();await ctx.exports.apply([{userID:'member-a'}]);
 assert.equal(ctx.groupMemberList.value[0].userID,'member-a');
 gate.reject(Error('unavailable'));await new Promise(r=>setImmediate(r));
 assert.match(ctx.adminLoadError.value,/管理员资料/);
 assert.equal(ctx.groupMemberList.value.length,1);
});
test('diagnostics contain only phases, origin, duration, local ID and numeric codes',()=>{
 const logs=[];const ctx={exports:{},console:{info:(...args)=>logs.push(args)},URL,Date,Math,globalThis:{}};
 vm.runInNewContext(compile(readFileSync(new URL('../src/utils/networkDiagnostics.ts',import.meta.url),'utf8')),ctx);
 ctx.exports.tracePhase('response',Date.now(),{requestId:'safe-id',origin:'https://api.prfgsu.com/private/phone?token=secret',status:503,code:'secret-token',password:'password'});
 const json=JSON.stringify(logs);assert(!json.includes('secret'));assert(!json.includes('phone'));assert(!json.includes('password'));assert(json.includes('https://api.prfgsu.com'));
});
test('late call-kit load initializes existing chat once and ignores an obsolete login',async()=>{
 const chat={};let inits=0,current=true;
 const gate=deferred();
 const ctx={exports:{},require:name=>name==='@tencentcloud/tui-core-lite'?{default:{getService:()=>null},TUIConstants:{TUICalling:{SERVICE:{NAME:'call'}},TUILogin:{EVENT:{LOGIN_STATE_CHANGED:'login'},EVENT_SUB_KEY:{USER_LOGIN_SUCCESS:'in',USER_LOGOUT_SUCCESS:'out'}}}}:name==='@tencentcloud/chat-uikit-engine-lite'?{default:{chat}}:{TUICallKitAPI:{handleExceptionExit:async()=>{},destroyed:async()=>{},init:async params=>{inits++;assert.equal(params.tim,chat);await gate.promise;}}}};
 loadCallRuntime(ctx);
 const session={sdkAppID:1,userID:'fixture',userSig:'fixture'};
 const a=ctx.exports.startCallIntegration(session,()=>current),b=ctx.exports.startCallIntegration(session,()=>current);
 assert.equal(a,b);await new Promise(r=>setImmediate(r));assert.equal(inits,1);
 gate.resolve();await a;
 await ctx.exports.resetCallIntegration();current=false;
 await ctx.exports.startCallIntegration(session,()=>current);assert.equal(inits,1);
});
test('late CallKit preserves its chat notifications without broadcasting login again',async()=>{
 const seen=[];const service={onNotifyEvent:async(event,sub)=>seen.push([event,sub])};
 const ctx={exports:{},require:name=>name==='@tencentcloud/tui-core-lite'?{
  default:{getService:()=>service,unregisterEvent(){},notifyEvent:()=>assert.fail('must not broadcast a second login')},
  TUIConstants:{TUICalling:{SERVICE:{NAME:'call'}},TUILogin:{EVENT:{LOGIN_STATE_CHANGED:'login'},EVENT_SUB_KEY:{USER_LOGIN_SUCCESS:'in',USER_LOGOUT_SUCCESS:'out'}}}
 }:name==='@tencentcloud/chat-uikit-engine-lite'?{default:{chat:{}}}:{TUICallKitAPI:{handleExceptionExit:async()=>{},init:async()=>{},destroyed:async()=>{}}}};
 loadCallRuntime(ctx);
 await ctx.exports.startCallIntegration({sdkAppID:1,userID:'a',userSig:'s'},()=>true);
 ctx.exports.resetCallIntegration();await new Promise(r=>setImmediate(r));
 assert.deepEqual(seen,[['login','out'],['login','in'],['login','out']]);
});

test('call initialization failures are retryable and stale init is cleaned before the next login',async()=>{
 const gate=deferred(), events=[];let current=true,fail=true;
 const ctx={exports:{},console,require:name=>name==='@tencentcloud/tui-core-lite'?{default:{getService:()=>null},TUIConstants:{TUICalling:{SERVICE:{NAME:'call'}},TUILogin:{EVENT:{LOGIN_STATE_CHANGED:'login'},EVENT_SUB_KEY:{USER_LOGIN_SUCCESS:'in',USER_LOGOUT_SUCCESS:'out'}}}}:name==='@tencentcloud/chat-uikit-engine-lite'?{default:{chat:{}}}:{TUICallKitAPI:{handleExceptionExit:async()=>{},init:async p=>{events.push('init:'+p.userID);if(fail){fail=false;throw Error('offline')}if(p.userID==='old')await gate.promise;},destroyed:async()=>events.push('destroy')}}};
 loadCallRuntime(ctx);
 const old={sdkAppID:1,userID:'old',userSig:'s'};
 await assert.rejects(ctx.exports.startCallIntegration(old,()=>current));
 const pending=ctx.exports.startCallIntegration(old,()=>current);await new Promise(r=>setImmediate(r));
 current=false;ctx.exports.resetCallIntegration();
 const next=ctx.exports.startCallIntegration({...old,userID:'new'},()=>true);
 await new Promise(r=>setImmediate(r));assert(!events.includes('init:new'));
 gate.resolve();await Promise.all([pending,next]);assert.equal(events.at(-1),'init:new');
 assert(events.slice(events.lastIndexOf('init:old')+1,-1).includes('destroy'));
});

function lifecycleFixture() {
 const events=[],watchers=new Set(); let status='idle',role='callee';
 const constants={StoreName:{CALL:'call'},NAME:{CALL_STATUS:'status',CALL_ROLE:'role',PREFIX:'fixture'},CallStatus:{IDLE:'idle',CALLING:'calling',CONNECTED:'connected'},CallRole:{CALLER:'caller'}};
 const TUIStore={getData:(_,key)=>key==='status'?status:role,watch:(_,o)=>watchers.add(o.status),unwatch:(_,o)=>watchers.delete(o.status)};
 const update=value=>{status=value;for(const cb of [...watchers])cb(value);};
 const ctx={exports:{},console:{error(){}},...constants,TUIStore};
 vm.runInNewContext(compile(`class Lifecycle { ${callLifecycleMethods} };exports.Lifecycle=Lifecycle;`),ctx);
 const api=new ctx.exports.Lifecycle();
 Object.assign(api,{_removeListenTuiCallEngineEvent(){},_resetCallStore:()=>update('idle'),
  hangup:async()=>{events.push('hangup');update('idle')},reject:async()=>{events.push('reject');update('idle')},
  init:async p=>{events.push('init:'+p.userSig);api._tuiCallEngine={destroyInstance:async()=>events.push('destroy')};}});
 const fixture={TUICallKitAPI:api,TUIStore,...constants};
 const runtime={exports:{},console,require:name=>name==='@tencentcloud/tui-core-lite'?{default:{getService:()=>null},TUIConstants:{TUICalling:{SERVICE:{NAME:'call'}},TUILogin:{EVENT:{LOGIN_STATE_CHANGED:'login'},EVENT_SUB_KEY:{USER_LOGIN_SUCCESS:'in',USER_LOGOUT_SUCCESS:'out'}}}}:name==='@tencentcloud/chat-uikit-engine-lite'?{default:{chat:{}}}:fixture};
 loadCallRuntime(runtime);
 return{api:runtime.exports,events,watchers,setStatus:update,setRole:r=>role=r};
}
test('actual vendor lifecycle rejects ringing and hangs up connected calls before confirming destroy',async()=>{
 for(const [status,role,expected] of [['calling','callee','reject'],['calling','caller','hangup'],['connected','callee','hangup']]){
  const f=lifecycleFixture();await f.api.startCallIntegration({sdkAppID:1,userID:'a',userSig:'s'},()=>true);
  f.setRole(role);f.setStatus(status);f.api.resetCallIntegration();await new Promise(r=>setImmediate(r));
  assert.deepEqual(f.events,['init:s',expected,'destroy']);assert.equal(f.watchers.size,0);
 }
});
test('signature rotation keeps live calls, coalesces latest credentials and cannot block logout',async()=>{
 const f=lifecycleFixture(),session={sdkAppID:1,userID:'a',userSig:'one'};
 await f.api.startCallIntegration(session,()=>true);f.setStatus('connected');
 const old=f.api.startCallIntegration({...session,userSig:'two'},()=>true);await new Promise(r=>setImmediate(r));
 assert.equal(f.api.callIntegrationState.status,'ready');assert.deepEqual(f.events,['init:one']);assert.equal(f.watchers.size,1);
 const latest=f.api.startCallIntegration({...session,userSig:'three'},()=>true);await new Promise(r=>setImmediate(r));
 f.setStatus('idle');await Promise.all([old,latest]);assert.deepEqual(f.events,['init:one','destroy','init:three']);assert.equal(f.watchers.size,0);
 f.setStatus('calling');const waiting=f.api.startCallIntegration({...session,userSig:'four'},()=>true);await new Promise(r=>setImmediate(r));
 f.api.resetCallIntegration();await waiting;await new Promise(r=>setImmediate(r));
 assert.equal(f.watchers.size,0);assert.deepEqual(f.events.slice(-2),['reject','destroy']);assert(!f.events.includes('init:four'));
});
test('same account/signature in a new IM scope binds the new inbox',async()=>{
 const f=lifecycleFixture(),session={sdkAppID:1,userID:'a',userSig:'s'},chat={};
 f.api.captureCallSignals(session,chat,()=>true,1);await f.api.startCallIntegration(session,()=>true);
 f.api.captureCallSignals(session,chat,()=>true,2);await f.api.startCallIntegration(session,()=>true);
 assert.deepEqual(f.events,['init:s','destroy','init:s']);
});

test('independent WASM readiness delays only call construction and ignores a previous login',async()=>{
 const gate=deferred(),events=[];
 const ctx={exports:{},console,wasmGate:gate,require:name=>name==='@tencentcloud/tui-core-lite'?{default:{getService:()=>null},TUIConstants:{TUICalling:{SERVICE:{NAME:'call'}},TUILogin:{EVENT:{LOGIN_STATE_CHANGED:'login'},EVENT_SUB_KEY:{USER_LOGIN_SUCCESS:'in',USER_LOGOUT_SUCCESS:'out'}}}}:name==='@tencentcloud/chat-uikit-engine-lite'?{default:{chat:{}}}:{TUICallKitAPI:{handleExceptionExit:async()=>{},init:async p=>events.push(p.userID),destroyed:async()=>{}}}};
 loadCallRuntime(ctx);
 const old=ctx.exports.startCallIntegration({sdkAppID:1,userID:'old',userSig:'s'},()=>true);await new Promise(r=>setImmediate(r));
 assert.deepEqual(events,[]);assert.equal(ctx.exports.callIntegrationState.status,'loading');
 ctx.exports.resetCallIntegration();
 const next=ctx.exports.startCallIntegration({sdkAppID:1,userID:'new',userSig:'s'},()=>true);gate.resolve();await Promise.all([old,next]);
 assert.deepEqual(events,['new']);assert.equal(ctx.exports.callIntegrationState.status,'ready');
});

function wasmTimeoutFixture({ throwFirstSubscription = false } = {}) {
 const timers=new Map(),events=[],listeners=[];let nextTimer=0,now=0,subscriptions=0;
 const recovery={exports:{},setTimeout:(fn,ms)=>{const id=++nextTimer;timers.set(id,{at:now+ms,fn});return id;},clearTimeout:id=>timers.delete(id)};
 // This is the actual timeout/error implementation used by production, with
 // only its clock under test control (no 15-second real sleeps).
 vm.runInNewContext(compile(readFileSync(new URL('../src/features/auth/recovery.ts',import.meta.url),'utf8')),recovery);
 const wasmModule={TUICallEngine:{once:(_event,fn)=>{subscriptions++;if(throwFirstSubscription&&subscriptions===1)throw Error('ready registration failed');listeners.push(fn);}}};
 const module={TUICallKitAPI:{handleExceptionExit:async()=>events.push('exit'),destroyed:async()=>events.push('destroy'),init:async p=>events.push('init:'+p.userID)}};
 const ctx={exports:{},console,recovery:recovery.exports,wasmModule,require:name=>name==='@tencentcloud/tui-core-lite'?{default:{getService:()=>null},TUIConstants:{TUICalling:{SERVICE:{NAME:'call'}},TUILogin:{EVENT:{LOGIN_STATE_CHANGED:'login'},EVENT_SUB_KEY:{USER_LOGIN_SUCCESS:'in',USER_LOGOUT_SUCCESS:'out'}}}}:name==='@tencentcloud/chat-uikit-engine-lite'?{default:{chat:{}}}:module};
 loadCallRuntime(ctx);
 return {api:ctx.exports,events,timers,subscriptions:()=>subscriptions,
  ready(){for(const fn of listeners.splice(0))fn();},
  advance(ms){now+=ms;for(const [id,timer] of [...timers])if(timer.at<=now){timers.delete(id);timer.fn();}}};
}
const turn=()=>new Promise(resolve=>setImmediate(resolve));
test('actual 15s WASM timeout is retryable, shares one ready listener and never initializes an old account',async()=>{
 const f=wasmTimeoutFixture(),oldSession={sdkAppID:1,userID:'old',userSig:'s'},newSession={...oldSession,userID:'new'};
 const old=f.api.startCallIntegration(oldSession,()=>true);const oldRejected=assert.rejects(old,e=>e.code==='ETIMEDOUT');await turn();
 f.api.resetCallIntegration();const next=f.api.startCallIntegration(newSession,()=>true);const nextRejected=assert.rejects(next,e=>e.code==='ETIMEDOUT');await turn();
 assert.equal(f.subscriptions(),1);f.advance(14999);await turn();assert.equal(f.api.callIntegrationState.status,'loading');
 f.advance(1);await Promise.all([oldRejected,nextRejected]);assert.equal(f.api.callIntegrationState.status,'error');assert(!f.events.some(x=>x.startsWith('init:')));
 const retry=f.api.retryCallIntegration();await turn();assert.equal(f.subscriptions(),1);f.ready();await retry;
 assert.deepEqual(f.events.filter(x=>x.startsWith('init:')),['init:new']);assert(f.events.indexOf('destroy')<f.events.indexOf('init:new'));
 assert.equal(f.api.callIntegrationState.status,'ready');assert.equal(f.timers.size,0);
});
test('late WASM ready after logout cannot revive calls and clears the real timeout timer',async()=>{
 const f=wasmTimeoutFixture();const pending=f.api.startCallIntegration({sdkAppID:1,userID:'old',userSig:'s'},()=>true);await turn();
 f.api.resetCallIntegration();await turn();f.ready();await pending;
 assert.equal(f.api.callIntegrationState.status,'idle');assert(!f.events.some(x=>x.startsWith('init:')));assert.equal(f.timers.size,0);
});
test('failed ready subscription discards its rejected gate and can retry without stale timers',async()=>{
 const f=wasmTimeoutFixture({throwFirstSubscription:true});
 await assert.rejects(f.api.startCallIntegration({sdkAppID:1,userID:'new',userSig:'s'},()=>true),/ready registration failed/);
 assert.equal(f.api.callIntegrationState.status,'error');assert.equal(f.timers.size,0);
 const retry=f.api.retryCallIntegration();await turn();assert.equal(f.subscriptions(),2);f.ready();await retry;
 assert.deepEqual(f.events.filter(x=>x.startsWith('init:')),['init:new']);assert.equal(f.api.callIntegrationState.status,'ready');assert.equal(f.timers.size,0);
});
