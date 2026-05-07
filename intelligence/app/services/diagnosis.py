"""Deterministic hypothesis assessment; numerical facts never depend on an LLM."""


class DiagnosisService:
    def diagnose(self, evidence: dict, prediction: dict) -> dict:
        purchase_order = evidence["purchase_order"]
        receipts = evidence["receipts"]
        inventory = evidence["inventory"][0]
        received = sum(float(receipt["quantity"]) for receipt in receipts)
        missing = round(float(purchase_order["quantity"]) - received, 3)
        synced = abs(received - float(purchase_order["received_quantity"])) < 0.001
        references = [f"delivery:{receipt['id']}" for receipt in receipts]
        hypotheses = [
            {
                "name": "Short delivery",
                "assessment": "supported" if missing > 0 else "contradicted",
                "evidence": [
                    f"Purchase order specifies {purchase_order['quantity']} kg; receiving records total {received:g} kg.",
                    f"External status is {purchase_order['external_status']}; canonical status is {purchase_order['status']}.",
                ],
                "sources": [f"purchase_order:{purchase_order['id']}", *references],
            },
            {
                "name": "Inventory synchronization failure",
                "assessment": "less_likely" if synced else "needs_investigation",
                "evidence": [
                    "Receipt total and recorded purchase-order received quantity agree."
                    if synced
                    else "Receipt and purchase-order totals disagree.",
                    "Physical count and upstream integration logs are not available.",
                ],
                "sources": references,
            },
            {
                "name": "Unexpected demand",
                "assessment": "possible_contributor",
                "evidence": [
                    f"Recorded 24-hour consumption: {inventory['consumption_24h']} kg. {prediction['weekday']} full-day forecast: {prediction['expected_kg']} kg.",
                    "Rolling 24-hour usage and full-day forecasts are different windows; this comparison does not establish a demand anomaly.",
                ],
                "sources": ["inventory_movements", "demand_history"],
            },
            {
                "name": "Unrecorded waste",
                "assessment": "unconfirmed",
                "evidence": [
                    f"Recorded waste in 24 hours: {inventory['waste_24h']} kg.",
                    "Missing waste records cannot rule out unrecorded waste.",
                ],
                "sources": ["inventory_movements"],
            },
        ]
        supported = missing > 0 and synced
        return {
            "root_cause": "Recorded delivery shortfall"
            if supported
            else "Receiving discrepancy requires review",
            "confidence": "high_for_recorded_shortfall" if supported else "insufficient",
            "summary": f"Receiving evidence accounts for {received:g} of {purchase_order['quantity']} kg. The {missing:g} kg shortfall is supported by records; kitchen impact is correlated, not proven causal.",
            "missing_kg": missing,
            "hypotheses": hypotheses,
            "limitations": [
                "No independent physical stock count or supplier-signed dispatch record.",
                "Kitchen measurements are branch-level, not dish-specific.",
            ],
            "forecast": prediction,
            "evidence_observed_at": evidence["observed_at"],
            "mode": "deterministic",
        }
