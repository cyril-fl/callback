import assert from 'node:assert/strict';
import { test } from 'node:test';
import { reorder } from '../app/utils/kanban.ts';

test('a card moves inside its column', () => {
  assert.deepEqual(reorder(['a', 'b', 'c'], 'a', 3), ['b', 'c', 'a']);
  assert.deepEqual(reorder(['a', 'b', 'c'], 'c', 0), ['c', 'a', 'b']);
  assert.deepEqual(reorder(['a', 'b', 'c'], 'a', 2), ['b', 'a', 'c']);
});

test('a drop on its own place moves nothing', () => {
  assert.equal(reorder(['a', 'b', 'c'], 'b', 1), undefined);
  assert.equal(reorder(['a', 'b', 'c'], 'b', 2), undefined);
});

test('a card from another column comes in at the index', () => {
  assert.deepEqual(reorder(['a', 'b'], 'x', 1), ['a', 'x', 'b']);
  assert.deepEqual(reorder([], 'x', 0), ['x']);
});
