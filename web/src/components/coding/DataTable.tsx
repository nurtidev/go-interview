const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'

/**
 * Таблица SQL-результата по спеке 1b: рамка+разделители --hairline, radius 10,
 * шапка mono 11/600 --ink-3 на тинте, ячейки mono 12.5, первый столбец приглушён.
 */
export function DataTable({ columns, rows }: { columns: string[]; rows: string[][] }) {
  return (
    <div className="overflow-x-auto rounded-[var(--r-md)] border border-[var(--hairline)]">
      <table style={{ fontFamily: FONT_CODE }} className="w-full border-collapse">
        <thead>
          <tr className="bg-[var(--accent-soft)]">
            {columns.map((col, i) => (
              <th
                key={i}
                className="px-3.5 py-2 text-left text-[11px] font-semibold whitespace-nowrap text-[var(--ink-3)]"
              >
                {col}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr>
              <td
                colSpan={Math.max(1, columns.length)}
                className="border-t border-[var(--hairline)] px-3.5 py-2 text-[12.5px] text-[var(--ink-3)]"
              >
                (пусто)
              </td>
            </tr>
          ) : (
            rows.map((row, ri) => (
              <tr key={ri}>
                {row.map((cell, ci) => (
                  <td
                    key={ci}
                    className="border-t border-[var(--hairline)] px-3.5 py-1.5 text-[12.5px] whitespace-nowrap tabular-nums first:text-[var(--ink-3)] text-[var(--ink)]"
                  >
                    {cell}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}
