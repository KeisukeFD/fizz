package main

import (
	"bytes"
	"testing"

	"github.com/KeisukeFD/fizz/internal/acceptance/at"
	"github.com/KeisukeFD/fizz/internal/conformance/contract"
)

// TestContractMarket drives the market from the spec it serves (P10.6). It
// lives here because the market is package main. The handlers do not echo
// their input, so the contract is that the requests the document describes
// are not rejected with a 400.
func TestContractMarket(t *testing.T) {
	h := bINewMarket(t)
	w := at.Do(h, "GET", "/openapi.json")
	if w.Code != 200 {
		t.Fatalf("GET /openapi.json = %d", w.Code)
	}
	contract.Run(t, contract.Target{
		Handler: h,
		Spec:    w.Body.Bytes(),
		// Deleting a sample name that is not in the market is the logic
		// of the handler, not a rejection of the request.
		Tolerate: func(status int, body []byte) bool { return bytes.Contains(body, []byte("fruit not found")) },
	})
}
