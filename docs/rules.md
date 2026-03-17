# ut-vet Detection Rules

A comprehensive catalog of all test anti-patterns that ut-vet detects or plans to detect.

---

## P0 — Critical

### `empty-test`

**Status:** ✅ Implemented

Test function body is empty or contains only comments. These tests always pass but verify nothing.

```go
// DETECTED: empty body
func TestCreateUser(t *testing.T) {}

// DETECTED: only comments
func TestDeleteUser(t *testing.T) {
    // TODO: implement this test
}
```

---

### `no-assertion`

**Status:** ✅ Implemented

Test function has no assertion calls. The test runs code but never checks any result.

Recognized assertion patterns:
- `t.Error`, `t.Errorf`, `t.Fatal`, `t.Fatalf`, `t.Fail`, `t.FailNow`
- `assert.*` and `require.*` (testify)

```go
// DETECTED: no assertion calls
func TestCalculate(t *testing.T) {
    result := Calculate(1, 2)
    _ = result
}

// NOT detected: has t.Errorf
func TestCalculate(t *testing.T) {
    result := Calculate(1, 2)
    if result != 3 {
        t.Errorf("expected 3, got %d", result)
    }
}
```

---

### `log-only-test`

**Status:** ✅ Implemented

Test only calls logging/printing functions but has no assertion calls. Extremely common in AI-generated tests.

Recognized log patterns:
- `t.Log`, `t.Logf`
- `fmt.Print`, `fmt.Printf`, `fmt.Println`
- `log.Print`, `log.Printf`, `log.Println`

```go
// DETECTED: only logs
func TestProcess(t *testing.T) {
    result := Process("input")
    t.Logf("result: %v", result)
    fmt.Println("done")
}

// NOT detected: has log AND assertion
func TestProcess(t *testing.T) {
    result := Process("input")
    t.Logf("result: %v", result)
    if result != "expected" {
        t.Errorf("unexpected result: %s", result)
    }
}
```

---

### `trivial-assertion`

**Status:** ✅ Implemented

Assertion checks a constant or literal expression that is always true. The test passes by definition, not because any code logic was verified.

Detected patterns:
- `assert.True(t, true)`
- `assert.False(t, false)`
- `assert.Nil(t, nil)`
- `assert.Equal(t, 1, 1)` (same literal on both sides)
- `assert.Equal(t, "hello", "hello")`
- `assert.NotNil(t, "literal")` (non-nil literal)
- `assert.NotEqual(t, 1, 2)` (both sides are literals)
- `require.*` equivalents of all the above

```go
// DETECTED: asserting a literal true
func TestAlwaysPass(t *testing.T) {
    assert.True(t, true)
}

// DETECTED: comparing identical literals
func TestMathIsStable(t *testing.T) {
    assert.Equal(t, 42, 42)
}

// DETECTED: asserting nil is nil
func TestNilIsNil(t *testing.T) {
    require.Nil(t, nil)
}

// NOT detected: comparing variable to literal
func TestRealCheck(t *testing.T) {
    result := Compute()
    assert.Equal(t, 42, result)
}
```

---

## P1 — High Value

### `error-not-checked`

**Status:** 🔲 Planned

Function under test returns an error but the test ignores it by assigning to `_` or not checking it at all.

```go
// WOULD BE DETECTED: error assigned to blank identifier
func TestSave(t *testing.T) {
    _, _ = repo.Save(entity)
    // test continues without checking error
}

// WOULD BE DETECTED: error return value completely ignored
func TestSave(t *testing.T) {
    repo.Save(entity)
    // Save returns (Entity, error) but both are dropped
}

// NOT detected: error is checked
func TestSave(t *testing.T) {
    _, err := repo.Save(entity)
    require.NoError(t, err)
}
```

---

### `no-code-under-test`

**Status:** 🔲 Planned

Test never calls any function from the package being tested. It only calls assertion helpers, standard library, or test utilities.

