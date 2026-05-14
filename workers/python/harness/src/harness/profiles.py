from __future__ import annotations

from typing import Any

from temporalio.worker import WorkerTuner

WORKER_PROFILE_ENV_VAR = "OMES_WORKER_PROFILE"
RESOURCE_BASED_DEFAULT_PROFILE = "resource-based-default"

WorkerProfile = dict[str, Any]


_profiles: dict[str, WorkerProfile] = {}


def _register_profile(name: str, profile: WorkerProfile) -> None:
    _profiles[name] = profile


def lookup_profile(name: str) -> dict[str, Any]:
    try:
        return dict(_profiles[name])
    except KeyError as err:
        raise ValueError(f"Unknown worker profile {name!r}") from err


_register_profile(
    RESOURCE_BASED_DEFAULT_PROFILE,
    {
        "tuner": WorkerTuner.create_resource_based(
            target_memory_usage=0.8,
            target_cpu_usage=0.8,
        )
    },
)
