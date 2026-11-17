import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import ts from 'typescript'

const source = readFileSync(new URL('../src/api/assistant.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 },
}).outputText
const requests = []
const response = { data: { code: 0, data: { id: 42 } } }
const request = { post: async (path, body) => { requests.push({ path, body }); return response } }
const module = { exports: {} }
new Function('require', 'module', 'exports', compiled)(name => {
  if (name === '../utils/adminAccount') {
    const helper = { exports: {} }
    const js = ts.transpileModule(readFileSync(new URL('../src/utils/adminAccount.ts', import.meta.url), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText
    new Function('module', 'exports', js)(helper, helper.exports)
    return helper.exports
  }
  assert.equal(name, './request')
  return { default: request }
}, module, module.exports)
const { createAssistant, normalizeCreateAssistantParams } = module.exports
const password = 'test-only-example-password'
assert.equal(await createAssistant({ username: '1242', password }), response)
assert.deepEqual(requests.pop(), { path: '/assistants', body: { username: '1242', password, display_name: '1242' } })
for (const name of ['', '  ', undefined]) {
  await createAssistant({ username: '  helper  ', password, display_name: name })
  assert.deepEqual(requests.pop().body, { username: 'helper', password, display_name: 'helper' })
}
await createAssistant({ username: 'helper', password, display_name: ' 睿助理 ' })
assert.equal(requests.pop().body.display_name, '睿助理')
assert.equal(normalizeCreateAssistantParams({ username: 'helper', password, display_name: '睿'.repeat(64) }).display_name.length, 64)
for (const params of [
  { username: 'ab', password }, { username: '   ', password },
  { username: 'u'.repeat(33), password }, { username: 'helper', password: 'short' },
  { username: 'helper', password, display_name: '睿'.repeat(65) },
]) {
  const before = requests.length
  assert.throws(() => createAssistant(params))
  assert.equal(requests.length, before, 'invalid input must not be dispatched')
}
console.log('PASS actual assistant API: legacy payload, default nickname, trimming, explicit nickname, Unicode limit and rejected invalid requests.')

// Execute the actual Vue submit function with only the UI/network boundaries stubbed.
const view = readFileSync(new URL('../src/views/AssistantsView.vue', import.meta.url), 'utf8')
const script = view.match(/<script[^>]*>([\s\S]*?)<\/script>/)[1]
const ast = ts.createSourceFile('AssistantsView.ts', script, ts.ScriptTarget.Latest, true)
const handler = ast.statements.find(node => ts.isFunctionDeclaration(node) && node.name?.text === 'handleCreate')
assert.ok(handler)
assert.match(view, /昵称（选填）/)
assert.match(view, /不填则使用用户名/)
const code = ts.transpileModule(handler.getText(ast), { compilerOptions: { target: ts.ScriptTarget.ES2020 } }).outputText
const form = { username: '1242', password, displayName: '' }
const loading = { value: false }, modal = { value: true }, messages = [], calls = []
const message = { warning: text => messages.push(text), error: text => messages.push(text), success: text => messages.push(text) }
const submit = new Function('createForm', 'submitting', 'showCreateModal', 'ElMessage', 'createAssistant', 'fetchAssistants', code + '; return handleCreate')(
  form, loading, modal, message, async params => {
    calls.push(normalizeCreateAssistantParams(params)); return response
  }, () => {},
)
await submit()
assert.equal(calls.length, 1)
assert.equal(calls[0].display_name, '1242')
assert.equal(modal.value, false)
assert.equal(loading.value, false)
assert.ok(messages.includes('助理已创建'))
assert.equal(form.password, '')
form.username = 'helper'; form.password = password; form.displayName = '测试昵称'; modal.value = true
await submit()
assert.equal(calls.at(-1).display_name, '测试昵称')
form.username = ' '; form.password = password
await submit()
assert.equal(calls.length, 2)
assert.ok(messages.includes('请填写用户名和密码'))
console.log('PASS actual Vue submit: missing nickname succeeds, explicit nickname preserved, blank account blocked, success resets form and loading state.')
