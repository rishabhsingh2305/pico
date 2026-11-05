import type { ReactNode } from "react"
import { Link } from "react-router"
import { CheckCircle2Icon } from "lucide-react"

import { Brand } from "@/components/brand"
import { Card } from "@/components/ui/card"

type AuthShellProps = {
  children: ReactNode
  footer?: ReactNode
}

const highlights = [
  "Organize projects by team or client",
  "Track expenses with tags and date filters",
  "See filtered totals without losing context",
]

export function AuthShell({ children, footer }: AuthShellProps) {
  return (
    <main className="min-h-svh bg-muted/30 lg:grid lg:grid-cols-[minmax(0,1fr)_minmax(28rem,36rem)]">
      <aside className="relative hidden overflow-hidden bg-primary p-10 text-primary-foreground lg:flex lg:flex-col lg:justify-between lg:p-12">
        <div
          className="pointer-events-none absolute -top-40 -right-32 size-[28rem] rounded-full bg-primary-foreground/5 blur-3xl"
          aria-hidden="true"
        />
        <div
          className="pointer-events-none absolute -bottom-40 -left-24 size-96 rounded-full bg-primary-foreground/5 blur-3xl"
          aria-hidden="true"
        />

        <Brand align="start" tone="inverse" className="relative" />

        <div className="relative max-w-lg">
          <p className="text-xs font-semibold tracking-widest text-primary-foreground/70 uppercase">
            Expense workspace
          </p>
          <h2 className="mt-3 font-heading text-4xl leading-[1.1] font-semibold tracking-tight">
            Projects, people, and spending in one clear view.
          </h2>
          <ul className="mt-8 flex flex-col gap-3">
            {highlights.map((highlight) => (
              <li key={highlight} className="flex items-start gap-2.5 text-sm">
                <CheckCircle2Icon
                  className="mt-0.5 size-4 shrink-0 text-primary-foreground/80"
                  aria-hidden="true"
                />
                <span className="text-primary-foreground/85">{highlight}</span>
              </li>
            ))}
          </ul>
        </div>

        <p className="relative max-w-sm text-xs leading-relaxed text-primary-foreground/60">
          Shared expense tracking for small teams. Your verification link opens
          this workspace after signup.
        </p>
      </aside>

      <section className="flex min-h-svh items-center justify-center px-4 py-10 sm:px-6">
        <div className="w-full max-w-md">
          <div className="mb-6 lg:hidden">
            <Brand />
          </div>

          <Card className="w-full border bg-card shadow-sm">{children}</Card>

          <div className="mt-4 flex flex-col gap-2 text-center text-xs text-muted-foreground">
            {footer ? <div>{footer}</div> : null}
            <Link
              to="/api-docs"
              className="underline-offset-4 hover:text-foreground hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
            >
              API documentation
            </Link>
          </div>
        </div>
      </section>
    </main>
  )
}
