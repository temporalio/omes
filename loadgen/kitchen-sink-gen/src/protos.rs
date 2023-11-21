pub mod temporal {
    pub mod omes {
        pub mod kitchen_sink {
            include!(concat!(env!("OUT_DIR"), "/temporal.omes.kitchen_sink.rs"));
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

    pub mod api {
        pub mod common {
            pub mod v1 {
                include!(concat!(env!("OUT_DIR"), "/temporal.api.common.v1.rs"));
            }
        }
        pub mod enums {
            pub mod v1 {
                include!(concat!(env!("OUT_DIR"), "/temporal.api.enums.v1.rs"));
            }
        }
        pub mod failure {
            pub mod v1 {
                include!(concat!(env!("OUT_DIR"), "/temporal.api.failure.v1.rs"));
            }
        }
    }
}
