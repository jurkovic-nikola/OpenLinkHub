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
            b: parseInt(result[3], 16)
        } : null;
    }

    $('.rgbList').on('click', function(){
        const deviceId = $(this).attr('id');
        $('.rgbList').removeClass('selected-effect');
        $(this).addClass('selected-effect');
        $.ajax({
            url: '/api/color/' + deviceId,
            dataType: 'JSON',
            success: function(response) {
                if (response.code === 0) {
                    toast.warning(response.message);
                } else {
                    $("#deviceId").val(deviceId);
                    $('#rgbDivList').empty();

                    $.each(response.data.profiles, function( index, value ) {
                        if (index === "keyboard" ||
                            index === "mouse" ||
                            index === "stand" ||
                            index === "mousepad" ||
                            index === "headset" ||
                            index === "custom" ||
                            index === "off") {
                            return true
                        }
                        const startColor = rgbToHex(value.start.red, value.start.green, value.start.blue);
                        const endColor = rgbToHex(value.end.red, value.end.green, value.end.blue);

                        let sc = '';
                        let ec = '';
                        let sp = '';
                        let sm = '';

                        sc = '<input type="color" id="startColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + startColor + '">';
                        ec = '<input type="color" id="endColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + endColor + '">';
                        sp = '<div style="display: flex; align-items: center; width: 250px;">' +
                            '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                            '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" /></div>' +
                            '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                            '</div>';


                        switch (index) {
                            case "colorwarp": {
                                sc = '<input type="color" id="startColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + startColor + '" disabled>'
                                ec = '<input type="color" id="endColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + endColor + '" disabled>';
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "rainbow": {
                                sc = '<input type="color" id="startColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + startColor + '" disabled>'
                                ec = '<input type="color" id="endColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + endColor + '" disabled>';
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "watercolor": {
                                sc = '<input type="color" id="startColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + startColor + '" disabled>'
                                ec = '<input type="color" id="endColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + endColor + '" disabled>';
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "cpu-temperature": {
                                sp = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" disabled /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "gpu-temperature": {
                                sp = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" disabled /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "liquid-temperature": {
                                sp = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" disabled /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "circle": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "circleshift": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "colorpulse": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "colorshift": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "flickering": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "rotator": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "spinner": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                            case "static": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                sp = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" disabled /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                                break;
                            case "storm": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                sp = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" disabled /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                                break;
                            case "wave": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            }
                                break;
                        }

                        const html = `
                            <div style="width: auto;">
                                <div class="card mb-4">
                                    <div class="card-header border-bottom border-dash-dark-1">
                                        <div class="ds-svg-placeholder-left" style="width: 46px;height: 52px;">
                                            <img src="/static/img/icons/icon-rgb.svg" width="46" height="46" alt="Device" />
                                        </div>
                                        <div class="ds-svg-placeholder-right2">
                                            <span>${index}</span><br />
                                        </div>
                                    </div>
                                    <div class="card-body" style="padding: 1rem 1rem;">
                                        <div style="text-align: center;">
                                            <div class="d-flex align-items-center justify-content-between mb-2">
                                                <div class="me-2">
                                                    <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Start</p>
                                                </div>
                                                <p class="text-sm lh-1 mb-0 text-dash-color-2">${sc}</p>
                                            </div>
                                            <div class="progress" style="height: 3px">
                                                <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                            </div>
                                            
                                            <div class="d-flex align-items-center justify-content-between mb-2" style="margin-top: 10px;">
                                                <div class="me-2">
                                                    <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">End</p>
                                                </div>
                                                <p class="text-sm lh-1 mb-0 text-dash-color-2">${ec}</p>
                                            </div>
                                            <div class="progress" style="height: 3px">
                                                <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                            </div>
                                            
                                            <div class="d-flex align-items-center justify-content-between mb-2" style="margin-top: 10px;">
                                                <div class="me-2">
                                                    <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Speed</p>
                                                </div>
                                                <p class="text-sm lh-1 mb-0 text-dash-color-2">${sp}</p>
                                            </div>
                                            <div class="progress" style="height: 3px">
                                                <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                            </div>
                                            <span class="btn btn-secondary saveRgbProfile" id="${index}" style="width: 100%;margin-top:10px;">
                                                Save
                                            </span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        `;
                        $('#rgbDivList').append(html);
                    });

                    $('.saveRgbProfile').on('click', function () {
                        let startColorRgb = {}
                        let endColorRgb = {}

                        const deviceId = $("#deviceId").val();
                        const profile = $(this).attr('id');
                        let speed = $("#speed_" + profile).val();
                        const startColorVal = $("#startColor_" + profile).val();
                        const endColorVal = $("#endColor_" + profile).val();

                        if (startColorVal == null) {
                            startColorRgb = {red: 0, green: 0, blue: 0}
                        } else {
                            const startColor = hexToRgb(startColorVal);
                            startColorRgb = {red: startColor.r, green: startColor.g, blue: startColor.b}
                        }
                        if (endColorVal == null) {
                            endColorRgb = {red: 0, green: 0, blue: 0}
                        } else {
                            const endColor = hexToRgb(endColorVal);
                            endColorRgb = {red: endColor.r, green: endColor.g, blue: endColor.b}
                        }

                        if (speed == null) {
                            speed = 1
                        }

                        const pf = {};
                        pf["deviceId"] = deviceId;
                        pf["profile"] = profile;
                        pf["startColor"] = startColorRgb;
                        pf["endColor"] = endColorRgb;
                        pf["speed"] = parseFloat(speed);

                        const json = JSON.stringify(pf, null, 2);

                        $.ajax({
                            url: '/api/color/change',
                            type: 'PUT',
                            data: json,
                            cache: false,
                            success: function (response) {
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
                }
            }
        });
    });
});