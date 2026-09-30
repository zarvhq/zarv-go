package ingestion

import (
	"errors"
	"reflect"
	"testing"
)

// An Event is the gateway's body, field for field; the package adds nothing
// and renames nothing.
func TestTheEnvelopeIsTheEventAsItIs(t *testing.T) {
	ev := Event{
		TableName:   "billing_plan",
		Operation:   "UPDATE",
		Description: "plan renamed",
		UniqueKey:   "id",
		Data:        map[string]any{"id": "p1", "name": "Pro"},
	}
	got, err := ev.envelope()
	if err != nil {
		t.Fatalf("envelope: %v", err)
	}
	want := map[string]any{
		"table_name":  "billing_plan",
		"operation":   "UPDATE",
		"description": "plan renamed",
		"unique_key":  "id",
		"data":        map[string]any{"id": "p1", "name": "Pro"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("envelope =\n  %v\nwant\n  %v", got, want)
	}
}

// The optional fields are left out when empty, so the gateway's own defaults
// apply: INSERT, no description, its default record key.
func TestEmptyOptionalFieldsAreLeftOut(t *testing.T) {
	got, err := Event{TableName: "billing_plan", Data: map[string]any{"id": "p1"}}.envelope()
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"operation", "description", "unique_key"} {
		if _, ok := got[k]; ok {
			t.Errorf("%s was sent empty", k)
		}
	}
}

// Nothing is translated: the gateway is what judges a table name or an
// operation, and its answer reaches the producer.
func TestNothingIsTranslated(t *testing.T) {
	got, err := Event{TableName: "Billing Plan", Operation: "create", Data: map[string]any{"id": 1}}.envelope()
	if err != nil {
		t.Fatal(err)
	}
	if got["table_name"] != "Billing Plan" || got["operation"] != "create" {
		t.Errorf("table_name, operation = %v, %v; want them as the producer sent them", got["table_name"], got["operation"])
	}
}

// The request cannot be built without a table or a record, so those two are
// refused before anything is sent; everything else is the gateway's to check.
func TestAnEventWithoutATableOrDataIsRefused(t *testing.T) {
	for name, ev := range map[string]Event{
		"no table": {Data: map[string]any{"id": 1}},
		"blank":    {TableName: "  ", Data: map[string]any{"id": 1}},
		"no data":  {TableName: "billing_plan"},
	} {
		if _, err := ev.envelope(); !errors.Is(err, ErrInvalidEvent) {
			t.Errorf("%s: err = %v, want ErrInvalidEvent", name, err)
		}
	}
}

// The caller's map is sent as is and never modified.
func TestTheCallersDataIsNotMutated(t *testing.T) {
	data := map[string]any{"id": "p1"}
	if _, err := (Event{TableName: "billing_plan", Data: data}).envelope(); err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 {
		t.Errorf("the caller's data was mutated: %v", data)
	}
}
