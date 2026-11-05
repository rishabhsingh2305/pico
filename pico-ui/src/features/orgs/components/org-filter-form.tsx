import { useId, useState } from "react"

import { TagInput } from "@/components/tag-input"
import { Button } from "@/components/ui/button"
import { Field, FieldLabel } from "@/components/ui/field"

type OrgFilterFormProps = {
  tags: string[]
  onChange: (tags: string[]) => void
}

export function OrgFilterForm({ tags, onChange }: OrgFilterFormProps) {
  const id = useId()
  const tagsId = `${id}-tags`
  const [tagInputKey, setTagInputKey] = useState(0)

  function clear() {
    onChange([])
    setTagInputKey((current) => current + 1)
  }

  return (
    <div className="flex flex-col gap-3 border-y border-border py-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <Field className="sm:min-w-0 sm:flex-1">
          <FieldLabel
            htmlFor={tagsId}
            className="text-label text-muted-foreground uppercase"
          >
            Filter tags (any)
          </FieldLabel>
          <TagInput
            key={tagInputKey}
            id={tagsId}
            name="orgTags"
            value={tags}
            onChange={onChange}
            placeholder="team, finance"
          />
        </Field>
        <Button
          type="button"
          variant="ghost"
          className="w-fit self-end"
          onClick={clear}
        >
          Clear
        </Button>
      </div>
    </div>
  )
}
