/** Every keyboard shortcut of the app, by what it does: the one place to change a key, for its handler (keys joined by _) and the kbds shown. */
export function useShortcuts() {
  const shortcuts = {
    search: ['/'],
    add: ['alt', 'n'],
    save: ['meta', 's'],
    selectAll: ['meta', 'a'],
    undo: ['meta', 'z'],
    redo: ['meta', 'y'],
    redoShift: ['meta', 'shift', 'z'],
  };

  return { shortcuts, key: (name: keyof typeof shortcuts) => shortcuts[name].join('_') };
}
