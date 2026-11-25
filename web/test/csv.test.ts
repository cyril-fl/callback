import assert from 'node:assert/strict';
import { test } from 'node:test';
import { toCsv } from '../app/utils/csv.ts';

type Row = { company?: string; rating?: number };

test('quotes every cell, doubles the quotes, keeps newlines inside a cell', () => {
  const csv = toCsv<Row>(['company', 'rating'], [{ company: 'Acme "SA"\nParis', rating: 4 }, {}]);
  assert.equal(csv, '﻿company,rating\r\n"Acme ""SA""\nParis","4"\r\n"",""');
});

test('a text that a spreadsheet would run as a formula is neutralised', () => {
  for (const text of ['=HYPERLINK("x")', '+1', '-1', '@SUM(A1)', '\tx', '\rx']) {
    assert.equal(toCsv<Row>(['company'], [{ company: text }]).split('\r\n')[1], `"'${text.replaceAll('"', '""')}"`);
  }
  assert.equal(toCsv<Row>(['company'], [{ company: 'Acme = SA' }]).split('\r\n')[1], '"Acme = SA"');
});
