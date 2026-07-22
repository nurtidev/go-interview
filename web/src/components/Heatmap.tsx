import { useMemo } from 'react'
import type { ActivityDay } from '@/lib/api'
import { cn } from '@/lib/utils'

const WEEKS = 12
const DAYS = 7
const TOTAL = WEEKS * DAYS

// Ramp из README (макет 1o): --heat-0 → --heat-3, от «меньше» к «больше».
const RAMP = ['var(--heat-0)', 'var(--heat-1)', 'var(--heat-2)', 'var(--heat-3)']

function level(count: number): number {
  if (count >= 6) return 3
  if (count >= 3) return 2
  if (count >= 1) return 1
  return 0
}

// Heatmap 7×12 (макет 1o). Данные — activity (≈84 дня). Недостающие дни дополняются нулями.
export function Heatmap({
  activity,
  className,
}: {
  activity: ActivityDay[]
  className?: string
}) {
  const cells = useMemo(() => {
    const sorted = [...activity]
      .filter((a) => a && a.date)
      .sort((a, b) => a.date.localeCompare(b.date))
      .slice(-TOTAL)
    const pad = Math.max(0, TOTAL - sorted.length)
    const padded: (ActivityDay | null)[] = [...Array<null>(pad).fill(null), ...sorted]
    return padded
  }, [activity])

  return (
    <div className={cn('flex flex-col gap-2.5', className)}>
      <div className="overflow-x-auto">
        <div className="flex w-max gap-[3px]">
          {Array.from({ length: WEEKS }).map((_, w) => (
            <div key={w} className="grid gap-[3px]" style={{ gridTemplateRows: `repeat(${DAYS}, 12px)` }}>
              {Array.from({ length: DAYS }).map((_, d) => {
                const cell = cells[w * DAYS + d]
                const lvl = cell ? level(cell.count) : 0
                return (
                  <div
                    key={d}
                    className="w-3 rounded-[3px]"
                    style={{ backgroundColor: RAMP[lvl] }}
                    title={cell ? `${cell.date}: ${cell.count}` : undefined}
                  />
                )
              })}
            </div>
          ))}
        </div>
      </div>
      <div className="flex items-center gap-2 font-mono text-[10.5px] text-ink-3">
        <span>меньше</span>
        <div className="flex gap-[3px]">
          {RAMP.map((c) => (
            <div key={c} className="size-2.5 rounded-[3px]" style={{ backgroundColor: c }} />
          ))}
        </div>
        <span>больше</span>
      </div>
    </div>
  )
}
