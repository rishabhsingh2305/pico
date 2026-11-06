# Engineering standards

These rules apply to the pico-ui codebase. Keep them short and update them as the
app grows.

## Principles

- **KISS:** choose the simplest design that satisfies the current requirement.
- **YAGNI:** do not build for hypothetical pages, state, or API layers.
- **DRY, with the rule of three:** extract shared code after the third real
  repetition, not the first.
- **Single responsibility:** each module, component, and function has one reason
  to change.
- **Explicit over clever:** readable code beats compact code.
- **Composition over inheritance:** small components, hooks, and plain functions.

## TypeScript

- `strict` is enabled. Do not use `any`. Prefer `unknown` at boundaries and
  narrow it.
- Use `import type` for type-only imports (`verbatimModuleSyntax`).
- `erasableSyntaxOnly` is enabled: do not add TypeScript enums or parameter
  properties.
- Name types and components in PascalCase; functions and variables in camelCase.
- Export named components and helpers. Avoid default exports except the root
  `App` required by the entrypoint.
- Keep API types in the feature's `types.ts`. Do not duplicate response shapes in
  components.
- Prefer explicit return types on exported functions when the inferred type is
  not obvious.
- Avoid type assertions unless the value is validated, such as JSON parsed by
  the API client.

## React

- Use function components. Keep components focused on rendering and user events.
- Business rules, validation, and HTTP belong outside JSX.
- Derive values during render instead of storing them in state. Use effects only
  to synchronize with external systems.
- Keep hooks at the top level. Do not call hooks conditionally.
- Give every list item a stable key.
- Keep props small and typed. Pass data and callbacks rather than global state.
- Use local state first. Introduce context or a store only after a repeated
  cross-tree need.
- Do not add `useMemo` or `useCallback` without a measured or structural reason.
- Forms use controlled inputs and a pure `validate` function. Server errors are
  rendered in an alert and cleared on the next edit.
- Expense amounts are positive decimal values converted to minor units; the
  debit/credit type carries direction. Debit means money out, credit means money
  in. The form defaults to debit and always sends the selected type.
- Tag entry uses removable badges. Comma or Enter commits the draft, blur
  commits a pending draft, duplicates are ignored, and removal has an accessible
  label. Keep the committed tags as an array in form state. Use the shared
  `TagInput` and tag limits from `src/lib/tags.ts` for organizations, projects,
  and expenses.
- Project and organization tag filters run on the server with any-match
  semantics. Expense tag filters require all tags. Document and test that
  difference.
- Date filters use the shared `DateRangePicker` with presets and a custom range
  calendar. It emits local `yyyy-MM-dd` strings; keep RFC 3339 conversion in the
  feature filter parser. Presets include today, and “This month” ends today.

## Structure and boundaries

- `src/app/` owns routing and app composition.
- `src/features/<feature>/` owns types, API, validation, components, and pages.
- `src/lib/` contains framework-free utilities and the API client. It must not
  import features or components.
- `src/components/ui/` contains generated shadcn primitives. Add with the
  shadcn CLI. Do not hand-edit except for generated-code fixes.
- `src/hooks/` contains hooks shared by multiple features.
- Do not create barrel files unless they remove real repetition.
- Do not create a generic abstraction for one page.

## API and errors

- All HTTP goes through `requestJSON` in `src/lib/api-client.ts`.
- Feature `api.ts` modules define request functions; components call those
  functions.
- Parse JSON and throw `ApiError` with the HTTP status and a safe message.
- Map known statuses to user-facing text in the feature's `error-message.ts`.
- Never display raw server errors, stack traces, or request internals.
- Never log passwords, tokens, or full request bodies.
- Use the backend's error envelope `{"error":"message"}`.
- For paginated lists with totals, render the server's `totalCount`,
  `totalDebit`, `totalCredit`, and `netTotal` for the full filtered set.
  `netTotal` is `totalCredit - totalDebit`; never sum only the loaded page.
- Changing filters resets `offset` to zero; changing pages must not change the
  totals. Reload the current page after create or delete so the totals stay
  correct.
- Dashboard and workspace summaries must group minor-unit totals by currency.
  Never add amounts across currencies. Fetch per-project totals from the API
  rather than summing returned page items.
- Keep `/api-docs` content in sync with the backend contract. The backend README
  is canonical; `src/features/api-docs/content.ts` is the curated UI guide.

## Forms and accessibility

- Every input has a visible label and a stable `id`. Radio groups use
  `FieldSet`/`FieldLegend` and keep their options labeled.
- Set `autoComplete` values (`name`, `email`, `new-password`).
- Set `aria-invalid` on invalid fields and connect errors with
  `aria-describedby`.
