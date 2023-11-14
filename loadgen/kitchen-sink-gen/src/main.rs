mod protos;

use crate::protos::temporal::{
    api::common::v1::{Payload, Payloads},
    omes::kitchen_sink::{
        action, client_action, do_query, do_signal, do_update, execute_activity_action, Action,
        ActionSet, ClientAction, ClientActionSet, ClientSequence, DoQuery, DoSignal, DoUpdate,
        ExecuteActivityAction, HandlerInvocation, RemoteActivityOptions, ReturnResultAction,
        TestInput, TimerAction, WorkflowInput,
    },
};
use anyhow::Error;
use arbitrary::{Arbitrary, Unstructured};
use clap::Parser;
use prost::Message;
use rand::{Rng, SeedableRng};
use std::{cell::RefCell, io::Write, ops::RangeInclusive, path::PathBuf, time::Duration};

/// A tool for generating client actions and inputs to the kitchen sink workflows in omes.
#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    #[command(subcommand)]
    command: Command,
}

#[derive(clap::Subcommand, Debug)]
enum Command {
    /// Generate test input.
    Generate(GenerateCmd),
    /// Generate the standard example test input, useful for testing kitchen sink implementations
    /// with a reasonably small input.
    Example(ExampleCmd),
}

#[derive(clap::Args, Debug)]
struct GenerateCmd {
    /// Use the specified seed as input, guaranteeing the same output as any invocation of the tool
    /// which used the same seed.
    #[arg(short, long)]
    explicit_seed: Option<u64>,

    #[command(flatten)]
    proto_output: OutputConfig,

    #[command(flatten)]
    generator_config: GeneratorConfig,
}

// TODO: Make this restorable from serialized form
#[derive(clap::Args, Debug, Default)]
struct GeneratorConfig {
    /// The maximum number of client action sets that will be generated.
    #[arg(long, default_value_t = 1000)]
    max_client_action_sets: usize,

    /// The maximum number of actions in a single client action set.
    #[arg(long, default_value_t = 5)]
    max_client_actions_per_set: usize,

    /// The maximum number of actions in an action set (in the workflow context).
    #[arg(long, default_value_t = 5)]
    max_actions_per_set: usize,

    /// The max timer duration in milliseconds
    #[arg(long, default_value_t = 1000)]
    max_timer_ms: u64,

    /// Max size in bytes that a payload will be
    #[arg(long, default_value_t = 256)]
    max_payload_size: usize,
}

#[derive(clap::Args, Debug)]
struct ExampleCmd {
    #[command(flatten)]
    proto_output: OutputConfig,
}

#[derive(clap::Args, Debug, Clone)]
#[clap(group(
    clap::ArgGroup::new("output").args(&["use_stdout", "output_path"]),
))]
struct OutputConfig {
    /// Output goes to stdout as protobuf binary, this is the default.
    #[clap(long, default_value_t = true)]
    use_stdout: bool,
    /// Output goes to the provided file path as protobuf binary.
    #[clap(long)]
    output_path: Option<PathBuf>,
}

fn main() -> Result<(), Error> {
    let args = Args::parse();
    match args.command {
        Command::Example(ex) => {
            example(ex)?;
        }
        Command::Generate(args) => {
            generate(args)?;
        }
    }

    Ok(())
}

fn example(args: ExampleCmd) -> Result<(), Error> {
    let mut example_input = TestInput::default();
    let mut client_sequence = ClientSequence::default();
    client_sequence.action_sets = vec![ClientActionSet {
        actions: vec![
            mk_client_signal_action([TimerAction {
                milliseconds: 100,
                awaitable_choice: None,
            }
            .into()]),
            mk_client_signal_action([
                TimerAction {
                    milliseconds: 100,
                    awaitable_choice: None,
                }
                .into(),
                ExecuteActivityAction {
                    activity_type: "noop".to_string(),
                    start_to_close_timeout: Some(Duration::from_secs(1).try_into().unwrap()),
                    ..Default::default()
                }
                .into(),
                ReturnResultAction {
                    return_this: Some(Payload::default()),
                }
                .into(),
            ]),
        ],
        concurrent: false,
    }];
    example_input.client_sequence = Some(client_sequence);
    output_proto(example_input, args.proto_output)?;
    Ok(())
}

