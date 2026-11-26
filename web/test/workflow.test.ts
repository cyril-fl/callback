import assert from 'node:assert/strict';
import { test } from 'node:test';
import { needsForm, showsPart, type WorkflowStatus } from '../app/utils/workflow.ts';

test('the interviews show from the interview step on, or when filled', () => {
  for (const s of ['applied', 'responded', 'no_reply', 'rejected'] as WorkflowStatus[])
    assert.equal(showsPart(s, 'interviews'), false);
  assert.ok(showsPart('interview', 'interviews') && showsPart('offer', 'interviews'));
  assert.ok(showsPart('rejected', 'interviews', true));
});

test('a move opens the form when the interviews are missing', () => {
  assert.ok(needsForm('interview', false));
  assert.equal(needsForm('interview', true), false);
  assert.equal(needsForm('rejected', false), false);
  assert.equal(needsForm('offer', false), false);
});