- Show errors with `role="alert"` or the existing `Alert` component.
- Keep the submit button disabled while a request is in flight and expose
  loading text.
- Support keyboard-only submission and visible focus.
- Respect reduced motion when adding animation.
- Put occasional create/edit forms in the shadcn `Dialog` with a
  `DialogTitle`, `DialogDescription`, and clear Cancel action. Keep the trigger
  focusable and return focus to it when the dialog closes.
- Check the page at mobile and desktop widths and in light and dark themes.

## Styling

- Use Tailwind utilities and the shadcn theme tokens.
- Use `cn` for conditional classes.
- Use Bricolage Grotesque through `font-display` for page titles, metrics, and
  section headings. Keep Inter for body copy, forms, and controls.
- Use the shared type scale (`text-display`, `text-title`, `text-metric`,
  `text-metric-fluid`, `text-heading`, `text-label`) instead of ad-hoc display
  sizes. `text-metric-fluid` is container-relative and is reserved for the
  project ledger net so it can fill its column without overflowing.
- Do not add hex/rgb colors outside the theme. Use `background`, `foreground`,
  `muted`, `primary`, `accent`, `chart-*`, `sidebar-*`, and related tokens.
- Credit values use `chart-2` (moss); debit values use `primary` (vermilion). Do
  not encode direction with arbitrary greens or reds.
- Keep the `/design` style tile in sync when the palette or type scale changes.
- Prefer the existing `Button`, `Card`, `Field`, `Input`, `Alert`, and `Spinner`
  primitives.
- Keep secondary list filters compact: one toolbar row on desktop with short
  labels, without a redundant card heading or helper copy.
- Keep layout responsive with `max-w-*`, `flex`, and `min-h-svh`.
- No CSS-in-JS and no second component library.

## Routing

- Register routes only in `src/app/router.tsx`.
- Keep `BrowserRouter` in `main.tsx`; tests use `MemoryRouter`.
- Redirects are explicit routes, currently `/` to `/orgs` or `/login` and `*`
  through `/`. Protected routes render inside `RequireSession`.
- When adding a page, add a route test and update `docs/architecture.md`.
- Static hosting must rewrite unknown paths to `index.html`.

## Configuration and security

- Only `VITE_*` variables may be read in browser code. They are public.
- Keep variable names and defaults in `src/lib/env.ts`.
- Keep `.env.example` in sync. Never commit `.env`, `.env.local`, tokens, or
  passwords.
- Treat all client validation as UX only; the backend is authoritative.
- The authenticated session lives in `AuthProvider` and is restored from
  `/auth/session`. The backend sets an HttpOnly `pico_auth` cookie; never read,
  store, or log tokens in JavaScript, and never put them in `localStorage`.

## Testing

- Every behavior change and bug fix gets a test.
- Use Vitest + React Testing Library. Colocate `*.test.ts` / `*.test.tsx` files.
- Test behavior through labels, roles, and visible text, not internal state.
- Mock `fetch`; never use a real network or external service.
- Cover success, validation, loading, and error paths.
- Keep tests deterministic: no sleeps or time-dependent assertions.
- Use `MemoryRouter` for route tests and `userEvent` for interactions.
- Run `corepack pnpm test` before declaring work done.

## Dependencies

- Prefer the existing stack: React, React Router, Tailwind, shadcn, Vitest, and
  Testing Library.
- Use pnpm via Corepack. Do not mix npm or yarn.
- Justify a new dependency and verify it is maintained.
- Update `package.json` and the lockfile with pnpm, never by hand.
- Run `format:check`, `typecheck`, `lint`, `test`, and `build` after dependency
  changes.

## Change checklist

- [ ] The change is scoped to one concern.
- [ ] Types, API calls, and validation live outside JSX.
- [ ] Loading, success, and error states are covered.
- [ ] Accessibility attributes and keyboard behavior are present.
- [ ] Light and dark themes and mobile widths were checked.
- [ ] Tests were added or updated.
- [ ] `format:check`, `typecheck`, `lint`, `test`, and `build` pass.
- [ ] `README.md` and `docs/architecture.md` are updated for contract changes.
- [ ] No secrets, environment files, `dist/`, or unrelated generated changes.

## References

- [React documentation](https://react.dev/)
- [TypeScript handbook](https://www.typescriptlang.org/docs/)
- [Vite documentation](https://vite.dev/)
- [Tailwind CSS documentation](https://tailwindcss.com/docs)
- [shadcn/ui documentation](https://ui.shadcn.com/)
- [React Router documentation](https://reactrouter.com/)
- [Vitest documentation](https://vitest.dev/)
- [Testing Library documentation](https://testing-library.com/)
- [WAI-ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