fn generate(args: GenerateCmd) -> Result<(), Error> {
    let (mut rng, seed) = if let Some(seed) = args.explicit_seed {
        (rand::rngs::StdRng::seed_from_u64(seed), seed)
    } else {
        let mut seed_maker = rand::rngs::StdRng::from_entropy();
        let seed = seed_maker.gen();
        (rand::rngs::StdRng::seed_from_u64(seed), seed)
    };
    println!("Using seed: {}", seed);
    println!("Using config: {:?}", &args.generator_config);
    let context = ArbContext {
        config: args.generator_config,
    };
    ARB_CONTEXT.set(context);

    let mut raw_dat = [0u8; 1024 * 100];
    rng.fill(&mut raw_dat[..]);
    let mut unstructured = Unstructured::new(&raw_dat);
    let generated_input: TestInput = unstructured.arbitrary()?;
    output_proto(generated_input, args.proto_output)?;
    Ok(())
}

// This is slightly hacky but better than needing to re-implement arbitrary for every stdlib
// container type under the sun so we can attach a context. We know the generator runs in one
// thread.
thread_local! {
    static ARB_CONTEXT: RefCell<ArbContext> = RefCell::new(ArbContext::default());
}

#[derive(Default)]
struct ArbContext {
    config: GeneratorConfig,
}

impl<'a> Arbitrary<'a> for TestInput {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(Self {
            // Input may or may not be present
            workflow_input: u.arbitrary()?,
            // We always want a client sequence
            client_sequence: Some(u.arbitrary()?),
        })
    }
}

impl<'a> Arbitrary<'a> for WorkflowInput {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // TODO impl
        Ok(Self {
            initial_actions: vec![],
        })
    }
}

impl<'a> Arbitrary<'a> for ClientSequence {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let num_action_sets = 1..=ARB_CONTEXT.with_borrow(|c| c.config.max_client_action_sets);
        let action_sets = vec_of_size(u, num_action_sets)?;
        Ok(Self { action_sets })
    }
}

impl<'a> Arbitrary<'a> for ClientActionSet {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let num_actions = 1..=ARB_CONTEXT.with_borrow(|c| c.config.max_client_actions_per_set);
        Ok(Self {
            actions: vec_of_size(u, num_actions)?,
            concurrent: u.arbitrary()?,
        })
    }
}

impl<'a> Arbitrary<'a> for ClientAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // TODO: Adjustable ratio of choice?
        let action_kind = u.int_in_range(0..=2)?;
        let variant = match action_kind {
            0 => client_action::Variant::DoSignal(u.arbitrary()?),
            1 => client_action::Variant::DoQuery(u.arbitrary()?),
            2 => client_action::Variant::DoUpdate(u.arbitrary()?),
            // TODO: Nested, if/when desired
            _ => unreachable!(),
        };
        Ok(Self {
            variant: Some(variant),
        })
    }
}

impl<'a> Arbitrary<'a> for DoSignal {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // TODO: Configurable?
        let variant = if u.ratio(95, 100)? {
            // 95% of the time do actions
            do_signal::Variant::DoActions(u.arbitrary()?)
        } else {
            // Sometimes do a not found signal
            do_signal::Variant::Custom(HandlerInvocation::nonexistent())
        };
        Ok(Self {
            variant: Some(variant),
        })
    }
}

impl<'a> Arbitrary<'a> for DoQuery {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // TODO: Configurable?
        let variant = if u.ratio(95, 100)? {
            // 95% of the time report state
            do_query::Variant::ReportState(u.arbitrary()?)
        } else {
            // Sometimes do a not found query
            do_query::Variant::Custom(HandlerInvocation::nonexistent())
        };
        Ok(Self {
            variant: Some(variant),
        })
    }
}

impl<'a> Arbitrary<'a> for DoUpdate {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // TODO: Configurable?
        let variant = if u.ratio(95, 100)? {
            // 95% of the time do actions
            do_update::Variant::DoActions(u.arbitrary()?)
        } else if u.ratio(50, 100)? {
            // 5% of the time do a rejection
            do_update::Variant::RejectMe(())
        } else {
            // Or not found
            do_update::Variant::Custom(HandlerInvocation::nonexistent())
        };
        Ok(Self {
            variant: Some(variant),
        })
    }
}

