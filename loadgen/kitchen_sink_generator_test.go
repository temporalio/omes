package loadgen

import (
	"math/rand"
	"testing"

	"github.com/temporalio/omes/loadgen/kitchensink"
)

func TestDeterministicKitchenSinkGeneration(t *testing.T) {
	seed := uint64(12345)

	// Generate multiple test inputs with the same seed
	const numTests = 5
	inputs := make([]*kitchensink.TestInput, numTests)

	for i := 0; i < numTests; i++ {
		input, err := GenerateKitchenSinkTestInput(seed, nil)
		if err != nil {
			t.Fatalf("Failed to generate input %d: %v", i, err)
		}
		inputs[i] = input
	}

	// Compare structural properties that should be identical
	baseline := inputs[0]

	for i := 1; i < numTests; i++ {
		current := inputs[i]

		// Compare number of action sets
		if len(baseline.ClientSequence.ActionSets) != len(current.ClientSequence.ActionSets) {
			t.Errorf("Input %d has different number of action sets: %d vs %d",
				i, len(baseline.ClientSequence.ActionSets), len(current.ClientSequence.ActionSets))
		}

		// Compare WorkflowInput presence
		baselineHasWI := baseline.WorkflowInput != nil
		currentHasWI := current.WorkflowInput != nil
		if baselineHasWI != currentHasWI {
			t.Errorf("Input %d has different WorkflowInput presence: %v vs %v",
				i, baselineHasWI, currentHasWI)
		}

		// Compare WithStartAction presence
		baselineHasWSA := baseline.WithStartAction != nil
		currentHasWSA := current.WithStartAction != nil
		if baselineHasWSA != currentHasWSA {
			t.Errorf("Input %d has different WithStartAction presence: %v vs %v",
				i, baselineHasWSA, currentHasWSA)
		}

		// Compare first action set details if available
		if len(baseline.ClientSequence.ActionSets) > 0 && len(current.ClientSequence.ActionSets) > 0 {
			baseAS := baseline.ClientSequence.ActionSets[0]
			currentAS := current.ClientSequence.ActionSets[0]

			if len(baseAS.Actions) != len(currentAS.Actions) {
				t.Errorf("Input %d first action set has different number of actions: %d vs %d",
					i, len(baseAS.Actions), len(currentAS.Actions))
			}

			if baseAS.Concurrent != currentAS.Concurrent {
				t.Errorf("Input %d first action set has different concurrent flag: %v vs %v",
					i, baseAS.Concurrent, currentAS.Concurrent)
			}

			// Compare wait times (should be deterministic)
			baseWait := baseAS.WaitAtEnd
			currentWait := currentAS.WaitAtEnd
			baseHasWait := baseWait != nil
			currentHasWait := currentWait != nil

			if baseHasWait != currentHasWait {
				t.Errorf("Input %d first action set has different WaitAtEnd presence: %v vs %v",
					i, baseHasWait, currentHasWait)
			} else if baseHasWait && currentHasWait {
				if baseWait.Seconds != currentWait.Seconds || baseWait.Nanos != currentWait.Nanos {
					t.Errorf("Input %d first action set has different WaitAtEnd duration: %v vs %v",
						i, baseWait, currentWait)
				}
			}
		}
	}

	t.Logf("SUCCESS: Same seed produced structurally identical outputs across %d generations", numTests)
	t.Logf("- Action sets: %d", len(baseline.ClientSequence.ActionSets))
	t.Logf("- Has workflow input: %v", baseline.WorkflowInput != nil)
	t.Logf("- Has with-start action: %v", baseline.WithStartAction != nil)
}

func TestExampleGeneration(t *testing.T) {
	input := GenerateExampleKitchenSinkInput()
	if input == nil {
		t.Fatal("Generated example input is nil")
	}

	if input.ClientSequence == nil {
		t.Fatal("Generated example input has no client sequence")
	}

	if len(input.ClientSequence.ActionSets) == 0 {
		t.Fatal("Generated example input has no action sets")
	}

	t.Logf("Generated example input with %d action sets", len(input.ClientSequence.ActionSets))
}

