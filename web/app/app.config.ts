export default defineAppConfig({
  // The default theme, lilac; app.vue applies the one picked in the settings.
  ui: {
    colors: {
      primary: 'callback-lilac',
      neutral: 'callback-lilac-gray',
      info: 'callback-lilac-info',
      success: 'callback-lilac-success',
      warning: 'callback-lilac-warning',
      error: 'callback-lilac-error',
    },
    // Badges are pastel pills.
    badge: { slots: { base: 'rounded-full' } },
    // Pastel solid buttons in light mode (dark mode's 400 already is): the 200 with dark text reads above 4.5:1.
    button: {
      compoundVariants: [
        {
          color: 'primary',
          variant: 'solid',
          class: 'light:bg-primary-200 light:text-primary-900 light:hover:bg-primary-300 light:active:bg-primary-300',
        },
      ],
    },
  },
});
