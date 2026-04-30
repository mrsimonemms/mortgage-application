# Claude Instructions

This repository contains a Temporal Proof of Concept for a mortgage orchestration
demonstration.

The project is intentionally split into three applications:

- `apps/worker`: Go Temporal worker
- `apps/api`: NestJS API
- `apps/ui`: SvelteKit UI

The goal of this repository is to demonstrate:

- Durable orchestration
- Asynchronous coordination
- Auditability
- Observability
- Resilience and compensation
- Safe workflow evolution

Claude should treat this repository as a small multi-service orchestration system,
not as a generic full-stack app and not as a production platform.

---

## Project intent

This project exists to demonstrate Temporal capabilities in a realistic
enterprise-style mortgage process.

The implementation should prioritise:

- Clarity
- Correctness
- Demo reliability
- Inspectable behaviour
- Minimal moving parts

This is a Proof of Concept, so the focus is on clearly demonstrating workflow
behaviour rather than building a fully generalised product.

Do not over-engineer the solution.

Prefer:

- Simple, explicit behaviour
- Small surface area
- Readable code
- Predictable demo flows

Avoid:

- Unnecessary abstractions
- Premature extensibility
- Infrastructure that distracts from the Temporal story
- Architecture that implies Temporal depends on external coordination systems

---

## Architectural overview

This repository is structured as a monorepo with three application boundaries.

### `apps/worker`

This is the orchestration engine layer.

Responsibilities:
- Temporal workflows
- Temporal activities
- Workflow state transitions
- Signals, queries, retries, compensation
- Versioning and safe workflow evolution

The worker is the authoritative home of orchestration logic.

Do not move orchestration decisions into the API or UI.

### `apps/api`

This is the control and ingress layer.

Responsibilities:
- Starting workflows
- Sending signals to workflows
- Querying workflow state
- Exposing operator actions
- Providing a stable interface for the UI

The API should be thin and explicit.

It must not become the orchestration brain.

### `apps/ui`

This is the presentation layer.

Responsibilities:
- Displaying workflow/application state
- Showing audit timeline and current status
- Triggering operator actions through the API
- Supporting a clear live demo

The UI should optimise for clarity and speed, not product completeness.

### Temporal

Temporal is an existing platform dependency, not an app in this repository.

Claude should treat Temporal as the orchestration runtime and execution backbone,
not as something to re-implement or abstract away.

---

## Technology stack (authoritative)

- **Worker:** Go
- **API:** NestJS / TypeScript
- **UI:** SvelteKit / TypeScript
- **Orchestration runtime:** Temporal
- **Local orchestration environment:** Docker Compose
- **Transport between UI and backend:** HTTP via API
- **Async coordination:** Temporal signals and workflow primitives

Do not introduce:

- External event buses unless explicitly requested
- Additional orchestration layers
- Hidden coordination mechanisms outside Temporal
- Unnecessary shared runtime dependencies between apps

---

## Core architectural principles

- Temporal is the orchestration engine
- The worker owns orchestration behaviour
- The API is an ingress and control surface
- The UI is a visualisation and interaction layer
- Clarity beats cleverness
- Explicitness beats abstraction
- Demo reliability beats theoretical elegance

Prefer:
- Small, focused services
- Explicit JSON contracts
- Clear state transitions
- Observable behaviour
- Straightforward deployment and startup

Avoid:
- Generic frameworks inside the app
- "Platform" abstractions that add little value
- Clever cross-app coupling
- Heavy shared libraries for trivial concerns

---

## Contract philosophy

Contracts in this repository are intentionally simple.

Use manual JSON contracts and keep them small, stable, and explicit.

Claude should assume:
- Cross-language contracts are maintained manually
- JSON payloads must remain easy to inspect
- The API contract is more important than internal model purity

Prefer:
- Flat request and response payloads
- Consistent field naming
- Small DTOs
- Explicit status values

Avoid:
- Over-modelling the mortgage domain
- Schema tooling unless explicitly requested
- Contract generation pipelines
- Deeply nested payloads without strong reason

Use `applicationId` consistently in JSON.

For Go structs, use JSON tags to preserve API field naming.

---

## Workflow design rules

Workflows must clearly demonstrate the PoC scenarios.

They should support:

- Happy path execution
- Durable waiting for async completion
- Correlation using application identifiers
- Failure injection
- Compensation
- Safe replay / re-run
- Versioned evolution of the workflow

Workflow code must remain deterministic.

Do not:
- Use non-deterministic APIs inside workflows
- Read wall clock time directly inside workflows
- Introduce hidden side effects
- Make workflow logic depend on API or UI state

