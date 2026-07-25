(function (root, factory) {
    const speed = factory();

    if (typeof module === "object" && module.exports) {
        module.exports = speed;
    }

    root.LumenForgeRgbSpeed = speed;
}(typeof globalThis !== "undefined" ? globalThis : this, function () {
    "use strict";

    const SOFTWARE_CONTROL = "software";
    const HARDWARE_CONTROL = "hardware";
    const DURATION_MAPPING = "duration";
    const IDENTITY_MAPPING = "identity";
    const HARDWARE_MAPPING = "hardware-discrete";
    const NO_SPEED_CONTROL = "none";

    const durationProfiles = new Set([
        "arc",
        "circle",
        "circleshift",
        "colorpulse",
        "colorshift",
        "colorwarp",
        "comet",
        "datastream",
        "flickering",
        "gradient",
        "marquee",
        "nebula",
        "pastelrainbow",
        "pastelspiralrainbow",
        "plasmacore",
        "rain",
        "rainbow",
        "rotarystack",
        "rotator",
        "sequential",
        "spinner",
        "spiralrainbow",
        "stardust",
        "storm",
        "visor",
        "watercolor",
        "wave"
    ]);
    const identityProfiles = new Set([
        "aurora",
        "cyberpunkglitch",
        "flame",
        "tokyonight"
    ]);
    const noSpeedProfiles = new Set([
        "cputemperature",
        "gputemperature",
        "liquidtemperature",
        "probetemperature",
        "static",
        "off"
    ]);
    const hardwareOnlyProfiles = new Set([
        "tlk",
        "tlr",
        "rainbowwave",
        "colorwave"
    ]);
    const discreteHardwareProfiles = new Set([
        "tlk",
        "tlr",
        "spiralrainbow",
        "rainbowwave",
        "rain",
        "visor",
        "colorwave"
    ]);
    const calibratedStoredRanges = Object.freeze({
        flame: {min: 0.1, max: 1.5},
        cyberpunkglitch: {min: 0.1, max: 1.0},
        rain: {min: 1, max: 3}
    });
    const profileAliases = Object.freeze({
        typelightingkey: "tlk",
        typelightingripple: "tlr"
    });

    function numberOr(value, fallback) {
        if (value === null || value === undefined || value === "") {
            return fallback;
        }
        const parsed = Number(value);
        return Number.isFinite(parsed) ? parsed : fallback;
    }

    function decimalPlaces(value) {
        const text = String(value).toLowerCase();
        if (text.includes("e-")) {
            return Number(text.split("e-")[1]);
        }

        const decimal = text.split(".")[1];
        return decimal ? decimal.length : 0;
    }

    function normalizeProfile(profile) {
        const normalized = String(profile || "")
            .trim()
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, "");
        return Object.prototype.hasOwnProperty.call(profileAliases, normalized)
            ? profileAliases[normalized]
            : normalized;
    }

    function normalizeControlMode(controlMode) {
        return controlMode === HARDWARE_CONTROL ? HARDWARE_CONTROL : SOFTWARE_CONTROL;
    }

    function profileKeys(profiles) {
        if (profiles == null || typeof profiles !== "object") {
            return [];
        }

        return Object.keys(profiles).map(normalizeProfile);
    }

    function controlModeForProfiles(profiles) {
        return profileKeys(profiles).some(function (profile) {
            return hardwareOnlyProfiles.has(profile);
        }) ? HARDWARE_CONTROL : SOFTWARE_CONTROL;
    }

    function classificationForProfile(profile, controlMode) {
        const normalized = normalizeProfile(profile);
        const mode = normalizeControlMode(controlMode);

        if (normalized === "" || noSpeedProfiles.has(normalized)) {
            return NO_SPEED_CONTROL;
        }
        if (mode === HARDWARE_CONTROL && discreteHardwareProfiles.has(normalized)) {
            return HARDWARE_MAPPING;
        }
        if (identityProfiles.has(normalized)) {
            return IDENTITY_MAPPING;
        }
        if (durationProfiles.has(normalized)) {
            return DURATION_MAPPING;
        }

        // Existing and third-party profiles historically used duration-like storage.
        return DURATION_MAPPING;
    }

    function hasSpeedControl(profile, controlMode) {
        return classificationForProfile(profile, controlMode) !== NO_SPEED_CONTROL;
    }

    function settings(minimum, maximum, step) {
        const min = numberOr(minimum, 0);
        const max = numberOr(maximum, min);
        const increment = Math.abs(numberOr(step, 1)) || 1;

        if (max < min) {
            return {
                min: max,
                max: min,
                step: increment,
                precision: decimalPlaces(increment)
            };
        }

        return {
            min,
            max,
            step: increment,
            precision: decimalPlaces(increment)
        };
    }

    function clamp(value, minimum, maximum) {
        return Math.min(maximum, Math.max(minimum, value));
    }

    function quantize(value, options) {
        const clamped = clamp(numberOr(value, options.min), options.min, options.max);
        const stepped = options.min + Math.round((clamped - options.min) / options.step) * options.step;
        return Number(clamp(stepped, options.min, options.max).toFixed(options.precision));
    }

    function mapRange(value, sourceMin, sourceMax, targetMin, targetMax) {
        if (sourceMax === sourceMin) {
            return targetMin;
        }

        const ratio = (value - sourceMin) / (sourceMax - sourceMin);
        return targetMin + ratio * (targetMax - targetMin);
    }

    function usesDurationSemantics(profile, controlMode) {
        const classification = classificationForProfile(profile, controlMode);
        return classification === DURATION_MAPPING || classification === HARDWARE_MAPPING;
    }

    function rangeForProfile(profile, controlMode) {
        if (classificationForProfile(profile, controlMode) === HARDWARE_MAPPING) {
            return {min: 1, max: 3, step: 1};
        }

        return {min: 1, max: 10, step: 0.1};
    }

    function storedRangeForProfile(profile, controlMode, uiOptions) {
        const normalized = normalizeProfile(profile);
        const classification = classificationForProfile(normalized, controlMode);

        if (classification === HARDWARE_MAPPING) {
            return {min: 1, max: 3};
        }
        if (normalizeControlMode(controlMode) === SOFTWARE_CONTROL &&
            Object.prototype.hasOwnProperty.call(calibratedStoredRanges, normalized)) {
            return calibratedStoredRanges[normalized];
        }

        return {min: uiOptions.min, max: uiOptions.max};
    }

    function configureSlider(slider, profile, controlMode) {
        const mode = normalizeControlMode(controlMode);
        const range = rangeForProfile(profile, mode);
        slider.min = String(range.min);
        slider.max = String(range.max);
        slider.step = String(range.step);
        slider.dataset.rgbProfile = profile || "";
        slider.dataset.speedControlMode = mode;
        delete slider.dataset.storedSpeed;
        slider.dataset.speedEdited = "false";
        return range;
    }

    function storedToUi(value, minimum, maximum, step, profile, controlMode) {
        const uiOptions = settings(minimum, maximum, step);
        const storedRange = storedRangeForProfile(profile, controlMode, uiOptions);
        const storedValue = clamp(numberOr(value, storedRange.min), storedRange.min, storedRange.max);
        const classification = classificationForProfile(profile, controlMode);
        const mapped = usesDurationSemantics(profile, controlMode)
            ? mapRange(storedValue, storedRange.max, storedRange.min, uiOptions.min, uiOptions.max)
            : mapRange(storedValue, storedRange.min, storedRange.max, uiOptions.min, uiOptions.max);

        if (classification === NO_SPEED_CONTROL) {
            return quantize(value, uiOptions);
        }
        return quantize(mapped, uiOptions);
    }

    function uiToStored(value, minimum, maximum, step, profile, controlMode) {
        const uiOptions = settings(minimum, maximum, step);
        const uiValue = quantize(value, uiOptions);
        const storedRange = storedRangeForProfile(profile, controlMode, uiOptions);
        const classification = classificationForProfile(profile, controlMode);
        const mapped = usesDurationSemantics(profile, controlMode)
            ? mapRange(uiValue, uiOptions.min, uiOptions.max, storedRange.max, storedRange.min)
            : mapRange(uiValue, uiOptions.min, uiOptions.max, storedRange.min, storedRange.max);

        if (classification === NO_SPEED_CONTROL) {
            return uiValue;
        }
        return Number(mapped.toFixed(classification === HARDWARE_MAPPING ? 0 : 3));
    }

    function sliderOptions(slider) {
        return {
            min: slider.min,
            max: slider.max,
            step: slider.step || 1,
            profile: slider.dataset.rgbProfile || "",
            controlMode: slider.dataset.speedControlMode || SOFTWARE_CONTROL
        };
    }

    function storedToUiForSlider(slider, storedValue) {
        const options = sliderOptions(slider);
        const numericStoredValue = numberOr(storedValue, options.min);
        slider.dataset.storedSpeed = String(numericStoredValue);
        slider.dataset.speedEdited = "false";
        return storedToUi(
            numericStoredValue,
            options.min,
            options.max,
            options.step,
            options.profile,
            options.controlMode
        );
    }

    function markEdited(slider) {
        slider.dataset.speedEdited = "true";
    }

    function uiToStoredForSlider(slider, uiValue) {
        const options = sliderOptions(slider);
        if (slider.dataset.speedEdited !== "true" && slider.dataset.storedSpeed !== undefined) {
            return numberOr(slider.dataset.storedSpeed, options.min);
        }

        return uiToStored(
            uiValue,
            options.min,
            options.max,
            options.step,
            options.profile,
            options.controlMode
        );
    }

    function formatForSlider(slider, value) {
        return Number(value).toFixed(decimalPlaces(slider.step || 1));
    }

    return Object.freeze({
        HARDWARE_CONTROL,
        SOFTWARE_CONTROL,
        classificationForProfile,
        configureSlider,
        controlModeForProfiles,
        formatForSlider,
        hasSpeedControl,
        markEdited,
        normalizeProfile,
        rangeForProfile,
        storedToUi,
        storedToUiForSlider,
        uiToStored,
        uiToStoredForSlider,
        usesDurationSemantics
    });
}));
