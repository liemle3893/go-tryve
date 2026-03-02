# Common Test Patterns

Practical examples and recipes for common E2E testing scenarios, extracted from the framework documentation.

## CRUD Operations

### Create and Verify Resource

Create a resource via API, then verify it exists in the database:

```yaml
name: TC-USER-CREATE-001
description: Create user and verify in database
priority: P0
tags: [user, crud]

variables:
  unique_email: "test-{{$uuid()}}@example.com"
  user_name: "Test User"

setup:
  - adapter: postgresql
    action: execute
    sql: "DELETE FROM users WHERE email LIKE 'test-%@example.com'"

execute:
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users"
    headers:
      Content-Type: "application/json"
    body:
      email: "{{unique_email}}"
      name: "{{user_name}}"
    capture:
      user_id: "$.id"
    assert:
      status: 201
      json:
        - path: "$.id"
          exists: true
          type: "string"
        - path: "$.email"
          equals: "{{unique_email}}"

verify:
  - adapter: postgresql
    action: queryOne
    sql: "SELECT * FROM users WHERE id = $1"
    params: ["{{captured.user_id}}"]
    assert:
      - column: "email"
        equals: "{{unique_email}}"
      - column: "name"
        equals: "{{user_name}}"
      - column: "deleted_at"
        isNull: true

teardown:
  - adapter: http
    action: request
    method: DELETE
    url: "{{baseUrl}}/users/{{captured.user_id}}"
    continueOnError: true
```

### Full CRUD Flow

Create, read, update, and delete a resource in sequence:

```yaml
name: TC-ORDER-CRUD-001
description: Test complete order flow - create, update, verify, delete
priority: P0
tags: [order, crud, e2e]
timeout: 60000
retries: 2

variables:
  order_amount: 99.99
  customer_email: "customer-{{$uuid()}}@test.com"

setup:
  - adapter: postgresql
    action: execute
    sql: "DELETE FROM orders WHERE customer_email LIKE 'customer-%@test.com'"

execute:
  # Create
  - id: create_order
    adapter: http
    action: request
    description: "Create new order"
    method: POST
    url: "{{baseUrl}}/orders"
    headers:
      Content-Type: "application/json"
    body:
      customer_email: "{{customer_email}}"
      amount: "{{order_amount}}"
      items:
        - sku: "ITEM-001"
          quantity: 2
    capture:
      order_id: "$.id"
      order_status: "$.status"
    assert:
      status: 201
      json:
        - path: "$.status"
          equals: "pending"

  # Read
  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/orders/{{captured.order_id}}"
    assert:
      status: 200
      json:
        - path: "$.id"
          equals: "{{captured.order_id}}"

  # Update
  - id: update_order
    adapter: http
    action: request
    description: "Confirm order"
    method: PATCH
    url: "{{baseUrl}}/orders/{{captured.order_id}}"
    body:
      status: "confirmed"
    assert:
      status: 200
      json:
        - path: "$.status"
          equals: "confirmed"

verify:
  - adapter: postgresql
    action: queryOne
    sql: "SELECT * FROM orders WHERE id = $1"
    params: ["{{captured.order_id}}"]
    assert:
      - column: status
        equals: "confirmed"
      - column: customer_email
        equals: "{{customer_email}}"

teardown:
  - adapter: http
    action: request
    method: DELETE
    url: "{{baseUrl}}/orders/{{captured.order_id}}"
    continueOnError: true
```

---

## Error Validation

### Duplicate Resource Error

Verify the API returns proper error responses for conflicts:

```yaml
name: TC-USER-DUPLICATE-001
description: Test duplicate email rejection
priority: P1
tags: [user, error, validation]

variables:
  email: "duplicate-{{$uuid()}}@example.com"

execute:
  # Create user first
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users"
    body:
      email: "{{email}}"
      name: "Original User"
    capture:
      user_id: "$.id"
    assert:
      status: 201

  # Attempt duplicate creation
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users"
    body:
      email: "{{email}}"
      name: "Duplicate User"
    assert:
      status: [400, 409]
      json:
        - path: "$.errors"
          type: "array"
          notEmpty: true
        - path: "$.errors[0].code"
          equals: 8006
        - path: "$.errors[0].message"
          contains: "already exists"

teardown:
  - adapter: http
    action: request
    method: DELETE
    url: "{{baseUrl}}/users/{{captured.user_id}}"
    continueOnError: true
```

### Validation Error Response

Verify field validation errors:

