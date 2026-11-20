import type { ReactNode } from "react"

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "cn"

type SelectOption = {
  value: string
  label: ReactNode
  disabled?: boolean
}

type SelectFieldProps = {
  id?: string
  name?: string
  value: string
  onValueChange: (value: string) => void
  options: readonly SelectOption[]
  disabled?: boolean
  invalid?: boolean
  className?: string
  "aria-label"?: string
  "aria-describedby"?: string
}

export function SelectField({
  id,
  name,
  value,
  onValueChange,
  options,
  disabled,
  invalid,
  className,
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
}: SelectFieldProps) {
  return (
    <Select
      name={name}
      value={value}
      onValueChange={(next) => {
        if (next !== null) {
          onValueChange(next)
        }
      }}
      items={options}
      disabled={disabled}
    >
      <SelectTrigger
        id={id}
        className={cn("w-full", className)}
        aria-label={ariaLabel}
        aria-describedby={ariaDescribedBy}
        aria-invalid={invalid || undefined}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent
        align="start"
        alignItemWithTrigger={false}
        className="w-auto max-w-(--available-width) min-w-(--anchor-width)"
      >
        {options.map((option) => (
          <SelectItem
            key={option.value}
            value={option.value}
            disabled={option.disabled}
          >
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
