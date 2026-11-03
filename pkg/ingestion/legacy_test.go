package ingestion

import (
	"errors"
	"reflect"
	"testing"
)

func TestLegacyEventBecomesTheGatewaysEvent(t *testing.T) {
	got, err := LegacyEvent{
		Domain: "ID", Service: "Verification", Operation: "create", Description: "d",
		Data:     map[string]any{"id": "v1"},
		Metadata: map[string]any{"origin": "app"},
	}.Event()
	if err != nil {
		t.Fatal(err)
	}
	want := Event{
		TableName: "id_verification", Operation: "INSERT", Description: "d",
		Data: map[string]any{"id": "v1", EventMetadataField: map[string]any{"origin": "app"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestLegacyEventWithoutMetadataAddsNoEventMetadata(t *testing.T) {
	got, err := LegacyEvent{Domain: "billing", Service: "record", Data: map[string]any{"a": 1}}.Event()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Data[EventMetadataField]; ok {
		t.Fatalf("event_metadata present: %#v", got.Data)
	}
}

func TestLegacyEventDoesNotModifyTheCallersData(t *testing.T) {
	data := map[string]any{"a": 1}
	if _, err := (LegacyEvent{Domain: "d", Service: "s", Data: data, Metadata: map[string]any{"m": 1}}).Event(); err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 {
		t.Fatalf("caller's data was changed: %#v", data)
	}
}

func TestLegacyOperationSynonyms(t *testing.T) {
	cases := []struct{ in, want string }{
		{"create", "INSERT"}, {" Replace ", "UPDATE"}, {"UPSERT", "UPDATE"},
		{"DELETE", "DELETE"}, {"", ""}, {"odd", "odd"},
	}
	for _, c := range cases {
		ev, err := LegacyEvent{Domain: "d", Service: "s", Operation: c.in, Data: map[string]any{}}.Event()
		if err != nil || ev.Operation != c.want {
			t.Errorf("%q -> %q, %v; want %q", c.in, ev.Operation, err, c.want)
		}
	}
}

func TestLegacyTableIsDomainAndServiceNormalised(t *testing.T) {
	for _, c := range [][3]string{
		{"asset", "provider-query", "asset_provider_query"},
		{"marketplace", "vendor.locavia.tarifas", "marketplace_vendor_locavia_tarifas"},
		{"id", "Verification failed", "id_verification_failed"},
		{"zarv_finance", "x", "zarv_finance_x"},
	} {
		ev, err := LegacyEvent{Domain: c[0], Service: c[1], Data: map[string]any{}}.Event()
		if err != nil || ev.TableName != c[2] {
			t.Errorf("%s/%s -> %q, %v; want %q", c[0], c[1], ev.TableName, err, c[2])
		}
	}
}

func TestLegacyProvidersAreSplitByProviderAndPath(t *testing.T) {
	ev, err := LegacyEvent{Domain: "id", Service: "providers", Data: map[string]any{},
		Metadata: map[string]any{"provider": "Serasa", "path": "basic/pf"}}.Event()
	if err != nil || ev.TableName != "id_providers_serasa_basic_pf" {
		t.Fatalf("got %q, %v", ev.TableName, err)
	}
	for _, md := range []map[string]any{nil, {"provider": "serasa"}, {"path": "x"}} {
		if _, err := (LegacyEvent{Domain: "id", Service: "providers", Data: map[string]any{}, Metadata: md}).Event(); !errors.Is(err, ErrInvalidEvent) {
			t.Errorf("metadata %v: err %v, want ErrInvalidEvent", md, err)
		}
	}
}

func TestLegacyEventThatNamesNoTableOrHasNoDataIsRefused(t *testing.T) {
	for _, l := range []LegacyEvent{
		{Service: "s", Data: map[string]any{}},
		{Domain: "d", Data: map[string]any{}},
		{Domain: "---", Service: "s", Data: map[string]any{}},
		{Domain: "d", Service: "s"},
	} {
		if _, err := l.Event(); !errors.Is(err, ErrInvalidEvent) {
			t.Errorf("%#v: err %v, want ErrInvalidEvent", l, err)
		}
	}
}
