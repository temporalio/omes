mod protos;

use crate::protos::temporal::omes::kitchen_sink::{ActivityCancellationType, ContinueAsNewAction};
use crate::protos::temporal::{
    api::common::v1::{Memo, Payload, Payloads},
    omes::kitchen_sink::{
        action, awaitable_choice, client_action, do_actions_update, do_query, do_signal,
        do_signal::do_signal_actions, do_signal::DoSignalActions, do_update, execute_activity_action, with_start_client_action, Action, ActionSet,
        AwaitWorkflowState, AwaitableChoice, ClientAction, ClientActionSet, ClientSequence,
        DoQuery, DoSignal, DoUpdate, ExecuteActivityAction, ExecuteChildWorkflowAction,
        HandlerInvocation, RemoteActivityOptions, ReturnResultAction, SetPatchMarkerAction,
        TestInput, TimerAction, UpsertMemoAction, UpsertSearchAttributesAction, WithStartClientAction,
        WorkflowInput, WorkflowState,
        execute_activity_action::{ClientActivity, PayloadActivity},
    },
};
use anyhow::Error;
use arbitrary::{Arbitrary, Unstructured};
use clap::Parser;
use prost::Message;
use rand::{Rng, SeedableRng};
use std::{
    cell::RefCell, collections::HashMap, io::Write, ops::RangeInclusive, path::PathBuf,
    time::Duration,
};

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

    /// When specified, override the generator configuration with the JSON config in the specified
    /// file.
    #[arg(long)]
    generator_config_override: Option<PathBuf>,
}

#[derive(Debug, serde::Serialize, serde::Deserialize)]
struct GeneratorConfig {
    /// The maximum number of client action sets that will be generated.
    max_client_action_sets: usize,
    /// The maximum number of actions in a single client action set.
    max_client_actions_per_set: usize,
    /// The maximum number of actions in an action set (in the workflow context).
    max_actions_per_set: usize,
    /// The max timer duration
    max_timer: Duration,
    /// Max size in bytes that a payload will be
    max_payload_size: usize,
    /// The maximum time that a client action set will wait at the end of the set before moving on
    /// to the next set.
    max_client_action_set_wait: Duration,
    /// How likely various actions are to be generated
    action_chances: ActionChances,
    /// Maximum number of initial actions in a workflow input
    max_initial_actions: usize,
}

impl Default for GeneratorConfig {
    fn default() -> Self {
        Self {
            max_client_action_sets: 250,
            max_client_actions_per_set: 5,
            max_actions_per_set: 5,
            max_timer: Duration::from_secs(1),
            max_payload_size: 256,
            max_client_action_set_wait: Duration::from_secs(1),
            action_chances: Default::default(),
            max_initial_actions: 10,
        }
    }
}

#[derive(clap::Args, Debug)]
struct ExampleCmd {
    #[command(flatten)]
    proto_output: OutputConfig,
}

#[derive(clap::Args, Debug, Clone)]
struct OutputConfig {
    /// Output goes to stdout, this is the default.
    #[clap(long, default_value_t = true)]
    use_stdout: bool,
    /// Output goes to the provided file path as protobuf binary. Not exclusive with stdout. Both
    /// may be used if desired.
    #[clap(long)]
    output_path: Option<PathBuf>,
    /// Output will be in Rust debug format if set true. JSON is obnoxious to use with prost at
    /// the moment, and this option is really meant for human inspection.
    #[clap(long, default_value_t = false)]
    debug: bool,
}

