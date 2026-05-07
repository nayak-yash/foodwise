"""Weekday/service forecasting and rolling-origin evaluation."""

from datetime import date
from statistics import mean, pstdev


class ForecastService:
    def forecast(self, history: list[dict], ingredient_id: str, target: date) -> dict:
        rows = [
            row
            for row in history
            if row["ingredient_id"] == ingredient_id and date.fromisoformat(row["date"]) < target
        ]
        services = [
            self._forecast_service(rows, service, target) for service in ("lunch", "dinner")
        ]
        return {
            "ingredient_id": ingredient_id,
            "date": target.isoformat(),
            "weekday": target.strftime("%A"),
            "method": "same-weekday service baseline",
            "services": services,
            "expected_kg": round(sum(item["expected_kg"] for item in services), 2),
            "planning_upper_kg": round(sum(item["planning_upper_kg"] for item in services), 2),
            "note": "Planning range, not a calibrated prediction interval. Holidays and seasonality are not yet modeled.",
        }

    def _forecast_service(self, rows: list[dict], service: str, target: date) -> dict:
        candidates = sorted(
            [row for row in rows if row["service"] == service],
            key=lambda row: row["date"],
        )
        matched = [
            float(row["consumption"])
            for row in candidates
            if date.fromisoformat(row["date"]).weekday() == target.weekday()
        ]
        fallback = len(matched) < 3
        values = (
            [float(row["consumption"]) for row in candidates[-28:]] if fallback else matched[-12:]
        )
        estimate = mean(values) if values else 0
        spread = max(pstdev(values), estimate * 0.1) if values else 0
        return {
            "service": service,
            "expected_kg": round(estimate, 2),
            "planning_upper_kg": round(estimate + 2 * spread, 2),
            "samples": len(values),
            "fallback": fallback,
        }

    def backtest(self, history: list[dict], ingredient_id: str) -> dict:
        days = sorted({row["date"] for row in history if row["ingredient_id"] == ingredient_id})
        seasonal_errors, naive_errors = [], []
        for day in days[-14:]:
            target = date.fromisoformat(day)
            past = [
                row
                for row in history
                if row["ingredient_id"] == ingredient_id and row["date"] < day
            ]
            if len(past) < 28:
                continue
            actual = sum(
                float(row["consumption"])
                for row in history
                if row["ingredient_id"] == ingredient_id and row["date"] == day
            )
            seasonal_errors.append(
                abs(actual - self.forecast(past, ingredient_id, target)["expected_kg"])
            )
            prior_days = sorted({row["date"] for row in past})[-7:]
            naive = sum(
                float(row["consumption"]) for row in past if row["date"] in prior_days
            ) / len(prior_days)
            naive_errors.append(abs(actual - naive))
        return {
            "evaluated_days": len(seasonal_errors),
            "weekday_mae_kg": round(mean(seasonal_errors), 3) if seasonal_errors else None,
            "seven_day_mean_mae_kg": round(mean(naive_errors), 3) if naive_errors else None,
            "dataset": "seeded synthetic history",
        }
