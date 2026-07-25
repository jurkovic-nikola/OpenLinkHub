package rgb

import (
	"math"
	"testing"
)

func TestProfileSpeedRange(t *testing.T) {
	tests := []struct {
		profile string
		minimum float64
		maximum float64
	}{
		{profile: "circle", minimum: 1, maximum: 10},
		{profile: "aurora", minimum: 1, maximum: 10},
		{profile: "flame", minimum: 0.1, maximum: 10},
		{profile: "cyberpunkglitch", minimum: 0.1, maximum: 10},
	}

	for _, test := range tests {
		t.Run(test.profile, func(t *testing.T) {
			minimum, maximum := ProfileSpeedRange(test.profile)
			if minimum != test.minimum || maximum != test.maximum {
				t.Fatalf(
					"ProfileSpeedRange(%q) = (%v, %v), want (%v, %v)",
					test.profile,
					minimum,
					maximum,
					test.minimum,
					test.maximum,
				)
			}
		})
	}
}

func TestHasSpeedControl(t *testing.T) {
	noSpeedProfiles := []string{
		"static",
		"cpu-temperature",
		"gpu-temperature",
		"liquid-temperature",
		"probe-temperature",
		"off",
	}
	for _, profile := range noSpeedProfiles {
		if HasSpeedControl(profile) {
			t.Errorf("HasSpeedControl(%q) = true, want false", profile)
		}
	}

	for _, profile := range []string{"storm", "circle"} {
		if !HasSpeedControl(profile) {
			t.Errorf("HasSpeedControl(%q) = false, want true", profile)
		}
	}
}

func TestProfileSpeedForUpdatePreservesNoSpeedProfiles(t *testing.T) {
	if actual := ProfileSpeedForUpdate("static", 0, 7.5); actual != 7.5 {
		t.Errorf("ProfileSpeedForUpdate(static) = %v, want 7.5", actual)
	}
	if actual := ProfileSpeedForUpdate("circle", 3.5, 7.5); actual != 3.5 {
		t.Errorf("ProfileSpeedForUpdate(circle) = %v, want 3.5", actual)
	}
}

func TestRainSpeedFactorPreservesLevelsAndInterpolates(t *testing.T) {
	tests := []struct {
		stored float64
		factor float64
	}{
		{stored: 1, factor: 0.5},
		{stored: 1.5, factor: 0.4},
		{stored: 2, factor: 0.3},
		{stored: 2.5, factor: 0.2},
		{stored: 3, factor: 0.1},
		{stored: 0.5, factor: 1},
		{stored: 4, factor: 1},
	}

	for _, test := range tests {
		if actual := rainSpeedFactor(test.stored); math.Abs(actual-test.factor) > 0.0000001 {
			t.Errorf("rainSpeedFactor(%v) = %v, want %v", test.stored, actual, test.factor)
		}
	}
}

func TestStormFlashChanceUsesDurationSpeed(t *testing.T) {
	tests := []struct {
		speed  float64
		chance float32
	}{
		{speed: 1, chance: 0.004},
		{speed: 4, chance: 0.001},
		{speed: 10, chance: 0.0004},
		{speed: 0, chance: 0.001},
	}

	for _, test := range tests {
		if actual := stormFlashChance(test.speed); math.Abs(float64(actual-test.chance)) > 0.0000001 {
			t.Errorf("stormFlashChance(%v) = %v, want %v", test.speed, actual, test.chance)
		}
	}
}
