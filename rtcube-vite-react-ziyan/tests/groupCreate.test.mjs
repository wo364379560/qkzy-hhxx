import assert from 'node:assert/strict'
import { test, after } from 'node:test'
import { build, transform } from 'esbuild'
import { mkdtemp, rm, readFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
const tmp = await mkdtemp(path.join(tmpdir(), 'group-create-'))
after(() => rm(tmp, { recursive: true, force: true }))
const outfile = path.join(tmp, 'group.mjs')
await build({ entryPoints: ['src/im/groupCreate.ts'], outfile, bundle: true, platform: 'node', format: 'esm', plugins: [{name:'mocks', setup(b) {
  const mocks = {
    '@/api/transport': "export const routedFetch=(p,i)=>globalThis.fetch('https://example.invalid/api'+p,i)",
    '@/api/session': "export const getToken=()=>'';export const getLocalSession=()=>({userId:'self'})",
    './ImClient': "export const getChat=()=>globalThis.chatMock;export const TencentCloudChat={TYPES:{CONV_GROUP:'GROUP'}}",
    './messageTips': 'export const markGroupJustCreated=()=>{}',
  }
  b.onResolve({filter:/^@\/utils\/groupId$/}, () => ({path:path.resolve('src/utils/groupId.ts')}))
  b.onResolve({filter:/.*/}, args => args.path in mocks ? {path:args.path,namespace:'mock'} : undefined)
  b.onLoad({filter:/.*/,namespace:'mock'}, args => ({contents:mocks[args.path]}))
}}] })
const { createGroupWithTip } = await import(pathToFileURL(outfile).href)
const ids = Array.from({length:100}, (_,i)=>`friend-${i}`)
function setup({failBatch=0,failTip=false,partial=false}={}) {
  const calls=[];let n=0
  globalThis.fetch=async()=>({json:async()=>({code:0})})
  globalThis.chatMock={
    createGroup:async options=>{ calls.push(options.memberList.map(x=>x.userID));return {data:{group:{groupID:'@TGS#test'}}} },
    addGroupMember:async options=>{
      calls.push(options.userIDList);n++
      if(n===failBatch) throw Error('network unavailable')
      return {data:{successUserIDList:partial?options.userIDList.slice(1):options.userIDList,failureUserIDList:partial?[options.userIDList[0]]:[]}}
    },
    createCustomMessage:x=>x,
    sendMessage:async message=>{if(failTip) throw Error('未实名最多5条');return {data:{message}}},
  }
  return calls
}
test('100 friends use five requests of 20 without duplicates',async()=>{
 const calls=setup();const result=await createGroupWithTip({name:'test',memberIDList:ids})
 assert.equal(result.groupID,'@TGS#test');assert.equal(result.warning,'')
 assert.deepEqual(calls.map(x=>x.length),[20,20,20,20,20]);assert.deepEqual(calls.flat(),ids)
})
test('300 friends cannot bypass the selection limit',async()=>{
 const calls=setup();await assert.rejects(createGroupWithTip({name:'test',memberIDList:Array.from({length:300},(_,i)=>String(i))}),/100/)
 assert.equal(calls.length,0)
})
test('deduplicates IDs and preserves final short batch',async()=>{
 const calls=setup();await createGroupWithTip({name:'test',memberIDList:[...ids.slice(0,21),ids[0]]})
 assert.deepEqual(calls.map(x=>x.length),[20,1]);assert.deepEqual(calls.flat(),ids.slice(0,21))
})
test('invite failure keeps created group and reports incomplete members',async()=>{
 const calls=setup({failBatch:2});const result=await createGroupWithTip({name:'test',memberIDList:ids})
 assert.equal(result.conversationID,'GROUP@TGS#test');assert.match(result.warning,/群已创建/);assert.equal(calls.length,3)
})
test('partial SDK success is not reported as full success',async()=>{
 setup({partial:true});const result=await createGroupWithTip({name:'test',memberIDList:ids})
 assert.match(result.warning,/部分好友/)
 assert.equal(JSON.parse(result.tipMessage.payload.data).members.length,96)
})
test('quota blocking the tip cannot turn an existing group into a creation failure',async()=>{
 setup({failTip:true});const result=await createGroupWithTip({name:'test',memberIDList:ids})
 assert.equal(result.groupID,'@TGS#test');assert.match(result.warning,/提示消息/)
})
test('React select-all chooses first 100 of 300 and toggles clear',async()=>{
 const source=await readFile('src/components/CreateGroupModal.tsx','utf8')
 const body=source.slice(source.indexOf('  function toggleSelectAll()'),source.indexOf('  function onRefreshId()'))
 let selected=[]
 const run=()=>new Function('filtered','selected','allVisibleSelected','setSelected',body+';toggleSelectAll()')(Array.from({length:300},(_,i)=>({userID:`friend-${i}`})),selected,selected.length>=100,value=>{selected=typeof value==='function'?value(selected):value})
 run();assert.deepEqual(selected,ids);run();assert.deepEqual(selected,[])
})
test('Vue select-all fills 100 slots rather than rejecting all 300',async()=>{
 const source=await readFile('../chengnan_web_原版/src/TUIKit/components/common/Transfer/index.vue','utf8')
 const body=source.slice(source.indexOf('const selectedAll = () =>'),source.indexOf('const submit = () =>'))
 const compiled=(await transform(body,{loader:'ts'})).code
 const selected={value:[]};const optional={value:Array.from({length:300},(_,i)=>({userID:`friend-${i}`}))}
 new Function('props','isAllOptionalSelected','transferSelectedList','optional','isSelected','maxSelectableCount','notifyMaxSelectionReached',compiled+';selectedAll()')({}, {value:false},selected,optional,x=>selected.value.some(s=>s.userID===x.userID),{value:100},()=>{})
 assert.deepEqual(selected.value.map(x=>x.userID),ids)
})
async function vueCreate(members, type='Community', failBatch=0) {
 const source=await readFile('../chengnan_web_原版/src/TUIKit/components/TUIGroup/create-group/index.vue','utf8')
 const body=source.slice(source.indexOf('const createGroup = async'),source.indexOf('const submit = () =>'))
 const compiled=(await transform(body,{loader:'ts'})).code
 const calls=[],toasts=[];let complete;let n=0
 const service={createGroup:async o=>{calls.push(o.memberList.map(m=>m.userID));return {data:{group:{groupID:'g',type}}}},getGroupMemberList:async()=>{},addGroupMember:async o=>{calls.push(o.userIDList);if(++n===failBatch)throw Error('network');return {data:{}}}}
 const run=new Function('TUIChatEngine','TUIGroupService','TUIStore','StoreName','isCreating','withBusy','setGroupMemberNicknameEnabled','Toast','TOAST_TYPE','TUITranslateService','handleCompleteCreate','GROUP_CREATE_INITIAL_MEMBER_LIMIT','GROUP_INVITE_BATCH_SIZE',compiled+';return createGroup')(
 {TYPES:{GRP_COMMUNITY:'Community',GRP_AVCHATROOM:'AVChatRoom',JOIN_OPTIONS_FREE_ACCESS:'FreeAccess'}},service,{update(){},getData(){return {userID:'me'}}},{GRP:'g',USER:'u'},{value:false},async(_,fn)=>fn(),()=>{},x=>toasts.push(x),{WARNING:'warning',ERROR:'error',SUCCESS:'success'},{t:x=>x},x=>{complete=x},20,20)
 await run({type,memberList:members.map(userID=>({userID}))})
 return {calls,toasts,complete}
}
test('Vue creates both Community and Public with five requests for 100 friends',async()=>{
 for(const type of ['Community','Public']) {
  const result=await vueCreate(ids,type)
  assert.deepEqual(result.calls.map(x=>x.length),[20,20,20,20,20]);assert.deepEqual(result.calls.flat(),ids);assert.equal(result.complete.groupID,'g')
 }
})
test('Vue keeps a created group and warns when an invite batch fails',async()=>{
 const result=await vueCreate(ids,'Community',2)
 assert.equal(result.complete.groupID,'g');assert.equal(result.calls.length,3);assert.equal(result.toasts.at(-1).type,'warning');assert.match(result.toasts.at(-1).message,/群已创建/)
})
