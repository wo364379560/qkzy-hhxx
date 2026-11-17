import { genTestUserSig } from './debug';
import { TUIChatKit } from './chat-kit';
import TUIComponents, {
  TUIChat,
  TUIConversation,
  TUIContact,
  TUISearch,
  TUIGroup,
} from './components';
import TUIKit from './index.vue';
import { hideTUIChatFeatures } from './components/TUIChat/config';

export {
  TUIKit,
  TUIChatKit,
  TUIComponents,
  TUIChat,
  TUIConversation,
  TUIContact,
  TUISearch,
  TUIGroup,
  hideTUIChatFeatures,
  genTestUserSig,
};
