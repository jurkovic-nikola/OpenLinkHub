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

                        let profileName = index;

                        if (value.profileName.length > 0) {
                            profileName = value.profileName;
                        }

                        const html = `
                            <div style="width: 300px;">
                                <div class="card mb-4">
                                    <div style="text-align: center">
                                        <div class="card-header border-bottom border-dash-dark-1">
                                            <img src="/static/img/icons/rgb/${index}.svg" width="64" height="64" alt="Device" />
                                            <div style="width:auto;">
                                                <span style="font-size: 20px;margin-top: 10px;">${profileName}</span><br />
                                            </div>
                                        </div>
                                        <div class="card-body" style="padding: 1rem 1rem;">
                                            <div style="text-align: center;">
                                                <span class="btn btn-secondary configureRgbMode" id="${index}" style="width: 100%;">
                                                    Configure
                                                </span>
                                            </div>
                                        </div> 
                                    </div>
                                </div>
                            </div>
                        `;
                        $('#rgbDivList').append(html);
                    });

                    $('.configureRgbMode').on('click', function () {
                        const profile = $(this).attr('id');
                        const deviceId = $("#deviceId").val();
                        $.ajax({
                            url: '/api/color/profile/' + deviceId + '/' + profile,
                            type: 'GET',
                            cache: false,
                            success: function(response) {
                                try {
                                    if (response.status === 1) {
                                        const data = response.data;
                                        const startColor = rgbToHex(data.start.red, data.start.green, data.start.blue);
                                        const endColor = rgbToHex(data.end.red, data.end.green, data.end.blue);
                                        let rgbDirectionHtml = '';
                                        let alternateColorsHtml = '';
                                        let profileName = profile;

                                        if (data.profileName.length > 0) {
                                            profileName = data.profileName;
                                        }

                                        if (parseInt(data.rgbDirection) > 0) {
                                            const directions = {
                                                1: "Top to Bottom",
                                                2: "Bottom to Top",
                                                4: "Left to Right",
                                                5: "Right to Left"
                                            };

                                            let selectHtml = `<select id="rgbDirection_${profile}" class="form-select keyLayout">`;
                                            for (const [val, label] of Object.entries(directions)) {
                                                if (parseInt(data.rgbDirection) === parseInt(val)) {
                                                    selectHtml += `<option value="${val}" selected>${label}</option>`;
                                                } else {
                                                    selectHtml += `<option value="${val}">${label}</option>`;
                                                }
                                            }
                                            selectHtml += `</select>`;

                                            rgbDirectionHtml = `
                                                <div style="margin-top:10px">
                                                    <div class="progress" style="height: 1px">
                                                        <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                                    </div>
                                                    <div class="rgb-direction-container">
                                                        <div class="rgb-direction-left">
                                                            Direction
                                                        </div>
                                                        <div class="rgb-direction-right">
                                                            ${selectHtml}
                                                        </div>
                                                    </div>                                                        
                                                </div>
                                            `;
                                        }

                                        // Speed slider starts //
                                        let alternateColors = '';
                                        if (data.alternateColors === true) {
                                            alternateColors = '<input id="alternateColors_' + profile + '" type="checkbox" checked/>';
                                        } else {
                                            alternateColors = '<input id="alternateColors_' + profile + '" type="checkbox"/>';
                                        }

                                        // Alternating starts //
                                        switch (profile) {
                                            case "colorpulse":
                                            case "colorshift":
                                            case "tlk":
                                            case "tlr":
                                            case "rain":
                                            case "visor":
                                            case "colorwave": {
                                                alternateColorsHtml = `
                                                    <div style="margin-top:10px">
                                                        <div class="progress" style="height: 1px">
                                                            <div class="progress-bar bg-dash-color-5" role="progressbar" style="width: 15%" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100"></div>
                                                        </div>
                                                        <div class="rgb-alternating-container">
                                                            <div class="rgb-alternating-left">
                                                                Alternating (Slipstream Only)
                                                            </div>
                                                            <div class="rgb-alternating-right">
                                                                ${alternateColors}
                                                            </div>
                                                        </div>                                                        
                                                    </div>
                                                `;
                                            } break;
                                        }
                                        // Alternating ends //

                                        // Speed slider starts //
                                        let speedSliderHtml = '';
                                        switch (profile) {
                                            case "cpu-temperature":
                                            case "gpu-temperature":
                                            case "liquid-temperature":
                                            case "static":
                                            case "storm":
                                            case "off": {
                                                speedSliderHtml = `<input class="brightness-slider" type="range" id="speed_${profile}" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="${data.speed}" step="0.1" disabled/>`;
                                            } break;
                                            case "tlk":
                                            case "tlr":
                                            case "spiralrainbow":
                                            case "rainbowwave":
                                            case "rain":
                                            case "visor":
                                            case "colorwave": {
                                                speedSliderHtml = `<input class="brightness-slider" type="range" id="speed_${profile}" name="speedSlider" style="margin-top: 0;" min="1" max="3" value="${data.speed}" step="1" />`;
                                            } break;
                                            default: {
                                                speedSliderHtml = `<input class="brightness-slider" type="range" id="speed_${profile}" name="speedSlider" style="margin-top: 0;" min="1" max="10" value="${data.speed}" step="0.1" />`;
                                            } break;
                                        }

                                        const speedHtml = `
                                            <div class="rgb-speed-container">
                                                <div class="rgb-speed-left">
                                                    <img src="/static/img/icons/icon-fast.svg" width="30" height="30" alt="Fast" />
                                                </div>
                                                <div class="rgb-speed-middle">
                                                    ${speedSliderHtml}
                                                </div>
                                                <div class="rgb-speed-right">
                                                    <img src="/static/img/icons/icon-slow.svg" width="30" height="30" alt="Sloe" />
                                                </div>
                                            </div>
                                        `;
                                        // Speed slider ends //

                                        // Colors starts //
                                        let colorHtmlElement = '';
                                        let gradients = false;

                                        if (data.gradients != null) {
                                            colorHtmlElement += `<div class="row" style="margin-top: 10px;">`;
                                            colorHtmlElement += `<div class="col-lg-12" id="gradient-colors-container">`;
                                            $.each(data.gradients, function (index, value) {
                                                const gradientColor = rgbToHex(value.red, value.green, value.blue);
                                                colorHtmlElement += `<input type="color" id="gradient_${profile}_${index}" class="rgb-color-gradient" value="${gradientColor}" style="border:0;padding:10px;"/>`;
                                            });
                                            colorHtmlElement += `</div>`;
                                            colorHtmlElement += `</div>`;

                                            // Control buttons
                                            colorHtmlElement += `<div class="row" style="margin-top: 10px;">`;
                                            colorHtmlElement += `<div class="col-lg-12">`;
                                            colorHtmlElement += `<span class="btn btn-secondary addGradientColor" type="button"">+</span>`;
                                            colorHtmlElement += `<span class="btn btn-secondary deleteGradientColor" style="margin-left: 20px;" type="button"">-</span>`;
                                            colorHtmlElement += `</div>`;
                                            colorHtmlElement += `</div>`;

                                        } else {
                                            colorHtmlElement = `
                                                <input type="color" id="startColor_${profile}" class="rgb-color-start" value="${startColor}" style="margin-right: 20px;border:0;"/>
                                                <input type="color" id="endColor_${profile}" class="rgb-color-end" value="${endColor}" style="border:0;" />
                                            `;
                                        }
                                        // Colors ends //
                                        let modalElement = `
                                            <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel" aria-hidden="true">
                                                <div class="modal-dialog modal-dialog-700">
                                                    <div class="modal-content" style="width: 700px;">
                                                    <div class="modal-header">
                                                      <h5 class="modal-title" id="keyboardControlDial">${profileName}</h5>
                                                      <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                                    </div>
                                                    <div class="modal-body">
                                                        <form>
                                                            <table class="table-rgb-data">
                                                                <tbody>
                                                                    <tr>
                                                                        <td class="left">
                                                                            ${colorHtmlElement}
                                                                        </td>
                                                                        <td class="right">
                                                                            ${speedHtml}
                                                                            ${rgbDirectionHtml}
                                                                            ${alternateColorsHtml}
                                                                        </td>
                                                                    </tr>
                                                                </tbody>
                                                            </table>
                                                         </form>
                                                    </div>
                                                    <div class="modal-footer">
                                                      <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                                      <button class="btn btn-primary saveRgbProfile" type="button" id="${profile}">Save</button>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>`;
                                        const modal = $(modalElement).modal('toggle');

                                        modal.on('hidden.bs.modal', function () {
                                            modal.data('bs.modal', null);
                                            modal.remove();
                                        })

                                        modal.on('shown.bs.modal', function (e) {
                                            modal.find('.addGradientColor').on('click', function () {
                                                const pf = {};
                                                pf["deviceId"] = deviceId;
                                                pf["profile"] = profile;
                                                const json = JSON.stringify(pf, null, 2);

                                                $.ajax({
                                                    url: '/api/color/gradient/add',
                                                    type: 'POST',
                                                    data: json,
                                                    cache: false,
                                                    success: function (response) {
                                                        try {
                                                            if (response.status === 1) {
                                                                const appendElementColor = rgbToHex(0, 255, 255);
                                                                const appendElement = `<input type="color" id="gradient_${profile}_${response.data}" class="rgb-color-gradient" value="${appendElementColor}" style="border:0;padding:10px;"/>`;
                                                                modal.find('#gradient-colors-container').append(appendElement)
                                                            } else {
                                                                toast.warning(response.message);
                                                            }
                                                        } catch (err) {
                                                            toast.warning(response.message);
                                                        }
                                                    }
                                                });
                                            });

                                            modal.find('.deleteGradientColor').on('click', function () {
                                                const pf = {};
                                                pf["deviceId"] = deviceId;
                                                pf["profile"] = profile;
                                                const json = JSON.stringify(pf, null, 2);

                                                $.ajax({
                                                    url: '/api/color/gradient/delete',
                                                    type: 'POST',
                                                    data: json,
                                                    cache: false,
                                                    success: function (response) {
                                                        try {
                                                            if (response.status === 1) {
                                                                $('#gradient_' + profile + '_' + response.data).remove();
                                                            } else {
                                                                toast.warning(response.message);
                                                            }
                                                        } catch (err) {
                                                            toast.warning(response.message);
                                                        }
                                                    }
                                                });
                                            });

                                            modal.find('.saveRgbProfile').on('click', function () {
                                                let startColorRgb = {}
                                                let endColorRgb = {}

                                                let speed = $("#speed_" + profile).val();
                                                let rgbDirection = $("#rgbDirection_" + profile).val();
                                                let alternateColors = $("#alternateColors_" + profile).is(':checked');
                                                const startColorVal = $("#startColor_" + profile).val();
                                                const endColorVal = $("#endColor_" + profile).val();

                                                if (startColorVal == null) {
                                                    startColorRgb = {red: 0, green: 0, blue: 0}
                                                } else {
                                                    const startColor = hexToRgb(startColorVal);
                                                    startColorRgb = {
                                                        red: startColor.r,
                                                        green: startColor.g,
                                                        blue: startColor.b
                                                    }
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

                                                const pf = {};
                                                pf["deviceId"] = deviceId;
                                                pf["profile"] = profile;
                                                pf["startColor"] = startColorRgb;
                                                pf["endColor"] = endColorRgb;
                                                pf["speed"] = parseFloat(speed);
                                                pf["alternateColors"] = alternateColors;
                                                pf["rgbDirection"] = parseInt(rgbDirection);

                                                let count = $('#gradient-colors-container input[id^="gradient_gradient_"]').length;
                                                if (count > 0) {
                                                    const colorZones = {};
                                                    for (let i= 0; i < count; i++) {
                                                        let color = modal.find("#gradient_gradient_" + i).val();
                                                        color = hexToRgb(color);
                                                        colorZones[i] = {red: color.r, green: color.g, blue: color.b};
                                                    }
                                                    pf["colorZones"] = colorZones;
                                                }

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
                                        })
                                    } else {
                                        toast.warning(response.data);
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