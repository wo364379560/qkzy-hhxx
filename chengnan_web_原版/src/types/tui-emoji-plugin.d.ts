declare module '@tencentcloud/tui-emoji-plugin' {
  import type { DefineComponent } from 'vue';

  export const TUIEmojiPlugin: DefineComponent<
    Record<string, unknown>,
    Record<string, unknown>,
    any
  >;

  export default TUIEmojiPlugin;
}
