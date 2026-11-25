/** A change that can be undone, then done again: each tells whether it went through. */
type Step = { undo: () => Promise<boolean>; redo: () => Promise<boolean> };
// The changes of this tab, for ⌘Z and ⌘Y: a new one forgets what was undone.
const done: Step[] = [];
let undone: Step[] = [];
// They run one after the other: a second ⌘Z waits for the first.
let queue: Promise<unknown> = Promise.resolve();
function inTurn(change: () => Promise<void>) {
  queue = queue.then(change).catch(() => {});
  return queue;
}

/** The undo and redo of the list's changes; a step that fails is dropped, its toast told why. */
export function useHistory() {
  function record(step: Step) {
    done.push(step);
    done.splice(0, done.length - 50);
    undone = [];
  }
  const undo = () =>
    inTurn(async () => {
      const step = done.pop();
      if (step && (await step.undo())) undone.push(step);
    });
  const redo = () =>
    inTurn(async () => {
      const step = undone.pop();
      if (step && (await step.redo())) done.push(step);
    });
  return { record, undo, redo };
}
