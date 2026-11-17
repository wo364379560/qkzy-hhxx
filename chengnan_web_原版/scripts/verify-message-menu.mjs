import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import vm from 'node:vm';

// No real browser or sends: actual menu SFC, actual list menu template/handlers,
// actual universal-api outsideClick, and a deliberately delayed async component.
const require = createRequire(import.meta.url);
const vue = require('vue');
const ts = require('typescript');
const { parse, compileScript } = require('@vue/compiler-sfc');
const root = new URL('../', import.meta.url);
const read = path => readFileSync(new URL(path, root), 'utf8');
function element(type) {
  return vue.markRaw({ type, props: {}, children: [], parent: null,
    get id() { return this.props.id; },
    contains(target) { for (let n = target; n; n = n.parent) if (n === this) return true; return false; },
  });
}
const renderer = vue.createRenderer({
  createElement: element, createText: text => ({ ...element('#text'), text }),
  createComment: text => ({ ...element('#comment'), text }),
  setText: (node, text) => { node.text = text; },
  setElementText: (node, text) => { node.text = text; node.children = []; },
  parentNode: node => node.parent,
  nextSibling: node => node.parent?.children[node.parent.children.indexOf(node) + 1] || null,
  patchProp: (node, key, _old, value) => { node.props[key] = value; },
  insert(node, parent, anchor = null) {
    if (node.parent) node.parent.children.splice(node.parent.children.indexOf(node), 1);
    node.parent = parent;
    const index = anchor ? parent.children.indexOf(anchor) : -1;
    if (index < 0) parent.children.push(node); else parent.children.splice(index, 0, node);
  },
  remove(node) { node.parent?.children.splice(node.parent.children.indexOf(node), 1); node.parent = null; },
});
const walk = node => [node, ...node.children.flatMap(walk)];
const label = node => (node.text || '') + node.children.map(label).join('');
const settle = async () => { for (let i = 0; i < 10; i++) await vue.nextTick(); };
const compile = source => ts.transpileModule(source, { compilerOptions: {
  module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020,
} }).outputText;
function sfc(source, imports) {
  const { descriptor, errors } = parse(source);
  assert.deepEqual(errors, []);
  const code = compileScript(descriptor, { id: 'menu-regression', inlineTemplate: true }).content;
  const module = { exports: {} };
  new Function('require', 'module', 'exports', compile(code))(name =>
    name === 'vue' || name.includes('adapter-vue') ? vue : imports(name), module, module.exports);
  return module.exports.default;
}
const list = read('src/TUIKit/components/TUIChat/message-list/index.vue');
const listScript = parse(list).descriptor.scriptSetup.content;
const ast = ts.createSourceFile('message-list.ts', listScript, ts.ScriptTarget.Latest, true);
function productionFunction(name, optional = false) {
  const node = ast.statements.find(n => ts.isFunctionDeclaration(n) && n.name?.text === name
    || ts.isVariableStatement(n) && n.declarationList.declarations.some(d => d.name.getText(ast) === name));
  assert(node || optional, `production handler ${name} missing`);
  return node?.getText(ast) || '';
}
const menuTemplate = list.match(/<MessageTool\s[\s\S]*?<\/MessageTool>/)?.[0];
assert(menuTemplate);

