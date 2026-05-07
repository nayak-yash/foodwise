import unittest
from datetime import date, datetime, timedelta, timezone

from app.services.diagnosis import DiagnosisService
from app.services.forecasting import ForecastService
from app.services.recovery import RecoveryVerifier

diagnose = DiagnosisService().diagnose
forecast = ForecastService().forecast
verify = RecoveryVerifier().verify


class EngineTests(unittest.TestCase):
    def test_weekend_is_not_compared_with_weekday(self):
        start = date(2026, 1, 1)
        history = [
            {
                "ingredient_id": "chicken",
                "date": (start + timedelta(days=i)).isoformat(),
                "service": service,
                "consumption": 20 if (start + timedelta(days=i)).weekday() >= 5 else 10,
            }
            for i in range(56)
            for service in ("lunch", "dinner")
        ]
        saturday = forecast(history, "chicken", date(2026, 2, 28))
        monday = forecast(history, "chicken", date(2026, 3, 2))
        self.assertEqual(saturday["expected_kg"], 40)
        self.assertEqual(monday["expected_kg"], 20)

    def evidence(self):
        return {
            "purchase_order": {
                "id": "po",
                "quantity": 40,
                "received_quantity": 25,
                "external_status": "completed",
                "status": "partial",
            },
            "receipts": [{"id": "receipt", "quantity": 25}],
            "inventory": [{"balance": 4, "waste_24h": 0, "consumption_24h": 26}],
            "observed_at": "2026-01-01",
            "actions": [],
            "kitchen": [],
        }

    def test_external_completed_does_not_override_receipt_evidence(self):
        result = diagnose(self.evidence(), {"weekday": "Saturday", "expected_kg": 40})
        self.assertEqual(result["missing_kg"], 15)
        self.assertEqual(result["root_cause"], "Recorded delivery shortfall")
        self.assertEqual(result["hypotheses"][3]["assessment"], "unconfirmed")

    def test_receipt_without_measured_recovery_cannot_resolve(self):
        evidence = self.evidence()
        evidence["actions"] = [{"status": "received", "received_at": "2026-01-01T10:00:00+00:00"}]
        self.assertFalse(verify(evidence)["recovered"])

    def test_recovery_requires_post_receipt_fresh_healthy_windows(self):
        evidence = self.evidence()
        now = datetime(2026, 1, 1, 10, 11, tzinfo=timezone.utc)
        evidence["actions"] = [{"status": "received", "received_at": "2026-01-01T10:00:00+00:00"}]
        evidence["kitchen"] = [
            {
                "recorded_at": f"2026-01-01T10:{minute:02d}:00+00:00",
                "prep_minutes": 14,
                "cancellation_rate": 0.02,
                "order_count": 10,
            }
            for minute in (0, 5, 10)
        ]
        self.assertTrue(verify(evidence, now=now)["recovered"])
        evidence["kitchen"][-1]["prep_minutes"] = 30
        self.assertFalse(verify(evidence, now=now)["recovered"])
        evidence["kitchen"][-1]["prep_minutes"] = 14
        self.assertFalse(verify(evidence, now=now + timedelta(hours=1))["recovered"])

    def test_backdated_kitchen_samples_do_not_count(self):
        evidence = self.evidence()
        evidence["actions"] = [{"status": "received", "received_at": "2026-01-01T11:00:00+00:00"}]
        evidence["kitchen"] = [
            {
                "recorded_at": "2026-01-01T10:00:00+00:00",
                "prep_minutes": 12,
                "cancellation_rate": 0,
                "order_count": 100,
            }
        ]
        self.assertEqual(verify(evidence)["sample_count"], 0)


if __name__ == "__main__":
    unittest.main()
