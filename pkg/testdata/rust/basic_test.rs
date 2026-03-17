// Fixture file for ut-vet Rust parser tests.
// This is NOT compiled by Rust — it is parsed by ut-vet's text-based parser.

#[cfg(test)]
mod tests {
    use super::*;

    // SHOULD TRIGGER: empty-test — empty body
    #[test]
    fn test_empty() {}

    // SHOULD TRIGGER: empty-test — only comments
    #[test]
    fn test_only_comments() {
        // TODO: implement this test
    }

    // SHOULD TRIGGER: no-assertion — no assert calls
    #[test]
    fn test_no_assertion() {
        let result = compute(1, 2);
        let _ = result;
    }

    // SHOULD TRIGGER: log-only-test — only println
    #[test]
    fn test_log_only() {
        let result = compute(1, 2);
        println!("result: {}", result);
    }

    // SHOULD TRIGGER: trivial-assertion — assert!(true)
    #[test]
    fn test_trivial_true() {
        assert!(true);
    }

    // SHOULD TRIGGER: trivial-assertion — assert_eq! with same literals
    #[test]
    fn test_trivial_eq() {
        assert_eq!(1, 1);
    }

    // SHOULD NOT TRIGGER: real assertion
    #[test]
    fn test_good_assertion() {
        let result = compute(1, 2);
        assert_eq!(result, 3);
    }

    // SHOULD NOT TRIGGER: real assertion with assert!
    #[test]
    fn test_good_assert() {
        let result = is_valid("hello");
        assert!(result);
    }

    // SHOULD TRIGGER: tautological-assert — comparing variable to itself
    #[test]
    fn test_tautological() {
        let result = compute(1, 2);
        assert_eq!(result, result);
    }

    // SHOULD TRIGGER: no-assertion — uses panic but not as test assertion
    #[test]
    fn test_panic_not_assert() {
        let _x = compute(1, 2);
    }

    // SHOULD NOT TRIGGER: has should_panic attribute
    #[test]
    #[should_panic]
    fn test_should_panic() {
        divide(1, 0);
    }

    // Async test
    #[tokio::test]
    async fn test_async_good() {
        let result = fetch_data().await;
        assert!(result.is_ok());
    }

    // SHOULD TRIGGER: async test with no assertion
    #[tokio::test]
    async fn test_async_no_assert() {
        let result = fetch_data().await;
        let _ = result;
    }

    // SHOULD TRIGGER: zero-value-input
    #[test]
    fn test_zero_value() {
        let result = create_user("", 0, false);
        assert!(result.is_ok());
    }

    // SHOULD NOT TRIGGER: meaningful inputs
    #[test]
    fn test_meaningful_inputs() {
        let result = create_user("Alice", 30, true);
        assert!(result.is_ok());
        assert_eq!(result.unwrap().name, "Alice");
    }

    // SHOULD TRIGGER: error not checked — unwrap_or_default swallows error
    #[test]
    fn test_error_swallowed() {
        let result = parse_config("bad").unwrap_or_default();
        let _ = result;
    }
}
