pub mod temporal {
    pub mod omes {
        pub mod kitchen_sink {
            include!(concat!(env!("OUT_DIR"), "/temporal.omes.kitchen_sink.rs"));
            pub mod activity {
                include!(concat!(env!("OUT_DIR"), "/temporal.omes.kitchen_sink.activity.rs"));
            }
            pub mod child_workflow {
                include!(concat!(env!("OUT_DIR"), "/temporal.omes.kitchen_sink.child_workflow.rs"));
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
