// Fixture file for ut-vet Rust parser — advanced patterns.

#[cfg(test)]
mod tests {
    use super::*;

    // SHOULD NOT TRIGGER: .unwrap() is an implicit assertion
    #[test]
    fn test_unwrap_is_assertion() {
        let result = parse_config("good.toml").unwrap();
        assert_eq!(result.name, "test");
    }

    // SHOULD NOT TRIGGER: .expect() is an implicit assertion
    #[test]
    fn test_expect_is_assertion() {
        let conn = connect_db().expect("db should be available");
        assert!(conn.is_alive());
    }

    // SHOULD NOT TRIGGER: only .unwrap() but still valid (panics on error)
    #[test]
    fn test_unwrap_only() {
        let _result = compute(42).unwrap();
    }

    // SHOULD TRIGGER: no-assertion — .unwrap_or_default() swallows errors
    #[test]
    fn test_swallowed_error() {
        let result = parse_config("bad").unwrap_or_default();
        let _ = result;
    }

    // SHOULD NOT TRIGGER: Result return type with ? operator
    #[test]
    fn test_result_return() -> Result<(), Box<dyn std::error::Error>> {
        let result = parse_config("good.toml")?;
        assert_eq!(result.name, "test");
        Ok(())
    }

    // SHOULD TRIGGER: empty-test — async empty
    #[tokio::test]
    async fn test_async_empty() {}

    // SHOULD NOT TRIGGER: async with assertion
    #[tokio::test]
    async fn test_async_with_assert() {
        let data = fetch_data("api/users").await.unwrap();
        assert!(!data.is_empty());
    }

    // SHOULD TRIGGER: dead-assertion — assertion after panic!
    #[test]
    fn test_dead_after_panic() {
        panic!("always fails");
        assert_eq!(1, 2);
    }

    // SHOULD TRIGGER: log-only-test — multiple print macros
    #[test]
    fn test_dbg_only() {
        let x = compute(1);
        dbg!(x);
        eprintln!("debug: {}", x);
    }

    // SHOULD NOT TRIGGER: has both log and assertion
    #[test]
    fn test_log_and_assert() {
        let result = compute(42);
        println!("result = {}", result);
        assert_eq!(result, 42);
    }

    // SHOULD TRIGGER: no-assertion — method chain but no assertion
    #[test]
    fn test_builder_no_assert() {
        let config = ConfigBuilder::new()
            .with_name("test")
            .with_timeout(30)
            .build();
        let _ = config;
    }

    // SHOULD NOT TRIGGER: #[should_panic(expected = "...")]
    #[test]
    #[should_panic(expected = "division by zero")]
    fn test_panic_expected_message() {
        divide(1, 0);
    }

    // SHOULD NOT TRIGGER: uses assert with variable
    #[test]
    fn test_assert_variable() {
        let valid = validate("hello@example.com");
        assert!(valid);
    }

    // SHOULD TRIGGER: trivial-assertion — assert_ne! with both literals
    #[test]
    fn test_trivial_ne() {
        assert_ne!(1, 2);
    }
}
