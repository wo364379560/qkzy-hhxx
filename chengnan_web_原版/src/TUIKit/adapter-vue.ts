import * as Vue from 'vue';
import { TUIGlobal } from '@tencentcloud/universal-api';

let vueVersion: number;
let framework = 'vue2';
const adapterVueBuildVersion = '20260824-cache-bust-1';
let createVNode = (
  arg1: any,
  arg2: any,
): { component: any; props: any; data: any } => {
  return {} as { component: any; props: any; data: any };
};
let render = (arg1: any, arg2: any) => {
  return;
};

try {
  if (
    (Vue as any)?.default?.version
    && (Vue as any)?.default?.version?.startsWith('2.7.')
  ) {
    // >= Vue 2.7.0
    vueVersion = 2.7;
    TUIGlobal.Vue = (Vue as any)?.getCurrentInstance()?.appContext?.app;
  } else if (
    (Vue as any)?.default?.version
    && (Vue as any)?.default?.version?.startsWith('2.')
  ) {
    // < Vue 2.7.0
    vueVersion = 2;
    TUIGlobal.Vue = (Vue as any).default;
  } else {
    // >= Vue 3.0.0
    vueVersion = 3;
    framework = 'vue3';
    createVNode = (Vue as any)?.createVNode;
    render = (Vue as any)?.render;
    TUIGlobal.Vue = (Vue as any)?.getCurrentInstance()?.appContext?.app;
    // exportedAPIOrigin = Vue;
  }
} catch (error: any) {
  // >= Vue 3.0.0
  vueVersion = 3;
  framework = 'vue3';
  createVNode = (Vue as any)?.createVNode;
  render = (Vue as any)?.render;
  TUIGlobal.Vue = (Vue as any)?.getCurrentInstance()?.appContext?.app;
}
console.warn(`[adapter-vue]: vue version is ${vueVersion}, build ${adapterVueBuildVersion}`);
export { vueVersion, framework, render, createVNode, adapterVueBuildVersion };

export * from 'vue';
