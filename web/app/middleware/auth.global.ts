// Without a session every page but the public ones sends to /login; with one, they send home. The preferences come with it.
const publicPaths = ['/login', '/signup'];

export default defineNuxtRouteMiddleware(async (to) => {
  const me = await loadMe();
  const isPublic = publicPaths.includes(to.path);
  if (!me && !isPublic) return navigateTo({ path: '/login', query: { redirect: to.fullPath } });
  if (me && isPublic) return navigateTo('/');
  if (me) await loadPreferences();
});
