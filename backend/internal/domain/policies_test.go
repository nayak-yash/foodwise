package domain

import "testing"

var receiptStatus = ReceiptStatus
var validQuantity = ValidQuantity
var allowedTransition = AllowedTransition

func TestReceiptCannotTreatPartialAsFulfilled(t *testing.T) {
	for _, tc := range []struct {
		ordered, received float64
		want              string
	}{{40, 0, "ordered"}, {40, 25, "partial"}, {40, 40, "fulfilled"}, {1.125, 1.125, "fulfilled"}} {
		if got := receiptStatus(tc.ordered, tc.received); got != tc.want {
			t.Fatalf("received %v of %v: got %s want %s", tc.received, tc.ordered, got, tc.want)
		}
	}
}
func TestQuantityPrecision(t *testing.T) {
	for _, q := range []float64{-1, 0, 0.0001, 100001} {
		if validQuantity(q) {
			t.Errorf("accepted invalid quantity %v", q)
		}
	}
	for _, q := range []float64{0.001, 25, 40.125} {
		if !validQuantity(q) {
			t.Errorf("rejected valid quantity %v", q)
		}
	}
}
func TestIncidentCannotSkipRecovery(t *testing.T) {
	for _, from := range []string{"detected", "investigating", "awaiting_approval", "approved", "resolved", "escalated"} {
		if from != "resolved" && allowedTransition(from, "resolved") {
			t.Errorf("can skip verification from %s", from)
		}
	}
	if !allowedTransition("verifying", "resolved") || !allowedTransition("verifying", "investigating") {
		t.Fatal("verification must support resolution and reopening")
	}
	if allowedTransition("awaiting_approval", "approved") {
		t.Fatal("agent report must not approve actions")
	}
}
