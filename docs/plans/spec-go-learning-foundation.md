---
title: 'Go learning foundation'
type: 'chore'
created: '2026-08-06'
status: 'done'
baseline_commit: 'NO_VCS'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent - do not modify unless human renegotiates">

## Intent

**Problem:** The repository is empty, and adding unrelated Go exercises directly at the root would quickly make it difficult for a beginner to understand where lessons, runnable programs, reusable code, and notes belong.

**Approach:** Create one small Go module with all major learning categories prepared in numbered, topic-based folders, a conventional application entry point, and Chinese documentation. Include the first runnable lesson and let future exercises be placed directly in the matching category.

## Boundaries & Constraints

**Always:** Keep every lesson independent and runnable with standard Go commands; prepare numbered topic folders matching the learning order; explain each directory in beginner-friendly Chinese; use Chinese directory names for the learner; use only the Go standard library; keep the repository root small.

**Ask First:** Adding third-party dependencies, a web framework, a database, multiple Go modules, or moving from learning examples to a real product architecture.

**Never:** Leave a category without a Chinese README; mix generated files or build output into source folders; introduce enterprise abstractions, dependency injection, or configuration systems before a lesson needs them; copy the referenced tutorial verbatim.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|---------------|----------------------------|----------------|
| Run first lesson | `go run .\learn\01-基础语法\01-hello\main.go` | Prints `Hello, Go!` | Go command reports compilation errors normally |
| Test first lesson | `go test .\learn\01-基础语法\01-hello\main.go .\learn\01-基础语法\01-hello\main_test.go` | The lesson test passes | Non-zero exit identifies the failing files |
| Discover next step | Open root or learning README | Shows directory responsibilities and a numbered study path | Links use repository-relative paths |

</frozen-after-approval>

## Code Map

- `go.mod` -- defines the single local learning module and Go language version.
- `README.md` -- explains how to begin, core commands, directory ownership, and growth rules.
- `.gitignore` -- excludes editor files, local binaries, coverage, and temporary output.
- `learn/README.md` -- maintains the numbered curriculum and lesson conventions.
- `learn/01-基础语法/01-hello/main.go` -- first independently runnable syntax example.
- `learn/01-基础语法/01-hello/main_test.go` -- verifies the first example's observable output.
- `learn/01-基础语法/01-hello/README.md` -- states concepts, commands, and a small practice task.
- `cmd/README.md` -- reserves conventional executable entry points for future complete applications.
- `internal/README.md` -- explains where future non-public reusable application code belongs.
- `docs/README.md` -- provides a home for longer notes and design records.

## Tasks & Acceptance

**Execution:**
- [x] `go.mod`, `.gitignore` -- establish a clean single-module repository foundation.
- [x] `README.md` -- document prerequisites, commands, structure, and rules for extending it.
- [x] `learn/README.md` -- define the complete tutorial-aligned learning path and directory naming rules.
- [x] `learn/01-基础语法/01-hello/main_test.go` -- first specify the expected `Hello, Go!` output and observe the test fail.
- [x] `learn/01-基础语法/01-hello/main.go` -- implement the smallest runnable first lesson that makes the test pass.
- [x] `learn/01-基础语法/01-hello/README.md` -- explain the lesson and give one hands-on exercise.
- [x] `learn/01-基础语法` through `learn/13-测试与工程实践` -- prepare categorized learning directories with Chinese README files.
- [x] `cmd/README.md`, `internal/README.md`, `pkg/README.md`, `projects/README.md`, `docs/README.md` -- document future folder responsibilities.

**Acceptance Criteria:**
- Given Go is installed, when `go run .\learn\01-基础语法\01-hello\main.go` is executed, then the terminal prints exactly `Hello, Go!` followed by a newline.
- Given the repository root, when a beginner reads `README.md`, then they can identify the first lesson, run it, test it, and understand where future files belong.
- Given a new lesson is added later, when its topic is selected, then it can be placed in the next numbered folder under `learn` without changing existing lesson imports.
- Given the first lesson is checked, when file-based `go test` and `go vet` commands run, then both commands exit successfully.

## Spec Change Log

## Design Notes

Numbered lesson folders optimize for learning order, while `cmd` and `internal` introduce standard Go conventions without forcing exercises into a production layout. Every prepared category contains a Chinese README, and only `01-hello` contains code until the learner starts the corresponding topic.

## Verification

**Commands:**
- `gofmt -w learn/01-基础语法/01-hello/*.go` -- expected: source is formatted without errors.
- `go test .\learn\01-基础语法\01-hello\main.go .\learn\01-基础语法\01-hello\main_test.go` -- expected: the lesson test passes.
- `go vet .\learn\01-基础语法\01-hello\main.go .\learn\01-基础语法\01-hello\main_test.go` -- expected: no suspicious constructs are reported.
- `go run .\learn\01-基础语法\01-hello\main.go` -- expected: output is exactly `Hello, Go!`.
