(function (root, factory) {
    "use strict";
    const api = factory();
    if (typeof module === "object" && module.exports) {
        module.exports = api;
    } else {
        root.ExternalSourcesUI = api;
    }
}(typeof globalThis !== "undefined" ? globalThis : this, function () {
    "use strict";

    function dropdownOptions(sources, emptyLabel) {
        const options = Array.isArray(sources) ? sources
            .filter(function (source) {
                return source &&
                    typeof source.id === "string" &&
                    source.id.length > 0 &&
                    typeof source.name === "string" &&
                    source.name.length > 0;
            })
            .map(function (source) {
                return {value: source.id, label: source.name, disabled: false};
            }) : [];

        if (options.length === 0) {
            return [{
                value: "",
                label: emptyLabel || "No external sources are configured",
                disabled: true
            }];
        }

        return options;
    }

    function selectionPayload(selectedID) {
        if (typeof selectedID !== "string" || selectedID.length === 0) {
            return null;
        }
        return {externalSourceId: selectedID};
    }

    return {
        dropdownOptions: dropdownOptions,
        selectionPayload: selectionPayload
    };
}));
