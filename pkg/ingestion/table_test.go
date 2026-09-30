package ingestion

import (
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// Every domain/service and every id/providers provider|path production has
// sent, from staging.platform_raw on 2026-09-30.
var (
	//go:embed testdata/services.json
	historicalServices []byte
	//go:embed testdata/providers.json
	historicalProviders []byte
)

func TestTableNameIsDomainAndServiceNormalised(t *testing.T) {
	for _, c := range []struct{ domain, service, want string }{
		{"billing", "plan", "billing_plan"},
		{"id", "Verification failed", "id_verification_failed"},
		{"marketplace", "vendor.locavia.formas-contrato", "marketplace_vendor_locavia_formas_contrato"},
		{"asset", "provider-query", "asset_provider_query"},
		{"lens", "research--analyze__review", "lens_research_analyze_review"},
		{" Users ", " Audit ", "users_audit"},
	} {
		got, err := TableName(c.domain, c.service, nil)
		if err != nil {
			t.Fatalf("TableName(%q, %q): %v", c.domain, c.service, err)
		}
		if got != c.want {
			t.Errorf("TableName(%q, %q) = %q, want %q", c.domain, c.service, got, c.want)
		}
	}
}

// id/providers is split by the provider called and the endpoint it was called
// on, both read from the event's metadata. Path, not dataset: dataset is a
// comma list whose every combination would be a table of its own.
func TestProvidersAreSplitByProviderAndPath(t *testing.T) {
	for _, c := range []struct{ provider, path, want string }{
		{"bigdatacorp", "peoplev2", "id_providers_bigdatacorp_peoplev2"},
		{"Judit", "/lawsuits", "id_providers_judit_lawsuits"},
		{"jucec", "/api-balcao-unico/fluxo/buscar-dados-rfb/", "id_providers_jucec_api_balcao_unico_fluxo_buscar_dados_rfb"},
		{"Procob", "v2/L0001", "id_providers_procob_v2_l0001"},
	} {
		md := map[string]any{"provider": c.provider, "path": c.path, "dataset": "basic_data,addresses,phones,emails"}
		got, err := TableName("id", "providers", md)
		if err != nil {
			t.Fatalf("TableName(id, providers, %s|%s): %v", c.provider, c.path, err)
		}
		if got != c.want {
			t.Errorf("TableName(id, providers, %s|%s) = %q, want %q", c.provider, c.path, got, c.want)
		}
	}
}

func TestAProvidersEventWithoutProviderOrPathIsRefusedNamingTheField(t *testing.T) {
	for field, md := range map[string]map[string]any{
		"provider": {"path": "peoplev2"},
		"path":     {"provider": "bigdatacorp", "path": "  "},
	} {
		_, err := TableName("id", "providers", md)
		if !errors.Is(err, ErrInvalidEvent) {
			t.Fatalf("missing %s: err = %v, want ErrInvalidEvent", field, err)
		}
		if !strings.Contains(err.Error(), "metadata."+field) {
			t.Errorf("the error does not name metadata.%s: %v", field, err)
		}
	}
	if _, err := TableName("id", "providers", nil); !errors.Is(err, ErrInvalidEvent) {
		t.Errorf("no metadata at all: err = %v, want ErrInvalidEvent", err)
	}
}

// One historical event carries neither domain nor service. It must be refused,
// not land in a table called "_".
func TestAnEventWithoutDomainOrServiceIsRefused(t *testing.T) {
	for _, c := range [][2]string{{"", ""}, {"billing", ""}, {"", "plan"}, {"--", "plan"}} {
		if _, err := TableName(c[0], c[1], nil); !errors.Is(err, ErrInvalidEvent) {
			t.Errorf("TableName(%q, %q): err = %v, want ErrInvalidEvent", c[0], c[1], err)
		}
	}
}

// Every domain/service production has ever sent, and every provider|path of
// id/providers, taken from staging.platform_raw on 2026-09-30. No two may land
// in one table, and each must be a name the gateway can route.
func TestEveryHistoricalNameIsDistinctAndRoutable(t *testing.T) {
	var services []struct{ Domain, Service string }
	var providers []struct{ Provider, Path string }
	load(t, historicalServices, &services)
	load(t, historicalProviders, &providers)

	seen := map[string]string{}
	add := func(name, from string) {
		if !TablePattern.MatchString(name) {
			t.Errorf("%s → %q does not match %s", from, name, TablePattern)
		}
		if prev, ok := seen[name]; ok {
			t.Errorf("%s and %s both land in %q", prev, from, name)
		}
		seen[name] = from
	}
	for _, s := range services {
		if s.Domain == "id" && s.Service == "providers" {
			continue // split below
		}
		name, err := TableName(s.Domain, s.Service, nil)
		if s.Domain == "" || s.Service == "" {
			if err == nil {
				t.Errorf("%q/%q was given a table: %q", s.Domain, s.Service, name)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s/%s: %v", s.Domain, s.Service, err)
			continue
		}
		add(name, s.Domain+"/"+s.Service)
	}
	for _, p := range providers {
		name, err := TableName("id", "providers", map[string]any{"provider": p.Provider, "path": p.Path})
		if err != nil {
			t.Errorf("id/providers %s|%s: %v", p.Provider, p.Path, err)
			continue
		}
		add(name, "id/providers "+p.Provider+"|"+p.Path)
	}

	longest := ""
	for name := range seen {
		if len(name) > len(longest) {
			longest = name
		}
	}
	t.Logf("%d tables; the longest is %d characters: %s", len(seen), len(longest), longest)
}

func load(t *testing.T, fixture []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(fixture, v); err != nil {
		t.Fatalf("fixture: %v", err)
	}
}
