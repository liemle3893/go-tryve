---
phase: quick
plan: 1
type: execute
wave: 1
depends_on: []
files_modified:
  - src/adapters/http.adapter.ts
  - src/core/yaml-loader.ts
  - docs/sections/adapters/http.md
  - docs/sections/yaml-test.md
  - docs/sections/examples.md
  - .claude/skills/e2e-runner/references/adapters/http.md
  - .claude/skills/e2e-runner/SKILL.md
autonomous: true
requirements: [QUICK-UPLOAD-01]

must_haves:
  truths:
    - "User can send multipart/form-data file uploads via HTTP adapter in YAML tests"
    - "User can mix file fields and text fields in a single multipart request"
    - "The Content-Type header is automatically set to multipart/form-data with correct boundary when multipart is used"
    - "File paths in multipart fields are resolved relative to the test file directory"
    - "Existing JSON body requests continue to work unchanged"
  artifacts:
    - path: "src/adapters/http.adapter.ts"
      provides: "Multipart/form-data request body construction using FormData"
      contains: "FormData"
    - path: "src/core/yaml-loader.ts"
      provides: "Validation of multipart field in HTTP steps"
      contains: "multipart"
    - path: "docs/sections/adapters/http.md"
      provides: "Documentation for multipart upload syntax"
      contains: "multipart"
  key_links:
    - from: "src/adapters/http.adapter.ts"
      to: "node:fs"
      via: "fs.readFileSync for reading file paths in multipart fields"
      pattern: "readFileSync|createReadStream"
    - from: "src/adapters/http.adapter.ts"
      to: "FormData"
      via: "Node.js built-in FormData API for multipart body construction"
      pattern: "new FormData"
    - from: "src/core/yaml-loader.ts"
      to: "src/adapters/http.adapter.ts"
      via: "multipart field passed through params to HTTP adapter"
      pattern: "multipart"
---

<objective>
Add multipart/form-data file upload support to the HTTP adapter so users can test
file upload endpoints directly from YAML tests.

Purpose: File upload is a common API pattern (profile pictures, document imports,
CSV uploads) that currently requires workarounds or TypeScript test files. Adding
native YAML support keeps the framework's declarative testing promise.

Output: HTTP adapter handles `multipart` field, builds FormData bodies with file
and text fields, documentation updated across all relevant files.
</objective>

<execution_context>
@./.claude/get-shit-done/workflows/execute-plan.md
@./.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@CLAUDE.md
@src/adapters/http.adapter.ts
@src/core/yaml-loader.ts
@src/types.ts
@docs/sections/adapters/http.md
</context>

<interfaces>
<!-- Key types and contracts the executor needs. Extracted from codebase. -->

From src/adapters/http.adapter.ts:
```typescript
export interface HTTPRequestParams {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD' | 'OPTIONS';
  url: string;
  headers?: Record<string, string>;
  body?: unknown;
  query?: Record<string, string>;
  timeout?: number;
  followRedirects?: boolean;
  capture?: Record<string, string>;
  assert?: HTTPAssertion;
}
```

From src/types.ts:
```typescript
export interface UnifiedStep {
  id: string;
  adapter: AdapterType;
  action: string;
  description?: string;
  params: Record<string, unknown>;
  capture?: Record<string, string>;
  assert?: unknown;
  continueOnError?: boolean;
  retry?: number;
  delay?: number;
}
```

From src/core/yaml-loader.ts (convertStep):
```typescript
function convertStep(raw: RawYAMLStep, phase: string, index: number): UnifiedStep {
  const { adapter, action, description, continueOnError, retry, delay, ...rest } = raw;
  const { capture, assert, ...params } = rest;
  // Everything not a known field goes into params — multipart will naturally flow through
  return { id: `${phase}-${index}`, adapter, action, description, params, capture, assert, continueOnError, retry, delay };
}
```
</interfaces>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Implement multipart/form-data support in HTTP adapter</name>
  <files>src/adapters/http.adapter.ts, src/core/yaml-loader.ts</files>
  <behavior>
    - When params.multipart is defined, build a FormData body instead of JSON.stringify
    - Each entry in multipart is an object with `name` (field name) and either `file` (path) or `value` (text)
    - File entries: read file from disk using fs.readFileSync, append as Blob with filename to FormData
    - Text entries: append as string to FormData
    - When multipart is used, do NOT set Content-Type header (let fetch set it with boundary)
    - When multipart is NOT present, existing JSON body behavior is unchanged
    - File paths are used as-is (variable interpolation handles resolution before adapter receives params)
    - Validation in yaml-loader: when adapter=http and multipart is present, each entry must have `name` and either `file` or `value`
  </behavior>
  <action>
1. In `src/adapters/http.adapter.ts`:
   - Add `multipart` field to `HTTPRequestParams` interface:
     ```typescript
     multipart?: Array<{
       name: string;
       file?: string;    // file path to read and attach
       value?: string;   // text field value
       filename?: string; // optional override for the uploaded filename
       contentType?: string; // optional MIME type override
     }>;
     ```
   - In the `request()` method, after building headers and before the fetch call, add multipart body logic:
     - If `params.multipart` is defined and is a non-empty array:
       a. Create `const formData = new FormData()`
       b. For each entry in `params.multipart`:
          - If entry has `file`: read the file with `fs.readFileSync(entry.file)`, create a `new Blob([fileBuffer])` with the contentType if provided, append to formData with `formData.append(entry.name, blob, entry.filename || path.basename(entry.file))`
          - If entry has `value`: `formData.append(entry.name, entry.value)`
          - If entry has neither `file` nor `value`: throw AdapterError
       c. Set `fetchOptions.body = formData`
       d. Remove `Content-Type` from headers (let fetch generate the multipart boundary automatically)
     - If `params.multipart` is NOT defined, keep existing JSON body logic as-is
   - Import `path` from `node:path` (already imported `fs` pattern exists in variable-interpolator, but http.adapter.ts needs its own — actually check first, use `import * as path from 'node:path'` and `import * as fs from 'node:fs'`)

