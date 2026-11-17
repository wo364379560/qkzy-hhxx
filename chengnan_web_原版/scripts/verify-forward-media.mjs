import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import { randomUUID } from 'node:crypto';
import ts from 'typescript';

// The fixture contracts are independent of the mapper implementation:
// Web SDK ImagePayload (0 original, 1 198px, 2 720px):
// https://web.sdk.qcloud.com/im/doc/v3/zh-cn/Message.html
// REST image Type (1 original, 2 large, 3 thumbnail), media UUID and DownloadFlag:
// https://cloud.tencent.com/document/product/269/2720
// SDK runtime rich-media-message.js also exposes thumbUUID/Size/Format plus
// snapshot aliases; its d.ts intentionally types Message.payload as any.
const source = fs.readFileSync(new URL('../src/TUIKit/components/TUIChat/forward/serverForward.ts', import.meta.url), 'utf8');
const code = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2021, module: ts.ModuleKind.CommonJS } }).outputText;
const context = vm.createContext({ exports: {}, require: () => ({}), URL, TextEncoder, crypto: { randomUUID } });
vm.runInContext(code, context);
const { toForwardMessage, supportsServerMessages, createServerForwardBatch } = context.exports;
const plain = value => JSON.parse(JSON.stringify(value));
const clone = plain;
const message = (type, payload) => ({ type, payload });
const convert = value => { const mapped = toForwardMessage(value); return mapped && plain(mapped); };
const frozen = value => { if (value && typeof value === 'object') { Object.values(value).forEach(frozen); Object.freeze(value); } return value; };

const image = message('TIMImageElem', {
  uuid: 'review-image-asset', imageFormat: 3,
  imageInfoArray: [
    { type: 0, size: 1853095, width: 2448, height: 3264, url: 'https://media.example.test/image-original.png' },
    { type: 1, size: 12535, width: 198, height: 264, url: 'https://media.example.test/image-thumb.png' },
    { type: 2, size: 256524, width: 720, height: 960, url: 'https://media.example.test/image-large.png' },
  ],
});
const imageREST = { UUID: 'review-image-asset', ImageFormat: 3, ImageInfoArray: [
  { Type: 1, Size: 1853095, Width: 2448, Height: 3264, URL: 'https://media.example.test/image-original.png' },
  { Type: 2, Size: 256524, Width: 720, Height: 960, URL: 'https://media.example.test/image-large.png' },
  { Type: 3, Size: 12535, Width: 198, Height: 264, URL: 'https://media.example.test/image-thumb.png' },
] };
assert.deepEqual(convert(frozen(clone(image))), { body: [{ MsgType: 'TIMImageElem', MsgContent: imageREST }], preview: '[图片]' });
const reordered = clone(image); reordered.payload.imageInfoArray.reverse();
assert.deepEqual(convert(reordered).body[0].MsgContent, imageREST, 'variant identity must follow type rather than input position');
for (const format of [1, 2, 3, 4, 255]) { const input = clone(image); input.payload.imageFormat = format; assert.equal(convert(input).body[0].MsgContent.ImageFormat, format); }
for (const format of [undefined, '3', 'png', 0, 5, NaN]) { const input = clone(image); input.payload.imageFormat = format; assert.equal(convert(input), null, 'unknown format must use SDK handling'); }
for (const mutate of [
  p => delete p.uuid,
  p => { p.uuid = 44; },
  p => { p.imageInfoArray[0].type = 3; },
  p => { p.imageInfoArray[0].type = '0'; },
  p => { p.imageInfoArray[0].type = 1; },
  p => { p.imageInfoArray[0].size = -1; },
  p => { p.imageInfoArray[0].size = '1853095'; },
  p => { p.imageInfoArray[0].size = NaN; },
  p => delete p.imageInfoArray[0].size,
  p => delete p.imageInfoArray[0].width,
  p => { p.imageInfoArray[0].height = 1.2; },
  p => { p.imageInfoArray[0].url = 'https://'; },
  p => { p.imageInfoArray[0].url = 'blob:local'; },
  p => { p.imageInfoArray[0].url = 'https://name:password@media.example.test/a'; },
  p => { p.imageInfoArray = {}; },
]) { const input = clone(image); input.ID = 'message-id-is-not-a-resource-uuid'; mutate(input.payload); assert.equal(convert(input), null); }
// REST documentation permits zero dimensions on compressed variants; preserve
// explicitly supplied zero metadata instead of replacing it with guessed data.
const zeroImage = clone(image); Object.assign(zeroImage.payload.imageInfoArray[1], { width: 0, height: 0, size: 0 });
assert.equal(convert(zeroImage).body[0].MsgContent.ImageInfoArray[2].Width, 0);