Keep the workflow story obvious.

A reader should be able to understand:
- what step is running
- what state the application is in
- what caused a pause
- what caused a retry
- what caused a failure
- what compensation occurred
- what version of the workflow is active

---

## Resilience and demonstration intent

This PoC is for a bank audience and should demonstrate production-grade thinking
about failure, recovery, and durability. Resilience is not a secondary concern.

### Default expectations

Workflows should succeed wherever reasonably possible. The happy path is the
primary demo scenario and must be reliable.

Failures are expected events, not exceptional ones. The system should handle them
gracefully and recover without manual intervention where Temporal makes that
possible.

### Use real Temporal mechanisms

When demonstrating resilience, use Temporal's actual capabilities, not simulated
outcomes in workflow control flow.

Prefer:
- Real activity execution, not conditional branches that skip the activity
- Real activity failures using `temporal.NewApplicationError` or
  `temporal.NewNonRetryableApplicationError`, not error returns from workflow code
- Temporal-managed retries via `RetryPolicy`, not manual retry loops
- Compensation via a real compensating activity, not a status flag set in the
  workflow without executing the reverse operation

Avoid:
- Simulating failure outcomes by branching in workflow code before calling the
  activity
- Returning a synthetic error from workflow code to mimic an activity failure
- Treating failure injection as a workflow-level conditional, rather than as
  activity-level behaviour

### Why this matters

Temporal's value proposition is that it durably manages execution, retries, and
state across failures. Demonstrating failure injection inside workflow control
flow misrepresents this. A viewer would reasonably conclude that Temporal requires
the developer to manage retry logic manually.

The correct pattern is: the activity fails, Temporal retries it, the workflow
sees the final result. The workflow should not need to know how many attempts were
made.

### Retry and compensation rules

- Configure `RetryPolicy` on the activity options, not in the workflow logic.
- Use `MaximumAttempts` to bound retries for demo predictability.
- If an activity is non-retryable, use `NewNonRetryableApplicationError` in the
  activity itself, not a workflow-level branch.
- If compensation is required after a failure, execute the compensating activity.
  Do not just update the workflow status without performing the reverse operation.

### Failure injection in demo scenarios

Controlled failure injection for demo purposes is acceptable and expected.

When implementing a failure scenario:
- Put the failure logic in the activity, keyed on the attempt count or an input
  flag passed from the workflow.
- Make the failure obvious and intentional. Add a log line that names the
  scenario.
- Prefer attempt-based failure (fail on attempts 1 to N, succeed on attempt N+1)
  over a permanent failure scenario, unless the scenario is specifically about
  compensation or rejection.

Do not hide failure injection behind opaque helpers or framework abstractions.

---

## Activity design rules

Activities model external work.

They may simulate:
- Mortgage application intake
- Credit and AML checks
- Offer reservation
- Fulfilment
- Property valuation

Activities should be:
- Small
- Explicit
- Serializable
- Easy to stub and reason about

Activities may include controlled demo behaviour such as:
- delayed completion
- forced failure
- compensating actions

Do not hide demo-critical behaviour behind opaque helpers.

If failure injection exists, make it obvious and intentional.

---

## Auditability and observability rules

Auditability is a first-class feature of this repository.

Claude should optimise for a clear and ordered business timeline, not just
technical success.

The system should make it easy to inspect:

- current status
- current step
- historical steps
- waits
- retries
- failures
- compensation
- workflow version

Prefer explicit audit timeline entries over implicitly reconstructing everything
from unrelated logs.

Logs should be structured where practical and should consistently include:

- `applicationId`
- workflow identifier where applicable
- relevant step or activity name

The UI should present a business-readable view of execution state.

---

## API design rules

The API is a thin control surface.

It should expose explicit endpoints for things such as:

- starting a mortgage application workflow
- completing or simulating async credit checks
- querying application state
- triggering retries or re-runs where appropriate

Prefer:
- explicit endpoint names
- explicit DTOs
- predictable response payloads
- clear HTTP semantics

Avoid:
- burying orchestration logic in controllers
- large service classes with mixed concerns
- magical request transformation
- vague endpoint naming

If a Temporal client call is made, the API should act as a transport layer,
not as a workflow engine.

---

## UI design rules

The UI exists to support understanding and demonstration.

It should prioritise:

- search or lookup by `applicationId`
- clear status display
- ordered timeline display
- visibility of pending async dependencies
- visibility of failures and retries
- simple operator actions where useful

