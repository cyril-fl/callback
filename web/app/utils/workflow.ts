// The application's status as a state machine, and the form's parts each status shows.
export type WorkflowStatus = 'applied' | 'responded' | 'interview' | 'offer' | 'rejected' | 'no_reply';
export type FormPart = 'interviews' | 'contacts';

// The statuses reachable from each one; going back one step stays possible to fix a mistake.
export const TRANSITIONS: Record<WorkflowStatus, WorkflowStatus[]> = {
  applied: ['responded', 'interview', 'rejected', 'no_reply'],
  no_reply: ['applied', 'responded', 'interview', 'rejected'],
  responded: ['applied', 'interview', 'rejected'],
  interview: ['responded', 'offer', 'rejected'],
  offer: ['interview', 'rejected'],
  rejected: ['applied', 'responded', 'interview', 'offer'],
};

// The parts of the follow-up section each status shows: the interviews from the interview step on.
export const PARTS: Record<WorkflowStatus, FormPart[]> = {
  applied: ['contacts'],
  no_reply: ['contacts'],
  responded: ['contacts'],
  interview: ['interviews', 'contacts'],
  offer: ['interviews', 'contacts'],
  rejected: ['contacts'],
};

// The statuses the select offers: the current one, then where it can go.
export const nextStatuses = (from: WorkflowStatus) => [from, ...TRANSITIONS[from]];

// A part is shown by the status, or as long as it holds data, so nothing filled disappears.
export const showsPart = (status: WorkflowStatus, part: FormPart, filled = false) =>
  filled || PARTS[status].includes(part);
