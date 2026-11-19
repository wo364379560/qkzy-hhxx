import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import ts from 'typescript';
import { test } from 'node:test';
const compile=s=>ts.transpileModule(s,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}}).outputText;
const {createCallSignalInbox}=await import('data:text/javascript;base64,'+Buffer.from(compile(readFileSync(new URL('../src/features/call/callSignalInbox.ts',import.meta.url),'utf8'))).toString('base64'));
const signaling={NEW_INVITATION_RECEIVED:'newInvitationReceived',INVITEE_ACCEPTED:'ts_invitee_accepted',INVITEE_REJECTED:'ts_invitee_rejected',INVITATION_CANCELLED:'ts_invitation_cancelled',INVITATION_TIMEOUT:'ts_invitation_timeout'};
const events={MESSAGE_RECEIVED:'onMessageReceived',ROOM_CUSTOM_DATA_RECEIVED:'onRoomCustomDataReceived',SERVER_CONFIG_UPDATED:'onServerConfigUpdated',ERROR:'error',KICKED_OUT:'kickedOut',NET_STATE_CHANGE:'netStateChange'};
// Exercise the installed CallEngine's real JS transport adapter (not a rewritten
// interpretation). WASM/RPC and media hardware remain external integration tests.
const map=JSON.parse(readFileSync(new URL('../node_modules/@trtc/call-engine-lite-js/index.esm.js.map',import.meta.url),'utf8'));
const vendor=map.sourcesContent.find(s=>s?.includes('class JSTim{'));
const begin=vendor.indexOf('class JSTim{'),end=vendor.indexOf('JSTim.logPrefix',begin);
const adapterSource=vendor.slice(begin,end)+';globalThis.JSTim=JSTim;';
function fixture(options={}) {
 const listeners=new Map(); let current=true,now=1000,logouts=0;
 const chat={on(n,f){const s=listeners.get(n)||new Set();s.add(f);listeners.set(n,s);},off(n,f){listeners.get(n)?.delete(f);},
  addSignalingListener(n,f){this.on(n,f)},removeSignalingListener(n,f){this.off(n,f)},
  emit(n,data){for(const f of [...(listeners.get(n)||[])])f(data)},getLoginUser:()=> 'self',isReady:()=>true,
  logout(){logouts++;return Promise.resolve()},destroy(){logouts++;return Promise.resolve()},login(){assert.fail('CallKit cannot start a second business login')}};
 const inbox=createCallSignalInbox(chat,{active:()=>current,userID:'self',events,signaling,now:()=>now,...options});
 const seen=[]; const context={TIM:{EVENT:events},TSignaling:signaling,Module:{logger:console},ChatLoginManager:class{},console};
 vm.runInNewContext(adapterSource,context);const adapter=new context.JSTim();adapter.tim=inbox.chat;adapter.bindObserver({onJSEvent:(name,data)=>seen.push({name,data}),delete(){}});
 return {chat,inbox,seen,adapter,bind:()=>adapter.bindTIMEvent(),setCurrent:v=>current=v,setNow:v=>now=v,logouts:()=>logouts,listeners};
}
const invite=(id,groupID='')=>({data:{inviteID:id,inviter:'caller',inviteeList:['self'],groupID,data:JSON.stringify({businessID:1,platform:'Android',data:{cmd:'videoCall',roomID:123}})}});
test('standard C2C and group invitations before SDK_READY survive delayed call chunk and reach real JSTim once',()=>{
 const f=fixture();let ordinary=0;f.chat.on(events.MESSAGE_RECEIVED,()=>ordinary++);
 f.chat.emit(signaling.NEW_INVITATION_RECEIVED,invite('c2c'));
 f.chat.emit(signaling.NEW_INVITATION_RECEIVED,invite('group','group-1000'));
 f.chat.emit(events.MESSAGE_RECEIVED,{data:[{type:'TIMTextElem',payload:{text:'ordinary'}}]});
 assert.equal(ordinary,1);assert.equal(f.seen.length,0);
 f.bind();f.inbox.activate();f.inbox.activate();
 assert.deepEqual(f.seen.map(e=>[e.name,e.data.inviteID,e.data.groupID]),[['onNewInvitationReceived','c2c',''],['onNewInvitationReceived','group','group-1000']]);
 assert.equal(ordinary,1,'never replay events on the real Chat');
 f.chat.emit(signaling.NEW_INVITATION_RECEIVED,invite('live'));assert.equal(f.seen.length,3);
});
test('v3 room control, server config and custom-message signaling preserve native adapter inputs and order',()=>{
 const f=fixture();const room={name:events.ROOM_CUSTOM_DATA_RECEIVED,data:{room_id:'room-a',custom_data:'invite'}};
 f.chat.emit(events.ROOM_CUSTOM_DATA_RECEIVED,room);
 f.chat.emit(events.SERVER_CONFIG_UPDATED,{data:{call:true}});
 f.chat.emit(events.MESSAGE_RECEIVED,{data:[{type:'TIMCustomElem',payload:{data:'call signal'}},{type:'TIMTextElem',payload:{text:'chat'}}]});
 f.bind();f.inbox.activate();assert.equal(f.seen[0].name,'onRoomCustomDataReceived');assert.equal(f.seen[0].data,room);
 assert.equal(f.seen[1].name,'onServerConfigUpdated');assert.equal(JSON.parse(f.seen[1].data).call,true);
 assert.equal(f.seen[2].name,'onNewMessageReceived');assert.equal(JSON.parse(f.seen[2].data).length,1);
});
test('cancelled/expired invitations do not ring after the loading window',()=>{
 for(const terminal of [signaling.INVITATION_CANCELLED,signaling.INVITATION_TIMEOUT]){
 const f=fixture();f.chat.emit(signaling.NEW_INVITATION_RECEIVED,invite('gone'));
 f.chat.emit(terminal,{data:{inviteID:'gone',inviteeList:['self']}});f.bind();f.inbox.activate();
 assert(!f.seen.some(e=>e.name==='onNewInvitationReceived'));
 }
});
test('SDK-ready delivery, cancellation and acceptance during draining cannot overtake prior invitations',()=>{
 const f=fixture();f.chat.emit(signaling.NEW_INVITATION_RECEIVED,invite('a'));
 f.chat.emit(signaling.INVITEE_ACCEPTED,{data:{inviteID:'a',invitee:'other'}});f.bind();
 f.inbox.activate();assert.deepEqual(f.seen.map(e=>e.name),['onNewInvitationReceived','onInviteeAccepted']);
 f.chat.emit(signaling.INVITATION_CANCELLED,{data:{inviteID:'a'}});assert.equal(f.seen.at(-1).name,'onInvitationCancelled');
});
test('logout/switch clears queued calls, guards late events and never logs out shared Chat',async()=>{
 const f=fixture();f.chat.emit(signaling.NEW_INVITATION_RECEIVED,invite('old'));
 f.bind();f.setCurrent(false);f.inbox.dispose();f.inbox.activate();
 f.chat.emit(signaling.NEW_INVITATION_RECEIVED,invite('new'));assert.equal(f.seen.length,0);
 await assert.rejects(f.inbox.chat.login({userID:'self'}));await f.inbox.chat.logout();await f.inbox.chat.destroy();assert.equal(f.logouts(),0);
 assert([...f.listeners.values()].every(s=>s.size===0));
});
test('backlog is bounded by count, bytes and age; rebind does not duplicate callbacks',()=>{
 const f=fixture({maxEvents:3,maxAgeMs:50,maxBytes:4000});
 for(let i=0;i<10;i++)f.chat.emit(signaling.NEW_INVITATION_RECEIVED,invite(String(i)));
 assert.equal(f.inbox.pendingCount(),3);f.setNow(1051);assert.equal(f.inbox.pendingCount(),0);
 f.bind();f.bind();f.inbox.activate();f.chat.emit(signaling.NEW_INVITATION_RECEIVED,invite('once'));assert.equal(f.seen.length,1);
 f.adapter.unbindTIMEvent();f.chat.emit(signaling.NEW_INVITATION_RECEIVED,invite('removed'));assert.equal(f.seen.length,1);
});