```yaml
name: TC-USER-VALIDATION-001
description: Test input validation
priority: P1
tags: [user, validation]

execute:
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users"
    body:
      email: "not-an-email"
      name: ""
    assert:
      status: 400
      json:
        - path: "$.errors"
          type: "array"
          notEmpty: true
        - path: "$.errors[0]"
          exists: true
```

### Not Found Response

Verify 404 handling:

```yaml
name: TC-USER-NOTFOUND-001
description: Test resource not found
priority: P2
tags: [user, error]

execute:
  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/users/nonexistent-id-12345"
    assert:
      status: 404
      json:
        - path: "$.message"
          exists: true
```

---

## Database Verification

### Verify Database State After API Call

```yaml
name: TC-DB-VERIFY-001
description: Verify database reflects API changes
priority: P0
tags: [database, verification]

variables:
  unique_email: "dbtest-{{$uuid()}}@example.com"

execute:
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users"
    body:
      email: "{{unique_email}}"
      name: "DB Test User"
    capture:
      user_id: "$.id"
    assert:
      status: 201

verify:
  - adapter: postgresql
    action: queryOne
    sql: "SELECT * FROM users WHERE id = $1"
    params: ["{{captured.user_id}}"]
    assert:
      - column: "email"
        equals: "{{unique_email}}"
      - column: "name"
        equals: "DB Test User"
      - column: "status"
        equals: "active"
      - column: "deleted_at"
        isNull: true
      - column: "id"
        isNotNull: true

teardown:
  - adapter: http
    action: request
    method: DELETE
    url: "{{baseUrl}}/users/{{captured.user_id}}"
    continueOnError: true
```

### Count Verification

```yaml
name: TC-DB-COUNT-001
description: Verify record count after bulk operation
priority: P1
tags: [database, count]

execute:
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users/bulk-import"
    body:
      users:
        - email: "bulk1-{{$uuid()}}@example.com"
        - email: "bulk2-{{$uuid()}}@example.com"
    assert:
      status: 200

verify:
  - adapter: postgresql
    action: count
    sql: "SELECT COUNT(*) FROM users WHERE email LIKE 'bulk%-@example.com'"
    assert:
      - column: count
        greaterThan: 0
```

---

## Multi-Step Flows

### Authentication Flow

```yaml
name: TC-AUTH-001
description: Login, use token, verify access
priority: P0
tags: [auth, e2e]

variables:
  email: "auth-test-{{$uuid()}}@example.com"
  password: "SecurePass123!"

setup:
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users/register"
    body:
      email: "{{email}}"
      password: "{{password}}"
    capture:
      user_id: "$.id"

execute:
  # Step 1: Login
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/auth/login"
    body:
      email: "{{email}}"
      password: "{{password}}"
    capture:
      access_token: "$.accessToken"
    assert:
      status: 200
      json:
        - path: "$.accessToken"
          exists: true
          type: "string"

  # Step 2: Access protected resource
  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/users/me"
    headers:
      Authorization: "Bearer {{captured.access_token}}"
    assert:
      status: 200
      json:
        - path: "$.email"
          equals: "{{email}}"

teardown:
  - adapter: http
    action: request
    method: DELETE
    url: "{{baseUrl}}/users/{{captured.user_id}}"
    continueOnError: true
```

### Async Job Processing

Test an endpoint that triggers a background job, then poll for completion:

```yaml
name: TC-JOB-001
description: Trigger job and wait for completion
priority: P1
tags: [async, job]

execute:
  # Trigger the job
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/jobs/export"
    body:
      format: "csv"
    capture:
      job_id: "$.jobId"
    assert:
      status: 202

  # Wait and check status
  - adapter: http
    action: request
    delay: 2000
    method: GET
    url: "{{baseUrl}}/jobs/{{captured.job_id}}"
    retry: 5
    assert:
      status: 200
      json:
        - path: "$.status"
          matches: "^(processing|completed)$"
```

---

## Cache Verification

### Redis Cache After API Call

```yaml
name: TC-CACHE-001
description: Verify cache is populated after API call
priority: P1
tags: [cache, redis]

variables:
  unique_email: "cache-{{$uuid()}}@example.com"

setup:
  - adapter: redis
    action: flushPattern
    pattern: "user:*:cache"

execute:
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users"
    body:
      email: "{{unique_email}}"
      name: "Cache Test User"
    capture:
      user_id: "$.id"
    assert:
      status: 201

verify:
  - adapter: redis
    action: get
    key: "user:{{captured.user_id}}:cache"
    assert:
      isNotNull: true
      contains: "{{unique_email}}"

teardown:
  - adapter: http
    action: request
    method: DELETE
    url: "{{baseUrl}}/users/{{captured.user_id}}"
    continueOnError: true
```

