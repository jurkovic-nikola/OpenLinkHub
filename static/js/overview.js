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
    let globalKeyId = 0;

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

    const keySelector = document.querySelectorAll('.keySelector');
    keySelector.forEach(div => {
        div.addEventListener('click', () => {
            keySelector.forEach(s => s.classList.remove('active'));
            div.classList.add('active');
        });
    });

    function fetchAssignmentTypes(deviceId, selectedType, callback) {
        $.ajax({
            url: '/api/keyboard/assignmentsTypes/' + deviceId,
            type: 'GET',
            cache: false,
            success: function(response) {
                let optionTypes = '';
                $.each(response.data, function(key, value) {
                    optionTypes += `<option value="${key}" ${parseInt(selectedType) === parseInt(key) ? 'selected' : ''}>${value}</option>`;
                });
                callback(optionTypes);
            }
        });
    }

    function fetchAssignmentModifiers(deviceId, selectedType, callback) {
        $.ajax({
            url: '/api/keyboard/assignmentsModifiers/' + deviceId,
            type: 'GET',
            cache: false,
            success: function(response) {
                let optionTypes = '';
                $.each(response.data, function(key, value) {
                    optionTypes += `<option value="${key}" ${parseInt(selectedType) === parseInt(key) ? 'selected' : ''}>${value}</option>`;
                });
                callback(optionTypes);
            }
        });
    }


    $('.keyboardPerformance').on('click', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/getPerformance/' + deviceId,
            type: 'GET',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        const data = response.data;
                        let modalElement = `
                          <div class="modal fade text-start" id="keyboardPerformance" tabindex="-1" aria-labelledby="keyboardPerformance">
                            <div class="modal-dialog">
                              <div class="modal-content">
                                <div class="modal-header">
                                  <h5 class="modal-title" id="keyboardPerformance">Keyboard Performance</h5>
                                  <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                </div>
                                <div class="modal-body">
                                  <form>
                                    <div class="mb-3">
                                        <table class="table mb-0">
                                            <thead>
                                            <tr>
                                                <th style="text-align: left;">When Win Lock is ON</th>
                                                <th style="text-align: right;"></th>
                                            </tr>
                                            </thead>
                                            <tbody>
                                            </tbody>
                                        </table>
                                    </div>
                                  </form>
                                </div>
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveKeyboardPerformance">Save</button>
                                </div>
                              </div>
                            </div>
                          </div>
                        `;
                        const modal = $(modalElement).modal('toggle');

                        $.each(data, function( index, value ) {
                            let element ='';
                            switch (value.type) {
                                case 'checkbox':
                                    const isChecked = value.value === true ? 'checked' : '';
                                    element = '<input id="' + value.internal + '" type="checkbox" ' + isChecked + ' />';
                                    break;
                            }
                            var newRow = `
                                <tr>
                                    <th scope="row" style="text-align: left;">${value.name}</th>
                                    <td style="text-align: right;">${element}</td>
                                </tr>
                            `;
                            modal.find('.table tbody').append(newRow);
                        });

                        modal.on('hidden.bs.modal', function () {
                            modal.data('bs.modal', null);
                            modal.remove();
                        })

                        modal.on('shown.bs.modal', function (e) {
                            modal.find('#btnSaveKeyboardPerformance').on('click', function () {
                                const pf = {};
                                pf["deviceId"] = deviceId;

                                $.each(data, function( index, value ) {
                                    switch (value.type) {
                                        case 'checkbox':
                                            const val = modal.find("#" + value.internal).is(':checked');
                                            pf[value.internal] = val
                                            break;
                                    }
                                });
                                const json = JSON.stringify(pf, null, 2);
                                $.ajax({
                                    url: '/api/keyboard/setPerformance',
                                    type: 'POST',
                                    data: json,
                                    cache: false,
                                    success: function(response) {
                                        try {
                                            if (response.status === 1) {
                                                const modalElement = $("#keyboardPerformance");
                                                $(modalElement).modal('hide');
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

    $('.controlDialColors').on('click', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/dial/getColors/' + deviceId,
            type: 'GET',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        const data = response.data;
                        let modalElement = `
                          <div class="modal fade text-start" id="keyboardControlDial" tabindex="-1" aria-labelledby="keyboardControlDial">
                            <div class="modal-dialog">
                              <div class="modal-content">
                                <div class="modal-header">
                                  <h5 class="modal-title" id="keyboardControlDial">Control Dial Colors</h5>
                                  <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                </div>
                                <div class="modal-body">
                                  <form>
                                    <div class="mb-3">
                                        <table class="table mb-0">
                                            <thead>
                                            <tr>
                                                <th style="text-align: left;">Option</th>
                                                <th style="text-align: right;">Color</th>
                                            </tr>
                                            </thead>
                                            <tbody>
                                            </tbody>
                                        </table>
                                    </div>
                                  </form>
                                </div>
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveControlDialColors">Save</button>
                                </div>
                              </div>
                            </div>
                          </div>
                        `;
                        const modal = $(modalElement).modal('toggle');

                        $.each(data, function( index, value ) {
                            const color = rgbToHex(value.Color.red, value.Color.green, value.Color.blue);
                            const optionId = value.Id;
                            var newRow = `
                                <tr>
                                    <th scope="row" style="text-align: left;">${value.Name}</th>
                                    <td style="text-align: right;"><input type="color" id="dial-color-${optionId}" value="${color}"></td>
                                </tr>
                            `;
                            modal.find('.table tbody').append(newRow);
                        });

                        modal.on('hidden.bs.modal', function () {
                            modal.data('bs.modal', null);
                            modal.remove();
                        })

                        modal.on('shown.bs.modal', function (e) {
                            modal.find('#btnSaveControlDialColors').on('click', function () {
                                const colorZones = {};
                                $.each(data, function(index, value) {
                                    let color = modal.find("#dial-color-" + value.Id).val();
                                    color = hexToRgb(color);
                                    colorZones[value.Id] = {red: color.r, green: color.g, blue: color.b};
                                });

                                pf["colorZones"] = colorZones;
                                const json = JSON.stringify(pf, null, 2);

                                $.ajax({
                                    url: '/api/keyboard/dial/setColors',
                                    type: 'POST',
                                    data: json,
                                    cache: false,
                                    success: function(response) {
                                        try {
                                            if (response.status === 1) {
                                                const modalElement = $("#keyboardControlDial");
                                                $(modalElement).modal('hide');
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

    function noColorChange(deviceId, keyId) {
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyId"] = parseInt(keyId);
        const json = JSON.stringify(pf, null, 2);

        return new Promise((noChange, allowChange) => {
            $.ajax({
                url: '/api/keyboard/getKey/',
                type: 'POST',
                data: json,
                cache: false,
                success: function(response) {
                    try {
                        if (response.status === 1 && response.data.noColor === true) {
                            noChange(true);
                        } else {
                            noChange(false);
                        }
                    } catch (err) {
                        noChange(false);
                    }
                },
                error: function() {
                    noChange(false);
                }
            });
        });
    }

    function noKeyAssignments(deviceId, keyId) {
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyId"] = parseInt(keyId);
        const json = JSON.stringify(pf, null, 2);

        return new Promise((noChange, allowChange) => {
            $.ajax({
                url: '/api/keyboard/getKey/',
                type: 'POST',
                data: json,
                cache: false,
                success: function(response) {
                    try {
                        if (response.status === 1 && response.data.functionKey === true) {
                            noChange(true);
                        } else {
                            noChange(false);
                        }
                    } catch (err) {
                        noChange(false);
                    }
                },
                error: function() {
                    noChange(false);
                }
            });
        });
    }

    $('.device-selectable').click(function (e) {
        if ($(e.target).closest('button, select, input, .newLabel, .newRgbLabel').length > 0) {
            return;
        }

        $(this).toggleClass('device-selected');

        const deviceSelected = $('.device-selectable.device-selected').map(function () {
            return $(this).data('info');
        }).get();

        $('#selectedDevices').val(
            deviceSelected.length ? deviceSelected.join(',') : ''
        );
    });

    $('.openKeyAssignments').on('click', function () {
        if (globalKeyId === 0) {
            toast.warning('Select a valid key');
            return false;
        }

        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyId"] = parseInt(globalKeyId);
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/getKey/',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        let data = response.data;
                        if (data.onlyColor === true) {
                            toast.warning('This object does not support Key Assignments');
                            return false;
                        }

                        let defaultCheckbox ='';
                        if (data.default === true) {
                            defaultCheckbox = '<input id="default" type="checkbox" checked/>';
                        } else {
                            defaultCheckbox = '<input id="default" type="checkbox"/>';
                        }

                        let holdCheckbox ='';
                        if (data.actionHold === true) {
                            holdCheckbox = '<input id="pressAndHold" type="checkbox" checked/>';
                        } else {
                            holdCheckbox = '<input id="pressAndHold" type="checkbox"/>';
                        }

                        let toggleDelayInput = '<input id="toggleDelay" type="text" value="' + data.toggleDelay + '"/>';

                        let modalElement = `
                          <div class="modal fade text-start" id="setupKeyAssignments" tabindex="-1" aria-labelledby="setupKeyAssignments">
                            <div class="modal-dialog modal-dialog-1000">
                              <div class="modal-content" style="width: 1000px;">
                                <div class="modal-header">
                                  <h5 class="modal-title" id="setupKeyAssignments">Setup Key Assignment - ${data.keyName}</h5>
                                  <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                </div>
                                <div class="modal-body">
                                  <form>
                                    <div class="mb-3">
                                        <table class="table mb-0">
                                            <thead>
                                            <tr>
                                                <th style="text-align: left;">Key</th>
                                                <th>
                                                    Default
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy defaultInfoToggle"></i>
                                                </th>
                                                <th>
                                                    Press and Hold / Toggle
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy pressAndHoldInfoToggle"></i>
                                                </th>
                                                <th>
                                                    Toggle Delay (ms)
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy toggleDelayInfoToggle"></i>
                                                </th>
                                                <th>Type</th>
                                                <th>Value</th>
                                            </tr>
                                            </thead>
                                            <tbody>
                                            <tr>
                                                <th scope="row" style="text-align: left;">${data.keyName}</th>
                                                <td>${defaultCheckbox}</td>
                                                <td>${holdCheckbox}</td>
                                                <td><input class="form-control" id="toggleDelay" type="text" value="${data.toggleDelay}" style="width: 100px;"/></td>
                                                <td><select class="form-select keyAssignmentType" id="keyAssignmentType"></select></td>
                                                <td><select class="form-select" id="keyAssignmentValue"></select></td>
                                            </tr>
                                            </tbody>
                                        </table>
                                    </div>
                                  </form>
                                </div>
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveKeyAssignments">Save</button>
                                </div>
                              </div>
                            </div>
                          </div>
                        `;
                        const modal = $(modalElement).modal('toggle');
                        const keyAssignmentValue = modal.find("#keyAssignmentValue");

                        // Fetch assignment types
                        fetchAssignmentTypes(deviceId, data.actionType, function(optionTypes) {
                            modal.find('#keyAssignmentType').html(optionTypes);
                        });

                        if (parseInt(data.actionType) === 0) {
                            $(keyAssignmentValue).empty();
                            $(keyAssignmentValue).append($('<option>', { value: 0, text: "None" }));
                        } else {
                            let url = '';
                            switch (data.actionType) {
                                case 1: {
                                    url = '/api/input/media';
                                }
                                    break;
                                case 3: {
                                    url = '/api/input/keyboard';
                                }
                                    break;
                                case 9: {
                                    url = '/api/input/mouse';
                                }
                                    break;
                                case 10: {
                                    url = '/api/macro/';
                                }
                                    break;
                            }

                            $.ajax({
                                url:url,
                                type:'get',
                                success:function(result){
                                    $(keyAssignmentValue).empty();
                                    $.each(result.data, function( index, value ) {
                                        const displayName = value.Name || value.name;
                                        $(keyAssignmentValue).append($('<option>', { value: index, text: displayName, selected: parseInt(index) === parseInt(data.actionCommand) }));
                                    });
                                }
                            });
                        }

                        modal.find('#keyAssignmentType').on('change', function () {
                            const selectedValue = parseInt($(this).val());
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

                        modal.find('.defaultInfoToggle').on('click', function () {
                            const modalDefault = `
                                <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel">
                                    <div class="modal-dialog">
                                        <div class="modal-content">
                                            <div class="modal-header">
                                                <h5 class="modal-title" id="infoToggleLabel">Keyboard Default Action</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>When enabled, the keyboard performs its default key action. This checkbox ignores all user custom assignments.</span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            `;
                            const infoDefault = $(modalDefault).modal('toggle');
                            infoDefault.on('hidden.bs.modal', function () {
                                infoDefault.data('bs.modal', null);
                            })
                        });

                        modal.find('.pressAndHoldInfoToggle').on('click', function () {
                            const modalPressAndHold = `
                                <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel">
                                    <div class="modal-dialog">
                                        <div class="modal-content">
                                            <div class="modal-header">
                                                <h5 class="modal-title" id="infoToggleLabel">Press and Hold</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>
                                                <b>Press and Hold:</b><br />When enabled, the keyboard continuously sends an action until the key is released.<br /><br />
                                                <b>Toggle:</b><br /> Used only for the mouse Key Assignment type. When enabled, the action is repeated until the key is pressed again.
                                                </span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            `;
                            const infoPressAndHold = $(modalPressAndHold).modal('toggle');
                            infoPressAndHold.on('hidden.bs.modal', function () {
                                infoPressAndHold.data('bs.modal', null);
                            })
                        });

                        modal.find('.toggleDelayInfoToggle').on('click', function () {
                            const modalPressAndHold = `
                                <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel">
                                    <div class="modal-dialog">
                                        <div class="modal-content">
                                            <div class="modal-header">
                                                <h5 class="modal-title" id="infoToggleLabel">Toggle Delay</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>
                                                <b>Toggle Delay:</b><br /> Used only for the mouse Key Assignment type. When enabled, the action repeat is delayed by the defined period of time.
                                                </span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            `;
                            const infoPressAndHold = $(modalPressAndHold).modal('toggle');
                            infoPressAndHold.on('hidden.bs.modal', function () {
                                infoPressAndHold.data('bs.modal', null);
                            })
                        });

                        modal.on('hidden.bs.modal', function () {
                            modal.data('bs.modal', null);
                            modal.remove();
                        })

                        modal.on('shown.bs.modal', function (e) {
                            modal.find('#btnSaveKeyAssignments').on('click', function () {
                                const enabled = modal.find("#default").is(':checked');
                                const pressAndHold = modal.find("#pressAndHold").is(':checked');
                                const keyAssignmentType = modal.find("#keyAssignmentType").val();
                                const keyAssignmentValue = modal.find("#keyAssignmentValue").val();
                                const toggleDelay = modal.find("#toggleDelay").val();

                                const pf = {};
                                pf["deviceId"] = deviceId;
                                pf["keyIndex"] = parseInt(globalKeyId);
                                pf["enabled"] = enabled;
                                pf["pressAndHold"] = pressAndHold;
                                pf["keyAssignmentType"] = parseInt(keyAssignmentType);
                                pf["keyAssignmentValue"] = parseInt(keyAssignmentValue);
                                pf["toggleDelay"] = parseInt(toggleDelay);

                                const json = JSON.stringify(pf, null, 2);
                                $.ajax({
                                    url: '/api/keyboard/updateKeyAssignment',
                                    type: 'POST',
                                    data: json,
                                    cache: false,
                                    success: function(response) {
                                        try {
                                            if (response.status === 1) {
                                                const modalElement = $("#setupKeyAssignments");
                                                $(modalElement).modal('hide');
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

    $('.openKeyAssignmentsWithModifier').on('click', function () {
        if (globalKeyId === 0) {
            toast.warning('Select a valid key');
            return false;
        }

        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyId"] = parseInt(globalKeyId);
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/getKey/',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        let data = response.data;
                        if (data.onlyColor === true) {
                            toast.warning('This object does not support Key Assignments');
                            return false;
                        }

                        let defaultCheckbox ='';
                        if (data.default === true) {
                            defaultCheckbox = '<input id="default" type="checkbox" checked/>';
                        } else {
                            defaultCheckbox = '<input id="default" type="checkbox"/>';
                        }

                        let holdCheckbox ='';
                        if (data.actionHold === true) {
                            holdCheckbox = '<input id="pressAndHold" type="checkbox" checked/>';
                        } else {
                            holdCheckbox = '<input id="pressAndHold" type="checkbox"/>';
                        }

                        let retainCheckbox ='';
                        if (data.retainOriginal === true) {
                            retainCheckbox = '<input id="retainOriginal" type="checkbox" checked/>';
                        } else {
                            retainCheckbox = '<input id="retainOriginal" type="checkbox"/>';
                        }

                        let modalElement = `
                          <div class="modal fade text-start" id="setupKeyAssignments" tabindex="-1" aria-labelledby="setupKeyAssignments">
                            <div class="modal-dialog modal-dialog-1000">
                              <div class="modal-content" style="width: 1000px;">
                                <div class="modal-header">
                                  <h5 class="modal-title" id="setupKeyAssignments">Setup Key Assignment - ${data.keyName}</h5>
                                  <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                </div>
                                <div class="modal-body">
                                  <form>
                                    <div class="mb-3">
                                        <table class="table mb-0">
                                            <thead>
                                            <tr>
                                                <th style="text-align: left;">Key</th>
                                                <th>
                                                    Default
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy defaultInfoToggle"></i>
                                                </th>
                                                <th>
                                                    Press and Hold / Toggle
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy pressAndHoldInfoToggle"></i>
                                                </th>
                                                <th>
                                                    Toggle Delay (ms)
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy toggleDelayInfoToggle"></i>
                                                </th>
                                                <th>
                                                    Original
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy originalInfoToggle"></i>
                                                </th>
                                                <th>Modifier</th>
                                                <th>Type</th>
                                                <th>Value</th>
                                            </tr>
                                            </thead>
                                            <tbody>
                                            <tr>
                                                <th scope="row" style="text-align: left;">${data.keyName}</th>
                                                <td>${defaultCheckbox}</td>
                                                <td>${holdCheckbox}</td>
                                                <td><input class="form-control" id="toggleDelay" type="text" value="${data.toggleDelay}" style="width: 100px;"/></td>
                                                <td>${retainCheckbox}</td>
                                                <td><select class="form-select keyAssignmentModifier" id="keyAssignmentModifier"></select></td>
                                                <td><select class="form-select keyAssignmentType" id="keyAssignmentType"></select></td>
                                                <td><select class="form-select" id="keyAssignmentValue"></select></td>
                                            </tr>
                                            </tbody>
                                        </table>
                                    </div>
                                  </form>
                                </div>
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveKeyAssignments">Save</button>
                                </div>
                              </div>
                            </div>
                          </div>
                        `;
                        const modal = $(modalElement).modal('toggle');
                        const keyAssignmentValue = modal.find("#keyAssignmentValue");

                        // Fetch assignment types
                        fetchAssignmentTypes(deviceId, data.actionType, function(optionTypes) {
                            modal.find('#keyAssignmentType').html(optionTypes);
                        });

                        fetchAssignmentModifiers(deviceId, data.modifierKey, function(optionTypes) {
                            modal.find('#keyAssignmentModifier').html(optionTypes);
                        });

                        if (parseInt(data.actionType) === 0) {
                            $(keyAssignmentValue).empty();
                            $(keyAssignmentValue).append($('<option>', { value: 0, text: "None" }));
                        } else {
                            let url = '';
                            switch (data.actionType) {
                                case 1: {
                                    url = '/api/input/media';
                                }
                                    break;
                                case 3: {
                                    url = '/api/input/keyboard';
                                }
                                    break;
                                case 9: {
                                    url = '/api/input/mouse';
                                }
                                    break;
                                case 10: {
                                    url = '/api/macro/';
                                }
                                    break;
                            }

                            $.ajax({
                                url:url,
                                type:'get',
                                success:function(result){
                                    $(keyAssignmentValue).empty();
                                    $.each(result.data, function( index, value ) {
                                        const displayName = value.Name || value.name;
                                        $(keyAssignmentValue).append($('<option>', { value: index, text: displayName, selected: parseInt(index) === parseInt(data.actionCommand) }));
                                    });
                                }
                            });
                        }

                        modal.find('#keyAssignmentType').on('change', function () {
                            const selectedValue = parseInt($(this).val());
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

                        modal.find('.defaultInfoToggle').on('click', function () {
                            const modalDefault = `
                                <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel">
                                    <div class="modal-dialog">
                                        <div class="modal-content">
                                            <div class="modal-header">
                                                <h5 class="modal-title" id="infoToggleLabel">Keyboard Default Action</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>When enabled, the keyboard performs its default key action. This checkbox ignores all user custom assignments.</span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            `;
                            const infoDefault = $(modalDefault).modal('toggle');
                            infoDefault.on('hidden.bs.modal', function () {
                                infoDefault.data('bs.modal', null);
                            })
                        });

                        modal.find('.pressAndHoldInfoToggle').on('click', function () {
                            const modalPressAndHold = `
                                <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel">
                                    <div class="modal-dialog">
                                        <div class="modal-content">
                                            <div class="modal-header">
                                                <h5 class="modal-title" id="infoToggleLabel">Press and Hold</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>Press and Hold: <br />When enabled, the keyboard continuously sends an action until the key is released. <br />
                                                Toggle: In the case of a mouse action, the action is repeated until the key is pressed again.</span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            `;
                            const infoPressAndHold = $(modalPressAndHold).modal('toggle');
                            infoPressAndHold.on('hidden.bs.modal', function () {
                                infoPressAndHold.data('bs.modal', null);
                            })
                        });

                        modal.find('.originalInfoToggle').on('click', function () {
                            const modalPressAndHold = `
                                <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel">
                                    <div class="modal-dialog">
                                        <div class="modal-content">
                                            <div class="modal-header">
                                                <h5 class="modal-title" id="infoToggleLabel">Press and Hold</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>When enabled, the original key is sent first, following the user-defined value.</span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            `;
                            const infoPressAndHold = $(modalPressAndHold).modal('toggle');
                            infoPressAndHold.on('hidden.bs.modal', function () {
                                infoPressAndHold.data('bs.modal', null);
                            })
                        });

                        modal.on('hidden.bs.modal', function () {
                            modal.data('bs.modal', null);
                            modal.remove();
                        })

                        modal.on('shown.bs.modal', function (e) {
                            modal.find('#btnSaveKeyAssignments').on('click', function () {
                                const enabled = modal.find("#default").is(':checked');
                                const pressAndHold = modal.find("#pressAndHold").is(':checked');
                                const retainOriginal = modal.find("#retainOriginal").is(':checked');
                                const keyAssignmentModifier = modal.find("#keyAssignmentModifier").val();
                                const keyAssignmentType = modal.find("#keyAssignmentType").val();
                                const keyAssignmentValue = modal.find("#keyAssignmentValue").val();
                                const toggleDelay = modal.find("#toggleDelay").val();

                                const pf = {};
                                pf["deviceId"] = deviceId;
                                pf["keyIndex"] = parseInt(globalKeyId);
                                pf["enabled"] = enabled;
                                pf["pressAndHold"] = pressAndHold;
                                pf["keyAssignmentOriginal"] = retainOriginal;
                                pf["keyAssignmentModifier"] = parseInt(keyAssignmentModifier);
                                pf["keyAssignmentType"] = parseInt(keyAssignmentType);
                                pf["keyAssignmentValue"] = parseInt(keyAssignmentValue);
                                pf["toggleDelay"] = parseInt(toggleDelay);

                                const json = JSON.stringify(pf, null, 2);
                                $.ajax({
                                    url: '/api/keyboard/updateKeyAssignment',
                                    type: 'POST',
                                    data: json,
                                    cache: false,
                                    success: function(response) {
                                        try {
                                            if (response.status === 1) {
                                                const modalElement = $("#setupKeyAssignments");
                                                $(modalElement).modal('hide');
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

    $('.userProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const userProfileValue = $(this).val();
        if (userProfileValue.length < 1) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["userProfileName"] = userProfileValue;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/userProfile/change',
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

    $('.brightness').on('change', function () {
        const deviceId = $("#deviceId").val();
        const brightness = $(this).val();
        const brightnessValue = parseInt(brightness);

        if (brightnessValue < 0 || brightnessValue > 3) {
            toast.warning('Invalid brightness selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["brightness"] = brightnessValue;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/brightness',
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

    $('#brightnessSlider').on('change', function () {
        const deviceId = $("#deviceId").val();
        const brightness = $(this).val();
        const brightnessValue = parseInt(brightness);

        if (brightnessValue < 0 || brightnessValue > 100) {
            toast.warning('Invalid brightness selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["brightness"] = brightnessValue;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/brightness/gradual',
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

    $('.saveUserProfile').on('click', function () {
        let modalElement = '<div class="modal fade text-start" id="newUserProfileModal" tabindex="-1" aria-labelledby="newUserProfileLabel" aria-hidden="true">';
        modalElement+='<div class="modal-dialog">';
        modalElement+='<div class="modal-content">';
        modalElement+='<div class="modal-header">';
        modalElement+='<h5 class="modal-title" id="newUserProfileLabel">Save user profile</h5>';
        modalElement+='<button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>';
        modalElement+='</div>';
        modalElement+='<div class="modal-body">';
        modalElement+='<form>';
        modalElement+='<div class="mb-3">';
        modalElement+='<label class="form-label" for="userProfileName">Profile Name</label>';
        modalElement+='<input class="form-control" id="userProfileName" type="text" placeholder="Enter profile name (a-z, A-Z, 0-9, -)">';
        modalElement+='</div>';
        modalElement+='</form>';
        modalElement+='</div>';
        modalElement+='<div class="modal-footer">';
        modalElement+='<button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>';
        modalElement+='<button class="btn btn-primary" type="button" id="btnSaveUserProfile">Save</button>';
        modalElement+='</div>';
        modalElement+='</div>';
        modalElement+='</div>';
        modalElement+='</div>';
        const modal = $(modalElement).modal('toggle');

        modal.on('hidden.bs.modal', function () {
            modal.data('bs.modal', null);
        })

        modal.on('shown.bs.modal', function (e) {
            const userProfileName = modal.find('#userProfileName');
            userProfileName.focus();

            modal.find('#btnSaveUserProfile').on('click', function () {
                const userProfileValue = userProfileName.val();
                if (userProfileValue.length < 1) {
                    toast.warning('Profile name can not be empty');
                    return false
                }
                const deviceId = $("#deviceId").val();

                const pf = {};
                pf["deviceId"] = deviceId;
                pf["userProfileName"] = userProfileValue;
                const json = JSON.stringify(pf, null, 2);

                $.ajax({
                    url: '/api/userProfile',
                    type: 'PUT',
                    data: json,
                    cache: false,
                    success: function(response) {
                        try {
                            if (response.status === 1) {
                                modal.modal('toggle');
                                $('.userProfile').append($('<option>', {
                                    value: userProfileValue,
                                    text: userProfileValue
                                }));
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
    });

    $('.moveLeft').on('click', function () {
        const data = $(this).attr('data').split(";");
        const deviceId = $("#deviceId").val();

        if (data.length < 2 || data.length > 2) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["position"] = parseInt(data[0]);
        pf["deviceIdString"] = data[1];
        pf["direction"] = 0;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/position',
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

    $('.moveRight').on('click', function () {
        const data = $(this).attr('data').split(";");
        const deviceId = $("#deviceId").val();

        if (data.length < 2 || data.length > 2) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["position"] = parseInt(data[0]);
        pf["deviceIdString"] = data[1];
        pf["direction"] = 1;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/position',
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

    $('.newLabel').on('click', function (e) {
        e.stopPropagation();

        const $this = $(this);
        const channelId = $this.find('.deviceData').val();
        const $label = $this.find('.labelValue');

        if ($label.find('input').length > 0) return;

        const originalText = $label.text().trim();
        const $input = $('<input type="text" class="form-control form-control-sm" />')
            .val(originalText)
            .css({
                'width': '100%',
                'display': 'inline-block'
            });

        $label.empty().append($input);
        $input.focus();

        function saveLabelIfChanged() {
            const newLabel = $input.val().trim();

            if (newLabel === originalText) {
                $label.text(originalText);
                return;
            }

            if (newLabel.length < 1) {
                toast.warning('Device label cannot be empty');
                $label.text(originalText);
                return;
            }

            $label.text(newLabel);

            const pf = {
                deviceId: $("#deviceId").val(),
                channelId: parseInt(channelId),
                deviceType: 0,
                label: newLabel
            };

            $.ajax({
                url: '/api/label',
                type: 'POST',
                data: JSON.stringify(pf),
                contentType: 'application/json',
                success: function (response) {
                    if (response.status === 1) {
                        toast.success("Label updated");
                    } else {
                        toast.warning(response.message);
                        $label.text(originalText);
                    }
                },
                error: function () {
                    toast.warning("Failed to update label");
                    $label.text(originalText);
                }
            });
        }

        $input.on('blur', saveLabelIfChanged);
        $input.on('keydown', function (e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                saveLabelIfChanged();
            } else if (e.key === 'Escape') {
                $label.text(originalText); // Cancel
            }
        });
    });

    $('.newRgbLabel').on('click', function (e) {
        e.stopPropagation();

        const $this = $(this);
        const channelId = $this.find('.deviceData').val();
        const $label = $this.find('.labelValue');

        if ($label.find('input').length > 0) return;

        const originalText = $label.text().trim();
        const $input = $('<input type="text" class="form-control form-control-sm" />')
            .val(originalText)
            .css({
                'width': '100%',
                'display': 'inline-block'
            });

        $label.empty().append($input);
        $input.focus();

        function saveLabelIfChanged() {
            const newLabel = $input.val().trim();

            if (newLabel === originalText) {
                $label.text(originalText);
                return;
            }

            if (newLabel.length < 1) {
                toast.warning('Device label cannot be empty');
                $label.text(originalText);
                return;
            }

            $label.text(newLabel);

            const pf = {
                deviceId: $("#deviceId").val(),
                channelId: parseInt(channelId),
                deviceType: 1,
                label: newLabel
            };

            $.ajax({
                url: '/api/label',
                type: 'POST',
                data: JSON.stringify(pf),
                contentType: 'application/json',
                success: function (response) {
                    if (response.status === 1) {
                        toast.success("Label updated");
                    } else {
                        toast.warning(response.message);
                        $label.text(originalText);
                    }
                },
                error: function () {
                    toast.warning("Failed to update label");
                    $label.text(originalText);
                }
            });
        }

        $input.on('blur', saveLabelIfChanged);
        $input.on('keydown', function (e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                saveLabelIfChanged();
            } else if (e.key === 'Escape') {
                $label.text(originalText); // Cancel
            }
        });
    });

    function autoRefresh() {
        setInterval(function(){
            const deviceId = $("#deviceId").val()
            $.ajax({
                url:'/api/devices/' + deviceId,
                type:'get',
                success:function(result){
                    if (result.device.devices == null) {
                        // Single device, e.g CPU block
                        const elementTemperatureId = "#temperature-0";
                        $(elementTemperatureId).html(result.device.TemperatureString);
                    } else {
                        const length = Object.keys(result.device.devices).length;
                        if (length > 0) {
                            $.each(result.device.devices, function( index, value ) {
                                const elementSpeedId = "#speed-" + value.deviceId;
                                const elementTemperatureId = "#temperature-" + value.deviceId;
                                $(elementSpeedId).html(value.rpm + " RPM");
                                $(elementTemperatureId).html(value.temperatureString);
                            });
                        }
                    }
                }
            });

            $.ajax({
                url:'/api/cpuTemp',
                type:'get',
                success:function(result){
                    $("#cpu_temp").html(result.data);
                }
            });
            $.ajax({
                url:'/api/gpuTemp',
                type:'get',
                success:function(result){
                    $("#gpu_temp").html(result.data);
                }
            });
        },1500);
    }

    autoRefresh();

    $('.tempProfile').on('change', function () {
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
            url: '/api/speed',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        toast.success(response.message);
                        $("#selectedProfile_" + parseInt(profile[0])).html(profile[1]);
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.globalTempProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const profile = $(this).val();
        const selectedDevices = $("#selectedDevices").val();
        let selectedDevicesArray = [];

        if (selectedDevices != null) {
            if (selectedDevices.length > 0) {
                selectedDevicesArray = selectedDevices
                    .split(',')
                    .map(str => parseInt(str.trim(), 10))
                    .filter(num => !isNaN(num)); // Ensure valid numbers only
            }
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = -1;
        pf["channelIds"] = selectedDevicesArray;
        pf["profile"] = profile;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/speed',
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

    $('.openRgbIntegration').on('change', function () {
        const deviceId = $("#deviceId").val();
        const mode = parseInt($(this).val());

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["mode"] = mode;

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/color/setOpenRgbIntegration',
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

    $('.rgbCluster').on('change', function () {
        const deviceId = $("#deviceId").val();
        const mode = parseInt($(this).val());

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["mode"] = mode;

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/color/setCluster',
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

    $('.globalRgb').on('change', function () {
        const deviceId = $("#deviceId").val();
        const profile = $(this).val();
        const selectedDevices = $("#selectedDevices").val();
        let selectedDevicesArray = [];

        if (selectedDevices != null) {
            if (selectedDevices.length > 0) {
                selectedDevicesArray = selectedDevices
                    .split(',')
                    .map(str => parseInt(str.trim(), 10))
                    .filter(num => !isNaN(num)); // Ensure valid numbers only
            }
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = -1;
        pf["channelIds"] = selectedDevicesArray;
        pf["profile"] = profile;

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

    $('.rgbProfile').on('change', function () {
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
                        $("#selectedRgb_" + parseInt(profile[0])).html(profile[1]);
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.allDevicesRgb').on('change', function () {
        const profile = $(this).val();
        if (profile === "none") {
            return false;
        }

        const pf = {
            "profile": profile
        };``

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color/global',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        toast.success(response.message);
                        $("#selectedRgb_" + parseInt(profile[0])).html(profile[1]);
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.linkAdapterRgbProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const profile = $(this).val().split(";");
        if (profile.length < 3 || profile.length > 3) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(profile[0]);
        pf["adapterId"] = parseInt(profile[1]);
        pf["profile"] = profile[2];

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color/linkAdapter',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        toast.success(response.message);
                        $("#selectedRgb_" + parseInt(profile[0])).html(profile[1]);
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.linkAdapterRgbProfileBulk').on('change', function () {
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
            url: '/api/color/linkAdapter/bulk',
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

    $('.keyboardRgbProfile').on('change', function () {
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

    $('.miscRgbProfile').on('change', function () {
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

    $('.rgbStrips').on('change', function () {
        const deviceId = $("#deviceId").val();
        const stripData = $(this).val().split(";");
        if (stripData.length < 2 || stripData.length > 2) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(stripData[0]);
        pf["stripId"] = parseInt(stripData[1]);

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/hub/strip',
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

    $('.linkAdapterChange').on('change', function () {
        const deviceId = $("#deviceId").val();
        const stripData = $(this).val().split(";");
        if (stripData.length < 2 || stripData.length > 2) {
            toast.warning('Invalid profile selected');
            return false;
        }

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(stripData[0]);
        pf["adapterId"] = parseInt(stripData[1]);

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/hub/linkAdapter',
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

    $('.lcdMode').on('change', function () {
        const deviceId = $("#deviceId").val();
        const mode = $(this).val().split(";");

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(mode[0]);
        pf["mode"] = parseInt(mode[1]);

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/lcd',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        toast.success(response.message);
                        if (parseInt(mode[1]) === 10) { // Animation
                            $(".lcdImages").show();
                        } else {
                            $(".lcdImages").hide();
                        }
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.lcdProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const mode = $(this).val();

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["profile"] = mode;

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/lcd/profile',
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

    $('.lcdDevices').on('change', function () {
        const deviceId = $("#deviceId").val();
        const device = $(this).val().split(";");

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(device[0]);
        pf["lcdSerial"] = device[1];

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/lcd/device',
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

    $('.lcdRotation').on('change', function () {
        const deviceId = $("#deviceId").val();
        const rotation = $(this).val().split(";");

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(rotation[0]);
        pf["rotation"] = parseInt(rotation[1]);

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/lcd/rotation',
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

    $('.lcdImages').on('change', function () {
        const deviceId = $("#deviceId").val();
        const image = $(this).val().split(";");

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(image[0]);
        pf["image"] = image[1];

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/lcd/image',
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

    $('#deviceSpeed').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = -1; // All devices
        pf["profile"] = $(this).val();

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/speed',
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

    $('#externalHubStatus').on('change', function () {
        const deviceId = $("#deviceId").val();
        const status = $(this).val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["enabled"] = status === "1";
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/hub/status',
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

    $('.externalHubDeviceType').change(function(){
        const container = $(this).closest(".externalHubDevice");
        const deviceId = $("#deviceId").val();
        const deviceType = $(this).val();
        const portId = container.find(".portId").val();
        const pf = {};

        pf["portId"] = parseInt(portId);
        pf["deviceId"] = deviceId;
        pf["deviceType"] = parseInt(deviceType);
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/hub/type',
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

    $('.externalHubDeviceAmount').change(function(){
        const container = $(this).closest(".externalHubDevice");
        const deviceId = $("#deviceId").val();
        const deviceAmount = $(this).val();
        const portId = container.find(".portId").val();
        const pf = {};

        pf["portId"] = parseInt(portId);
        pf["deviceId"] = deviceId;
        pf["deviceAmount"] = parseInt(deviceAmount);
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/hub/amount',
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

    $('#deviceRgb').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["profile"] = $(this).val();

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

    $('.addCustomARGBDevice').on('click', function () {
        const deviceId = $("#deviceId").val();
        const portId = $(".customLedPort").val();
        const deviceType = $(".customLedPortLEDAmount").val();

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["portId"] = parseInt(portId);
        pf["deviceType"] = parseInt(deviceType);

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/argb',
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

    $('.rgbOverride').on('click', function () {
        const deviceId = $("#deviceId").val();
        const channelId = $(this).attr("data-info");

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(channelId);
        pf["subDeviceId"] = 0;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color/getOverride',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        const data = response.data;

                        const startColor = rgbToHex(data.RGBStartColor.red, data.RGBStartColor.green, data.RGBStartColor.blue);
                        const endColor = rgbToHex(data.RGBEndColor.red, data.RGBEndColor.green, data.RGBEndColor.blue);

                        let modalElement = `
                          <div class="modal fade text-start" id="rgbOverrideModel" tabindex="-1" aria-labelledby="rgbOverrideModel">
                            <div class="modal-dialog modal-dialog-800">
                              <div class="modal-content" style="width: 800px;">
                                <div class="modal-header">
                                  <h5 class="modal-title" id="rgbOverrideModel">RGB Override</h5>
                                  <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                </div>
                                <div class="modal-body">
                                  <form>
                                    <div class="mb-3">
                                        <table class="table mb-0">
                                            <thead>
                                            <tr>
                                                <th style="text-align: left;">Enabled</th>
                                                <th style="text-align: left;">Start</th>
                                                <th style="text-align: left;">End</th>
                                                <th style="text-align: right;">Speed</th>
                                            </tr>
                                            </thead>
                                            <tbody>
                                            <tr>
                                                <th style="text-align: left;">
                                                    <input type="checkbox" id="enabledCheckbox" ${data.Enabled ? "checked" : ""}>
                                                </th>
                                                <th style="text-align: left;">
                                                    <input type="color" id="startColor" value="${startColor}">
                                                </th>
                                                <th style="text-align: left;">
                                                    <input type="color" id="endColor" value="${endColor}">
                                                </th>
                                                <th style="text-align: right;">
                                                    <input class="brightness-slider" type="range" id="speedSlider" name="speedSlider" style="margin-top: 5px;" min="1" max="10" value="${data.RgbModeSpeed}" step="0.1" />
                                                </th>
                                            </tr>
                                            </tbody>
                                        </table>
                                    </div>
                                  </form>
                                </div>
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveRgbOverride">Save</button>
                                </div>
                              </div>
                            </div>
                          </div>
                        `;
                        const modal = $(modalElement).modal('toggle');

                        modal.on('hidden.bs.modal', function () {
                            modal.data('bs.modal', null);
                            modal.remove();
                        })

                        modal.on('shown.bs.modal', function (e) {
                            modal.find('#btnSaveRgbOverride').on('click', function () {
                                const pf = {};
                                let startColorRgb = {}
                                let endColorRgb = {}

                                let speed = $("#speedSlider").val();
                                const startColorVal = $("#startColor").val();
                                const endColorVal = $("#endColor").val();

                                const startColor = hexToRgb(startColorVal);
                                startColorRgb = {red:startColor.r, green:startColor.g, blue:startColor.b}

                                const endColor = hexToRgb(endColorVal);
                                endColorRgb = {red:endColor.r, green:endColor.g, blue:endColor.b}

                                const enabled = $("#enabledCheckbox").is(':checked');

                                pf["deviceId"] = deviceId;
                                pf["channelId"] = parseInt(channelId);
                                pf["subDeviceId"] = 0;
                                pf["enabled"] = enabled;
                                pf["startColor"] = startColorRgb;
                                pf["endColor"] = endColorRgb;
                                pf["speed"] = parseFloat(speed);

                                const json = JSON.stringify(pf, null, 2);
                                $.ajax({
                                    url: '/api/color/setOverride',
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

    function createLinearLEDs(cnt, leds, spacing, data, startX = 0, startY = 0) {
        let count = leds.length;
        cnt.style.width = `${startX + count * spacing + spacing/2}px`;
        for (let i = 0; i < count; i++) {
            const x = startX + i * spacing;
            const y = startY;

            let c = data[leds[i]];
            const ledColor = rgbToHex(c.red, c.green, c.blue);

            const $led = $('<input>', {
                type: 'color',
                value: ledColor,
                id: 'ledId_' + leds[i],
                class: 'led',
                'data-info': 'ledId_' + leds[i],
                css: {
                    position: 'absolute',
                    left: `${x}px`,
                    top: `${y}px`,
                    border: '1px solid #121212'
                }
            });
            $(cnt).append($led);
        }
    }

    function createRingLEDs(cnt, leds, radius, data, center) {
        let count = leds.length;
        for (let i = 0; i < count; i++) {
            const angle = (i / count) * 2 * Math.PI;
            const x = Math.cos(angle) * radius + center - 6;
            const y = Math.sin(angle) * radius + center - 6;

            let c = data[leds[i]];
            const ledColor = rgbToHex(c.red, c.green, c.blue);

            const $led = $('<input>', {
                type: 'color',
                value: ledColor,
                id: 'ledId_' + leds[i],
                class: 'led',
                'data-info': 'ledId_' + leds[i],
                css: {
                    position: 'absolute',
                    left: `${x}px`,
                    top: `${y}px`,
                    border: '1px solid #121212'
                }
            });
            $(cnt).append($led);
        }
    }

    function generateLedDataPerDevice(ledAmount, subDevice, device, data) {
        const wrapperDiv = document.createElement('div');
        let result = [];

        let frontOuter = [];
        let frontInner = [];
        let backOuter = [];
        let backInner = [];
        let containerHtml = '';

        switch (device) {
            case "lsh": {
                // LINK System Hub
                switch (ledAmount) {
                    case 34: {
                        if (subDevice) {
                            frontOuter = [10,11,12,13,14,15,16,17,18,19,20,21];
                            frontInner = [0,1,2,3];
                            backOuter = [22,23,24,25,26,27,28,29,30,31,32,33];
                            backInner = [4,5,6,7,8,9];
                        } else {
                            frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11];
                            frontInner = [24,25,26,27,28,29];
                            backOuter = [12,13,14,15,16,17,18,19,20,21,22,23];
                            backInner = [30,31,32,33];
                        }
                        wrapperDiv.innerHTML = `
                            <div style="">
                                <div style="text-align: center;">FRONT</div>
                                <div class="device-container" id="container">
                                    <div class="center-circle"></div>
                                </div>
                            </div>
                            <div style="">
                                <div style="text-align: center;">BACK</div>
                                <div class="device-container" id="container1" style="margin-left: 10px;">
                                    <div class="center-circle"></div>
                                </div>
                            </div>
                        `;
                        const container = wrapperDiv.querySelector('#container');
                        const container1 = wrapperDiv.querySelector('#container1');
                        createRingLEDs(container, frontInner, 45, data, 100);
                        createRingLEDs(container, frontOuter, 80, data, 100);
                        createRingLEDs(container1, backInner, 45, data, 100);
                        createRingLEDs(container1, backOuter, 80, data, 100);
                    } break;
                    case 8: {
                        frontInner = [0,1,2,3,4,5,6,7];
                        wrapperDiv.innerHTML = `<div class="device-container" id="container"><div class="center-circle"></div></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createRingLEDs(container, frontInner, 45, data, 100);
                    } break;
                    case 10: {
                        frontOuter = [0,1,2,3,4,5,6,7,8,9];
                        wrapperDiv.innerHTML = `<div class="device-container-strip" id="container"></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createLinearLEDs(container, frontOuter, 15, data, 10, 9);
                    } break;
                    case 18: {
                        frontInner = [0,1,2,3,4,5];
                        frontOuter = [6,7,8,9,10,11,12,13,14,15,16,17];
                        wrapperDiv.innerHTML = `<div class="device-container" id="container"><div class="center-circle"></div></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createRingLEDs(container, frontInner, 45, data, 100);
                        createRingLEDs(container, frontOuter, 80, data, 100);
                    } break;
                    case 40: {
                        frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39];
                        wrapperDiv.innerHTML = `<div class="device-container-strip" id="container"></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createLinearLEDs(container, frontOuter, 15, data, 10, 9);
                    } break;
                    case 49: {
                        frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48];
                        wrapperDiv.innerHTML = `<div class="device-container-strip" id="container"></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createLinearLEDs(container, frontOuter, 15, data, 10, 9);
                    } break;
                    case 38: {
                        frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37];
                        wrapperDiv.innerHTML = `<div class="device-container-strip" id="container"></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createLinearLEDs(container, frontOuter, 15, data, 10, 9);
                    } break;
                    case 32: {
                        frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31];
                        wrapperDiv.innerHTML = `<div class="device-container-strip" id="container"></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createLinearLEDs(container, frontOuter, 15, data, 10, 9);
                    } break;
                    case 24: {
                        frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23];
                        wrapperDiv.innerHTML = `<div class="device-container-block" id="container"><div class="center-circle"></div></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createLinearLEDs(container, frontOuter, 15, data, 10, 9);
                    } break;
                    case 22: {
                        frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21];
                        wrapperDiv.innerHTML = `<div class="device-container-pump" id="container"><div class="center-circle"></div></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createLinearLEDs(container, frontOuter, 15, data, 10, 9);
                    } break;
                    case 44: {
                        frontInner = [20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43];
                        frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19];
                        wrapperDiv.innerHTML = `<div class="device-container-pump" id="container"><div class="center-circle"></div></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createRingLEDs(container, frontOuter, 120, data, 150);
                        createRingLEDs(container, frontInner, 90, data, 150);
                    } break;
                    case 20: {
                        frontInner = [16,17,18,19];
                        frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15];
                        wrapperDiv.innerHTML = `<div class="device-container-pump" id="container"><div class="center-circle"></div></div>`;
                        const container = wrapperDiv.querySelector('#container');
                        createRingLEDs(container, frontOuter, 120, data, 150);
                        createRingLEDs(container, frontInner, 45, data, 150);
                    } break;
                    case 16: {
                        if (subDevice) {
                            frontInner = [0,1,2,3];
                            frontOuter = [4,5,6,7,8,9,10,11,12,13,14,15];
                            wrapperDiv.innerHTML = `<div class="device-container" id="container"><div class="center-circle"></div></div>`;
                            const container = wrapperDiv.querySelector('#container');
                            createRingLEDs(container, frontInner, 45, data, 100);
                            createRingLEDs(container, frontOuter, 80, data, 100);
                        } else {
                            frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15];
                            wrapperDiv.innerHTML = `<div class="device-container-block" id="container"><div class="center-circle"></div></div>`;
                            const container = wrapperDiv.querySelector('#container');
                            createRingLEDs(container, frontOuter, 120, data, 150);
                        }
                    } break;
                }
            } break;
            case "memory": {
                // Memory
                switch (ledAmount) {
                    case 6: {
                        frontOuter = [0,1,2,3,4,5];
                        wrapperDiv.innerHTML = `<div class="device-container-strip" id="container"></div>`;
                    } break;
                    case 10: {
                        frontOuter = [0,1,2,3,4,5,6,7,8,9];
                        wrapperDiv.innerHTML = `<div class="device-container-strip" id="container"></div>`;
                    } break;
                    case 11: {
                        frontOuter = [0,1,2,3,4,5,6,7,8,9,10];
                        wrapperDiv.innerHTML = `<div class="device-container-strip" id="container"></div>`;
                    } break;
                    case 12: {
                        frontOuter = [0,1,2,3,4,5,6,7,8,9,10,11];
                        wrapperDiv.innerHTML = `<div class="device-container-strip" id="container"></div>`;
                    } break;
                }
                const container = wrapperDiv.querySelector('#container');
                createLinearLEDs(container, frontOuter, 15, data, 10, 9);
            } break;
            case "elite": {
                // Elite coolers
                switch (ledAmount) {
                    case 16: {
                        if (subDevice) {
                            frontInner = [0,1,2,3];
                            frontOuter = [4,5,6,7,8,9,10,11,12,13,14,15];
                            wrapperDiv.innerHTML = `<div class="device-container" id="container"><div class="center-circle"></div></div>`;
                            const container = wrapperDiv.querySelector('#container');
                            createRingLEDs(container, frontInner, 45, data, 100);
                            createRingLEDs(container, frontOuter, 80, data, 100);
                        } else {
                            frontInner = [0,1,2,3];
                            frontOuter = [4,5,6,7,8,9,10,11,12,13,14,15];
                            wrapperDiv.innerHTML = `<div class="device-container-block" id="container"><div class="center-circle"></div></div>`;
                            const container = wrapperDiv.querySelector('#container');
                            createRingLEDs(container, frontOuter, 120, data, 150);
                            createRingLEDs(container, frontInner, 45, data, 150);
                        }
                    } break;
                }
            } break;
        }
        result = wrapperDiv.innerHTML
        return result
    }

    $('.rgbPerLed').on('click', function () {
        const deviceId = $("#deviceId").val();
        const channelData = $(this).attr("data-info").split(';');
        const channelName = channelData[0];
        const channelId = parseInt(channelData[1]);
        const ledAmount = parseInt(channelData[2]);
        const subDeviceId = parseInt(channelData[3]);
        const subDevice = parseInt(channelData[4]) === 1;
        const deviceType = channelData[5];
        let containerHtml = '';

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = channelId;
        pf["subDeviceId"] = subDeviceId;
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/color/getLedData',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        const data = response.data;
                        const count = Object.keys(data).length;
                        containerHtml = generateLedDataPerDevice(ledAmount, subDevice, deviceType, data)

                        let modalElement = `
                          <div class="modal fade text-start" id="rgbPerLedModel" tabindex="-1" aria-labelledby="rgbPerLedModel">
                            <div class="modal-dialog modal-dialog-800">
                              <div class="modal-content" style="width: 800px;">
                                <div class="modal-header">
                                  <h5 class="modal-title" id="rgbPerLedModel">${channelName}</h5>
                                  <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                </div>
                                <div class="modal-body" style="display: flex;margin: 0 auto;">
                                  ${containerHtml}
                                </div>
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveLedData">Save</button>
                                </div>
                              </div>
                            </div>
                          </div>
                        `;
                        const modal = $(modalElement).modal('toggle');

                        modal.on('hidden.bs.modal', function () {
                            modal.data('bs.modal', null);
                            modal.remove();
                        })

                        modal.on('shown.bs.modal', function (e) {
                            modal.find('#btnSaveLedData').on('click', function () {
                                let ledColors = {};

                                for (let i = 0; i < count; i++) {
                                    let ledColor = modal.find('#ledId_' + i).val();
                                    const colorRgb = hexToRgb(ledColor)
                                    ledColors[i] = {red: colorRgb.r, green: colorRgb.g, blue: colorRgb.b};
                                }
                                const pf = {};

                                pf["deviceId"] = deviceId;
                                pf["channelId"] = channelId;
                                pf["subDeviceId"] = subDeviceId;
                                pf["colorZones"] = ledColors;
                                pf["save"] = true;

                                const json = JSON.stringify(pf, null, 2);
                                $.ajax({
                                    url: '/api/color/setLedData',
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

    $('.rgbOverrideLinkAdapter').on('click', function () {
        const deviceId = $("#deviceId").val();
        const channelId = $(this).attr("data-info").split(';');

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(channelId[0]);
        pf["subDeviceId"] = parseInt(channelId[1]);
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color/getOverride',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        const data = response.data;

                        const startColor = rgbToHex(data.RGBStartColor.red, data.RGBStartColor.green, data.RGBStartColor.blue);
                        const endColor = rgbToHex(data.RGBEndColor.red, data.RGBEndColor.green, data.RGBEndColor.blue);

                        let modalElement = `
                          <div class="modal fade text-start" id="rgbOverrideModel" tabindex="-1" aria-labelledby="rgbOverrideModel">
                            <div class="modal-dialog modal-dialog-800">
                              <div class="modal-content" style="width: 800px;">
                                <div class="modal-header">
                                  <h5 class="modal-title" id="rgbOverrideModel">RGB Override</h5>
                                  <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                </div>
                                <div class="modal-body">
                                  <form>
                                    <div class="mb-3">
                                        <table class="table mb-0">
                                            <thead>
                                            <tr>
                                                <th style="text-align: left;">Enabled</th>
                                                <th style="text-align: left;">Start</th>
                                                <th style="text-align: left;">End</th>
                                                <th style="text-align: right;">Speed</th>
                                            </tr>
                                            </thead>
                                            <tbody>
                                            <tr>
                                                <th style="text-align: left;">
                                                    <input type="checkbox" id="enabledCheckbox" ${data.Enabled ? "checked" : ""}>
                                                </th>
                                                <th style="text-align: left;">
                                                    <input type="color" id="startColor" value="${startColor}">
                                                </th>
                                                <th style="text-align: left;">
                                                    <input type="color" id="endColor" value="${endColor}">
                                                </th>
                                                <th style="text-align: right;">
                                                    <input class="brightness-slider" type="range" id="speedSlider" name="speedSlider" style="margin-top: 5px;" min="1" max="10" value="${data.RgbModeSpeed}" step="0.1" />
                                                </th>
                                            </tr>
                                            </tbody>
                                        </table>
                                    </div>
                                  </form>
                                </div>
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveRgbOverrideLinkAdapter">Save</button>
                                </div>
                              </div>
                            </div>
                          </div>
                        `;
                        const modal = $(modalElement).modal('toggle');

                        modal.on('hidden.bs.modal', function () {
                            modal.data('bs.modal', null);
                            modal.remove();
                        })

                        modal.on('shown.bs.modal', function (e) {
                            modal.find('#btnSaveRgbOverrideLinkAdapter').on('click', function () {
                                const pf = {};
                                let startColorRgb = {}
                                let endColorRgb = {}

                                let speed = $("#speedSlider").val();
                                const startColorVal = $("#startColor").val();
                                const endColorVal = $("#endColor").val();

                                const startColor = hexToRgb(startColorVal);
                                startColorRgb = {red:startColor.r, green:startColor.g, blue:startColor.b}

                                const endColor = hexToRgb(endColorVal);
                                endColorRgb = {red:endColor.r, green:endColor.g, blue:endColor.b}

                                const enabled = $("#enabledCheckbox").is(':checked');

                                pf["deviceId"] = deviceId;
                                pf["channelId"] = parseInt(channelId[0]);
                                pf["subDeviceId"] = parseInt(channelId[1]);
                                pf["enabled"] = enabled;
                                pf["startColor"] = startColorRgb;
                                pf["endColor"] = endColorRgb;
                                pf["speed"] = parseFloat(speed);

                                const json = JSON.stringify(pf, null, 2);
                                $.ajax({
                                    url: '/api/color/setOverride',
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

    $('.keyboardColor').on('click', function () {
        const applyButton = $('#applyColors')
        applyButton.unbind('click');

        const deviceId = $("#deviceId").val();
        const keyInfo = $(this).attr("data-info").split(";");
        const keyId = parseInt(keyInfo[0]);
        noColorChange(deviceId, keyId).then(result => {
            if (result) {
                $(".keyColorArea").hide();
            } else {
                $(".keyColorArea").show();
            }
        });

        noKeyAssignments(deviceId, keyId).then(result => {
            if (result) {
                $(".keyAssignmentsArea").hide();
            } else {
                $(".keyAssignmentsArea").show();
            }
        });

        const colorR = parseInt(keyInfo[1]);
        const colorG = parseInt(keyInfo[2]);
        const colorB = parseInt(keyInfo[3]);
        const hex = rgbToHex(colorR, colorG, colorB);
        $("#keyColor").val('' + hex + '');
        globalKeyId = keyId;

        applyButton.on('click', function () {
            const keyOption = $(".keyOptions").val();
            const keyColor = $('#keyColor').val();
            const rgb = hexToRgb(keyColor);
            if (rgb.r === colorR && rgb.g === colorG && rgb.b === colorB) {
                toast.warning('Old and new colors are identical');
                return false;
            }

            const pf = {};
            const color = {red:rgb.r, green:rgb.g, blue:rgb.b}
            pf["deviceId"] = deviceId;
            pf["keyId"] = keyId;
            pf["keyOption"] = parseInt(keyOption);
            pf["color"] = color;

            const json = JSON.stringify(pf, null, 2);
            $.ajax({
                url: '/api/keyboard/color',
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
    });

    $('.miscColor').on('click', function () {
        const applyButton = $('#applyColors')
        applyButton.unbind('click');

        const deviceId = $("#deviceId").val();
        const miscInfo = $(this).attr("data-info").split(";");
        const areaId = parseInt(miscInfo[0]);
        const colorR = parseInt(miscInfo[1]);
        const colorG = parseInt(miscInfo[2]);
        const colorB = parseInt(miscInfo[3]);
        const hex = rgbToHex(colorR, colorG, colorB);
        $("#miscColor").val('' + hex + '');

        applyButton.on('click', function () {
            const miscOptions = $(".miscOptions").val();
            const miscColor = $('#miscColor').val();
            const rgb = hexToRgb(miscColor);
            if (rgb.r === colorR && rgb.g === colorG && rgb.b === colorB) {
                toast.warning('Old and new colors are identical');
                return false;
            }

            const pf = {};
            const color = {red:rgb.r, green:rgb.g, blue:rgb.b}
            pf["deviceId"] = deviceId;
            pf["areaId"] = areaId;
            pf["areaOption"] = parseInt(miscOptions);
            pf["color"] = color;

            const json = JSON.stringify(pf, null, 2);
            $.ajax({
                url: '/api/misc/color',
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
    });

    $('#saveAsProfile').on('click', function () {
        let modalElement = '<div class="modal fade text-start" id="newUserProfileModal" tabindex="-1" aria-labelledby="newUserProfileLabel" aria-hidden="true">';
        modalElement+='<div class="modal-dialog">';
        modalElement+='<div class="modal-content">';
        modalElement+='<div class="modal-header">';
        modalElement+='<h5 class="modal-title" id="newUserProfileLabel">Save keyboard profile</h5>';
        modalElement+='<button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>';
        modalElement+='</div>';
        modalElement+='<div class="modal-body">';
        modalElement+='<form>';
        modalElement+='<div class="mb-3">';
        modalElement+='<label class="form-label" for="userProfileName">Profile Name</label>';
        modalElement+='<input class="form-control" id="userProfileName" type="text" placeholder="Enter profile name (a-z, A-Z, 0-9)">';
        modalElement+='</div>';
        modalElement+='</form>';
        modalElement+='</div>';
        modalElement+='<div class="modal-footer">';
        modalElement+='<button class="btn btn-secondary" type="button" data-bs-dismiss="modal">Close</button>';
        modalElement+='<button class="btn btn-primary" type="button" id="btnSaveKeyboardProfile">Save</button>';
        modalElement+='</div>';
        modalElement+='</div>';
        modalElement+='</div>';
        modalElement+='</div>';
        const modal = $(modalElement).modal('toggle');

        modal.on('hidden.bs.modal', function () {
            modal.data('bs.modal', null);
        })

        modal.on('shown.bs.modal', function (e) {
            const keyboardProfileName = modal.find('#userProfileName');
            keyboardProfileName.focus();

            modal.find('#btnSaveKeyboardProfile').on('click', function () {
                const keyboardProfileValue = keyboardProfileName.val();
                if (keyboardProfileValue.length < 1) {
                    toast.warning('Profile name can not be empty');
                    return false
                }
                const deviceId = $("#deviceId").val();

                const pf = {};
                pf["deviceId"] = deviceId;
                pf["keyboardProfileName"] = keyboardProfileValue;
                pf["new"] = true;

                const json = JSON.stringify(pf, null, 2);

                $.ajax({
                    url: '/api/keyboard/profile/new',
                    type: 'PUT',
                    data: json,
                    cache: false,
                    success: function(response) {
                        try {
                            if (response.status === 1) {
                                modal.modal('toggle');
                                $('.keyboardProfile').append($('<option>', {
                                    value: keyboardProfileValue,
                                    text: keyboardProfileValue
                                }));
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
    });

    $('.keyboardProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyboardProfileName"] = $(this).val();

        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/profile/change',
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

    $('#saveProfile').on('click', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyboardProfileName"] = "0";
        pf["new"] = false;

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/keyboard/profile/save',
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

    $('#deleteProfile').on('click', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyboardProfileName"] = $(".keyboardProfile").val();

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/keyboard/profile/delete',
            type: 'DELETE',
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

    $('.keyLayout').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyboardLayout"] = $(this).val();
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/layout',
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

    $('.controlDial').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyboardControlDial"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/dial',
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

    $('.sleepModes').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["sleepMode"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/sleep',
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

    $('.hardwareLights').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["hardwareLight"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color/hardware',
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

    $('.keyboardPollingRate').on('change', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["pollingRate"] = parseInt($(this).val());
        const json = JSON.stringify(pf, null, 2);

        $('.keyboardPollingRate').prop('disabled', true);
        $.ajax({
            url: '/api/keyboard/pollingRate',
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
                $('.keyboardPollingRate').prop('disabled', false);
            }
        });
    });
});