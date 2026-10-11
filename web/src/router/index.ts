import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', name: 'home', component: () => import('../views/Home.vue') },
  { path: '/login', name: 'login', component: () => import('../views/Login.vue') },
  { path: '/:owner/:repo', name: 'repo', component: () => import('../views/Repo.vue') },
]

export default createRouter({
  history: createWebHistory(),
  routes,
})