/// The relative likelihood of each action type being generated as floats which must sum to exactly
/// 100.
#[derive(Clone, Debug, serde::Serialize, serde::Deserialize)]
struct ActionChances {
    timer: f32,
    activity: f32,
    child_workflow: f32,
    patch_marker: f32,
    set_workflow_state: f32,
    await_workflow_state: f32,
    upsert_memo: f32,
    upsert_search_attributes: f32,
    nested_action_set: f32,
}
impl Default for ActionChances {
    fn default() -> Self {
        Self {
            timer: 25.0,
            activity: 25.0,
            child_workflow: 25.0,
            nested_action_set: 12.5,
            patch_marker: 2.5,
            set_workflow_state: 2.5,
            await_workflow_state: 2.5,
            upsert_memo: 2.5,
            upsert_search_attributes: 2.5,
        }
    }
}
impl ActionChances {
    fn verify(&self) -> bool {
        let sum = self.timer
            + self.activity
            + self.child_workflow
            + self.patch_marker
            + self.set_workflow_state
            + self.await_workflow_state
            + self.upsert_memo
            + self.upsert_search_attributes
            + self.nested_action_set;
        sum == 100.0
    }
}
macro_rules! define_action_chances {
    ($($field:ident),+) => {
        impl ActionChances {
            pub fn get_field_order(&self) -> Vec<(&'static str, f32)> {
                vec![$((stringify!($field), self.$field),)+]
            }

            $(
                pub fn $field(&self, value: f32) -> bool {
                    let chances = self.get_field_order();
                    let mut lower_bound = 0.0;
                    for (_, chance) in chances.iter().take_while(|(f, _)| *f != stringify!($field)) {
                        lower_bound += chance;
                    }
                    let upper_bound = lower_bound + self.$field;
                    value >= lower_bound && value <= upper_bound
                }
            )+
        }
    };
}
define_action_chances!(
    timer,
    activity,
    child_workflow,
    nested_action_set,
    patch_marker,
    set_workflow_state,
    await_workflow_state,
    upsert_memo,
    upsert_search_attributes
);

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
    let client_sequence = ClientSequence {
        action_sets: vec![
            ClientActionSet {
                actions: vec![mk_client_signal_action([TimerAction {
                    milliseconds: 100,
                    awaitable_choice: None,
                }
                .into()])],
                concurrent: false,
                wait_at_end: Some(Duration::from_secs(1).try_into().unwrap()),
                wait_for_current_run_to_finish_at_end: false,
            },
            ClientActionSet {
                actions: vec![mk_client_signal_action([
                    TimerAction {
                        milliseconds: 100,
                        awaitable_choice: None,
                    }
                    .into(),
                    ExecuteActivityAction {
                        activity_type: Some(execute_activity_action::ActivityType::Noop(())),
                        start_to_close_timeout: Some(Duration::from_secs(1).try_into().unwrap()),
                        ..Default::default()
                    }
                    .into(),
                    ReturnResultAction {
                        return_this: Some(empty_payload()),
                    }
                    .into(),
                ])],
                ..Default::default()
            },
        ],
    };
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
    eprintln!("Using seed: {}", seed);
    let config = if let Some(path) = args.generator_config_override {
        let config_str = std::fs::read_to_string(path)?;
        serde_json::from_str(&config_str)?
    } else {
        GeneratorConfig::default()
    };
    if !config.action_chances.verify() {
        return Err(anyhow::anyhow!(
            "ActionChances must sum to exactly 100, got {:?}",
            config.action_chances
        ));
    }
    eprintln!("Using config: {:?}", serde_json::to_string(&config)?);
    let context = ArbContext {
        config,
        cur_workflow_state: Default::default(),
        action_set_nest_level: 0,
    };
    ARB_CONTEXT.set(context);

    let mut raw_dat = [0u8; 1024 * 200];
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

static WF_STATE_FIELD_VALUE: &str = "x";
static WF_TYPE_NAME: &str = "kitchenSink";

#[derive(Default)]
struct ArbContext {
    config: GeneratorConfig,
    cur_workflow_state: WorkflowState,
    action_set_nest_level: usize,
}

impl<'a> Arbitrary<'a> for TestInput {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // We always want a client sequence
        let mut client_sequence: ClientSequence = u.arbitrary()?;

        // Sometimes we want a with-start client action
        let with_start_action = if u.ratio(80, 100)? {
            None
        } else {
            Some(WithStartClientAction::arbitrary(u)?)
        };

        let mut ti = Self {
            // Input may or may not be present
            workflow_input: u.arbitrary()?,
            client_sequence: None,
            with_start_action: with_start_action,
        };

