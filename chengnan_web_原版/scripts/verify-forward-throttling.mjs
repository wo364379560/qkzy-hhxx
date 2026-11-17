import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require = createRequire(import.meta.url);
const compiled = ts.transpileModule(readFileSync(new URL(
  "../src/TUIKit/components/TUIChat/forward/forwardBatch.ts", import.meta.url,
), "utf8"), {
  compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.CommonJS },
}).outputText;
const compiledDispatch = ts.transpileModule(readFileSync(new URL(
  "../src/TUIKit/components/TUIChat/forward/forwardDispatch.ts", import.meta.url,
), "utf8"), {
  compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.CommonJS },
}).outputText;

// Exercise production delays/budgets without sending real messages or sleeping.
function harness() {
  let now = 1_000_000, nextTimer = 0;
  const timers = new Map(), api = {};
  const later = (callback, delay) => {
    const id = ++nextTimer;
    timers.set(id, { callback, at: now + Math.max(0, delay) });
    return id;
  };
  vm.runInNewContext(compiled, {
    exports: api, require, Promise, console, Date: { now: () => now },
    setTimeout: later, clearTimeout: (id) => timers.delete(id),
  });
  const dispatchApi = {};
  vm.runInNewContext(compiledDispatch, {
    exports: dispatchApi, require, Promise, console, Date: { now: () => now },
    setTimeout: later, clearTimeout: (id) => timers.delete(id),
  });
  return {
    ...api, ...dispatchApi, later, now: () => now,
    batch(targets = 1, messages = 1) {
      return api.createForwardBatch(
        Array.from({ length: targets }, (_, i) => ({ conversationID: `GROUP${i}` })),
        Array.from({ length: messages }, (_, i) => ({ ID: `source-${i}` })),
        false, (c) => c.conversationID,
      );
    },
    async finish(operation) {
      let done = false, error;
      operation.then(() => { done = true; }, (e) => { error = e; done = true; });
      for (let turn = 0; !done && turn < 10000; turn++) {
        // Let the SDK ACK/rejection and its observer settle before advancing time.
        for (let i = 0; i < 100; i++) await Promise.resolve();
        if (done) break;
        const next = [...timers].sort((a, b) => a[1].at - b[1].at)[0];
        assert.ok(next, "queue must make progress without a missing timer");
        timers.delete(next[0]);
        now = next[1].at;
        next[1].callback();
      }
      assert.ok(done, "bounded queue must eventually settle");
      if (error) throw error;
    },
  };
}
const defaults = { isActive: () => true, onChange: () => {}, resend: async () => assert.fail("unexpected resend") };

const fieldFailure = '[getGroupMemberProfile] getGroupMemberProfile fail websocket frequency block)';

// Exact field failure, not an invented SDK code. Numeric business/transport
// outcomes always win over frequency text nested in the same error.
{
  const h = harness();
  for (const error of [fieldFailure, new Error(fieldFailure), {message:fieldFailure}, {message:'Unknown failed. error: {"message":"websocket frequency block"}'}])
    assert.equal(h.isForwardRateLimited(error), true);
  for (const error of [
    {code:10017,message:fieldFailure}, {code:20009,message:'{"code":60018}'},
    {code:403,message:fieldFailure}, {code:2800,message:fieldFailure},
    {code:-1,message:fieldFailure}, {message:`{code: 10017, message: '${fieldFailure}'}`},
    {code:'ETIMEDOUT',message:fieldFailure}, {code:60008,message:'{"code":10023}'},
    {message:`timeout ${fieldFailure}`}, {forwardTimeout:true,message:fieldFailure},
    {message:'unknown network failure'},
  ]) assert.equal(h.isForwardRateLimited(error), false);
}

// A raw websocket rejection during dispatch is definite rejection, while a
// transport failure still remains unknown. Retrying uses the captured SDK
// object's identity instead of creating another message or random ID.
{
  const h=harness(), batch=h.batch(), objects=[];
  let created=0;
  const sdk={
    getLoginUser:()=> 'alice',
    getGroupMemberProfile:async()=>({data:{memberList:[{userID:'alice',role:'Member',muteUntil:0}]}}),
    createForwardMessage:options=>({...options,ID:`sdk${++created}`,conversationID:`GROUP${options.to}`}),
    sendMessage:async message=>{objects.push(message); throw 'sendMessage fail websocket frequency block)';},
    resendMessage:async message=>{objects.push(message); return {data:{message}};},
  };
  const sender=h.createCapturedForwardSender(sdk,()=>{},()=>true,'alice');
  await h.finish(h.runForwardBatch(batch,{...defaults,prepare:sender.prepare,send:sender.send,resend:sender.resend}));
  assert.equal(created,1);
  assert.equal(objects.length,2);
  assert.equal(objects[0],objects[1]);
  assert.equal(batch.targets[0].items[0].status,'success');
}

