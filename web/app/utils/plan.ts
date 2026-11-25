// The mode of the app, as tide's plans: base keeps the stats above the list, pro gives them a dashboard page of their own.
// shortcut: stays on this device like the theme, move it to the API's settings if it must follow the account.
export const plans = ['base', 'pro'] as const;

export type Plan = (typeof plans)[number];

export const usePlan = () => useLocalStorage<Plan>('callback-plan', 'base');