        // Finally, return at the end
        client_sequence.action_sets.push(ClientActionSet {
            actions: vec![mk_client_signal_action([ReturnResultAction {
                return_this: Some(empty_payload()),
            }
            .into()])],
            ..Default::default()
        });
        ti.client_sequence = Some(client_sequence);
        Ok(ti)
    }
}

impl<'a> Arbitrary<'a> for WorkflowInput {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let num_actions = 1..=ARB_CONTEXT.with_borrow(|c| c.config.max_initial_actions);
        let initial_actions = vec_of_size(u, num_actions)?;
        Ok(Self { 
            initial_actions, 
            expected_signal_count: 0,
            expected_signal_ids: vec![],
            received_signal_ids: vec![]
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
        let nest_level = ARB_CONTEXT.with_borrow_mut(|c| {
            c.action_set_nest_level += 1;
            c.action_set_nest_level
        });
        // Small chance of continuing as new
        let action_set = if nest_level == 1 && u.ratio(1, 100)? {
            let actions = vec![mk_client_signal_action([ContinueAsNewAction {
                workflow_type: WF_TYPE_NAME.to_string(),
                arguments: vec![to_proto_payload(
                    WorkflowInput {
                        initial_actions: vec![mk_action_set([action::Variant::SetWorkflowState(
                            ARB_CONTEXT.with_borrow(|c| c.cur_workflow_state.clone()),
                        )])],
                        expected_signal_count: 0,
                        expected_signal_ids: vec![],
                        received_signal_ids: vec![]
                    },
                    "temporal.omes.kitchen_sink.WorkflowInput",
                )],
                ..Default::default()
            }
            .into()])];
            Self {
                actions,
                wait_for_current_run_to_finish_at_end: true,
                ..Default::default()
            }
        } else {
            let num_actions = 1..=ARB_CONTEXT.with_borrow(|c| c.config.max_client_actions_per_set);
            Self {
                actions: vec_of_size(u, num_actions)?,
                concurrent: u.arbitrary()?,
                wait_at_end: u.arbitrary::<Option<ClientActionWait>>()?.map(Into::into),
                wait_for_current_run_to_finish_at_end: false,
            }
        };
        ARB_CONTEXT.with_borrow_mut(|c| c.action_set_nest_level -= 1);
        Ok(action_set)
    }
}

impl<'a> Arbitrary<'a> for ClientAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        // Too much nesting can lead to very long action sets, which are also very confusing
        // to understand. One level of nesting ought to be sufficient for coverage.
        let action_kind = if ARB_CONTEXT.with_borrow(|c| c.action_set_nest_level == 1) {
            u.int_in_range(0..=3)?
        } else {
            u.int_in_range(0..=2)?
        };
        let variant = match action_kind {
            0 => client_action::Variant::DoSignal(u.arbitrary()?),
            1 => client_action::Variant::DoQuery(u.arbitrary()?),
            2 => client_action::Variant::DoUpdate(u.arbitrary()?),
            3 => client_action::Variant::NestedActions(u.arbitrary()?),
            _ => unreachable!(),
        };
        Ok(Self {
            variant: Some(variant),
        })
    }
}

impl<'a> Arbitrary<'a> for WithStartClientAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let action_kind = u.int_in_range(0..=1)?;
        let variant = match action_kind {
            0 => with_start_client_action::Variant::DoSignal({
                let mut signal_action: DoSignal = u.arbitrary()?;
                signal_action.with_start = true;
                signal_action
            }),
            1 => with_start_client_action::Variant::DoUpdate({
                let mut update_action: DoUpdate = u.arbitrary()?;
                update_action.with_start = true;
                update_action
            }),
            _ => unreachable!(),
        };
        Ok(Self { variant: Some(variant) })
    }
}

