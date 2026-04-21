package app

import "testing"

func TestAddress(t *testing.T) {
	got := Address("127.0.0.1", "18081")
	if got != "127.0.0.1:18081" {
		t.Fatalf("Address() = %q", got)
	}
}