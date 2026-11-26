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

// Only the interview step requires an interview: an offer can come without one.
export const requiresInterview = (status: WorkflowStatus) => status === 'interview';

// A move to the interview step without one opens the modal to fill it, instead of saving blind.
export const needsForm = (to: WorkflowStatus, hasInterview: boolean) => requiresInterview(to) && !hasInterview;
