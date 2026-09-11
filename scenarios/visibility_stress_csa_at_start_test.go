package scenarios

import (
	"testing"

	"go.temporal.io/api/enums/v1"
)

// Verifies buildStartSearchAttributes covers every CSA in a preset and maps each declared
// type to a usable typed key. A silent type mismatch here would populate the corpus with
// attributes the querier's filters cannot match, which is the exact failure the option
// exists to prevent.
func TestBuildStartSearchAttributes_CoversPresetAndTypes(t *testing.T) {
	for _, preset := range []string{"small", "medium", "heavy"} {
		names := vsCSAPresets[preset]
		defs := make([]csaDef, 0, len(names))
		for _, n := range names {
			d, err := parseCSAName(n)
			if err != nil {
				t.Fatalf("preset %s: parseCSAName(%q): %v", preset, n, err)
			}
			defs = append(defs, d)
		}

		rng := newVSRand(42, vsDefaultVocabZipfSkew)
		sa, err := buildStartSearchAttributes(defs, rng)
		if err != nil {
			t.Fatalf("preset %s: %v", preset, err)
		}
		if got, want := sa.Size(), len(defs); got != want {
			t.Errorf("preset %s: got %d attributes, want %d", preset, got, want)
		}
		// Every declared type must round-trip; UNSPECIFIED would mean a def we cannot map.
		for _, d := range defs {
			if d.indexedValueType() == enums.INDEXED_VALUE_TYPE_UNSPECIFIED {
				t.Errorf("preset %s: %s has unspecified type", preset, d.Name)
			}
			if _, err := startSearchAttribute(d, randomCSAValue(d, rng)); err != nil {
				t.Errorf("preset %s: startSearchAttribute(%s): %v", preset, d.Name, err)
			}
		}
	}
}

// A value of the wrong Go type must be rejected rather than silently coerced.
func TestStartSearchAttribute_RejectsWrongValueType(t *testing.T) {
	d, err := parseCSAName("VS_Int_01")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := startSearchAttribute(d, "not-a-number"); err == nil {
		t.Error("expected an error for a string value on an Int CSA, got nil")
	}
}
