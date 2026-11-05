import { MailCheckIcon } from "lucide-react"

import { Button } from "@/components/ui/button"

type SignUpSuccessProps = {
  email: string
  onReset: () => void
}

export function SignUpSuccess({ email, onReset }: SignUpSuccessProps) {
  return (
    <div className="flex flex-col items-center gap-6 py-2 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
        <MailCheckIcon className="size-6" aria-hidden="true" />
      </div>

      <div className="flex flex-col gap-2">
        <h1 className="font-heading text-xl font-semibold">Check your email</h1>
        <p className="text-sm text-muted-foreground">
          We sent a verification link to{" "}
          <span className="font-medium text-foreground">{email}</span>.
        </p>
        <p className="text-xs text-muted-foreground">
          The link expires in 24 hours.
        </p>
      </div>

      <Button
        type="button"
        variant="outline"
        className="w-full"
        onClick={onReset}
      >
        Use a different email
      </Button>
    </div>
  )
}
