# Deep Review Checklists

## 1. Security (OWASP Top 10 + extras)
- Injection vulnerabilities (SQL, command, LDAP, XSS, SSTI, CRLF)
- Broken authentication / session management
- Sensitive data exposure (secrets, PII, tokens in logs or source)
- XML/JSON External Entity (XXE) processing
- Broken access control and privilege escalation paths
- Security misconfiguration (default creds, open ports, verbose errors)
- Cross-site scripting (Stored, Reflected, DOM-based)
- Insecure deserialization
- Using components with known vulnerabilities (outdated deps)
- Insufficient logging and monitoring
- Path traversal and file inclusion
- Race conditions, TOCTOU, deadlocks
- Cryptographic weaknesses (weak algorithms, key reuse, bad IV)
- Supply chain / dependency confusion risks

## 2. Performance
- Algorithmic complexity: O(n²) or worse in hot paths
- Unnecessary allocations, copies, or clones
- Database N+1 query patterns
- Missing indexes on frequently queried fields
- Blocking I/O in async contexts
- Unbounded loops or recursion
- Memory leaks or resource leaks (file handles, sockets)
- Caching opportunities

## 3. Maintainability & Code Quality
- Functions / methods exceeding 50 lines
- Deep nesting (>4 levels)
- Duplicated logic (DRY violations)
- Magic numbers and strings without named constants
- Misleading names (variables, functions, types)
- Dead code and unused imports
- Overly complex conditionals
- Coupling: tight coupling between unrelated modules

## 4. Error Handling
- Swallowed errors (empty catch blocks, `unwrap()` without context)
- Panic-able paths in library code
- Missing input validation at trust boundaries
- Unclear error messages that hinder debugging
- Error type inconsistency across the codebase

## 5. Test Coverage
- Missing unit tests for critical logic
- Missing integration tests for external boundaries
- Tests with no assertions
- Tests that are brittle (time-dependent, order-dependent)
- Missing negative / edge-case tests
- Mocking strategy concerns

## 6. API Design
- Unclear or inconsistent naming conventions
- Functions with too many parameters (>5)
- Mutable global state
- Missing or incorrect use of visibility modifiers
- Breaking changes risk in public interfaces
- Lack of builder or fluent patterns where appropriate

## 7. Documentation
- Missing doc comments on public items
- Outdated or misleading comments
- Undocumented panics, unsafe blocks, or invariants
- Missing README or high-level architectural overview

## 8. Architectural Concerns
- Single Responsibility Principle violations
- Circular dependencies
- Missing abstraction layers
- Hardcoded configuration that should be externalised
- Observability gaps (missing tracing, metrics, structured logs)
