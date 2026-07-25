"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const speed = require("./rgb-speed.js");

const durationProfiles = [
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
];
const identityProfiles = [
    "aurora",
    "cyberpunkglitch",
    "flame",
    "tokyonight"
];
const noSpeedProfiles = [
    "cpu-temperature",
    "gpu-temperature",
    "liquid-temperature",
    "probe-temperature",
    "static",
    "off"
];
const discreteHardwareProfiles = [
    "tlk",
    "tlr",
    "spiralrainbow",
    "rainbowwave",
    "rain",
    "visor",
    "colorwave"
];

function slider() {
    return {
        min: "",
        max: "",
        step: "",
        value: "",
        dataset: {}
    };
}

test("every software-rendered animation has an explicit audited classification", function () {
    for (const profile of durationProfiles) {
        assert.equal(
            speed.classificationForProfile(profile, speed.SOFTWARE_CONTROL),
            "duration",
            profile
        );
        assert.equal(speed.usesDurationSemantics(profile, speed.SOFTWARE_CONTROL), true);
        assert.deepEqual(
            speed.rangeForProfile(profile, speed.SOFTWARE_CONTROL),
            {min: 1, max: 10, step: 0.1},
            profile
        );
    }

    for (const profile of identityProfiles) {
        assert.equal(
            speed.classificationForProfile(profile, speed.SOFTWARE_CONTROL),
            "identity",
            profile
        );
        assert.equal(speed.usesDurationSemantics(profile, speed.SOFTWARE_CONTROL), false);
        assert.deepEqual(
            speed.rangeForProfile(profile, speed.SOFTWARE_CONTROL),
            {min: 1, max: 10, step: 0.1},
            profile
        );
    }

    for (const profile of noSpeedProfiles) {
        assert.equal(
            speed.classificationForProfile(profile, speed.SOFTWARE_CONTROL),
            "none",
            profile
        );
        assert.equal(speed.hasSpeedControl(profile, speed.SOFTWARE_CONTROL), false);
    }
});

test("only renderer-backed speed controls are exposed", function () {
    for (const profile of noSpeedProfiles) {
        assert.equal(
            speed.hasSpeedControl(profile, speed.SOFTWARE_CONTROL),
            false,
            profile
        );
    }
    assert.equal(speed.hasSpeedControl("", speed.SOFTWARE_CONTROL), false);

    for (const profile of ["storm", "circle"]) {
        assert.equal(
            speed.hasSpeedControl(profile, speed.SOFTWARE_CONTROL),
            true,
            profile
        );
    }
    assert.equal(
        speed.classificationForProfile("storm", speed.SOFTWARE_CONTROL),
        "duration"
    );
});

test("normalized keys, aliases, and display names select the same mapping", function () {
    const aliases = new Map([
        ["Color Pulse", "colorpulse"],
        ["color-pulse", "colorpulse"],
        ["COLOR_PULSE", "colorpulse"],
        ["Cyberpunk Glitch", "cyberpunkglitch"],
        ["cyberpunk-glitch", "cyberpunkglitch"],
        ["Tokyo Night", "tokyonight"],
        ["tokyo-night", "tokyonight"],
        ["Circle Shift", "circleshift"],
        ["pastel-spiral-rainbow", "pastelspiralrainbow"],
        ["CPU Temperature", "cputemperature"]
    ]);

    for (const [alias, normalized] of aliases) {
        assert.equal(speed.normalizeProfile(alias), normalized, alias);
        assert.equal(
            speed.classificationForProfile(alias, speed.SOFTWARE_CONTROL),
            speed.classificationForProfile(normalized, speed.SOFTWARE_CONTROL),
            alias
        );
    }
});

test("duration mappings put slow on the left and fast on the right", function () {
    for (const profile of [
        "circle",
        "arc",
        "comet",
        "colorshift",
        "colorpulse",
        "colorwarp",
        "flickering"
    ]) {
        assert.equal(speed.uiToStored(1, 1, 10, 0.1, profile, "software"), 10, profile);
        assert.equal(speed.uiToStored(10, 1, 10, 0.1, profile, "software"), 1, profile);
        assert.equal(speed.storedToUi(10, 1, 10, 0.1, profile, "software"), 1, profile);
        assert.equal(speed.storedToUi(1, 1, 10, 0.1, profile, "software"), 10, profile);
    }
});

