let integration: Promise<typeof import('@trtc/calls-uikit-vue')> | null = null;
export function prepareCallIntegration() {
  if (!integration) integration = import('@trtc/calls-uikit-vue').then(module => { void module.TUICallKitAPI; return module; }).catch(error => { integration = null; throw error; });
  return integration;
}
