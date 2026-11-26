import type { PreferenceChanges } from '~/types/items';

// The themes to pick from: the Nuxt UI colors each sets, palettes of main.css. The choice is a preference of the account.
const own = (id: string) => ({
  id,
  primary: `callback-${id}`,
  neutral: `callback-${id}-gray`,
  info: `callback-${id}-info`,
  success: `callback-${id}-success`,
  warning: `callback-${id}-warning`,
  error: `callback-${id}-error`,
});

export const themes = [own('lilac'), own('pink'), own('blue'), own('green')];

export type Theme = (typeof themes)[number];

export function useTheme() {
  const { preferences, save } = usePreferences();
  return computed({
    get: () => preferences.value?.theme ?? themes[0]!.id,
    set: (theme: string) => save({ theme: theme as NonNullable<PreferenceChanges['theme']> }),
  });
}
