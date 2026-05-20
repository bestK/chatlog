import { createApp } from 'vue';
import { Markdown as VueStreamMarkdown } from 'vue-stream-markdown';
import App from './App.vue';
import './styles.css';

createApp(App).component('VueStreamMarkdown', VueStreamMarkdown).mount('#app');
