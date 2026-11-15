
# AGENTS.md

# KurbanTrack Backend — AI Agent Instructions

## 1. Project Overview

KurbanTrack is a RESTful API backend for managing qurban animals, meat package estimates, recipient allocations, and distribution coupons.

### Technology Stack
- Language: Go
- HTTP Framework: Fiber
- Database: PostgreSQL
- PostgreSQL Driver: pgx
- Architecture: Clean Architecture, following the course's Module 4 structure
- API Format: JSON
- API Prefix: `/api/v1`

### Project Goals
1. Build a working Animals CRUD feature from request to database.
2. Provide a structured scaffold for Distribution features.
3. Maintain clean, readable, testable, and maintainable Go code.
4. Follow the architecture and technical constraints defined in the course materials.

## 2. Scope and Implementation Priorities

### Fully Implemented
The following functionality must work end-to-end:

- `GET /health`
- `POST /api/v1/animals`
- `GET /api/v1/animals`
- `GET /api/v1/animals/:id`
- `PUT /api/v1/animals/:id`
- `DELETE /api/v1/animals/:id`

Animals CRUD includes:
- Request parsing
- Input validation
- Business rule validation
- Repository operations
- PostgreSQL integration
- Consistent JSON responses
- Appropriate HTTP status codes
- Automated tests

### Scaffold Only
Distribution functionality is currently limited to clean, compilable scaffolding:

- `POST /api/v1/distributions`
- `GET /api/v1/distributions`
- `PATCH /api/v1/distributions/:code/redeem`
- `PATCH /api/v1/distributions/:code/cancel`

Do not implement the complete distribution workflow unless explicitly requested.

Do not expand the project scope without approval from the project team.

## 3. Mandatory Project Structure

Follow this structure based on the course's Module 4:

```text
kurbantrack-backend/
├── app/
│   ├── model/
│   ├── repository/
│   └── service/
├── config/
│   ├── app.go
│   ├── env.go
│   └── logger.go
├── database/
│   └── postgres.go
├── helper/
│   └── response.go
├── middleware/
├── route/
├── migrations/
├── logs/
├── .agents/
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
├── README.md
└── main.go
```

Do not introduce a different architecture or reorganize the project without explicit approval.

### Package Responsibilities

#### `app/model/`
- Define entity structs.
- Define request and response structs.
- Define JSON tags and data representations.
- Keep models independent from Fiber and database queries.
- Do not put business logic or SQL queries here.

#### `app/repository/`
- Define repository interfaces where appropriate.
- Implement PostgreSQL queries.
- Map database rows into model structs.
- Handle database-related errors.
- Do not import Fiber or use HTTP status codes.
- Do not put business rules here.

#### `app/service/`
- Handle HTTP request parsing through `fiber.Ctx`.
- Coordinate business processes.
- Call repository methods.
- Map application errors to appropriate HTTP responses.
- Keep SQL queries and database connection creation out of this package.
- Write business rules as ordinary Go functions that receive structs and return results.
- Keep pure business rules independent from `fiber.Ctx` so they can be tested without starting a server.

#### `config/`
- Load environment variables.
- Configure Fiber.
- Configure logging.
- Register middleware and routes through application setup.
- Do not put SQL queries or business rules here.

#### `database/`
- Create and manage PostgreSQL connections and connection pools.
- Handle connection lifecycle.
- Do not put entity-specific queries here.

#### `helper/`
- Provide reusable, stateless helper functions.
- Keep helpers independent from repository and service packages.
- Use this package for shared response formatting when appropriate.

#### `middleware/`
- Define Fiber middleware.
- Keep database queries and business rules out of middleware.

#### `route/`
- Register API routes.
- Map URLs and HTTP methods to service methods.
- Do not put business logic, validation decisions, or SQL queries here.

#### `main.go`
- Load configuration.
- Initialize logger and database.
- Initialize repositories and services.
- Assemble and start the application.
- Handle graceful shutdown.
- Do not put HTTP request processing, SQL queries, or business rules here.

## 4. Architecture Rules

The project must follow these dependency boundaries:

- `model` must not depend on other application packages.
- `repository` may depend on `model`.
- `service` may depend on `model` and `repository`.
- `route` maps requests to service methods.
- `config` assembles the application.
- `main.go` initializes and connects dependencies.

Important restrictions:

- Never create packages, folders, files, or structs named `controller` or `handler`.
- Do not introduce `domain`, `usecase`, or `delivery` layers from another architecture template.
- Do not move the course's `app/service` responsibilities into a different package.
- Do not import Fiber inside `app/repository`.
- Do not execute SQL queries inside `app/service`.
- Do not put business logic inside `route`.
- Do not put business logic inside `main.go`.
- Avoid circular package dependencies.

Before adding a new package, explain why the existing structure cannot accommodate the requirement.

## 5. Business Rules

### Animal Rules
- Animal code must be unique.
- Supported animal species: `sapi`, `kambing`, and `domba`.
- Animal weight must be greater than zero.
- Estimated package quantity must be greater than zero.
- Allocated package quantity must not exceed estimated package quantity.
- An animal must not be deleted while it still has active allocations.
- Invalid input must return a meaningful error response.

### Distribution Rules
Distribution is currently scaffold-only. Keep these rules documented for future implementation:

