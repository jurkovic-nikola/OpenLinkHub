"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const externalSources = require("./external-sources.js");

const configuredSources = [
    {id: "gpu-temperature", name: "GPU Temperature"}
];

test("external source dropdown uses registry ids and names", function () {
    assert.deepEqual(
        externalSources.dropdownOptions([
            {id: "gpu-temperature", name: "GPU Temperature"},
            {id: "ambient", name: "Ambient Probe"}
        ]),
        [
            {value: "gpu-temperature", label: "GPU Temperature", disabled: false},
            {value: "ambient", label: "Ambient Probe", disabled: false}
        ]
    );
});

test("external source selection submits only the opaque id", function () {
    assert.deepEqual(
        externalSources.selectionPayload("gpu-temperature"),
        {externalSourceId: "gpu-temperature"}
    );
    assert.equal(externalSources.selectionPayload(""), null);
});

test("empty registry produces a disabled empty-state option", function () {
    assert.deepEqual(
        externalSources.dropdownOptions([], "Nothing configured"),
        [{value: "", label: "Nothing configured", disabled: true}]
    );
});

test("registry with only malformed entries produces a disabled empty-state option", function () {
    assert.deepEqual(
        externalSources.dropdownOptions([
            null,
            {},
            {id: "", name: "Missing ID"},
            {id: "missing-name", name: ""}
        ], "Nothing configured"),
        [{value: "", label: "Nothing configured", disabled: true}]
    );
});

test("external source sensor displays the registry selector", function () {
    const visibility = externalSources.sensorControlVisibility("7");
    const state = externalSources.externalSourceControlState("7", configuredSources);

    assert.equal(visibility.externalSource, true);
    assert.equal(visibility.storage, false);
    assert.equal(visibility.hwmon, false);
    assert.equal(state.visible, true);
    assert.deepEqual(
        state.options,
        [{value: "gpu-temperature", label: "GPU Temperature", disabled: false}]
    );
});

test("CPU sensor does not display the external source selector", function () {
    const visibility = externalSources.sensorControlVisibility("0");
    const state = externalSources.externalSourceControlState("0", configuredSources);

    assert.equal(visibility.externalSource, false);
    assert.equal(state.visible, false);
    assert.deepEqual(state.options, []);
});

test("storage sensor displays only its storage selector", function () {
    const visibility = externalSources.sensorControlVisibility("3");

    assert.equal(visibility.storage, true);
    assert.equal(visibility.externalSource, false);
    assert.equal(visibility.hwmon, false);
});

test("external HWMON sensor displays only its HWMON selector", function () {
    const visibility = externalSources.sensorControlVisibility("6");

    assert.equal(visibility.hwmon, true);
    assert.equal(visibility.externalSource, false);
    assert.equal(visibility.storage, false);
});

test("switching away hides and switching back restores the registry selector", function () {
    const externalState = externalSources.externalSourceControlState("7", configuredSources);
    const cpuState = externalSources.externalSourceControlState("0", configuredSources);
    const restoredState = externalSources.externalSourceControlState("7", configuredSources);

    assert.equal(externalState.visible, true);
    assert.equal(cpuState.visible, false);
    assert.deepEqual(cpuState.options, []);
    assert.deepEqual(restoredState, externalState);
});

test("temperature UI has no arbitrary executable input", function () {
    const script = fs.readFileSync(path.join(__dirname, "temperature.js"), "utf8");
    assert.match(script, /url:\s*['"]\/api\/external-sources['"]/);
    assert.match(script, /selectionPayload\(\$\("#externalSourceId"\)\.val\(\)\)/);
    assert.match(script, /updateSensorControls\(\$\("#sensor"\)\.val\(\)\)/);
    assert.match(script, /externalSourceControlState\(/);
    assert.match(script, /\$\("#externalSourceId"\)\.empty\(\)\.val\(""\)\.prop\("disabled", true\)/);
    assert.doesNotMatch(script, /externalExecutable|binary-probeData|Path to binary/);

    const repositoryRoot = path.join(__dirname, "..", "..");
    for (const templateName of ["temperature.html", "temperatureGraph.html"]) {
        const template = fs.readFileSync(path.join(repositoryRoot, "web", templateName), "utf8");
        assert.match(template, /<option value="7">\{\{ \.Lang "txtExternalSource" \}\}<\/option>/);
        assert.match(template, /id="external-source-data" data-sensor-type="7" hidden/);
        assert.match(template, /<select[^>]+id="externalSourceId"[^>]+aria-label="\{\{ \.Lang "txtExternalSource" \}\}"/);
        assert.doesNotMatch(template, /External binary|binary-probeData|Path to binary|type="text"[^>]+external/i);
    }

    assert.match(script, /i18n\.t\('txtExternalSource'\)/);
    assert.match(script, /i18n\.t\('txtSensorExternalSourceInfo'\)/);
    assert.doesNotMatch(script, /External binary|txtSensorExternalBinaryInfo/);

    const english = JSON.parse(fs.readFileSync(
        path.join(repositoryRoot, "database", "language", "en_US.json"),
        "utf8"
    ));
    assert.equal(english.values.txtExternalSource, "External source");
    assert.match(english.values.txtSensorExternalSourceInfo, /external source/i);

    const styles = fs.readFileSync(
        path.join(repositoryRoot, "static", "css", "themes", "default.css"),
        "utf8"
    );
    assert.match(styles, /#external-source-data\s*\{\s*display:\s*none;/);
    assert.doesNotMatch(styles, /#binary-sensors-probe-data/);
});

test("saved external source profiles use current localized terminology", function () {
    const repositoryRoot = path.join(__dirname, "..", "..");
    const obsoleteVisibleWording = /External (?:Executable|Binary)/i;

    for (const templateName of ["temperature.html", "temperatureGraph.html"]) {
        const template = fs.readFileSync(path.join(repositoryRoot, "web", templateName), "utf8");
        assert.match(
            template,
            /\{\{ if eq \$value\.Sensor 7 \}\}\{\{ \$root\.Lang "txtExternalSource" \}\}\{\{ else \}\}\{\{ \$value\.SensorString \}\}\{\{ end \}\}/
        );
        assert.doesNotMatch(template, obsoleteVisibleWording);
    }

    const temperatures = fs.readFileSync(
        path.join(repositoryRoot, "src", "temperatures", "temperatures.go"),
        "utf8"
    );
    assert.match(temperatures, /SensorTypeExternalExecutable:\s+"External Source"/);
    assert.doesNotMatch(temperatures, /"External Executable"/);

    const languageDirectory = path.join(repositoryRoot, "database", "language");
    for (const fileName of fs.readdirSync(languageDirectory).filter((name) => name.endsWith(".json"))) {
        const language = JSON.parse(
            fs.readFileSync(path.join(languageDirectory, fileName), "utf8")
        );
        assert.equal(typeof language.values.txtExternalSource, "string", fileName);
        assert.notEqual(language.values.txtExternalSource.trim(), "", fileName);
        assert.doesNotMatch(language.values.txtExternalSource, obsoleteVisibleWording, fileName);
    }
});
