//! Consume memory and CPU by writing pseudorandom bytes into a buffer until
//! run_for elapses, sleeping at the requested cadence. The CPU work runs on a
//! blocking thread, never on the Tokio executor.
use crate::{kitchen_sink::duration, proto::execute_activity_action::ResourcesActivity, wire};
use std::{
    hint::black_box,
    time::{Duration, Instant},
};
use temporalio_sdk::activities::{ActivityContext, ActivityError};
use tokio_util::sync::CancellationToken;

pub(crate) async fn run(
    ctx: ActivityContext,
    input: ResourcesActivity,
) -> Result<(), ActivityError> {
    let run_for = duration(input.run_for)?.unwrap_or_default();
    let deadline = Instant::now()
        .checked_add(run_for)
        .ok_or_else(|| wire::invalid("resources run_for is too large"))?;
    let bytes = usize::try_from(input.bytes_to_allocate)
        .map_err(|_| wire::invalid("resources bytes_to_allocate is out of range"))?;
    let yield_every = u64::from(input.cpu_yield_every_n_iterations);
    let yield_for = Duration::from_millis(input.cpu_yield_for_ms.into());
    let stop = CancellationToken::new();
    // Dropping the activity future (timeout, shutdown) must also stop the
    // blocking thread; aborting a spawn_blocking handle does not.
    let _stop_on_drop = stop.clone().drop_guard();
    let thread_stop = stop.clone();
    let mut task = tokio::task::spawn_blocking(move || -> Result<(), ActivityError> {
        let mut buffer = Vec::new();
        buffer
            .try_reserve_exact(bytes)
            .map_err(|_| wire::invalid("resources memory allocation failed"))?;
        // Touch every byte rather than relying on lazy zero pages.
        buffer.resize(bytes, 1u8);
        let mut random = 0x9e3779b97f4a7c15_u64;
        let mut i = 0_u64;
        while Instant::now() < deadline && !thread_stop.is_cancelled() {
            if yield_every != 0 && i.is_multiple_of(yield_every) {
                std::thread::sleep(
                    yield_for.min(deadline.saturating_duration_since(Instant::now())),
                );
            }
            random ^= random << 13;
            random ^= random >> 7;
            random ^= random << 17;
            if !buffer.is_empty() {
                let index = (i % buffer.len() as u64) as usize;
                buffer[index] = random as u8;
            }
            i += 1;
        }
        black_box(&buffer);
        Ok(())
    });
    tokio::select! {
        result = &mut task => result.map_err(|e| wire::invalid(format!("resources task failed: {e}")))?,
        _ = ctx.cancelled() => {
            stop.cancel();
            // Release the memory before reporting cancellation.
            let _ = task.await;
            Err(ActivityError::cancelled())
        }
    }
}
