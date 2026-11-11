package pricing

import (
	"strconv"
	"strings"
	"testing"
)

var testCatalog = `{
  "openai": {
    "id": "openai",
    "name": "OpenAI",
    "models": {
      "gpt-4o": {
        "id": "gpt-4o",
        "name": "GPT-4o",
        "description": "` + strings.Repeat("x", 4096) + `",
        "cost": {"input": 2.5, "output": 10, "cache_read": 1.25}
      },
      "gpt-4o-2024-08-06": {
        "id": "gpt-4o-2024-08-06",
        "cost": {"input": 2.5, "output": 10, "cache_read": 1.25}
      },
      "gpt-4.1": {
        "id": "gpt-4.1",
        "cost": {"input": 2, "output": 8, "cache_read": 0.5}
      }
    }
  },
  "reseller": {
    "id": "reseller",
    "models": {
      "openai/gpt-4o": {
        "id": "openai/gpt-4o",
        "cost": {"input": 2.4, "output": 9.6, "cache_read": 1.2}
      }
    }
  },
  "anthropic": {
    "id": "anthropic",
    "models": {
      "claude-sonnet-4-5": {
        "id": "claude-sonnet-4-5",
        "cost": {"input": 3, "output": 15, "cache_read": 0.3}
      },
      "claude-opus-4-5": {
        "id": "claude-opus-4-5",
        "cost": {"input": 5, "output": 25, "cache_read": 0.5}
      }
    }
  },
  "gateway": {
    "id": "gateway",
    "models": {
      "anthropic/claude-sonnet-4-5": {
        "id": "anthropic/claude-sonnet-4-5",
        "cost": {"input": 3.3, "output": 16.5, "cache_read": 0.33}
      }
    }
  },
  "no-price": {
    "id": "no-price",
    "models": {
      "gpt-4o": {"id": "gpt-4o", "cost": null}
    }
  }
}`

func TestDecodeMatchesAndRanksProviders(t *testing.T) {
	suggestions, err := Decode(strings.NewReader(testCatalog), []string{
		"gpt-4o",
		"openai/gpt-4o",
		"claude-sonnet-4-5",
		"missing",
	})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("suggestions = %d, want 3", len(suggestions))
	}

	gpt := suggestions["gpt-4o"]
	if gpt.Provider != "openai" || gpt.Input != 2.5 || gpt.Output != 10 || gpt.Cache != 1.25 {
		t.Fatalf("gpt-4o suggestion = %+v, want the first-party OpenAI offer", gpt)
	}
	if gpt.Providers != 3 {
		t.Fatalf("gpt-4o providers = %d, want 3 (openai, reseller, no-price is skipped)", gpt.Providers)
	}

	prefixed := suggestions["openai/gpt-4o"]
	if prefixed.Provider != "openai" || prefixed.Input != 2.5 {
		t.Fatalf("openai/gpt-4o suggestion = %+v, want the first-party offer over the reseller", prefixed)
	}

	claude := suggestions["claude-sonnet-4-5"]
	if claude.Provider != "anthropic" || claude.Input != 3 || claude.Output != 15 || claude.Cache != 0.3 {
		t.Fatalf("claude-sonnet-4-5 suggestion = %+v, want the Anthropic offer", claude)
	}

	if _, exists := suggestions["missing"]; exists {
		t.Fatal("missing model received a suggestion")
	}
}

func TestDecodeMatchesDateAndVersionSuffixes(t *testing.T) {
	catalog := `{
	  "anthropic": {
	    "models": {
	      "claude-opus-4-5": {"cost": {"input": 5, "output": 25}},
	      "claude-sonnet-4-5@20250929": {"cost": {"input": 3, "output": 15}}
	    }
	  }
	}`
	suggestions, err := Decode(strings.NewReader(catalog), []string{
		"claude-opus-4-5-20251101",
		"claude-sonnet-4-5",
	})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got := suggestions["claude-opus-4-5-20251101"]; got.Provider != "anthropic" || got.Input != 5 {
		t.Fatalf("dated model suggestion = %+v, want the base Anthropic entry", got)
	}
	if got := suggestions["claude-sonnet-4-5"]; got.Provider != "anthropic" || got.Input != 3 {
		t.Fatalf("aliased model suggestion = %+v, want the @-suffixed Anthropic entry", got)
	}
}

func TestDecodeSkipsModelsWithoutPrices(t *testing.T) {
	catalog := `{
	  "free": {"models": {"gpt-4o": {"cost": null}}},
	  "partial": {"models": {"gpt-4o": {"cost": {"cache_read": 1}}}}
	}`
	suggestions, err := Decode(strings.NewReader(catalog), []string{"gpt-4o"})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(suggestions) != 0 {
		t.Fatalf("suggestions = %+v, want none for entries without input/output prices", suggestions)
	}
}

func TestDecodeRoundsPrices(t *testing.T) {
	catalog := `{
	  "vendor": {"models": {"glm-5.3": {"cost": {"input": 1.19516, "output": 4.18306, "cache_read": 0.29879}}}}
	}`
	suggestions, err := Decode(strings.NewReader(catalog), []string{"glm-5.3"})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	got := suggestions["glm-5.3"]
	if got.Input != 1.195 || got.Output != 4.183 || got.Cache != 0.299 {
		t.Fatalf("rounded suggestion = %+v, want 1.195/4.183/0.299", got)
	}
}

func TestDecodeRejectsMalformedCatalog(t *testing.T) {
	if _, err := Decode(strings.NewReader(`{"openai":`), []string{"gpt-4o"}); err == nil {
		t.Fatal("Decode() error = nil for truncated catalog")
	}
	if _, err := Decode(strings.NewReader(`[]`), []string{"gpt-4o"}); err == nil {
		t.Fatal("Decode() error = nil for a non-object catalog")
	}
}

func TestDecodeEmptyWantedReturnsEmptyWithoutReading(t *testing.T) {
	reader := &errorReader{}
	suggestions, err := Decode(reader, nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(suggestions) != 0 {
		t.Fatalf("suggestions = %+v, want empty", suggestions)
	}
	if reader.reads != 0 {
		t.Fatalf("reader was read %d times, want 0", reader.reads)
	}
}

func TestNormalizeModels(t *testing.T) {
	got, err := NormalizeModels([]string{" gpt-4o ", "GPT-4o", "", "claude-sonnet-4-5"})
	if err != nil {
		t.Fatalf("NormalizeModels() error = %v", err)
	}
	if len(got) != 2 || got[0] != "gpt-4o" || got[1] != "claude-sonnet-4-5" {
		t.Fatalf("NormalizeModels() = %#v, want deduplicated trimmed names", got)
	}

	tooMany := make([]string, MaxModels+1)
	for index := range tooMany {
		tooMany[index] = "model-" + strconv.Itoa(index)
	}
	if _, err := NormalizeModels(tooMany); err == nil {
		t.Fatal("NormalizeModels() error = nil for too many models")
	}
}

type errorReader struct {
	reads int
}

func (r *errorReader) Read([]byte) (int, error) {
	r.reads++
	return 0, nil
}
