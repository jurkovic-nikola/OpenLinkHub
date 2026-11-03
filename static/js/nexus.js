"use strict";
$(document).ready(function () {
    const toastKey = "nexus.screenMessage." + $("#deviceId").val();
    const savedMessage = sessionStorage.getItem(toastKey);

    if (savedMessage) {
        sessionStorage.removeItem(toastKey);
        toast.success(savedMessage);
    }

    function reloadWithToast(message) {
        sessionStorage.setItem(toastKey, message);
        window.location.reload();
    }

    const catalog = JSON.parse($("#screenCatalog").text());
    const targets = {
        macro: "#targetMacro",
        media: "#targetMedia",
        profile: "#targetProfile",
        screen: "#targetScreen"
    };

    let draft;
    let track = "—";
    let fetchingTrack = false;
    let index = 0;
    let id;

    function selectValue(selector, value) {
        const field = $(selector);
        field.find("option[data-missing]").remove();
        field.val(value);
        if (value && field.val() === null) {
            $("<option>").attr("data-missing", "1").val(value).text(value).appendTo(field);
            field.val(value);
        }
    }

    function preview() {
        const strip = $("#tilePreview").empty();
        draft.tiles.forEach(function (tile, position) {
            let text = tile.label;

            if (tile.type === "clock") {
                text += " " + new Date().toLocaleTimeString([], {hour: "2-digit", minute: "2-digit"});
            } else if (tile.type === "track") {
                text += " " + track;
            } else if (tile.type === "sensor" || tile.type === "battery") {
                const source = $(tile.type === "battery" ? "#tileBattery option" : "#tileSource option").filter(function () { return this.value === tile.source; });
                text += " " + (source.attr("data-reading") || "—");
            }

            $("<button>").attr({type: "button", "aria-pressed": position === index})
                .addClass("system-button").css({flex: tile.width + " 1 0", minWidth: 0, overflow: "hidden", padding: 0,
                background: position === index ? "#343435" : ""})
                .text(text.trim() || String(position + 1)).appendTo(strip).on("click", function () {
                index = position;
                edit();
                strip.children().eq(index).trigger("focus");
            });
        });

        $("#tileLeft").prop("disabled", index === 0);
        $("#tileRight").prop("disabled", index === draft.tiles.length - 1);
        $("#tileRemove").prop("disabled", draft.tiles.length === 1);
        $("#tileAdd").prop("disabled", draft.tiles.length >= 6);
    }

    function refreshTrack() {
        if (fetchingTrack || document.hidden || !draft.tiles.some(function (tile) { return tile.type === "track"; })) {
            return;
        }

        fetchingTrack = true;
        $.getJSON("/api/media/playback").done(function (response) {
            const player = response.data;
            track = "—";
            if (response.status === 1 && player && player.title) {
                track = player.title;
                if (player.artists && player.artists.length) {
                    track += " · " + player.artists.join(", ");
                }
            }

            $("#tilePreview button").each(function (position) {
                const tile = draft.tiles[position];
                if (tile.type === "track") {
                    $(this).text((tile.label + " " + track).trim());
                }
            });
        }).always(function () {
            fetchingTrack = false;
        });
    }

    function edit() {
        const tile = draft.tiles[index];
        $("#tileType").val(tile.type);
        $("#tileLabel").val(tile.label);
        $("#tileWidth").val(tile.width);

        selectValue("#tileSource", tile.type === "sensor" ? tile.source : "cpu");
        if (tile.type === "battery" && tile.source) {
            selectValue("#tileBattery", tile.source);
        }

        $("#tileBattery").closest("label").toggle(tile.type === "battery");
        $("#tileSource").closest("label").toggle(tile.type === "sensor");
        $("#tileAction").val(tile.action || "");

        Object.keys(targets).forEach(function (action) {
            const selector = targets[action];
            $(selector).closest("label").toggle(action === tile.action);
            if (action === tile.action) {
                selectValue(selector, tile.target);
            }
        });

        preview();
    }

    function load() {
        const key = $("#screenSource").val();
        draft = JSON.parse(JSON.stringify(catalog[key]));

        id = key;
        $("#screenDelete").toggle(id.startsWith("custom-"));
        $("#screenName").val(draft.name);
        index = 0;

        edit();
    }

    let activeMode = $("#lcdProfile").val();
    $("#lcdProfile").on("change", function () {
        const select = $(this);
        const mode = select.val();
        select.prop("disabled", true);
        $.ajax({
            url: "/api/lcd/profile",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({deviceId: $("#deviceId").val(), profile: mode})
        }).done(function (response) {
            if (response.status === 1) {
                activeMode = mode;
                if (catalog[mode]) {
                    $("#screenSource").val(mode);
                    load();
                }
                toast.success(response.message);
            } else {
                toast.warning(response.message);
            }
        }).fail(function () {
            toast.warning(i18n.t("txtUnableToChangeLcdProfile"));
        }).always(function () {
            select.val(activeMode).prop("disabled", false);
        });
    });

    $("#screenDelete").on("click", function () {
        if (!id.startsWith("custom-")) {
            return;
        }

        const button = $(this).prop("disabled", true);
        $.ajax({
            url: "/api/nexus/screen/delete",
            type: "DELETE",
            contentType: "application/json",
            data: JSON.stringify({deviceId: $("#deviceId").val(), id: id})
        }).done(function (response) {
            if (response.status === 1) {
                reloadWithToast(response.message);
            } else {
                toast.warning(response.message);
            }
        }).fail(function () {
            toast.warning(i18n.t("txtUnableToChangeLcdProfile"));
        }).always(function () {
            button.prop("disabled", false);
        });
    });

    $("#screenSource").on("change", load);
    $("#tileType, #tileWidth, #tileAction, #tileSource, #targetMedia, #targetScreen, #targetProfile, #tileBattery, #targetMacro").on("change", function () {
        const tile = draft.tiles[index];

        tile.type = $("#tileType").val();
        tile.width = Number($("#tileWidth").val());
        tile.source = tile.type === "sensor" ? $("#tileSource").val() || "cpu" : "";
        if (tile.type === "battery") {
            tile.source = $("#tileBattery").val() || "";
        }
        tile.action = $("#tileAction").val();
        tile.target = targets[tile.action] ? $(targets[tile.action]).val() || "" : "";

        edit();
    });

    $("#tileLabel").on("input", function () {
        draft.tiles[index].label = this.value;
        preview();
    });

    $("#tileAdd").on("click", function () {
        draft.tiles.push({type: "clock", width: 1, label: ""});
        index = draft.tiles.length - 1;
        edit();
    });

    $("#tileRemove").on("click", function () {
        draft.tiles.splice(index, 1);
        index = Math.min(index, draft.tiles.length - 1);
        edit();
    });

    $("#tileLeft, #tileRight").on("click", function () {
        const tile = draft.tiles.splice(index, 1)[0];
        index += this.id === "tileLeft" ? -1 : 1;
        draft.tiles.splice(index, 0, tile);
        edit();
    });

    $("#screenSave, #screenCopy").on("click", function () {
        draft.name = $("#screenName").val().trim();
        if (!draft.name) {
            $("#screenName").trigger("focus");
            return;
        }
        const copy = this.id === "screenCopy";
        if (copy && Object.values(catalog).some(function (screen) {
            return screen.name.trim().toLowerCase() === draft.name.toLowerCase();
        })) {
            toast.warning(i18n.t("txtNexusUniqueName"));
            $("#screenName").trigger("focus").trigger("select");
            return;
        }

        if (!copy && JSON.stringify(draft) === JSON.stringify(catalog[id])) {
            if ($("#lcdProfile").val() !== id) {
                $("#lcdProfile").val(id).trigger("change");
            }
            return;
        }

        const controls = $("#screenSave, #screenCopy").prop("disabled", true);
        $.ajax({
            url: "/api/nexus/screen",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({deviceId: $("#deviceId").val(), id: copy ? "" : id, screen: draft}),
            success: function (response) {
                if (response.status === 1) {
                    reloadWithToast(response.message);
                } else {
                    toast.warning(response.message);
                }
            },
            error: function () {
                toast.warning(i18n.t("txtUnableToChangeLcdProfile"));
            },
            complete: function () {
                controls.prop("disabled", false);
            }
        });
    });

    const active = $("#lcdProfile").val();
    $("#screenSource").val(catalog[active] ? active : "constructor");

    load();
    refreshTrack();
    window.setInterval(refreshTrack, 5000);
});
