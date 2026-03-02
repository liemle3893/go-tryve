# TOTP Built-in Function Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `$totp(secret)` built-in function that generates RFC 6238 TOTP codes, enabling YAML tests to authenticate against 2FA-protected endpoints.

**Architecture:** A `$totp` entry in the existing `BUILT_IN_FUNCTIONS` registry in `variable-interpolator.ts`, backed by two helper functions (`base32Decode` and `generateTOTP`) in the same file. Uses `createHmac` from Node.js `crypto` — no new dependencies.

**Tech Stack:** TypeScript, `node:crypto` (createHmac, HMAC-SHA1)

---

### Task 1: Add `createHmac` to crypto import

**Files:**
- Modify: `src/core/variable-interpolator.ts:7`

**Step 1: Update the import**

Change line 7 from:
```typescript
import { createHash, randomUUID } from 'node:crypto';
```
to:
```typescript
import { createHash, createHmac, randomUUID } from 'node:crypto';
```

**Step 2: Build to verify**

Run: `npm run build`
Expected: Clean build (no errors). `createHmac` is a valid export from `node:crypto`.

**Step 3: Commit**

```bash
git add src/core/variable-interpolator.ts
git commit -m "chore: import createHmac from node:crypto for TOTP support"
```

---

### Task 2: Add `base32Decode` helper function

**Files:**
- Modify: `src/core/variable-interpolator.ts` (insert before `BUILT_IN_FUNCTIONS`)

**Step 1: Add the helper**

Insert between the imports (line 11) and the `BUILT_IN_FUNCTIONS` block (line 20), inside the "Built-in Functions" section:

```typescript
/**
 * Decode a base32-encoded string (RFC 4648) to a Buffer
 */
function base32Decode(encoded: string): Buffer {
    const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
    const stripped = encoded.replace(/=+$/, '').toUpperCase();

    let bits = 0;
    let value = 0;
    const output: number[] = [];

    for (const char of stripped) {
        const idx = alphabet.indexOf(char);
        if (idx === -1) {
            throw new InterpolationError(
                `Invalid base32 character in TOTP secret: ${char}`,
                '$totp()'
            );
        }
        value = (value << 5) | idx;
        bits += 5;
        if (bits >= 8) {
            bits -= 8;
            output.push((value >>> bits) & 0xff);
        }
    }

    return Buffer.from(output);
}
```

**Step 2: Build to verify**

Run: `npm run build`
Expected: Clean build. The function is defined but not yet called — no errors.

**Step 3: Commit**

```bash
git add src/core/variable-interpolator.ts
git commit -m "feat: add base32Decode helper for TOTP secret decoding"
```

---

### Task 3: Add `generateTOTP` helper function

**Files:**
- Modify: `src/core/variable-interpolator.ts` (insert after `base32Decode`)

**Step 1: Add the helper**

Insert directly after `base32Decode`:

```typescript
/**
 * Generate a TOTP code per RFC 6238 (6 digits, 30s period, HMAC-SHA1)
 */
function generateTOTP(secret: string): string {
    const key = base32Decode(secret);
    const epoch = Math.floor(Date.now() / 1000);
    const counter = Math.floor(epoch / 30);

    // Encode counter as 8-byte big-endian buffer
    const counterBuf = Buffer.alloc(8);
    counterBuf.writeUInt32BE(Math.floor(counter / 0x100000000), 0);
    counterBuf.writeUInt32BE(counter & 0xffffffff, 4);

    // HMAC-SHA1
    const hmac = createHmac('sha1', key).update(counterBuf).digest();

    // Dynamic truncation
    const offset = hmac[hmac.length - 1] & 0x0f;
    const code =
        ((hmac[offset] & 0x7f) << 24) |
        ((hmac[offset + 1] & 0xff) << 16) |
        ((hmac[offset + 2] & 0xff) << 8) |
        (hmac[offset + 3] & 0xff);

    // 6-digit zero-padded string
    return (code % 1000000).toString().padStart(6, '0');
}
```

**Step 2: Build to verify**

Run: `npm run build`
Expected: Clean build.

**Step 3: Commit**

```bash
git add src/core/variable-interpolator.ts
git commit -m "feat: add generateTOTP helper implementing RFC 6238"
```

---

### Task 4: Register `$totp` in `BUILT_IN_FUNCTIONS`

**Files:**
- Modify: `src/core/variable-interpolator.ts` (add entry to `BUILT_IN_FUNCTIONS`)

**Step 1: Add the function entry**

Add after the `$trim` entry (the last entry in `BUILT_IN_FUNCTIONS`), before the closing `};`:

```typescript
  // TOTP
  $totp: (secret: string) => {
    if (!secret) {
      throw new InterpolationError('TOTP secret is required', '$totp()');
    }
    return generateTOTP(secret);
  },
```

**Step 2: Build to verify**

Run: `npm run build`
Expected: Clean build.

**Step 3: Manual verification**

Quick smoke test using Node.js REPL to verify the TOTP output against a known secret:

```bash
node -e "
const { BUILT_IN_FUNCTIONS } = require('./dist/core/variable-interpolator');
const code = BUILT_IN_FUNCTIONS['\$totp']('JBSWY3DPEHPK3PXP');
console.log('TOTP code:', code);
console.log('Is 6 digits:', /^\d{6}$/.test(code));
"
```

Expected: A 6-digit numeric string.

**Step 4: Commit**

```bash
git add src/core/variable-interpolator.ts
git commit -m "feat: register \$totp built-in function for 2FA test support"
```

---

### Task 5: Final verification

**Step 1: Clean build**

Run: `npm run clean && npm run build`
Expected: Clean build with no errors.

**Step 2: Verify interpolation end-to-end**

Test that `{{$totp(SECRET)}}` resolves correctly through the interpolation engine:

```bash
node -e "
const { interpolate } = require('./dist/core/variable-interpolator');
const ctx = { variables: {}, captured: {}, baseUrl: '', env: {} };
const result = interpolate('{{$totp(JBSWY3DPEHPK3PXP)}}', ctx);
console.log('Interpolated TOTP:', result);
console.log('Is 6 digits:', /^\d{6}$/.test(result));
"
```

Expected: A 6-digit numeric string — confirms the full `{{$totp(...)}}` → interpolation → function evaluation → TOTP generation pipeline works.

---

## Verification Summary

- `npm run build` succeeds with no type errors
- `$totp(JBSWY3DPEHPK3PXP)` returns a 6-digit string
- `{{$totp(SECRET)}}` resolves through the interpolation engine
- Invalid base32 throws `InterpolationError`
- Missing secret throws `InterpolationError`
- No new dependencies added
- Only `src/core/variable-interpolator.ts` modified
