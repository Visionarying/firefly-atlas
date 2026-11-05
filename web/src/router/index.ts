import { createRouter, createWebHashHistory } from 'vue-router'
import { getToken } from '@/api/request'
import Home from '@/views/Home.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'home', component: Home, meta: { title: '主页' } },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/Login.vue'),
      meta: { title: '登录' },
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · Firefly Panel` : 'Firefly Panel'
  if (to.name === 'login' && getToken()) {
    return { name: 'home' }
  }
  return true
})

export default router
