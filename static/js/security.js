"use strict";

(function (root, factory) {
    const api = factory();
    if (typeof module === "object" && module.exports) {
        module.exports = api;
    }
    if (root && root.document) {
        api.install(root);
    }
})(typeof window === "undefined" ? null : window, function () {
    const cookieName = "lumenforge_request_proof";
    const proofHeader = "X-LumenForge-Request-Proof";
    const proofFailureHeader = "X-LumenForge-Request-Proof-Failure";
    const mutationMethods = new Set(["POST", "PUT", "PATCH", "DELETE"]);

    function readCookie(cookieString, name) {
        const prefix = encodeURIComponent(name) + "=";
        for (const part of String(cookieString || "").split(";")) {
            const cookie = part.trim();
            if (cookie.startsWith(prefix)) {
                return decodeURIComponent(cookie.slice(prefix.length));
            }
        }
        return "";
    }

    function isMutation(method) {
        return mutationMethods.has(String(method || "GET").toUpperCase());
    }

    function contentTypeForPath(pathname) {
        if (pathname === "/api/temperatures/update") {
            return "application/x-www-form-urlencoded; charset=UTF-8";
        }
        if (pathname === "/api/openrgbimport/discover" ||
            pathname === "/api/openrgbimport/refresh" ||
            pathname.startsWith("/api/media/")) {
            return "";
        }
        return "application/json";
    }

    function install(browser) {
        const jquery = browser.jQuery;
        const proof = function () {
            return readCookie(browser.document.cookie, cookieName);
        };
        let proofWarningShown = false;

        function isSameOrigin(rawURL) {
            return new browser.URL(rawURL, browser.location.href).origin === browser.location.origin;
        }

        function warnProofFailure() {
            if (proofWarningShown) return;
            proofWarningShown = true;
            const message = "LumenForge rejected the local request proof. Reload the dashboard and try again.";
            if (browser.toast && typeof browser.toast.warning === "function") {
                browser.toast.warning(message);
            } else {
                browser.console.error(message);
            }
        }

        if (jquery) {
            jquery.ajaxPrefilter(function (options, originalOptions) {
                const method = String(options.type || options.method || "GET").toUpperCase();
                if (!isMutation(method) || !isSameOrigin(options.url || browser.location.href)) return;

                options.headers = Object.assign({}, options.headers, {[proofHeader]: proof()});
                const path = new browser.URL(options.url, browser.location.href).pathname;
                const expectedContentType = contentTypeForPath(path);
                const formData = browser.FormData && originalOptions.data instanceof browser.FormData;
                if (expectedContentType && !formData && options.contentType !== false) {
                    options.contentType = expectedContentType;
                }
            });

            jquery(browser.document).ajaxError(function (_, xhr) {
                if (xhr.status === 403 && xhr.getResponseHeader(proofFailureHeader)) {
                    warnProofFailure();
                }
            });
        }

        if (typeof browser.fetch === "function") {
            const nativeFetch = browser.fetch.bind(browser);
            browser.fetch = async function (input, init) {
                const options = Object.assign({}, init);
                const rawURL = typeof input === "string" || input instanceof browser.URL ? input : input.url;
                const method = String(options.method || (input && input.method) || "GET").toUpperCase();
                if (isMutation(method) && isSameOrigin(rawURL)) {
                    const headers = new browser.Headers(options.headers || (input && input.headers) || {});
                    headers.set(proofHeader, proof());
                    const expectedContentType = contentTypeForPath(new browser.URL(rawURL, browser.location.href).pathname);
                    const formData = browser.FormData && options.body instanceof browser.FormData;
                    if (expectedContentType && options.body != null && !formData && !headers.has("Content-Type")) {
                        headers.set("Content-Type", expectedContentType);
                    }
                    options.headers = headers;
                }
                const response = await nativeFetch(input, options);
                if (response.status === 403 && response.headers.get(proofFailureHeader)) {
                    warnProofFailure();
                }
                return response;
            };
        }
    }

    return {contentTypeForPath, install, isMutation, readCookie};
});
