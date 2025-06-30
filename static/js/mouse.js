"use strict";

document.addEventListener("DOMContentLoaded", function () {
    const dpiStageAmount = 5;
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
        for (let i = 0; i <= dpiStageAmount; i++) {
            const stage = document.getElementById("stage" + i);
            if (stage == null) {
                continue
            }

            const stageValue = document.getElementById("stageValue" + i);
            stage.oninput = function() {
                stageValue.value = this.value;
            }
        }
    });

    $('#defaultDPI').on('click', function () {
        for (let i = 0; i <= dpiStageAmount; i++) {
            const stage = document.getElementById("stage" + i);
            if (stage == null) {
                continue
            }

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
        for (let i = 0; i <= dpiStageAmount; i++) {
            const stage = document.getElementById("stage" + i);
            if (stage == null) {
                continue
            }

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
                case 5:
                    stage.value = 200
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
        for (let i = 0; i <= dpiStageAmount; i++) {
            const stage = $("#stageValue" + i).val();
            if (stage == null) {
                continue
            }

            stages[i] =  parseInt(stage);
        }

        pf["stages"] = stages;
        const json = JSON.stringify(pf, null, 2);
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

    $('.mousePollingRate').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["pollingRate"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $('.mousePollingRate').prop('disabled', true);
        $.ajax({
            url: '/api/mouse/pollingRate',
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
                $('.mousePollingRate').prop('disabled', false);
            }
        });
    });

    $('.mouseAngleSnapping').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["angleSnapping"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $('.mouseAngleSnapping').prop('disabled', true);
        $.ajax({
            url: '/api/mouse/angleSnapping',
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
                $('.mouseAngleSnapping').prop('disabled', false);
            }
        });
    });

    $('.mouseButtonOptimization').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["buttonOptimization"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $('.mouseButtonOptimization').prop('disabled', true);
        $.ajax({
            url: '/api/mouse/buttonOptimization',
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
                $('.mouseButtonOptimization').prop('disabled', false);
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
                        <span>When enabled, the mouse performs its default key action. This checkbox ignores all user custom assignments.</span>
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
                        <span>When enabled, the mouse continuously sends action until the button is released.</span>
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
            url: '/api/mouse/updateKeyAssignment',
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