/**
 * Tests for HTTP adapter multipart/form-data support.
 *
 * Verifies that the HTTP adapter correctly builds FormData bodies
 * for file upload requests and validates multipart parameters.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import * as fs from 'node:fs';
import * as path from 'node:path';

// ---------------------------------------------------------------------------
// We test the adapter through its public execute() interface.
// fetch is globally mocked so no real HTTP calls are made.
// ---------------------------------------------------------------------------

describe('HTTP Adapter - Multipart/Form-Data Support', () => {
  let HTTPAdapter: typeof import('../../src/adapters/http.adapter').HTTPAdapter;
  let originalFetch: typeof globalThis.fetch;

  const mockLogger = {
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  };

  const mockCtx = {
    variables: {},
    captured: {},
    capture: vi.fn(),
    logger: mockLogger,
    baseUrl: 'http://localhost:3000',
    cookieJar: new Map<string, string>(),
  };

  beforeEach(async () => {
    // Dynamic import to pick up the patched module
    const mod = await import('../../src/adapters/http.adapter');
    HTTPAdapter = mod.HTTPAdapter;

    // Save original fetch and install mock
    originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      statusText: 'OK',
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => ({ success: true }),
      text: async () => '{"success":true}',
    });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  // -----------------------------------------------------------------------
  // Interface contract: multipart field exists on HTTPRequestParams
  // -----------------------------------------------------------------------
  it('should accept multipart field in params', async () => {
    const adapter = new HTTPAdapter({ baseUrl: 'http://localhost:3000' }, mockLogger);
    await adapter.connect();

    // Executing with multipart field should not throw
    const result = await adapter.execute('request', {
      method: 'POST',
      url: 'http://localhost:3000/upload',
      multipart: [
        { name: 'description', value: 'test upload' },
      ],
    }, mockCtx);

    expect(result.success).toBe(true);
  });

  // -----------------------------------------------------------------------
  // FormData body construction
  // -----------------------------------------------------------------------
  it('should build FormData body with text fields', async () => {
    const adapter = new HTTPAdapter({ baseUrl: 'http://localhost:3000' }, mockLogger);
    await adapter.connect();

    await adapter.execute('request', {
      method: 'POST',
      url: 'http://localhost:3000/upload',
      multipart: [
        { name: 'title', value: 'My Document' },
        { name: 'description', value: 'A test document' },
      ],
    }, mockCtx);

    const fetchCall = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    const fetchOptions = fetchCall[1] as RequestInit;

    // Body should be a FormData instance
    expect(fetchOptions.body).toBeInstanceOf(FormData);

    const formData = fetchOptions.body as FormData;
    expect(formData.get('title')).toBe('My Document');
    expect(formData.get('description')).toBe('A test document');
  });

  it('should build FormData body with file fields', async () => {
    // Create a temp file to upload
    const tmpDir = path.join(__dirname, '..', 'fixtures');
    const tmpFile = path.join(tmpDir, 'test-upload.txt');

    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(tmpFile, 'Hello, this is test file content');

    try {
      const adapter = new HTTPAdapter({ baseUrl: 'http://localhost:3000' }, mockLogger);
      await adapter.connect();

      await adapter.execute('request', {
        method: 'POST',
        url: 'http://localhost:3000/upload',
        multipart: [
          { name: 'file', file: tmpFile },
        ],
      }, mockCtx);

      const fetchCall = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const fetchOptions = fetchCall[1] as RequestInit;

      expect(fetchOptions.body).toBeInstanceOf(FormData);

      const formData = fetchOptions.body as FormData;
      const fileEntry = formData.get('file');
      expect(fileEntry).toBeTruthy();
      // The file entry should be a Blob (or File)
      expect(fileEntry).toBeInstanceOf(Blob);
    } finally {
      // Cleanup
      fs.unlinkSync(tmpFile);
    }
  });

  it('should mix file and text fields in a single multipart request', async () => {
    const tmpDir = path.join(__dirname, '..', 'fixtures');
    const tmpFile = path.join(tmpDir, 'test-mixed.txt');

    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(tmpFile, 'mixed content');

    try {
      const adapter = new HTTPAdapter({ baseUrl: 'http://localhost:3000' }, mockLogger);
      await adapter.connect();

      await adapter.execute('request', {
        method: 'POST',
        url: 'http://localhost:3000/upload',
        multipart: [
          { name: 'document', file: tmpFile },
          { name: 'title', value: 'Report' },
          { name: 'category', value: 'finance' },
        ],
      }, mockCtx);

      const fetchCall = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const fetchOptions = fetchCall[1] as RequestInit;
      const formData = fetchOptions.body as FormData;

      expect(formData.get('document')).toBeInstanceOf(Blob);
      expect(formData.get('title')).toBe('Report');
      expect(formData.get('category')).toBe('finance');
    } finally {
      fs.unlinkSync(tmpFile);
    }
  });

  // -----------------------------------------------------------------------
  // Content-Type header behavior
  // -----------------------------------------------------------------------
  it('should NOT set Content-Type header when using multipart', async () => {
    const adapter = new HTTPAdapter({ baseUrl: 'http://localhost:3000' }, mockLogger);
    await adapter.connect();

    await adapter.execute('request', {
      method: 'POST',
      url: 'http://localhost:3000/upload',
      multipart: [
        { name: 'field', value: 'test' },
      ],
    }, mockCtx);

    const fetchCall = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    const fetchOptions = fetchCall[1] as RequestInit;
    const headers = fetchOptions.headers as Record<string, string>;

    // Content-Type should be absent so fetch can set multipart boundary
    expect(headers['Content-Type']).toBeUndefined();
  });

  // -----------------------------------------------------------------------
  // Existing JSON body unchanged
  // -----------------------------------------------------------------------
  it('should still send JSON body when multipart is not used', async () => {
    const adapter = new HTTPAdapter({ baseUrl: 'http://localhost:3000' }, mockLogger);
    await adapter.connect();

    await adapter.execute('request', {
      method: 'POST',
      url: 'http://localhost:3000/users',
      body: { email: 'test@example.com' },
    }, mockCtx);

    const fetchCall = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    const fetchOptions = fetchCall[1] as RequestInit;

    expect(typeof fetchOptions.body).toBe('string');
    expect(JSON.parse(fetchOptions.body as string)).toEqual({ email: 'test@example.com' });
  });

  // -----------------------------------------------------------------------
  // Custom filename and contentType
  // -----------------------------------------------------------------------
  it('should use custom filename when provided', async () => {
    const tmpDir = path.join(__dirname, '..', 'fixtures');
    const tmpFile = path.join(tmpDir, 'test-custom.txt');

    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(tmpFile, 'custom named file');

    try {
      const adapter = new HTTPAdapter({ baseUrl: 'http://localhost:3000' }, mockLogger);
      await adapter.connect();

      await adapter.execute('request', {
        method: 'POST',
        url: 'http://localhost:3000/upload',
        multipart: [
          { name: 'file', file: tmpFile, filename: 'custom-name.txt', contentType: 'text/plain' },
        ],
      }, mockCtx);

      const fetchCall = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const fetchOptions = fetchCall[1] as RequestInit;
      const formData = fetchOptions.body as FormData;
      const fileEntry = formData.get('file') as File;

      expect(fileEntry).toBeInstanceOf(Blob);
      // The File/Blob should have the custom name
      expect(fileEntry.name).toBe('custom-name.txt');
    } finally {
      fs.unlinkSync(tmpFile);
    }
  });

  // -----------------------------------------------------------------------
  // Error: entry with neither file nor value
  // -----------------------------------------------------------------------
  it('should throw AdapterError for entry with neither file nor value', async () => {
    const adapter = new HTTPAdapter({ baseUrl: 'http://localhost:3000' }, mockLogger);
    await adapter.connect();

    await expect(
      adapter.execute('request', {
        method: 'POST',
        url: 'http://localhost:3000/upload',
        multipart: [
          { name: 'invalid_entry' },
        ],
      }, mockCtx)
    ).rejects.toThrow(/file.*value|value.*file/i);
  });
});

// ---------------------------------------------------------------------------
// YAML Loader validation tests
// ---------------------------------------------------------------------------
describe('YAML Loader - Multipart Validation', () => {
  it('should reject step with both body and multipart', async () => {
    const tmpDir = path.join(__dirname, '..', 'fixtures');
    const tmpFile = path.join(tmpDir, 'body-and-multipart.test.yaml');

    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(tmpFile, `
name: TC-INVALID-001
description: Invalid test with body and multipart
execute:
  - adapter: http
    action: request
    method: POST
    url: "http://localhost:3000/upload"
    body:
      key: "value"
    multipart:
      - name: "file"
        value: "test"
`);

    try {
      const { loadYAMLTest } = await import('../../src/core/yaml-loader');
      await expect(loadYAMLTest(tmpFile)).rejects.toThrow(/body.*multipart|multipart.*body/i);
    } finally {
      fs.unlinkSync(tmpFile);
    }
  });

  it('should reject multipart entry without name', async () => {
    const tmpDir = path.join(__dirname, '..', 'fixtures');
    const tmpFile = path.join(tmpDir, 'no-name.test.yaml');

    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(tmpFile, `
name: TC-INVALID-002
description: Invalid multipart entry without name
execute:
  - adapter: http
    action: request
    method: POST
    url: "http://localhost:3000/upload"
    multipart:
      - file: "./test.txt"
`);

    try {
      const { loadYAMLTest } = await import('../../src/core/yaml-loader');
      await expect(loadYAMLTest(tmpFile)).rejects.toThrow(/name/i);
    } finally {
      fs.unlinkSync(tmpFile);
    }
  });

  it('should reject multipart entry without file or value', async () => {
    const tmpDir = path.join(__dirname, '..', 'fixtures');
    const tmpFile = path.join(tmpDir, 'no-file-value.test.yaml');

    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(tmpFile, `
name: TC-INVALID-003
description: Invalid multipart entry without file or value
execute:
  - adapter: http
    action: request
    method: POST
    url: "http://localhost:3000/upload"
    multipart:
      - name: "field"
`);

    try {
      const { loadYAMLTest } = await import('../../src/core/yaml-loader');
      await expect(loadYAMLTest(tmpFile)).rejects.toThrow(/file.*value|value.*file/i);
    } finally {
      fs.unlinkSync(tmpFile);
    }
  });

  it('should accept valid multipart step', async () => {
    const tmpDir = path.join(__dirname, '..', 'fixtures');
    const tmpFile = path.join(tmpDir, 'valid-multipart.test.yaml');

    fs.mkdirSync(tmpDir, { recursive: true });
    fs.writeFileSync(tmpFile, `
name: TC-VALID-MULTIPART-001
description: Valid test with multipart
execute:
  - adapter: http
    action: request
    method: POST
    url: "http://localhost:3000/upload"
    multipart:
      - name: "file"
        file: "./fixtures/test.txt"
      - name: "description"
        value: "A test upload"
`);

    try {
      const { loadYAMLTest } = await import('../../src/core/yaml-loader');
      const test = await loadYAMLTest(tmpFile);
      expect(test.execute[0].params.multipart).toBeDefined();
    } finally {
      fs.unlinkSync(tmpFile);
    }
  });
});
