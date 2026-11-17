// @ts-nocheck
import { Marked } from 'marked';
import { markedHighlight } from 'marked-highlight';
import hljs from 'highlight.js/lib/core';
import bash from 'highlight.js/lib/languages/bash';
import css from 'highlight.js/lib/languages/css';
import dart from 'highlight.js/lib/languages/dart';
import go from 'highlight.js/lib/languages/go';
import java from 'highlight.js/lib/languages/java';
import javascript from 'highlight.js/lib/languages/javascript';
import json from 'highlight.js/lib/languages/json';
import markdown from 'highlight.js/lib/languages/markdown';
import plaintext from 'highlight.js/lib/languages/plaintext';
import python from 'highlight.js/lib/languages/python';
import sql from 'highlight.js/lib/languages/sql';
import typescript from 'highlight.js/lib/languages/typescript';
import xml from 'highlight.js/lib/languages/xml';
import DOMPurify from 'dompurify';
import { TUITranslateService } from '@tencentcloud/chat-uikit-engine-lite';

[
  ['bash', bash],
  ['css', css],
  ['dart', dart],
  ['go', go],
  ['java', java],
  ['javascript', javascript],
  ['json', json],
  ['markdown', markdown],
  ['plaintext', plaintext],
  ['python', python],
  ['sql', sql],
  ['typescript', typescript],
  ['xml', xml],
].forEach(([name, language]) => {
  hljs.registerLanguage(name, language as any);
});

const escapeTest = /[&<>"']/;
const escapeReplace = new RegExp(escapeTest.source, 'g');
const escapeTestNoEncode = /[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/;
const escapeReplaceNoEncode = new RegExp(escapeTestNoEncode.source, 'g');
type Iescape = {
  [key: string]: string;
};
const escapeReplacements: Iescape = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  '\'': '&#39;',
};
const getEscapeReplacement = (ch: string): string => escapeReplacements[ch as keyof Iescape];

export const marked = new Marked(
  {
    mangle: false,
    headerIds: false,
  },
  markedHighlight({
    highlight(code: string) {
      return hljs.highlightAuto(code).value;
    },
  }),
  {
    renderer: {
      code(this: any, code: string, language: string | undefined, isEscaped: boolean) {
        const lang = language || '';
        return `
          <pre class="hljs message-marked_code-container">
            <div class="message-marked_code-header">
              <span class="message-marked_language-txt">${escape(lang) || ''}</span>
              <span class="message-marked_copy-btn">${TUITranslateService.t('TUIChat.复制')}</span>
            </div>
            <code class="hljs message-marked_code-content language-${escape(lang)}" >${isEscaped ? code : escape(code)}</code>
          </pre>
        `;
      },
      image(this: any, href: string | null, title: string | null, text: string) {
        return `<img class="message-marked_image message-image" src="${href}" alt="${text}" title="${title}" />`;
      },
      link(this: any, href: string | null, title: string | null, text: string) {
        return `<a target="_blank" rel="noopener noreferrer" class="message-marked_link" href="${href}" title="${title}">${text}</a>`;
      },
    },
  },
);

export const markedWithPurify = (text: string) => {
  const parsedContent = marked.parse(text);
  return DOMPurify.sanitize(parsedContent, { ADD_ATTR: ['target'] });
};

function escape(html: string, encode: boolean = false) {
  if (encode) {
    if (escapeTest.test(html)) {
      return html.replace(escapeReplace, getEscapeReplacement);
    }
  } else if (escapeTestNoEncode.test(html)) {
    return html.replace(escapeReplaceNoEncode, getEscapeReplacement);
  }
  return html;
}
