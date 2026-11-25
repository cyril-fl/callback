export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  // Static client: `nuxt generate` is served by jat-server on the API's origin.
  ssr: false,
  modules: ['@nuxt/eslint', '@nuxt/ui', '@nuxt/fonts', '@vueuse/nuxt', '@nuxtjs/i18n'],
  css: ['~/assets/css/main.css'],
  i18n: {
    strategy: 'no_prefix',
    defaultLocale: 'fr',
    locales: [{ code: 'fr', file: 'fr.json' }],
  },
  runtimeConfig: {
    // '' in production (same origin); NUXT_PUBLIC_API_BASE overrides.
    public: { apiBase: import.meta.dev ? 'http://localhost:8080' : '' },
  },
});
