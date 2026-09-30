package ingestion

import (
	"errors"
	"reflect"
	"testing"
)

func TestTheEnvelopeIsTheGatewaysShape(t *testing.T) {
	ev := Event{
		Domain: "billing", Service: "plan", Operation: "UPDATE", Description: "plan renamed",
		Data:     map[string]any{"id": "p1", "name": "Pro"},
		Metadata: map[string]any{"source": "zarv-billing"},
	}
	got, err := ev.envelope()
	if err != nil {
		t.Fatalf("envelope: %v", err)
	}
	want := map[string]any{
		"table_name":  "billing_plan",
		"operation":   "UPDATE",
		"description": "plan renamed",
		"data": map[string]any{
			"id": "p1", "name": "Pro",
			EventMetadataField: map[string]any{"source": "zarv-billing"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("envelope =\n  %v\nwant\n  %v", got, want)
	}
}

// The producer's own `metadata` field is theirs: asset/asset, users/audit and
// id/verification send one in data. The event's metadata goes beside it.
func TestTheProducersOwnMetadataFieldIsKept(t *testing.T) {
	ev := Event{
		Domain: "asset", Service: "asset",
		Data:     map[string]any{"id": "a1", "metadata": map[string]any{"color": "red"}},
		Metadata: map[string]any{"source": "asset-management"},
	}
	got, err := ev.envelope()
	if err != nil {
		t.Fatal(err)
	}
	data := got["data"].(map[string]any)
	if !reflect.DeepEqual(data["metadata"], map[string]any{"color": "red"}) {
		t.Errorf("data.metadata = %v, the producer's field was overwritten", data["metadata"])
	}
	if !reflect.DeepEqual(data[EventMetadataField], map[string]any{"source": "asset-management"}) {
		t.Errorf("data.%s = %v", EventMetadataField, data[EventMetadataField])
	}
}

// Building the envelope must not touch the caller's map: a retry, or the
// caller's own use of it afterwards, would see the event metadata inside.
func TestTheCallersDataIsNotMutated(t *testing.T) {
	data := map[string]any{"id": "p1"}
	ev := Event{Domain: "billing", Service: "plan", Data: data, Metadata: map[string]any{"k": "v"}}
	if _, err := ev.envelope(); err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 {
		t.Errorf("the caller's data was mutated: %v", data)
	}
}

func TestNoMetadataMeansNoEventMetadataField(t *testing.T) {
	got, err := Event{Domain: "billing", Service: "plan", Data: map[string]any{"id": "p1"}}.envelope()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["data"].(map[string]any)[EventMetadataField]; ok {
		t.Errorf("%s was added with no metadata to carry", EventMetadataField)
	}
}

// The gateway accepts INSERT, UPDATE and DELETE, case-insensitive, and reads an
// empty one as INSERT. History also has CREATE, REPLACE and UPSERT, which have
// an unambiguous meaning; anything else goes through for the gateway to refuse.
func TestTheOperationIsTranslatedToTheGatewaysVerbs(t *testing.T) {
	for in, want := range map[string]any{
		"":        nil,
		"insert":  "INSERT",
		"update":  "UPDATE",
		"DELETE":  "DELETE",
		"CREATE":  "INSERT",
		"create":  "INSERT",
		"REPLACE": "UPDATE",
		"UPSERT":  "UPDATE",
		"NOOP":    "NOOP",
	} {
		got, err := Event{Domain: "billing", Service: "plan", Operation: in, Data: map[string]any{"id": 1}}.envelope()
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got["operation"] != want {
			t.Errorf("operation %q became %v, want %v", in, got["operation"], want)
		}
	}
}

func TestAnEventWithoutDataIsRefused(t *testing.T) {
	_, err := Event{Domain: "billing", Service: "plan"}.envelope()
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("err = %v, want ErrInvalidEvent", err)
	}
}

func TestAProvidersEventLandsInItsProvidersTable(t *testing.T) {
	got, err := Event{
		Domain: "id", Service: "providers", Data: map[string]any{"cpf": "x"},
		Metadata: map[string]any{"provider": "bigdatacorp", "path": "peoplev2", "dataset": "basic_data,addresses"},
	}.envelope()
	if err != nil {
		t.Fatal(err)
	}
	if got["table_name"] != "id_providers_bigdatacorp_peoplev2" {
		t.Errorf("table_name = %v", got["table_name"])
	}
	md := got["data"].(map[string]any)[EventMetadataField].(map[string]any)
	if md["dataset"] != "basic_data,addresses" {
		t.Errorf("the dataset did not travel in the metadata: %v", md)
	}
}
