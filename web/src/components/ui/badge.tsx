import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { Slot } from "radix-ui"

import { cn } from "@/lib/utils"

const badgeVariants = cva(
  "inline-flex w-fit shrink-0 items-center justify-center gap-1 rounded-full px-2.5 py-0.5 text-[11.5px] leading-[1.5] font-medium whitespace-nowrap transition-colors duration-150 ease-out [&>svg]:pointer-events-none [&>svg]:size-3",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground",
        secondary: "bg-accent-soft text-ink",
        neutral: "bg-tint-neutral text-ink-2",
        outline: "border border-hairline text-ink-3",
        ok: "bg-ok-soft text-ok",
        warn: "bg-warn-soft text-warn",
        err: "bg-err-soft text-err",
        info: "bg-info-soft text-info",
        staff: "bg-staff-soft text-staff",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

function Badge({
  className,
  variant = "default",
  asChild = false,
  ...props
}: React.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot.Root : "span"

  return (
    <Comp
      data-slot="badge"
      data-variant={variant}
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
