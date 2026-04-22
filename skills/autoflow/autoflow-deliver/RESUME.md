# Resume Protocol

Reference loaded on-demand when `workflow-progress.json` exists for a ticket.
Parent skill: `autoflow-deliver/SKILL.md`

---

## How Resume Works

The step controller handles all resume logic. You just call `next` and it returns the right instruction for wherever the workflow left off.

### Steps

1. Call `autoflow deliver next --ticket <KEY>`. It reads `workflow-progress.json` and returns the instruction for the current step.

2. Rebuild the TodoWrite list in ONE call. Mark every completed step as `completed`, `current_step` as `in_progress`, rest as `pending`.

3. Continue the normal loop: execute the instruction, call `complete`, call `next`.

### Mid-loop Resume

All loop steps use artifact-based phase detection — the controller reads state files on disk to determine where it left off. No round tracking needed on your part.

- **Step 4 (AC coverage):** Reads `coverage-review-state.json` for round status.
- **Step 5 (Implement):** Checks if `PLAN.md` exists (plan done) or `SUMMARY.md` exists (execution done).
- **Step 6 (Build gate):** Reads `build-gate-state.json` for attempt count and last result.
- **Step 7 (E2E tests):** Reads `e2e-fix-state.json` for round status and fix markers.
- **Step 9 (Review + fix):** Checks which `REVIEW-*.md` files exist and whether `REVIEW-FIX.md` was written.

### No Progress File

If `autoflow deliver next` returns a Step 1 instruction with no progress file, the workflow is starting fresh. Follow the normal SKILL.md flow from Step 0.
