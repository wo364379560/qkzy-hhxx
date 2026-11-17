import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';

// Mount the production SFCs in Vue's renderer. No browser, network or messages
// are used: only the IM store and server responses are controlled by this test.
const require = createRequire(import.meta.url);
const vue = require('vue');
const ts = require('typescript');
const { parse, compileScript } = require('@vue/compiler-sfc');
const root = new URL('../', import.meta.url);
const read = path => readFileSync(new URL(path, root), 'utf8');
const element = type => ({ type, props: {}, children: [], parent: null });
const renderer = vue.createRenderer({
  createElement: element,
  createText: text => ({ ...element('#text'), text }),
  createComment: text => ({ ...element('#comment'), text }),
  setText: (node, text) => { node.text = text; },
  setElementText: (node, text) => { node.text = text; node.children = []; },
  parentNode: node => node.parent,
  nextSibling: node => node.parent?.children[node.parent.children.indexOf(node) + 1] || null,
  patchProp: (node, key, _previous, value) => { node.props[key] = value; },
  insert(node, parent, anchor = null) {
    if (node.parent) node.parent.children.splice(node.parent.children.indexOf(node), 1);
    node.parent = parent;
    const index = anchor ? parent.children.indexOf(anchor) : -1;
    if (index < 0) parent.children.push(node); else parent.children.splice(index, 0, node);
  },
  remove(node) { node.parent?.children.splice(node.parent.children.indexOf(node), 1); node.parent = null; },
});
const walk = node => [node, ...node.children.flatMap(walk)];
const settle = async () => { for (let i = 0; i < 5; i++) await vue.nextTick(); };
function clock() {
  let now = 0, id = 0;
  const pending = new Map();
  return {
    setTimeout(fn, ms) { pending.set(++id, { at: now + ms, fn }); return id; },
    clearTimeout(id) { pending.delete(id); },
    advance(ms) {
      now += ms;
      for (const [id, timer] of [...pending]) if (timer.at <= now) {
        pending.delete(id); timer.fn();
      }
    },
    get size() { return pending.size; },
  };
}
function evaluate(source, imports = require, timers = globalThis) {
  const code = ts.transpileModule(source, { compilerOptions: {
    module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020,
  } }).outputText;
  const module = { exports: {} };
  new Function('require', 'module', 'exports', 'setTimeout', 'clearTimeout', code)(
    imports, module, module.exports, timers.setTimeout.bind(timers), timers.clearTimeout.bind(timers),
  );
  return module.exports;
}
function component(path, imports, timers) {
  const { descriptor, errors } = parse(read(path), { filename: fileURLToPath(new URL(path, root)) });
  assert.deepEqual(errors, []);
  const script = compileScript(descriptor, { id: 'forward-panel-regression', inlineTemplate: true });
  return evaluate(script.content, name => name === 'vue' || name.includes('adapter-vue')
    ? vue : imports(name), timers).default;
}
const overlayPath = 'src/TUIKit/components/common/Overlay/index.vue';
const forwardPath = 'src/TUIKit/components/TUIChat/forward/index.vue';
const batch = evaluate(read('src/TUIKit/components/TUIChat/forward/forwardBatch.ts'));
const failures = [];
async function test(name, run) {
  try { await run(); console.log(`PASS: ${name}`); }
  catch (error) { failures.push(name); console.error(`FAIL: ${name}: ${error.message}`); }
}

await test('opening within 150ms of initial mount stays visible', async () => {
  const timers = clock(), visible = vue.ref(false), container = element('root');
  const Overlay = component(overlayPath, require, timers);
  const app = renderer.createApp({ setup: () => () => vue.h(Overlay, { visible: visible.value }) });
  app.mount(container);
  try {
    visible.value = true; await settle();
    timers.advance(200); await settle();
    assert(walk(container).some(n => n.props.class === 'overlay-container'), 'stale hide timer removed an open overlay');
  } finally { app.unmount(); }
});
await test('close releases pointer events immediately; quick reopen cancels old hide; unmount clears timers', async () => {
  const timers = clock(), visible = vue.ref(true), container = element('root');
  const Overlay = component(overlayPath, require, timers);
  const app = renderer.createApp({ setup: () => () => vue.h(Overlay, { visible: visible.value }) });
  app.mount(container);
  try {
    visible.value = false; await settle();
    assert.equal(walk(container).find(n => n.props.class === 'overlay-container')?.props.style.pointerEvents, 'none');
    visible.value = true; await settle();
    timers.advance(200); await settle();
    assert(walk(container).some(n => n.props.class === 'overlay-container'));
    visible.value = false; await settle();
  } finally { app.unmount(); }
  assert.equal(timers.size, 0);
});