Prefer:
- a small number of clear screens
- simple state management
- readable typography and layout
- direct mapping between API data and UI presentation

Avoid:
- heavy client-side architecture
- over-designed component systems
- generic admin-framework complexity
- speculative features

The UI should feel like a focused demo tool, not a productised control centre.

---

## SvelteKit data loading rules

Initial page data must be loaded through SvelteKit `load` functions, not in `onMount`.

Use `+page.ts` for universal load (runs on both server and client).
Use `+page.server.ts` only if the data must be kept server-side.

`onMount` is for browser-only behaviour: DOM APIs, event listeners,
and third-party integrations that require a browser context. It must
not be used for initial data fetching.

When snapshotting `data` prop values into local `$state`, use
`untrack()` to signal the intent explicitly and suppress the Svelte
compiler warning.

Prefer:
- `+page.ts` load for scenarios, application state from URL params,
  and any data the page needs on first render
- polling and post-load refresh via `$effect` (client-side only)
- `data` prop destructuring into local state only where the component
  manages that state independently after initialisation

Avoid:
- `onMount` for API calls or query param reading
- duplicating load logic in `onMount` and `load`
- fetching initial page state imperatively in component lifecycle hooks

---

## Monorepo rules

This repository is a monorepo. Claude should preserve clean app boundaries.

Expected structure:

- `apps/api`
- `apps/worker`
- `apps/ui`

Shared code should be introduced cautiously.

Prefer:
- duplication over premature shared abstractions
- explicit contracts over tightly coupled shared libraries
- app-local ownership of implementation details

Avoid:
- cross-app imports that blur boundaries
- monorepo-wide utility layers without clear value
- leaking UI concerns into the API
- leaking API concerns into the worker

A small amount of duplication is acceptable if it preserves clarity.

---

## Docker Compose and local development

Local development should remain simple and reproducible.

Assume the primary local developer experience is:

- `docker compose up` for infrastructure and app startup
- clear environment variables
- predictable ports
- minimal manual setup

Claude should preserve or improve:
- reproducibility
- startup reliability
- health checks
- clarity of service dependencies

Do not add unnecessary local infrastructure.

Keep the Compose story easy to explain in a demo or handover.

---

## Naming and terminology

Use clear, business-relevant naming.

Prefer:
- `applicationId`
- `mortgage application`
- `credit check`
- `offer reservation`
- `fulfilment`
- `property valuation`
- `audit timeline`
- `workflow version`

Be careful with the term `control-plane`.

It is acceptable as an internal naming convention for the API app, especially if
that matches existing project conventions.

However, Claude should avoid implying that the API owns orchestration behaviour.

In documentation and user-facing explanations, prefer terms like:
- API layer
- workflow control API
- control interface

Temporal and the worker should remain clearly positioned as the orchestration
engine.

---

## Code style and implementation guidance

General guidance:
- Prefer small, focused functions
- Prefer explicit naming over brevity
- Keep control flow readable
- Avoid deep nesting
- Avoid clever indirection

### Go guidance

- Prefer explicit structs and methods
- Keep Temporal workflow code deterministic
- Keep activity boundaries clear
- Handle errors explicitly
- Prefer standard library solutions where practical

Do not:
- hide workflow logic behind excessive abstraction
- introduce non-replay-safe behaviour
- over-generalise activity implementations

### TypeScript guidance

- Prefer explicit DTOs and interfaces
- Keep NestJS dependency injection idiomatic
- Prefer constructor injection in NestJS
- Keep Svelte components focused and readable
- Avoid unnecessary type complexity

Do not:
- weaken typing globally to silence local issues
- hide simple data flow behind complex patterns
- add framework cleverness for its own sake

---

## Testing and validation expectations

A task is not complete until the relevant code has been exercised.

Claude should run or recommend running the smallest relevant validation for the
change.

Examples:

### Worker

- unit tests for workflow and activity logic
- targeted workflow behaviour checks
- compile checks for Go code

### API

- unit tests for services/controllers where appropriate
- compile/type checks
- endpoint sanity testing

### UI

- type checks
- component or route-level validation where appropriate
- basic manual smoke testing for demo flows

Prefer targeted validation during iteration and broader validation before
completion.

Do not claim work is complete if code has not been checked in some meaningful
way.

---

## Documentation expectations

Documentation is part of the deliverable.

Docs should explain:
- what each app does
- how to run the system locally
- how the demo flow works
- what is intentionally simplified
- how the PoC maps to Temporal concepts

Prefer:
- practical setup instructions
- architecture explanations grounded in actual code
- concise endpoint and workflow documentation
- screenshots or examples only when accurate