- Recipient NIK must be unique.
- Coupon code must be unique.
- A coupon can only be redeemed when its status is `allocated`.
- Allocation must never exceed the animal's estimated package quota.
- Status changes should be recorded in `distribution_logs`.

## 6. Database Rules

- Use PostgreSQL with pgx.
- Keep SQL queries inside repository implementations.
- Use parameterized SQL queries.
- Never concatenate user input directly into SQL statements.
- Use database constraints for data integrity where appropriate.
- Use migrations to manage schema changes.
- Do not commit database credentials or local environment files.
- Handle database errors explicitly.
- Use transactions when multiple database operations must succeed or fail together.

For future distribution implementation:
- Prevent concurrent allocations from exceeding animal quotas.
- Prevent concurrent redemption of the same coupon.
- Use atomic conditional updates or transactions for these operations.
- Record distribution status changes in the same transaction as the status update.

Do not implement the complete distribution transaction workflow unless explicitly requested.

## 7. Go Coding Standards

- Follow idiomatic Go conventions.
- Use meaningful package, function, method, and variable names.
- Keep functions focused and reasonably small.
- Handle errors explicitly; do not silently ignore errors.
- Avoid unnecessary global variables.
- Avoid unnecessary abstractions and dependencies.
- Use `context.Context` for database operations where appropriate.
- Use dependency injection instead of creating database connections inside services.
- Format all Go files using `gofmt`.
- Keep exported identifiers documented when appropriate.
- Do not use `panic` for ordinary application errors.

Before adding a dependency:
1. Check whether the standard library or existing dependencies are sufficient.
2. Explain why the new dependency is needed.
3. Avoid adding packages that are unnecessary for the current assignment.

## 8. API and Response Standards

- Use JSON for request and response bodies.
- Use consistent response structures.
- Use appropriate HTTP status codes.
- Validate request data before calling repository operations.
- Return meaningful error messages without exposing internal database details.
- Handle invalid IDs, missing records, duplicate values, and database failures.

Do not change an existing API contract without explaining the impact.

## 9. Testing and Quality Assurance

Before considering a task complete, run the relevant checks:

```bash
gofmt -w .
go test ./...
go build ./...
```

For database-related changes:
- Verify that migrations can be applied.
- Test repository operations against PostgreSQL when possible.
- Verify both successful and failure scenarios.

For Animals CRUD, test at least:
- Successful create, list, detail, update, and delete.
- Invalid animal species.
- Zero or negative weight.
- Zero or negative estimated packages.
- Duplicate animal code.
- Non-existent animal ID.
- Attempt to delete an animal that still has allocations.

Never claim that tests pass unless they have actually been executed.

If a test cannot be executed, state the reason clearly.

## 10. Environment and Security

- Never commit `.env`.
- Only commit `.env.example`.
- Add `logs/` and generated binaries to `.gitignore`.
- Never hardcode database credentials, secrets, or tokens.
- Do not expose sensitive information in logs or API responses.
- Keep environment variable names consistent across the project.

## 11. Git Workflow

The project uses a branch-based workflow.

Rules:
- Never push directly to `main`.
- Every member, including Git Master, must work on a separate branch.
- Use Pull Requests to merge changes into `main`.
- Git Master is responsible for reviewing and merging PRs.
- Always use **Create a merge commit** when merging PRs.
- Do not use Squash and Merge or Rebase and Merge.
- Prefer small, focused commits (micro commits).
- Do not commit unrelated changes together.
- Do not overwrite another member's work.

Suggested branch naming:

- `feat/animal-model`
- `feat/animal-repository`
- `feat/animal-service`
- `feat/animal-route`
- `feat/database-migration`
- `feat/distribution-scaffold`
- `docs/project-documentation`
- `fix/bug-description`

Suggested commit format:

```text
feat(animal): add animal creation
fix(repository): handle missing animal record
test(animal): add validation tests
docs(readme): update local setup instructions
chore(config): add environment example
```

Before preparing a PR:
1. Ensure the branch is based on the latest `main`.
2. Run formatting, tests, and build checks.
3. Review the diff for unrelated changes.
4. Explain the implementation and testing results in the PR description.

## 12. AI Agent Working Rules

Before modifying code:

1. Inspect the existing project structure.
2. Read relevant source files.
3. Understand existing interfaces and dependencies.
4. Identify which package should own the change.
5. Explain the proposed approach briefly when the change affects architecture or multiple packages.

While modifying code:

- Follow existing project conventions.
- Make the smallest reasonable change.
- Do not rewrite unrelated files.
- Do not silently change API contracts or database schemas.
- Do not create duplicate implementations.
- Do not assume a feature is implemented just because its route exists.

After modifying code:

1. Summarize files created or modified.
2. Explain important implementation decisions.
3. Report commands and tests actually executed.
4. Report any remaining issues or limitations.
5. Mention any migration or configuration steps required.

If a requirement is ambiguous or conflicts with these instructions, do not guess. Explain the conflict and ask for clarification.

## 13. Priority of Instructions

When working on this project, follow this priority:

1. Explicit instructions from the user for the current task.
2. Project requirements and approved SRS.
3. Course Module 4 architecture and folder responsibilities.
4. This `AGENTS.md`.
5. Existing code conventions.

Do not silently resolve conflicts by introducing a different architecture. Explain the conflict first.
