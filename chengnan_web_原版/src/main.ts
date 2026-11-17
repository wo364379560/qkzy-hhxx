import { createApp } from 'vue';
import App from './App.vue';
import router from './router';
import { warmUpChatAPIConnection } from './config/runtime';

warmUpChatAPIConnection();

const app = createApp(App);
app.use(router);

app.mount('#app');
