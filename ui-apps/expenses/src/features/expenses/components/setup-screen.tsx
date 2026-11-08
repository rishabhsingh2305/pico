import { CircleAlertIcon } from "lucide-react"

import { Brand } from "@/components/brand"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"

export function CheckingScreen() {
  return (
    <main className="flex min-h-dvh flex-col items-center justify-center gap-5 bg-background px-4">
      <Brand />
      <Spinner className="size-5 text-muted-foreground" />
      <p className="sr-only" role="status">
        Loading your expenses
      </p>
    </main>
  )
}

export function SetupScreen() {
  return (
    <main className="flex min-h-dvh items-center justify-center bg-background p-4">
      <div className="w-full max-w-lg border-2 border-foreground bg-primary p-6 text-primary-foreground sm:p-8">
        <Brand tone="inverse" />

        <h1 className="mt-10 font-display text-4xl leading-[0.9] tracking-tight sm:text-5xl">
          Setting up your expenses
        </h1>
        <div className="mt-4 h-2 w-20 bg-accent" aria-hidden="true" />
        <p className="mt-4 text-sm font-medium text-primary-foreground/80">
          We&apos;re preparing your expense tracker. This will only take a
          moment.
        </p>

        <div className="mt-8 flex gap-2" aria-hidden="true">
          <span className="size-3 animate-bounce bg-accent [animation-delay:-0.3s]" />
          <span className="size-3 animate-bounce bg-accent [animation-delay:-0.15s]" />
          <span className="size-3 animate-bounce bg-accent" />
        </div>

        <p className="sr-only" role="status" aria-live="polite">
          Setting up your expenses
        </p>
      </div>
    </main>
  )
}

type SetupErrorScreenProps = {
  message: string
  onRetry: () => void
  onSignOut: () => void
}

export function SetupErrorScreen({
  message,
  onRetry,
  onSignOut,
}: SetupErrorScreenProps) {
  return (
    <main className="flex min-h-dvh items-center justify-center bg-background p-4">
      <div className="w-full max-w-lg border-2 border-foreground bg-card p-6 sm:p-8">
        <div className="flex size-12 items-center justify-center border-2 border-foreground bg-destructive text-destructive-foreground">
          <CircleAlertIcon className="size-6" aria-hidden="true" />
        </div>

        <h1 className="mt-6 font-display text-3xl leading-[0.95] tracking-tight">
          We couldn&apos;t set up your expenses
        </h1>
        <p className="mt-3 text-sm text-muted-foreground">{message}</p>

        <div className="mt-6 flex flex-col gap-2">
          <Button type="button" className="w-full" onClick={onRetry}>
            Try again
          </Button>
          <Button
            type="button"
            variant="outline"
            className="w-full"
            onClick={onSignOut}
          >
            Sign out
          </Button>
        </div>
      </div>
    </main>
  )
}
