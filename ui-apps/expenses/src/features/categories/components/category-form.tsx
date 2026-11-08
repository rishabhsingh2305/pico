import { useId, useState, type FormEvent } from "react"
import { CircleAlertIcon } from "lucide-react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { createCategory } from "@/features/categories/api"
import { categoryErrorMessage } from "@/features/categories/error-message"
import type { Category } from "@/features/categories/types"
import { validateCategoryName } from "@/features/categories/validation"
import { useExpenses } from "@/features/expenses/expenses-provider"

type CategoryFormProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (category: Category) => void
}

export function CategoryForm({
  open,
  onOpenChange,
  onCreated,
}: CategoryFormProps) {
  const id = useId()
  const nameId = `${id}-name`
  const { orgId } = useExpenses()

  const [name, setName] = useState("")
  const [error, setError] = useState("")
  const [formError, setFormError] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)

  function reset() {
    setName("")
    setError("")
    setFormError("")
    setIsSubmitting(false)
  }

  function handleOpenChange(next: boolean) {
    if (!next) {
      reset()
    }
    onOpenChange(next)
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (isSubmitting) {
      return
    }

    const nextError = validateCategoryName(name)
    setError(nextError ?? "")
    setFormError("")
    if (nextError) {
      return
    }

    setIsSubmitting(true)
    try {
      const category = await createCategory(orgId, { name })
      setIsSubmitting(false)
      onCreated(category)
      handleOpenChange(false)
    } catch (caught) {
      setFormError(categoryErrorMessage(caught))
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New category</DialogTitle>
          <DialogDescription>
            Create a category like Food, Travel, or Bills.
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={handleSubmit}
          noValidate
          aria-busy={isSubmitting}
          className="flex flex-col gap-4"
        >
          <div className="flex flex-col gap-2">
            <Label htmlFor={nameId}>Category name</Label>
            <Input
              id={nameId}
              name="name"
              type="text"
              autoFocus
              placeholder="Food"
              className="h-11"
              value={name}
              onChange={(event) => {
                setName(event.target.value)
                if (error) {
                  setError("")
                }
                if (formError) {
                  setFormError("")
                }
              }}
              disabled={isSubmitting}
              aria-invalid={Boolean(error)}
              aria-describedby={error ? `${id}-name-error` : undefined}
            />
            {error ? (
              <p id={`${id}-name-error`} className="text-sm text-destructive">
                {error}
              </p>
            ) : null}
          </div>

          {formError ? (
            <Alert variant="destructive">
              <CircleAlertIcon aria-hidden="true" />
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          ) : null}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? (
                <>
                  <Spinner data-icon="inline-start" />
                  Creating...
                </>
              ) : (
                "Create category"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
