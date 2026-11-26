import assert from 'node:assert/strict';
import { test } from 'node:test';
import { TRANSITIONS, canMove, needsForm, nextStatuses, showsPart, type WorkflowStatus } from '../app/utils/workflow.ts';

test('every transition leads to a known status, never to itself', () => {
  for (const [from, to] of Object.entries(TRANSITIONS))
    for (const s of to) {
      assert.ok(s in TRANSITIONS);
      assert.notEqual(s, from);
    }
});

test('the select offers the current status first', () => {
  assert.deepEqual(nextStatuses('offer'), ['offer', 'interview', 'rejected']);
});

test('the interviews show from the interview step on, or when filled', () => {
  for (const s of ['applied', 'responded', 'no_reply', 'rejected'] as WorkflowStatus[])
    assert.equal(showsPart(s, 'interviews'), false);
  assert.ok(showsPart('interview', 'interviews') && showsPart('offer', 'interviews'));
  assert.ok(showsPart('rejected', 'interviews', true));
});

test('a move follows the transitions, and opens the form when the interviews are missing', () => {
  assert.ok(canMove('applied', 'interview'));
  assert.equal(canMove('applied', 'offer'), false);
  assert.ok(needsForm('interview', false));
  assert.equal(needsForm('interview', true), false);
  assert.equal(needsForm('rejected', false), false);
});
