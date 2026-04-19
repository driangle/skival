---
title: "Add post-verification process cleanup to executor"
id: "01kpkc32c"
status: pending
priority: high
type: feature
tags: ["executor", "cleanup"]
created: "2026-04-19"
---

# Add post-verification process cleanup to executor

## Objective

After verification completes for a sample, skival should kill any processes that were spawned in the working directory. Evals using `http_check` or `tcp_check` require the agent to start long-running servers, and those processes currently outlive the skival run — causing port conflicts on subsequent runs and leaking resources.

## Tasks

- [ ] Track child processes spawned during agent execution in the sample working directory
- [ ] After the verification pipeline finishes in `runSample` (`internal/executor/executor.go`), kill any processes still running in the working directory
- [ ] Handle cleanup gracefully (SIGTERM first, SIGKILL after timeout)
- [ ] Add tests verifying that spawned processes are cleaned up after verification
- [ ] Verify the `http-check` and `tcp-check` correctness examples don't leave orphan processes

## Acceptance Criteria

- No orphan processes remain after `skival run` completes for evals that start servers
- Cleanup uses graceful shutdown (SIGTERM) with a fallback to SIGKILL
- Cleanup runs even if verification fails
- Existing evals that don't spawn servers are unaffected