// A huge SDK fallback plan has no targets×messages allocation at submission or
// summary time. Source content is snapshotted once; only touched status records
// are allocated. This includes plans well beyond 2000/3000 total deliveries.
{
  const h = harness();
  const source = Array.from({length:1000}, (_, i) => ({ID:`source${i}`,payload:{text:`text${i}`}}));
  const large = h.createForwardBatch(Array.from({length:4000}, (_, i) => ({conversationID:`GROUP${i}`})), source, false, c=>c.conversationID);
  source[0].payload.text='changed original';
  assert.equal(large.targets[0].items[0].messages[0].payload.text, 'text0');
  assert.equal(large.targets[0].items[0].messages, large.targets[3999].items[0].messages);
  assert.equal(Object.keys(large.targets[0].items).length, 0, 'pending array must not expand four million slots');
  assert.equal(h.getForwardPendingWork(large).reduce((sum, w) => sum+w.items.length, 0), 4_000_000);
  large.targets[0].items[0].status='success';
  large.targets[0].items[1].status='unknown';
  assert.equal(h.summarizeForwardTarget(large.targets[0]).pending, 998);
  assert.equal(h.summarizeForwardTarget(large.targets[3999]).pending, 1000);
  assert.equal(h.canRetryForwardTarget(large.targets[0]), false, 'unknown blocks later ordered messages');
}

function capturedFixture(h, failure) {
  let reads=0, created=0;
  const delivered=[], attempts=new Map();
  const sdk={
    getLoginUser:()=> 'alice',
    getGroupMemberProfile:async ({groupID})=>{
      reads++; const count=(attempts.get(groupID)||0)+1; attempts.set(groupID,count);
      const error=failure(groupID,count);
      if (error) throw error;
      return {data:{memberList:[{userID:'alice',role:'Member',muteUntil:0}]}};
    },
    createForwardMessage:options=>({...options,ID:`sdk${++created}`,conversationID:`GROUP${options.to}`}),
    createMergerMessage:options=>({...options,ID:`sdk${++created}`,conversationID:`GROUP${options.to}`}),
    sendMessage:async message=>{delivered.push(message); return {data:{message}};},
    resendMessage:async message=>{delivered.push(message); return {data:{message}};},
  };
  const sender=h.createCapturedForwardSender(sdk,()=>{},()=>true,'alice');
  return {sender,delivered,reads:()=>reads,created:()=>created};
}

// Exercise the captured SDK sender + scheduler together for all fallback shapes,
// with the same bare production text across twenty target groups.
for (const kind of ['text','image','merged','legacy-image']) {
  const h=harness(), batch=h.batch(20);
  batch.merge=kind==='merged';
  for (const target of batch.targets) target.items[0].messages[0].type=kind==='text'?'TIMTextElem':'TIMImageElem';
  const f=capturedFixture(h,(_,count)=>count===1?fieldFailure:undefined);
  await h.finish(h.runForwardBatch(batch,{...defaults,prepare:f.sender.prepare,send:f.sender.send,resend:f.sender.resend}));
  assert.equal(f.reads(),40, `${kind}: one rejected preflight and one successful preflight per group`);
  assert.equal(f.created(),20, `${kind}: no SDK send created before permission succeeds`);
  assert.equal(f.delivered.length,20);
  assert.ok(batch.targets.every(t=>h.summarizeForwardTarget(t).success===1));
}

// Exhausted preflight retries remain safe for "retry unfinished"; successful
// destinations are skipped and the manual run handles the same bare text again.
{
  const h=harness(), batch=h.batch(20);
  let secondRun=false;
  const f=capturedFixture(h,(_,count)=>!secondRun||count===3?new Error(fieldFailure):undefined);
  const deps={...defaults,autoRetryCount:1,prepare:f.sender.prepare,send:f.sender.send,resend:f.sender.resend};
  await h.finish(h.runForwardBatch(batch,deps));
  assert.ok(batch.targets.every(t=>h.summarizeForwardTarget(t).failed===1));
  assert.equal(f.created(),0);
  secondRun=true;
  await h.finish(h.runForwardBatch(batch,deps));
  assert.equal(f.delivered.length,20);
  await h.finish(h.runForwardBatch(batch,deps));
  assert.equal(f.delivered.length,20, 'completed retry cannot re-send');
}

