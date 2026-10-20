use std::{env, path::PathBuf};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:rerun-if-env-changed=OMES_PROTO_DIR");
    println!("cargo:rerun-if-env-changed=OMES_API_PROTO_DIR");
    let manifest = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    let proto = env::var_os("OMES_PROTO_DIR")
        .map(PathBuf::from)
        .or_else(|| {
            manifest
                .ancestors()
                .map(|dir| dir.join("proto"))
                .find(|dir| dir.join("kitchen_sink/kitchen_sink.proto").is_file())
        })
        .ok_or("shared Omes schemas not found; set OMES_PROTO_DIR")?;
    let api = env::var_os("OMES_API_PROTO_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|| proto.join("api_upstream"));
    let kitchen_sink = proto.join("kitchen_sink");
    let harness = proto.join("harness/api");
    for directory in [&kitchen_sink, &harness, &api] {
        println!("cargo:rerun-if-changed={}", directory.display());
    }
    let output = PathBuf::from(env::var_os("OUT_DIR").expect("Cargo OUT_DIR"));
    prost_build::Config::new()
        .file_descriptor_set_path(output.join("kitchen_sink.bin"))
        .enable_type_names()
        .extern_path(
            ".temporal.api",
            "::temporalio_common::protos::temporal::api",
        )
        .btree_map(["."])
        .compile_protos(
            &[kitchen_sink.join("kitchen_sink.proto")],
            &[kitchen_sink, api],
        )?;
    tonic_prost_build::configure()
        .build_client(false)
        .compile_protos(&[harness.join("api.proto")], &[harness])?;
    Ok(())
}