Avoid:
- aspirational documentation
- speculative production claims
- overstating what the PoC does

If implementation changes the intended behaviour, documentation must be updated
to match intentional changes only.

Do not update docs to match accidental regressions.

---

## Writing style rules

- Use British English spelling and punctuation.
- Do not use em dashes.
- Use full stops, commas or sentence restructuring instead.
- Prefer direct, technical language.
- Avoid marketing language and hype.
- Avoid vague architectural prose.

Write as if explaining the system to a competent engineer or architect who
values clarity and precision.

---

## Scope control

This repository is intentionally narrow.

Claude should not:
- turn the PoC into a generic framework
- add enterprise platform features unrelated to the demo
- introduce multi-tenancy, auth, RBAC, provisioning or policy engines unless
  explicitly requested
- add external messaging infrastructure unless explicitly requested
- over-model the mortgage domain

When implementing changes:
- start with the smallest correct solution
- preserve the architectural split between worker, API and UI
- keep Temporal central to the story
- optimise for a reliable demo

---

## When unsure

If something is unclear:
- ask before expanding scope
- preserve the simplest working design
- prefer explicit behaviour
- avoid guessing hidden requirements

Correctness beats cleverness.
Clarity beats flexibility.
Temporal remains the orchestration engine.

---

## Linting, formatting and pre-commit

This defines the minimum standard for considering any task complete.

This repository uses pre-commit to enforce basic code quality, formatting and
validation across all applications.

All changes must pass pre-commit checks before they are considered complete.

Claude must:

- Run pre-commit after making changes where possible:
  - `pre-commit run --all-files`
- Ensure all checks pass before considering a task complete
- Fix issues rather than bypassing them

If pre-commit is not installed, install it:

- `pip install pre-commit`
- `pre-commit install`

---

## Formatting (API - NestJS)

The API (`apps/api`) uses Prettier via `npm run format`.

When making changes to any files under `apps/api`, Claude must:

- Run formatting before running pre-commit:
  - `cd apps/api && npm run format && npm run lint`
- Ensure formatting changes are applied before final validation
- Then run:
  - `pre-commit run --all-files`

This ensures:
- consistent formatting in TypeScript code
- minimal noise in diffs
- pre-commit checks run on already-formatted files

Do not:
- skip formatting for small changes
- rely on pre-commit to fix formatting after the fact

Formatting is part of the definition of done for API changes.

---

## Formatting (UI - SvelteKit)

The UI (`apps/ui`) uses ESLint + Prettier via `npm run lint`.

When making changes to any files under `apps/ui`, Claude must:

- Run linting before running pre-commit:
  - `cd apps/ui && npm run lint`
- Ensure all linting and formatting issues are resolved before final validation
- Then run:
  - `pre-commit run --all-files`

This ensures:
- consistent formatting across Svelte, TypeScript and CSS
- adherence to SvelteKit and Svelte 5 idioms
- minimal noise in diffs
- pre-commit checks run on already-clean files

Do not:
- skip linting for small changes
- rely on pre-commit to fix issues after the fact

Linting is part of the definition of done for UI changes.

---

## Application builds

Application builds are required in addition to pre-commit checks. A task is not
complete if the relevant app build fails.

When files in these apps are changed, Claude must run the corresponding build
before considering the task complete:

- `apps/api`: `cd apps/api && npm run build`
- `apps/worker`: `cd apps/worker && go build -o /tmp/worker .`
- `apps/ui`: `cd apps/ui && npm run build`

Run only the builds for the apps that were changed. If changes span multiple
apps, run all relevant builds.

Do not:
- skip the build step for small changes
- assume pre-commit passing is sufficient
- claim a task is complete if the build fails

---

### Expectations by area

#### Go (worker)

- Code must compile: `go build ./...`
- Run tests where present: `go test ./...`
- Formatting must be correct (gofmt/gofumpt)
- Linting issues must be resolved where configured

#### API (NestJS / TypeScript)

- TypeScript must compile without errors
- Linting must pass (if configured)
- Avoid weakening compiler settings to silence errors

#### UI (SvelteKit)

- Type checks must pass
- Linting must pass (if configured)
- Do not bypass errors with unsafe casts unless justified

---

### General rules

- Do not disable linters or checks without discussion
- Do not add ignore directives to bypass failures without justification
- Do not leave the repository in a failing state
- Fix the root cause of issues, not just the symptom

A change is not complete until:

- Relevant code compiles
- Pre-commit checks pass
- No new linting or formatting errors are introduced
- The app build passes for every app that was changed
