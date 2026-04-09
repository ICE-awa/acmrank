# ACMRank Agent Guide

## 1. Purpose
- This file defines the default working rules for Codex and other coding agents in this repository.
- Treat it as the execution baseline unless the user gives a newer explicit instruction.
- If this file, `docs/`, and the latest user instruction conflict, use this priority:
  1. Latest user instruction
  2. `AGENTS.md`
  3. `docs/`

## 2. Startup Checklist
- Confirm the working directory is `/opt/acmrank`.
- Do not assume the old path `/mnt/f/archive/acmrank` is still valid or up to date.
- Before implementing anything, read these files in order:
  1. `docs/PRD.md`
  2. `docs/technical-architecture.md`
  3. `docs/platform-data-integration.md`
  4. `docs/atcoder-extension-sync.md`
  5. `docs/task-breakdown.md`
- If the docs and the current confirmed requirements conflict, update the docs first, then continue.
- If the task is part of the full project build-out, select the current implementation target from `docs/task-breakdown.md` before writing code.
- In user-facing replies, first state your current understanding, then implement.
- If anything important is ambiguous, ask the user early instead of guessing on high-risk behavior.

## 3. Product Truths That Must Stay Stable
- ACMRank is an internal competition archive and training leaderboard system for South China Normal University.
- Public user pages are visible to everyone.
- Public user pages show:
  - aggregated solved problems from `Codeforces / AtCoder / 洛谷`
  - platform aggregate views
  - single-account views
  - `ICPC` award history
- The main user view shows only `AC` lists, not full submission history.
- Personal pages must show:
  - an `SCNU Rating` line chart
  - a GitHub-like contribution heatmap
- The heatmap measures `daily newly accepted problem count`.
- `SCNU Rating` is computed only from `AtCoder + Codeforces`.
- `洛谷` and `ICPC` do not participate in `SCNU Rating`.
- `Clist problem.rating` is the problem weight source for `SCNU Rating`.
- `Clist` is not the source of truth for whether a problem was accepted.
- `Codeforces` main chain is the official API, with `Clist` as fallback.
- `AtCoder` is a multi-level fallback chain:
  1. Main chain: operator-managed `AtCoder` login and long-lived `Cookie`
  2. First fallback: `Clist`
  3. Second fallback: third-party API
  4. Last fallback: user browser extension window-based sync
- The `AtCoder` user extension is not the main chain.
- The `AtCoder` user extension uploads only accepted records inside the sync window, not all submissions.
- The default sync window is `last successful sync time -> current trigger time`.
- The server merges these `AtCoder` accepted records into:
  - solved problem list
  - `first_ac_at`
  - per-contest accepted problem summaries
- `AtCoder` main chain failure, `Clist` failure, and third-party API failure all require proactive alerting.
- `PostgreSQL` is the only source of truth.
- `Redis` is only for sessions, rate limits, cache, and short-lived state.
- `NATS + JetStream` is only for async tasks and events.
- `ElasticSearch` is only for derived indexes and search enhancement.
- Python is only for sync and crawling, not core business rules.

## 4. Core Engineering Principles
- Prefer clear, testable, incremental changes over clever shortcuts.
- Do not continue coding on top of contradictory product assumptions.
- Do not hide uncertainty. Surface it and ask.
- No change is complete without verification.
- If behavior changes, tests and docs must change with it.
- Never claim a command or test passed unless you actually ran it successfully.

## 5. Backend Architecture Rules

### 5.1 Three-Layer Rule
- The backend must use a three-layer architecture:
  - `handler`
  - `service`
  - `repository`
- Dependency direction must remain:
  - `handler -> service -> repository`
- Reverse dependencies are not allowed.
- `DTO` types are boundary objects, not business objects.
- `DTO` types must not leak into `service` or `repository` logic.
- Business rules belong in `service`, not in `handler`, `repository`, SQL glue, or Python sync scripts.
- Persistence details belong in `repository`, not in `handler` or `service`.

### 5.2 Dependency Injection
- Use constructor-based dependency injection.
- Wire dependencies in the application bootstrap or composition root.
- Avoid package-level mutable globals and hidden singletons.
- Prefer depending on interfaces at the service boundary when it improves testability.
- Keep transaction boundaries explicit. They should usually live in the service layer, not in handlers.

