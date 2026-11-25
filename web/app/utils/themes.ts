// The themes to pick from: the Nuxt UI colors each sets, palettes of main.css. The choice stays on this device.
const own = (id: string) => ({ id, primary: `jat-${id}`, neutral: `jat-${id}-gray` });

export const themes = [own('red'), own('purple'), own('blue'), own('green')];

export type Theme = (typeof themes)[number];

export const useTheme = () => useLocalStorage('jat-theme', themes[0]!.id);