function harness(type) {
  const documentListeners = [];
  const sandbox = { console: { log() {} }, navigator: { maxTouchPoints: 0 },
    window: { navigator: { userAgent: 'Windows PC' } },
    document: { addEventListener: (event, handler) => documentListeners.push({ event, handler }) },
    exports: {},
  };
  vm.runInNewContext(compile(read('node_modules/@tencentcloud/universal-api/index.js')), sandbox);
  const outsideClick = sandbox.exports.outsideClick;
  const registrations = [], originalListen = outsideClick.listen.bind(outsideClick);
  outsideClick.listen = options => { registrations.push(options); originalListen(options); };
  const updates = [], copied = [], errors = [];
  const model = { ID: 'fixture-message', type, status: 'success', flow: 'in',
    conversationID: 'C2Cfixture', payload: { text: 'sample text' }, getSignalingInfo: () => undefined };
  const store = { getMessageModel: id => id === model.ID ? model : undefined, getData: () => undefined,
    watch() {}, unwatch() {}, update: (...args) => updates.push(args) };
  const TYPES = { MSG_TEXT: 'TIMTextElem', MSG_AUDIO: 'TIMSoundElem', MSG_IMAGE: 'TIMImageElem',
    MSG_VIDEO: 'TIMVideoFileElem', MSG_FILE: 'TIMFileElem', CONV_GROUP: 'GROUP' };
  const Menu = sfc(read('src/TUIKit/components/TUIChat/message-list/message-tool/index.vue'), name => {
    if (name.includes('chat-uikit-engine')) return {
      default: { TYPES }, TUIStore: store, StoreName: { CHAT: 'CHAT', CUSTOM: 'CUSTOM' },
      TUITranslateService: { t: value => value }, TUIChatService: {},
    };
    if (name.includes('universal-api')) return {};
    if (name.endsWith('Icon.vue')) return { default: { render: () => vue.h('i') } };
    if (/\.(svg|png)$/.test(name)) return '';
    if (name.endsWith('/config')) return { default: { getFeatureConfig: () => new Proxy({}, { get: () => true }) } };
    if (name.endsWith('/utils/env')) return { isPC: true, isH5: false, isUniFrameWork: false };
    if (name.endsWith('/aiRobotManager')) return { default: {
      isRobotMessage: () => false, isStreamingMessage: () => false, getRobotRenderText: () => '',
    } };
    if (name.endsWith('/utils/copy')) return { default: {
      saveCurrentSelection: () => copied.push('selection'), copySelection: () => copied.push('copy'),
    } };
    if (name.includes('/Toast/')) return { Toast: info => errors.push(info), TOAST_TYPE: {} };
    if (name.endsWith('/emoji-config')) return { transformTextWithKeysToEmojiNames: x => x };
    if (name.endsWith('/enableSampleTaskStatus')) return { enableSampleTaskStatus() {} };
    if (name === '@/api') return {};
    throw Error(`Unexpected menu dependency: ${name}`);
  });
  let release;
  const AsyncMenu = vue.defineAsyncComponent(() => new Promise(resolve => { release = () => resolve(Menu); }));
  const Parent = sfc(`<template><div @click="closeChatPop"><div v-for="item in [message]" :key="item.ID"
      :id="'tui-'+item.ID" ref="messageElementListRef">${menuTemplate}</div></div></template>
    <script setup lang="ts">
    import { ref, nextTick } from 'vue';
    import MessageTool from './async-menu';
    import { outsideClick, message } from './fixture';
    const emits = defineEmits(['toggleMultipleSelectMode', 'groupMessagePinChanged']);
    const props = { isMultipleSelectMode: false, isNotInGroup: false };
    const isMultipleSelectMode = false, canModerateIncomingMessage = false, isShowEmojiPlugin = false;
    const isTopMessageDom = false, isPC = true, componentDisposed = false;
    const getGroupMessageModerationContext = () => undefined;
    const handleModerateGroupMember = () => {};
    const ensureMessageTargetModerationContext = () => {};
    const filterTopMessageDom = () => {};
    const toggleID = ref('');
    const targetMessageDom = ref();
    const messageElementListRef = ref([]);
    const messageToolListRef = ref([]);
    let messageMenuMouseButton = 2;
    ${productionFunction('closeChatPop')}
    ${productionFunction('handleToggleMessageItemForPC')}
    ${productionFunction('onMessageToolReady', true)}
    defineExpose({ open: () => handleToggleMessageItemForPC({ button: 2, target: {} }, message),
      close: closeChatPop, ready: typeof onMessageToolReady === 'undefined' ? undefined : onMessageToolReady });
    </script>`, name => {
      if (name === './async-menu') return { default: AsyncMenu };
      if (name === './fixture') return { outsideClick, message: model };
      throw Error(name);
    });
  const container = element('root');
  const app = renderer.createApp(Parent);
  app.component('TUIEmojiPlugin', { render: () => null });
  app.config.errorHandler = err => errors.push(err);
  const parent = app.mount(container);
  const button = text => walk(container).find(n => n.props.class === 'list-item' && label(n).includes(text));
  function dispatch(node, type) {
    const event = { target: node, button: 0, stopped: false, defaultPrevented: false,
      stopPropagation() { this.stopped = true; }, preventDefault() { this.defaultPrevented = true; } };
    for (let n = node; n; n = n.parent) {
      event.currentTarget = n;
      const handler = n.props[type === 'mousedown' ? 'onMousedown' : 'onClick'];
      if (Array.isArray(handler)) handler.forEach(f => f(event)); else handler?.(event);
      if (event.stopped) break;
    }
    if (!event.stopped) for (const listener of documentListeners) if (listener.event === type) listener.handler(event);
    return event;
  }
  return { app, parent, container, registrations, updates, copied, errors, button,
    release: () => release(),
    async click(text) {
      const node = button(text); assert(node, `menu action ${text} missing`);
      const down = dispatch(node, 'mousedown');
      await settle(); // mousedown closes/unmounts a cold menu before the browser's later click.
      const attached = container.contains(node);
      if (attached) dispatch(node, 'click');
      await settle(); return { attached, down };
    },
    outside() { for (const listener of documentListeners) if (listener.event === 'mousedown') listener.handler({ target: element('outside'), button: 0 }); },
  };
}

