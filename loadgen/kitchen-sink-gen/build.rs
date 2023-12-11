use prost_build::protoc_from_env;
use std::fs;
use std::io::{Read, Write};
use std::path::PathBuf;
use std::process::Command;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:rerun-if-changed=../../workers/proto");
    let ks_protos = ["../../workers/proto/kitchen_sink/kitchen_sink.proto"];
    let include_paths = [
        "../../workers/proto/api_upstream",
        "../../workers/proto/kitchen_sink",
    ];
    let mut cfg = prost_build::Config::default();
    cfg.type_attribute(
        "temporal.omes.kitchen_sink.Action.variant",
        "#[derive(::derive_more::From)]",
    );
    cfg.type_attribute(
        "temporal.omes.kitchen_sink.DoUpdate.variant",
        "#[derive(::derive_more::From)]",
    );
    cfg.type_attribute(
        "temporal.omes.kitchen_sink.DoActionsUpdate.variant",
        "#[derive(::derive_more::From)]",
    );
    cfg.type_attribute(
        "temporal.omes.kitchen_sink.DoActionsUpdate",
        "#[derive(::derive_more::From)]",
    );
    cfg.type_attribute(
        "temporal.omes.kitchen_sink.DoSignal.DoSignalActions",
        "#[derive(::derive_more::From)]",
    );
    cfg.compile_protos(&ks_protos, &include_paths)?;

    // Compile for python
    let protoc = protoc_from_env();
    for (lang, out_dir) in [
        ("python", "../../workers/python/protos"),
        ("java", "../../workers/java"),
        ("go", "../../loadgen/kitchensink"),
    ] {
        let mut cmd = Command::new(protoc.clone());

        for include in include_paths {
            cmd.arg("-I").arg(include);
        }
        for proto in ks_protos {
            cmd.arg(proto);
        }

        if lang == "python" {
            cmd.arg(format!("--python_out={out_dir}"));
            cmd.arg(format!("--pyi_out={out_dir}"));
        } else if lang == "java" {
            cmd.arg(format!("--java_out={out_dir}"));
        } else if lang == "go" {
            cmd.arg("--go_opt=paths=source_relative");
            cmd.arg(format!("--go_out={out_dir}"));
        }

        if !PathBuf::from(out_dir).exists() {
            fs::create_dir_all(out_dir)?;
        }
        let cmd_res = cmd.status()?;
        assert!(cmd_res.success(), "{lang} protoc failed");

        if lang == "python" {
            // Do some obnoxious shit to fix up the import paths. Luckily we can just re-use the
            // SDK's definitions of the upstream protos
            for fpath in [
                format!("{out_dir}/kitchen_sink_pb2.py"),
                format!("{out_dir}/kitchen_sink_pb2.pyi"),
            ] {
                let mut file = fs::File::open(&fpath)?;
                let mut content = String::new();
                file.read_to_string(&mut content)?;
                drop(file);

                let new_data = content.replace("from temporal.api", "from temporalio.api");

                let mut dst = fs::File::create(&fpath)?;
                dst.write_all(new_data.as_bytes())?;
            }
        }
    }

    Ok(())
}
