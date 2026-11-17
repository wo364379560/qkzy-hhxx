import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import ts from 'typescript';
import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);
function load(name) {
  const source = readFileSync(new URL('../src/TUIKit/components/TUIChat/forward/' + name + '.ts', import.meta.url), 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 } }).outputText;
  const module = { exports: {} };
  new Function('require', 'module', 'exports', code)(require, module, module.exports);
  return module.exports;
}
const { resolveForwardTargets } = load('forwardTargets');
const { createForwardBatch, runForwardBatch, configureForwardPacingPolicy, summarizeForwardTarget } = load('forwardBatch');
configureForwardPacingPolicy({sendIntervalMs: 1, prepareIntervalMs: 1, concurrency: 2});
const targets = await resolveForwardTargets(['a', 'b', 'c'].map(id => ({conversationID:'C2C' + id})), target => {
  if (target.conversationID === 'C2Cb') throw Error('fixture resolution failure');
  return target;
});
assert.equal(targets.length, 3);
const batch = createForwardBatch(targets, [{ID:'message-1', type:'TIMTextElem',payload:{text:'fixture'}}], false, c => c.conversationID);
const sent = [];
await runForwardBatch(batch, {isActive:()=>true, onChange:()=>{}, prepare:target=>{
  if (target.conversation.resolutionError) throw Object.assign(Error('resolution failed'), {notDispatched:true});
},send:async target=>{sent.push(target.conversation.conversationID);return{data:{message:{status:'success'}}};}});
assert.deepEqual(sent.sort(), ['C2Ca','C2Cc']);
assert.equal(summarizeForwardTarget(batch.targets[1]).failed, 1);
assert.equal(batch.targets.length, 3);
console.log('F01: resolution failure retained; earlier/later targets completed independently');
