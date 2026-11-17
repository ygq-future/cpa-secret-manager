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
	if len(gpt.Offers) != 3 || gpt.Offers[0].Provider != "openai" || gpt.Offers[0].Model != "gpt-4o" {
		t.Fatalf("gpt-4o offers = %+v, want the three ranked listings with the OpenAI offer first", gpt.Offers)
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

func TestDecodeMatchesPrefixFallback(t *testing.T) {
	catalog := `{
	  "google": {
	    "models": {
	      "gemini-3.5-flash": {"id": "gemini-3.5-flash", "cost": {"input": 0.3, "output": 2.5, "cache_read": 0.075}},
	      "gemini-3.5-flash-lite": {"id": "gemini-3.5-flash-lite", "cost": {"input": 0.1, "output": 0.4}},
	      "gemini-2.5-pro": {"id": "gemini-2.5-pro", "cost": {"input": 1.25, "output": 10}}
	    }
	  },
	  "reseller": {
	    "models": {
	      "gemini-3.5-flash-lite": {"id": "gemini-3.5-flash-lite", "cost": {"input": 0.12, "output": 0.5}}
	    }
	  }
	}`
	suggestions, err := Decode(strings.NewReader(catalog), []string{"gemini-3.5-flash-high"})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	got, exists := suggestions["gemini-3.5-flash-high"]
	if !exists {
		t.Fatal("a model with a trailing marker did not fall back to its prefix family")
	}
	if got.Model != "gemini-3.5-flash" || got.Provider != "google" {
		t.Fatalf("best offer = %s/%s, want the first-party gemini-3.5-flash", got.Provider, got.Model)
	}
	if got.Providers != 3 || len(got.Offers) != 3 {
		t.Fatalf("offers = %d/%d, want the three listings at the deepest prefix stage", got.Providers, len(got.Offers))
	}
	if first := got.Offers[0]; first.Model != "gemini-3.5-flash" || first.Provider != "google" {
		t.Fatalf("first offer = %+v, want the exact-prefix first-party listing", first)
	}
	for _, offer := range got.Offers {
		if offer.Model == "gemini-2.5-pro" {
			t.Fatal("a shallower prefix stage leaked into the offer list")
		}
	}
}

func TestDecodeRequiresSegmentBoundary(t *testing.T) {
	catalog := `{
	  "google": {
	    "models": {
	      "gemini-3.5-flashlight": {"id": "gemini-3.5-flashlight", "cost": {"input": 9, "output": 9}},
	      "gemini-3.5": {"id": "gemini-3.5", "cost": {"input": 0.3, "output": 2.5}}
	    }
	  }
	}`
	suggestions, err := Decode(strings.NewReader(catalog), []string{"gemini-3.5-flash"})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	got := suggestions["gemini-3.5-flash"]
	if got.Model != "gemini-3.5" {
		t.Fatalf("best offer = %s, want the shorter gemini-3.5 prefix match", got.Model)
	}
	if len(got.Offers) != 2 {
		t.Fatalf("offers = %+v, want both boundary matches", got.Offers)
	}
}

func TestDecodePrefersDeepestPrefixStage(t *testing.T) {
	catalog := `{
	  "google": {
	    "models": {
	      "gemini-3.5-flash": {"id": "gemini-3.5-flash", "cost": {"input": 0.3, "output": 2.5}}
	    }
	  },
	  "reseller": {
	    "models": {
	      "gemini-3.5-flash-high": {"id": "gemini-3.5-flash-high", "cost": {"input": 9, "output": 9}}
	    }
	  }
	}`
	suggestions, err := Decode(strings.NewReader(catalog), []string{"gemini-3.5-flash-high"})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	got := suggestions["gemini-3.5-flash-high"]
	if got.Model != "gemini-3.5-flash-high" || got.Provider != "reseller" {
		t.Fatalf("best offer = %s/%s, want the deeper exact match", got.Provider, got.Model)
	}
	if len(got.Offers) != 1 {
		t.Fatalf("offers = %+v, want only the deepest stage", got.Offers)
	}
}

func TestDecodeRejectsWeakFamilyPrefix(t *testing.T) {
	catalog := `{
	  "anthropic": {
	    "models": {
	      "claude-sonnet-4-5": {"id": "claude-sonnet-4-5", "cost": {"input": 3, "output": 15}}
	    }
	  }
	}`
	suggestions, err := Decode(strings.NewReader(catalog), []string{"claude-3-5-haiku"})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if _, exists := suggestions["claude-3-5-haiku"]; exists {
		t.Fatal("a single family segment must not match an unrelated multi-token name")
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

func TestDecodeMatchesPrefixSkipAndCompact(t *testing.T) {
	catalog := `{
	  "google": {
	    "models": {
	      "gemini-3.5-flash": {"id": "gemini-3.5-flash", "cost": {"input": 0.3, "output": 2.5}},
	      "gemini-3.5-flashlight": {"id": "gemini-3.5-flashlight", "cost": {"input": 0.9, "output": 3.0}}
	    }
	  },
	  "tencent": {
	    "models": {
	      "tencent/hy4-flash": {"id": "tencent/hy4-flash", "cost": {"input": 0.14, "output": 0.28}}
	    }
	  }
	}`

	suggestions, err := Decode(strings.NewReader(catalog), []string{
		"wb-gemini-3.5-flash",
		"custom/wb-gemini-3.5-flash",
		"hy4flash",
		"wb-hy4flash",
	})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	for _, name := range []string{"wb-gemini-3.5-flash", "custom/wb-gemini-3.5-flash"} {
		got, ok := suggestions[name]
		if !ok || got.Provider != "google" || got.Model != "gemini-3.5-flash" {
			t.Fatalf("%s suggestion = %+v (ok=%v), want google/gemini-3.5-flash", name, got, ok)
		}
	}

	for _, name := range []string{"hy4flash", "wb-hy4flash"} {
		got, ok := suggestions[name]
		if !ok || got.Provider != "tencent" || got.Input != 0.14 {
			t.Fatalf("%s suggestion = %+v (ok=%v), want tencent/hy4-flash (input=0.14)", name, got, ok)
		}
	}

	compactCatalog := `{
	  "custom": {
	    "models": {
	      "hy4flash": {"id": "hy4flash", "cost": {"input": 0.2, "output": 0.4}}
	    }
	  }
	}`
	compactSuggestions, err := Decode(strings.NewReader(compactCatalog), []string{"hy4-flash"})
	if err != nil {
		t.Fatalf("Decode(compactCatalog) error = %v", err)
	}
	if got, ok := compactSuggestions["hy4-flash"]; !ok || got.Provider != "custom" || got.Input != 0.2 {
		t.Fatalf("hy4-flash suggestion = %+v (ok=%v), want custom/hy4flash (input=0.2)", got, ok)
	}
}