### 5.3 Recommended Backend Package Layout
- Use versioned API boundary packages such as:
  - `dto/v1`
  - `dto/v2`
  - `handler/v1`
  - `handler/v2`
- Keep shared business logic below the handler layer, for example in packages like:
  - `service`
  - `repository`
  - `domain` or `model` for shared business types if needed
- `domain` or `model` packages may exist, but they are shared support packages, not an extra dependency-inverting layer.

### 5.4 API Versioning Rules
- Version request and response contracts through versioned DTO and handler packages.
- Do not mix `v1` DTOs into `v2` handlers or the reverse.
- If a request or response change is breaking, create a new API version instead of silently mutating the old one.
- Prefer sharing service and repository logic across versions when the business behavior is actually the same.
- Route versioning should be explicit and predictable, for example `/api/v1/...` and `/api/v2/...`.

### 5.5 Data and Domain Rules
- `PostgreSQL` is the source of truth for users, platform accounts, accepted facts, `first_ac_at`, rating snapshots, and leaderboard state.
- `Redis` and `ElasticSearch` must never become the only storage location for business facts.
- `first_ac_at` must be stored explicitly and treated as a first-class fact.
- Do not move core rating logic out of Go backend services into Python sync workers.

## 6. Frontend Rules
- The default frontend toolchain is:
  - `pnpm`
  - `vitest`
  - `eslint`
  - `prettier`
  - `playwright`
- Keep route modules or pages thin.
- Put reusable UI into components and reusable behavior into hooks or client services.
- Do not scatter raw API contract knowledge across many components.
- Use typed request and response models where possible.
- Keep public profile, leaderboard, and account management behavior aligned with the product truths above.
- Do not redesign the UI around full submission history when the product only exposes accepted lists.

## 7. Testing And Verification

### 7.1 General Rule
- Every code change must include the right level of tests.
- Every completed change must include actual verification commands.
- If tooling is missing, broken, or not yet scaffolded, state that explicitly and do not pretend verification happened.

### 7.2 Backend Mandatory Checks
If backend code is modified, run the relevant Go checks before handoff. At minimum:
- `go fmt ./...`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `go test -race ./...`
- `golangci-lint run ./...`

Additional rules:
- During iteration, you may run narrower package-level tests first.
- Before final handoff, run repo-level checks when the repo structure supports them.
- If the repo later uses `Makefile`, `mage`, or task runners for these checks, prefer the project-standard entrypoints only if they cover the same gates.

### 7.3 Frontend Mandatory Checks
If frontend code is modified, use `pnpm` as the package manager and run the project-standard commands for:
- `prettier`
- `eslint`
- type checker
- `vitest`
- `playwright`
- build

Practical rule:
- Frontend package management is fixed to `pnpm`.
- Unit tests are fixed to `vitest`.
- Formatting is fixed to `prettier`.
- Linting is fixed to `eslint`.
- Browser e2e and regression tests are fixed to `playwright`.
- Prefer repository scripts such as `pnpm lint`, `pnpm format:check`, `pnpm typecheck`, `pnpm test`, `pnpm test:e2e`, `pnpm build` when they exist.
- If scripts do not exist yet, add or document the missing scripts instead of silently skipping checks.
- If a frontend change affects user flows, pages, routing, forms, auth, or client-server integration, run the relevant `Playwright` coverage too.

### 7.4 Test Layer Requirements
- After writing or changing an API handler, add or update unit tests in the same change.
- After a feature is functionally complete, add integration tests for the end-to-end backend flow.
- If you fix a bug or change an existing behavior, add a regression test that would have caught it.
- Favor targeted tests close to the changed code, but do not skip broader verification when the change crosses boundaries.

### 7.5 Suggested Test Scope By Layer
- `handler` tests:
  - request parsing
  - validation
  - auth or permission gates
  - status codes
  - response mapping
- `service` tests:
  - business rules
  - orchestration
  - transaction behavior
  - branch coverage for normal and edge paths
- `repository` tests:
  - query behavior
  - persistence correctness
  - migration-sensitive behavior when applicable
- frontend unit tests:
  - component behavior
  - hooks
  - client-side state transitions
- browser e2e tests:
  - critical user journeys
  - page integration behavior
  - UI regressions across real browser flows
- integration tests:
  - cross-layer flows
  - API contract behavior
  - persistence and queue interactions where relevant
