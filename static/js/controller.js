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

    $('.defaultInfoToggle').on('click', function () {
        const modalElement = `
        <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="infoToggleLabel">Mouse Default Action</h5>
                        <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <span>When enabled, the controller performs its default key action. This checkbox ignores all user custom assignments.</span>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                    </div>
                </div>
            </div>
        </div>`;
        const modal = $(modalElement).modal('toggle');
        modal.on('hidden.bs.modal', function () {
            modal.data('bs.modal', null);
        })
    });

    $('.pressAndHoldInfoToggle').on('click', function () {
        const modalElement = `
        <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="infoToggleLabel">Press and Hold</h5>
                        <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <span>When enabled, the controller continuously sends action until the button is released.</span>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                    </div>
                </div>
            </div>
        </div>`;
        const modal = $(modalElement).modal('toggle');
        modal.on('hidden.bs.modal', function () {
            modal.data('bs.modal', null);
        })
    });

    $('.keyAssignmentType').on('change', function () {
        const selectedValue = parseInt($(this).val());
        const indexElements = $(this).attr("id").split('_')
        const keyAssignmentValue = $("#keyAssignmentValue_" + indexElements[1]);

        switch (selectedValue) {
            case 0: {
                $(keyAssignmentValue).empty();
                $(keyAssignmentValue).append($('<option>', { value: 0, text: "None" }));
            }
                break;

            case 1: { // Media keys
                $.ajax({
                    url:'/api/input/media',
                    type:'get',
                    success:function(result){
                        $(keyAssignmentValue).empty();
                        $.each(result.data, function( index, value ) {
                            $(keyAssignmentValue).append($('<option>', { value: index, text: value.Name }));
                        });
                    }
                });
            }
                break;

            case 2: { // DPI
                $(keyAssignmentValue).empty();
                $(keyAssignmentValue).append($('<option>', { value: 0, text: "None" }));
            }
                break;

            case 3: { // Keyboard
                $.ajax({
                    url:'/api/input/keyboard',
                    type:'get',
                    success:function(result){
                        $(keyAssignmentValue).empty();
                        $.each(result.data, function( index, value ) {
                            $(keyAssignmentValue).append($('<option>', { value: index, text: value.Name }));
                        });
                    }
                });
            }
                break;

            case 8: { // Sniper
                $(keyAssignmentValue).empty();
                $(keyAssignmentValue).append($('<option>', { value: 0, text: "None" }));
            }
                break;

            case 9: { // Mouse
                $.ajax({
                    url:'/api/input/mouse',
                    type:'get',
                    success:function(result){
                        $(keyAssignmentValue).empty();
                        $.each(result.data, function( index, value ) {
                            $(keyAssignmentValue).append($('<option>', { value: index, text: value.Name }));
                        });
                    }
                });
            }
                break;

            case 10: { // Macro
                $.ajax({
                    url:'/api/macro/',
                    type:'get',
                    success:function(result){
                        $(keyAssignmentValue).empty();
                        $.each(result.data, function( index, value ) {
                            $(keyAssignmentValue).append($('<option>', { value: index, text: value.name }));
                        });
                    }
                });
            }
                break;
        }
    });

    $('.saveKeyAssignment').on('click', function () {
        const deviceId = $("#deviceId").val();
        const keyIndex = $(this).attr("data-info");
        const enabled = $("#default_" + keyIndex).is(':checked');
        const pressAndHold = $("#pressAndHold_" + keyIndex).is(':checked');
        const keyAssignmentType = $("#keyAssignmentType_" + keyIndex).val();
        const keyAssignmentValue = $("#keyAssignmentValue_" + keyIndex).val();

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyIndex"] = parseInt(keyIndex);
        pf["enabled"] = enabled;
        pf["pressAndHold"] = pressAndHold;
        pf["keyAssignmentType"] = parseInt(keyAssignmentType);
        pf["keyAssignmentValue"] = parseInt(keyAssignmentValue);
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/controller/updateKeyAssignment',
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

    $('.controllerSleepModes').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["sleepMode"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/headset/sleep',
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