function harness({ remembered = '', hold = false } = {}) {
  const timers = clock(), data = new Map(), watchers = new Map(), pending = [];
  const events = [], forgotten = [], errors = [];
  let submissions = 0;
  data.set('USER:userProfile', { userID: 'alice' });
  const model = { ID: 'message-1', type: 'TIMTextElem', status: 'success', time: 1, payload: { text: 'fixture only' } };
  data.set('CHAT:messageList', [model]);
  const store = {
    getData: (namespace, key) => data.get(`${namespace}:${key}`),
    getMessageModel: id => id === model.ID ? model : undefined,
    getConversationModel: () => undefined,
    watch(namespace, handlers) {
      for (const [key, handler] of Object.entries(handlers)) {
        const id = `${namespace}:${key}`, set = watchers.get(id) || new Set();
        watchers.set(id, set); set.add(handler); handler(data.get(id));
      }
    },
    unwatch(namespace, handlers) {
      for (const [key, handler] of Object.entries(handlers)) watchers.get(`${namespace}:${key}`)?.delete(handler);
    },
    update(namespace, key, value) {
      const id = `${namespace}:${key}`;
      if (data.get(id) === value) return;
      data.set(id, value);
      for (const handler of watchers.get(id) || []) handler(value);
    },
  };
  const server = {
    rememberedForwardJob: () => remembered,
    forgetForwardJob: owner => { forgotten.push(owner); remembered = ''; },
    supportsServerMessages: () => true,
    createServerForwardBatch() {
      submissions++;
      remembered = 'new-job';
      return { targets: [], serverRequestID: remembered, serverAccepted: true };
    },
    runServerForward(task, _owner, active, change) {
      const finish = () => {
        if (active()) {
          task.serverState = { state: 'done', success: 1, total: 1, items: [] };
          change();
        }
      };
      if (!hold) { finish(); return Promise.resolve(); }
      return new Promise(resolve => pending.push(() => { finish(); resolve(); }));
    },
  };
  const Overlay = component(overlayPath, require, timers);
  const Transfer = { emits: ['cancel', 'submit'], setup(_props, { emit }) {
    return () => vue.h('div', { class: 'fixture-transfer' }, [
      vue.h('button', { onClick: () => emit('cancel') }, '取消'),
      vue.h('button', { onClick: () => emit('submit', [{ conversationID: 'GROUPfixture', conversationType: 'GROUP' }]) }, '提交'),
    ]);
  } };
  const Forward = component(forwardPath, name => {
    if (name === 'lodash') return require(name);
    if (name.endsWith('/serverForward')) return server;
    if (name.endsWith('/forwardBatch')) return batch;
    if (name.endsWith('/forwardTargets')) return { resolveForwardTargets: async (items, resolve) => Promise.all(items.map(resolve)) };
    if (name.endsWith('/forwardDispatch')) return {};
    if (name.endsWith('/api')) return {};
    if (name.endsWith('/Overlay/index.vue')) return { default: Overlay };
    if (name.endsWith('/Transfer/index.vue')) return { default: Transfer };
    if (name.endsWith('/Toast')) return { Toast: info => errors.push(info), TOAST_TYPE: {} };
    if (name.endsWith('/utils/env')) return { isUniFrameWork: false };
    if (name.endsWith('/utils/utils')) return {};
    if (name.includes('offlinePushInfoManager')) return {};
    if (name.includes('chat-uikit-engine')) return {
      default: { TYPES: { CONV_C2C: 'C2C', CONV_GROUP: 'GROUP' } }, TUIStore: store,
      StoreName: Object.fromEntries(['USER', 'CUSTOM', 'CONV', 'FRIEND', 'GRP', 'CHAT'].map(x => [x, x])),
      TUITranslateService: { t: value => value },
    };
    throw Error(`Unexpected import ${name}`);
  }, timers);
  const container = element('root');
  const app = renderer.createApp(Forward, { onToggleMultipleSelectMode: value => events.push(value) });
  app.config.errorHandler = error => { errors.push(error); };
  app.mount(container);
  return {
    app, timers, container, errors, events, forgotten,
    get submissions() { return submissions; },
    command(value = model.ID) { store.update('CUSTOM', 'singleForwardMessageID', value); },
    multi() { store.update('CUSTOM', 'multipleForwardMessageID', { messageIDList: [model.ID], isMergeForward: false }); },
    clear() { store.update('CUSTOM', 'singleForwardMessageID', undefined); store.update('CUSTOM', 'multipleForwardMessageID', undefined); },
    complete() { for (const finish of pending.splice(0)) finish(); },
    click(text) {
      const button = walk(container).find(n => n.type === 'button' && (n.text || '').includes(text));
      assert(button, `button '${text}' missing`); button.props.onClick({});
    },
    has(className) { return walk(container).some(n => n.props.class === className); },
  };
}
await test('refresh restores pending job without automatically opening or blocking chat', async () => {
  const h = harness({ remembered: 'existing-job', hold: true });
  try {
    await settle();
    assert.equal(h.has('forward-progress'), false);
    assert.equal(h.has('overlay-container'), false);
    assert(h.has('forward-background-status'));
    assert.equal(h.submissions, 0);
  } finally { h.app.unmount(); h.complete(); }
});
await test('repeated same-message forward commands can reopen pending results after hide', async () => {
  const h = harness({ remembered: 'existing-job', hold: true });
  try {
    await settle(); h.timers.advance(200); await settle();
    for (let i = 0; i < 3; i++) {
      h.command(); await settle();
      assert(h.has('forward-progress'), 'same message command did not reopen results');
      h.click('后台转发'); await settle(); h.timers.advance(200); await settle();
      assert.equal(h.has('overlay-container'), false);
    }
    assert.equal(h.submissions, 0);
  } finally { h.app.unmount(); h.complete(); }
});
await test('completed result closes, forgets saved job and exits multi-select; next forwarding still works', async () => {
  const h = harness({ remembered: 'existing-job' });
  try {
    await settle(); h.click('查看转发结果'); await settle();
    h.click('结束查看'); await settle(); h.timers.advance(200); await settle();
    assert.deepEqual(h.forgotten, ['alice']);
    assert.deepEqual(h.events, [false]);
    assert.equal(h.has('overlay-container'), false);
    for (const open of [() => h.command(), () => h.multi(), () => h.command()]) {
      open(); await settle(); assert(h.has('fixture-transfer'));
      h.click('取消'); await settle(); h.timers.advance(200); await settle();
      assert.equal(h.has('overlay-container'), false);
    }
    assert.deepEqual(h.errors, []);
  } finally { h.app.unmount(); }
});
await test('first submit, hide, finish in background, then forward again without an invisible modal', async () => {
  const h = harness({ hold: true });
  try {
    h.command(); await settle(); h.click('提交'); await settle();
    assert.equal(h.submissions, 1);
    h.click('后台转发'); await settle(); h.timers.advance(200); await settle();
    h.complete(); await settle(); h.timers.advance(200); await settle();
    assert.equal(h.has('overlay-container'), false, 'completion reopened a hidden overlay');
    assert(h.has('forward-background-status'));
    h.command(); await settle(); h.timers.advance(200); await settle();
    assert(h.has('fixture-transfer'), 'next forward picker failed to open');
    assert.deepEqual(h.errors, []);
  } finally { h.app.unmount(); h.complete(); }
});
assert.deepEqual(failures, [], 'Forward panel lifecycle regressions');
