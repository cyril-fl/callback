/** Every icon of the app, by what it stands for: the one place to change an icon everywhere. */
export function useIcons() {
  const icons = {
    // Applications
    add: 'i-mingcute-add-line',
    edit: 'i-mingcute-pencil-line',
    delete: 'i-mingcute-delete-2-line',
    trash: 'i-mingcute-delete-2-line',
    restore: 'i-mingcute-back-2-line',
    reset: 'i-mingcute-refresh-anticlockwise-1-line',
    empty: 'i-mingcute-inbox-line',
    followUp: 'i-mingcute-bell-ringing-line',
    location: 'i-mingcute-location-line',
    fullAddress: 'i-mingcute-map-pin-plus-line',
    contact: 'i-mingcute-user-add-line',
    mail: 'i-mingcute-mail-line',
    website: 'i-mingcute-external-link-line',
    calendar: 'i-mingcute-calendar-line',
    rating: 'i-mingcute-star-fill',
    ratingEmpty: 'i-mingcute-star-line',
    rejected: 'i-mingcute-close-circle-line',
    noReply: 'i-mingcute-sandglass-line',

    // Sources
    spontaneous: 'i-mingcute-send-plane-line',
    jobPost: 'i-mingcute-briefcase-line',
    recruiter: 'i-mingcute-user-3-line',
    otherSource: 'i-mingcute-tag-line',

    // Navigation and views
    applications: 'i-mingcute-list-check-line',
    dashboard: 'i-mingcute-chart-vertical-line',
    settings: 'i-mingcute-settings-3-line',
    table: 'i-mingcute-table-2-line',
    kanban: 'i-mingcute-columns-3-line',
    search: 'i-mingcute-search-line',
    chevronDown: 'i-mingcute-down-line',
    sort: 'i-mingcute-transfer-vertical-line',
    ascending: 'i-mingcute-sort-ascending-line',
    descending: 'i-mingcute-sort-descending-line',

    // Settings
    lightMode: 'i-mingcute-sun-line',
    darkMode: 'i-mingcute-moon-line',
    systemMode: 'i-mingcute-computer-line',
    apiToken: 'i-mingcute-key-2-line',
    copy: 'i-mingcute-copy-line',
    export: 'i-mingcute-download-line',
    import: 'i-mingcute-upload-line',
    logout: 'i-mingcute-exit-line',
  };

  return { icons };
}