impl<'a> Arbitrary<'a> for DoSignal {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let variant = if u.ratio(95, 100)? {
            // 95% of the time do actions
            // Half of that in the handler half in main
            if u.ratio(50, 100)? {
                do_signal::Variant::DoSignalActions(
                    DoSignalActions {
                        signal_id: u.arbitrary()?,
                        variant: Some(do_signal_actions::Variant::DoActions(u.arbitrary()?)),
                    }
                )
            } else {
                do_signal::Variant::DoSignalActions(
                    DoSignalActions {
                        signal_id: u.arbitrary()?,
                        variant: Some(do_signal_actions::Variant::DoActionsInMain(u.arbitrary()?)),
                    }
                )
            }
        } else {
            // Sometimes do a not found signal
            do_signal::Variant::Custom(HandlerInvocation::nonexistent())
        };
        Ok(Self {
            variant: Some(variant),
            with_start: u.arbitrary()?,
        })
    }
}

impl<'a> Arbitrary<'a> for DoQuery {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let mut failure_expected = false;
        let variant = if u.ratio(95, 100)? {
            // 95% of the time report state
            do_query::Variant::ReportState(u.arbitrary()?)
        } else {
            // Sometimes do a not found query
            failure_expected = true;
            do_query::Variant::Custom(HandlerInvocation::nonexistent())
        };
        Ok(Self {
            variant: Some(variant),
            failure_expected,
        })
    }
}

impl<'a> Arbitrary<'a> for DoUpdate {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let mut failure_expected = false;
        let variant = if u.ratio(95, 100)? {
            // 95% of the time do actions
            do_update::Variant::DoActions(
                Some(do_actions_update::Variant::DoActions(u.arbitrary()?)).into(),
            )
        } else if u.ratio(50, 100)? {
            // 5% of the time do a rejection
            failure_expected = true;
            do_update::Variant::DoActions(Some(do_actions_update::Variant::RejectMe(())).into())
        } else {
            // Or not found
            failure_expected = true;
            do_update::Variant::Custom(HandlerInvocation::nonexistent())
        };
        Ok(Self {
            variant: Some(variant),
            failure_expected,
            with_start: u.arbitrary()?,
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
        let action_kind = u.int_in_range(0..=1_000)? as f32 / 10.0;
        let chances = ARB_CONTEXT.with_borrow(|c| c.config.action_chances.clone());
        let variant = if chances.timer(action_kind) {
            action::Variant::Timer(u.arbitrary()?)
        } else if chances.activity(action_kind) {
            action::Variant::ExecActivity(u.arbitrary()?)
        } else if chances.child_workflow(action_kind) {
            action::Variant::ExecChildWorkflow(u.arbitrary()?)
        } else if chances.patch_marker(action_kind) {
            action::Variant::SetPatchMarker(u.arbitrary()?)
        } else if chances.set_workflow_state(action_kind) {
            action::Variant::SetWorkflowState({
                let chosen_int = u.int_in_range(1..=100)?;
                ARB_CONTEXT.with_borrow_mut(|c| {
                    c.cur_workflow_state
                        .kvs
                        .insert(chosen_int.to_string(), WF_STATE_FIELD_VALUE.to_string());
                    c.cur_workflow_state.clone()
                })
            })
        } else if chances.await_workflow_state(action_kind) {
            let key = ARB_CONTEXT.with_borrow(|c| {
                if c.cur_workflow_state.kvs.is_empty() {
                    None
                } else {
                    let mut keys = c.cur_workflow_state.kvs.keys().collect::<Vec<_>>();
                    keys.sort();
                    Some(u.choose(&keys).map(|s| s.to_string()))
                }
            });
            if let Some(key) = key {
                action::Variant::AwaitWorkflowState(AwaitWorkflowState {
                    key: key?,
                    value: WF_STATE_FIELD_VALUE.to_string(),
                })
            } else {
                // Pick a different action if we've never set anything in state
                let action: Action = u.arbitrary()?;
                action.variant.unwrap()
            }
        } else if chances.upsert_memo(action_kind) {
            action::Variant::UpsertMemo(u.arbitrary()?)
        } else if chances.upsert_search_attributes(action_kind) {
            action::Variant::UpsertSearchAttributes(u.arbitrary()?)
        } else if chances.nested_action_set(action_kind) {
            action::Variant::NestedActionSet(u.arbitrary()?)
        } else {
            unreachable!()
        };
        Ok(Self {
            variant: Some(variant),
        })
    }
}

impl<'a> Arbitrary<'a> for TimerAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(Self {
            milliseconds: u.int_in_range(
                0..=ARB_CONTEXT.with_borrow(|c| c.config.max_timer.as_millis() as u64),
            )?,
            awaitable_choice: Some(u.arbitrary()?),
        })
    }
}

