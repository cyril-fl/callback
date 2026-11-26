import assert from 'node:assert/strict';
import { test } from 'node:test';
import { groupValues } from '../app/utils/values.ts';

test('groups the ids by value, most used first, then by name, without empty values', () => {
  const items = [
    { id: '1', city: 'Lyon' },
    { id: '2', city: 'Paris' },
    { id: '3', city: 'Lyon' },
    { id: '4', city: '' },
    { id: '5' },
    { id: '6', city: 'Brest' },
  ];
  assert.deepEqual(groupValues<{ id: string; city?: string }>(items, 'city'), [
    { value: 'Lyon', ids: ['1', '3'] },
    { value: 'Brest', ids: ['6'] },
    { value: 'Paris', ids: ['2'] },
  ]);
});
