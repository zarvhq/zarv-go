package ingestion_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/zarvhq/zarv-go/pkg/ingestion"
)

func Example() {
	// A stand-in for the gateway; in a service, the URL and the key come from
	// ZARV_INGESTION_URL and ZARV_INGESTION_KEY.
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"accepted":1,"rejected":null}`)
	}))
	defer gateway.Close()

	client, err := ingestion.New(ingestion.Config{URL: gateway.URL, Key: "example-key"})
	if err != nil {
		panic(err)
	}

	err = client.Send(context.Background(), ingestion.Event{
		TableName: "billing_plan",
		Operation: "UPDATE",
		Data:      map[string]any{"id": "p1", "name": "Pro"},
	})
	switch {
	case errors.Is(err, ingestion.ErrRefused):
		fmt.Println("refused, do not resend:", err)
	case errors.Is(err, ingestion.ErrUnavailable):
		fmt.Println("not taken, safe to resend later:", err)
	case err != nil:
		fmt.Println("error:", err)
	default:
		fmt.Println("accepted")
	}
	// Output: accepted
}
