import type { ReactNode } from "react"

import { Brand } from "@/components/brand"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
} from "@/components/ui/card"

type AuthLayoutProps = {
  title: string
  description: string
  children: ReactNode
  footer?: ReactNode
}

export function AuthLayout({
  title,
  description,
  children,
  footer,
}: AuthLayoutProps) {
  return (
    <main className="min-h-dvh lg:grid lg:grid-cols-2">
      <aside className="hidden bg-primary p-10 text-primary-foreground lg:flex lg:flex-col lg:justify-between xl:p-16">
        <Brand tone="inverse" />

        <div className="max-w-md">
          <h2 className="font-display text-5xl leading-[0.9] tracking-tight xl:text-7xl">
            See where it goes.
          </h2>
          <p className="mt-5 max-w-sm text-sm font-medium text-primary-foreground/80">
            A simple ledger for everyday money. No spreadsheets, no jargon.
          </p>

          <div className="mt-10 grid grid-cols-2 gap-4">
            <div className="border-2 border-foreground bg-accent p-4 text-accent-foreground">
              <span className="text-xs font-semibold">This month</span>
              <span className="mt-2 block font-display text-3xl leading-none">
                ₹1,749
              </span>
            </div>
            <div className="border-2 border-foreground bg-destructive p-4 text-destructive-foreground">
              <span className="text-xs font-semibold">Spent</span>
              <span className="mt-2 block font-display text-3xl leading-none">
                ₹250
              </span>
            </div>
          </div>
          <div className="mt-4 h-2 w-24 bg-foreground" aria-hidden="true" />
        </div>

        <p className="text-xs font-medium text-primary-foreground/60">
          Simple expenses for everyday life.
        </p>
      </aside>

      <section className="flex min-h-dvh flex-col items-center justify-center bg-background px-4 py-8 sm:py-10">
        <div className="w-full max-w-sm">
          <div className="mb-8 flex justify-center lg:hidden">
            <Brand />
          </div>

          <Card className="[--card-spacing:--spacing(5)] sm:[--card-spacing:--spacing(6)]">
            <CardHeader className="gap-2 text-left">
              <h1 className="font-display text-3xl leading-[0.95] tracking-tight">
                {title}
              </h1>
              <CardDescription>{description}</CardDescription>
            </CardHeader>
            <CardContent className="pt-2">{children}</CardContent>
          </Card>

          {footer ? (
            <div className="mt-5 text-center text-sm text-muted-foreground">
              {footer}
            </div>
          ) : null}
        </div>
      </section>
    </main>
  )
}
