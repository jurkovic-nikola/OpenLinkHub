"use strict";
$(document).ready(function () {
    window.i18n = {
        locale: null,
        values: {},

        setTranslations: function (locale, values) {
            this.locale = locale;
            this.values = values || {};
        },

        t: function (key, fallback = '') {
            return this.values[key] ?? fallback ?? key;
        }
    };

    $.ajax({
        url: '/api/language',
        method: 'GET',
        dataType: 'json',
        success: function (response) {
            if (response.status === 1 && response.data) {
                i18n.setTranslations(
                    response.data.code,
                    response.data.values
                );
            }
        },
        error: function () {
            console.error('Failed to load translations');
        }
    });

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
                          <div class="modal fade" id="systemModal" tabindex="-1" aria-hidden="true">
                            <div class="modal-dialog modal-custom modal-500">
                              <div class="modal-content">
                                <div class="modal-header">
                                  <h5 class="modal-title">${i18n.t('txtPerformance')}</h5>
                                  <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                                </div>
                                <div class="modal-body">
                                    <div class="settings-list"></div>
                                </div>
                        
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveKeyboardPerformance">${i18n.t('txtSave')}</button>
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
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">${value.name}</span>
                                    <label class="system-toggle compact">
                                        ${element}
                                        <span class="toggle-track"></span>
                                    </label>
                                </div>
                            `;
                            modal.find('.settings-list').append(newRow);
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

    $('.keyboardFlashTap').on('click', function () {
        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/keyboard/getFlashTap/' + deviceId,
            type: 'GET',
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        let flashTapOptions = null;

                        const data = response.data;
                        const color = rgbToHex(data.color.red, data.color.green, data.color.blue);
                        let modalElement = `
                          <div class="modal fade" id="systemModal" tabindex="-1" aria-hidden="true">
                            <div class="modal-dialog modal-custom modal-500">
                              <div class="modal-content">
                                <div class="modal-header">
                                  <h5 class="modal-title">${i18n.t('txtFlashTap')}</h5>
                                  <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                                </div>
                                <div class="modal-body">
                                    <div class="settings-list">
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtEnable')}</span>
                                            <label class="system-toggle compact">
                                                <input type="checkbox" id="flashTapActive" ${data.active ? "checked" : ""}>
                                                <span class="toggle-track"></span>
                                            </label>
                                        </div>
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtColor')}</span>
                                            <div class="system-color compact">
                                                <label for="flashTapColor">
                                                    <input type="color" id="flashTapColor" value="${color}">
                                                </label>
                                            </div>
                                        </div>
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtModes')}</span>
                                            <div class="no-padding-top">
                                                <select class="system-select compact full-width flashTapMode" id="flashTapMode"></select>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveFlashTap">${i18n.t('txtSave')}</button>
                                </div>
                              </div>
                            </div>
                          </div>
                        `;
                        const modal = $(modalElement).modal('toggle');

                        function loadFlashTapKeys(callback) {
                            if (flashTapOptions) {
                                callback && callback();
                                return;
                            }
                            $.ajax({
                                url: '/api/keyboard/getKeys/',
                                type: 'POST',
                                data: json,
                                cache: false,
                                success: function (resp) {
                                    if (resp.status === 1) {
                                        flashTapOptions = $();
                                        $.each(resp.data, function (index, value) {
                                            flashTapOptions = flashTapOptions.add(
                                                new Option(value, index)
                                            );
                                        });
                                        callback && callback();
                                    }
                                }
                            });
                        }

                        loadFlashTapKeys(function () {
                            $.each(data.keys, function (index, value) {
                                var $row = $(`
                                    <div class="settings-row">
                                        <span class="settings-label text-ellipsis">${i18n.t('txtKey')} ${index}</span>
                                        <div class="no-padding-top">
                                            <select class="system-select compact full-width flashTapKey" id="flashTapKey${index}"></select>
                                        </div>
                                    </div>
                                `);
                                const $select = $row.find('.flashTapKey');
                                $select.append(flashTapOptions.clone());
                                if (value.Key !== undefined) {
                                    $select.val(value.Key);
                                }

                                $select.find('option').filter(function () {
                                    return $(this).text() === value.Name;
                                }).prop('selected', true);

                                modal.find('.settings-list').append($row);
                            });
                        });

                        const modeValue = data.mode;
                        const modes = data.modes;
                        const $modeSelect = modal.find('#flashTapMode');
                        $modeSelect.empty();
                        $.each(modes, function (value, label) {
                            $modeSelect.append(
                                $('<option>', {
                                    value: value,
                                    text: label
                                })
                            );
                        });
                        $modeSelect.val(String(modeValue));

                        modal.on('shown.bs.modal', function (e) {
                            modal.find('#btnSaveFlashTap').on('click', function () {
                                const pf = {};
                                const active = modal.find("#flashTapActive").is(':checked');
                                const flashTapKey0 = parseInt(modal.find("#flashTapKey0").val());
                                const flashTapKey1 = parseInt(modal.find("#flashTapKey1").val());
                                const flashTapMode = parseInt(modal.find("#flashTapMode").val());
                                let color = modal.find("#flashTapColor").val();
                                color = hexToRgb(color);

                                pf["deviceId"] = deviceId;
                                pf["flashTapActive"] = active ? 1 : 0;
                                pf["flashTapKeys"] = [flashTapKey0, flashTapKey1];
                                pf["flashTapMode"] = flashTapMode;
                                pf["flashTapColor"] = {red: color.r, green: color.g, blue: color.b};
                                const json = JSON.stringify(pf, null, 2);
                                $.ajax({
                                    url: '/api/keyboard/setFlashTap',
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
                            <div class="modal fade" id="systemModal" tabindex="-1" aria-hidden="true">
                              <div class="modal-dialog modal-custom modal-500">
                                <div class="modal-content">
                                  <div class="modal-header">
                                    <h5 class="modal-title">${i18n.t('txtControlDialColors')}</h5>
                                    <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                                  </div>
                                  <div class="modal-body">
                                      <div class="settings-list"></div>
                                  </div>
                            
                                  <div class="modal-footer">
                                    <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                                    <button class="btn btn-primary" type="button" id="btnSaveControlDialColors">${i18n.t('txtSave')}</button>
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
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">${value.Name}</span>
                                    <div class="system-color compact">
                                        <label for="startColor">
                                            <input type="color" id="dial-color-${optionId}" value="${color}">
                                        </label>
                                    </div>
                                </div>
                            `;
                            modal.find('.settings-list').append(newRow);
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

    function noKeyActuation(deviceId, keyId) {
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
                        if (response.status === 1 && response.data.noActuation === true) {
                            noChange(true);
                        } else if (response.data.onlyColor === true && response.data.modifier === false) {
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
        const selectedDevices = $("#selectedDevices").val();
        let selectedDevicesArray = [];
        let lastIndex = null;

        if (selectedDevices != null) {
            if (selectedDevices.length > 0) {
                selectedDevicesArray = selectedDevices
                    .split(',')
                    .map(str => parseInt(str.trim(), 10))
                    .filter(num => !isNaN(num)); // Ensure valid numbers only

                lastIndex = selectedDevicesArray.at(-1) ?? null;
            } else if (selectedDevices.length === 1) {
                lastIndex = parseInt(selectedDevices);
            }
        }

        if (lastIndex === null) {
            toast.warning(i18n.t('txtSelectValidKey'));
            return false;
        }

        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyId"] = parseInt(lastIndex);
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
                            toast.warning(i18n.t('txtNoKeyAssignments'));
                            return false;
                        }

                        let defaultCheckbox = `
                            <label class="system-toggle compact">
                                <input type="checkbox" id="default" ${data.default ? "checked" : ""}>
                                <span class="toggle-track"></span>
                            </label>
                        `;

                        let holdCheckbox = `
                            <label class="system-toggle compact">
                                <input type="checkbox" id="pressAndHold" ${data.actionHold ? "checked" : ""}>
                                <span class="toggle-track"></span>
                            </label>
                        `;

                        let toggleDelayInput = '<input id="toggleDelay" type="text" value="' + data.toggleDelay + '"/>';

                        let modalElement = `
                          <div class="modal fade" id="systemModal" tabindex="-1" aria-hidden="true">
                            <div class="modal-dialog modal-custom modal-1200">
                              <div class="modal-content">
                                <div class="modal-header">
                                  <h5 class="modal-title" id="setupKeyAssignments">${i18n.t('txtKeyAssignment')} - ${data.keyName}</h5>
                                  <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                </div>
                                <div class="modal-body">
                                  <form>
                                    <div class="mb-3">
                                        <table class="dataTable text-sm">
                                            <thead>
                                            <tr>
                                                <th>${i18n.t('txtKey')}</th>
                                                <th>
                                                    ${i18n.t('defaultValue')}
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy defaultInfoToggle"></i>
                                                </th>
                                                <th>
                                                    ${i18n.t('txtPressHoldToggle')}
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy pressAndHoldInfoToggle"></i>
                                                </th>
                                                <th>
                                                    ${i18n.t('txtToggleDelayMs')}
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy toggleDelayInfoToggle"></i>
                                                </th>
                                                <th>${i18n.t('txtType')}</th>
                                                <th>${i18n.t('txtValue')}</th>
                                            </tr>
                                            </thead>
                                            <tbody>
                                            <tr>
                                                <td class="key-assignments">${data.keyName}</td>
                                                <td>${defaultCheckbox}</td>
                                                <td>${holdCheckbox}</td>
                                                <td>
                                                    <div class="system-input text-input compact">
                                                        <label for="userProfileName">
                                                            <input type="text" id="toggleDelay" autocomplete="off" value="${data.toggleDelay}">
                                                        </label>
                                                    </div>                                    
                                                </td>
                                                <td>
                                                    <select class="system-select compact keyAssignmentType" id="keyAssignmentType"></select>
                                                </td>
                                                <td>
                                                    <select class="system-select compact" id="keyAssignmentValue"></select>
                                                </td>
                                            </tr>
                                            </tbody>
                                        </table>
                                    </div>
                                  </form>
                                </div>
                                <div class="modal-footer">
                                  <button class="system-button secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                                  <button class="system-button" type="button" id="btnSaveKeyAssignments">${i18n.t('txtSave')}</button>
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
                                                <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtKeyboardDefaultAction')}</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>${i18n.t('txtKeyboardDefaultActionInfo')}</span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
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
                                                <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtPressAndHold')}</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>
                                                <b>${i18n.t('txtPressAndHold')}:</b><br />${i18n.t('txtPressAndHoldInfoKeyboard')}<br /><br />
                                                <b>${i18n.t('txtToggle')}:</b><br /> ${i18n.t('txtPressAndHoldInfoKeyboardToggle')}
                                                </span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
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
                                                <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtToggleDelayMs')}</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>
                                                <b>${i18n.t('txtToggleDelayMs')}:</b><br /> ${i18n.t('txtToggleDelayInfo')}
                                                </span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
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
                                pf["keyIndex"] = parseInt(lastIndex);
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

    $('.openKeyActuation').on('click', function () {
        const selectedDevices = $("#selectedDevices").val();
        let selectedDevicesArray = [];
        let lastIndex = null;

        if (selectedDevices != null) {
            if (selectedDevices.length > 0) {
                selectedDevicesArray = selectedDevices
                    .split(',')
                    .map(str => parseInt(str.trim(), 10))
                    .filter(num => !isNaN(num)); // Ensure valid numbers only

                lastIndex = selectedDevicesArray.at(-1) ?? null;
            } else if (selectedDevices.length === 1) {
                lastIndex = parseInt(selectedDevices);
            }
        }

        if (lastIndex === null) {
            toast.warning(i18n.t('txtSelectValidKey'));
            return false;
        }

        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyId"] = parseInt(lastIndex);
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
                        if (data.onlyColor === true && data.modifier === false) {
                            toast.warning(i18n.t('txtNoKeyActuation'));
                            return false;
                        }
                        if (data.noActuation === true) {
                            toast.warning(i18n.t('txtNoKeyActuation'));
                            return false;
                        }

                        let modalElement = `
                          <div class="modal fade" id="systemModal" tabindex="-1" aria-hidden="true">
                            <div class="modal-dialog modal-custom modal-600">
                              <div class="modal-content">
                        
                                <div class="modal-header">
                                  <h5 class="modal-title">${i18n.t('txtKeyActuation')} - ${data.keyName}</h5>
                                  <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                                </div>
                                <div class="modal-body">
                                    <div class="settings-list">
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtApplyAllKeys')}</span>
                                            <label class="system-toggle compact">
                                                <input type="checkbox" id="actuationAllKeys">
                                                <span class="toggle-track"></span>
                                            </label>
                                        </div>
                                        
                                        <div class="settings-row settings-actuation">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtActuation')}</span>
                                            <div class="system-slider no-padding-top">
                                                <input type="range" class="actuationPoint" id="actuationPoint" min="1" max="40" value="${data.actuationPoint}" step="1">
                                            </div>
                                            <span class="settings-label text-ellipsis" id="actuationValue">${data.actuationPoint / 10} mm</span>
                                        </div>
                                        
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtEnableResetPoint')}</span>
                                            <label class="system-toggle compact">
                                                <input type="checkbox" id="enableActuationPointReset" ${data.enableActuationPointReset ? "checked" : ""}>
                                                <span class="toggle-track"></span>
                                            </label>
                                        </div>
                                        
                                        <div class="settings-row settings-actuation" id="primaryReset">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtActuationReset')}</span>
                                            <div class="system-slider no-padding-top">
                                                <input type="range" class="actuationResetPoint" id="actuationResetPoint" min="1" max="40" value="${data.actuationResetPoint}" step="1">
                                            </div>
                                            <span class="settings-label text-ellipsis" id="actuationValueReset">${data.actuationResetPoint / 10} mm</span>
                                        </div>

                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtSecondaryActuation')}</span>
                                            <label class="system-toggle compact">
                                                <input type="checkbox" id="enableSecondaryActuationPoint" ${data.enableSecondaryActuationPoint ? "checked" : ""}>
                                                <span class="toggle-track"></span>
                                            </label>
                                        </div>
                                        
                                        <div id="secondaryContainer">
                                            <div class="settings-row settings-actuation" id="secondaryPoint">
                                                <span class="settings-label text-ellipsis">${i18n.t('txtActuation')}</span>
                                                <div class="system-slider no-padding-top">
                                                    <input type="range" class="secondaryActuationPoint" id="secondaryActuationPoint" min="1" max="40" value="${data.secondaryActuationPoint}" step="1">
                                                </div>
                                                <span class="settings-label text-ellipsis" id="secondaryActuationValue">${data.secondaryActuationPoint / 10} mm</span>
                                            </div>
                                            
                                            <div class="settings-row settings-actuation" id="secondaryReset">
                                                <span class="settings-label text-ellipsis">${i18n.t('txtActuationReset')}</span>
                                                <div class="system-slider no-padding-top">
                                                    <input type="range" class="secondaryActuationResetPoint" id="secondaryActuationResetPoint" min="1" max="40" value="${data.secondaryActuationResetPoint}" step="1">
                                                </div>
                                                <span class="settings-label text-ellipsis" id="secondaryActuationValueReset">${data.secondaryActuationResetPoint / 10} mm</span>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                        
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveActuationValue">${i18n.t('txtSave')}</button>
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
                            const enableSecondaryActuationPoint = $('#enableSecondaryActuationPoint');
                            const container = $("#secondaryContainer");

                            function updateSecondaryVisibility() {
                                if (enableSecondaryActuationPoint.is(":checked")) {
                                    container.stop(true, true).slideDown(150);
                                } else {
                                    container.stop(true, true).slideUp(150);
                                }

                                container
                                    .find("input")
                                    .prop("disabled", !enableSecondaryActuationPoint.is(":checked"));
                            }

                            updateSecondaryVisibility();
                            enableSecondaryActuationPoint.on('change', function () {
                                updateSecondaryVisibility();
                            });

                            function updateActuationSlider(el) {
                                const $slider = $(el);
                                const min = Number($slider.attr("min"));
                                const max = Number($slider.attr("max"));
                                const value = Number($slider.val());
                                const percent = ((value - min) / (max - min)) * 100;
                                $slider.css("--slider-progress", percent + "%");
                            }

                            $(".actuationPoint").each(function () {
                                $("#actuationValue").html(this.value / 10 + " mm");
                                updateActuationSlider(this);
                            }).on("input", function () {
                                $("#actuationValue").html(this.value / 10 + " mm");
                                updateActuationSlider(this);
                            });

                            $(".actuationResetPoint").each(function () {
                                $("#actuationValueReset").html(this.value / 10 + " mm");
                                updateActuationSlider(this);
                            }).on("input", function () {
                                $("#actuationValueReset").html(this.value / 10 + " mm");
                                updateActuationSlider(this);
                            });

                            $(".secondaryActuationPoint").each(function () {
                                $("#secondaryActuationValue").html(this.value / 10 + " mm");
                                updateActuationSlider(this);
                            }).on("input", function () {
                                $("#secondaryActuationValue").html(this.value / 10 + " mm");
                                updateActuationSlider(this);
                            });

                            $(".secondaryActuationResetPoint").each(function () {
                                $("#secondaryActuationValueReset").html(this.value / 10 + " mm");
                                updateActuationSlider(this);
                            }).on("input", function () {
                                $("#secondaryActuationValueReset").html(this.value / 10 + " mm");
                                updateActuationSlider(this);
                            });

                            modal.find('#btnSaveActuationValue').on('click', function () {
                                const actuationAllKeys = modal.find("#actuationAllKeys").is(':checked');
                                const actuationPoint = modal.find("#actuationPoint").val();
                                const enableActuationPointReset = modal.find("#enableActuationPointReset").is(':checked');
                                const actuationResetPoint = modal.find("#actuationResetPoint").val();
                                const enableSecondaryActuationPoint = modal.find("#enableSecondaryActuationPoint").is(':checked');
                                const secondaryActuationPoint = modal.find("#secondaryActuationPoint").val();
                                const secondaryActuationResetPoint = modal.find("#secondaryActuationResetPoint").val();

                                const pf = {};
                                pf["deviceId"] = deviceId;
                                pf["keyIndex"] = parseInt(lastIndex);
                                pf["actuationAllKeys"] = actuationAllKeys;
                                pf["actuationPoint"] = parseInt(actuationPoint);
                                pf["enableActuationPointReset"] = enableActuationPointReset;
                                pf["actuationResetPoint"] = parseInt(actuationResetPoint);
                                pf["enableSecondaryActuationPoint"] = enableSecondaryActuationPoint;
                                pf["secondaryActuationPoint"] = parseInt(secondaryActuationPoint);
                                pf["secondaryActuationResetPoint"] = parseInt(secondaryActuationResetPoint);

                                const json = JSON.stringify(pf, null, 2);
                                $.ajax({
                                    url: '/api/keyboard/updateActuation',
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
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.openKeyAssignmentsWithModifier').on('click', function () {
        const selectedDevices = $("#selectedDevices").val();
        let selectedDevicesArray = [];
        let lastIndex = null;

        if (selectedDevices != null) {
            if (selectedDevices.length > 0) {
                selectedDevicesArray = selectedDevices
                    .split(',')
                    .map(str => parseInt(str.trim(), 10))
                    .filter(num => !isNaN(num)); // Ensure valid numbers only

                lastIndex = selectedDevicesArray.at(-1) ?? null;
            } else if (selectedDevices.length === 1) {
                lastIndex = parseInt(selectedDevices);
            }
        }

        if (lastIndex === null) {
            toast.warning(i18n.t('txtSelectValidKey'));
            return false;
        }

        const deviceId = $("#deviceId").val();
        const pf = {};
        pf["deviceId"] = deviceId;
        pf["keyId"] = parseInt(lastIndex);
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
                            toast.warning(i18n.t('txtNoKeyAssignments'));
                            return false;
                        }

                        let defaultCheckbox = `
                            <label class="system-toggle compact">
                                <input type="checkbox" id="default" ${data.default ? "checked" : ""}>
                                <span class="toggle-track"></span>
                            </label>
                        `;

                        let holdCheckbox = `
                            <label class="system-toggle compact">
                                <input type="checkbox" id="pressAndHold" ${data.actionHold ? "checked" : ""}>
                                <span class="toggle-track"></span>
                            </label>
                        `;

                        let retainCheckbox = `
                            <label class="system-toggle compact">
                                <input type="checkbox" id="retainOriginal" ${data.retainOriginal ? "checked" : ""}>
                                <span class="toggle-track"></span>
                            </label>
                        `;

                        let modalElement = `
                          <div class="modal fade" id="systemModal" tabindex="-1" aria-hidden="true">
                            <div class="modal-dialog modal-custom modal-1300">
                              <div class="modal-content">
                                <div class="modal-header">
                                  <h5 class="modal-title" id="setupKeyAssignments">${i18n.t('txtKeyAssignment')} - ${data.keyName}</h5>
                                  <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                </div>
                                <div class="modal-body">
                                  <form>
                                    <div class="mb-3">
                                        <table class="dataTable text-sm">
                                            <thead>
                                            <tr>
                                                <th>${i18n.t('txtKey')}</th>
                                                <th>
                                                    ${i18n.t('defaultValue')}
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy defaultInfoToggle"></i>
                                                </th>
                                                <th>
                                                    ${i18n.t('txtPressHoldToggle')}
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy pressAndHoldInfoToggle"></i>
                                                </th>
                                                <th>
                                                    ${i18n.t('txtToggleDelayMs')}
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy toggleDelayInfoToggle"></i>
                                                </th>
                                                <th>
                                                    ${i18n.t('txtOriginal')}
                                                    <i style="cursor: pointer;" class="bi bi-info-circle-fill svg-icon svg-icon-sm svg-icon-heavy originalInfoToggle"></i>
                                                </th>
                                                <th>${i18n.t('txtModifier')}</th>
                                                <th>${i18n.t('txtType')}</th>
                                                <th>${i18n.t('txtValue')}</th>
                                            </tr>
                                            </thead>
                                            <tbody>
                                            <tr>
                                                <td class="key-assignments">${data.keyName}</td>
                                                <td>${defaultCheckbox}</td>
                                                <td>${holdCheckbox}</td>
                                                <td>
                                                    <div class="system-input text-input compact">
                                                        <label for="toggleDelay">
                                                            <input type="text" id="toggleDelay" autocomplete="off" value="${data.toggleDelay}">
                                                        </label>
                                                    </div>                                    
                                                </td>
                                                <td>${retainCheckbox}</td>
                                                <td>
                                                    <select class="system-select compact keyAssignmentModifier" id="keyAssignmentModifier"></select>
                                                </td>
                                                <td>
                                                    <select class="system-select compact keyAssignmentType" id="keyAssignmentType"></select>
                                                </td>
                                                <td>
                                                    <select class="system-select compact" id="keyAssignmentValue"></select>
                                                </td>
                                            </tr>
                                            </tbody>
                                        </table>
                                    </div>
                                  </form>
                                </div>
                                <div class="modal-footer">
                                  <button class="system-button secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                                  <button class="system-button" type="button" id="btnSaveKeyAssignments">${i18n.t('txtSave')}</button>
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
                                                <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtKeyboardDefaultAction')}</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>${i18n.t('txtKeyboardDefaultActionInfo')}</span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
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
                                                <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtPressAndHold')}</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>${i18n.t('txtPressAndHold')}: <br />${i18n.t('txtPressAndHoldInfoKeyboard')} <br />
                                                ${i18n.t('txtToggle')}: ${i18n.t('txtToggleMouseAction')}</span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
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
                                                <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtRetainOriginal')}</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>${i18n.t('txtRetainOriginalInfo')}</span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
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
                                                <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtToggleDelayMs')}</h5>
                                                <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                                            </div>
                                            <div class="modal-body">
                                                <span>
                                                <b>${i18n.t('txtToggleDelayMs')}:</b><br /> ${i18n.t('txtToggleDelayInfo')}
                                                </span>
                                            </div>
                                            <div class="modal-footer">
                                                <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
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
                                pf["keyIndex"] = parseInt(lastIndex);
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
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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
            toast.warning(i18n.t('txtInvalidBrightness'));
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
            toast.warning(i18n.t('txtInvalidBrightness'));
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
        const modalElement = `
            <div class="modal fade text-start" id="systemModal" tabindex="-1" aria-hidden="true">
                <div class="modal-dialog modal-custom modal-600">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title" id="newTempModalLabel">${i18n.t('txtSaveUserProfile')}</h5>
                            <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                        </div>
                        <div class="modal-body modal-title">
                            <div class="settings-list">
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">${i18n.t('txtProfileName')}</span>
                                    <div class="system-input text-input">
                                        <label for="userProfileName">
                                            <input type="text" id="userProfileName" autocomplete="off" placeholder="${i18n.t('txtProfileOnlyLettersNumbers')}">
                                        </label>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="modal-footer">
                            <button class="system-button secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                            <button class="system-button" type="button" id="btnSaveUserProfile">${i18n.t('txtSave')}</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        const modal = $(modalElement).modal('toggle');

        modal.on('hidden.bs.modal', function () {
            modal.data('bs.modal', null);
        })

        modal.on('shown.bs.modal', function () {
            const userProfileName = modal.find('#userProfileName');
            const saveBtn = modal.find('#btnSaveUserProfile');

            userProfileName.focus();

            // Trigger save on Enter
            userProfileName.on('keydown', function (e) {
                if (e.key === 'Enter') {
                    e.preventDefault(); // prevent form submit / modal close
                    saveBtn.trigger('click');
                }
            });

            saveBtn.on('click', function () {
                const userProfileValue = userProfileName.val();
                if (userProfileValue.length < 3) {
                    toast.warning(i18n.t('txtProfileNameTooShort'));
                    return false;
                }

                const deviceId = $("#deviceId").val();
                const pf = {
                    deviceId: deviceId,
                    userProfileName: userProfileValue
                };

                $.ajax({
                    url: '/api/userProfile',
                    type: 'PUT',
                    data: JSON.stringify(pf),
                    cache: false,
                    success: function (response) {
                        if (response.status === 1) {
                            modal.modal('toggle');

                            $('.userProfile').append(
                                $('<option>', { value: userProfileValue, text: userProfileValue })
                            );
                            $('.deleteUserProfiles').append(
                                $('<option>', { value: userProfileValue, text: userProfileValue })
                            );

                            toast.success(response.message);
                        } else {
                            toast.warning(response.message);
                        }
                    }
                });
            });
        });
    });

    $('.moveLeft').on('click', function () {
        const data = $(this).attr('data').split(";");
        const deviceId = $("#deviceId").val();

        if (data.length < 2 || data.length > 2) {
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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

        const $inputWrapper = $(
            '<div class="system-input text-input compact">' +
            '<input type="text" autocomplete="off">' +
            '</div>'
        );

        const $input = $inputWrapper.find('input');
        $input.val(originalText);

        $label.empty().append($inputWrapper);
        $input.focus();

        function saveLabelIfChanged() {
            const newLabel = $input.val().trim();

            if (newLabel === originalText) {
                $label.text(originalText);
                return;
            }

            if (newLabel.length < 1) {
                toast.warning(i18n.t('txtDeviceLabelEmpty'));
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
                        toast.success(i18n.t('txtDeviceLabelApplied'));
                    } else {
                        toast.warning(response.message);
                        $label.text(originalText);
                    }
                },
                error: function () {
                    toast.warning(i18n.t('txtUnableToApplyLabel'));
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

        const $inputWrapper = $(
            '<div class="system-input text-input compact">' +
            '<input type="text" autocomplete="off">' +
            '</div>'
        );

        const $input = $inputWrapper.find('input');
        $input.val(originalText);

        $label.empty().append($inputWrapper);
        $input.focus();

        function saveLabelIfChanged() {
            const newLabel = $input.val().trim();

            if (newLabel === originalText) {
                $label.text(originalText);
                return;
            }

            if (newLabel.length < 1) {
                toast.warning(i18n.t('txtDeviceLabelEmpty'));
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
                        toast.success(i18n.t('txtDeviceLabelApplied'));
                    } else {
                        toast.warning(response.message);
                        $label.text(originalText);
                    }
                },
                error: function () {
                    toast.warning(i18n.t('txtUnableToApplyLabel'));
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

                                if (value.IsPSU) {
                                    const elementPowerOut = "#powerOut-" + value.channelId;
                                    if (elementPowerOut != null) {
                                        $(elementPowerOut).html(value.powerOutString + " W");
                                    }

                                    $.each(value.volts, function( index, value ) {
                                        const elementVolts = "#volts-" + index;
                                        if (elementVolts != null) {
                                            $(elementVolts).html(value.ValueString + " V");
                                        }
                                    });

                                    $.each(value.amps, function( index, value ) {
                                        const elementAmps = "#amps-" + index;
                                        if (elementAmps != null) {
                                            $(elementAmps).html(value.ValueString + " A");
                                        }
                                    });

                                    $.each(value.watts, function( index, value ) {
                                        const elementWatts = "#watts-" + index;
                                        if (elementWatts != null) {
                                            $(elementWatts).html(value.ValueString + " W");
                                        }
                                    });
                                }
                            });
                        }
                    }
                }
            });
        },1500);
    }

    autoRefresh();

    $('.tempProfile').on('change', function () {
        const deviceId = $("#deviceId").val();
        const profile = $(this).val().split(";");
        if (profile.length < 2 || profile.length > 2) {
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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
                        /*location.reload();*/
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
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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
            toast.warning(i18n.t('txtInvalidProfileSelected'));
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
                        if (parseInt(mode[1]) === 10) {
                            $(".lcdImagesHolder").show();
                        } else {
                            $(".lcdImagesHolder").hide();
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

    $('.commanderDuoOverride').on('click', function () {
        const deviceId = $("#deviceId").val();
        const channelId = $(this).attr("data-info");

        const pf = {};
        pf["deviceId"] = deviceId;
        pf["channelId"] = parseInt(channelId);
        pf["subDeviceId"] = 0;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/color/override/' + deviceId,
            type: 'GET',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        $.each(response.data, function(key, value) {
                            if (parseInt(key) === parseInt(channelId)) {
                                let modalElement = `
                                  <div class="modal fade" id="systemModal" tabindex="-1" aria-hidden="true">
                                    <div class="modal-dialog modal-custom modal-500">
                                      <div class="modal-content">
                                
                                        <div class="modal-header">
                                          <h5 class="modal-title">${i18n.t('txtLedOverride')}</h5>
                                          <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                                        </div>
                                        <div class="modal-body">
                                            <div class="settings-list">
                                                <div class="settings-row">
                                                    <span class="settings-label text-ellipsis">${i18n.t('txtEnable')}</span>
                                                    <label class="system-toggle compact">
                                                        <input type="checkbox" id="enabledCheckbox" ${value.Enabled ? "checked" : ""}>
                                                        <span class="toggle-track"></span>
                                                    </label>
                                                </div>
            
                                                <div class="settings-row">
                                                    <span class="settings-label text-ellipsis">${i18n.t('txtLedAmount')}</span>
                                                    <div class="system-input text-input">
                                                        <label for="ledChannels">
                                                            <input type="text" id="ledChannels" autocomplete="off" placeholder="Enter LED amount" value="${value.LedChannels}">
                                                        </label>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                
                                        <div class="modal-footer">
                                          <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                                          <button class="btn btn-primary" type="button" id="btnSaveOverride">${i18n.t('txtSave')}</button>
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
                                    modal.find('#btnSaveOverride').on('click', function () {
                                        const pf = {};
                                        const enabled = $("#enabledCheckbox").is(':checked');
                                        const ledChannels = $("#ledChannels").val();

                                        pf["deviceId"] = deviceId;
                                        pf["channelId"] = parseInt(channelId);
                                        pf["enabled"] = enabled;
                                        pf["ledChannels"] = parseInt(ledChannels);

                                        const json = JSON.stringify(pf, null, 2);
                                        $.ajax({
                                            url: '/api/color/override/update',
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
                            }
                        });
                    } else {
                        toast.warning(response.data);
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
                          <div class="modal fade" id="systemModal" tabindex="-1" aria-hidden="true">
                            <div class="modal-dialog modal-custom modal-500">
                              <div class="modal-content">
                        
                                <div class="modal-header">
                                  <h5 class="modal-title">${i18n.t('txtRgbOverride')}</h5>
                                  <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                                </div>
                                <div class="modal-body">
                                    <div class="settings-list">
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtEnable')}</span>
                                            <label class="system-toggle compact">
                                                <input type="checkbox" id="enabledCheckbox" ${data.Enabled ? "checked" : ""}>
                                                <span class="toggle-track"></span>
                                            </label>
                                        </div>
    
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtStartColor')}</span>
                                            <div class="system-color">
                                                <label for="startColor">
                                                    <input type="color" id="startColor" value="${startColor}">
                                                </label>
                                            </div>
                                        </div>
                                        
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtEndColor')}</span>
                                            <div class="system-color">
                                                    <input type="color" class="system-color" id="endColor" value="${endColor}">
                                            </div>
                                        </div>
                                        
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtSpeed')}</span>
                                            <div class="system-slider no-padding-top">
                                                <img src="/static/img/icons/icon-fast.svg" width="20" height="20" alt="Fast" />
                                                <label for="speedSlider" class="margin-lr-10">
                                                    <input type="range" id="speedSlider" name="speedSlider" min="1" max="10" value="${data.RgbModeSpeed}" step="0.1">
                                                </label>
                                                <img src="/static/img/icons/icon-slow.svg" width="20" height="20" alt="Sloe" />
                                            </div>
                                        </div>
                                    </div>
                                </div>
                        
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveRgbOverride">${i18n.t('txtSave')}</button>
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
                            const $speedSlider = modal.find("#speedSlider");
                            const $speedSliderValue = modal.find("#speedSliderValue");

                            function updateSpeedSlider() {
                                const min = Number($speedSlider.attr("min"));
                                const max = Number($speedSlider.attr("max"));
                                const value = Number($speedSlider.val());

                                const percent = ((value - min) / (max - min)) * 100;

                                $speedSlider.css("--slider-progress", percent + "%");
                                $speedSliderValue.text(value);
                            }

                            if ($speedSlider.length) {
                                $speedSlider.on("input", updateSpeedSlider);
                                updateSpeedSlider();
                            }

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
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveLedData">${i18n.t('txtSave')}</button>
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
                          <div class="modal fade" id="systemModal" tabindex="-1" aria-hidden="true">
                            <div class="modal-dialog modal-custom modal-500">
                              <div class="modal-content">
                        
                                <div class="modal-header">
                                  <h5 class="modal-title">${i18n.t('txtRgbOverride')}</h5>
                                  <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                                </div>
                                <div class="modal-body">
                                    <div class="settings-list">
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtEnable')}</span>
                                            <label class="system-toggle compact">
                                                <input type="checkbox" id="enabledCheckbox" ${data.Enabled ? "checked" : ""}>
                                                <span class="toggle-track"></span>
                                            </label>
                                        </div>
    
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtStartColor')}</span>
                                            <div class="system-color">
                                                <label for="startColor">
                                                    <input type="color" id="startColor" value="${startColor}">
                                                </label>
                                            </div>
                                        </div>
                                        
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtEndColor')}</span>
                                            <div class="system-color">
                                                    <input type="color" class="system-color" id="endColor" value="${endColor}">
                                            </div>
                                        </div>
                                        
                                        <div class="settings-row">
                                            <span class="settings-label text-ellipsis">${i18n.t('txtSpeed')}</span>
                                            <div class="system-slider no-padding-top">
                                                <img src="/static/img/icons/icon-fast.svg" width="20" height="20" alt="Fast" />
                                                <label for="speedSlider" class="margin-lr-10">
                                                    <input type="range" id="speedSlider" name="speedSlider" min="1" max="10" value="${data.RgbModeSpeed}" step="0.1">
                                                </label>
                                                <img src="/static/img/icons/icon-slow.svg" width="20" height="20" alt="Sloe" />
                                            </div>
                                        </div>
                                    </div>
                                </div>
                        
                                <div class="modal-footer">
                                  <button class="btn btn-secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                                  <button class="btn btn-primary" type="button" id="btnSaveRgbOverrideLinkAdapter">${i18n.t('txtSave')}</button>
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
                            const $speedSlider = modal.find("#speedSlider");
                            const $speedSliderValue = modal.find("#speedSliderValue");

                            function updateSpeedSlider() {
                                const min = Number($speedSlider.attr("min"));
                                const max = Number($speedSlider.attr("max"));
                                const value = Number($speedSlider.val());

                                const percent = ((value - min) / (max - min)) * 100;

                                $speedSlider.css("--slider-progress", percent + "%");
                                $speedSliderValue.text(value);
                            }

                            if ($speedSlider.length) {
                                $speedSlider.on("input", updateSpeedSlider);
                                updateSpeedSlider();
                            }
                            
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

    $('#applyColors').on('click', function () {
        const keyOption = $(".keyOptions").val();
        if (parseInt(keyOption) === 2) {
            const deviceId = $("#deviceId").val();
            const keyColor = $('#keyColor').val();
            const rgb = hexToRgb(keyColor);

            const pf = {};
            const color = {red:rgb.r, green:rgb.g, blue:rgb.b}
            pf["deviceId"] = deviceId;
            pf["keyId"] = 1;
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
        } else {
            toast.warning(i18n.t('txtInvalidKeyOptionSelected'));
        }
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

        noKeyActuation(deviceId, keyId).then(result => {
            if (result) {
                $(".openKeyActuation").hide();
            } else {
                $(".openKeyActuation").show();
            }
        });

        const colorR = parseInt(keyInfo[1]);
        const colorG = parseInt(keyInfo[2]);
        const colorB = parseInt(keyInfo[3]);
        const hex = rgbToHex(colorR, colorG, colorB);
        $("#keyColor").val('' + hex + '');

        applyButton.on('click', function () {
            const keyOption = $(".keyOptions").val();
            const keyColor = $('#keyColor').val();
            const rgb = hexToRgb(keyColor);

            const pf = {};
            const color = {red:rgb.r, green:rgb.g, blue:rgb.b}

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

            pf["deviceId"] = deviceId;
            pf["keyId"] = keyId;
            pf["keyOption"] = parseInt(keyOption);
            pf["keys"] = selectedDevicesArray;
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
        const modalElement = `
            <div class="modal fade text-start" id="systemModal" tabindex="-1" aria-hidden="true">
                <div class="modal-dialog modal-custom modal-600">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title" id="newTempModalLabel">${i18n.t('txtSaveUserProfile')}</h5>
                            <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                        </div>
                        <div class="modal-body modal-title">
                            <div class="settings-list">
                                <div class="settings-row">
                                    <span class="settings-label text-ellipsis">${i18n.t('txtProfileName')}</span>
                                    <div class="system-input text-input">
                                        <label for="userProfileName">
                                            <input type="text" id="userProfileName" autocomplete="off" placeholder="${i18n.t('txtProfileOnlyLettersNumbers')}">
                                        </label>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="modal-footer">
                            <button class="system-button secondary" type="button" data-bs-dismiss="modal">${i18n.t('txtClose')}</button>
                            <button class="system-button" type="button" id="btnSaveKeyboardProfile">${i18n.t('txtSave')}</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
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
                    toast.warning(i18n.t('txtInvalidProfileName'));
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

    $('.debounceTime').on('change', function () {
        const $el = $(this);
        const deviceId = $("#deviceId").val();
        const pf = {
            deviceId: deviceId,
            debounceTime: parseInt($el.val(), 10)
        };
        $el.prop('disabled', true);

        $.ajax({
            url: '/api/keyboard/debounceTime',
            type: 'POST',
            data: JSON.stringify(pf),
            cache: false,
            success: function (response) {
                if (response?.status === 1) {
                    toast.success(response.message);
                } else {
                    toast.warning(response?.message || 'Unknown response');
                }
            },
            error: function () {
                toast.error('Failed to update debounce time');
            },
            complete: function () {
                // Always re-enable (success OR error)
                $el.prop('disabled', false);
            }
        });
    });

    $(".toggleRgbCluster").on("change", function () {
        const $toggle = $(this);
        const previousState = !$toggle.prop("checked");
        const newState = $toggle.prop("checked");
        const deviceId = $("#deviceId").val();

        $toggle.prop("disabled", true);

        $.ajax({
            url: "/api/color/setCluster",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({
                deviceId: deviceId,
                mode: newState ? 1 : 0
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

    $(".toggleOpenRGB").on("change", function () {
        const $toggle = $(this);
        const previousState = !$toggle.prop("checked");
        const newState = $toggle.prop("checked");
        const deviceId = $("#deviceId").val();

        $toggle.prop("disabled", true);

        $.ajax({
            url: "/api/color/setOpenRgbIntegration",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({
                deviceId: deviceId,
                mode: newState ? 1 : 0
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

    $(".toggleAutoBrightness").on("change", function () {
        const $toggle = $(this);
        const previousState = !$toggle.prop("checked");
        const newState = $toggle.prop("checked");
        const deviceId = $("#deviceId").val();

        $toggle.prop("disabled", true);

        $.ajax({
            url: "/api/keyboard/autoBrightness",
            type: "POST",
            contentType: "application/json",
            data: JSON.stringify({
                deviceId: deviceId,
                autoBrightness: newState ? 1 : 0
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

    $(".deleteUserProfile").on("click", function () {
        const profile = $("#deleteUserProfiles").val();
        const deviceId = $("#deviceId").val();

        if (profile.length < 1) {
            toast.warning(i18n.t('txtInvalidProfileSelected'));
            return false;
        }

        if (profile === "none") {
            toast.warning(i18n.t('txtInvalidProfileSelected'));
            return false;
        }

        if (profile === "default") {
            toast.warning(i18n.t('txtUnableToDeleteDefaultProfile'));
            return false;
        }

        $.ajax({
            url: "/api/userProfile/delete",
            type: "DELETE",
            contentType: "application/json",
            data: JSON.stringify({
                deviceId: deviceId,
                userProfileName: profile
            }),
            success(response) {
                if (response?.status !== 1) {
                    toast.warning(response?.message || "Operation failed");
                } else {
                    toast.success(response?.message || "Operation failed");
                    $('.userProfile option[value="' + profile + '"]').remove();
                    $('#deleteUserProfiles option[value="' + profile + '"]').remove();
                }
            },
            error() {
                toast.warning("Request failed");
            }
        });
    });

    const $brightnessSlider = $("#brightnessSlider");
    const $brightnessSliderValue = $("#brightnessSliderValue");
    function updateSlider() {
        const min = Number($brightnessSlider.attr("min"));
        const max = Number($brightnessSlider.attr("max"));
        const value = Number($brightnessSlider.val());

        const percent = ((value - min) / (max - min)) * 100;

        $brightnessSlider.css("--slider-progress", percent + "%");
        $brightnessSliderValue.text(value + " %");
    }

    if ($brightnessSlider.length) {
        $brightnessSlider.on("input", updateSlider);
        updateSlider();
    }
});