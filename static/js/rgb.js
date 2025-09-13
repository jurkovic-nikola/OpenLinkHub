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

                        let alternateColorsHtml = '';
                        let rgbDirectionHtml = '';
                        let keyboardOnlyText = '';
                        let startColorHtml = '';
                        let endColorHtml = '';
                        let speedHtml = '';
                        let profileName = index;

                        if (value.profileName.length > 0) {
                            profileName = value.profileName;
                        }

                        startColorHtml = '<input type="color" id="startColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + startColor + '">';
                        endColorHtml = '<input type="color" id="endColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + endColor + '">';
                        speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                            '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                            '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" /></div>' +
                            '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                            '</div>';

                        let alternateColors = '';
                        if (value.alternateColors === true) {
                            alternateColors = '<input id="alternateColors_' + index + '" type="checkbox" checked/>';
                        } else {
                            alternateColors = '<input id="alternateColors_' + index + '" type="checkbox"/>';
                        }

                        if (parseInt(value.rgbDirection) > 0) {
                            const directions = {
                                1: "Top to Bottom",
                                2: "Bottom to Top",
                                4: "Left to Right",
                                5: "Right to Left"
                            };

                            let selectHtml = `<select id="rgbDirection_${index}" class="form-select keyLayout">`;
                            for (const [val, label] of Object.entries(directions)) {
                                if (parseInt(value.rgbDirection) === parseInt(val)) {
                                    selectHtml += `<option value="${val}" selected>${label}</option>`;
                                } else {
                                    selectHtml += `<option value="${val}">${label}</option>`;
                                }
                            }
                            selectHtml += `</select>`;

                            rgbDirectionHtml = `
                                <div class="d-flex align-items-center justify-content-between mb-2" style="margin-top: 10px;">
                                    <div class="me-2">
                                        <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Direction</p>
                                    </div>
                                    <p class="text-sm lh-1 mb-0 text-dash-color-2">${selectHtml}</p>
                                </div>
                                <div class="progress" style="height: 3px">
                                    <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                </div>
                            `;
                        }
                        switch (index) {
                            case "colorpulse": {
                                keyboardOnlyText = '<p class="text-md-start lh-1 mb-0 text-dash-color-3" style="margin-top: 5px;"><span style="color: #37929d !important;">Keyboard + Other Devices</span></p>';
                                alternateColorsHtml = `
                                    <div class="d-flex align-items-center justify-content-between" style="margin-bottom: 10px;">
                                        <div class="me-2">
                                            <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Alternating</p>
                                        </div>
                                        <p class="text-sm lh-1 mb-0 text-dash-color-2">${alternateColors}</p>
                                    </div>
                                    <div class="progress" style="height: 3px;margin-bottom: 10px;">
                                        <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                    </div>
                                `;
                            }
                            break;
                            case "colorshift": {
                                keyboardOnlyText = '<p class="text-md-start lh-1 mb-0 text-dash-color-3" style="margin-top: 5px;"><span style="color: #37929d !important;">Keyboard + Other Devices</span></p>';
                                alternateColorsHtml = `
                                    <div class="d-flex align-items-center justify-content-between" style="margin-bottom: 10px;">
                                        <div class="me-2">
                                            <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Alternating</p>
                                        </div>
                                        <p class="text-sm lh-1 mb-0 text-dash-color-2">${alternateColors}</p>
                                    </div>
                                    <div class="progress" style="height: 3px;margin-bottom: 10px;">
                                        <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                    </div>
                                `;
                            }
                            break;
                            case "colorwarp": {
                                startColorHtml = '<input type="color" id="startColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + startColor + '" disabled>'
                                endColorHtml = '<input type="color" id="endColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + endColor + '" disabled>';
                            }
                            break;
                            case "rainbow": {
                                startColorHtml = '<input type="color" id="startColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + startColor + '" disabled>'
                                endColorHtml = '<input type="color" id="endColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + endColor + '" disabled>';
                            }
                            break;
                            case "watercolor": {
                                startColorHtml = '<input type="color" id="startColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + startColor + '" disabled>'
                                endColorHtml = '<input type="color" id="endColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + endColor + '" disabled>';
                            }
                            break;
                            case "nebula": {
                                startColorHtml = '<input type="color" id="startColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + startColor + '" disabled>'
                                endColorHtml = '<input type="color" id="endColor_' + index + '" style="width: 100px;height: 38px;padding: 0;float: left;margin-top: 2px;" value="' + endColor + '" disabled>';
                            }
                            break;
                            case "cpu-temperature": {
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" disabled /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                            break;
                            case "gpu-temperature": {
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" disabled /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                            break;
                            case "liquid-temperature": {
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" disabled /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                            break;
                            case "static": {
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" disabled /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                            break;
                            case "storm": {
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="' + value.speed + '" step="0.1" disabled /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                            break;
                            case "tlk": {
                                keyboardOnlyText = '<p class="text-md-start lh-1 mb-0 text-dash-color-3" style="margin-top: 5px;"><span style="color: #37929d !important;">Keyboard Only</span></p>';
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="3" value="' + value.speed + '" step="1" /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                                alternateColorsHtml = `
                                    <div class="d-flex align-items-center justify-content-between" style="margin-bottom: 10px;">
                                        <div class="me-2">
                                            <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Alternating</p>
                                        </div>
                                        <p class="text-sm lh-1 mb-0 text-dash-color-2">${alternateColors}</p>
                                    </div>
                                    <div class="progress" style="height: 3px;margin-bottom: 10px;">
                                        <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                    </div>
                                `;
                            }
                            break;
                            case "tlr": {
                                keyboardOnlyText = '<p class="text-md-start lh-1 mb-0 text-dash-color-3" style="margin-top: 5px;"><span style="color: #37929d !important;">Keyboard Only</span></p>';
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="3" value="' + value.speed + '" step="1" /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                                alternateColorsHtml = `
                                    <div class="d-flex align-items-center justify-content-between" style="margin-bottom: 10px;">
                                        <div class="me-2">
                                            <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Alternating</p>
                                        </div>
                                        <p class="text-sm lh-1 mb-0 text-dash-color-2">${alternateColors}</p>
                                    </div>
                                    <div class="progress" style="height: 3px;margin-bottom: 10px;">
                                        <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                    </div>
                                `;
                            }
                            break
                            case "spiralrainbow": {
                                startColorHtml = 'N/A';
                                endColorHtml = 'N/A';
                                keyboardOnlyText = '<p class="text-md-start lh-1 mb-0 text-dash-color-3" style="margin-top: 5px;"><span style="color: #37929d !important;">Keyboard + Other Devices</span></p>';
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="3" value="' + value.speed + '" step="1" /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                            break;
                            case "rainbowwave": {
                                startColorHtml = 'N/A';
                                endColorHtml = 'N/A';
                                keyboardOnlyText = '<p class="text-md-start lh-1 mb-0 text-dash-color-3" style="margin-top: 5px;"><span style="color: #37929d !important;">Keyboard Only</span></p>';
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="3" value="' + value.speed + '" step="1" /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                            break;
                            case "rain": {
                                alternateColorsHtml = `
                                    <div class="d-flex align-items-center justify-content-between" style="margin-bottom: 10px;">
                                        <div class="me-2">
                                            <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Alternating</p>
                                        </div>
                                        <p class="text-sm lh-1 mb-0 text-dash-color-2">${alternateColors}</p>
                                    </div>
                                    <div class="progress" style="height: 3px;margin-bottom: 10px;">
                                        <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                    </div>
                                `;
                                keyboardOnlyText = '<p class="text-md-start lh-1 mb-0 text-dash-color-3" style="margin-top: 5px;"><span style="color: #37929d !important;">Keyboard Only</span></p>';
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="3" value="' + value.speed + '" step="1" /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                            break;
                            case "visor": {
                                alternateColorsHtml = `
                                    <div class="d-flex align-items-center justify-content-between" style="margin-bottom: 10px;">
                                        <div class="me-2">
                                            <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Alternating</p>
                                        </div>
                                        <p class="text-sm lh-1 mb-0 text-dash-color-2">${alternateColors}</p>
                                    </div>
                                    <div class="progress" style="height: 3px;margin-bottom: 10px;">
                                        <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                    </div>
                                `;
                                keyboardOnlyText = '<p class="text-md-start lh-1 mb-0 text-dash-color-3" style="margin-top: 5px;"><span style="color: #37929d !important;">Keyboard + Other Devices</span></p>';
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="3" value="' + value.speed + '" step="1" /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                            break;
                            case "colorwave": {
                                alternateColorsHtml = `
                                    <div class="d-flex align-items-center justify-content-between" style="margin-bottom: 10px;">
                                        <div class="me-2">
                                            <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Alternating</p>
                                        </div>
                                        <p class="text-sm lh-1 mb-0 text-dash-color-2">${alternateColors}</p>
                                    </div>
                                    <div class="progress" style="height: 3px;margin-bottom: 10px;">
                                        <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                    </div>
                                `;
                                keyboardOnlyText = '<p class="text-md-start lh-1 mb-0 text-dash-color-3" style="margin-top: 5px;"><span style="color: #37929d !important;">Keyboard Only</span></p>';
                                speedHtml = '<div style="display: flex; align-items: center; width: 250px;">' +
                                    '<div style="float: left;width: 15%;"><img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" /></div>' +
                                    '<div style="float: left;width: 70%;margin-top:4px;margin-left: 5px;"><input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 0;" min="1" max="3" value="' + value.speed + '" step="1" /></div>' +
                                    '<div style="float: right;width: 15%;text-align: right;"><img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" /></div>' +
                                    '</div>';
                            }
                            break;
                        }

                        const html = `
                            <div style="width: auto;">
                                <div class="card mb-4">
                                    <div class="card-header border-bottom border-dash-dark-1">
                                        <div class="ds-svg-placeholder-left">
                                            <img src="/static/img/icons/icon-rgb.svg" width="46" height="46" alt="Device" />
                                        </div>
                                        <div class="ds-svg-placeholder-left" style="width:auto;margin-left: 30px;">
                                            <span>${profileName}</span><br />
                                            ${keyboardOnlyText}
                                        </div>
                                    </div>
                                    <div class="card-body" style="padding: 1rem 1rem;">
                                        <div style="text-align: center;">
                                            ${alternateColorsHtml}
                                            <div class="d-flex align-items-center justify-content-between mb-2">
                                                <div class="me-2">
                                                    <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Start</p>
                                                </div>
                                                <p class="text-sm lh-1 mb-0 text-dash-color-2">${startColorHtml}</p>
                                            </div>
                                            <div class="progress" style="height: 3px">
                                                <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                            </div>
                                            
                                            <div class="d-flex align-items-center justify-content-between mb-2" style="margin-top: 10px;">
                                                <div class="me-2">
                                                    <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">End</p>
                                                </div>
                                                <p class="text-sm lh-1 mb-0 text-dash-color-2">${endColorHtml}</p>
                                            </div>
                                            <div class="progress" style="height: 3px">
                                                <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                            </div>
                                            
                                            <div class="d-flex align-items-center justify-content-between mb-2" style="margin-top: 10px;">
                                                <div class="me-2">
                                                    <p class="text-sm text-uppercase text-gray-600 lh-1 mb-0">Speed</p>
                                                </div>
                                                <p class="text-sm lh-1 mb-0 text-dash-color-2">${speedHtml}</p>
                                            </div>
                                            <div class="progress" style="height: 3px">
                                                <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                            </div>
                                            ${rgbDirectionHtml}
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
                        let rgbDirection = $("#rgbDirection_" + profile).val();
                        let alternateColors = $("#alternateColors_" + profile).is(':checked');
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

                        if (alternateColors == null) {
                            alternateColors = false;
                        }

                        if (rgbDirection == null) {
                            rgbDirection = 0;
                        }

                        console.log(rgbDirection);

                        const pf = {};
                        pf["deviceId"] = deviceId;
                        pf["profile"] = profile;
                        pf["startColor"] = startColorRgb;
                        pf["endColor"] = endColorRgb;
                        pf["speed"] = parseFloat(speed);
                        pf["alternateColors"] = alternateColors;
                        pf["rgbDirection"] = parseInt(rgbDirection);

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