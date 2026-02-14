---
name: monolith-job-development
description: Use when creating or extending asynchronous background jobs, including generator usage, queue registration, payload handling, and worker lifecycle.
---

# Monolith Job Development

## Use this skill when
- Adding new async work to `app/jobs/`.
- Updating queue handlers or job scheduling behavior.

## Fast path
Run: `make generator job <Name>`

This typically:
- creates `app/jobs/<name>_job.go`
- creates matching test file
- adds job type constant in `app/models/job.go`
- registers handler in `app/jobs/job_queue.go`

## Runtime architecture
- Queue initialized via `jobs.InitJobQueue()` in `main.go`.
- Worker count controlled by `config.JOB_QUEUE_NUM_WORKERS`.
- Jobs are typed via `models.JobType*` enums and payload bytes.

## Implementation workflow
1. Define payload struct and marshal/unmarshal strategy.
2. Implement generated job handler body.
3. Register handler (generator usually does this).
4. Enqueue from controller/service as needed.
5. Add focused tests for decoding + handler behavior.

## Reliability tips
- Keep handlers idempotent where possible.
- Log failures with useful context.
- Treat malformed payloads as explicit errors.