test("identity mappings increase the stored multiplier toward fast", function () {
    for (const profile of ["aurora", "tokyonight"]) {
        assert.equal(speed.uiToStored(1, 1, 10, 0.1, profile, "software"), 1, profile);
        assert.equal(speed.uiToStored(10, 1, 10, 0.1, profile, "software"), 10, profile);
        assert.equal(speed.storedToUi(4.1, 1, 10, 0.1, profile, "software"), 4.1, profile);
    }
});

test("Flame uses a calibrated 0.1 to 1.5 multiplier range", function () {
    assert.equal(speed.uiToStored(1, 1, 10, 0.1, "flame", "software"), 0.1);
    assert.equal(speed.uiToStored(5.5, 1, 10, 0.1, "flame", "software"), 0.8);
    assert.equal(speed.uiToStored(10, 1, 10, 0.1, "flame", "software"), 1.5);
    assert.equal(speed.storedToUi(0.1, 1, 10, 0.1, "flame", "software"), 1);
    assert.equal(speed.storedToUi(0.8, 1, 10, 0.1, "flame", "software"), 5.5);
    assert.equal(speed.storedToUi(1.5, 1, 10, 0.1, "flame", "software"), 10);
});

test("Cyberpunk Glitch uses a calibrated 0.1 to 1.0 multiplier range", function () {
    assert.equal(speed.uiToStored(1, 1, 10, 0.1, "cyberpunkglitch", "software"), 0.1);
    assert.equal(speed.uiToStored(5.5, 1, 10, 0.1, "cyberpunkglitch", "software"), 0.55);
    assert.equal(speed.uiToStored(10, 1, 10, 0.1, "cyberpunkglitch", "software"), 1);
    assert.equal(speed.storedToUi(0.1, 1, 10, 0.1, "cyberpunkglitch", "software"), 1);
    assert.equal(speed.storedToUi(0.55, 1, 10, 0.1, "cyberpunkglitch", "software"), 5.5);
    assert.equal(speed.storedToUi(1, 1, 10, 0.1, "cyberpunkglitch", "software"), 10);
});

test("software Rain is continuous while preserving its three original timing points", function () {
    assert.deepEqual(
        speed.rangeForProfile("rain", speed.SOFTWARE_CONTROL),
        {min: 1, max: 10, step: 0.1}
    );
    assert.equal(speed.uiToStored(1, 1, 10, 0.1, "rain", "software"), 3);
    assert.equal(speed.uiToStored(5.5, 1, 10, 0.1, "rain", "software"), 2);
    assert.equal(speed.uiToStored(10, 1, 10, 0.1, "rain", "software"), 1);
    assert.equal(speed.uiToStored(3.2, 1, 10, 0.1, "rain", "software"), 2.511);
});

test("only verified firmware modes use the discrete hardware range", function () {
    for (const profile of discreteHardwareProfiles) {
        assert.equal(
            speed.classificationForProfile(profile, speed.HARDWARE_CONTROL),
            "hardware-discrete",
            profile
        );
        assert.deepEqual(
            speed.rangeForProfile(profile, speed.HARDWARE_CONTROL),
            {min: 1, max: 3, step: 1},
            profile
        );
        assert.equal(speed.uiToStored(1, 1, 3, 1, profile, "hardware"), 3, profile);
        assert.equal(speed.uiToStored(3, 1, 3, 1, profile, "hardware"), 1, profile);
    }

    assert.deepEqual(
        speed.rangeForProfile("spiralrainbow", speed.SOFTWARE_CONTROL),
        {min: 1, max: 10, step: 0.1}
    );
    assert.deepEqual(
        speed.rangeForProfile("visor", speed.SOFTWARE_CONTROL),
        {min: 1, max: 10, step: 0.1}
    );
});

test("the editor derives hardware context from hardware-only profile keys", function () {
    assert.equal(
        speed.controlModeForProfiles({circle: {}, rainbow: {}, visor: {}}),
        speed.SOFTWARE_CONTROL
    );
    assert.equal(
        speed.controlModeForProfiles({rain: {}, tlk: {}, colorwave: {}}),
        speed.HARDWARE_CONTROL
    );
    assert.equal(
        speed.controlModeForProfiles({"Type Lighting - Key": {}, rain: {}}),
        speed.HARDWARE_CONTROL
    );
});

