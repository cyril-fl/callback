/** Every icon of the app, by what it stands for: the one place to change an icon everywhere. */
export function useIcons() {
  const icons = {
    // Applications
    add: 'i-lucide-plus',
    edit: 'i-lucide-pencil',
    delete: 'i-lucide-trash-2',
    more: 'i-lucide-ellipsis',
    trash: 'i-lucide-trash-2',
    restore: 'i-lucide-undo-2',
    reset: 'i-lucide-rotate-ccw',
    empty: 'i-lucide-inbox',
    followUp: 'i-lucide-bell-ring',
    location: 'i-lucide-map-pin',
    fullAddress: 'i-lucide-map-pin-plus',
    contact: 'i-lucide-user-plus',
    person: 'i-lucide-contact',
    interview: 'i-lucide-calendar-clock',
    comment: 'i-lucide-message-square-text',
    followedUp: 'i-lucide-bell',
    mail: 'i-lucide-mail',
    website: 'i-lucide-external-link',
    calendar: 'i-lucide-calendar',
    rating: 'i-callback-star-filled',
    ratingEmpty: 'i-lucide-star',
    rejected: 'i-lucide-circle-x',
    noReply: 'i-lucide-hourglass',

    // Sources
    spontaneous: 'i-lucide-send',
    jobPost: 'i-lucide-briefcase',
    recruiter: 'i-lucide-user-round',
    otherSource: 'i-lucide-tag',

    // Navigation and views
    applications: 'i-lucide-list',
    dashboard: 'i-lucide-chart-column',
    settings: 'i-lucide-settings',
    table: 'i-lucide-table',
    kanban: 'i-lucide-columns-3',
    search: 'i-lucide-search',
    chevronDown: 'i-lucide-chevron-down',
    sort: 'i-lucide-arrow-up-down',
    ascending: 'i-lucide-arrow-up-narrow-wide',
    descending: 'i-lucide-arrow-down-wide-narrow',

    // Settings
    lightMode: 'i-lucide-sun',
    darkMode: 'i-lucide-moon',
    appearance: 'i-lucide-palette',
    data: 'i-lucide-database',
    systemMode: 'i-lucide-monitor',
    apiToken: 'i-lucide-key-round',
    copy: 'i-lucide-copy',
    export: 'i-lucide-download',
    import: 'i-lucide-upload',
    logout: 'i-lucide-log-out',
    accounts: 'i-lucide-users',
    resetPassword: 'i-lucide-rotate-ccw-key',
    invite: 'i-lucide-mail-plus',
  };

  return { icons };
}
