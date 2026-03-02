# TOTP Built-in Function Design

**Date:** 2026-03-02
**Status:** Approved

## Goal

Add a `$totp` built-in function so YAML tests can generate valid TOTP codes from a shared secret, enabling end-to-end testing of 2FA login flows.

## Usage

```yaml
steps:
  - adapter: http
    action: request
    params:
      method: POST
      url: /auth/verify-totp
      body:
        code: "{{$totp(JBSWY3DPEHPK3PXP)}}"
        # or from env:
        # code: "{{$totp($env(TOTP_SECRET))}}"
```

## Approach

Single entry in the existing `BUILT_IN_FUNCTIONS` registry in `src/core/variable-interpolator.ts`. Two helper functions in the same file. No new files, no new dependencies.

## TOTP Algorithm (RFC 6238)

Parameters (standard defaults, not configurable):
- **Digits:** 6
- **Period:** 30 seconds
- **Algorithm:** HMAC-SHA1

Steps:
1. Decode base32 secret to bytes
2. Compute time counter: `floor(unix_seconds / 30)`
3. HMAC-SHA1(secret_bytes, counter_as_8_byte_big_endian)
4. Dynamic truncation: extract 4 bytes at offset determined by last nibble
5. Mask to 31 bits, modulo 10^6, zero-pad to 6 digits

## Implementation

### Helpers (same file, above BUILT_IN_FUNCTIONS)

- `base32Decode(encoded: string): Buffer` — RFC 4648 base32 decoding, strips padding, throws on invalid characters
- `generateTOTP(secret: string): string` — full TOTP computation, returns 6-digit string

### Built-in function

```typescript
$totp: (secret: string) => {
    if (!secret) {
        throw new InterpolationError('TOTP secret is required', '$totp()');
    }
    return generateTOTP(secret);
}
```

### Import change

Add `createHmac` to existing `node:crypto` import.

## Error Handling

- Missing secret: `InterpolationError` with message "TOTP secret is required"
- Invalid base32: `InterpolationError` with message "Invalid base32 character in TOTP secret: X"

## Files Modified

- `src/core/variable-interpolator.ts` — only file touched

## Dependencies

None new. Uses `createHmac` from `node:crypto` (already partially imported).
