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

    // Left thumb stick slider update - X
    $('#leftThumbStickSensitivityValueX').on('input change', function() {
        $('#leftThumbStickSensitivityX').text($(this).val());
    });

    // Left thumb stick slider update - X
    $('#leftThumbStickSensitivityValueY').on('input change', function() {
        $('#leftThumbStickSensitivityY').text($(this).val());
    });

    // Right thumb stick slider update - X
    $('#rightThumbStickSensitivityValueX').on('input change', function() {
        $('#rightThumbStickSensitivityX').text($(this).val());
    });

    // Right thumb stick slider update - Y
    $('#rightThumbStickSensitivityValueY').on('input change', function() {
        $('#rightThumbStickSensitivityY').text($(this).val());
    });

    // Handle deadZoneMin sliders
    $('.deadZoneMinValue').on('input change', function() {
        // find the closest card-body and update the text inside it
        $(this).closest('.card-body').find('.deadZoneMin').text($(this).val());
    });

    // Handle deadZoneMax sliders
    $('.deadZoneMaxValue').on('input change', function() {
        $(this).closest('.card-body').find('.deadZoneMax').text($(this).val());
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

    $('#btnSaveLeftThumbstick').on('click', function () {
        const deviceId = $("#deviceId").val();
        const emulationMode = parseInt($("#leftThumbStickEmulationMode").val());
        const sensitivityX = parseInt($("#leftThumbStickSensitivityValueX").val());
        const sensitivityY = parseInt($("#leftThumbStickSensitivityValueY").val());
        const invertYAxis = $("#leftThumbStickInvertY").is(':checked');

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["emulationDevice"] = 0;
        pf["emulationMode"] = emulationMode;
        pf["sensitivityX"] = sensitivityX;
        pf["sensitivityY"] = sensitivityY;
        pf["invertYAxis"] = invertYAxis;

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/controller/emulation',
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

    $('#btnSaveRightThumbstick').on('click', function () {
        const deviceId = $("#deviceId").val();
        const emulationMode = parseInt($("#rightThumbStickEmulationMode").val());
        const sensitivityX = parseInt($("#rightThumbStickSensitivityValueX").val());
        const sensitivityY = parseInt($("#rightThumbStickSensitivityValueY").val());
        const invertYAxis = $("#rightThumbStickInvertY").is(':checked');

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["emulationDevice"] = 1;
        pf["emulationMode"] = emulationMode;
        pf["sensitivityX"] = sensitivityX;
        pf["sensitivityY"] = sensitivityY;
        pf["invertYAxis"] = invertYAxis;

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/controller/emulation',
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

    $('.sensitivityInfoToggle').on('click', function () {
        const modalElement = `
        <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="infoToggleLabel">Sensitivity X, Y</h5>
                        <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <span>This value is used only for Mouse emulation mode.</span>
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

    $(document).ready(function() {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/controller/getGraph',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {

                        $.each(response.data, function( index, value ) {
                            const graphName = 'graph_' + index;
                            const buttonName = 'btnSaveAnalogSettings_' + index;
                            renderCanvas(graphName, value, buttonName);
                        });
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });

    });

    function renderCanvas(canvasName, points, buttonName) {
        const maxValue = 100;
        function resizeCanvasToDisplaySize(canvas) {
            const rect = canvas.getBoundingClientRect();
            canvas.width = rect.width;
            canvas.height = rect.height;
        }

        const canvas = document.getElementById(canvasName);
        resizeCanvasToDisplaySize(canvas);
        const ctx = canvas.getContext('2d');
        ctx.clearRect(0, 0, canvas.width, canvas.height); // Clear any existing data

        const margin = 40;
        const width = canvas.width;
        const height = canvas.height;
        const graphWidth = width - 2 * margin;
        const graphHeight = height - 2 * margin;
        const state = {
            dragging: false,
            dragIndex: -1
        };

        function valToX(temp) {
            return margin + (temp / maxValue) * graphWidth;
        }

        function valToY(speed) {
            return height - margin - (speed / 100) * graphHeight;
        }

        function yToVal(y) {
            return Math.max(0, Math.min(100, ((height - margin - y) / graphHeight) * 100));
        }

        function draw() {
            ctx.clearRect(0, 0, width, height);

            ctx.strokeStyle = "#333";
            ctx.lineWidth = 1;
            ctx.font = "12px sans-serif";
            ctx.fillStyle = "#aaa";
            ctx.textAlign = "right";
            ctx.textBaseline = "middle";

            for (let i = 0; i <= 10; i++) {
                const val = i * 10;
                const y = valToY(val);
                ctx.beginPath();
                ctx.moveTo(margin, y);
                ctx.lineTo(width - margin, y);
                ctx.stroke();
                ctx.fillText(`${val}`, margin - 10, y);
            }

            ctx.textAlign = "center";
            ctx.textBaseline = "top";
            for (let i = 0; i <= 10; i++) {
                const val = i * 10;
                const x = valToX(val);
                ctx.beginPath();
                ctx.moveTo(x, height - margin);
                ctx.lineTo(x, margin);
                ctx.stroke();
                ctx.fillText(`${val}`, x, height - margin + 5);
            }

            ctx.strokeStyle = "#888";
            ctx.beginPath();
            ctx.moveTo(margin, margin);
            ctx.lineTo(margin, height - margin);
            ctx.lineTo(width - margin, height - margin);
            ctx.stroke();

            points.sort((a, b) => a.x - b.x);
            ctx.strokeStyle = "#42a5f5";
            ctx.lineWidth = 2;
            ctx.beginPath();
            points.forEach((p, i) => {
                if (p.x > 100) {
                    p.x = 100
                }
                const x = valToX(p.x);
                const y = valToY(p.y);
                if (i === 0) ctx.moveTo(x, y);
                else ctx.lineTo(x, y);
            });
            ctx.stroke();

            points.forEach(p => {
                const x = valToX(p.x);
                const y = valToY(p.y);
                ctx.fillStyle = "#42a5f5";
                ctx.beginPath();
                ctx.arc(x, y, 6, 0, Math.PI * 2);
                ctx.fill();
            });
        }

        function getMousePos(evt) {
            const rect = canvas.getBoundingClientRect();
            return {
                x: evt.clientX - rect.left,
                y: evt.clientY - rect.top
            };
        }

        function findNearbyPoint(mx, my) {
            return points.findIndex(p => {
                const dx = valToX(p.x) - mx;
                const dy = valToY(p.y) - my;
                return dx * dx + dy * dy < 100; // within 10px radius
            });
        }

        canvas.addEventListener("mousedown", (e) => {
            const { x, y } = getMousePos(e);
            const index = findNearbyPoint(x, y);

            if (index > 0 && index < points.length - 1) {
                state.dragging = true;
                state.dragIndex = index;
            }
        });

        canvas.addEventListener("mousemove", (e) => {
            if (!state.dragging) return;
            const { y } = getMousePos(e);
            const speed = yToVal(y);

            // Keep the same X, only update Y
            const currentPoint = points[state.dragIndex];
            points[state.dragIndex] = { x: currentPoint.x, y: Math.round(speed) };

            draw();
        });

        canvas.addEventListener("mouseup", () => {
            state.dragging = false;
            state.dragIndex = -1;
        });

        canvas.addEventListener("contextmenu", (e) => {
            e.preventDefault(); // Disable default right-click menu
            const { x, y } = getMousePos(e);
            const index = findNearbyPoint(x, y);
            if (index !== -1) {
                points.splice(index, 1); // Remove the point
                draw(); // Redraw graph
            }
        });
        draw();

        // Button cleanup
        const button = document.getElementById(buttonName);
        if (button._clickListener) {
            button.removeEventListener("click", button._clickListener);
        }

        button._clickListener = function () {
            button.disabled = true;

            let capturedPoints = points.map(p => ({ ...p }));
            const key = this.dataset.info; // <-- grab data-info value
            const deviceId = $("#deviceId").val();
            const deadZoneMin = $("#deadZoneMinValue_" + key).val();
            const deadZoneMax = $("#deadZoneMaxValue_" + key).val();
            const pf = {};
            pf["deviceId"] = deviceId;
            pf["analogDevice"] = parseInt(key);
            pf["curveData"] = capturedPoints;
            pf["deadZoneMin"] = parseInt(deadZoneMin);
            pf["deadZoneMax"] = parseInt(deadZoneMax);

            const json = JSON.stringify(pf, null, 2);
            $.ajax({
                url: '/api/controller/setGraph',
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
                },
                error: function() {
                    toast.error("An error occurred while sending data.");
                },
                complete: function() {
                    button.disabled = false;
                }
            });
        };

        button.addEventListener("click", button._clickListener);
    }

    $('.btnDataCurvePreset').on('click', function () {
        const keyIndex = $(this).attr("data-info").split(';');
        const graphId = parseInt(keyIndex[0]);
        const preSetId = parseInt(keyIndex[1]);

        let points = [
            { x: 0, y: 0 },
            { x: 20, y: 20 },
            { x: 40, y: 40 },
            { x: 60, y: 60 },
            { x: 80, y: 80 },
            { x: 100, y: 100 }
        ];
        switch (preSetId) {
            case 0:
                points = [
                    { x: 0, y: 0 },
                    { x: 20, y: 20 },
                    { x: 40, y: 40 },
                    { x: 60, y: 60 },
                    { x: 80, y: 80 },
                    { x: 100, y: 100 }
                ];
                break;
            case 1:
                points = [
                    { x: 0, y: 0 },
                    { x: 20, y: 35 },
                    { x: 40, y: 45 },
                    { x: 60, y: 55 },
                    { x: 80, y: 70 },
                    { x: 100, y: 100 }
                ];
                break;
            case 2:
                points = [
                    { x: 0, y: 0 },
                    { x: 20, y: 5 },
                    { x: 40, y: 15 },
                    { x: 60, y: 30 },
                    { x: 80, y: 60 },
                    { x: 100, y: 100 }
                ];
                break;
            case 3:
                points = [
                    { x: 0, y: 0 },
                    { x: 20, y: 40 },
                    { x: 40, y: 70 },
                    { x: 60, y: 85 },
                    { x: 80, y: 95 },
                    { x: 100, y: 100 }
                ];
                break;
        }
        const graphName = 'graph_' + graphId;
        const buttonName = 'btnSaveAnalogSettings_' + graphId;
        renderCanvas(graphName, points, buttonName);
    });
});