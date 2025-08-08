pub mod temporal {
    pub mod omes {
        #[allow(clippy::all)]
        pub mod kitchen_sink {
            include!("temporal.omes.kitchen_sink.rs");
            impl HandlerInvocation {
                pub fn nonexistent() -> Self {
                    HandlerInvocation {
                        name: "nonexistent handler on purpose".to_string(),
                        args: vec![],
                    }
                }
            }
        }
    }

    #[allow(clippy::all)]
    pub mod api {
        pub mod common {
            pub mod v1 {
                include!("temporal.api.common.v1.rs");
            }
        }
        pub mod enums {
            pub mod v1 {
                include!("temporal.api.enums.v1.rs");
            }
        }
        pub mod failure {
            pub mod v1 {
                include!("temporal.api.failure.v1.rs");
            }
        }
    }
}
