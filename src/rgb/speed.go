package rgb

const (
	defaultMinimumProfileSpeed = 1.0
	maximumProfileSpeed        = 10.0
	calibratedMinimumSpeed     = 0.1
)

// HasSpeedControl reports whether a profile's software renderer uses its
// persisted speed. Profiles that do not use speed retain the field only for
// compatibility with existing profile JSON.
func HasSpeedControl(profile string) bool {
	switch profile {
	case "static",
		"cpu-temperature",
		"gpu-temperature",
		"liquid-temperature",
		"probe-temperature",
		"off":
		return false
	default:
		return true
	}
}

// ProfileSpeedForUpdate preserves stored compatibility data when a renderer
// does not expose a speed control.
func ProfileSpeedForUpdate(profile string, requestedSpeed, storedSpeed float64) float64 {
	if HasSpeedControl(profile) {
		return requestedSpeed
	}
	return storedSpeed
}

// ProfileSpeedRange returns the accepted persisted range for an RGB profile.
// Flame and Cyberpunk Glitch use sub-1 multipliers for their calibrated UI,
// while existing values through 10 remain valid for backward compatibility.
func ProfileSpeedRange(profile string) (float64, float64) {
	switch profile {
	case "flame", "cyberpunkglitch":
		return calibratedMinimumSpeed, maximumProfileSpeed
	default:
		return defaultMinimumProfileSpeed, maximumProfileSpeed
	}
}
