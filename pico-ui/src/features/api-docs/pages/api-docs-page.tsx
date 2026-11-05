import { useState } from "react"
import { Link } from "react-router"
import { BookOpenIcon, CircleAlertIcon, SearchIcon, XIcon } from "lucide-react"

import { Brand } from "@/components/brand"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  API_INTRO,
  API_SECTIONS,
  COMMON_ERRORS,
} from "@/features/api-docs/content"
import { ApiDocsNavigation } from "@/features/api-docs/components/api-docs-navigation"
import { ApiEndpointCard } from "@/features/api-docs/components/api-endpoint-card"
import { CodeBlock } from "@/features/api-docs/components/code-block"
import { CopyButton } from "@/features/api-docs/components/copy-button"
import type { ApiEndpoint, ApiSection } from "@/features/api-docs/types"
import { getSession } from "@/features/auth/session"
import { usePageTitle } from "@/hooks/use-page-title"
import { API_BASE_URL } from "@/lib/env"

export function ApiDocsPage() {
  const session = getSession()
  const [query, setQuery] = useState("")
  usePageTitle("API docs · Pico")

  const normalizedQuery = query.trim().toLowerCase()
  const visibleSections = API_SECTIONS.map((section) => ({
    ...section,
    endpoints: section.endpoints.filter((endpoint) =>
      matchesSearch(endpoint, section, normalizedQuery)
    ),
  })).filter((section) => section.endpoints.length > 0)
  const endpointCount = visibleSections.reduce(
    (total, section) => total + section.endpoints.length,
    0
  )

  return (
    <div className="min-h-svh bg-background">
      <header className="sticky top-0 z-30 border-b bg-background/90 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4 sm:px-6">
          <Link
            to="/"
            aria-label="Pico home"
            className="rounded-md focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          >
            <Brand align="start" />
          </Link>
          <div className="flex items-center gap-2">
            <Badge variant="outline" className="hidden sm:inline-flex">
              <BookOpenIcon aria-hidden="true" />
              API reference
            </Badge>
            <Link
              to={session ? "/orgs" : "/signup"}
              className={buttonVariants({ variant: "outline", size: "sm" })}
            >
              {session ? "Back to app" : "Create an account"}
            </Link>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:py-10">
        <div className="grid gap-8 lg:grid-cols-[13rem_minmax(0,1fr)]">
          <aside className="hidden lg:block">
            <div className="sticky top-24">
              <ApiDocsNavigation sections={visibleSections} />
            </div>
          </aside>

          <div className="flex min-w-0 flex-col gap-8">
            <details className="rounded-xl bg-card p-3 ring-1 ring-foreground/10 lg:hidden">
              <summary className="cursor-pointer text-sm font-medium">
                Browse sections
              </summary>
              <ApiDocsNavigation sections={visibleSections} className="mt-3" />
            </details>

            <section
              id="overview"
              aria-labelledby="overview-heading"
              className="scroll-mt-24"
            >
              <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                API reference
              </p>
              <h1
                id="overview-heading"
                className="mt-1 font-heading text-3xl font-semibold"
              >
                {API_INTRO.title}
              </h1>
              <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
                {API_INTRO.summary}
              </p>

              <div className="mt-5 grid gap-3 sm:grid-cols-2">
                <div className="rounded-xl bg-card p-4 ring-1 ring-foreground/10">
                  <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                    Base URL
                  </p>
                  <div className="mt-1 flex items-center gap-1">
                    <code className="font-mono text-sm break-all">
                      {API_BASE_URL}
                    </code>
                    <CopyButton value={API_BASE_URL} label="Copy base URL" />
                  </div>
                  <p className="mt-2 text-xs text-muted-foreground">
                    Development uses the Vite proxy at `/api`. Production builds
                    use the configured Function URL.
                  </p>
                </div>

                <div className="rounded-xl bg-card p-4 ring-1 ring-foreground/10">
                  <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                    Read endpoints
                  </p>
                  <p className="mt-1 text-sm">
                    Support both <code className="font-mono">GET</code> and{" "}
                    <code className="font-mono">HEAD</code>. A HEAD response
                    carries headers and no body.
                  </p>
                  <p className="mt-2 text-xs text-muted-foreground">
                    List endpoints accept{" "}
                    <code className="font-mono">limit</code> and{" "}
                    <code className="font-mono">offset</code>. Default page size
                    is 20; maximum is 100.
                  </p>
                </div>
              </div>

              <Alert className="mt-3">
                <CircleAlertIcon aria-hidden="true" />
                <AlertDescription>
                  <strong className="text-foreground">
                    Identity is not authentication.
                  </strong>{" "}
                  {API_INTRO.identityNote}
                </AlertDescription>
              </Alert>
            </section>

            <section aria-labelledby="search-heading">
              <h2 id="search-heading" className="sr-only">
                Search API documentation
              </h2>
              <div className="rounded-xl bg-card p-4 ring-1 ring-foreground/10">
                <Field>
                  <FieldLabel htmlFor="api-docs-search">
                    Search endpoints
                  </FieldLabel>
                  <div className="flex gap-2">
                    <div className="relative min-w-0 flex-1">
                      <SearchIcon
                        className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
                        aria-hidden="true"
                      />
                      <Input
                        id="api-docs-search"
                        type="search"
                        value={query}
                        onChange={(event) => setQuery(event.target.value)}
                        placeholder="Search by path, method, title, or section"
                        className="pl-8"
                      />
                    </div>
                    {query ? (
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        onClick={() => setQuery("")}
                        aria-label="Clear endpoint search"
                      >
                        <XIcon aria-hidden="true" />
                      </Button>
                    ) : null}
                  </div>
                </Field>
                <p
                  className="mt-2 text-xs text-muted-foreground"
                  aria-live="polite"
                >
                  {normalizedQuery
                    ? `${endpointCount} matching ${
                        endpointCount === 1 ? "endpoint" : "endpoints"
                      }`
                    : `${endpointCount} documented endpoints`}
                </p>
              </div>
            </section>

            {normalizedQuery && endpointCount === 0 ? (
              <div className="rounded-xl bg-card px-4 py-10 text-center ring-1 ring-foreground/10">
                <p className="font-medium">No endpoints match your search</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  Try a method, path segment, or section name.
                </p>
                <Button
                  type="button"
                  variant="outline"
                  className="mt-4"
                  onClick={() => setQuery("")}
                >
                  Clear search
                </Button>
              </div>
            ) : null}

            {visibleSections.map((section) => (
              <section
                key={section.id}
                id={section.id}
                aria-labelledby={`${section.id}-heading`}
                className="scroll-mt-24"
              >
                <h2
                  id={`${section.id}-heading`}
                  className="font-heading text-xl font-semibold"
                >
                  {section.title}
                </h2>
                <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
                  {section.description}
                </p>
                <div className="mt-4 flex flex-col gap-3">
                  {section.endpoints.map((endpoint) => (
                    <ApiEndpointCard key={endpoint.id} endpoint={endpoint} />
                  ))}
                </div>
              </section>
            ))}

            <section
              id="common-behavior"
              aria-labelledby="common-behavior-heading"
              className="scroll-mt-24"
            >
              <h2
                id="common-behavior-heading"
                className="font-heading text-xl font-semibold"
              >
                Common behavior
              </h2>
              <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
                {API_INTRO.errorsNote} All request and response bodies are JSON
                unless the status is 204.
              </p>

              <div className="mt-4 overflow-x-auto rounded-xl bg-card ring-1 ring-foreground/10">
                <table className="w-full min-w-100 border-collapse text-left text-sm">
                  <thead>
                    <tr className="border-b">
                      <th className="px-4 py-2.5 font-medium">Status</th>
                      <th className="px-4 py-2.5 font-medium">Meaning</th>
                    </tr>
                  </thead>
                  <tbody>
                    {COMMON_ERRORS.map((error) => (
                      <tr
                        key={error.status}
                        className="border-b border-border/60 last:border-0"
                      >
                        <td className="px-4 py-2.5 font-mono">
                          {error.status}
                        </td>
                        <td className="px-4 py-2.5 text-muted-foreground">
                          {error.description}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="mt-3">
                <CodeBlock
                  code={'{\n  "error": "validation message"\n}'}
                  label="Error response shape"
                />
              </div>
            </section>
          </div>
        </div>
      </main>
    </div>
  )
}

function matchesSearch(
  endpoint: ApiEndpoint,
  section: ApiSection,
  query: string
): boolean {
  if (query === "") {
    return true
  }

  const searchable = [
    section.title,
    endpoint.method,
    endpoint.path,
    endpoint.title,
    endpoint.summary,
    endpoint.access,
    ...(endpoint.pathParams?.map((param) => param.name) ?? []),
    ...(endpoint.queryParams?.map((param) => param.name) ?? []),
    ...(endpoint.notes ?? []),
    ...(endpoint.errors ?? []),
  ]
    .join(" ")
    .toLowerCase()

  return searchable.includes(query)
}