const failures = [];
for (const type of ['TIMSoundElem', 'TIMTextElem', 'TIMImageElem', 'TIMVideoFileElem', 'TIMFileElem']) {
  const h = harness(type);
  try {
    h.parent.open(); await settle();
    const prematureRegistrations = h.registrations.length;
    h.release(); await settle();
    assert(h.button('转发'));
    if (process.argv.includes('--baseline')) {
      assert.equal(prematureRegistrations, 1);
      const first = await h.click('转发');
      assert.equal(first.attached, false);
      assert.equal(h.updates.length, 0);
      h.parent.open(); await settle(); await h.click('转发');
      assert.equal(h.updates.length, 1);
      console.log(`BASELINE ${type}: first mousedown unmounted menu before click; second forward succeeded`);
      continue;
    }
    assert.equal(prematureRegistrations, 0, 'outside listener registered before async menu mounted');
    assert(h.registrations.length > 0, 'mounted menu did not register outside listener');
    assert(h.registrations.every(r => r.ignoreDomRefs && r.domRefs.contains(r.ignoreDomRefs)), 'listener retained empty/stale menu node');
    const click = await h.click('转发');
    assert(click.attached, 'mousedown unmounted the menu before click');
    assert.equal(click.down.stopped, true, 'menu mousedown must not reach other outside listeners');
    assert.equal(click.down.defaultPrevented, false, 'menu must not suppress native selection');
    assert.deepEqual(h.updates, [['CUSTOM', 'singleForwardMessageID', 'fixture-message']]);
    // Repeat on the same already-loaded chunk and test normal outside dismissal.
    h.parent.open(); await settle();
    await h.click('转发'); assert.equal(h.updates.length, 2);
    h.parent.open(); await settle(); h.outside(); await settle();
    assert.equal(h.button('转发'), undefined);
    // A late ready event after close cannot register another listener.
    const count = h.registrations.length;
    h.parent.ready?.({ messageID: 'fixture-message', element: element('stale-menu') });
    assert.equal(h.registrations.length, count);
    if (type === 'TIMTextElem') {
      h.parent.open(); await settle(); await h.click('复制');
      assert.deepEqual(h.copied, ['selection', 'copy'], 'beforeCopy and click must both run');
    }
    assert.deepEqual(h.errors, []);
    console.log(`PASS ${type}: cold first forward, warm repeat, outside dismissal, stale ready guard${type === 'TIMTextElem' ? ', copy selection preserved' : ''}`);
  } catch (error) { failures.push(`${type}: ${error.message}`); }
  finally { h.app.unmount(); }
}
if (!process.argv.includes('--baseline')) {
  const h = harness('TIMSoundElem');
  try {
    h.parent.open(); await settle();
    h.parent.close(); await settle();
    h.release(); await settle();
    assert.equal(h.registrations.length, 0, 'closed async menu must not install a late listener');
    assert.equal(h.button('转发'), undefined);
    h.parent.open(); await settle(); await h.click('转发');
    assert.equal(h.updates.length, 1);
    assert.deepEqual(h.errors, []);
    console.log('PASS: close while first menu import is pending, then reopen and forward on first click');
  } catch (error) { failures.push(`cancelled async menu: ${error.message}`); }
  finally { h.app.unmount(); }
}
assert.deepEqual(failures, [], 'Message menu first-click regression');
