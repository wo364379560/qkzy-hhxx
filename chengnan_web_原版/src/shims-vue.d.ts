declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, any>
  export default component
}

declare module "@vue/composition-api";

declare module '*.svg' {
  const content: any;
  export default content;
}