// Trusted settings independently schedule reads and actual sends; updates reject
// invalid values instead of disabling a rate gate or creating unbounded workers.
{
  const h=harness(), reads=[], sends=[];
  h.configureForwardPacingPolicy({prepareIntervalMs:250,sendIntervalMs:750,concurrency:3});
  await h.finish(h.runForwardBatch(h.batch(8),{...defaults,prepare:async()=>{reads.push(h.now());},send:async()=>{sends.push(h.now());}}));
  for(let i=1;i<reads.length;i++) assert.ok(reads[i]-reads[i-1]>=250);
  for(let i=1;i<sends.length;i++) assert.ok(sends[i]-sends[i-1]>=750);
  assert.ok(reads[1]-reads[0]<750, 'query budget must not be aliased to send budget');
  assert.throws(()=>h.configureForwardPacingPolicy({sendIntervalMs:0}), /配置无效/);
  assert.throws(()=>h.configureForwardPacingPolicy({concurrency:Infinity}), /配置无效/);
}

// Normal permission queries must be paced before Tencent rejects any of them.
// This also covers mixed/legacy/merged SDK batches sharing the same window.
{
  const h = harness(), first = h.batch(20), second = h.batch(10), reads = [], sends = [];
  const deps = {...defaults,
    prepare: async () => {reads.push(h.now()); await new Promise(resolve => h.later(resolve, 1500));},
    send: async () => {sends.push(h.now());},
  };
  await h.finish(Promise.all([h.runForwardBatch(first, deps), h.runForwardBatch(second, deps)]));
  assert.equal(reads.length, 30);
  assert.equal(sends.length, 30);
  for (let i=1; i<reads.length; i++) assert.ok(reads[i]-reads[i-1]>=1000, 'shared permission queries cannot burst');
  for (let i=1; i<sends.length; i++) assert.ok(sends[i]-sends[i-1]>=1000, 'slow queries cannot bunch actual sends');
}

{
  const h = harness(), batch = h.batch(8);
  let active=true, reads=0, sends=0;
  h.later(() => {active=false;}, 500);
  await h.finish(h.runForwardBatch(batch, {...defaults, isActive:()=>active,
    prepare:async()=>{reads++;}, send:async()=>{sends++;},
  }));
  assert.equal(reads,1);
  assert.equal(sends,1);
}

// Concurrent workers and separate batches must share the normal one-second cap.
{
  const h = harness(), first = h.batch(8, 2), second = h.batch(3, 1), starts = [];
  const deps = {
    ...defaults,
    send: async () => {
      starts.push(h.now());
      // A slow ACK leaves multiple sends in flight, but cannot multiply the rate.
      await new Promise(resolve => h.later(resolve, 1500));
    },
  };
  await h.finish(Promise.all([
    h.runForwardBatch(first, deps),
    h.runForwardBatch(second, deps),
  ]));
  assert.equal(starts.length, 19);
  for (let i = 1; i < starts.length; i++)
    assert.ok(starts[i] - starts[i - 1] >= 1000, "all destinations/batches share one send per second");
  assert.ok([first, second].every(b => b.targets.every(t => t.items.every(i => i.status === "success"))));
}

