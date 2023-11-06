mod protos;

use clap::Parser;
use rand::SeedableRng;
use crate::protos::temporal::omes::kitchen_sink::TestInput;

/// A tool for generating client actions and inputs to the kitchen sink workflows in omes.
#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Use the specified seed as input, guaranteeing the same output as any invocation of the tool
    /// which used the same seed.
    #[arg(short, long)]
    explicit_seed: Option<u64>
}

fn main() {
    let args = Args::parse();

    if let Some(seed) = args.explicit_seed {
        rand::rngs::StdRng::seed_from_u64(seed)
    } else {
        rand::rngs::StdRng::from_entropy()
    };

    let generated_input = TestInput::default();
    dbg!(generated_input);
}