test("unchanged special-range values retain their exact stored timing", function () {
    const control = slider();
    speed.configureSlider(control, "flame", speed.SOFTWARE_CONTROL);

    const displayed = speed.storedToUiForSlider(control, 4.1);
    assert.equal(displayed, 10);
    assert.equal(speed.uiToStoredForSlider(control, displayed), 4.1);

    speed.markEdited(control);
    assert.equal(speed.uiToStoredForSlider(control, displayed), 1.5);
});

test("reflected and calibrated values round-trip after editing", function () {
    const circleUi = speed.storedToUi(5.2, 1, 10, 0.1, "circle", "software");
    assert.equal(circleUi, 5.8);
    assert.equal(speed.uiToStored(circleUi, 1, 10, 0.1, "circle", "software"), 5.2);

    const flameStored = speed.uiToStored(7.3, 1, 10, 0.1, "flame", "software");
    assert.equal(
        speed.storedToUi(flameStored, 1, 10, 0.1, "flame", "software"),
        7.3
    );
});

test("switching effects resets range, step, context, and preservation state", function () {
    const control = slider();

    speed.configureSlider(control, "rain", speed.HARDWARE_CONTROL);
    speed.storedToUiForSlider(control, 2);
    speed.markEdited(control);
    assert.equal(control.min, "1");
    assert.equal(control.max, "3");
    assert.equal(control.step, "1");
    assert.equal(control.dataset.speedEdited, "true");

    speed.configureSlider(control, "flame", speed.SOFTWARE_CONTROL);
    assert.equal(control.min, "1");
    assert.equal(control.max, "10");
    assert.equal(control.step, "0.1");
    assert.equal(control.dataset.rgbProfile, "flame");
    assert.equal(control.dataset.speedControlMode, "software");
    assert.equal(control.dataset.speedEdited, "false");
    assert.equal(control.dataset.storedSpeed, undefined);
});

test("switching Static to Circle restores a continuous enabled speed control", function () {
    const control = slider();
    control.disabled = true;

    assert.equal(speed.hasSpeedControl("static", speed.SOFTWARE_CONTROL), false);

    speed.configureSlider(control, "circle", speed.SOFTWARE_CONTROL);
    control.disabled = !speed.hasSpeedControl("circle", speed.SOFTWARE_CONTROL);
    control.value = String(speed.storedToUiForSlider(control, 4));

    assert.equal(control.disabled, false);
    assert.equal(control.min, "1");
    assert.equal(control.max, "10");
    assert.equal(control.step, "0.1");
    assert.equal(control.value, "7");
    assert.equal(control.dataset.rgbProfile, "circle");
    assert.equal(control.dataset.speedControlMode, "software");
});

test("switching Circle to Static removes the complete speed-control state", function () {
    const control = slider();
    speed.configureSlider(control, "circle", speed.SOFTWARE_CONTROL);
    control.value = "7";

    const visible = speed.hasSpeedControl("static", speed.SOFTWARE_CONTROL);
    const renderedControl = visible ? {
        label: "Speed",
        icons: ["Slow", "Fast"],
        value: control.value
    } : null;

    assert.equal(visible, false);
    assert.equal(renderedControl, null);
});

