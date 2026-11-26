import type { Contact } from '~/types/items';

// In the pro plan the recruiter of a recruiter source is a contact with this role: its e-mail is the source's detail.
export const RECRUITER_ROLE = 'Recruteur';

const isRecruiter = (c: Contact) => c.role?.trim().toLowerCase() === RECRUITER_ROLE.toLowerCase();

/** The e-mail of the first recruiter among the contacts that has one. */
export const recruiterEmail = (contacts: Contact[]) =>
  contacts.find((c) => isRecruiter(c) && c.email?.trim())?.email?.trim();

/** The contacts with this e-mail among them, added with role when none has it. */
export function withContact(contacts: Contact[], email: string, role = ''): Contact[] {
  if (!email || contacts.some((c) => c.email?.trim().toLowerCase() === email.trim().toLowerCase())) return contacts;
  return [...contacts, { name: '', email: email.trim(), role }];
}

/** The contacts with a recruiter row, an empty one added when none is there. */
export const withRecruiterRow = (contacts: Contact[]): Contact[] =>
  contacts.some(isRecruiter) ? contacts : [...contacts, { name: '', email: '', role: RECRUITER_ROLE }];