// More than one rejection recovers automatically, in order, with the same SDK identity.
{
  const h = harness(), batch = h.batch(55, 3), attempts = new Map(), deliveries = new Map();
  const starts = [], blocks = [], retriedObjects = [], waiting = [];
  let created = 0;
  const dispatch = async (message) => {
    const key = `${message.conversationID}/${message.source}`;
    const count = (attempts.get(key) || 0) + 1;
    attempts.set(key, count);
    starts.push(h.now());
    if (key === "GROUP0/source-1" && count <= 3) {
      retriedObjects.push(message);
      blocks.push({ at: h.now(), until: h.now() + 2000 * 2 ** (count - 1) });
      throw { message: 'Unknown failed. error: {"code":60018,"message":"websocket frequency block"}', data: { message } };
    }
    if (key === "GROUP0/source-1") retriedObjects.push(message);
    deliveries.set(message.conversationID, [...(deliveries.get(message.conversationID) || []), message.source]);
  };
  const deps = {
    ...defaults,
    send: async (target, item) => dispatch({ ID: `sdk-${++created}`, conversationID: target.conversation.conversationID, source: item.messages[0].ID }),
    resend: dispatch,
    onChange: () => waiting.push(batch.targets.some(t => t.items.some(i => i.status === "retrying"))),
  };
  await h.finish(h.runForwardBatch(batch, deps));
  assert.equal(deliveries.size, 55);
  for (const values of deliveries.values()) assert.deepEqual(values, ["source-0", "source-1", "source-2"]);
  assert.equal(created, 165);
  assert.equal(retriedObjects.length, 4);
  assert.ok(retriedObjects.every(message => message === retriedObjects[0]));
  assert.ok(waiting.includes(true), "temporary throttling has visible waiting progress");
  for (const block of blocks) assert.ok(!starts.some(at => at > block.at && at < block.until), "all workers respect the cooldown");
  const afterFirstBlock = starts.filter(at => at >= blocks[0].until && at < blocks[1].at);
  for (let i = 1; i < afterFirstBlock.length; i++) assert.ok(afterFirstBlock[i] - afterFirstBlock[i - 1] >= 2000, "pacing slows to two seconds after throttling");
  assert.ok(batch.targets.every(t => t.items.every(i => i.status === "success")));
  await h.finish(h.runForwardBatch(batch, deps));
  assert.equal(created, 165, "rerunning a completed batch never resends successful items");
}

// Persistent rejection has an actual bounded schedule, rather than an infinite spinner.
{
  const h = harness(), batch = h.batch(), times = [];
  await h.finish(h.runForwardBatch(batch, {
    ...defaults, send: async () => { times.push(h.now()); throw { code: 10023 }; },
  }));
  assert.deepEqual(times.map(t => t - times[0]), [0, 2000, 6000, 14000, 30000, 50000, 70000]);
  assert.equal(batch.targets[0].items[0].status, "failed");
  assert.equal(batch.running, false);
}

// A smaller overall retry budget stops even if the retry-count allowance remains.
{
  const h = harness(), batch = h.batch();
  let sends = 0;
  await h.finish(h.runForwardBatch(batch, {
    ...defaults, retryBudgetMs: 5000,
    send: async () => { sends++; throw { code: 60018 }; },
  }));
  assert.equal(sends, 2);
  assert.equal(batch.targets[0].items[0].status, "failed");
}

// Permission-read throttling is recoverable, but permission is still checked before sending.
{
  const h = harness(), batch = h.batch();
  let reads = 0, sends = 0;
  await h.finish(h.runForwardBatch(batch, {
    ...defaults,
    prepare: async () => { if (++reads <= 3) throw { code: 60018, notDispatched: true }; },
    send: async () => { assert.equal(reads, 4); sends++; },
  }));
  assert.equal(sends, 1);
  assert.equal(batch.targets[0].items[0].status, "success");
}

// Never use automatic retries for policy denials or ambiguous transport failures.
for (const error of [
  { code: 10017 }, { code: 20009 }, { code: 10010 }, { code: 80001 },
  { code: 2800 }, { code: 60008 },
  { code: 2800, message: 'previous failure {"code":60018}' },
]) {
  const h = harness(), batch = h.batch(1, 2);
  let sends = 0;
  await h.finish(h.runForwardBatch(batch, {
    ...defaults, send: async () => { sends++; throw error; },
  }));
  assert.equal(sends, 1, `must not retry ${error.code}`);
  assert.equal(batch.targets[0].items[1].status, "pending");
}

// Account change during a cooldown cancels all queued work promptly.
{
  const h = harness(), batch = h.batch(8, 2);
  let active = true, sends = 0;
  h.later(() => { active = false; }, 1000);
  await h.finish(h.runForwardBatch(batch, {
    ...defaults, isActive: () => active,
    send: async () => { sends++; throw { code: 60018 }; },
  }));
  assert.equal(sends, 1);
  assert.equal(batch.running, false);
  assert.ok(h.now() <= 1_001_400);
}
console.log("PASS: exact no-code websocket frequency block across 20 groups and text/image/merge/legacy SDK fallbacks; exhausted preflight manual retry; bare dispatch rejection preserves SDK identity; explicit policy/transport codes win; independent configurable read/send budgets; 4000 × 1000 sparse plan; 55 × 3 ordered deliveries; bounded backoff; cancellation; no duplicate successful/unknown sends.");
