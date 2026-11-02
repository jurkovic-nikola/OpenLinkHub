"use strict";
$(document).ready(function () {
    window.i18n = {
        locale: null,
        values: {},

        setTranslations: function (locale, values) {
            this.locale = locale;
            this.values = values || {};
        },

        t: function (key) {
            return this.values[key] ?? key;
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
        },
        complete: function () {
            load();
        }
    });

    function load() {
        $.ajax({
            url: '/api/application/profiles',
            method: 'GET',
            dataType: 'json',
            success: function (response) {
                if (response.status !== 1 || !response.data) {
                    toast.warning(i18n.t('txtAppsLoadFailed'));
                    return;
                }

                $('.application-card').not('.application-new').each(function () {
                    const card = $(this);
                    const profile = response.data[card.attr('data-info')];

                    if (!profile || card.data('deleting')) {
                        return;
                    }

                    card.data('profile', profile);
                    card.find('.application-enabled').prop('checked', profile.enabled);
                    card.find('.application-manual').prop('checked', profile.manual);

                    const serials = Array.from(new Set(
                        Object.keys(profile.activeProfiles || {}).concat(
                            Object.keys(profile.passiveProfiles || {})
                        )
                    ));

                    const available = card.find('.device').first()
                        .find('option')
                        .map(function () {
                            return this.value;
                        }).get();

                    const assigned = serials.filter(function (serial) {
                        return serial !== '' && available.includes(serial);
                    });

                    while (card.find('.application-row').length < assigned.length) {
                        addDeviceRow(card);
                    }

                    card.find('.application-row').each(function (index) {
                        const row = $(this);
                        row.find('.device').val(assigned[index] || '');
                        loadDeviceProfiles(row);
                    });

                    card.find('.saveApplicationProfile').prop('disabled', false);
                });

                $('#newApplicationId option').each(function () {
                    const exists = this.value !== '' &&
                        Object.prototype.hasOwnProperty.call(response.data, this.value);

                    $(this).prop('disabled', exists).prop('hidden', exists);
                });
            },
            error: function () {
                toast.warning(i18n.t('txtAppsLoadFailed'));
            }
        });
    }

    function addDeviceRow(card) {
        const device = $('<select>', {
            class: 'system-select compact w-100 device'
        });

        card.find('.device').first().find('option').each(function () {
            device.append($('<option>', {
                value: this.value,
                text: $(this).text(),
                disabled: this.disabled
            }));
        });
        device.val('');

        const row = $('<div>', {
            class: 'settings-row application-row application-extra-row'
        }).append(device);

        $.each(['application-active', 'application-passive'], function (_, className) {
            row.append($('<select>', {
                class: 'system-select compact w-100 ' + className,
                disabled: true
            }).append($('<option>', {
                value: '',
                text: i18n.t('txtNone')
            })));
        });

        const label = $('<span>', {
            class: 'settings-label text-ellipsis application-extra-label',
            text: i18n.t('txtDeviceApps')
        });

        card.find('.application-row').last().after(label, row);
        return row;
    }

    function restoreProfileSelects(card) {
        const busy = !!(card.data('saving') || card.data('deleting'));

        card.find('.application-row').each(function () {
            const row = $(this);
            const disabled = busy || !row.find('.device').val() ||
                !!row.data('loading') || !!row.data('failed');

            row.find('.application-active, .application-passive')
                .prop('disabled', disabled);
        });
    }

    function loadDeviceProfiles(row) {
        const serial = row.find('.device').val();
        const selects = row.find('.application-active, .application-passive');
        const card = row.closest('.application-card');
        const profile = card.data('profile') || {};
        const requestId = (row.data('requestId') || 0) + 1;

        row.data('requestId', requestId);
        row.data('loading', false);
        row.data('failed', false);

        selects.empty().each(function () {
            $(this).append($('<option>', {
                value: '',
                text: i18n.t('txtNone')
            }));
        }).prop('disabled', true);

        if (!serial) {
            return;
        }

        row.data('loading', true);

        $.ajax({
            url: '/api/devices/' + encodeURIComponent(serial),
            method: 'GET',
            dataType: 'json',
            success: function (response) {
                if (row.data('requestId') !== requestId) {
                    return;
                }

                if (!response.device || !response.device.userProfiles) {
                    row.data('failed', true);
                    toast.warning(i18n.t('txtAppsProfilesUnavailable'));
                    return;
                }

                const profiles = response.device.userProfiles;

                selects.each(function () {
                    const select = $(this);
                    const assignments = select.hasClass('application-active')
                        ? profile.activeProfiles
                        : profile.passiveProfiles;
                    const selected = (assignments || {})[serial] || '';

                    $.each(profiles, function (name) {
                        select.append($('<option>', {
                            value: name,
                            text: name
                        }));
                    });

                    if (selected && !Object.prototype.hasOwnProperty.call(profiles, selected)) {
                        select.append($('<option>', {
                            value: selected,
                            text: selected + i18n.t('txtAppsUnavailableSuffix')
                        }));
                    }

                    select.val(selected);
                });
            },
            error: function () {
                if (row.data('requestId') !== requestId) {
                    return;
                }

                row.data('failed', true);
                toast.warning(i18n.t('txtAppsProfilesUnavailable'));
            },
            complete: function () {
                if (row.data('requestId') === requestId) {
                    row.data('loading', false);
                    restoreProfileSelects(card);
                }
            }
        });
    }

    $(document).on('change', '.application-card .device', function () {
        loadDeviceProfiles($(this).closest('.application-row'));
    });

    $('#newApplication').on('click', function () {
        const existing = new Set(
            $('.application-card').not('.application-new').map(function () {
                return $(this).attr('data-info');
            }).get()
        );

        $('#newApplicationId option').each(function () {
            const exists = existing.has(this.value);
            $(this).prop('disabled', exists).prop('hidden', exists);
        });

        $('#newApplicationBox').removeClass('d-none');
        $(this).prop('disabled', true);
        $('#applicationSearch').val('').trigger('input').trigger('focus');
    });

    $('#newApplicationId').on('change', function () {
        const card = $(this).closest('.application-card');
        const applicationId = $(this).val() || '';

        card.data('profile', {
            applicationId: applicationId,
            applicationName: applicationId ? $(this).find('option:selected').text() : '',
            activeProfiles: {},
            passiveProfiles: {}
        });

        card.find('.saveApplicationProfile')
            .attr('data-info', applicationId)
            .prop('disabled', !applicationId);
    });

    $('#cancelNewApplication').on('click', function () {
        const card = $(this).closest('.application-card');

        if (card.data('saving')) {
            return;
        }

        card.removeData('profile');
        card.find('.application-enabled').prop('checked', true);
        card.find('.application-manual').prop('checked', false);
        $('#applicationSearch').val('').trigger('input');

        card.find('.application-row').each(function () {
            const row = $(this);
            row.find('.device').val('');
            loadDeviceProfiles(row);
        });

        card.find('.application-extra-label, .application-extra-row').remove();

        card.find('.saveApplicationProfile')
            .attr('data-info', '')
            .prop('disabled', true);

        $('#newApplicationBox').addClass('d-none');
        $('#newApplication').prop('disabled', false);
    });

    $('.addApplicationDevice').on('click', function () {
        const card = $(this).closest('.application-card');

        if (card.data('saving') || card.data('deleting')) {
            return;
        }

        addDeviceRow(card).find('.device').trigger('focus');
    });

    $('.saveApplicationProfile').on('click', function () {
        const button = $(this);
        const card = button.closest('.application-card');
        const profile = card.data('profile');
        const applicationId = button.attr('data-info');
        const isNew = card.hasClass('application-new');

        if (!profile || !applicationId || card.data('saving') || card.data('deleting')) {
            return;
        }

        const selected = new Set();
        const activeProfiles = {};
        const passiveProfiles = {};
        let ready = true;

        card.find('.application-row').each(function () {
            const row = $(this);
            const serial = row.find('.device').val();

            if (row.data('loading') || row.data('failed')) {
                toast.warning(i18n.t('txtAppsProfilesUnavailable'));
                ready = false;
                return false;
            }

            if (!serial) {
                return;
            }

            if (selected.has(serial)) {
                toast.warning(i18n.t('txtAppsDuplicateDevice'));
                ready = false;
                return false;
            }

            selected.add(serial);

            const active = row.find('.application-active').val();
            const passive = row.find('.application-passive').val();

            if (active) {
                activeProfiles[serial] = active;
            }

            if (passive) {
                passiveProfiles[serial] = passive;
            }
        });

        if (!ready) {
            return;
        }

        const pf = {};
        pf["applicationId"] = applicationId;
        pf["applicationName"] = profile.applicationName;
        pf["enabled"] = card.find('.application-enabled').prop('checked');
        pf["manual"] = card.find('.application-manual').prop('checked');
        pf["activeProfiles"] = activeProfiles;
        pf["passiveProfiles"] = passiveProfiles;

        const json = JSON.stringify(pf, null, 2);

        card.data('saving', true);
        card.find('input, select, button').prop('disabled', true);

        $.ajax({
            url: isNew ? '/api/application/profiles/new' : '/api/application/profiles/save',
            method: 'POST',
            contentType: 'application/json',
            dataType: 'json',
            data: json,
            success: function (response) {
                if (response.status === 1) {
                    if (isNew) {
                        window.location.reload();
                        return;
                    }

                    card.data('profile', pf);
                    toast.success(response.message);
                } else {
                    toast.warning(response.message);
                }
            },
            error: function () {
                toast.warning(i18n.t('txtAppsSaveFailed'));
            },
            complete: function () {
                card.data('saving', false);

                card.find(
                    '.application-enabled, .application-manual, .device, .saveApplicationProfile, ' +
                    '.deleteApplicationProfile, .addApplicationDevice, ' +
                    '#newApplicationId, #cancelNewApplication, #applicationSearch'
                ).prop('disabled', false);

                card.find('.saveApplicationProfile')
                    .prop('disabled', !card.find('.saveApplicationProfile').attr('data-info'));

                restoreProfileSelects(card);
            }
        });
    });

    $('.deleteApplicationProfile').on('click', function () {
        const button = $(this);
        const card = button.closest('.application-card');
        const applicationId = button.attr('data-info') || card.attr('data-info');

        if (!applicationId || card.data('saving') || card.data('deleting')) {
            return;
        }

        const enabledControls = card.find('input, select, button').filter(':enabled');

        const pf = {};
        pf["applicationId"] = applicationId;

        const json = JSON.stringify(pf, null, 2);

        card.data('deleting', true);
        card.find('input, select, button').prop('disabled', true);

        $.ajax({
            url: '/api/application/profiles/delete',
            method: 'DELETE',
            contentType: 'application/json',
            dataType: 'json',
            data: json,
            success: function (response) {
                if (response.status !== 1) {
                    toast.warning(response.message);
                    return;
                }

                card.find('.application-row').each(function () {
                    const row = $(this);
                    row.data('requestId', (row.data('requestId') || 0) + 1);
                });

                card.closest('.col-md-3').remove();

                $('#newApplicationId option').each(function () {
                    if (this.value === applicationId) {
                        $(this).prop('disabled', false).prop('hidden', false);
                    }
                });

                toast.success(response.message);
            },
            error: function () {
                toast.warning(i18n.t('txtAppsDeleteFailed'));
            },
            complete: function () {
                card.data('deleting', false);
                enabledControls.prop('disabled', false);

                if (card.data('profile')) {
                    card.find('.saveApplicationProfile').prop('disabled', false);
                }

                restoreProfileSelects(card);
            }
        });
    });

    $('#applicationSearch').on('input', function () {
        const search = $(this).val().trim().toLowerCase();
        const select = $('#newApplicationId');
        const selected = select.val();

        select.find('option').each(function () {
            if (!this.value) {
                return;
            }

            const matches = $(this).text().toLowerCase().includes(search);

            $(this).prop('hidden',
                this.disabled || (!matches && this.value !== selected)
            );
        });
    });

    $('.manualActivationToggle').on('click', function () {
        const modalDefault = `
            <div class="modal fade text-start" id="infoToggle" tabindex="-1" aria-labelledby="infoToggleLabel">
                <div class="modal-dialog">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title" id="infoToggleLabel">${i18n.t('txtManualActivation')}</h5>
                            <button class="btn-close btn-close-white" type="button" data-bs-dismiss="modal" aria-label="Close"></button>
                        </div>
                        <div class="modal-body">
                            <span>${i18n.t('txtManualActivationInfo')}</span>
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
});