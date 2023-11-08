use prost_build::protoc_from_env;
use std::path::PathBuf;
use std::process::Command;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:rerun-if-changed=../../workers/proto");
    let ks_proto = ["../../workers/proto/kitchen_sink/kitchen_sink.proto"];
    let include_paths = [
        "../../workers/proto/api_upstream",
        "../../workers/proto/kitchen_sink",
    ];
    let mut cfg = prost_build::Config::default();
    cfg.type_attribute(
        "temporal.omes.kitchen_sink.Action.variant",
        "#[derive(::derive_more::From)]",
    );
    cfg.compile_protos(&ks_proto, &include_paths)?;

    // Compile for python
    let protoc = protoc_from_env();
    let mut cmd = Command::new(protoc.clone());
    cmd.arg("--include_imports").arg("--include_source_info");

    for include in include_paths {
        cmd.arg("-I").arg(include);
    }
    for proto in ks_proto {
        cmd.arg(proto);
    }
    cmd.arg("--python_out=../../workers/python/protos");
    cmd.arg("--pyi_out=../../workers/python/protos");
    if !PathBuf::from("../../workers/python/protos").exists() {
        std::fs::create_dir_all("../../workers/python/protos")?;
    }
    let cmd_res = cmd.status()?;
    assert!(cmd_res.success(), "python protoc failed");

    Ok(())
}