impl<'a> Arbitrary<'a> for ActionSet {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let num_actions = 1..=ARB_CONTEXT.with_borrow(|c| c.config.max_actions_per_set);
        Ok(Self {
            actions: vec_of_size(u, num_actions)?,
            concurrent: u.arbitrary()?,
        })
    }
}

impl<'a> Arbitrary<'a> for Action {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // TODO: Adjustable ratio of choice?
        // TODO: The rest of the kinds of actions
        let action_kind = u.int_in_range(0..=1)?;
        let variant = match action_kind {
            0 => action::Variant::Timer(u.arbitrary()?),
            1 => action::Variant::ExecActivity(u.arbitrary()?),
            _ => unreachable!(),
        };
        Ok(Self {
            variant: Some(variant),
        })
    }
}

impl<'a> Arbitrary<'a> for TimerAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(Self {
            milliseconds: u.int_in_range(0..=ARB_CONTEXT.with_borrow(|c| c.config.max_timer_ms))?,
            // TODO: implement
            awaitable_choice: None,
        })
    }
}

impl<'a> Arbitrary<'a> for ExecuteActivityAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // TODO: configurable ratio?
        let locality = if u.ratio(50, 100)? {
            execute_activity_action::Locality::Remote(u.arbitrary()?)
        } else {
            execute_activity_action::Locality::IsLocal(())
        };
        Ok(Self {
            activity_type: "echo".to_string(),
            task_queue: "".to_string(),
            headers: Default::default(),
            arguments: vec![],
            schedule_to_close_timeout: None,
            schedule_to_start_timeout: None,
            start_to_close_timeout: Some(Duration::from_secs(5).try_into().unwrap()),
            heartbeat_timeout: None,
            retry_policy: None,
            awaitable_choice: None,
            locality: Some(locality),
        })
    }
}

impl<'a> Arbitrary<'a> for RemoteActivityOptions {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // TODO: impl
        Ok(Self::default())
    }
}

impl<'a> Arbitrary<'a> for Payloads {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // TODO: configurable ratio?
        let payloads = if u.ratio(80, 100)? {
            vec![u.arbitrary()?]
        } else if u.ratio(50, 100)? {
            vec![u.arbitrary()?, u.arbitrary()?]
        } else {
            vec![]
        };
        Ok(Self { payloads })
    }
}

impl<'a> Arbitrary<'a> for Payload {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let num_bytes =
            u.int_in_range(0..=ARB_CONTEXT.with_borrow(|c| c.config.max_payload_size))?;
        Ok(Self {
            metadata: Default::default(),
            data: u.bytes(num_bytes)?.to_vec(),
        })
    }
}

fn vec_of_size<'a, T: Arbitrary<'a>>(
    u: &mut Unstructured<'a>,
    size_range: RangeInclusive<usize>,
) -> arbitrary::Result<Vec<T>> {
    let size = u.int_in_range(size_range)?;
    let mut vec = Vec::with_capacity(size);
    for _ in 0..size {
        vec.push(u.arbitrary()?)
    }
    Ok(vec)
}

fn output_proto(generated_input: TestInput, output_kind: OutputConfig) -> Result<(), Error> {
    let mut buf = Vec::with_capacity(1024 * 10);
    generated_input.encode(&mut buf)?;
    if output_kind.use_stdout {
        std::io::stdout().write_all(&buf)?;
    } else {
        let path = output_kind
            .output_path
            .expect("Output path must have been set");
        let mut file = std::fs::File::create(path)?;
        file.write_all(&buf)?;
    }
    Ok(())
}

fn mk_client_signal_action(actions: impl IntoIterator<Item = action::Variant>) -> ClientAction {
    ClientAction {
        variant: Some(client_action::Variant::DoSignal(DoSignal {
            variant: Some(do_signal::Variant::DoActions(mk_action_set(actions))),
        })),
    }
}

fn mk_action_set(actions: impl IntoIterator<Item = action::Variant>) -> ActionSet {
    ActionSet {
        actions: actions
            .into_iter()
            .map(|variant| Action {
                variant: Some(variant),
            })
            .collect(),
        concurrent: true,
    }
}
