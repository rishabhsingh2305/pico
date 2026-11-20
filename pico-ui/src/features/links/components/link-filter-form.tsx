import { useId } from "react"

import { TagInput } from "@/components/tag-input"
import { Button } from "@/components/ui/button"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

type LinkFilterFormProps = {
  query: string
  tags: string[]
  onQueryChange: (value: string) => void
  onTagsChange: (tags: string[]) => void
  onClear: () => void
}

export function LinkFilterForm({
  query,
  tags,
  onQueryChange,
  onTagsChange,
  onClear,
}: LinkFilterFormProps) {
  const id = useId()
  const queryId = `${id}-query`
  const tagsId = `${id}-tags`

  return (
    <div className="flex flex-col gap-3 border-y border-border py-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <Field className="sm:min-w-0 sm:flex-1">
          <FieldLabel
            htmlFor={queryId}
            className="text-label text-muted-foreground uppercase"
          >
            Search aliases
          </FieldLabel>
          <Input
            id={queryId}
            name="query"
            type="search"
            autoComplete="off"
            placeholder="launch"
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
          />
        </Field>

        <Field className="sm:min-w-0 sm:flex-1">
          <FieldLabel
            htmlFor={tagsId}
            className="text-label text-muted-foreground uppercase"
          >
            Filter tags (any)
          </FieldLabel>
          <TagInput
            id={tagsId}
            name="linkTags"
            value={tags}
            onChange={onTagsChange}
            placeholder="marketing, docs"
          />
        </Field>

        <Button
          type="button"
          variant="ghost"
          className="w-fit self-end"
          onClick={onClear}
        >
          Clear
        </Button>
      </div>
    </div>
  )
}