### Redis Hash Verification

```yaml
name: TC-CACHE-HASH-001
description: Verify hash data in Redis
priority: P2
tags: [cache, redis]

setup:
  - adapter: redis
    action: hset
    key: "user:123"
    field: "status"
    value: "active"

  - adapter: redis
    action: hset
    key: "user:123"
    field: "email"
    value: "test@example.com"

verify:
  - adapter: redis
    action: hget
    key: "user:123"
    field: "email"
    capture: user_email
    assert:
      contains: "@example.com"

  - adapter: redis
    action: hgetall
    key: "user:123"
    capture: user_data
    assert:
      exists: true

teardown:
  - adapter: redis
    action: del
    key: "user:123"
    continueOnError: true
```

---

## MongoDB Patterns

### Document Create and Verify

```yaml
name: TC-MONGO-001
description: Insert and find MongoDB document
priority: P1
tags: [mongodb, crud]

execute:
  - adapter: mongodb
    action: insertOne
    collection: "users"
    document:
      email: "mongo-{{$uuid()}}@example.com"
      name: "Mongo Test User"
      createdAt: "{{$isoDate()}}"
    capture:
      inserted_id: "insertedId"

verify:
  - adapter: mongodb
    action: findOne
    collection: "users"
    filter:
      _id: "{{captured.inserted_id}}"
    assert:
      - path: "name"
        equals: "Mongo Test User"
      - path: "email"
        contains: "@example.com"

teardown:
  - adapter: mongodb
    action: deleteOne
    collection: "users"
    filter:
      _id: "{{captured.inserted_id}}"
    continueOnError: true
```

### Aggregation Pipeline

```yaml
name: TC-MONGO-AGG-001
description: Test MongoDB aggregation
priority: P2
tags: [mongodb, aggregation]

setup:
  - adapter: mongodb
    action: insertMany
    collection: "orders"
    documents:
      - customerId: "C001"
        amount: 100
        status: "completed"
      - customerId: "C001"
        amount: 200
        status: "completed"
      - customerId: "C002"
        amount: 50
        status: "completed"

execute:
  - adapter: mongodb
    action: aggregate
    collection: "orders"
    pipeline:
      - $match:
          status: "completed"
      - $group:
          _id: "$customerId"
          total: { $sum: "$amount" }
    capture:
      totals: "result"

teardown:
  - adapter: mongodb
    action: deleteMany
    collection: "orders"
    filter:
      customerId: { $in: ["C001", "C002"] }
    continueOnError: true
```

---

## Performance Testing

### Response Time Validation

```yaml
name: TC-PERF-001
description: Verify endpoint response time
priority: P1
tags: [performance]

execute:
  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/health"
    assert:
      status: 200
      duration:
        lessThan: 1000

  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/users?limit=100"
    headers:
      Authorization: "Bearer {{$env(API_TOKEN)}}"
    assert:
      status: 200
      duration:
        lessThan: 2000
      json:
        - path: "$.items"
          type: "array"
```

---

## Environment-Dependent Tests

### Using Environment Variables

```yaml
name: TC-ENV-001
description: Test with environment-specific config
priority: P0
tags: [smoke, e2e]

execute:
  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/health"
    headers:
      Authorization: "Bearer {{$env(API_TOKEN)}}"
      X-API-Key: "{{$env(API_KEY)}}"
    assert:
      status: 200
```

### Using Dynamic Test Data

```yaml
name: TC-DYNAMIC-001
description: Test with generated unique data
priority: P1
tags: [e2e]

variables:
  unique_email: "test-{{$uuid()}}@example.com"
  random_age: "{{$random(18, 65)}}"
  created_at: "{{$isoDate()}}"
  password_hash: "{{$sha256(test-password)}}"

execute:
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users"
    headers:
      Content-Type: "application/json"
      Authorization: "Basic {{$base64(admin:secret)}}"
    body:
      email: "{{unique_email}}"
      age: "{{random_age}}"
      createdAt: "{{created_at}}"
```

---

## Cookie Jar (Session Persistence)

The HTTP adapter includes an automatic cookie jar that persists cookies across steps within a test. Cookies from `Set-Cookie` response headers are stored and automatically sent in subsequent requests via the `Cookie` header.

### Login and Use Session Cookie

