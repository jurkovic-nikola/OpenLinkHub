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

    // Left vibration slider update
    $('#leftVibrationValue').on('input change', function() {
        $('#leftVibrationVal').text($(this).val());
    });

    // Right vibration slider update
    $('#rightVibrationValue').on('input change', function() {
        $('#rightVibrationVal').text($(this).val());
    });

    $('.controllerRgbProfile').on('change', function () {
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

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["colorZones"] = colors

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/controller/zoneColors',
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

    $('#btnSaveLeftVibrationValue').on('click', function () {
        const deviceId = $("#deviceId").val();
        const vibrationValue = parseInt($("#leftVibrationValue").val());
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["vibrationModule"] = 0;
        pf["vibrationValue"] = vibrationValue;

        const json = JSON.stringify(pf, null, 2);

        console.log(json)
        $.ajax({
            url: '/api/controller/vibration',
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

    $('#btnSaveRightVibrationValue').on('click', function () {
        const deviceId = $("#deviceId").val();
        const vibrationValue = parseInt($("#rightVibrationValue").val());
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["vibrationModule"] = 1;
        pf["vibrationValue"] = vibrationValue;

        const json = JSON.stringify(pf, null, 2);

        console.log(json)
        $.ajax({
            url: '/api/controller/vibration',
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