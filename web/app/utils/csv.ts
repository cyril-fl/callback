// A CSV file of rows, one column per key, every cell quoted. A text starting like a formula (= + - @, a tab or a
// carriage return) gets a ' before it: a spreadsheet would run it otherwise (CSV injection).
export function toCsv<T>(columns: readonly (keyof T & string)[], rows: T[]): string {
  const cell = (value: unknown) => {
    let text = String(value ?? '');
    if (typeof value === 'string' && /^[=+\-@\t\r]/.test(text)) text = `'${text}`;
    return `"${text.replaceAll('"', '""')}"`;
  };
  // The BOM tells a spreadsheet the file is UTF-8.
  return '﻿' + [columns.join(','), ...rows.map((row) => columns.map((c) => cell(row[c])).join(','))].join('\r\n');
}