const video = message('TIMVideoFileElem', {
  videoUUID: 'review-video-asset', videoUrl: 'https://media.example.test/movie.mp4', videoSize: 1194603, videoSecond: 5, videoFormat: 'mp4',
  thumbUUID: 'review-video-thumbnail', thumbUrl: 'https://media.example.test/movie.jpg', thumbSize: 13907, thumbWidth: 720, thumbHeight: 1280, thumbFormat: 'JPG',
});
const videoREST = { VideoUUID: 'review-video-asset', VideoUrl: 'https://media.example.test/movie.mp4', VideoSize: 1194603, VideoSecond: 5, VideoFormat: 'mp4', VideoDownloadFlag: 2,
  ThumbUUID: 'review-video-thumbnail', ThumbUrl: 'https://media.example.test/movie.jpg', ThumbSize: 13907, ThumbWidth: 720, ThumbHeight: 1280, ThumbFormat: 'JPG', ThumbDownloadFlag: 2 };
assert.deepEqual(convert(frozen(clone(video))), { body: [{ MsgType: 'TIMVideoFileElem', MsgContent: videoREST }], preview: '[视频]' });
const snapshotVideo = clone(video);
for (const suffix of ['Url', 'Size', 'Width', 'Height', 'Format']) { snapshotVideo.payload[`snapshot${suffix}`] = snapshotVideo.payload[`thumb${suffix}`]; delete snapshotVideo.payload[`thumb${suffix}`]; }
assert.deepEqual(convert(snapshotVideo).body[0].MsgContent, videoREST, 'snapshot aliases preserve actual thumbnail metadata');
const bothVideo = clone(video); bothVideo.payload.snapshotUrl = 'https://media.example.test/obsolete.jpg'; bothVideo.payload.snapshotSize = 1;
assert.deepEqual(convert(bothVideo).body[0].MsgContent, videoREST, 'canonical thumb fields take precedence');
const webVideo = clone(video); webVideo.payload.videoSecond = 0;
assert.equal(convert(webVideo).body[0].MsgContent.VideoSecond, 0, 'Web SDK documents a zero duration');
for (const field of ['videoUUID', 'thumbUUID', 'videoUrl', 'videoSize', 'videoSecond', 'thumbUrl', 'thumbSize', 'thumbWidth', 'thumbHeight']) {
  const input = clone(video); input.ID = 'message-id-not-video-uuid'; input.payload.uuid = 'unrelated-uuid'; delete input.payload[field];
  assert.equal(convert(input), null, `missing ${field} must use SDK, never manufactured identity/metadata`);
}
for (const field of ['videoSize', 'videoSecond', 'thumbSize', 'thumbWidth', 'thumbHeight']) {
  const input = clone(video); input.payload[field] = '5'; assert.equal(convert(input), null);
}

const text = message('TIMTextElem', { text: '【转发验收】图片前面的文字 😊' });
const file = message('TIMFileElem', { uuid: 'review-file-asset', fileUrl: 'https://media.example.test/document.tn6', fileName: '验收文档.tn6', fileSize: 800 });
const audio = message('TIMSoundElem', { uuid: 'review-audio-asset', url: 'https://media.example.test/voice.mp3', size: 9021, second: 3 });
const face = message('TIMFaceElem', { index: 1, data: 'smile' });
const location = message('TIMLocationElem', { description: '验收位置', latitude: 29.340656774469956, longitude: 116.77497920478824 });
const custom = message('TIMCustomElem', { data: 'opaque-custom-body', description: '验收自定义消息', extension: '' });
const card = message('TIMCustomElem', { data: JSON.stringify({ businessID: 'zhihuan_contact_card', version: 1, userId: 'synthetic-user', nickname: '验收名片', avatar: '' }), description: '[个人名片] 验收名片', extension: '个人名片' });
assert.deepEqual(convert(file), { body: [{ MsgType: 'TIMFileElem', MsgContent: { UUID: 'review-file-asset', Url: 'https://media.example.test/document.tn6', FileSize: 800, FileName: '验收文档.tn6', Download_Flag: 2 } }], preview: '[文件] 验收文档.tn6' });
assert.deepEqual(convert(audio).body[0].MsgContent, { UUID: 'review-audio-asset', Url: 'https://media.example.test/voice.mp3', Size: 9021, Second: 3, Download_Flag: 2 });
assert.deepEqual(convert(face).body[0].MsgContent, { Index: 1, Data: 'smile' });
assert.deepEqual(convert(location).body[0].MsgContent, { Desc: '验收位置', Latitude: 29.340656774469956, Longitude: 116.77497920478824 });
assert.deepEqual(convert(custom).body[0].MsgContent, { Data: 'opaque-custom-body', Desc: '验收自定义消息', Ext: '' });
assert.equal(convert(card).preview, '[名片] 验收名片');
for (const invalid of [
  message('TIMTextElem', { text: 1 }), message('TIMTextElem', { text: '' }), message('TIMTextElem', { text: '多'.repeat(4000) }),
  message('TIMFaceElem', { index: '1', data: 'smile' }), message('TIMFaceElem', {}), message('TIMFaceElem', { index: 1, data: {} }),
  message('TIMLocationElem', { description: 'bad', latitude: '29', longitude: 110 }), message('TIMLocationElem', { description: 'bad', latitude: NaN, longitude: 110 }),
  message('TIMLocationElem', { description: 'bad', latitude: 91, longitude: 110 }), message('TIMCustomElem', { data: {} }), message('TIMCustomElem', { data: 'valid', extension: 2 }),
  { ...file, ID: 'not-a-file-uuid', payload: { ...file.payload, uuid: undefined } }, { ...file, payload: { ...file.payload, fileSize: -1 } },
  { ...audio, ID: 'not-an-audio-uuid', payload: { ...audio.payload, uuid: undefined } }, { ...audio, payload: { ...audio.payload, second: '3' } },
]) assert.equal(convert(invalid), null);
assert.equal(convert(message('TIMFaceElem', { data: 'smile' })).preview, '[表情]');
assert.equal(convert(message('TIMFaceElem', { index: 0 })).body[0].MsgContent.Index, 0);
assert.equal(convert({ ...file, payload: { ...file.payload, fileSize: 0 } }).body[0].MsgContent.FileSize, 0);