2. In `src/core/yaml-loader.ts`:
   - In `validateAdapterStep()` under the `case 'http':` block, add validation:
     If `step.multipart` is present, validate it is an array and each entry has `name` (string) and either `file` or `value`.
     Also validate that `body` and `multipart` are not both defined (mutually exclusive).

3. Add `import * as fs from 'node:fs'` and `import * as path from 'node:path'` at the top of `http.adapter.ts`.
  </action>
  <verify>
    <automated>cd /Users/liemlhd/Documents/git/Personal/e2e-runner && npm run build</automated>
  </verify>
  <done>
    - HTTPRequestParams has multipart field
    - request() method builds FormData when multipart is provided
    - request() method omits Content-Type header when using multipart (lets fetch set boundary)
    - Existing JSON body requests compile and work unchanged
    - yaml-loader validates multipart entries and rejects body+multipart together
    - TypeScript compiles without errors
  </done>
</task>

<task type="auto">
  <name>Task 2: Update documentation for multipart upload support</name>
  <files>docs/sections/adapters/http.md, docs/sections/yaml-test.md, docs/sections/examples.md, .claude/skills/e2e-runner/references/adapters/http.md, .claude/skills/e2e-runner/SKILL.md</files>
  <action>
1. In `docs/sections/adapters/http.md`:
   - Add a new section "## Multipart/Form-Data Uploads" after the "Action: request" section.
   - Document the `multipart` field syntax:
     ```yaml
     - adapter: http
       action: request
       method: POST
       url: "{{baseUrl}}/upload"
       multipart:
         - name: "file"
           file: "./fixtures/test-image.png"
         - name: "description"
           value: "Profile picture"
         - name: "document"
           file: "./fixtures/report.pdf"
           filename: "custom-name.pdf"
           contentType: "application/pdf"
       assert:
         status: 200
     ```
   - Explain: `multipart` and `body` are mutually exclusive. When `multipart` is used, Content-Type is automatically set to `multipart/form-data` with the correct boundary by fetch.
   - Document each multipart entry field: `name` (required), `file` (path to file), `value` (text field), `filename` (optional override), `contentType` (optional MIME type).
   - In the "Action: request" parameter table, add `multipart` as an optional field.

2. In `docs/sections/yaml-test.md`:
   - In the "Step Definition" section, add `multipart` to the adapter-specific parameters comment block.

3. In `docs/sections/examples.md`:
   - Add a "File Upload" example section showing a complete test that uploads a file and asserts the response.

4. In `.claude/skills/e2e-runner/references/adapters/http.md`:
   - Mirror the multipart documentation from `docs/sections/adapters/http.md` (add multipart section after examples).

5. In `.claude/skills/e2e-runner/SKILL.md`:
   - In the "Step Definition" section, add a brief note that HTTP steps support `multipart` for file uploads.
   - In the HTTP Assertions section or near it, add a brief example of multipart usage.
  </action>
  <verify>
    <automated>cd /Users/liemlhd/Documents/git/Personal/e2e-runner && grep -q "multipart" docs/sections/adapters/http.md && grep -q "multipart" docs/sections/yaml-test.md && grep -q "multipart" .claude/skills/e2e-runner/references/adapters/http.md && echo "PASS: docs updated" || echo "FAIL: docs missing multipart"</automated>
  </verify>
  <done>
    - docs/sections/adapters/http.md has Multipart/Form-Data Uploads section with syntax reference and examples
    - docs/sections/yaml-test.md mentions multipart in step definition
    - docs/sections/examples.md has a file upload example
    - Skill reference mirrors the documentation
    - All documentation accurately reflects the implemented multipart field schema
  </done>
</task>

</tasks>

<verification>
1. `npm run build` compiles without errors
2. Grep confirms multipart support in adapter: `grep -n "FormData\|multipart" src/adapters/http.adapter.ts`
3. Grep confirms validation in loader: `grep -n "multipart" src/core/yaml-loader.ts`
4. Grep confirms docs updated: `grep -rn "multipart" docs/sections/`
5. Existing HTTP test YAML files still pass validation: `./bin/e2e.js validate`
</verification>

<success_criteria>
- HTTP adapter builds FormData body when `multipart` field is present in params
- File paths in multipart entries are read from disk and attached as Blob
- Text values in multipart entries are appended as strings
- Content-Type header is NOT manually set when using multipart (fetch auto-sets with boundary)
- body and multipart are mutually exclusive (validated by yaml-loader)
- Existing JSON body requests work unchanged (no regression)
- All documentation files updated to describe multipart syntax and usage
- TypeScript compiles cleanly
</success_criteria>

<output>
After completion, create `.planning/quick/1-add-support-for-file-upload-multipart-up/1-SUMMARY.md`
</output>
