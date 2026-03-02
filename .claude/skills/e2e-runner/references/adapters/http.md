# HTTP Adapter

For testing REST APIs and HTTP endpoints.

## Action: `request`

Execute HTTP request with full REST support.

```yaml
- adapter: http
  action: request
  method: GET | POST | PUT | PATCH | DELETE | HEAD | OPTIONS
  url: string                        # Full URL or relative to baseUrl
  headers?: Record<string, string>   # Request headers
  body?: any                         # Request body (auto-JSON stringified)
  multipart?: MultipartField[]       # Multipart/form-data fields (mutually exclusive with body)
  query?: Record<string, string>     # Query parameters
  timeout?: number                   # Request timeout (ms)
  followRedirects?: boolean          # Follow redirects (default: true)
```

## Multipart/Form-Data Uploads

Upload files and send mixed text/file fields using `multipart`. When `multipart` is used, the adapter builds a `FormData` body and lets `fetch` set the `Content-Type` header with the correct multipart boundary automatically.

> **`multipart` and `body` are mutually exclusive.** Use one or the other, not both.

### Multipart Field Schema

Each entry in the `multipart` array accepts:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Form field name |
| `file` | string | No* | Path to the file to upload |
| `value` | string | No* | Text field value |
| `filename` | string | No | Override the uploaded filename (defaults to basename of `file`) |
| `contentType` | string | No | MIME type override for file uploads |

\* Each entry must have either `file` or `value` (but not both).

### Multipart Examples

**Simple file upload:**
```yaml
- adapter: http
  action: request
  method: POST
  url: "{{baseUrl}}/upload"
  multipart:
    - name: "file"
      file: "./fixtures/test-image.png"
  assert:
    status: 200
```

**File upload with text fields:**
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
  assert:
    status: 200
```

**Custom filename and content type:**
```yaml
- adapter: http
  action: request
  method: POST
  url: "{{baseUrl}}/documents"
  multipart:
    - name: "document"
      file: "./fixtures/report.pdf"
      filename: "quarterly-report.pdf"
      contentType: "application/pdf"
    - name: "title"
      value: "Q4 Financial Report"
  assert:
    status: 201
```

## Examples

**GET Request:**
```yaml
- adapter: http
  action: request
  method: GET
  url: "{{baseUrl}}/users/123"
  headers:
    Authorization: "Bearer {{access_token}}"
  assert:
    status: 200
```

**POST Request:**
```yaml
- adapter: http
  action: request
  method: POST
  url: "{{baseUrl}}/users"
  headers:
    Content-Type: "application/json"
  body:
    email: "user@example.com"
    name: "John Doe"
  capture:
    user_id: "$.id"
  assert:
    status: 201
    json:
      - path: "$.id"
        exists: true
```

**With Query Parameters:**
```yaml
- adapter: http
  action: request
  method: GET
  url: "{{baseUrl}}/search"
  query:
    q: "test"
    limit: "10"
  assert:
    status: 200
```

## HTTP Assertions

```yaml
assert:
  # Status code
  status: 200                        # Exact status
  status: [200, 201]                 # One of multiple
  statusRange: [200, 299]            # Range check

  # Headers
  headers:
    Content-Type: "application/json"
    X-Custom: "/pattern/"            # Regex pattern

  # JSON body assertions
  json:
    - path: "$.id"
      exists: true
    - path: "$.name"
      equals: "John"
    - path: "$.email"
      contains: "@example.com"
    - path: "$.age"
      greaterThan: 18
      lessThan: 100
    - path: "$.tags"
      length: 3
    - path: "$.status"
      matches: "^(active|pending)$"
    - path: "$.type"
      type: "string"

  # Raw body assertions
  body:
    contains: "success"
    matches: "\\d{4}-\\d{2}-\\d{2}"
    equals: "OK"

  # Response time
  duration:
    lessThan: 1000                   # Response under 1 second
    greaterThan: 100
```

## Value Capture

Extract values from HTTP responses using JSONPath:

```yaml
- adapter: http
  action: request
  url: "{{baseUrl}}/users"
  capture:
    user_id: "$.id"              # Simple path
    user_email: "$.email"
    first_item: "$.items[0].id"  # Array access
    all_ids: "$.items[*].id"     # Wildcard
```

Use captured values in subsequent steps with `{{captured.keyName}}`:

```yaml
- adapter: http
  action: request
  method: GET
  url: "{{baseUrl}}/users/{{captured.user_id}}"
```
