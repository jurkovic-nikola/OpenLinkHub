"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const externalSources = require("./external-sources.js");

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

test("temperature UI has no arbitrary executable input", function () {
    const script = fs.readFileSync(path.join(__dirname, "temperature.js"), "utf8");
    assert.match(script, /url:\s*['"]\/api\/external-sources['"]/);
    assert.match(script, /selectionPayload\(\$\("#externalSourceId"\)\.val\(\)\)/);
    assert.doesNotMatch(script, /externalExecutable|binary-probeData|Path to binary/);

    const repositoryRoot = path.join(__dirname, "..", "..");
    for (const templateName of ["temperature.html", "temperatureGraph.html"]) {
        const template = fs.readFileSync(path.join(repositoryRoot, "web", templateName), "utf8");
        assert.match(template, /<option value="7">\{\{ \.Lang "txtExternalSource" \}\}<\/option>/);
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
