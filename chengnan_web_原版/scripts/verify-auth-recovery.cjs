/* Offline regression tests: actual session modules, API interceptors and App 401
 * handler; network/SDK/storage boundaries are mocked. No real accounts used. */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const ts = require('typescript');
const vm = require('node:vm');
const root = path.resolve(__dirname, '..');
const vue3 = fs.readFileSync(path.join(root, 'src/api/index.ts'), 'utf8').includes('/access/session/renew');
const store = () => { const m = new Map(); return { getItem:k=>m.get(k)??null, setItem:(k,v)=>m.set(k,v), removeItem:k=>m.delete(k) }; };
const flush = async () => { for(let i=0;i<35;i++) await Promise.resolve(); };
function deferred() { let resolve,reject;const promise=new Promise((a,b)=>{resolve=a;reject=b;});return{promise,resolve,reject}; }
function ast(file) { let code=fs.readFileSync(path.join(root,file),'utf8');if(file.endsWith('.vue'))code=code.match(/<script[^>]*>([\s\S]*?)<\/script>/)[1];return ts.createSourceFile(file,code,ts.ScriptTarget.Latest,true); }
function extract(file,prefix) {const tree=ast(file),node=tree.statements.find(n=>n.getText(tree).startsWith(prefix));assert(node,`${file}: ${prefix}`);return node.getText(tree);}
function compile(text) {return ts.transpileModule(text,{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2020,esModuleInterop:true}}).outputText;}
function harness(persisted, sharedStorage) {
 const h={now:Date.now(),user:'fixture-a',refreshes:0,sdkLogins:0,sdkLogouts:0,routes:[],replays:[],timers:new Map(),registered:false,initialized:false};
 const localStorage=sharedStorage || store(),sessionStorage=store();
 h.confirmReload=false;h.reloads=0;h.confirmations=[];
 const browser={addEventListener(){},location:{hash:'',reload:()=>h.reloads++},confirm:message=>{h.confirmations.push(message);return h.confirmReload;}};
 if(persisted)localStorage.setItem('fixture-session',JSON.stringify(persisted));
 let timerID=0;
 const setTimer=(fn,ms)=>{const id=++timerID;h.timers.set(id,{fn,at:h.now+ms});return id;};
 const clearTimer=id=>h.timers.delete(id);
 const Clock=class extends Date {static now(){return h.now;}};
 h.advance=async ms=>{h.now+=ms;for(let n=0;n<20;n++){const due=[...h.timers].filter(([,t])=>t.at<=h.now);if(!due.length)break;for(const[id,t]of due){h.timers.delete(id);t.fn();}await flush();}};
 h.auth=()=>({user_id:h.user,sdk_app_id:7,token:`access-${h.user}-${h.refreshes}`,refresh_token:`refresh-${h.user}-${h.refreshes}`,user_sig:`sig-${h.user}`,token_expires_at:new Date(h.now+86400000).toISOString(),user_sig_expires_at:new Date(h.now+7*86400000).toISOString(),is_activated:true,must_change_password:false});
 h.refresh=async()=>h.auth();h.sdk=async()=>{};h.presence=async()=>{};
 let apiContext;
 const mocks={
  vue:{reactive:x=>x,readonly:x=>x,shallowRef:x=>({value:x}),computed:fn=>({get value(){return fn();}})},
  '@/config/runtime':{APP_NAME:'fixture',SDK_APP_ID:7},
  '@/TUIKit/adapter-vue':{framework:'vue3'},
  '@/bootstrap/startupMetrics':{markStartup(){}},
  '@/features/user/online-status':{initOnlineStatusSync(){},resetOnlineStatusSubscriptions(){}},
  '@/features/presence/invisible':{getInvisiblePreference:()=>false,INVISIBLE_CUSTOM_STATUS:'offline'},
  '@/bootstrap/callRuntime':{prepareCallIntegration:async()=>{h.registered=true;return{};},disposeCallIntegration:async()=>{}},
  '@/bootstrap/chatRuntime':{ensureChatRuntime(){h.initialized=true;},loginChatRuntime(){}},
  '@/features/call/doMattxCallBridge':{ensureDoMattxCallBridge:()=>true,disposeDoMattxCallBridge:async()=>{}},
  '@tencentcloud/chat-uikit-engine-lite':{TUIUserService:{switchUserStatus:()=>h.presence()}},
  '@tencentcloud/tui-core-lite':{TUILogin:{login:async()=>{if(!vue3)assert(h.registered&&h.initialized);h.sdkLogins++;await h.sdk();},logout:async()=>{h.sdkLogouts++;}}},
  '@/api':{login:async()=>h.auth(),refreshSession:async payload=>{h.refreshes++;h.refreshPayload=payload;return h.refresh(payload);},setChatAuthToken:(...args)=>apiContext?.setChatAuthToken(...args),setChatCacheScope(){},logout:async()=>{},updateProfile:async()=>{},changePassword:async()=>h.auth()},
 };
 const loaded=new Map();
 function load(file) {
  const resolved=path.resolve(root,file);if(loaded.has(resolved))return loaded.get(resolved).exports;
  const module={exports:{}};loaded.set(resolved,module);
  const requireLocal=name=>{if(mocks[name])return mocks[name];if(name.startsWith('@/'))return load(`src/${name.slice(2)}.ts`);if(name.startsWith('.'))return load(path.resolve(path.dirname(resolved),`${name}.ts`));return require(name);};
  new Function('require','module','exports','localStorage','sessionStorage','window','setTimeout','clearTimeout','Date','console',compile(fs.readFileSync(resolved,'utf8')))(requireLocal,module,module.exports,localStorage,sessionStorage,browser,setTimer,clearTimer,Clock,{warn(){}});
  return module.exports;
 }
 h.helpers=load('src/features/auth/recovery.ts');h.session=load('src/features/auth/session.ts').useAuthSession();
 const noCache={clear(){}};
 const context={exports:{},...h.helpers,authSession:h.session,console:{warn(){}},Date:Clock,Promise,Error,Boolean,
  router:{currentRoute:{value:{name:'home',meta:{requiresAuth:true}}},replace:async route=>h.routes.push(route.name)},
  axios:{isAxiosError:e=>e?.isAxiosError===true},retryWithNextBaseURL:()=>null,
  shouldHandleUnauthorized:()=>true,clearRequestCache(){},setRequestCacheScope(){},
  profileReads:noCache,conversationPinReads:noCache,userLastOnlineCache:noCache,groupMessagePinCache:noCache,groupAdminsCache:noCache,
  toStatusErrorMessage:status=>`HTTP ${status}`,
  chatClient:{interceptors:{request:{use:fn=>h.request=fn},response:{use:(ok,bad)=>{h.response=ok;h.failure=bad;}}},request:async config=>{const sent=await h.request(config);h.replays.push(sent);return{status:200,config:sent};}},
 };
 apiContext=vm.createContext(context);
 const declarations=['let authToken:','let unauthorizedHandler:','let unauthorizedHandlingPromise:','let authEpoch =','let authContext:'].map(p=>extract('src/api/index.ts',p)).join('\n');
 const code=declarations+'\n'+['chatClient.interceptors.request.use(','chatClient.interceptors.response.use(','export function setChatAuthToken(','export function setChatUnauthorizedHandler(','function toApiError('].map(p=>extract('src/api/index.ts',p)).join('\n')+'\n'+extract('src/App.vue','setChatUnauthorizedHandler(');
 vm.runInContext(compile(code),apiContext);
 h.normalize=err=>apiContext.toApiError(err);
 h.business=async status=>{const config=await h.request({url:'/fixture',method:'get'});return h.failure({isAxiosError:true,config,response:{status,data:{}}});};
 h.pendingBusiness=async()=>{const config=await h.request({url:'/fixture',method:'get'});return{reject:()=>h.failure({isAxiosError:true,config,response:{status:401,data:{}}}),resolve:()=>h.response({config,status:200,data:'old'})};};
 h.storage=localStorage;h.login=async()=>{await h.session.loginWithPassword('fixture','fixture');await flush();};
 h.error=status=>new h.helpers.ApiError(status?`HTTP ${status}`:'offline',{status,code:status||'ETIMEDOUT',retryAfterMs:5000});
 return h;
}
(async()=>{
 // Real 401 handler + refresh + interceptor: retain credentials on temporary errors.
 for(const status of [undefined,429,503]) {
  const h=harness();await h.login();h.refresh=async()=>{throw h.error(status);};
  await assert.rejects(h.business(401));assert(h.session.isAuthenticated.value);assert.equal(h.routes.length,0);assert.equal(h.refreshes,1);
  await assert.rejects(h.business(401));assert.equal(h.refreshes,1,'cooldown suppresses refresh storm');
  await h.advance(4999);assert.equal(h.refreshes,1,'honor Retry-After');
  await h.advance(1);assert.equal(h.refreshes,2);
  await h.advance(6000);assert.equal(h.refreshes,3);
  await h.advance(60000);assert.equal(h.refreshes,3,'only two automatic retries');
  assert.equal(h.session.state.recoveryStatus,'unavailable');
 }
 {
  const h=harness();await h.login();h.refresh=async()=>{throw h.error(401);};await assert.rejects(h.business(401));await flush();
  assert.equal(h.session.isAuthenticated.value,false);assert(h.routes.includes('login'));
 }
 {
  const h=harness();await h.login();await assert.rejects(h.business(503));assert.equal(h.refreshes,0);assert(h.session.isAuthenticated.value);
  const gate=deferred();h.refresh=()=>gate.promise;
  const requests=Array.from({length:5},()=>h.business(401));await flush();assert.equal(h.refreshes,1);gate.resolve(h.auth());await Promise.all(requests);
  assert.equal(h.replays.length,5);assert.equal(h.refreshPayload.sdkAppID,7);assert(h.session.isAuthenticated.value);
 }
 // An ordinary business 401 renews only the access token, without reconnecting IM.
 {
  const h=harness();await h.login();await h.session.ensureChatConnected();const logins=h.sdkLogins;
  h.refresh=async()=>{const value=h.auth();delete value.user_sig;delete value.user_sig_expires_at;return value;};
  await h.business(401);assert.equal(h.refreshPayload.needUserSigRefresh,false);assert.equal(h.sdkLogins,logins);
 }
 // Late response from a previous account must neither clear nor replay as the new account.
 for(const rejectOld of [true,false]) {
  const h=harness();await h.login();const gate=deferred();h.refresh=()=>gate.promise;
  const oldRefresh=h.session.refreshCurrentSession({force:true});const oldBusiness=await h.pendingBusiness();
  await h.session.clearSession();h.user='fixture-b';await h.login();
  if(rejectOld)gate.reject(h.error(401));else gate.resolve({...h.auth(),user_id:'fixture-a'});
  await oldRefresh;await assert.rejects(oldBusiness.reject());await assert.rejects(Promise.resolve().then(oldBusiness.resolve));
  assert.equal(h.session.currentSession.value.userID,'fixture-b');assert.equal(h.replays.length,0);
 }
 // The refresh can succeed while the IM connection fails: business login survives.
 {
  const h=harness();await h.login();h.sdk=async()=>{throw Error('SDK offline');};
  h.refresh=async()=>({...h.auth(),user_sig:'new-sig'});await h.business(401);await flush();
  assert(h.session.isAuthenticated.value);assert.equal(h.session.state.chatStatus,'error');assert.equal(h.replays.length,1);
 }
 // A refresh error during startup retains the persisted login; next automatic retry can recover.
 {
  const a=harness();await a.login();const saved=JSON.parse(a.storage.getItem('fixture-session'));saved.accessTokenExpiresAt=new Date(0).toISOString();
  const h=harness(saved);h.refresh=async()=>{throw h.error(503);};await h.session.ensureInitialized();assert(h.session.isAuthenticated.value);
  h.refresh=async()=>h.auth();await h.advance(6000);assert.equal(h.session.state.recoveryStatus,'idle');assert(h.session.isAuthenticated.value);
 }
 // Startup with healthy cached credentials does not force a refresh round trip.
 {
  const a=harness();await a.login();const h=harness(JSON.parse(a.storage.getItem('fixture-session')));await h.session.ensureInitialized();assert.equal(h.refreshes,0);
 }
 // Password-change credentials must survive reload as a newly authorized family.
 if(!vue3) {
  const h=harness();await h.login();await h.session.ensureChatConnected();const oldFamily=h.session.currentSession.value.sessionFamilyID;const sdkLogins=h.sdkLogins;
  await h.session.completePasswordChange('new-password');const changed=h.session.currentSession.value;
  assert.notEqual(changed.sessionFamilyID,oldFamily);assert.equal(JSON.parse(h.storage.getItem('fixture-session')).sessionFamilyID,changed.sessionFamilyID);assert.equal(h.sdkLogins,sdkLogins,'business password change does not reconnect IM');
  const reloaded=harness(undefined,h.storage);await reloaded.session.ensureInitialized();assert.equal(reloaded.session.currentSession.value.sessionFamilyID,changed.sessionFamilyID);assert.equal(reloaded.session.currentSession.value.token,changed.token);
 }
 // Two tabs restoring one login share rotations, never a later explicit login.
 {
  const a=harness();await a.login();const b=harness(undefined,a.storage);await b.session.ensureInitialized();
  const family=a.session.currentSession.value.sessionFamilyID;
  await a.session.refreshCurrentSession({force:true});const rotated=a.session.currentSession.value.refreshToken;
  b.refresh=async payload=>{assert.equal(payload.refreshToken,rotated);return b.auth();};
  await b.advance(70000);await b.session.refreshCurrentSession({force:true});
  assert.equal(b.refreshes,0,'adopt healthy same-family rotation before network');
  assert.equal(b.session.currentSession.value.refreshToken,rotated);assert.equal(b.session.currentSession.value.sessionFamilyID,family);
 }
 // The shared rotation may arrive while an old-token request is already pending.
 {
  const a=harness();await a.login();const b=harness(undefined,a.storage);await b.session.ensureInitialized();const old=deferred();
  b.refresh=payload=>b.refreshes===1?old.promise:Promise.resolve({...b.auth(),refresh_token:'latest-after-retry'});
  const pending=b.session.refreshCurrentSession({force:true});await flush();
  await a.session.refreshCurrentSession({force:true});old.reject(b.error(401));
  assert.equal(await pending,true);assert.equal(b.refreshes,2,'one retry with new same-family token');assert.equal(b.session.currentSession.value.refreshToken,'latest-after-retry');
 }
 // A second local caller may adopt the other tab's rotation before old 401 arrives.
 {
  const a=harness();await a.login();const b=harness(undefined,a.storage);await b.session.ensureInitialized();const old=deferred();
  b.refresh=()=>b.refreshes===1?old.promise:Promise.resolve({...b.auth(),refresh_token:'after-shared-adoption'});
  const pending=b.session.refreshCurrentSession({force:true});await flush();await a.session.refreshCurrentSession({force:true});
  await b.session.refreshCurrentSession({force:true});assert.equal(b.refreshes,1);
  old.reject(b.error(401));assert.equal(await pending,true);assert.equal(b.session.isAuthenticated.value,true);assert.equal(b.refreshes,2);
 }
 // A late successful old-token response cannot overwrite a newer rotation.
 {
  const a=harness();await a.login();const b=harness(undefined,a.storage);await b.session.ensureInitialized();const old=deferred();
  b.refresh=()=>old.promise;const pending=b.session.refreshCurrentSession({force:true});await flush();
  await a.session.refreshCurrentSession({force:true});const latest=a.storage.getItem('fixture-session');
  old.resolve({...b.auth(),refresh_token:'stale-response'});await pending;
  assert.equal(a.storage.getItem('fixture-session'),latest);assert.equal(b.session.currentSession.value.refreshToken,a.session.currentSession.value.refreshToken);
 }
 // An old tab's 401, successful refresh, or logout must not erase/replace another login.
 for(const sameUser of [true,false]) for(const outcome of ['401','success','logout']) {
  const oldTab=harness();await oldTab.login();const newTab=harness(undefined,oldTab.storage);await newTab.session.ensureInitialized();
  const gate=deferred();oldTab.refresh=()=>gate.promise;
  const pending=outcome==='logout'?null:oldTab.session.refreshCurrentSession({force:true});await flush();
  newTab.user=sameUser?'fixture-a':'fixture-b';await newTab.login();const newer=newTab.storage.getItem('fixture-session');
  assert.notEqual(oldTab.session.currentSession.value.sessionFamilyID,newTab.session.currentSession.value.sessionFamilyID);
  if(outcome==='logout')await oldTab.session.logoutSession();
  else {if(outcome==='401')gate.reject(oldTab.error(401));else gate.resolve(oldTab.auth());await pending;}
  assert.equal(newTab.storage.getItem('fixture-session'),newer,`old ${outcome} preserves new ${sameUser?'same-user':'other-user'} login`);
  assert.equal(newTab.session.isAuthenticated.value,true);
 }
 // Legacy shared records migrate to one deterministic family and can rotate.
 {
  const seed=harness();await seed.login();const record=JSON.parse(seed.storage.getItem('fixture-session'));delete record.sessionFamilyID;delete record.credentialsRevision;
  if(vue3)delete record.sdkAppID;
  const a=harness(record),b=harness(undefined,a.storage);await a.session.ensureInitialized();await b.session.ensureInitialized();
  assert.equal(a.session.currentSession.value.sessionFamilyID,b.session.currentSession.value.sessionFamilyID);
  await a.session.refreshCurrentSession({force:true});await b.session.refreshCurrentSession({force:true});
  assert.equal(b.session.currentSession.value.refreshToken,a.session.currentSession.value.refreshToken);assert.equal(b.session.currentSession.value.sdkAppID,7);
 }
 const h=harness();
 const e=h.normalize({isAxiosError:true,response:{status:429,headers:{'retry-after':'7'},data:{code:123,message:'limit'}}});
 assert.equal(e.status,429);assert.equal(e.code,123);assert.equal(e.retryAfterMs,7000);
 console.log('Rui auth recovery / account generation regressions passed');
})().catch(error=>{console.error(error);process.exitCode=1;});
