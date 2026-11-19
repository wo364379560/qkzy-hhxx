import { defineAsyncComponent } from 'vue';
import { prepareCallIntegration } from '@/bootstrap/callRuntime';

// Every CallKit entry point uses the same guarded vendor import.
export default defineAsyncComponent(async () => (await prepareCallIntegration()).TUICallKit);
