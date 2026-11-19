// Keep transport and eager login-registration behavior tied to the pinned vendor
// implementation. Native media/WASM is deliberately outside these local fixtures.
import { readFileSync } from 'node:fs';
import ts from 'typescript';
const read = path => readFileSync(new URL('../../node_modules/' + path, import.meta.url), 'utf8');
const engineMap = JSON.parse(read('@trtc/call-engine-lite-js/index.esm.js.map'));
const engine = engineMap.sourcesContent.find(s => s?.includes('class JSTim{'));
export const jsTimSource = engine.slice(engine.indexOf('class JSTim{'), engine.indexOf('JSTim.logPrefix', engine.indexOf('class JSTim{')));
const core = read('@tencentcloud/tui-core-lite/index.js');
const begin = core.indexOf('class U{constructor(){this.eventMap');
if (begin < 0) throw new Error('Pinned TUI-core event manager changed; review registration contract');
export const coreEventManagerSource = core.slice(begin, core.indexOf('class ', begin + 6)).replace(/^class U/, 'class VendorEventManager');
const combine = read('@trtc/calls-uikit-vue/src/TUICallService/CallService/chatCombine.ts');
const tree = ts.createSourceFile('vendor.ts', combine, ts.ScriptTarget.Latest, true);
export const chatCombineSource = ts.transpileModule(tree.statements.filter(n => !ts.isImportDeclaration(n)).map(n => n.getText(tree)).join('\n').replace('export default class', 'class'), { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } }).outputText;
const service = ts.createSourceFile('vendor.ts', read('@trtc/calls-uikit-vue/src/TUICallService/CallService/index.ts'), ts.ScriptTarget.Latest, true);
const serviceClass = service.statements.find(ts.isClassDeclaration);
export const callLifecycleMethods = ['handleExceptionExit', 'destroyed'].map(name => serviceClass.members.find(n => n.name?.getText(service) === name).getText(service)).join('\n');
