"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const repositoryRoot = path.resolve(__dirname, "../..");
const settingsScript = fs.readFileSync(path.join(__dirname, "settings.js"), "utf8");
const settingsTemplate = fs.readFileSync(path.join(repositoryRoot, "web/settings.html"), "utf8");

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

test("settings links to the focused restore guide without automatic restart behavior", function () {
    assert.match(settingsTemplate, /docs\/backup-restore\.md/);
    const restoreHandler = settingsScript.match(
        /\$\("#restoreForm"\)\.on\("submit"[\s\S]+?\n    \}\);/
    );
    assert.ok(restoreHandler, "restore submit handler is present");
    assert.doesNotMatch(restoreHandler[0], /systemctl|\/api\/restart|location\.reload/);
});