```yaml
name: TC-SESSION-001
description: Login with cookies and access protected resource
priority: P0
tags: [auth, cookie, session]

variables:
  email: "session-test-{{$uuid}}@example.com"
  password: "SecurePass123!"

setup:
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users/register"
    body:
      email: "{{email}}"
      password: "{{password}}"
    capture:
      user_id: "$.id"
    assert:
      status: 201

execute:
  # Step 1: Login — server returns Set-Cookie header
  - adapter: http
    action: request
    description: "Login to get session cookie"
    method: POST
    url: "{{baseUrl}}/auth/login"
    body:
      email: "{{email}}"
      password: "{{password}}"
    assert:
      status: 200

  # Step 2: Access protected resource — cookie is sent automatically
  - adapter: http
    action: request
    description: "Access profile using session cookie"
    method: GET
    url: "{{baseUrl}}/users/me"
    assert:
      status: 200
      json:
        - path: "$.email"
          equals: "{{email}}"

  # Step 3: Logout — cookie is still sent
  - adapter: http
    action: request
    description: "Logout using session cookie"
    method: POST
    url: "{{baseUrl}}/auth/logout"
    assert:
      status: 200

teardown:
  - adapter: http
    action: request
    method: DELETE
    url: "{{baseUrl}}/users/{{captured.user_id}}"
    continueOnError: true
```

---

## TOTP (Two-Factor Authentication)

Use the `$totp()` built-in function to generate time-based one-time passwords (RFC 6238, 6-digit, 30s period, HMAC-SHA1). Pass a base32-encoded secret as the argument.

### Login with TOTP

```yaml
name: TC-LOGIN-TOTP-001
description: Login with email, password, and TOTP code
priority: P0
tags: [auth, totp, mfa]

variables:
  email: "totp-user@example.com"
  password: "SecurePass123!"
  totp_secret: "JBSWY3DPEHPK3PXP"

execute:
  # Step 1: Initial login
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/auth/login"
    body:
      email: "{{email}}"
      password: "{{password}}"
    capture:
      mfa_token: "$.mfaToken"
    assert:
      status: 200
      json:
        - path: "$.mfaRequired"
          equals: true

  # Step 2: Submit TOTP code
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/auth/mfa/verify"
    body:
      mfaToken: "{{captured.mfa_token}}"
      code: "{{$totp(JBSWY3DPEHPK3PXP)}}"
    capture:
      access_token: "$.accessToken"
    assert:
      status: 200
      json:
        - path: "$.accessToken"
          exists: true

  # Step 3: Access protected resource with token
  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/users/me"
    headers:
      Authorization: "Bearer {{captured.access_token}}"
    assert:
      status: 200
      json:
        - path: "$.email"
          equals: "{{email}}"
```

---

## File Upload

### Multipart/Form-Data Upload

Upload a file with metadata fields using the HTTP adapter's `multipart` support:

```yaml
name: TC-UPLOAD-001
description: Upload a document with metadata
priority: P1
tags: [upload, file]

execute:
  # Upload file with text fields
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/documents/upload"
    multipart:
      - name: "file"
        file: "./fixtures/sample-report.pdf"
        filename: "report.pdf"
        contentType: "application/pdf"
      - name: "title"
        value: "Monthly Report"
      - name: "category"
        value: "reports"
    capture:
      document_id: "$.id"
    assert:
      status: 201
      json:
        - path: "$.id"
          exists: true
        - path: "$.filename"
          equals: "report.pdf"

  # Verify the upload by fetching the document metadata
  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/documents/{{captured.document_id}}"
    assert:
      status: 200
      json:
        - path: "$.title"
          equals: "Monthly Report"
        - path: "$.category"
          equals: "reports"

teardown:
  - adapter: http
    action: request
    method: DELETE
    url: "{{baseUrl}}/documents/{{captured.document_id}}"
    continueOnError: true
```

### Multiple File Upload

Upload multiple files in a single request:

```yaml
name: TC-UPLOAD-MULTI-001
description: Upload multiple files at once
priority: P2
tags: [upload, file]

execute:
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/gallery/upload"
    multipart:
      - name: "images"
        file: "./fixtures/photo1.jpg"
      - name: "images"
        file: "./fixtures/photo2.jpg"
      - name: "album"
        value: "vacation"
    assert:
      status: 200
      json:
        - path: "$.uploaded"
          equals: 2
```

---

## Smoke Test Pattern

A minimal test that verifies basic connectivity:

```yaml
name: TC-SMOKE-001
description: Basic health check
priority: P0
tags: [smoke]

execute:
  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/health"
    assert:
      status: 200
      duration:
        lessThan: 5000
```
