// The themes to pick from: the Nuxt UI colors each sets, palettes of main.css. The choice stays on this device.
const own = (id: string) => ({ id, primary: `callback-${id}`, neutral: `callback-${id}-gray` });

export const themes = [own('pink'), own('lilac'), own('blue'), own('green')];

export type Theme = (typeof themes)[number];

export const useTheme = () => useLocalStorage('callback-theme', themes[0]!.id);
