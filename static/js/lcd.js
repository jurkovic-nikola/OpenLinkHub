"use strict";
document.addEventListener("DOMContentLoaded", function () {
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

    function componentToHex(c) {
        const hex = c.toString(16);
        return hex.length === 1 ? "0" + hex : hex;
    }

    function rgbToHex(r, g, b) {
        return "#" + componentToHex(r) + componentToHex(g) + componentToHex(b);
    }

    function hexToRgb(hex) {
        const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
        return result ? {
            r: parseInt(result[1], 16),
            g: parseInt(result[2], 16),
            b: parseInt(result[3], 16),
            "hex": rgbToHex(parseInt(result[1], 16),parseInt(result[2], 16),parseInt(result[3], 16),)
        } : null;
    }

    $('.saveArcProfile').on('click', function(){
        const profileId = $(this).attr('data-info');
        const sensorTypeVal = $("#sensorType_" + profileId).val();
        const marginVal = $("#margin_" + profileId).val();
        const thicknessVal = $("#thickness_" + profileId).val();
        const gapRadiansVal = $("#gapRadians_" + profileId).val();
        const backgroundColorVal = $("#backgroundColor_" + profileId).val();
        const borderColorVal = $("#borderColor_" + profileId).val();
        const startColorVal = $("#startColor_" + profileId).val();
        const endColorVal = $("#endColor_" + profileId).val();
        const textColorVal = $("#textColor_" + profileId).val();

        let backgroundColorRgb = {}
        let borderColorRgb = {}
        let startColorRgb = {}
        let endColorRgb = {}
        let textColorRgb = {}

        const backgroundColor = hexToRgb(backgroundColorVal);
        backgroundColorRgb = {red:backgroundColor.r, green:backgroundColor.g, blue:backgroundColor.b, hex:backgroundColor.hex}

        const borderColor = hexToRgb(borderColorVal);
        borderColorRgb = {red:borderColor.r, green:borderColor.g, blue:borderColor.b, hex:borderColor.hex}

        const startColor = hexToRgb(startColorVal);
        startColorRgb = {red:startColor.r, green:startColor.g, blue:startColor.b, hex:startColor.hex}

        const endColor = hexToRgb(endColorVal);
        endColorRgb = {red:endColor.r, green:endColor.g, blue:endColor.b, hex:endColor.hex}

        const textColor = hexToRgb(textColorVal);
        textColorRgb = {red:textColor.r, green:textColor.g, blue:textColor.b, hex:textColor.hex}

        const pf = {};
        pf["profileId"] = parseInt(profileId);
        pf["margin"] = parseFloat(marginVal);
        pf["sensor"] = parseInt(sensorTypeVal);
        pf["thickness"] = parseFloat(thicknessVal);
        pf["gapRadians"] = parseFloat(gapRadiansVal);
        pf["backgroundColor"] = backgroundColorRgb;
        pf["borderColor"] = borderColorRgb;
        pf["startColor"] = startColorRgb;
        pf["endColor"] = endColorRgb;
        pf["textColor"] = textColorRgb;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/lcd/modes',
            type: 'PUT',
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

    $('.saveDoubleArcProfile').on('click', function(){
        let profileId = $(this).attr('data-info');
        let marginVal = $("#margin_" + profileId).val();
        let thicknessVal = $("#thickness_" + profileId).val();
        let gapRadiansVal = $("#gapRadians_" + profileId).val();
        let backgroundColorVal = $("#backgroundColor_" + profileId).val();
        let borderColorVal = $("#borderColor_" + profileId).val();
        let separatorColorVal = $("#separatorColor_" + profileId).val();

        let backgroundColorRgb = {}
        let borderColorRgb = {}
        let separatorColorRgb = {}

        const backgroundColor = hexToRgb(backgroundColorVal);
        backgroundColorRgb = {red:backgroundColor.r, green:backgroundColor.g, blue:backgroundColor.b, hex:backgroundColor.hex}

        const borderColor = hexToRgb(borderColorVal);
        borderColorRgb = {red:borderColor.r, green:borderColor.g, blue:borderColor.b, hex:borderColor.hex}

        const separatorColor = hexToRgb(separatorColorVal);
        separatorColorRgb = {red:separatorColor.r, green:separatorColor.g, blue:separatorColor.b, hex:separatorColor.hex}

        let arcs = {}
        for (let i = 0; i < 2; i++) {
            let startColorRgb = {}
            let endColorRgb = {}
            let textColorRgb = {}

            let sensorTypeVal = $("#sensorType_" + i).val();
            let startColorVal = $("#startColor_" + i).val();
            let endColorVal = $("#endColor_" + i).val();
            let textColorVal = $("#textColor_" + i).val();

            const startColor = hexToRgb(startColorVal);
            startColorRgb = {red:startColor.r, green:startColor.g, blue:startColor.b, hex:startColor.hex}

            const endColor = hexToRgb(endColorVal);
            endColorRgb = {red:endColor.r, green:endColor.g, blue:endColor.b, hex:endColor.hex}

            const textColor = hexToRgb(textColorVal);
            textColorRgb = {red:textColor.r, green:textColor.g, blue:textColor.b, hex:textColor.hex}

            arcs[i] = {
                "sensor": parseInt(sensorTypeVal),
                "startColor": startColorRgb,
                "endColor": endColorRgb,
                "textColor": textColorRgb,
            }
        }

        const pf = {};
        pf["profileId"] = parseInt(profileId);
        pf["margin"] = parseFloat(marginVal);
        pf["thickness"] = parseFloat(thicknessVal);
        pf["gapRadians"] = parseFloat(gapRadiansVal);
        pf["backgroundColor"] = backgroundColorRgb;
        pf["borderColor"] = borderColorRgb;
        pf["separatorColor"] = separatorColorRgb;
        pf["arcs"] = arcs;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/lcd/modes',
            type: 'PUT',
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