func TestRandomSeedGeneration(t *testing.T) {
	seed1, err := GenerateRandomSeed()
	if err != nil {
		t.Fatalf("Failed to generate first random seed: %v", err)
	}

	seed2, err := GenerateRandomSeed()
	if err != nil {
		t.Fatalf("Failed to generate second random seed: %v", err)
	}

	if seed1 == seed2 {
		t.Errorf("Generated identical random seeds: %d", seed1)
	}

	t.Logf("Generated random seeds: %d, %d", seed1, seed2)
}

// TestGoRandDeterminism tests that Go's rand produces identical sequences
func TestGoRandDeterminism(t *testing.T) {
	seed := int64(54321)

	// Generate two rand instances with the same seed
	rng1 := rand.New(rand.NewSource(seed))
	rng2 := rand.New(rand.NewSource(seed))

	// Generate sequences of numbers
	const numValues = 100
	values1 := make([]int, numValues)
	values2 := make([]int, numValues)

	for i := 0; i < numValues; i++ {
		values1[i] = rng1.Int()
		values2[i] = rng2.Int()
	}

	// Compare sequences
	for i := 0; i < numValues; i++ {
		if values1[i] != values2[i] {
			t.Errorf("Rand sequences differ at index %d: %d vs %d", i, values1[i], values2[i])
		}
	}

	// Test other methods as well
	rng3 := rand.New(rand.NewSource(seed))
	rng4 := rand.New(rand.NewSource(seed))

	for i := 0; i < 50; i++ {
		if rng3.Float32() != rng4.Float32() {
			t.Errorf("Rand Float32() differs at call %d", i)
		}

		if rng3.Intn(100) != rng4.Intn(100) {
			t.Errorf("Rand Intn(100) differs at call %d", i)
		}
	}

	t.Logf("SUCCESS: Go rand produces identical sequences with same seed")
}

// TestDeepStructuralDeterminism tests deeper structural properties
func TestDeepStructuralDeterminism(t *testing.T) {
	seed := uint64(98765)

	// Generate two inputs
	input1, err := GenerateKitchenSinkTestInput(seed, nil)
	if err != nil {
		t.Fatalf("Failed to generate first input: %v", err)
	}

	input2, err := GenerateKitchenSinkTestInput(seed, nil)
	if err != nil {
		t.Fatalf("Failed to generate second input: %v", err)
	}

	// Deep structural comparison of first few action sets
	maxToCheck := len(input1.ClientSequence.ActionSets)
	if len(input2.ClientSequence.ActionSets) < maxToCheck {
		maxToCheck = len(input2.ClientSequence.ActionSets)
	}
	if maxToCheck > 3 {
		maxToCheck = 3 // Limit to first 3 for performance
	}

	for i := 0; i < maxToCheck; i++ {
		as1 := input1.ClientSequence.ActionSets[i]
		as2 := input2.ClientSequence.ActionSets[i]

		// Check action count
		if len(as1.Actions) != len(as2.Actions) {
			t.Errorf("Action set %d: different action count: %d vs %d",
				i, len(as1.Actions), len(as2.Actions))
		}

		// Check each action type at high level
		minActions := len(as1.Actions)
		if len(as2.Actions) < minActions {
			minActions = len(as2.Actions)
		}
		if minActions > 5 {
			minActions = 5 // Limit for performance
		}

		for j := 0; j < minActions; j++ {
			action1 := as1.Actions[j]
			action2 := as2.Actions[j]

			// Check action variant types
			variant1 := getActionVariantType(action1)
			variant2 := getActionVariantType(action2)

			if variant1 != variant2 {
				t.Errorf("Action set %d, action %d: different variants: %s vs %s",
					i, j, variant1, variant2)
			}
		}
	}

	t.Logf("SUCCESS: Deep structural comparison passed for first %d action sets", maxToCheck)
}

// Helper function to get action variant type as string
func getActionVariantType(action *kitchensink.ClientAction) string {
	if action.GetDoSignal() != nil {
		return "DoSignal"
	}
	if action.GetDoQuery() != nil {
		return "DoQuery"
	}
	if action.GetDoUpdate() != nil {
		return "DoUpdate"
	}
	if action.GetNestedActions() != nil {
		return "NestedActions"
	}
	return "Unknown"
}
