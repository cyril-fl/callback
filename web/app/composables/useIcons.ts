/** Every icon of the app, by what it stands for: the one place to change an icon everywhere. */
export function useIcons() {
  const icons = {
    // Applications
    add: 'i-ph-plus',
    edit: 'i-ph-pencil-simple',
    delete: 'i-ph-trash',
    trash: 'i-ph-trash',
    restore: 'i-ph-arrow-u-up-left',
    reset: 'i-ph-arrow-counter-clockwise',
    empty: 'i-ph-tray',
    followUp: 'i-ph-bell-ringing',
    location: 'i-ph-map-pin',
    fullAddress: 'i-ph-map-pin-plus',
    contact: 'i-ph-user-plus',
    mail: 'i-ph-envelope-simple',
    website: 'i-ph-arrow-square-out',
    calendar: 'i-ph-calendar-blank',
    rating: 'i-ph-star-fill',
    ratingEmpty: 'i-ph-star',
    rejected: 'i-ph-x-circle',
    noReply: 'i-ph-hourglass',

    // Sources
    spontaneous: 'i-ph-paper-plane-tilt',
    jobPost: 'i-ph-briefcase',
    recruiter: 'i-ph-user',
    otherSource: 'i-ph-tag',

    // Navigation and views
    applications: 'i-ph-list-bullets',
    dashboard: 'i-ph-chart-bar',
    settings: 'i-ph-gear-six',
    table: 'i-ph-table',
    kanban: 'i-ph-kanban',
    search: 'i-ph-magnifying-glass',
    chevronDown: 'i-ph-caret-down',
    sort: 'i-ph-arrows-down-up',
    ascending: 'i-ph-sort-ascending',
    descending: 'i-ph-sort-descending',

    // Settings
    lightMode: 'i-ph-sun',
    darkMode: 'i-ph-moon',
    systemMode: 'i-ph-desktop',
    apiToken: 'i-ph-key',
    copy: 'i-ph-copy',
    export: 'i-ph-download-simple',
    import: 'i-ph-upload-simple',
    logout: 'i-ph-sign-out',
  };

  return { icons };
}
