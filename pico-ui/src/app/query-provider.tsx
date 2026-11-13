import { QueryClientProvider } from "@tanstack/react-query"
import { useState, type ReactNode } from "react"

import { createQueryClient } from "@/app/query-client"

type QueryProviderProps = {
  children: ReactNode
}

export function QueryProvider({ children }: QueryProviderProps) {
  const [client] = useState(createQueryClient)

  return <QueryClientProvider client={client}>{children}</QueryClientProvider>
}
