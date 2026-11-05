import { Badge } from "@/components/ui/badge"
import { CodeBlock } from "@/features/api-docs/components/code-block"
import { CopyButton } from "@/features/api-docs/components/copy-button"
import type {
  ApiEndpoint,
  ApiMethod,
  ApiParameter,
} from "@/features/api-docs/types"

type ApiEndpointCardProps = {
  endpoint: ApiEndpoint
}

export function ApiEndpointCard({ endpoint }: ApiEndpointCardProps) {
  const titleId = `${endpoint.id}-title`

  return (
    <article
      id={endpoint.id}
      aria-labelledby={titleId}
      className="scroll-mt-24 rounded-xl bg-card p-4 ring-1 ring-foreground/10 sm:p-5"
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={methodVariant(endpoint.method)}>
          {endpoint.method}
        </Badge>
        <code className="font-mono text-xs break-all text-muted-foreground">
          {endpoint.path}
        </code>
        <CopyButton
          value={endpoint.path}
          label={`Copy ${endpoint.method} ${endpoint.path}`}
        />
      </div>

      <div className="mt-3 flex flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <h3 id={titleId} className="font-heading text-base font-semibold">
            {endpoint.title}
          </h3>
          <Badge variant="outline">{endpoint.access}</Badge>
        </div>
        <p className="text-sm text-muted-foreground">{endpoint.summary}</p>
      </div>

      {endpoint.pathParams?.length || endpoint.queryParams?.length ? (
        <div className="mt-4 flex flex-col gap-3">
          {endpoint.pathParams?.length ? (
            <ParameterTable
              title="Path parameters"
              params={endpoint.pathParams}
            />
          ) : null}
          {endpoint.queryParams?.length ? (
            <ParameterTable
              title="Query parameters"
              params={endpoint.queryParams}
            />
          ) : null}
        </div>
      ) : null}

      {endpoint.requestBody ? (
        <div className="mt-4">
          <CodeBlock code={endpoint.requestBody} label="Request body" />
        </div>
      ) : null}

      <div className="mt-4">
        <div className="mb-2 flex items-center gap-2">
          <Badge variant="secondary">{endpoint.response.status}</Badge>
          <p className="text-sm text-muted-foreground">
            {endpoint.response.description}
          </p>
        </div>
        {endpoint.response.body ? (
          <CodeBlock
            code={endpoint.response.body}
            label={`${endpoint.response.status} response`}
          />
        ) : null}
      </div>

      {endpoint.errors?.length ? (
        <div className="mt-4">
          <h4 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Errors
          </h4>
          <ul className="mt-1.5 list-disc space-y-1 pl-5 text-sm text-muted-foreground">
            {endpoint.errors.map((error) => (
              <li key={error}>{error}</li>
            ))}
          </ul>
        </div>
      ) : null}

      {endpoint.notes?.length ? (
        <div className="mt-4">
          <h4 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Notes
          </h4>
          <ul className="mt-1.5 list-disc space-y-1 pl-5 text-sm text-muted-foreground">
            {endpoint.notes.map((note) => (
              <li key={note}>{note}</li>
            ))}
          </ul>
        </div>
      ) : null}
    </article>
  )
}

function ParameterTable({
  title,
  params,
}: {
  title: string
  params: ApiParameter[]
}) {
  return (
    <div>
      <h4 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
        {title}
      </h4>
      <div className="mt-1.5 overflow-x-auto">
        <table className="w-full min-w-80 border-collapse text-left text-xs">
          <thead>
            <tr className="border-b">
              <th className="py-1.5 pr-3 font-medium">Name</th>
              <th className="py-1.5 pr-3 font-medium">Required</th>
              <th className="py-1.5 font-medium">Description</th>
            </tr>
          </thead>
          <tbody>
            {params.map((param) => (
              <tr
                key={param.name}
                className="border-b border-border/60 last:border-0"
              >
                <td className="py-1.5 pr-3 font-mono text-foreground">
                  {param.name}
                </td>
                <td className="py-1.5 pr-3 text-muted-foreground">
                  {param.required ? "Yes" : "No"}
                </td>
                <td className="py-1.5 text-muted-foreground">
                  {param.description}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function methodVariant(
  method: ApiMethod
): "default" | "secondary" | "destructive" | "outline" {
  switch (method) {
    case "POST":
    case "PATCH":
      return "default"
    case "DELETE":
      return "destructive"
    case "HEAD":
      return "outline"
    default:
      return "secondary"
  }
}
