import type { RunResponse } from '@/lib/api'
import { countTestResults } from '@/lib/coding'
import { cn } from '@/lib/utils'

const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'

// Палитра консоли тестов (спека 1b): PASS #7DBE8E, FAIL #E08A80,
// детали #B9BEC4, команда/итог #8B9096.
function lineClass(line: string): string {
  if (/(FAIL|panic:|Error|error|undefined:|syntax error|not used|expected)/.test(line)) {
    return 'text-[#E08A80]'
  }
  if (/(---\s+PASS|^ok\b|^PASS\b|PASS\s)/.test(line)) return 'text-[#7DBE8E]'
  if (/^(===\s|---\s+SKIP|\?\s)/.test(line)) return 'text-[#8B9096]'
  return 'text-[#B9BEC4]'
}

export function GoConsole({ result }: { result: RunResponse | null }) {
  return (
    <div className="overflow-hidden rounded-[var(--r-md)] bg-[var(--editor)]">
      <div
        style={{ fontFamily: FONT_CODE }}
        className="border-b border-[var(--editor-line)] px-4 py-1.5 text-[11px] tracking-[0.08em] text-[#8B9096] uppercase"
      >
        консоль
      </div>
      <div
        style={{ fontFamily: FONT_CODE }}
        className="max-h-[340px] overflow-auto px-4 py-3 text-[12.5px] leading-[1.75]"
      >
        {result === null ? (
          <p className="text-[#8B9096]">Нажмите «Запустить тесты», чтобы увидеть результат.</p>
        ) : (
          <>
            <div className="whitespace-pre text-[#8B9096]">
              $ go test ./...
            </div>
            {result.output
              .replace(/\n$/, '')
              .split('\n')
              .map((line, i) => (
                <div key={i} className={cn('min-h-[1.75em] whitespace-pre', lineClass(line))}>
                  {line || ' '}
                </div>
              ))}
          </>
        )}
      </div>
    </div>
  )
}

const SUMMARY_LABEL: Record<RunResponse['result'], string> = {
  passed: 'Все тесты пройдены',
  tests_failed: 'Тесты не пройдены',
  compile_error: 'Ошибка компиляции',
  timeout: 'Превышено время выполнения',
}

export function GoRunSummary({ result }: { result: RunResponse }) {
  const { passed, failed } = countTestResults(result.output)
  const hasCounts = passed + failed > 0

  return (
    <div style={{ fontFamily: FONT_CODE }} className="mt-3.5 text-[12.5px]">
      {hasCounts ? (
        <span className="text-[var(--ink-2)]">
          <span className="text-[var(--ok)]">{passed} passed</span>
          {' · '}
          <span className={failed > 0 ? 'text-[var(--err)]' : 'text-[var(--ink-3)]'}>
            {failed} failed
          </span>
        </span>
      ) : (
        <span className={result.result === 'passed' ? 'text-[var(--ok)]' : 'text-[var(--err)]'}>
          {SUMMARY_LABEL[result.result]}
        </span>
      )}
    </div>
  )
}
