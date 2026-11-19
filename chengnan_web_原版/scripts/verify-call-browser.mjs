// Local-only Chrome integration: production session/call runtime/inbox modules,
// controlled IM/API transport and the installed vendor JSTim event adapter.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createServer } from 'node:http';
import { build } from 'esbuild';
import { fileURLToPath } from 'node:url';
import { jsTimSource as vendor, coreEventManagerSource, chatCombineSource } from './fixtures/installed-call-sources.mjs';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const root=fileURLToPath(new URL('../',import.meta.url));
const constants=`{MESSAGE_RECEIVED:'onMessageReceived',ROOM_CUSTOM_DATA_RECEIVED:'onRoomCustomDataReceived',SERVER_CONFIG_UPDATED:'onServerConfigUpdated',SDK_READY:'ready',ERROR:'error',KICKED_OUT:'kick',NET_STATE_CHANGE:'net'}`;
const coreConstants=`{TUICalling:{SERVICE:{NAME:'call'}},TUIChat:{EXTENSION:{INPUT_MORE:{EXT_ID:'call-extension'}}},TUILogin:{EVENT:{LOGIN_STATE_CHANGED:'login'},EVENT_SUB_KEY:{USER_LOGIN_SUCCESS:'in',USER_LOGOUT_SUCCESS:'out'}}}`;
const signal=`{NEW_INVITATION_RECEIVED:'newInvitationReceived',INVITEE_ACCEPTED:'ts_invitee_accepted',INVITEE_REJECTED:'ts_invitee_rejected',INVITATION_CANCELLED:'ts_invitation_cancelled',INVITATION_TIMEOUT:'ts_invitation_timeout'}`;
const kit=`const f=window.fixture;const Module={logger:console},TIM={EVENT:${constants}},TSignaling=${signal};class ChatLoginManager{};${vendor}
const TUICore=f.core,TUIConstants=${coreConstants},TUILogin={getContext:()=>({chat:f.chat,userID:f.user,userSig:'fixture',SDKAppID:7})};
const TuiStore={getInstance:()=>({})},COMPONENT={TIM_CALL_KIT:'chat'},LOG_LEVEL={NORMAL:0},NAME={PREFIX:'fixture'},TencentCloudChat={EVENT:{GROUP_ATTRIBUTES_UPDATED:'group-attrs'}};${chatCombineSource}
let adapter,tim;export const TUICallKitAPI={init:async p=>{if(!f.wasmReady)throw Error('Please listen to the ready event and then execute the constructor method.');if(adapter)return;if(p.tim===f.chat)f.unsafeAutoInits++;tim=p.tim;adapter=new JSTim();adapter.tim=tim;adapter.bindObserver({onJSEvent:(name,data)=>f.incoming.push({name,data}),delete(){}});adapter.bindTIMEvent();await f.initGate?.();},handleExceptionExit:async()=>{f.activeCall=false;},destroyed:async()=>{if(f.activeCall)throw Error('destroy during active call');if(adapter){adapter.unbindTIMEvent();adapter.unBindObserver();adapter=null;}},getTim:()=>tim,setIsFromChat(){},setLogLevel(){}};
f.callService=new ChatCombine({callService:TUICallKitAPI});
export const TUICallKit={setup(){return()=>f.vue.h('div',{id:'standard-call-kit'},f.incoming.filter(x=>x.name==='onNewInvitationReceived').map(x=>f.vue.h('div',{'data-invite':x.data.inviteID},x.data.groupID?'群来电':'标准来电')))}};`;
const mocks={
 '@trtc/call-engine-lite-js':`export const TUICallEngine={once:(_event,fn)=>{const f=window.fixture;if(f.wasmReady)fn();else f.wasmListeners.push(fn)}};`,
 '@tencentcloud/chat-uikit-engine-lite':`export const TUIUserService={switchUserStatus:async()=>{}};export default window.fixture.engine;`,
 '@tencentcloud/tui-core-lite':`export const TUIConstants=${coreConstants};export const TUILogin={getContext:()=>({chat:window.fixture.chat}),login:()=>window.fixture.login(),logout:async()=>{window.fixture.businessLogouts++}};export default window.fixture.core;`,
 '@/api':`export const setChatAuthToken=()=>{};export const login=async()=>window.fixture.auth();export const refreshSession=async()=>window.fixture.auth();export const logout=async()=>{};export const changePassword=login,deleteAccount=logout,register=login,setupVerifyLoginTOTP=login,updateProfile=login,verifyLoginTOTP=login;`,
 '@/config/runtime':`export const APP_NAME='fixture',SDK_APP_ID=7;`,
 '@/TUIKit/adapter-vue':`export const framework='vue3';`,
 '@/bootstrap/chatRuntime':`export const ensureChatRuntime=()=>{},loginChatRuntime=()=>{};`,
 '@/features/presence/invisible':`export const getInvisiblePreference=()=>false,restoreSelfInvisibleStatus=async()=>{};`,
 '@/features/call/doMattxCallBridge':`export const ensureDoMattxCallBridge=()=>true,disposeDoMattxCallBridge=async()=>{};`,
};
const entry=`import * as vue from 'vue';import {useAuthSession} from './src/features/auth/session';import {callIntegrationState,prepareCallIntegration,retryCallIntegration} from './src/bootstrap/callRuntime';
const f=window.fixture;f.vue=vue;f.incoming=vue.reactive([]);const session=useAuthSession();f.session=session;f.retry=retryCallIntegration;
const CallKit=vue.defineAsyncComponent(async()=>(await prepareCallIntegration()).TUICallKit);
vue.createApp({setup(){return()=>vue.h('main',[
 vue.h('div',{id:'chat-status'},session.state.chatStatus),vue.h('div',{id:'call-status'},callIntegrationState.status),
 vue.h('button',{id:'send',onClick:()=>f.messages.push('已发送')},'发消息'),vue.h('div',{id:'messages'},f.messages.join('|')),
 callIntegrationState.loaded?vue.h(CallKit):null]);}}).mount('#app');
f.messages=vue.reactive(f.messages);await session.loginWithPassword('fixture','fixture');`;
const bundle=await build({stdin:{contents:entry,resolveDir:root,sourcefile:'browser-fixture.ts',loader:'ts'},bundle:true,format:'esm',target:'es2022',write:false,absWorkingDir:root,plugins:[{name:'transport-fixtures',setup(b){b.onResolve({filter:/.*/},args=>{if(args.path==='@trtc/calls-uikit-vue')return{path:'/test-call-kit.js',external:true};if(mocks[args.path])return{path:args.path,namespace:'fixture'}});b.onLoad({filter:/.*/,namespace:'fixture'},args=>({contents:mocks[args.path],loader:'ts'}));}}]});
const bootstrap=`window.fixture={user:'self',businessLogouts:0,unsafeAutoInits:0,wasmReady:false,wasmListeners:[],messages:[],incoming:[]};const f=window.fixture;f.markWasmReady=()=>{f.wasmReady=true;for(const fn of f.wasmListeners.splice(0))fn();};const handlers=new Map();f.chat={on(n,h){const s=handlers.get(n)||new Set();s.add(h);handlers.set(n,s)},off(n,h){handlers.get(n)?.delete(h)},addSignalingListener(n,h){this.on(n,h)},removeSignalingListener(n,h){this.off(n,h)},emit(n,data){for(const h of [...(handlers.get(n)||[])])h(data)},getLoginUser:()=>f.user,isReady:()=>f.ready,logout:async()=>{f.businessLogouts++},login:async()=>{throw Error('Call engine must not log in shared Chat')}};f.engine={chat:f.chat,EVENT:${constants},TSignaling:${signal}};f.auth=()=>({user_id:f.user,sdk_app_id:7,token:'fixture-access',refresh_token:'fixture-refresh',user_sig:'fixture-sig',token_expires_at:new Date(Date.now()+86400000).toISOString(),user_sig_expires_at:new Date(Date.now()+7*86400000).toISOString(),is_activated:true});f.invite=(id,groupID='')=>({data:{inviteID:id,inviter:'fixture-caller',inviteeList:[f.user],groupID,data:JSON.stringify({businessID:1,data:{cmd:'videoCall',roomID:123}})}});const _=${coreConstants},l={getInstance:()=>({getContext:()=>({chat:f.chat})})};${coreEventManagerSource};const manager=new VendorEventManager();f.core={registerEvent:(...a)=>manager.registerEvent(...a),unregisterEvent:(...a)=>manager.unregisterEvent(...a),getService:()=>f.callService,registerService:(_,s)=>f.callService=s,registerExtension(){}};f.login=()=>new Promise(resolve=>{queueMicrotask(()=>{f.chat.emit('newInvitationReceived',f.invite('pre-ready'));f.ready=true;f.chat.emit('ready',{});setTimeout(resolve,40)})});`;
const server=createServer((req,res)=>{if(req.url==='/app.js'){res.setHeader('Content-Type','text/javascript');res.end(bundle.outputFiles[0].text);}else if(req.url==='/test-call-kit.js'){res.setHeader('Content-Type','text/javascript');res.end(kit);}else{res.setHeader('Content-Type','text/html');res.end(`<div id="app"></div><script>${bootstrap}</script><script type="module" src="/app.js"></script>`);}});
await new Promise(r=>server.listen(0,'127.0.0.1',r));const origin=`http://127.0.0.1:${server.address().port}`;
const browser=await chromium.launch({executablePath:process.env.CHROME_EXECUTABLE_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',headless:true,args:['--no-first-run']});
try{
 // Hold the complete call UI/engine chunk while actual production auth finishes.
 const context=await browser.newContext();const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));let pendingRoute;
 await page.route('**/test-call-kit.js',route=>{pendingRoute=route;});
 await page.goto(origin);await page.waitForFunction(()=>document.querySelector('#chat-status')?.textContent==='ready');
 await page.evaluate(()=>{fixture.chat.emit('newInvitationReceived',fixture.invite('standard'));fixture.chat.emit('newInvitationReceived',fixture.invite('group','community-1000'));fixture.chat.emit('newInvitationReceived',fixture.invite('canceled'));fixture.chat.emit('ts_invitation_cancelled',{data:{inviteID:'canceled'}});fixture.messages.push('已收到');});
 await page.click('#send');assert.match(await page.locator('#messages').innerText(),/已收到.*已发送/);
 await page.evaluate(()=>{fixture.unrelated=0;fixture.core.registerEvent('login','in',{onNotifyEvent(){fixture.unrelated++}})});assert.equal(await page.evaluate(()=>fixture.unrelated),1,'ordinary chat plugin login notifications must remain immediate');
 assert.equal(await page.locator('#call-status').innerText(),'loading');
 assert.equal(await page.evaluate(()=>fixture.incoming.length),0);
 await pendingRoute.fulfill({contentType:'text/javascript',body:kit});
 await page.waitForFunction(()=>fixture.wasmListeners.length===1);assert.equal(await page.locator('#call-status').innerText(),'loading');assert.equal(await page.locator('#chat-status').innerText(),'ready');
 await page.click('#send');await page.evaluate(()=>fixture.markWasmReady());
 await page.waitForFunction(()=>document.querySelector('#call-status')?.textContent==='ready');
 assert.deepEqual(await page.locator('[data-invite]').evaluateAll(nodes=>nodes.map(n=>n.getAttribute('data-invite'))),['pre-ready','standard','group']);
 assert.equal(await page.evaluate(()=>fixture.businessLogouts),0);assert.equal(await page.evaluate(()=>fixture.unsafeAutoInits),0,'actual vendor constructor must not renotify init against real Chat');assert.deepEqual(errors,[]);
 await page.evaluate(async()=>{await fixture.session.completePasswordChange('fixture-new');fixture.chat.emit('newInvitationReceived',fixture.invite('after-password'));});
 await page.waitForSelector('[data-invite="after-password"]');
 console.log('PASS Chrome: password rotation preserves incoming calls on the existing IM connection.');
 await page.evaluate(async()=>{await fixture.session.loginWithPassword('fixture','fixture');});await page.waitForFunction(()=>document.querySelector('#call-status')?.textContent==='ready');
 await page.evaluate(()=>fixture.chat.emit('newInvitationReceived',fixture.invite('second-login')));await page.waitForSelector('[data-invite="second-login"]');
 console.log('PASS Chrome: same-account login with identical signature binds the new IM scope.');
 console.log('PASS Chrome: independent delayed WASM ready gates call construction while ordinary chat stays usable; no manual retry.');
 console.log('PASS Chrome: blocked CallKit does not block IM/chat; SDK_READY-window, standard and group invitations reach installed JSTim and mounted UI once; cancelled call omitted.');
 await context.close();
 // A permanently failed chunk degrades calls only, with ordinary messaging alive.
 const failed=await browser.newContext();const p=await failed.newPage();await p.route('**/test-call-kit.js',r=>r.abort('failed'));await p.goto(origin);
 await p.waitForFunction(()=>document.querySelector('#chat-status')?.textContent==='ready'&&document.querySelector('#call-status')?.textContent==='error');
 await p.click('#send');assert.match(await p.locator('#messages').innerText(),/已发送/);assert(await p.evaluate(()=>fixture.session.isAuthenticated.value));
 console.log('PASS Chrome: rejected call chunk keeps business login and ordinary chat usable.');await failed.close();
 // Old queued invites and in-flight initialization cannot reach the next user.
 const switched=await browser.newContext();const s=await switched.newPage();let oldRoute;
 await s.route('**/test-call-kit.js',r=>{oldRoute=r;});await s.goto(origin);await s.waitForFunction(()=>document.querySelector('#chat-status')?.textContent==='ready');
 await s.evaluate(async()=>{fixture.chat.emit('newInvitationReceived',fixture.invite('old-user'));await fixture.session.clearSession();fixture.user='next';await fixture.session.loginWithPassword('fixture','fixture');});
 await oldRoute.fulfill({contentType:'text/javascript',body:kit});await s.waitForFunction(()=>fixture.wasmListeners.length===1);await s.evaluate(()=>fixture.markWasmReady());await s.waitForFunction(()=>document.querySelector('#call-status')?.textContent==='ready');
 const ids=await s.locator('[data-invite]').evaluateAll(ns=>ns.map(n=>n.getAttribute('data-invite')));assert(!ids.includes('old-user'));assert.deepEqual(ids,['pre-ready']);
 console.log('PASS Chrome: account switch during chunk load discards the previous account invitations.');await switched.close();
}finally{await browser.close();await new Promise(r=>server.close(r));}
