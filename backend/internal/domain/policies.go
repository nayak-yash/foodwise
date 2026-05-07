// Package domain contains business policies independent of HTTP and storage.
package domain

import "math"

const BranchID = "11111111-1111-1111-1111-111111111111"
const ChickenID = "22222222-2222-2222-2222-222222222222"

func ValidQuantity(quantity float64) bool {
	return quantity > 0 && quantity <= 100000 &&
		!math.IsNaN(quantity) && !math.IsInf(quantity, 0) &&
		math.Abs(quantity*1000-math.Round(quantity*1000)) < 0.00001
}

func ReceiptStatus(ordered, received float64) string {
	if math.Abs(ordered-received) < 0.0005 {
		return "fulfilled"
	}
	if received > 0 {
		return "partial"
	}
	return "ordered"
}

func AllowedTransition(from, to string) bool {
	if from == to {
		return true
	}
	transitions := map[string]map[string]bool{
		"detected":          {"investigating": true, "escalated": true},
		"investigating":     {"awaiting_approval": true, "escalated": true},
		"awaiting_approval": {"escalated": true},
		"approved":          {"executing": true, "escalated": true},
		"executing":         {"verifying": true, "escalated": true},
		"verifying":         {"resolved": true, "investigating": true, "escalated": true},
	}
	return transitions[from][to]
}
