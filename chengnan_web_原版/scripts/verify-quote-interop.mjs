import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

const file = new URL('../src/TUIKit/components/TUIChat/message-list/message-elements/message-quote/quote-protocol.ts', import.meta.url);
const source = ts.transpileModule(readFileSync(file, 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.ES2020, target: ts.ScriptTarget.ES2020 },
}).outputText;
const { decodeMessageQuote, encodeMessageQuote } = await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`);

const native = { msgID: 'app-original', msgSender: '群昵称', msgAbstract: 'App 原消息 😀' };
const web = { messageID: 'web-original', messageSender: 'Web', messageAbstract: '引用回复', messageType: 1 };
assert.equal(decodeMessageQuote(JSON.stringify({ atomicx_quote_v1: native })).messageAbstract, native.msgAbstract);
assert.equal(decodeMessageQuote(JSON.stringify({ messageReply: web })).messageID, web.messageID);
assert.equal(decodeMessageQuote(JSON.stringify({ messageReply: {}, atomicx_quote_v1: native })).messageID, native.msgID);
assert.equal(decodeMessageQuote(JSON.stringify({ messageReply: web, atomicx_quote_v1: native })).messageID, web.messageID);
assert.equal(decodeMessageQuote(JSON.stringify({ messageReply: { ...web, messageType: '3' } })).messageType, 3);
for (const bad of [undefined, '', 'not-json', '[]', 'null', '{}', '{"messageReply":{"messageID":null}}']) {
  assert.equal(decodeMessageQuote(bad), undefined);
}
console.log('Quote interoperability: all checks passed');

const encoded = JSON.parse(encodeMessageQuote({ ID: 'web-new', type: 'TIMTextElem', nameCard: '群名片', from: 'sender', time: 1700000000, sequence: 42, payload: { text: '原消息 😀\n第二行' } }));
assert.equal(encoded.atomicx_quote_v1.msgID, 'web-new');
assert.equal(encoded.atomicx_quote_v1.msgAbstract, '原消息 😀\n第二行');
assert.equal(encoded.messageReply.messageTime, 1700000000);
assert.equal(encoded.messageReply.messageSequence, 42);
assert.equal(encoded.messageReply.messageSender, '群名片');
for (const key of ['messageReply', 'atomicx_quote_v1']) {
  const decoded = decodeMessageQuote(JSON.stringify({ [key]: encoded[key] }));
  assert.equal(decoded.messageID, 'web-new');
  assert.equal(decoded.messageAbstract, '原消息 😀\n第二行');
}
const image = JSON.parse(encodeMessageQuote({ ID: 'image', type: 'TIMImageElem', time: 123 }));
assert.equal(image.messageReply.messageType, 3);
assert.equal(image.atomicx_quote_v1.msgAbstract, '[图片]');
console.log('Outgoing web-to-native quotes: all checks passed');
