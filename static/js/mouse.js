"use strict";

document.addEventListener("DOMContentLoaded", function () {
    function hexToRgb(hex) {
        const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
        return result ? {
            r: parseInt(result[1], 16),
            g: parseInt(result[2], 16),
            b: parseInt(result[3], 16)
        } : null;
    }

    function CreateToastr() {
        toastr.options = {
            "closeButton": true,
            "debug": false,
            "newestOnTop": false,
            "progressBar": true,
            "positionClass": "toast-top-right",
            "preventDuplicates": true,
            "onclick": null,
            "showDuration": 300,
            "hideDuration": 1000,
            "timeOut": 7000,
            "extendedTimeout": "1000",
            "showEasing": "swing",
            "hideEasing": "linear",
            "showMethod": "fadeIn",
            "hideMethod": "fadeOut",
        }
        return toastr
    }

    // Init toastr
    const toast = CreateToastr();

    $(function() {
        for (let i = 0; i <= 2; i++) {
            const stage = document.getElementById("stage" + i);
            const stageValue = document.getElementById("stageValue" + i);
            stage.oninput = function() {
                stageValue.value = this.value;
            }
        }
    });

    $('#defaultDPI').on('click', function () {
        for (let i = 0; i <= 2; i++) {
            const stage = document.getElementById("stage" + i);
            const stageValue = document.getElementById("stageValue" + i);
            switch (i) {
                case 0:
                    stage.value = 800
                    break;
                case 1:
                    stage.value = 1500
                    break;
                case 2:
                    stage.value = 3000
                    break;
            }
            stageValue.value = stage.value
        }
    });

    $('#default5DPI').on('click', function () {
        for (let i = 0; i <= 4; i++) {
            const stage = document.getElementById("stage" + i);
            const stageValue = document.getElementById("stageValue" + i);
            switch (i) {
                case 0:
                    stage.value = 400
                    break;
                case 1:
                    stage.value = 800
                    break;
                case 2:
                    stage.value = 1200
                    break;
                case 3:
                    stage.value = 1600
                    break;
                case 4:
                    stage.value = 3200
                    break;
            }
            stageValue.value = stage.value
        }
    });

    $('#saveDPI').on('click', function () {
        const deviceId = $("#deviceId").val();

        const pf = {};
        let stages = {};

        pf["deviceId"] = deviceId;
        for (let i = 0; i <= 2; i++) {
            const stage = $("#stageValue" + i).val();
            stages[i] =  parseInt(stage);
        }
        pf["stages"] = stages;
        const json = JSON.stringify(pf, null, 2);

        console.log(json)
        $.ajax({
            url: '/api/mouse/dpi',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        toast.success(response.message);
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.mouseRgbProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const profile = $(this).val().split(";");
        if (profile.length < 2 || profile.length > 2) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(profile[0]);
        pf["profile"] = profile[1];

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        location.reload();
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('#saveZoneColors').on('click', function () {
        const deviceId = $("#deviceId").val();
        const zones = parseInt($("#zones").val());

        let colors = {};
        for (let i = 0; i < zones; i++) {
            const zoneColor = $("#zoneColor"+i).val();
            const zoneColorRgb = hexToRgb(zoneColor);
            colors[i] = {red: zoneColorRgb.r, green: zoneColorRgb.g, blue: zoneColorRgb.b}
        }

        const dpiColor = $("#dpiColor").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["colorZones"] = colors
        if (dpiColor != null) {
            const dpiColorRgb = hexToRgb(dpiColor);
            pf["colorDpi"] = {red:dpiColorRgb.r, green:dpiColorRgb.g, blue:dpiColorRgb.b}
        }

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/mouse/zoneColors',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        location.reload()
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('#saveDpiColors').on('click', function () {
        const deviceId = $("#deviceId").val();
        const dpis = parseInt($("#dpis").val());

        let colors = {};
        for (let i = 0; i < dpis; i++) {
            const dpiColor = $("#dpiColor"+i).val();
            const dpiColorRgb = hexToRgb(dpiColor);
            colors[i] = {red: dpiColorRgb.r, green: dpiColorRgb.g, blue: dpiColorRgb.b}
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["colorZones"] = colors
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/mouse/dpiColors',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        location.reload()
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.mouseSleepModes').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["sleepMode"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/mouse/sleep',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        toast.success(response.message);
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });
});