// The form's parts each status shows; any status can follow any other.
export type WorkflowStatus = 'applied' | 'responded' | 'interview' | 'offer' | 'rejected' | 'no_reply';
export type FormPart = 'interviews' | 'contacts';

// The parts of the follow-up section each status shows: the interviews from the interview step on.
export const PARTS: Record<WorkflowStatus, FormPart[]> = {
  applied: ['contacts'],
  no_reply: ['contacts'],
  responded: ['contacts'],
  interview: ['interviews', 'contacts'],
  offer: ['interviews', 'contacts'],
  rejected: ['contacts'],
};

// A part is shown by the status, or as long as it holds data, so nothing filled disappears.
export const showsPart = (status: WorkflowStatus, part: FormPart, filled = false) =>
  filled || PARTS[status].includes(part);

// A move to a status whose parts are still empty opens the form to fill them, instead of saving blind.
export const needsForm = (to: WorkflowStatus, hasInterview: boolean) =>
  PARTS[to].includes('interviews') && !hasInterview;
