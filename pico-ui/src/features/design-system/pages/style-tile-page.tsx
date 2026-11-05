import type { ReactNode } from "react"
import { Link } from "react-router"
import { ArrowLeftIcon, PlusIcon } from "lucide-react"

import { PageHeader } from "@/components/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { usePageTitle } from "@/hooks/use-page-title"
import { cn } from "cn"

const palette = [
  { name: "Background", className: "bg-background", token: "background" },
  { name: "Card", className: "bg-card", token: "card" },
  { name: "Ink", className: "bg-foreground", token: "foreground" },
  { name: "Primary", className: "bg-primary", token: "primary" },
  { name: "Secondary", className: "bg-secondary", token: "secondary" },
  { name: "Accent", className: "bg-accent", token: "accent" },
  { name: "Credit", className: "bg-chart-2", token: "chart-2" },
  { name: "Debit", className: "bg-chart-1", token: "chart-1" },
  { name: "Sidebar", className: "bg-sidebar", token: "sidebar" },
]

export function StyleTilePage() {
  usePageTitle("Design system · Pico")

  return (
    <div className="mx-auto w-full max-w-[1200px] px-4 py-10 sm:px-6 lg:px-10 lg:py-14">
      <div className="mb-10 flex items-center justify-between gap-4">
        <Button
          variant="ghost"
          size="sm"
          className="-ml-2 text-label uppercase"
          nativeButton={false}
          render={<Link to="/" />}
        >
          <ArrowLeftIcon data-icon="inline-start" />
          Back
        </Button>
        <Badge variant="outline">Editorial Brutalist</Badge>
      </div>

      <PageHeader
        eyebrow="Reference"
        title="Design system"
        description="A living tile for the Pico visual language: paper, ink, vermilion, moss, and acid lime. Use these tokens and scales instead of ad-hoc values."
      />

      <div className="mt-14 flex flex-col gap-14">
        <section
          aria-labelledby="palette-heading"
          className="flex flex-col gap-5"
        >
          <SectionTitle id="palette-heading" index="01">
            Palette
          </SectionTitle>
          <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-5">
            {palette.map((swatch) => (
              <div
                key={swatch.token}
                className="overflow-hidden rounded-md border border-border bg-card"
              >
                <div
                  className={cn("h-20", swatch.className)}
                  aria-hidden="true"
                />
                <div className="p-3">
                  <p className="text-sm font-medium">{swatch.name}</p>
                  <p className="mt-0.5 font-mono text-[10px] tracking-wider text-muted-foreground uppercase">
                    {swatch.token}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </section>

        <section aria-labelledby="type-heading" className="flex flex-col gap-5">
          <SectionTitle id="type-heading" index="02">
            Type scale
          </SectionTitle>
          <div className="divide-y divide-border border-y border-border">
            <TypeRow label="Display / 96" sampleClassName="text-display">
              Ledger
            </TypeRow>
            <TypeRow label="Title / 68" sampleClassName="text-title">
              Organizations
            </TypeRow>
            <TypeRow label="Metric / 144" sampleClassName="text-metric">
              $1,637.49
            </TypeRow>
            <TypeRow
              label="Metric fluid / column"
              sampleClassName="text-metric-fluid"
            >
              $11,595.00
            </TypeRow>
            <TypeRow label="Heading / 28" sampleClassName="text-heading">
              Expenses
            </TypeRow>
            <TypeRow label="Label / 11" sampleClassName="text-label uppercase">
              Filter tags (all)
            </TypeRow>
            <TypeRow label="Body / 14" sampleClassName="text-sm">
              Record a debit or credit for this project.
            </TypeRow>
          </div>
        </section>

        <section
          aria-labelledby="components-heading"
          className="flex flex-col gap-5"
        >
          <SectionTitle id="components-heading" index="03">
            Components
          </SectionTitle>
          <div className="grid gap-8 rounded-lg border border-border bg-card p-6 lg:grid-cols-2">
            <div className="flex flex-col gap-4">
              <p className="text-label text-muted-foreground uppercase">
                Actions
              </p>
              <div className="flex flex-wrap gap-2">
                <Button type="button">
                  <PlusIcon data-icon="inline-start" />
                  Primary
                </Button>
                <Button type="button" variant="secondary">
                  Secondary
                </Button>
                <Button type="button" variant="outline">
                  Outline
                </Button>
                <Button type="button" variant="ghost">
                  Ghost
                </Button>
              </div>
              <div className="flex flex-wrap gap-2">
                <Badge>Owner</Badge>
                <Badge variant="secondary">Credit</Badge>
                <Badge variant="outline">travel</Badge>
              </div>
            </div>

            <div className="flex flex-col gap-4">
              <p className="text-label text-muted-foreground uppercase">
                Field
              </p>
              <Field>
                <FieldLabel
                  htmlFor="style-tile-amount"
                  className="text-label text-muted-foreground uppercase"
                >
                  Amount
                </FieldLabel>
                <Input id="style-tile-amount" placeholder="12.99" />
              </Field>
              <div className="flex flex-col gap-2">
                <span className="text-label text-muted-foreground uppercase">
                  Tags
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {["travel", "food", "team"].map((tag) => (
                    <span
                      key={tag}
                      className="inline-flex items-center rounded-sm border border-border px-1.5 py-0.5 text-[10px] font-medium tracking-[0.12em] text-muted-foreground uppercase"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </section>

        <section
          aria-labelledby="patterns-heading"
          className="flex flex-col gap-5"
        >
          <SectionTitle id="patterns-heading" index="04">
            Patterns
          </SectionTitle>

          <div className="relative overflow-hidden rounded-lg border border-border bg-card">
            <div
              className="absolute inset-y-0 left-0 w-1 bg-primary"
              aria-hidden="true"
            />
            <div className="grid gap-8 p-6 sm:p-8 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
              <div>
                <p className="text-label text-muted-foreground uppercase">
                  Net total
                </p>
                <p className="mt-3 font-display text-metric tabular-nums">
                  $1,637.49
                </p>
                <p className="mt-3 font-mono text-[11px] tracking-[0.16em] text-muted-foreground uppercase">
                  3 expenses · USD
                </p>
              </div>
              <div className="grid grid-cols-2 gap-8 border-t border-border pt-6 sm:gap-12 lg:border-t-0 lg:border-l lg:pt-0 lg:pl-8">
                <div>
                  <p className="text-label text-muted-foreground uppercase">
                    Credit
                  </p>
                  <p className="mt-2 font-display text-2xl font-semibold text-chart-2 tabular-nums">
                    $5,000.00
                  </p>
                </div>
                <div>
                  <p className="text-label text-muted-foreground uppercase">
                    Debit
                  </p>
                  <p className="mt-2 font-display text-2xl font-semibold text-primary tabular-nums">
                    $3,362.51
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div className="border-y border-border">
            <ul className="divide-y divide-border">
              {[
                {
                  type: "Debit",
                  description: "Team offsite travel",
                  meta: "Oct 28, 2026 · 6:00 PM",
                  amount: "-$1,299.00",
                  tone: "text-primary",
                },
                {
                  type: "Credit",
                  description: "Client payment",
                  meta: "Nov 1, 2026 · 9:00 AM",
                  amount: "+$5,000.00",
                  tone: "text-chart-2",
                },
              ].map((row) => (
                <li
                  key={row.description}
                  className="flex flex-col gap-3 py-5 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex flex-col gap-2">
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                      <span
                        className={cn(
                          "inline-flex items-center gap-1.5 text-label uppercase",
                          row.tone
                        )}
                      >
                        <span
                          className={cn("size-1.5 rounded-full bg-current")}
                          aria-hidden="true"
                        />
                        {row.type}
                      </span>
                      <p className="font-display text-xl font-semibold tracking-tight">
                        {row.description}
                      </p>
                    </div>
                    <p className="font-mono text-[11px] tracking-wide text-muted-foreground uppercase">
                      {row.meta}
                    </p>
                  </div>
                  <span
                    className={cn(
                      "font-display text-2xl font-semibold tracking-tight tabular-nums",
                      row.tone
                    )}
                  >
                    {row.amount}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        </section>
      </div>
    </div>
  )
}

function SectionTitle({
  id,
  index,
  children,
}: {
  id: string
  index: string
  children: ReactNode
}) {
  return (
    <div className="flex items-baseline gap-3 border-b border-border pb-3">
      <span className="font-mono text-[10px] tracking-wider text-primary">
        {index}
      </span>
      <h2 id={id} className="font-display text-heading">
        {children}
      </h2>
    </div>
  )
}

function TypeRow({
  label,
  sampleClassName,
  children,
}: {
  label: string
  sampleClassName: string
  children: ReactNode
}) {
  return (
    <div className="grid gap-2 py-5 sm:grid-cols-[10rem_minmax(0,1fr)] sm:items-baseline">
      <p className="font-mono text-[10px] tracking-wider text-muted-foreground uppercase">
        {label}
      </p>
      <div className={cn("@container font-display", sampleClassName)}>
        {children}
      </div>
    </div>
  )
}
