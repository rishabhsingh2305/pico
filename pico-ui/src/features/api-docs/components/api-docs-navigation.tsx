import type { ApiSection } from "@/features/api-docs/types"
import { cn } from "cn"

type ApiDocsNavigationProps = {
  sections: ApiSection[]
  className?: string
}

export function ApiDocsNavigation({
  sections,
  className,
}: ApiDocsNavigationProps) {
  return (
    <nav
      aria-label="API documentation sections"
      className={cn("text-sm", className)}
    >
      <p className="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
        On this page
      </p>
      <ul className="flex flex-col gap-1">
        <li>
          <a
            href="#overview"
            className="flex items-center justify-between rounded-md px-2 py-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          >
            Overview
          </a>
        </li>
        {sections.map((section) => (
          <li key={section.id}>
            <a
              href={`#${section.id}`}
              className="flex items-center justify-between gap-2 rounded-md px-2 py-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
            >
              <span>{section.title}</span>
              <span className="text-xs tabular-nums">
                {section.endpoints.length}
              </span>
            </a>
          </li>
        ))}
        <li>
          <a
            href="#common-behavior"
            className="flex items-center justify-between rounded-md px-2 py-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          >
            Common behavior
          </a>
        </li>
      </ul>
    </nav>
  )
}
