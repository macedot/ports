import { createApp } from 'vue'
import { createPinia } from 'pinia'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import App from './App.vue'
import './style.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.mount('#app')