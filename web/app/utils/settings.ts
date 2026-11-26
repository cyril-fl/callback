/** The sections of the settings, in order: a tab and a page each, at /settings/<id>. */
export const settingsSections = [
  { id: 'general', icon: 'settings' },
  { id: 'appearance', icon: 'appearance' },
  { id: 'data', icon: 'data' },
  { id: 'tokens', icon: 'apiToken' },
  // Only for an admin, in the multi mode.
  { id: 'accounts', icon: 'accounts' },
] as const;

export type SettingsSection = (typeof settingsSections)[number]['id'];
