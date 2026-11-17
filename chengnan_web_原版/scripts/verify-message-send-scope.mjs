import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";
import ts from "typescript";
const source = readFileSync(
  new URL(
    "../src/TUIKit/components/TUIChat/utils/messageSendScope.ts",
    import.meta.url
  ),
  "utf8"
);
const exports = {};
let owner = "A";
const watchers = [];
vm.runInNewContext(
  ts.transpileModule(source, {
    compilerOptions: {
      target: ts.ScriptTarget.ES2020,
      module: ts.ModuleKind.CommonJS,
    },
  }).outputText,
  {
    exports,
    require: () => ({
      StoreName: { USER: "user" },
      TUIStore: {
        getData: () => ({ userID: owner }),
        watch: (_store, options) => watchers.push(options.userProfile),
      },
    }),
  }
);
const oldComposerScope = exports.captureMessageSendScope();
assert.equal(oldComposerScope(), true);
const changeOwner = (value) => {
  owner = value;
  watchers.forEach((callback) => callback());
};
// No calls to the retained job while the composer is unmounted.
changeOwner("B");
changeOwner("A");
assert.equal(
  oldComposerScope(),
  false,
  "A -> B -> A invalidates old queued sends even during composer unmount"
);
const newComposerScope = exports.captureMessageSendScope();
assert.equal(newComposerScope(), true);
assert.equal(
  watchers.length,
  1,
  "all composers share one lifetime account watcher"
);
changeOwner("");
changeOwner("A");
assert.equal(
  newComposerScope(),
  false,
  "logout and same-account re-login invalidates old jobs"
);
console.log(
  "PASS: retained message jobs observe account generation across unmount and same-user re-login."
);
