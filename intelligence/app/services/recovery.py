from datetime import datetime, timezone


def parse_time(value: str) -> datetime:
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


class RecoveryVerifier:
    def __init__(self, window_seconds: float = 600):
        self.window_seconds = window_seconds

    def verify(self, evidence: dict, now: datetime | None = None) -> dict:
        now = now or datetime.now(timezone.utc)
        received = [
            action
            for action in evidence["actions"]
            if action["status"] == "received" and action.get("received_at")
        ]
        balance = float(evidence["inventory"][0]["balance"])
        samples = []
        if received:
            since = max(parse_time(action["received_at"]) for action in received)
            samples = sorted(
                [
                    sample
                    for sample in evidence["kitchen"]
                    if since <= parse_time(sample["recorded_at"]) <= now
                ],
                key=lambda sample: sample["recorded_at"],
            )[-3:]
        span = (
            (
                parse_time(samples[-1]["recorded_at"]) - parse_time(samples[0]["recorded_at"])
            ).total_seconds()
            if len(samples) >= 2
            else 0
        )
        checks = {
            "replacement_received": bool(received),
            "positive_stock": balance > 0,
            "enough_observations": len(samples) >= 3
            and sum(sample["order_count"] for sample in samples) >= 30,
            "observation_window": span >= self.window_seconds,
            "healthy_preparation": bool(samples)
            and all(sample["prep_minutes"] <= 18 for sample in samples),
            "healthy_cancellations": bool(samples)
            and all(sample["cancellation_rate"] <= 0.05 for sample in samples),
            "fresh_measurements": bool(samples)
            and (now - parse_time(samples[-1]["recorded_at"])).total_seconds() < 900,
        }
        return {
            "recovered": all(checks.values()),
            "checks": checks,
            "balance_kg": balance,
            "sample_count": len(samples),
            "observation_seconds": round(span, 1),
            "required_window_seconds": self.window_seconds,
            "checked_at": now.isoformat(),
        }