impl<'a> Arbitrary<'a> for ExecuteActivityAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let locality = if u.ratio(50, 100)? {
            execute_activity_action::Locality::Remote(u.arbitrary()?)
        } else {
            execute_activity_action::Locality::IsLocal(())
        };
        
        let activity_type_choice = u.int_in_range(1..=100)?;
        let activity_type = match activity_type_choice {
            1..=85 => {
                let delay = u.int_in_range(0..=1_000)?;
                execute_activity_action::ActivityType::Delay(
                    Duration::from_millis(delay)
                        .try_into()
                        .expect("proto duration works"),
                )
            }
            86..=90 => {
                execute_activity_action::ActivityType::Payload(u.arbitrary()?)
            }
            91..=100 => {
                execute_activity_action::ActivityType::Client(u.arbitrary()?)
            }
            _ => unreachable!(),
        };
        
        Ok(Self {
            activity_type: Some(activity_type),
            start_to_close_timeout: Some(Duration::from_secs(5).try_into().unwrap()),
            locality: Some(locality),
            awaitable_choice: Some(u.arbitrary()?),
            ..Default::default()
        })
    }
}

impl<'a> Arbitrary<'a> for ExecuteChildWorkflowAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let input = WorkflowInput {
            initial_actions: vec![ActionSet {
                actions: vec![
                    Action {
                        variant: Some(action::Variant::Timer(TimerAction {
                            milliseconds: u.int_in_range(0..=1_000)?,
                            awaitable_choice: None,
                        })),
                    },
                    Action {
                        variant: Some(action::Variant::ReturnResult(ReturnResultAction {
                            return_this: Some(empty_payload()),
                        })),
                    },
                ],
                concurrent: false,
            }],
            expected_signal_count: 0,
            expected_signal_ids: vec![],
            received_signal_ids: vec![]
        };
        let input = to_proto_payload(input, "temporal.omes.kitchen_sink.WorkflowInput");
        Ok(Self {
            // Use KS as own child, with an input to just return right away
            workflow_type: WF_TYPE_NAME.to_string(),
            input: vec![input],
            awaitable_choice: Some(u.arbitrary()?),
            ..Default::default()
        })
    }
}

impl<'a> Arbitrary<'a> for SetPatchMarkerAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let patch_id = u.int_in_range(1..=10)?;
        Ok(Self {
            patch_id: patch_id.to_string(),
            // Patches should be consistently deprecated or not for the same ID
            deprecated: patch_id % 2 == 0,
            inner_action: Some(u.arbitrary()?),
        })
    }
}

impl<'a> Arbitrary<'a> for UpsertMemoAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(Self {
            upserted_memo: Some(Memo {
                fields: {
                    let mut hm = HashMap::new();
                    let chosen_int = u.int_in_range(1..=100)?;
                    hm.insert(
                        chosen_int.to_string(),
                        Payload {
                            metadata: Default::default(),
                            data: vec![chosen_int],
                        },
                    );
                    hm
                },
            }),
        })
    }
}

static SEARCH_ATTR_KEYS: [&str; 2] = ["KS_Keyword", "KS_Int"];

impl<'a> Arbitrary<'a> for UpsertSearchAttributesAction {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let chosen_sa = *u.choose(&SEARCH_ATTR_KEYS)?;
        Ok(Self {
            search_attributes: {
                let mut hm = HashMap::new();
                let data = if chosen_sa == "KS_Keyword" {
                    serde_json::to_vec(&u.int_in_range(1..=255)?.to_string())
                } else {
                    serde_json::to_vec(&u.int_in_range(1..=255)?)
                }
                .expect("serializes");
                hm.insert(
                    chosen_sa.to_string(),
                    Payload {
                        metadata: {
                            let mut m = HashMap::new();
                            m.insert("encoding".to_string(), "json/plain".into());
                            m
                        },
                        data,
                    },
                );
                hm
            },
        })
    }
}

