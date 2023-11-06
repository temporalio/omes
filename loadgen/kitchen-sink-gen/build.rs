fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:rerun-if-changed=../../workers/proto");
    prost_build::
        compile_protos(
            &[
                "../../workers/proto/kitchen_sink/kitchen_sink.proto",
            ],
            &[
                "../../workers/proto/api_upstream",
                "../../workers/proto/kitchen_sink",
            ],
        )?;

    Ok(())
}
