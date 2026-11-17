import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import {randomUUID, webcrypto} from 'node:crypto';
import ts from 'typescript';
const source=fs.readFileSync(new URL('../src/TUIKit/components/TUIChat/forward/serverForward.ts',import.meta.url),'utf8');
const code=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2021,module:ts.ModuleKind.CommonJS}}).outputText;
const plain=x=>JSON.parse(JSON.stringify(x));
const text=(n=0)=>({ID:`m${n}`,type:'TIMTextElem',payload:{text:`hello${n}`}});
function harness(overrides={}) {
 const memory=new Map(),plans=new Map(),calls=[],jobs=new Map();
 const meta=j=>({request_id:j.id,total:j.targets*j.messages,target_count:j.targets,message_count:j.messages,uploaded_targets:j.ts.length,uploaded_messages:j.ms.length,state:j.state==='done'?'completed':j.state,running:false,success:j.state==='done'?j.targets*j.messages:0,failed:0,unknown:0,pending:j.state==='uploading'?j.targets*j.messages:0,retryable_count:0});
 const base={
  initForwardJob:async p=>{calls.push(['init',p]);let j=jobs.get(p.request_id);if(!j){j={id:p.request_id,targets:p.target_count,messages:p.message_count,hash:p.content_hash,ts:[],ms:[],state:'uploading'};jobs.set(j.id,j);}assert.equal(j.hash,p.content_hash);return meta(j);},
  putForwardTargets:async(id,p)=>{calls.push(['targets',p]);assert.ok(p.targets.length<=200);const j=jobs.get(id);if(p.offset<j.ts.length)assert.deepEqual(plain(j.ts.slice(p.offset,p.offset+p.targets.length)),plain(p.targets));else{assert.equal(p.offset,j.ts.length);j.ts.push(...p.targets);}},
  putForwardMessages:async(id,p)=>{calls.push(['messages',p]);assert.ok(p.messages.length<=100);const j=jobs.get(id);if(p.offset===j.ms.length)j.ms.push(...p.messages);else assert.deepEqual(plain(j.ms.slice(p.offset,p.offset+p.messages.length)),plain(p.messages));},
  commitForwardJob:async id=>{const j=jobs.get(id);assert.equal(j.ts.length,j.targets);assert.equal(j.ms.length,j.messages);j.state='done';calls.push(['commit',id]);return meta(j);},
  readForwardStatus:async id=>{calls.push(['status',id]);if(!jobs.has(id))throw{status:404};return meta(jobs.get(id));},
  readForwardJob:async(id,cursor='0')=>{calls.push(['page',cursor]);const j=jobs.get(id),total=j.targets*j.messages,start=Number(cursor),end=Math.min(total,start+200);return{...meta(j),page_size:200,next_cursor:end<total?String(end):'',items:Array.from({length:end-start},(_,k)=>{const i=start+k;return{conversation_id:j.ts[Math.floor(i/j.messages)].conversation_id,index:i%j.messages,preview:'hello',attempts:1,status:'success'};})};},
  retryForwardJob:async(id,retry)=>{calls.push(['retry',retry]);return meta(jobs.get(id));},
  cancelForwardJob:async id=>{const j=jobs.get(id);j.state='cancelled';return meta(j);},
 };
 const api={...base};for(const[k,v]of Object.entries(overrides))api[k]=(...args)=>v(base,...args);
 const store={saveForwardPlan:async(owner,id,p)=>plans.set(`${owner}:${id}`,plain(p)),loadForwardPlan:async(owner,id)=>plans.get(`${owner}:${id}`),deleteForwardPlan:async(owner,id)=>plans.delete(`${owner}:${id}`)};
 const ctx=vm.createContext({exports:{},require:n=>n==='../../../../api'?api:n==='@/config/runtime'?{CHAT_API_BASE_URL:'https://rui.fixture/api/v1/chat',SDK_APP_ID:7}:store,crypto:{randomUUID,subtle:webcrypto.subtle},TextEncoder,Uint8Array,URL,localStorage:new Proxy({setItem:(k,v)=>memory.set(k,v),getItem:k=>memory.get(k)||null,removeItem:k=>memory.delete(k)}, {ownKeys:()=>[...memory.keys()],getOwnPropertyDescriptor:()=>({enumerable:true,configurable:true})}),setTimeout:fn=>queueMicrotask(fn),console});
 vm.runInContext(code,ctx);
 return{...ctx.exports,api,base,plans,memory,jobs,calls,batch:(n,m=1)=>ctx.exports.createServerForwardBatch(Array.from({length:n},(_,i)=>({conversationID:`GROUPg${i}`})),Array.from({length:m},(_,i)=>text(i)),c=>c.conversationID)};
}
{
 const h=harness();assert.equal(h.supportsServerMessages([text()],false),true);assert.equal(h.supportsServerMessages([text()],true),false);
 assert.equal(h.supportsServerMessages([{type:'TIMImageElem',payload:{}}],false),false);
 assert.equal(h.supportsServerMessages([text(),{type:'TIMImageElem',payload:{}}],false),false);
 assert.equal(h.toForwardMessage({type:'TIMFileElem',ID:'f',payload:{fileUrl:'blob:local'}}),null);
 assert.equal(h.toForwardMessage({type:'TIMCustomElem',payload:{data:'card'}}).body[0].MsgContent.Data,'card');
}
// No total quota/product expansion; only one bounded page per refresh.
{
 const h=harness(),b=h.batch(4301,7);assert.equal(b.targets.length,0);assert.equal(b.serverPlan.targets.length,4301);
 await h.runServerForward(b,'alice',()=>true,()=>{});assert.equal(b.serverState.total,30107);assert.equal(b.serverState.success,30107);assert.equal(b.serverState.items.length,200);
 assert.equal(h.calls.filter(c=>c[0]==='targets').length,22);assert.equal(h.calls.filter(c=>c[0]==='page').length,1);assert.equal(h.plans.size,0);
 await h.loadServerProgressPage(b,()=>true,1);assert.equal(b.serverState.page_index,1);assert.equal(b.serverState.items[0].index,200%7);
 await h.loadServerProgressPage(b,()=>true,-1);assert.equal(b.serverState.page_index,0);
}
// Lost chunk ACK keeps identical offset and content.
{
 let lost=true;const h=harness({putForwardTargets:async(base,id,p)=>{await base.putForwardTargets(id,p);if(lost){lost=false;throw Error('lost ACK');}}}),b=h.batch(501);
 await h.runServerForward(b,'alice',()=>true,()=>{});assert.equal(h.jobs.size,1);assert.equal(h.jobs.get(b.serverRequestID).ts.length,501);
 assert.equal(h.calls.filter(c=>c[0]==='commit').length,1);const chunks=h.calls.filter(c=>c[0]==='targets');assert.deepEqual(chunks[0],chunks[1]);
}
// Closing/switching account mid-upload never commits; original browser resumes.
{
 let active=true;const h=harness({putForwardTargets:async(base,id,p)=>{await base.putForwardTargets(id,p);active=false;}}),b=h.batch(700);
 await h.runServerForward(b,'alice',()=>active,()=>{});assert.equal(h.calls.filter(c=>c[0]==='commit').length,0);assert.equal(h.plans.size,1);assert.equal(h.rememberedForwardJob('bob'),'');
 const restored={targets:[],merge:false,running:false,serverRequestID:h.rememberedForwardJob('alice')};h.api.putForwardTargets=h.base.putForwardTargets;active=true;
 await h.runServerForward(restored,'alice',()=>active,()=>{});assert.equal(restored.serverState.success,700);assert.equal(h.jobs.size,1);assert.equal(h.calls.filter(c=>c[0]==='targets')[1][1].offset,200);
}
// Missing V2 route retains local ID/sources and does not dispatch SDK messages.
{
 const h=harness({initForwardJob:async()=>{throw{status:404};}}),b=h.batch(20);await h.runServerForward(b,'alice',()=>true,()=>{});
 assert.match(b.serverNotice,/父任务接口/);assert.equal(h.plans.size,1);assert.equal(h.rememberedForwardJob('alice'),b.serverRequestID);
 h.api.initForwardJob=h.base.initForwardJob;const restored={targets:[],merge:false,running:false,serverRequestID:b.serverRequestID};await h.runServerForward(restored,'alice',()=>true,()=>{});assert.equal(restored.serverState.success,20);
}
// Explicit failure-only retry preserves its durable retry ID across lost ACK.
{
 let lost=true;const h=harness({retryForwardJob:async(base,id,retry)=>{const result=await base.retryForwardJob(id,retry);if(lost){lost=false;throw Error('lost ACK');}return result;}}),b=h.batch(5);
 await h.runServerForward(b,'alice',()=>true,()=>{});const commits=h.calls.filter(c=>c[0]==='commit').length;await h.retryServerForward(b,'alice',()=>true);
 const retries=h.calls.filter(c=>c[0]==='retry');assert.equal(retries.length,2);assert.equal(retries[0][1],retries[1][1]);assert.equal(h.calls.filter(c=>c[0]==='commit').length,commits);
}
// Large message ordinals accepted; bounded/malformed pages checked atomically.
{
 const h=harness(),b={serverRequestID:'r',serverState:{message_count:9000}},row={conversation_id:'GROUPg',index:7000,status:'success',attempts:1},page={request_id:'r',page_size:200,total:18000,running:true,items:[row]};
 h.applyServerProgress(b,page);assert.equal(b.serverState.items[0].index,7000);assert.throws(()=>h.applyServerProgress(b,{...page,items:[row,row]}),/无效/);
 assert.throws(()=>h.applyServerProgress(b,{...page,items:Array.from({length:201},(_,i)=>({...row,index:i}))}),/无效/);h.applyServerProgress(b,{...page,items:[]});assert.equal(b.serverState.items.length,0);
}
console.log('PASS: unbounded parent tasks; 4301×7 without product expansion; bounded chunks/pages; same-ID lost ACK recovery; partial refresh/account isolation; missing endpoint safe resume; idempotent retry; page validation.');
// After a browser refresh, an uncertain manual retry is reconciled with its
// existing retry identity before reading status, not turned into a new round.
{
 const h=harness(),b=h.batch(3);await h.runServerForward(b,'alice',()=>true,()=>{});
 const retry='pending-retry-00000001', key=`rui-forward:${encodeURIComponent("https://rui.fixture/api/v1/chat")}:7:alice:retry:${b.serverRequestID}`;h.memory.set(key,retry);
 const restored={targets:[],merge:false,running:false,serverRequestID:b.serverRequestID};await h.runServerForward(restored,'alice',()=>true,()=>{});
 assert.deepEqual(h.calls.filter(c=>c[0]==='retry'),[['retry',retry]]);assert.equal(h.memory.has(key),false);
}
// A transient page outage recovers without a second commit or parent.
{
 let failures=1;const h=harness({readForwardJob:async(base,id,cursor)=>{if(failures-- > 0)throw Error('offline');return base.readForwardJob(id,cursor);}}),b=h.batch(10);
 await h.runServerForward(b,'alice',()=>true,()=>{});assert.equal(b.serverState.success,10);assert.equal(h.calls.filter(c=>c[0]==='commit').length,1);
}
// A late polling page cannot replace a newer user-selected page.
{
 const h=harness(),b=h.batch(600);await h.runServerForward(b,'alice',()=>true,()=>{});
 let deliver;const original=h.api.readForwardJob;
 h.api.readForwardJob=(id,cursor)=>cursor==='0'?new Promise(resolve=>{deliver=()=>original(id,cursor).then(resolve);}):original(id,cursor);
 const stale=h.loadServerProgressPage(b,()=>true);
 await h.loadServerProgressPage(b,()=>true,1);assert.equal(b.serverState.page_index,1);
 await deliver();await stale;assert.equal(b.serverState.page_index,1);assert.equal(b.serverState.items[0].conversation_id,'GROUPg200');
}
console.log('PASS: pending retry reconciliation, transient progress recovery, and stale pagination suppression.');

{
 const h=harness(),b={serverRequestID:'legacy',serverState:{items:[],cursor:'0',page_index:0,cursors:['0']}};
 h.applyServerStatus(b,{request_id:'legacy',state:'completed',total:1,success:1,message_count:0,running:false});
 assert.equal(b.serverState.state,'done');
 h.applyServerProgress(b,{request_id:'legacy',page_size:200,total:1,running:false,items:[{conversation_id:'GROUPg',index:0,attempts:1,status:'success'}]});
 assert.equal(b.serverState.items.length,1);
}
