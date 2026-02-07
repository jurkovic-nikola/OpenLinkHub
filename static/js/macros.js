"use strict";
$(document).ready(function () {
    let dt = null;
    window.i18n = {
        locale: null,
        values: {},
        setTranslations(locale, values) {
            this.locale = locale;
            this.values = values || {};
        },
        t(key, fallback = '') {
            return this.values[key] ?? fallback ?? key;
        }
    };

    $.ajax({
        url: '/api/language',
        method: 'GET',
        dataType: 'json',
        success(response) {
            if (response.status === 1 && response.data) {
                i18n.setTranslations(
                    response.data.code,
                    response.data.values
                );
            }
            initDataTable();
        },
        error() {
            console.warn('Translations failed to load, using fallback text');
            initDataTable();
        }
    });

    let allOptions = [];

    function initDataTable() {
        if (dt) return;
        dt = $('#table').DataTable({
            order: [[0, 'asc']],
            select: {
                style: 'os',
                selector: 'td:first-child'
            },
            paging: false,
            searching: false,
            info: false,
            language: {
                emptyTable: i18n.t(
                    'txtNoMacros',
                    'No profile selected or profile has no macros defined. Select profile from left side or define macros'
                )
            }
        });
    }

    $('#addMacroValueModal').on('show.bs.modal', function () {
        $('#macroType').val('0');
        $('#macroText').val('');
        $('#macroDelay').val('');
        $('#macroKeySearch').val('');

        $(".macroKeyId").hide();
        $(".macroDelayId").hide();
        $(".macroTextId").hide();
    });

    $('#btnSaveNewMacroProfile').on('click', function(){
        const profile = $("#profileName").val();
        if (profile.length < 3) {
            toast.warning(i18n.t('txtProfileNameTooShort'));
            return false;
        }

        const pf = {};
        pf["macroName"] = profile;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/macro/new',
            type: 'PUT',
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

    // Attach once outside AJAX call
    $('#table').on('click', '.deleteMacroValue', function () {
        const $btn = $(this);
        const macroInfo = $btn.data('id');
        const macro = macroInfo.split(";");

        if (macro.length !== 2) {
            toast.warning(i18n.t('txtInvalidMacroProfile'));
            return false;
        }

        const pf = {
            macroId: parseInt(macro[0]),
            macroIndex: parseInt(macro[1])
        };

        $.ajax({
            url: '/api/macro/value',
            type: 'DELETE',
            data: JSON.stringify(pf),
            cache: false,
            success: function(response) {
                if (response.status === 1) {
                    dt.row($btn.closest('tr')).remove().draw();
                    toast.success(i18n.t('txtMacroValueDeleted'));
                } else {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('#table').on('click', '.updateMacroValue', function () {
        const $btn = $(this);
        const macroInfo = $btn.data('id');
        const macro = macroInfo.split(";");
        const $row = $btn.closest('tr');
        const pressAndHold = $row.find('.pressAndHold').is(':checked');
        const actionRepeatValue = $row.find('.actionRepeatValue').val();
        const actionRepeatDelayValue = $row.find('.actionRepeatDelayValue').val();

        if (macro.length !== 2) {
            toast.warning(i18n.t('txtInvalidMacroProfile'));
            return false;
        }

        const pf = {
            macroId: parseInt(macro[0]),
            macroIndex: parseInt(macro[1]),
            pressAndHold: pressAndHold,
            actionRepeatValue: parseInt(actionRepeatValue),
            actionRepeatDelay: parseInt(actionRepeatDelayValue)
        };

        $.ajax({
            url: '/api/macro/updateValue',
            type: 'POST',
            data: JSON.stringify(pf),
            cache: false,
            success: function(response) {
                if (response.status === 1) {
                    toast.success(i18n.t('txtMacroValueUpdated'));
                } else {
                    toast.warning(response.message);
                }
            }
        });
    });

    function loadMacroValues(pf) {
        $.ajax({
            url: '/api/macro/' + pf,
            dataType: 'JSON',
            success: function(response) {
                if (response.code === 0) {
                    toast.warning(response.message);
                } else {
                    const data = response.data.actions;
                    dt.clear().draw();
                    $("#profile").val(pf);
                    $.each(data, function(i, item) {
                        let actionType = '';
                        switch (item.actionType) {
                            case 1:
                                actionType = 'Media Keys';
                                break;
                            case 3:
                                actionType = 'Keyboard';
                                break;
                            case 5:
                                actionType = 'Delay';
                                break;
                            case 6:
                                actionType = 'Text';
                                break;
                            case 9:
                                actionType = 'Mouse';
                                break;
                            default:
                                actionType = 'n/a';
                                break;
                        }

                        const actionHold = `
                            <label class="system-toggle compact">
                                <input type="checkbox" class="pressAndHold" ${item.actionHold === true ? 'checked' : ''}>
                                <span class="toggle-track"></span>
                            </label>
                        `;

                        if (item.actionType === 5)
                        { // 5 is always Delay option
                            dt.row.add([
                                i,
                                actionType,
                                item.actionDelay,
                                'N/A',
                                'N/A',
                                'N/A',
                                '' +
                                `<input class="system-button danger deleteMacroValue" id="deleteMacroValue" data-id="${pf};${i}" type="button" value="DELETE" style="width: 100%;">`
                            ]).draw();
                        }
                    else
                        if (item.actionType === 6) { // String
                            dt.row.add([
                                i,
                                actionType,
                                `<span id="macroText" class="settings-label text-ellipsis">${item.actionText}</span>`,
                                'N/A',
                                `<div class="system-input text-input">
                                    <label for="profileName">
                                        <input type="text" class="actionRepeatValue" value="${item.actionRepeat}" placeholder="Define how many times the action will be repeated.">
                                    </label>
                                </div>`,
                                `<div class="system-input text-input">
                                    <label for="profileName">
                                        <input type="text" class="actionRepeatDelayValue" value="${item.actionRepeatDelay}" placeholder="The amount of delay in milliseconds between the Repeat action.">
                                    </label>
                                </div>`,
                                `<div class="settings-list">
                                    <div class="settings-row">
                                        <input class="system-button secondary updateMacroValue auto-width" id="updateMacroValue" data-id="${pf};${i}" type="button" value="${i18n.t('txtUpdate')}">
                                        <input class="system-button danger deleteMacroValue auto-width" id="deleteMacroValue" data-id="${pf};${i}" type="button" value="${i18n.t('txtDelete')}">
                                    </div>
                                </div>`,
                            ]).draw();
                        } else {
                            // Render row if we have actual key
                            getKeyName(item.actionCommand, function (result) {
                                dt.row.add([
                                    i,
                                    actionType,
                                    result,
                                    actionHold,
                                    `<div class="system-input text-input">
                                        <label for="profileName">
                                            <input type="text" class="actionRepeatValue" value="${item.actionRepeat}" placeholder="${i18n.t('txtRepeatAmount')}">
                                        </label>
                                    </div>`,
                                    `<div class="system-input text-input">
                                        <label for="profileName">
                                            <input type="text" class="actionRepeatDelayValue" value="${item.actionRepeatDelay}" placeholder="${i18n.t('txtRepeatDelay')}">
                                        </label>
                                    </div>`,
                                    `<div class="settings-list">
                                        <div class="settings-row">
                                            <input class="system-button secondary updateMacroValue auto-width" id="updateMacroValue" data-id="${pf};${i}" type="button" value="${i18n.t('txtUpdate')}">
                                            <input class="system-button danger deleteMacroValue auto-width" id="deleteMacroValue" data-id="${pf};${i}" type="button" value="${i18n.t('txtDelete')}">
                                        </div>
                                    </div>`,
                                ]).draw();
                            });
                        }
                    });
                    $("#deleteBtn").show();
                    $("#addMacroValueBtn").show();
                    $("#newMacroValue").show();
                }
            }
        });
    }

    $('.pressAndHoldMacroInfoToggle').on('click', function () {
        const modalPressAndHold = `
            <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel">
                <div class="modal-dialog">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtPressAndHold')}</h5>
                            <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                        </div>
                        <div class="modal-body">
                            <span>${i18n.t('txtPressAndHoldInfo')}</span>
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

    $('.actionRepeatMacroInfoToggle').on('click', function () {
        const modalPressAndHold = `
            <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel">
                <div class="modal-dialog">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtActionRepeat')}</h5>
                            <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                        </div>
                        <div class="modal-body">
                            <span>${i18n.t('txtRepeatInfo')}</span>
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

    $('.actionRepeatDelayInfoToggle').on('click', function () {
        const modalPressAndHold = `
            <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel">
                <div class="modal-dialog">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtActionRepeatDelay')}</h5>
                            <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                        </div>
                        <div class="modal-body">
                            <span>${i18n.t('txtRepeatDelayInfo')}</span>
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

    $('.macroList').on('click', function () {
        const profile = $(this).attr('id');
        $('.macroList').removeClass('selected');
        $(this).addClass('selected');

        let pf = parseInt(profile);
        loadMacroValues(pf);
    });

    $('#deleteMacroProfile').on('click', function () {
        const macroId = $("#profile").val();
        const pf = {};
        pf["macroId"] = parseInt(macroId);
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/macro/profile',
            type: 'DELETE',
            data: json,
            cache: false,
            success: function (response) {
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

    $('#addMacroValue').on('click', function () {
        const macroId = $("#profile").val();
        const macroType = $("#macroType").val();
        const macroValue = $("#macroKeyId").val();
        const macroDelay = $("#macroDelay").val();
        const macroText = $("#macroText").val();

        if (parseInt(macroType) === 0) {
            toast.warning(i18n.t('txtSelectMacroType'));
            return false;
        }

        if (parseInt(macroType) === 3 && parseInt(macroValue) === 0) {
            toast.warning(i18n.t('txtSelectMacroValue'));
            return false;
        }

        if (parseInt(macroType) === 5 && parseInt(macroDelay) < 1) {
            toast.warning(i18n.t('txtNoMacroDelayMs'));
            return false;
        }

        const pf = {};
        pf["macroId"] = parseInt(macroId);
        pf["macroType"] = parseInt(macroType);
        pf["macroValue"] = parseInt(macroValue);
        pf["macroDelay"] = parseInt(macroDelay);
        pf["macroText"] = macroText;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/macro/newValue',
            type: 'POST',
            data: json,
            cache: false,
            success: function (response) {
                try {
                    if (response.status === 1) {
                        //location.reload();
                        loadMacroValues(macroId);
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('#macroKeySearch').on('input', function () {
        const query = $(this).val().toLowerCase();
        const filtered = allOptions.filter(opt =>
            opt.text.toLowerCase().includes(query)
        );
        renderOptions(filtered);
    });

    function renderOptions(options) {
        const select = $('#macroKeyId');
        const currentValue = select.val();

        select.empty();
        options.forEach(opt => {
            const option = $('<option>', {
                value: opt.value,
                text: opt.text
            });

            if (opt.value === currentValue) {
                option.prop('selected', true);
            }

            select.append(option);
        });
        $(".macroKeyId").show();
    }

    function loadMacroOptions(url) {
        return $.get(url).then(result => {
            allOptions = Object.entries(result.data || {}).map(
                ([key, value]) => ({
                    value: key,
                    text: value.Name
                })
            );
            renderOptions(allOptions);
        });
    }

    $('#macroType').on('change', function () {
        const selectedValue = parseInt($(this).val());

        const mki = $(".macroKeyId")
        const mdi = $(".macroDelayId")
        const mti = $(".macroTextId")
        mki.hide();
        mdi.hide();
        mti.hide();

        switch (selectedValue) {
            case 3:
                loadMacroOptions('/api/input/keyboard');
                break;
            case 5:
                mdi.show();
                break;
            case 6:
                mti.show();
                break;
            case 9:
                loadMacroOptions('/api/input/mouse');
                break;
        }
    });

    function getKeyName(keyIndex, callback) {
        $.ajax({
            url: '/api/macro/keyInfo/' + parseInt(keyIndex),
            type: 'GET',
            cache: false,
            success: function (response) {
                if (response.status === 1) {
                    callback(response.data);
                } else {
                    callback('');
                }
            },
            error: function () {
                callback('');
            }
        });
    }
});