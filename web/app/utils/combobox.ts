// Enter in a combobox picks the highlighted suggestion if there is one; else it is the typed text that counts.
export const noneHighlighted = (event: Event) => !(event.target as HTMLElement).getAttribute('aria-activedescendant');