test("cluster and individual editors share icon-only endpoint semantics", function () {
    const sources = {
        rgb: fs.readFileSync(path.join(__dirname, "rgb.js"), "utf8"),
        cluster: fs.readFileSync(path.join(__dirname, "cluster.js"), "utf8"),
        overview: fs.readFileSync(path.join(__dirname, "overview.js"), "utf8"),
        clusterTemplate: fs.readFileSync(path.join(__dirname, "..", "..", "web", "cluster.html"), "utf8"),
        rgbTemplate: fs.readFileSync(path.join(__dirname, "..", "..", "web", "rgb.html"), "utf8"),
        head: fs.readFileSync(path.join(__dirname, "..", "..", "web", "head.html"), "utf8")
    };

    assert.match(sources.head, /\/static\/js\/rgb-speed\.js\?v=3/);
    assert.match(sources.rgbTemplate, /\/static\/js\/rgb\.js\?v=3/);
    assert.match(sources.clusterTemplate, /\/static\/js\/cluster\.js\?v=3/);
    assert.match(sources.rgb, /controlModeForProfiles\(\s*response\.data\.profiles/);
    assert.match(sources.rgb, /storedToUiForSlider/);
    assert.match(sources.rgb, /uiToStoredForSlider/);
    assert.match(sources.cluster, /storedToUiForSlider/);
    assert.match(sources.cluster, /uiToStoredForSlider/);
    assert.equal((sources.overview.match(/storedToUiForSlider/g) || []).length, 2);

    for (const source of [sources.rgb, sources.overview, sources.clusterTemplate]) {
        assert.match(
            source,
            /title="Slow" aria-label="Slow"[\s\S]*icon-slow\.svg[\s\S]*type="range"[\s\S]*title="Fast" aria-label="Fast"[\s\S]*icon-fast\.svg/
        );
        assert.match(source, /aria-label="Animation speed from Slow to Fast"/);
        assert.doesNotMatch(source, /\/> Slow/);
        assert.doesNotMatch(source, /Fast <img/);
    }
});

test("speed endpoints share one centered flex row in slow-range-fast order", function () {
    const sources = {
        rgb: fs.readFileSync(path.join(__dirname, "rgb.js"), "utf8"),
        overview: fs.readFileSync(path.join(__dirname, "overview.js"), "utf8"),
        cluster: fs.readFileSync(
            path.join(__dirname, "..", "..", "web", "cluster.html"),
            "utf8"
        )
    };
    const orderedFlexRow = /class="system-slider[^"]*d-flex[^"]*align-items-center[^"]*"[\s\S]*?<span[^>]*title="Slow" aria-label="Slow">[\s\S]*?icon-slow\.svg[\s\S]*?<\/span>\s*<label[^>]*>[\s\S]*?<input[^>]*type="range"[\s\S]*?<\/label>\s*<span[^>]*title="Fast" aria-label="Fast">[\s\S]*?icon-fast\.svg[\s\S]*?<\/span>\s*<\/div>/g;

    assert.equal((sources.rgb.match(orderedFlexRow) || []).length, 1);
    assert.equal((sources.overview.match(orderedFlexRow) || []).length, 2);
    assert.equal((sources.cluster.match(orderedFlexRow) || []).length, 1);

    for (const source of Object.values(sources)) {
        assert.match(source, /class="flex-grow-1 m-0" style="min-width: 0;"/);
        assert.match(source, /class="text-nowrap d-inline-flex align-items-center"/);
    }
});

test("all visible-control payload paths use the shared stored conversion", function () {
    const rgb = fs.readFileSync(path.join(__dirname, "rgb.js"), "utf8");
    const cluster = fs.readFileSync(path.join(__dirname, "cluster.js"), "utf8");
    const overview = fs.readFileSync(path.join(__dirname, "overview.js"), "utf8");

    assert.match(rgb, /if \(speedSlider\) \{[\s\S]*pf\["speed"\][\s\S]*uiToStoredForSlider/);
    assert.match(cluster, /pf\["speed"\] = storedDuration/);
    assert.equal(
        (overview.match(/if \(\$speedSlider\.length\) \{[\s\S]*?pf\["speed"\] = LumenForgeRgbSpeed\.uiToStoredForSlider/g) || []).length,
        2
    );
});

test("cluster, individual, and overview controls share capability visibility", function () {
    const rgb = fs.readFileSync(path.join(__dirname, "rgb.js"), "utf8");
    const cluster = fs.readFileSync(path.join(__dirname, "cluster.js"), "utf8");
    const overview = fs.readFileSync(path.join(__dirname, "overview.js"), "utf8");
    const clusterTemplate = fs.readFileSync(
        path.join(__dirname, "..", "..", "web", "cluster.html"),
        "utf8"
    );

    assert.match(rgb, /const hasSpeedControl = LumenForgeRgbSpeed\.hasSpeedControl/);
    assert.match(rgb, /const speedRowHtml = hasSpeedControl \?/);
    assert.match(rgb, /\$\{speedRowHtml\}/);
    assert.doesNotMatch(rgb, /speedDisabled/);

    assert.match(cluster, /function updateSpeedControlVisibility\(profile\)/);
    assert.match(cluster, /LumenForgeRgbSpeed\.hasSpeedControl/);
    assert.match(cluster, /\$speedControlGroup\.prop\("hidden", !visible\)/);
    assert.match(cluster, /\.addClass\(visible \? "col-md-4" : "col-md-6"\)/);
    assert.match(cluster, /\$speedSlider\.prop\("disabled", !visible\)/);
    assert.match(clusterTemplate, /id="clusterSpeedControlGroup" hidden/);
    assert.doesNotMatch(clusterTemplate, /\$speedDisabled/);

    assert.equal(
        (overview.match(/const speedRowHtml = LumenForgeRgbSpeed\.hasSpeedControl/g) || []).length,
        2
    );
});
