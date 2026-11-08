import type { ReactNode } from "react"

import { AppHeader } from "@/components/app-header"
import { cn } from "@/lib/utils"

type AppShellProps = {
  children: ReactNode
  className?: string
}

export function AppShell({ children, className }: AppShellProps) {
  return (
    <div className="flex h-dvh flex-col overflow-hidden bg-background">
      <AppHeader />
      <div className={cn("flex min-h-0 flex-1 flex-col", className)}>
        {children}
      </div>
    </div>
  )
}
