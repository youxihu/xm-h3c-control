import { createRouter, createWebHistory } from 'vue-router'
import PortManagement from '../views/PortManagement.vue'

const routes = [
  {
    path: '/',
    name: 'PortManagement',
    component: PortManagement
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router