import { CheckIcon } from "lucide-react"

import { Button } from "@/components/ui/button"

type VerificationSuccessProps = {
  email: string
  onContinue: () => void
}

export function VerificationSuccess({
  email,
  onContinue,
}: VerificationSuccessProps) {
  return (
    <div className="flex flex-col items-center gap-5 py-2 text-center">
      <div className="flex size-12 items-center justify-center border-2 border-foreground bg-primary text-primary-foreground">
        <CheckIcon className="size-6" aria-hidden="true" />
      </div>
      <p className="text-sm text-muted-foreground">
        <span className="font-medium text-foreground">{email}</span> is
        verified. Sign in to start tracking your expenses.
      </p>
      <Button type="button" className="h-11 w-full" onClick={onContinue}>
        Continue to sign in
      </Button>
    </div>
  )
}