impl<'a> Arbitrary<'a> for ClientActivity {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let client_sequence = ClientSequence {
            action_sets: vec![ClientActionSet {
                actions: vec![u.arbitrary()?],
                concurrent: false,
                wait_at_end: None,
                wait_for_current_run_to_finish_at_end: false,
            }],
        };
        
        Ok(Self {
            client_sequence: Some(client_sequence),
        })
    }
}


impl<'a> Arbitrary<'a> for PayloadActivity {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let max_payload_size = ARB_CONTEXT.with_borrow(|c| c.config.max_payload_size) as i32;
        
        Ok(Self {
            bytes_to_receive: u.int_in_range(0..=max_payload_size)?,
            bytes_to_return: u.int_in_range(0..=max_payload_size)?,
        })
    }
}


impl<'a> Arbitrary<'a> for RemoteActivityOptions {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(RemoteActivityOptions {
            cancellation_type: u.arbitrary::<ActivityCancellationType>()?.into(),
            do_not_eagerly_execute: false,
            versioning_intent: 0,
        })
    }
}

impl<'a> Arbitrary<'a> for ActivityCancellationType {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let choices = [
            ActivityCancellationType::Abandon,
            ActivityCancellationType::TryCancel,
            ActivityCancellationType::WaitCancellationCompleted,
        ];
        Ok(*u.choose(&choices)?)
    }
}

impl<'a> Arbitrary<'a> for Payloads {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
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

struct ClientActionWait {
    duration: Duration,
}
impl<'a> Arbitrary<'a> for ClientActionWait {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let duration_ms = u.int_in_range(
            0..=ARB_CONTEXT.with_borrow(|c| c.config.max_client_action_set_wait.as_millis() as u64),
        )?;
        Ok(Self {
            duration: Duration::from_millis(duration_ms),
        })
    }
}
impl From<ClientActionWait> for prost_types::Duration {
    fn from(v: ClientActionWait) -> Self {
        v.duration.try_into().unwrap()
    }
}

impl<'a> Arbitrary<'a> for AwaitableChoice {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let choices = [
            awaitable_choice::Condition::WaitFinish(()),
            awaitable_choice::Condition::Abandon(()),
            awaitable_choice::Condition::CancelBeforeStarted(()),
            awaitable_choice::Condition::CancelAfterStarted(()),
            awaitable_choice::Condition::CancelAfterCompleted(()),
        ];
        Ok(AwaitableChoice {
            condition: Some(u.choose(&choices)?.clone()),
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
    if output_kind.debug {
        let as_str = format!("{:#?}", generated_input);
        buf.write_all(as_str.as_bytes())?;
    } else {
        generated_input.encode(&mut buf)?;
    }
    if let Some(path) = output_kind.output_path {
        let mut file = std::fs::File::create(path)?;
        file.write_all(&buf)?;
    }
    if output_kind.use_stdout {
        std::io::stdout().write_all(&buf)?;
    }
    Ok(())
}

fn mk_client_signal_action(actions: impl IntoIterator<Item = action::Variant>) -> ClientAction {
    ClientAction {
        variant: Some(client_action::Variant::DoSignal(DoSignal {
            variant: Some(do_signal::Variant::DoSignalActions(
                DoSignalActions {
                    signal_id: 0, // Default signal_id for client actions
                    variant: Some(do_signal_actions::Variant::DoActionsInMain(mk_action_set(
                        actions,
                    ))),
                }
            )),
            with_start: false,
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
        concurrent: false,
    }
}

fn to_proto_payload(msg: impl Message, type_name: &str) -> Payload {
    Payload {
        metadata: {
            let mut m = HashMap::new();
            m.insert("encoding".to_string(), "binary/protobuf".into());
            m.insert("messageType".to_string(), type_name.into());
            m
        },
        data: msg.encode_to_vec(),
    }
}

fn empty_payload() -> Payload {
    Payload {
        metadata: {
            let mut m = HashMap::new();
            m.insert("encoding".to_string(), "binary/null".into());
            m
        },
        data: vec![], // Empty
    }
}
