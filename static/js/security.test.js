"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const security = require("./security.js");

test("mutation methods are classified centrally", function () {
    for (const method of ["POST", "PUT", "PATCH", "DELETE", "post"]) {
        assert.equal(security.isMutation(method), true, method);
    }
    for (const method of ["GET", "HEAD", "OPTIONS"]) {
        assert.equal(security.isMutation(method), false, method);
    }
});

test("request proof cookie is decoded", function () {
    assert.equal(
        security.readCookie("other=x; lumenforge_request_proof=a%2Bb%2Fc; final=y", "lumenforge_request_proof"),
        "a+b/c"
    );
    assert.equal(security.readCookie("other=x", "lumenforge_request_proof"), "");
});

test("content types preserve special mutation routes", function () {
    assert.equal(security.contentTypeForPath("/api/temperatures/update"), "application/x-www-form-urlencoded; charset=UTF-8");
    assert.equal(security.contentTypeForPath("/api/openrgbimport/discover"), "");
    assert.equal(security.contentTypeForPath("/api/openrgbimport/refresh"), "");
    assert.equal(security.contentTypeForPath("/api/media/play"), "");
    assert.equal(security.contentTypeForPath("/api/lcd/upload"), "application/json");
    assert.equal(security.contentTypeForPath("/api/color"), "application/json");
});

test("jQuery mutations receive proof and route-appropriate content types", function () {
    let prefilter;
    function jquery() {
        return {ajaxError: function () {}};
    }
    jquery.ajaxPrefilter = function (callback) {
        prefilter = callback;
    };
    class FakeFormData {}
    const browser = {
        URL,
        FormData: FakeFormData,
        console,
        document: {cookie: "lumenforge_request_proof=test-proof"},
        jQuery: jquery,
        location: {
            href: "http://localhost:28080/",
            origin: "http://localhost:28080"
        }
    };
    security.install(browser);

    const jsonOptions = {url: "/api/color", type: "POST"};
    prefilter(jsonOptions, {data: "{}"});
    assert.equal(jsonOptions.headers["X-LumenForge-Request-Proof"], "test-proof");
    assert.equal(jsonOptions.contentType, "application/json");

    const formOptions = {url: "/api/temperatures/update", type: "PUT"};
    prefilter(formOptions, {data: {profile: "test", data: "{}"}});
    assert.equal(formOptions.contentType, "application/x-www-form-urlencoded; charset=UTF-8");

    const multipartOptions = {url: "/api/lcd/upload", type: "POST", contentType: false};
    prefilter(multipartOptions, {data: new FakeFormData()});
    assert.equal(multipartOptions.contentType, false);

    const readOptions = {url: "/api/dashboard", type: "GET"};
    prefilter(readOptions, {});
    assert.equal(readOptions.headers, undefined);
});

test("native fetch mutations receive proof and JSON content type", async function () {
    let captured;
    const browser = {
        URL,
        Headers,
        FormData: class {},
        console,
        document: {cookie: "lumenforge_request_proof=fetch-proof"},
        location: {
            href: "http://127.0.0.1:28080/",
            origin: "http://127.0.0.1:28080"
        },
        fetch: async function (input, options) {
            captured = {input, options};
            return {status: 200, headers: new Headers()};
        }
    };
    security.install(browser);
    await browser.fetch("/api/color", {method: "POST", body: "{}"});

    assert.equal(captured.input, "/api/color");
    assert.equal(captured.options.headers.get("X-LumenForge-Request-Proof"), "fetch-proof");
    assert.equal(captured.options.headers.get("Content-Type"), "application/json");
});

test("empty-body and media callers use their protected mutation forms", function () {
    const settings = fs.readFileSync(path.join(__dirname, "settings.js"), "utf8");
    for (const endpoint of ["discover", "refresh"]) {
        const request = settings.match(new RegExp(
            `url: '/api/openrgbimport/${endpoint}',[\\s\\S]*?dataType: 'json'`
        ));
        assert.ok(request, endpoint);
        assert.doesNotMatch(request[0], /\bdata:|\bcontentType:/, endpoint);
    }

    const xeneon = fs.readFileSync(path.join(__dirname, "xeneon.js"), "utf8");
    assert.match(xeneon, /url:\s*urlAction,\s*method:\s*'POST'/);
});
