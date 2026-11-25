// Without a session every page but /login sends to /login; with one, /login sends home.
export default defineNuxtRouteMiddleware(async (to) => {
  const me = await loadMe();
  if (!me && to.path !== '/login') return navigateTo({ path: '/login', query: { redirect: to.fullPath } });
  if (me && to.path === '/login') return navigateTo('/');
});
