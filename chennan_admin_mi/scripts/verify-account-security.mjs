import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import ts from 'typescript'

const requests = []
const http = { post: async (path, body) => { requests.push({ path, body }); return { data: { data: {} } } } }
function load(relative) {
  const source = readFileSync(new URL(relative, import.meta.url), 'utf8')
  const compiled = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 } }).outputText
  const mod = { exports: {} }
  new Function('require', 'module', 'exports', compiled)(name => {
    if (name === './request') return { default: http }
    if (name === '../utils/adminAccount') return load('../src/utils/adminAccount.ts')
    throw new Error(`Unexpected dependency ${name}`)
  }, mod, mod.exports)
  return mod.exports
}
const assistant = load('../src/api/assistant.ts'), admin = load('../src/api/admin.ts')
for (const create of [assistant.createAssistant, admin.createAdmin]) {
  await create({ username: ' helper123 ', password: 'test-only-password', display_name: '中文昵称' })
  assert.equal(requests.at(-1).body.username, 'helper123')
  assert.equal(requests.at(-1).body.display_name, '中文昵称')
  for (const username of ['一路向南', 'user_name', 'user-name', 'user name', 'ａｂｃ', 'ab', 'a'.repeat(33)]) {
    const before = requests.length
    assert.throws(() => create({ username, password: 'test-only-password', display_name: '中文' }), /英文字母或数字/)
    assert.equal(requests.length, before)
  }
}
for (const update of [assistant.updateAssistantProfile, admin.updateAdminProfile]) {
  await update(7, { username: ' new123 ', display_name: '中文昵称' })
  assert.equal(requests.at(-1).body.username, 'new123')
  assert.match(requests.at(-1).path, /\/7\/profile$/)
  await update(7, { display_name: '只改昵称' })
  assert.equal(Object.hasOwn(requests.at(-1).body, 'username'), false)
  assert.throws(() => update(7, { username: '中文账号', display_name: '中文昵称' }), /英文字母或数字/)
}

function extractHandler(file, name) {
  const view = readFileSync(new URL('../src/' + file, import.meta.url), 'utf8')
  const script = view.match(/<script[^>]*>([\s\S]*?)<\/script>/)[1]
  const ast = ts.createSourceFile(file, script, ts.ScriptTarget.Latest, true)
  for (const stmt of ast.statements) {
    if (ts.isFunctionDeclaration(stmt) && stmt.name?.text === name) return stmt.getText(ast)
    if (ts.isVariableStatement(stmt) && stmt.declarationList.declarations.some(d => d.name.getText(ast) === name)) return stmt.getText(ast)
  }
  throw new Error(`Missing ${name}`)
}
for (const [file, name, apiName] of [['TOTPBindDialog.vue', 'handleBind', 'bindTOTP'], ['TOTPVerifyDialog.vue', 'handleVerify', 'verifyTOTP']]) {
  const props = { visible: true, tempToken: 'test-challenge-a' }
  const code = { value: '026 988' }, loading = { value: false }, emitted = [], calls = []
  let resolve
  const api = (token, value) => { calls.push({ token, value }); return new Promise(r => { resolve = r }) }
  const message = { error: () => {}, warning: () => {} }
  const source = ts.transpileModule(extractHandler('components/' + file, name), { compilerOptions: { target: ts.ScriptTarget.ES2020 } }).outputText
  const fn = new Function('props', 'code', 'loading', 'useBackup', 'backupCode', apiName, 'emit', 'ElMessage', source + `;return ${name}`)(props, code, loading, { value: false }, { value: '' }, api, (...args) => emitted.push(args), message)
  const pending = fn()
  await fn()
  assert.equal(calls.length, 1, 'duplicate submit must not dispatch')
  assert.equal(apiName === 'bindTOTP' ? calls[0].value : calls[0].value.code, '026988')
  props.tempToken = 'test-challenge-b'
  resolve({ data: { data: { token: 'fixture-token', user: {}, backup_codes: [] } } })
  await pending
  assert.equal(emitted.length, 0, 'old challenge response must not authenticate the new dialog')
}
for (const view of ['AssistantsView.vue', 'AdminsView.vue']) {
  const text = readFileSync(new URL('../src/views/' + view, import.meta.url), 'utf8')
  assert.doesNotMatch(text, /v-if="row.totpEnabled"[^>]*resetTotp/)
  assert.match(text, /action: 'resetTotp'/)
  assert.match(text, /action: 'delete'/)
  assert.match(text, /v-model="profileForm.username"/)
  assert.match(text, /max-height="60vh"/)
}
console.log('PASS account validation, Unicode nicknames, rename compatibility, leading-zero codes, duplicate submit and stale challenge responses, reset/delete menu visibility.')
