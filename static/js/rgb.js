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
                                            case "colorwave":
                                            case "watercolor": {
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
                                        let size = 700;
                                        let leftTableCss = "left";
                                        let rightTableCss = "right";

                                        if (data.gradients != null) {
                                            colorHtmlElement += `<div id="gradientWrapper">`;
                                            colorHtmlElement += `<canvas id="gradientCanvas" width="600" height="80"></canvas>`;
                                            colorHtmlElement += `</div>`;

                                            /*
                                            colorHtmlElement += `<div class="row" style="margin-top: 10px;">`;
                                            colorHtmlElement += `<div class="col-lg-12" id="gradient-colors-container">`;
                                            $.each(data.gradients, function (index, value) {
                                                const gradientColor = rgbToHex(value.red, value.green, value.blue);
                                                colorHtmlElement += `<input type="color" id="gradient_${profile}_${index}" class="rgb-color-gradient" value="${gradientColor}" style="border:0;padding:10px;"/>`;
                                            });
                                            colorHtmlElement += `</div>`;
                                            colorHtmlElement += `</div>`;
                                            */

                                            // Control buttons
                                            colorHtmlElement += `<div class="row" style="margin-top: 10px;">`;
                                            colorHtmlElement += `<div class="col-lg-12">`;
                                            colorHtmlElement += `<span class="btn btn-secondary addGradientColor" id="addGradientColor" type="button"">+</span>`;
                                            colorHtmlElement += `<span class="btn btn-secondary deleteGradientColor" id="deleteGradientColor" style="margin-left: 20px;" type="button"">-</span>`;
                                            colorHtmlElement += `</div>`;
                                            colorHtmlElement += `</div>`;
                                            size = 1000;
                                            leftTableCss = "left-gradient";
                                            rightTableCss = "right-gradient";
                                        } else {
                                            colorHtmlElement = `
                                                <input type="color" id="startColor_${profile}" class="rgb-color-start" value="${startColor}" style="margin-right: 20px;border:0;"/>
                                                <input type="color" id="endColor_${profile}" class="rgb-color-end" value="${endColor}" style="border:0;" />
                                            `;
                                        }
                                        // Colors ends //

                                        let modalElement = `
                                            <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel" aria-hidden="true">
                                                <div class="modal-dialog modal-dialog-${size}">
                                                    <div class="modal-content" style="width: ${size}px;">
                                                    <div class="modal-header">
                                                      <h5 class="modal-title" id="keyboardControlDial">${profileName}</h5>
                                                      <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                                    </div>
                                                    <div class="modal-body">
                                                        <form>
                                                            <table class="table-rgb-data">
                                                                <tbody>
                                                                    <tr>
                                                                        <td class="${leftTableCss}">
                                                                            ${colorHtmlElement}
                                                                        </td>
                                                                        <td class="${rightTableCss}">
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
                                            let gradientColors = [];
                                            let selectedColor = null;

                                            if (data.gradients != null) {
                                                const wrapper = $("#gradientWrapper");
                                                const canvas = document.getElementById("gradientCanvas");
                                                const ctx = canvas.getContext("2d");
                                                let colorPicker = $('<input type="color" style="position:absolute; width:0; height:0; opacity:0;">');
                                                $("body").append(colorPicker);

                                                function initGradientColors() {
                                                    let keys = Object.keys(data.gradients);
                                                    let count = keys.length;

                                                    keys.forEach((key, i) => {
                                                        let g = data.gradients[key];
                                                        let pos = i / (count - 1);
                                                        createGradientColor(g.position, g.red, g.green, g.blue, g.brightness);
                                                    });

                                                    drawGradient();
                                                }

                                                function createGradientColor(position, r, g, b, brightness) {
                                                    let div = $('<div class="stop"></div>');

                                                    // Compute Y position from brightness
                                                    let maxY = wrapper.height() - 16;
                                                    let top = (1 - brightness) * maxY;

                                                    div.css({
                                                        left: position * (wrapper.width() - 16) + "px",
                                                        top: top + "px",
                                                        background: `rgb(${Math.round(r * brightness)},${Math.round(g * brightness)},${Math.round(b * brightness)})`
                                                    });

                                                    wrapper.append(div);

                                                    let gradientObj = { el: div, r, g, b, brightness };
                                                    gradientColors.push(gradientObj);

                                                    // draggable
                                                    div.draggable({
                                                        containment: "parent",
                                                        drag: function (event, ui) {
                                                            applyBrightnessFromY(gradientObj, ui.position.top);
                                                            drawGradient();
                                                        },
                                                        stop: function () {
                                                            sortGradientsByX();
                                                        }
                                                    });
                                                }

                                                colorPicker.on("input", function () {
                                                    let gradientObj = colorPicker.data("stop");
                                                    if (!gradientObj) return;

                                                    let hex = $(this).val();
                                                    let rgb = hexToRgb(hex);

                                                    // Update the base color
                                                    gradientObj.r = rgb.r;
                                                    gradientObj.g = rgb.g;
                                                    gradientObj.b = rgb.b;

                                                    // Reapply brightness to update DOM
                                                    let top = parseFloat(gradientObj.el.css("top")) || 0;
                                                    applyBrightnessFromY(gradientObj, top, true);

                                                    // Redraw gradient
                                                    drawGradient();
                                                });

                                                function applyBrightnessFromY(gradientObj, y, updatePosition = true) {
                                                    let maxY = wrapper.height() - 16;

                                                    if (y < 0) y = 0;
                                                    if (y > maxY) y = maxY;

                                                    let brightness = 1 - (y / maxY);
                                                    gradientObj.brightness = brightness;

                                                    // Compute actual RGB based on brightness
                                                    let r = Math.round(gradientObj.r * brightness);
                                                    let g = Math.round(gradientObj.g * brightness);
                                                    let b = Math.round(gradientObj.b * brightness);

                                                    gradientObj.el.css("background", `rgb(${r},${g},${b})`);

                                                    // Update dot position based on brightness if needed
                                                    if (!updatePosition) {
                                                        let newY = (1 - gradientObj.brightness) * maxY;
                                                        gradientObj.el.css("top", newY + "px");
                                                    }
                                                }

                                                function drawGradient() {
                                                    ctx.clearRect(0, 0, canvas.width, canvas.height);

                                                    let gradient = ctx.createLinearGradient(0, 0, canvas.width, 0);
                                                    let w = wrapper.width() - 16;

                                                    gradientColors.forEach(stop => {
                                                        let x = parseInt(stop.el.css("left"));
                                                        let pos = x / w;

                                                        let r = Math.round(stop.r * stop.brightness);
                                                        let g = Math.round(stop.g * stop.brightness);
                                                        let b = Math.round(stop.b * stop.brightness);

                                                        gradient.addColorStop(pos, `rgb(${r},${g},${b})`);
                                                    });

                                                    ctx.fillStyle = gradient;
                                                    ctx.fillRect(0, 0, canvas.width, canvas.height);
                                                }

                                                wrapper.on("click", ".stop", function (e) {
                                                    let gradientObj = gradientColors.find(s => s.el[0] === this);
                                                    selectedColor = gradientObj;

                                                    colorPicker.val(rgbToHex(gradientObj.r, gradientObj.g, gradientObj.b));

                                                    // Move input over the stop
                                                    let offset = $(this).offset();
                                                    colorPicker.css({
                                                        left: offset.left + "px",
                                                        top: offset.top + "px",
                                                        display: "block"
                                                    });

                                                    // Force browser to reflow
                                                    colorPicker[0].offsetWidth; // reading offsetWidth triggers reflow

                                                    // Now trigger the color picker
                                                    colorPicker.trigger("click");

                                                    colorPicker.off("input").on("input", function () {
                                                        let hex = $(this).val();
                                                        let rgb = hexToRgb(hex);

                                                        gradientObj.r = rgb.r;
                                                        gradientObj.g = rgb.g;
                                                        gradientObj.b = rgb.b;

                                                        applyBrightnessFromY(gradientObj, parseFloat(gradientObj.el.css("top")), false);
                                                        drawGradient();
                                                    });
                                                });

                                                function hexToRgb(hex) {
                                                    hex = hex.replace("#", "");
                                                    return {
                                                        r: parseInt(hex.substring(0, 2), 16),
                                                        g: parseInt(hex.substring(2, 4), 16),
                                                        b: parseInt(hex.substring(4, 6), 16)
                                                    };
                                                }

                                                function rgbStringToHex(rgb) {
                                                    let v = rgb.match(/\d+/g);
                                                    return rgbToHex(+v[0], +v[1], +v[2]);
                                                }

                                                function rgbToHex(r, g, b) {
                                                    return (
                                                        "#" +
                                                        ((1 << 24) + (r << 16) + (g << 8) + b)
                                                            .toString(16)
                                                            .slice(1)
                                                            .toUpperCase()
                                                    );
                                                }

                                                function sortGradientsByX() {
                                                    gradientColors.sort((a, b) => {
                                                        let ax = parseFloat(a.el.css("left"));
                                                        let bx = parseFloat(b.el.css("left"));
                                                        return ax - bx;
                                                    });
                                                }

                                                initGradientColors();

                                                modal.find('#addGradientColor').on('click', function () {
                                                    createGradientColor(0.5, 0, 255, 255, 1.0);
                                                    drawGradient();
                                                });

                                                modal.find('#deleteGradientColor').on('click', function () {
                                                    if (!selectedColor) return;
                                                    if (gradientColors.length <= 2) return;
                                                    selectedColor.el.remove();
                                                    gradientColors = gradientColors.filter(s => s !== selectedColor);
                                                    selectedColor = null;
                                                    drawGradient();
                                                });
                                            }

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

                                                if (data.gradients != null) {
                                                    const wrapper = $("#gradientWrapper");
                                                    let w = wrapper.width() - 16;
                                                    let output = {};

                                                    gradientColors.forEach((s, index) => {
                                                        let x = parseFloat(s.el.css("left"));
                                                        let pos = Number((x / w).toFixed(2));
                                                        let brightness = Number(s.brightness.toFixed(2));

                                                        output[index] = {
                                                            red: s.r,
                                                            green: s.g,
                                                            blue: s.b,
                                                            brightness: brightness,
                                                            position: pos,
                                                            Hex: rgbToHex(Math.round(s.r), Math.round(s.g), Math.round(s.b))
                                                        };
                                                    });
                                                    pf["colorZones"] = output;
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