- regression tests:
  - any bug fix
  - any previously broken edge case

## 8. Documentation Rules
- If you change product behavior, update the relevant files under `docs/`.
- If you change architecture or workflow conventions, update `AGENTS.md`.
- If you add a new API version, document the versioning and compatibility expectations.
- Do not leave behavior changes undocumented when they affect future implementation decisions.

## 9. Git Workflow Expectations
- Follow a Git Flow style workflow.
- Treat `dev` as the default integration branch unless the user explicitly says otherwise.
- Do not implement task work directly on long-lived branches such as `main`, `master`, or `dev`.
- Start each new task from the latest `dev` state on a new branch.
- Exception:
  - very small documentation-only fixes may be committed directly on `dev`
  - very small maintenance work expected to take only one or two commits may also stay on `dev` if the user prefers not to open a separate branch
- For anything beyond those narrow exceptions, still create a dedicated task branch from `dev`.
- If `dev` does not exist yet or the branch model is unclear, stop and ask the user before inventing a different workflow.
- Use branch names that describe the task type, for example:
  - `feature/<topic>`
  - `fix/<topic>`
  - `docs/<topic>`
  - `refactor/<topic>`
  - `chore/<topic>`
  - `test/<topic>`
- Do not rewrite branch history unless the user explicitly asks for it.
- Do not force-push unless the user explicitly approves it.

### 9.1 Commit And Push Requirements
- At the end of each completed turn, the agent must generate a Git commit message with both:
  - a subject line
  - a non-empty body
- The commit body must use real paragraph breaks.
- Do not rely on literal `\n` escape sequences inside a shell string and assume GitHub will render them as newlines.
- Prefer one of these safe approaches:
  - multiple `-m` flags
  - a temporary commit message file
  - a quoted multi-line message that contains actual newline characters
- After the work for that turn is complete, the agent must automatically:
  - `git add`
  - `git commit`
  - `git push`
- This applies after the required verification for the changed area has been run.
- If a required check cannot run, the agent must state that explicitly, then still use a commit body that records the limitation.
- Do not leave completed work only in the working tree unless the user explicitly asks not to commit.

### 9.2 Pull Request Handoff
- After pushing a completed task branch, explicitly tell the user that the branch is ready and that a PR should now be opened against `dev`.
- The user will create the PR on GitHub, review the code, and provide review feedback.
- The agent should not merge the PR on the user's behalf unless explicitly asked.

### 9.3 Review Iteration Rules
- When the user returns with PR review comments, continue working on the same task branch unless the user explicitly asks to change branches.
- Fix the review issues, rerun the relevant verification, then again:
  - `git add`
  - `git commit` with subject and body
  - `git push`
- After the review-fix branch is pushed, explicitly tell the user that the PR is ready for another review pass.

### 9.4 Post-Merge Responsibility Split
- After the user merges the PR, the user will manually switch back to `dev` and pull the latest code.
- The next task should start only after that refresh step has happened.
- Do not assume the local branch is ready for the next task until the user indicates that `dev` is current again.

### 9.5 Final Response Requirements For Git Work
- When a turn ends with committed work, include:
  - the branch name
  - the commit subject
  - a short verification summary
  - an explicit note telling the user whether a PR should now be opened or re-reviewed

## 10. Implementation Workflow Expectations
- Start by restating the task in concrete terms.
- Read the current repo state before deciding on structure.
- Make the smallest coherent change that solves the problem.
- Prefer explicit constructors, explicit interfaces, and explicit tests over hidden magic.
- Finish the change end-to-end when feasible:
  - implementation
  - tests
  - verification
  - doc updates if needed

## 11. Communication Expectations
- Be concise, direct, and factual.
- State assumptions clearly.
- If a requirement is unclear, ask instead of inventing behavior.
- If verification could not be completed, say exactly what was not run and why.
- If the repo lacks expected tooling, say so and either add the missing scaffolding or ask before making a broader tooling decision.

## 12. Current Repository Reality
- The repository may be in an early or partially migrated state.
- If expected files such as `go.mod`, `package.json`, lint configs, or test configs are absent, do not assume the final toolchain is already present.
- In that case:
  - preserve the architecture and testing rules in new code
  - scaffold missing pieces carefully when needed
  - report any missing verification tooling explicitly