```go
// WOULD BE DETECTED: only calls stdlib and assertions
func TestNothing(t *testing.T) {
    x := strings.ToUpper("hello")
    assert.Equal(t, "HELLO", x)
}

// NOT detected: calls package function
func TestToUpper(t *testing.T) {
    result := mypackage.Transform("hello")
    assert.Equal(t, "HELLO", result)
}
```

---

### `only-nil-check`

**Status:** 🔲 Planned

Test only asserts that the error is nil but never checks the actual return value.

```go
// WOULD BE DETECTED: only checks error, ignores result
func TestGetUser(t *testing.T) {
    _, err := service.GetUser(42)
    assert.NoError(t, err)
}

// NOT detected: checks both error and result
func TestGetUser(t *testing.T) {
    user, err := service.GetUser(42)
    assert.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

---

### `zero-value-input`

**Status:** 🔲 Planned

Function under test is called with only zero-values (`nil`, `0`, `""`, `false`, empty struct) as arguments, suggesting no meaningful test scenario was devised.

```go
// WOULD BE DETECTED: all zero-value arguments
func TestCreateUser(t *testing.T) {
    user, err := service.CreateUser("", 0, false)
    assert.NoError(t, err)
    assert.NotNil(t, user)
}

// NOT detected: meaningful inputs
func TestCreateUser(t *testing.T) {
    user, err := service.CreateUser("Alice", 30, true)
    assert.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

---

## P2 — Advanced

### `tautological-assert`

**Status:** 🔲 Planned

Assertion compares a variable to itself, which is always true.

```go
// WOULD BE DETECTED: comparing variable to itself
func TestSelfCompare(t *testing.T) {
    result := Compute()
    assert.Equal(t, result, result)
}
```

---

### `dead-assertion`

**Status:** 🔲 Planned

Assertion appears after `t.Fatal`, `t.FailNow`, or `return` — it can never be reached.

```go
// WOULD BE DETECTED: assertion after t.Fatal is unreachable
func TestUnreachable(t *testing.T) {
    result, err := DoSomething()
    if err != nil {
        t.Fatal(err)
    }
    t.Fatal("always fails here")
    assert.Equal(t, 42, result) // dead code
}
```

---

### `no-arrange`

**Status:** 🔲 Planned

Test has assertions but no meaningful setup — calls the function under test with only zero-value or nil arguments and no prior arrangement.

```go
// WOULD BE DETECTED: no arrange phase, nil arguments
func TestHandler(t *testing.T) {
    handler := NewHandler(nil, nil)
    err := handler.Process(nil)
    assert.NoError(t, err)
}

// NOT detected: has meaningful setup
func TestHandler(t *testing.T) {
    db := setupTestDB(t)
    logger := zap.NewNop()
    handler := NewHandler(db, logger)
    err := handler.Process(newTestRequest())
    assert.NoError(t, err)
}
```

---

## Summary

| Rule | Severity | Status | Description |
|------|----------|--------|-------------|
| `empty-test` | P0 | ✅ Implemented | Empty or comments-only test body |
| `no-assertion` | P0 | ✅ Implemented | No assertion calls |
| `log-only-test` | P0 | ✅ Implemented | Only logs/prints, no assertions |
| `trivial-assertion` | P0 | ✅ Implemented | Assertion on constant expression |
| `error-not-checked` | P1 | 🔲 Planned | Returned error ignored |
| `no-code-under-test` | P1 | 🔲 Planned | Never calls package functions |
| `only-nil-check` | P1 | 🔲 Planned | Only checks err == nil |
| `zero-value-input` | P1 | 🔲 Planned | All arguments are zero-values |
| `tautological-assert` | P2 | 🔲 Planned | Variable compared to itself |
| `dead-assertion` | P2 | 🔲 Planned | Assertion after fatal/return |
| `no-arrange` | P2 | 🔲 Planned | No meaningful test setup |
