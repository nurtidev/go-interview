import * as React from "react"
import { Progress as ProgressPrimitive } from "radix-ui"

import { cn } from "@/lib/utils"

/** Прогресс-бар: высота 4px, трек --hairline, заливка --ink, radius 2px, без скруглённых концов. */
function Progress({
  className,
  value,
  ...props
}: React.ComponentProps<typeof ProgressPrimitive.Root>) {
  return (
    <ProgressPrimitive.Root
      data-slot="progress"
      className={cn(
        "relative h-1 w-full overflow-hidden rounded-[2px] bg-hairline",
        className
      )}
      {...props}
    >
      <ProgressPrimitive.Indicator
        data-slot="progress-indicator"
        className="h-full w-full flex-1 bg-primary transition-transform duration-150 ease-out"
        style={{ transform: `translateX(-${100 - (value || 0)}%)` }}
      />
    </ProgressPrimitive.Root>
  )
}

/** Сегментный прогресс (для глав учебника): flex с gap 2px, каждый сегмент = глава. */
function SegmentedProgress({
  total,
  filled,
  className,
}: {
  total: number
  filled: number
  className?: string
}) {
  const count = Math.max(0, total)
  return (
    <div className={cn("flex h-1 gap-[2px] overflow-hidden rounded-[2px]", className)}>
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={i}
          className={cn("flex-1", i < filled ? "bg-primary" : "bg-hairline")}
        />
      ))}
    </div>
  )
}

export { Progress, SegmentedProgress }
