"use strict";

document.addEventListener("DOMContentLoaded", function () {
    // Global maximum amount of DPI stages
    const dpiStageAmount = 5;

    function clamp(value, min, max) {
        return Math.min(Math.max(value, min), max);
    }

    function hexToRgb(hex) {
        const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
        return result ? {
            r: parseInt(result[1], 16),
            g: parseInt(result[2], 16),
            b: parseInt(result[3], 16)
        } : null;
    }

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
        const $btn = $(this);
        $btn.prop('disabled', true);

        const deviceId = $("#deviceId").val();
        const pf = {};
        let stages = {};

        pf["deviceId"] = deviceId;

        for (let i = 0; i <= dpiStageAmount; i++) {
            const stage = $("#stageValue" + i).val();
            if (stage == null) continue;
            stages[i] = parseInt(stage);
        }
        pf["stages"] = stages;

        $.ajax({
            url: '/api/mouse/dpi',
            type: 'POST',
            data: JSON.stringify(pf),
            contentType: 'application/json',
            cache: false,
            success: function (response) {
                if (response?.status === 1) {
                    toast.success(response.message);
                } else {
                    toast.warning(response.message);
                }
            },
            error: function () {
                toast.error("Request failed");
            },
            complete: function () {
                $btn.prop('disabled', false);
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

    $('#saveZoneColors').on('click', function () {
        const deviceId = $("#deviceId").val();
        const zones = parseInt($("#zones").val());

        let colors = {};
        for (let i = 0; i < zones; i++) {
            const $zoneColor = $("#zoneColor"+i).val();
            if (!$zoneColor.length) continue;

            const zoneColorRgb = hexToRgb($zoneColor);
            colors[i] = {red: zoneColorRgb.r, green: zoneColorRgb.g, blue: zoneColorRgb.b}
        }

        const dpiColor = $("#dpiColor").val();
        const sniperColor = $("#sniperColor").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["colorZones"] = colors

        if (dpiColor != null) {
            const dpiColorRgb = hexToRgb(dpiColor);
            pf["colorDpi"] = {red:dpiColorRgb.r, green:dpiColorRgb.g, blue:dpiColorRgb.b}
        }

        if (sniperColor != null) {
            const sniperColorRgb = hexToRgb(sniperColor);
            pf["colorSniper"] = {red:sniperColorRgb.r, green:sniperColorRgb.g, blue:sniperColorRgb.b}
            pf["isSniper"] = true
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

    $('.defaultInfoToggle').on('click', function () {
        const modalElement = `
        <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtMouseDefaultAction')}</h5>
                        <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <span>${i18n.t('txtMouseDefaultActionInfo')}</span>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
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
                        <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtPressAndHold')}</h5>
                        <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <span>${i18n.t('txtPressAndHoldMouse')}</span>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                    </div>
                </div>
            </div>
        </div>`;
        const modal = $(modalElement).modal('toggle');
        modal.on('hidden.bs.modal', function () {
            modal.data('bs.modal', null);
        })
    });

    $('.onReleaseInfoToggle').on('click', function () {
        const modalElement = `
        <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtOnRelease')}</h5>
                        <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <span>${i18n.t('txtOnReleaseInfo')}</span>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
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
        const $btn = $(this);
        const deviceId = $("#deviceId").val();
        const keyIndex = $(this).attr("data-info");
        const enabled = $("#default_" + keyIndex).is(':checked');
        const onRelease = $("#onRelease_" + keyIndex).is(':checked');
        const pressAndHold = $("#pressAndHold_" + keyIndex).is(':checked');
        const keyAssignmentType = $("#keyAssignmentType_" + keyIndex).val();
        const keyAssignmentValue = $("#keyAssignmentValue_" + keyIndex).val();

        if (onRelease === true && pressAndHold === true) {
            toast.warning(i18n.t('txtPressAndHoldBlocked'));
            return false;
        }

        // Disable button immediately
        $btn.prop('disabled', true);

        const pf = {
            deviceId: deviceId,
            keyIndex: parseInt(keyIndex),
            enabled: enabled,
            pressAndHold: pressAndHold,
            keyAssignmentType: parseInt(keyAssignmentType),
            keyAssignmentValue: parseInt(keyAssignmentValue),
            onRelease: onRelease
        };

        $.ajax({
            url: '/api/mouse/updateKeyAssignment',
            type: 'POST',
            data: JSON.stringify(pf),
            cache: false,
            success: function (response) {
                if (response?.status === 1) {
                    toast.success(response.message);
                } else {
                    toast.warning(response.message);
                }
            },
            error: function () {
                toast.error('Request failed');
            },
            complete: function () {
                $btn.prop('disabled', false);
            }
        });
    });

    $(".toggleAngleSnapping").on("change", function () {
        const $toggle = $(this);
        const previousState = !$toggle.prop("checked");
        const newState = $toggle.prop("checked");
        const deviceId = $("#deviceId").val();

        $toggle.prop("disabled", true);

        $.ajax({
            url: "/api/mouse/angleSnapping",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({
                deviceId: deviceId,
                angleSnapping: newState ? 1 : 0
            }),
            success(response) {
                if (response?.status !== 1) {
                    $toggle.prop("checked", previousState);
                    toast.warning(response?.message || "Operation failed");
                } else {
                    toast.success(response?.message || "Operation failed");
                }
            },
            error() {
                $toggle.prop("checked", previousState);
                toast.warning("Request failed");
            },
            complete() {
                $toggle.prop("disabled", false);
            }
        });
    });

    $(".toggleButtonOptimization").on("change", function () {
        const $toggle = $(this);
        const previousState = !$toggle.prop("checked");
        const newState = $toggle.prop("checked");
        const deviceId = $("#deviceId").val();

        $toggle.prop("disabled", true);

        $.ajax({
            url: "/api/mouse/buttonOptimization",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({
                deviceId: deviceId,
                buttonOptimization: newState ? 1 : 0
            }),
            success(response) {
                if (response?.status !== 1) {
                    $toggle.prop("checked", previousState);
                    toast.warning(response?.message || "Operation failed");
                } else {
                    toast.success(response?.message || "Operation failed");
                }
            },
            error() {
                $toggle.prop("checked", previousState);
                toast.warning("Request failed");
            },
            complete() {
                $toggle.prop("disabled", false);
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

    $(".toggleLeftHandMode").on("change", function () {
        const $toggle = $(this);
        const previousState = !$toggle.prop("checked");
        const newState = $toggle.prop("checked");
        const deviceId = $("#deviceId").val();

        $toggle.prop("disabled", true);

        $.ajax({
            url: "/api/mouse/leftHandMode",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({
                deviceId: deviceId,
                leftHandMode: newState ? 1 : 0
            }),
            success(response) {
                if (response?.status !== 1) {
                    $toggle.prop("checked", previousState);
                    toast.warning(response?.message || "Operation failed");
                } else {
                    location.reload();
                }
            },
            error() {
                $toggle.prop("checked", previousState);
                toast.warning("Request failed");
            },
            complete() {
                $toggle.prop("disabled", false);
            }
        });
    });

    $(".mouseLiftHeight").on("change", function () {
        const value = $(this).val();
        const deviceId = $("#deviceId").val();
        
        $.ajax({
            url: "/api/mouse/liftHeight",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({
                deviceId: deviceId,
                liftHeight: parseInt(value)
            }),
            success(response) {
                if (response?.status !== 1) {
                    toast.warning(response?.message || "Operation failed");
                } else {
                    toast.success(response?.message || "Operation failed");
                }
            },
            error() {
                toast.warning("Request failed");
            }
        });
    });

    $('#saveGestures').on('click', function () {
        const $btn = $(this);
        const deviceId = $("#deviceId").val();
        const multiGestures = $(".toggleMultiGestures").prop("checked");
        const pf = {};
        let zoneTilts = {};

        pf["deviceId"] = deviceId;
        $btn.prop('disabled', true);
        for (let i = 0; i < 4; i++) {
            const tiltValue = $("#zoneTilt" + i).val();
            if (tiltValue == null) {
                continue
            }

            zoneTilts[i] =  parseInt(tiltValue);
        }

        $.ajax({
            url: '/api/mouse/gestures',
            type: 'POST',
            contentType: 'application/json',
            data: JSON.stringify({
                deviceId: deviceId,
                multiGestures: multiGestures ? 1 : 0,
                zoneTilts: zoneTilts
            }),
            cache: false,
            success: function (response) {
                if (response?.status === 1) {
                    toast.success(response.message);
                } else {
                    toast.warning(response.message);
                }
            },
            error: function () {
                toast.error('Request failed');
            },
            complete: function () {
                $btn.prop('disabled', false);
            }
        });
    });

    function updateDPISlider(el) {
        const $slider = $(el);
        const min = Number($slider.attr("min"));
        const max = Number($slider.attr("max"));
        const value = Number($slider.val());
        const percent = ((value - min) / (max - min)) * 100;
        $slider.css("--slider-progress", percent + "%");
    }

    function updateTiltSlider(el) {
        const $slider = $(el);
        const min = Number($slider.attr("min"));
        const max = Number($slider.attr("max"));
        const value = Number($slider.val());
        const percent = ((value - min) / (max - min)) * 100;
        $slider.css("--slider-progress", percent + "%");
    }

    $(".dpiSlider").each(function () {
        const index = this.id.replace("stage", "");
        $("#stageValue" + index).val(this.value);
        updateDPISlider(this);
    }).on("input", function () {
        const index = this.id.replace("stage", "");
        $("#stageValue" + index).val(this.value);
        updateDPISlider(this);
    });

    $(".tiltSlider").each(function () {
        const index = this.id.replace("zoneTilt", "");
        $("#zoneTiltValue" + index).text(this.value + ' °');
        updateTiltSlider(this);
    }).on("input", function () {
        const index = this.id.replace("zoneTilt", "");
        $("#zoneTiltValue" + index).text(this.value + ' °');
        updateTiltSlider(this);
    });

    function finalizeDPIInput(input) {
        const index = input.id.replace("stageValue", "");
        const $slider = $("#stage" + index);
        if (!$slider.length) return;

        let value = Number(input.value);
        if (isNaN(value)) value = Number($slider.attr("min"));

        const min = Number($slider.attr("min"));
        const max = Number($slider.attr("max"));

        value = clamp(value, min, max);

        input.value = value;
        $slider.val(value);
        updateDPISlider($slider[0]);
    }

    $(".system-input input[type='text']").on("blur", function () {
        finalizeDPIInput(this);
    });

    $(".system-input input[type='text']").on("keydown", function (e) {
        if (e.key === "Enter") {
            e.preventDefault();
            finalizeDPIInput(this);
            this.blur(); // optional, but feels right
        }
    });
});