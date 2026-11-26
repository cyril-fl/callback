import { test } from 'node:test';
import assert from 'node:assert/strict';
import { recruiterEmail, withContact, withRecruiterRow, RECRUITER_ROLE } from '../app/utils/recruiter.ts';

test('the recruiter is the first contact of that role with an e-mail', () => {
  assert.equal(
    recruiterEmail([
      { name: 'Léa', role: 'recruteur' },
      { email: ' rh@x.fr ', role: RECRUITER_ROLE },
    ]),
    'rh@x.fr',
  );
  assert.equal(recruiterEmail([{ email: 'a@x.fr', role: 'CTO' }]), undefined);
});

test('a contact is added once per e-mail, and one recruiter row', () => {
  const once = withContact([], 'RH@x.fr', RECRUITER_ROLE);
  assert.equal(once.length, 1);
  assert.equal(withContact(once, 'rh@x.fr').length, 1);
  assert.equal(withContact(once, '').length, 1);
  assert.equal(withRecruiterRow(once).length, 1);
  assert.equal(withRecruiterRow([{ name: 'Max' }]).length, 2);
});
