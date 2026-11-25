// A page of the pro plan: in base it sends to the list.
export default defineNuxtRouteMiddleware(async () => {
  if ((await loadPreferences())?.plan !== 'pro') return navigateTo('/', { replace: true });
});
