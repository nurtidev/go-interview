import { DataTable } from '@/components/coding/DataTable'
import type { RowsDiff } from '@/lib/coding'
import type { SqlRunResult } from '@/lib/sqlRunner'

const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'

export interface SqlOutcome {
  result?: SqlRunResult
  diff?: RowsDiff
  error?: string
}

function pluralRows(n: number): string {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return 'строка'
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return 'строки'
  return 'строк'
}

function tuple(row: string[]): string {
  return `(${row.join(', ')})`
}

export function SqlResultView({ outcome }: { outcome: SqlOutcome | null }) {
  if (outcome === null) {
    return (
      <div className="rounded-[var(--r-md)] border border-[var(--hairline)] p-4">
        <p style={{ fontFamily: FONT_CODE }} className="text-[13px] text-[var(--ink-3)]">
          Нажмите «Выполнить запрос», чтобы увидеть результат.
        </p>
      </div>
    )
  }

  if (outcome.error) {
    return (
      <div className="rounded-[var(--r-md)] border border-[var(--err)]/40 bg-[var(--err-soft)] p-4">
        <pre
          style={{ fontFamily: FONT_CODE }}
          className="overflow-x-auto text-[13px] leading-relaxed whitespace-pre-wrap text-[var(--err)]"
        >
          {outcome.error}
        </pre>
      </div>
    )
  }

  const { result, diff } = outcome
  if (!result || !diff) return null

  return (
    <div className="space-y-3">
      <DataTable columns={result.columns} rows={result.rows} />

      {/* Футер по спеке 1b: «N строк · N мс · совпадает ✓» */}
      <p style={{ fontFamily: FONT_CODE }} className="text-[11px] tabular-nums text-[var(--ink-3)]">
        {result.rowCount} {pluralRows(result.rowCount)} · {result.elapsedMs} мс
        {diff.matched ? (
          <span className="text-[var(--ok)]"> · совпадает с ожидаемым ✓</span>
        ) : (
          <span className="text-[var(--err)]"> · не совпадает с эталоном</span>
        )}
      </p>

      {!diff.matched && (
        <div style={{ fontFamily: FONT_CODE }} className="space-y-1 text-[12.5px] text-[var(--ink-2)]">
          {diff.missing.length > 0 && (
            <p>
              Не хватает строк: {diff.missing.slice(0, 5).map(tuple).join('  ')}
              {diff.missing.length > 5 && ` … ещё ${diff.missing.length - 5}`}
            </p>
          )}
          {diff.extra.length > 0 && (
            <p>
              Лишние строки: {diff.extra.slice(0, 5).map(tuple).join('  ')}
              {diff.extra.length > 5 && ` … ещё ${diff.extra.length - 5}`}
            </p>
          )}
        </div>
      )}
    </div>
  )
}
