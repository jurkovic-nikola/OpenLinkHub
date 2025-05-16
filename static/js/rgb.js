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


    // Init dataTable
    const dt = $('#table').DataTable(
        {
            order: [[1, 'asc']],
            select: {
                style: 'os',
                selector: 'td:first-child'
            },
            paging: false,
            searching: false,
            language: {
                emptyTable: "No selected devices. Select a device from left side"
            }
        }
    );

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
                    dt.clear();
                    $("#deviceId").val(deviceId);
                    $.each(response.data.profiles, function( index, value ) {
                        if (index === "keyboard" ||
                            index === "mouse" ||
                            index === "stand" ||
                            index === "mousepad" ||
                            index === "headset" ||
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
                        sp = '<input class="brightness-slider" type="range" id="speed_' + index + '" name="speedSlider" style="margin-top: 15px;" min="1" max="10" value="' + value.speed + '" step="0.1" />';
                        sm = '<input class="brightness-slider" type="range" id="smoothness_' + index + '" name="smoothnessSlider" style="margin-top: 15px;" min="0" max="100" value="' + value.smoothness + '" step="1" />';

                        switch (index) {
                            case "colorwarp": {
                                sc = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                ec = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "rainbow": {
                                sc = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                ec = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "watercolor": {
                                sc = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                ec = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "cpu-temperature": {
                                sp = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "gpu-temperature": {
                                sp = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "liquid-temperature": {
                                sp = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "circle": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "circleshift": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "colorpulse": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "colorshift": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "flickering": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "rotator": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "spinner": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "static": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                sp = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "storm": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                                sp = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                            case "wave": {
                                sm = '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;"></p>'
                            } break;
                        }

                        if (value.speed < 1) {
                            value.speed = 1
                        }
                        dt.row.add([
                            '<p class="text-md-start fw-normal mb-0" style="margin-top: 10px;">' + index + '</p>',
                            sc,
                            ec,
                            sp,
                            '<button class="btn btn-secondary saveRgbProfile" name="saveRgbProfile" id="' + index + '" style="float: right;"> <span>Save</span></button>',
                        ]).draw();
                    });

                    $('.saveRgbProfile').on('click', function(){
                        let startColorRgb = {}
                        let endColorRgb = {}

                        const deviceId =  $("#deviceId").val();
                        const profile = $(this).attr('id');
                        let speed = $("#speed_" + profile).val();
                        const startColorVal = $("#startColor_" + profile).val();
                        const endColorVal = $("#endColor_" + profile).val();

                        if (startColorVal == null) {
                            startColorRgb = {red:0, green:0, blue:0}
                        } else {
                            const startColor = hexToRgb(startColorVal);
                            startColorRgb = {red:startColor.r, green:startColor.g, blue:startColor.b}
                        }
                        if (endColorVal == null) {
                            endColorRgb = {red:0, green:0, blue:0}
                        } else {
                            const endColor = hexToRgb(endColorVal);
                            endColorRgb = {red:endColor.r, green:endColor.g, blue:endColor.b}
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
                }
            }
        });
    });
});