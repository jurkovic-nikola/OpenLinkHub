"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const repositoryRoot = path.resolve(__dirname, "../..");
const settingsScript = fs.readFileSync(path.join(__dirname, "settings.js"), "utf8");
const settingsTemplate = fs.readFileSync(path.join(repositoryRoot, "web/settings.html"), "utf8");
const languageDirectory = path.join(repositoryRoot, "database/language");

test("restore success keeps the server restart message prominent", function () {
    assert.match(settingsScript, /toast\.success\(response\)/);
    assert.match(settingsScript, /restoreRestartWarning/);
    assert.match(settingsTemplate, /id="restoreRestartWarning"[^>]*text-warning[^>]*fw-bold/);
    assert.match(settingsTemplate, /txtRestoreRestartWarning/);
});

test("restore errors and file selection use localized template messages", function () {
    assert.match(settingsTemplate, /data-select-message="\{\{ \.Lang "txtSelectBackupFile" \}\}"/);
    assert.match(settingsTemplate, /data-failure-prefix="\{\{ \.Lang "txtRestoreFailed" \}\}"/);
    assert.match(settingsScript, /restoreForm\.data\("select-message"\)/);
    assert.match(settingsScript, /restoreForm\.data\("failure-prefix"\)/);
});

test("settings identifies the repository-only backup guide without a moving branch link", function () {
    assert.match(settingsTemplate, /<code>docs\/backup-restore\.md<\/code>\./);
    assert.match(settingsTemplate, /txtRestoreGuide/);
    assert.doesNotMatch(settingsTemplate, /LumenForge-Dev\/docs\/backup-restore|main\/docs\/backup-restore/);
    assert.doesNotMatch(settingsTemplate, /href="[^"]*backup-restore\.md/);
    const english = JSON.parse(fs.readFileSync(path.join(languageDirectory, "en_US.json"), "utf8"));
    assert.equal(
        english.values.txtRestoreGuide,
        "Backup and restore guide: Available in the LumenForge GitHub repository under"
    );
    for (const filename of fs.readdirSync(languageDirectory).filter((name) => name.endsWith(".json"))) {
        const language = JSON.parse(fs.readFileSync(path.join(languageDirectory, filename), "utf8"));
        assert.match(language.values.txtRestoreGuide, /GitHub/, `${filename} identifies the GitHub repository`);
    }
    const restoreHandler = settingsScript.match(
        /\$\("#restoreForm"\)\.on\("submit"[\s\S]+?\n    \}\);/
    );
    assert.ok(restoreHandler, "restore submit handler is present");
    assert.doesNotMatch(restoreHandler[0], /systemctl|\/api\/restart|location\.reload/);
});