const mixed = [text, image, video, file, audio, face, location, custom, card];
assert.equal(supportsServerMessages(mixed, false), true);
assert.equal(supportsServerMessages(mixed, true), false, 'merged records retain SDK semantics');
assert.equal(supportsServerMessages([...mixed, { type: 'TIMVideoFileElem', payload: { videoUUID: 'partial' } }], false), false);
const batch = createServerForwardBatch([{ conversationID: 'GROUPsynthetic' }], mixed, () => '验收群');
assert.deepEqual(plain(batch.serverPlan.messages).map(m => m.body[0].MsgType), mixed.map(m => m.type), 'text/media selection order and all messages must survive conversion');
assert.deepEqual(plain(batch.serverPlan.messages).map(m => m.preview), [text.payload.text, '[图片]', '[视频]', '[文件] 验收文档.tn6', '[语音]', '[表情]', '[位置] 验收位置', '[自定义消息]', '[名片] 验收名片']);

// This is the installed SDK's actual shape: payload/type reflect _elements[0],
// while getElements()/transformElementsToServerFormat retain the full message.
const composite = {
  ...clone(text),
  _elements: [{ type: text.type, content: clone(text.payload) }, { type: image.type, content: clone(image.payload) }],
  getElements() { return this._elements; },
};
const compositeBefore = plain(composite);
assert.equal(convert(composite), null, 'a single text+image SDK message must not become just its text');
assert.equal(supportsServerMessages([text, composite, image], false), false, 'one composite routes the whole selected list through SDK without filtering it out');
assert.deepEqual(plain(composite), compositeBefore, 'fallback must preserve all original elements');
assert.equal(convert({ ...composite, getElements: undefined }), null);
assert.equal(convert({ ...text, getElements() { throw Error('SDK data temporarily unavailable'); } }), null);
assert.equal(convert({ ...text, getElements() { return undefined; } }), null);
for (const elements of [[], {}, null, undefined, 'not-elements', [null], [[]]]) {
  assert.equal(convert({ ...text, _elements: elements }), null);
  assert.equal(convert({ ...text, getElements() { return elements; } }), null);
}
const single = { ...clone(image), _elements: [{ type: image.type, content: clone(image.payload) }], getElements() { return this._elements; } };
assert.deepEqual(convert(single).body[0].MsgContent, imageREST, 'ordinary SDK media with one element remains eligible');
assert.equal(convert({ ...composite, getElements() { return [this._elements[0]]; } }), null, 'conflicting private/public element counts must not drop data');

if (process.argv.includes('--write-fixtures')) {
  fs.writeFileSync('/tmp/forward-media-fixed-fixtures.json', JSON.stringify({
    purpose: 'Synthetic SDK inputs and actual toForwardMessage output; reserved .example.test URLs, no credentials or user data',
    sdk_sources: ['https://web.sdk.qcloud.com/im/doc/v3/zh-cn/Message.html', 'https://cloud.tencent.com/document/product/269/2720'],
    cases: mixed.map(input => ({ sdk: input, converted: convert(input) })),
  }, null, 2));
}
console.log('PASS: official SDK/REST image enum and original/large/thumbnail order; preserved resource identities; canonical/snapshot video metadata; duration zero; strict media/type validation; mixed text/media ordering; composite SDK message preservation; readable previews; incomplete records stay on SDK.');
