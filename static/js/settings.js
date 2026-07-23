"use strict";
$(document).ready(function () {
    // Init dataTable
    const dt = $('#dataTable').DataTable(
        {
            order: [[0, 'asc']],
            select: {
                style: 'os',
                selector: 'td:first-child'
            },
            paging: true,
            searching: true,
            language: {
                emptyTable: "Supported device list",
                searchPlaceholder: "Search for device..."
            },
            layout: {
                topStart: null,
                topEnd: 'search',
                bottomStart: ['paging', 'info'],
                bottomEnd: 'pageLength'
            },
            columns: [
                { data: 'ProductId', title: 'Product Id' },
                {
                    data: 'ProductId',
                    title: 'Product Id - Hexadecimal',
                    render: function(data, type, row, meta) {
                        return '0x' + Number(data).toString(16).toUpperCase().padStart(4, '0');
                    }
                },
                { data: 'Name', title: 'Product Name' }, // JSON uses Name
                {
                    data: 'Enabled',
                    title: 'Enabled',
                    orderable: false,
                    render: function(data, type, row, meta) {
                        const checked = data ? 'checked' : '';
                        return `
                            <label class="system-toggle compact">
                                <input type="checkbox" class="device-checkbox" data-id="${row.ProductId}" ${checked}>
                                <span class="toggle-track"></span>
                            </label>
                        `;
                    }
                }
            ]
        }
    );

    function initializeOpenRGBImportUI() {
        const openRGBState = {
            modalOpen: false,
            activeRequest: null,
            latestDiscoveryData: null,
            selectedImportKeys: new Set(),
            selectedRemovalSerials: new Set()
        };

        const buttonStates = [
            {
                selector: '#btnOpenRGBDiscover',
                request: 'discover',
                idleLabel: 'Discover & Manage Controllers',
                busyLabel: 'Discovering...'
            },
            {
                selector: '#btnOpenRGBDiscoverAgain',
                request: 'discover',
                idleLabel: 'Discover Again',
                busyLabel: 'Discovering...'
            },
            {
                selector: '#btnOpenRGBImportSelected',
                request: 'import',
                idleLabel: 'Import Selected',
                busyLabel: 'Importing...'
            },
            {
                selector: '#btnOpenRGBRemoveSelected',
                request: 'remove',
                idleLabel: 'Remove Selected',
                busyLabel: 'Removing...'
            },
            {
                selector: '#btnOpenRGBRefresh',
                request: 'refresh',
                idleLabel: 'Refresh Imported Controllers',
                busyLabel: 'Refreshing...'
            }
        ];

        const loadingMessages = {
            discover: 'Discovering controllers from the local OpenRGB SDK server...',
            import: 'Importing the selected OpenRGB controllers...',
            remove: 'Removing the selected live OpenRGB imports...',
            refresh: 'Refreshing imported OpenRGB controllers...'
        };

        function responseMessage(response, fallback) {
            if (response && typeof response.message === 'string' && response.message.length > 0) {
                return response.message;
            }
            return fallback;
        }

        function isOpenRGBResponse(response) {
            return $.isPlainObject(response) && (response.status === 0 || response.status === 1);
        }

        function setOpenRGBCardStatus(message, level) {
            const status = $('#openrgbCardStatus');
            status
                .removeClass('text-success text-warning text-danger text-muted')
                .addClass(level === 'success' ? 'text-success' :
                    level === 'warning' ? 'text-warning' :
                        level === 'danger' ? 'text-danger' : 'text-muted')
                .text(message);
        }

        function setOpenRGBDiscoveryAlert(message, level) {
            const alert = $('#openrgbDiscoveryAlert');
            alert
                .removeClass('alert-secondary alert-success alert-warning alert-danger')
                .addClass(level === 'success' ? 'alert-success' :
                    level === 'warning' ? 'alert-warning' :
                        level === 'danger' ? 'alert-danger' : 'alert-secondary')
                .text(message);
        }

        function updateOpenRGBActionButtons() {
            const busy = openRGBState.activeRequest !== null;

            buttonStates.forEach(function (buttonState) {
                const button = $(buttonState.selector);
                const isActiveButton = openRGBState.activeRequest === buttonState.request;
                button
                    .text(isActiveButton ? buttonState.busyLabel : buttonState.idleLabel)
                    .prop('disabled', busy);
            });

            if (!busy) {
                $('#btnOpenRGBImportSelected').prop('disabled', openRGBState.selectedImportKeys.size === 0);
                $('#btnOpenRGBRemoveSelected').prop('disabled', openRGBState.selectedRemovalSerials.size === 0);
            }

            $('.openrgb-import-checkbox, .openrgb-remove-checkbox').prop('disabled', busy);
            $('#openrgbImportModal').attr('aria-busy', busy ? 'true' : 'false');
        }

        function setOpenRGBBusy(request) {
            const busy = request !== null;
            const loadingState = $('#openrgbLoadingState');

            openRGBState.activeRequest = request;
            loadingState.toggleClass('d-none', !busy).toggleClass('d-flex', busy);
            $('#openrgbLoadingMessage').text(busy ? loadingMessages[request] : '');
            updateOpenRGBActionButtons();
        }

        function beginOpenRGBRequest(request) {
            if (openRGBState.activeRequest !== null) {
                return false;
            }
            setOpenRGBBusy(request);
            return true;
        }

        function finishOpenRGBRequest(request) {
            if (openRGBState.activeRequest === request) {
                setOpenRGBBusy(null);
            }
        }

        function resetOpenRGBSelections() {
            openRGBState.selectedImportKeys.clear();
            openRGBState.selectedRemovalSerials.clear();
            $('.openrgb-import-checkbox, .openrgb-remove-checkbox').prop('checked', false);
            updateOpenRGBActionButtons();
        }

        function checkedOpenRGBValues(selector) {
            const values = [];
            $(selector + ':checked').each(function () {
                const value = $(this).val();
                if (typeof value === 'string' && value.length > 0) {
                    values.push(value);
                }
            });
            return Array.from(new Set(values));
        }

        function syncOpenRGBSelections() {
            openRGBState.selectedImportKeys = new Set(
                checkedOpenRGBValues('.openrgb-import-checkbox')
            );
            openRGBState.selectedRemovalSerials = new Set(
                checkedOpenRGBValues('.openrgb-remove-checkbox')
            );
            updateOpenRGBActionButtons();
        }

        function appendOpenRGBDetail(container, label, value, useCode) {
            if (value === null || typeof value === 'undefined' || value === '') {
                return;
            }

            const row = $('<div>').addClass('d-flex flex-wrap justify-content-between gap-2');
            const labelElement = $('<span>').addClass('text-muted').text(label);
            const valueElement = useCode ? $('<code>') : $('<span>');

            valueElement.addClass('text-break text-end').text(String(value));
            row.append(labelElement, valueElement);
            container.append(row);
        }

        function renderConfiguredImports(configured) {
            const container = $('#openrgbConfiguredImports');
            container.empty();

            if (!Array.isArray(configured) || configured.length === 0) {
                container.append(
                    $('<div>').addClass('settings-row').append(
                        $('<span>').addClass('settings-label').text(
                            'No OpenRGB imports are currently configured.'
                        )
                    )
                );
                return;
            }

            configured.forEach(function (entry) {
                const summary = $.isPlainObject(entry) ? entry : {};
                const product = typeof summary.product === 'string' && summary.product.length > 0 ?
                    summary.product : 'Imported OpenRGB controller';
                const serial = typeof summary.serial === 'string' ? summary.serial : '';
                const disabled = summary.disabled === true;
                const row = $('<article>').addClass(
                    'settings-row flex-column flex-lg-row align-items-start'
                );
                const details = $('<div>').addClass('flex-grow-1 w-100');
                const heading = $('<h6>').addClass('mb-2 text-break').text(product);
                const metadata = $('<div>').addClass('d-grid gap-1 small text-start');
                const controls = $('<div>').addClass(
                    'd-flex flex-wrap align-items-center gap-2 flex-shrink-0'
                );
                const badge = $('<span>').addClass(
                    disabled ? 'badge text-bg-secondary' : 'badge text-bg-success'
                ).text(disabled ? 'Preserved · Not imported' : 'Enabled');

                appendOpenRGBDetail(metadata, 'Internal serial', serial || 'Not reported', true);
                details.append(heading, metadata);
                controls.append(badge);

                if (disabled) {
                    controls.append(
                        $('<span>').addClass('small text-muted').text(
                            'Not imported. Retained configuration can be reused if this controller is discovered again.'
                        )
                    );
                } else if (serial.length > 0) {
                    const label = $('<label>').addClass(
                        'd-flex align-items-center gap-2 mb-0'
                    );
                    const checkbox = $('<input>')
                        .attr('type', 'checkbox')
                        .addClass('form-check-input openrgb-remove-checkbox')
                        .val(serial);
                    const labelText = $('<span>').text('Select for removal');

                    checkbox.attr('aria-label', 'Select ' + product + ' for removal');
                    label.append(checkbox, labelText);
                    controls.append(label);
                } else {
                    controls.append(
                        $('<span>').addClass('small text-warning').text(
                            'This import cannot be selected because no internal serial was returned.'
                        )
                    );
                }

                row.append(details, controls);
                container.append(row);
            });
        }

        function openRGBIdentityLabel(identityKind) {
            switch (identityKind) {
            case 'external-serial':
                return 'Unique external serial';
            case 'location-product-vendor':
                return 'Unique device location and product';
            case 'product-vendor-name':
                return 'Unique product and vendor';
            default:
                return '';
            }
        }

        function openRGBControllerState(state) {
            switch (state) {
            case 'selectable':
                return {label: 'Available', badgeClass: 'text-bg-success'};
            case 'imported':
                return {label: 'Imported', badgeClass: 'text-bg-primary'};
            case 'ambiguous':
                return {label: 'Ambiguous', badgeClass: 'text-bg-warning'};
            default:
                return {label: 'Invalid', badgeClass: 'text-bg-danger'};
            }
        }

        function renderOpenRGBZones(container, zones) {
            if (!Array.isArray(zones) || zones.length === 0) {
                return;
            }

            const heading = $('<div>').addClass('small text-muted mt-2 mb-1').text('Zones');
            const list = $('<ul>').addClass('list-unstyled d-grid gap-1 mb-0');

            zones.forEach(function (entry) {
                const zone = $.isPlainObject(entry) ? entry : {};
                const item = $('<li>').addClass(
                    'd-flex flex-wrap justify-content-between gap-2 small'
                );
                const name = $('<span>').addClass('text-break').text(
                    typeof zone.name === 'string' && zone.name.length > 0 ?
                        zone.name : 'Unnamed zone'
                );
                const summary = $('<span>').addClass('text-muted text-end');
                const parts = [];

                if (Number.isFinite(zone.ledCount)) {
                    parts.push(String(zone.ledCount) + ' LEDs');
                }
                if (typeof zone.classification === 'string' && zone.classification.length > 0) {
                    parts.push(zone.classification);
                }

                summary.text(parts.length > 0 ? parts.join(' · ') : 'No zone details reported');
                item.append(name, summary);
                list.append(item);
            });

            container.append(heading, list);
        }

        function renderControllerPreviews(controllers) {
            const container = $('#openrgbControllerPreviews');
            container.empty();

            if (!Array.isArray(controllers) || controllers.length === 0) {
                container.append(
                    $('<div>').addClass('settings-row').append(
                        $('<span>').addClass('settings-label').text(
                            'No controllers were reported by the local OpenRGB SDK server.'
                        )
                    )
                );
                return;
            }

            controllers.forEach(function (entry) {
                const controller = $.isPlainObject(entry) ? entry : {};
                const state = typeof controller.state === 'string' ? controller.state : 'invalid';
                const statePresentation = openRGBControllerState(state);
                const key = typeof controller.key === 'string' ? controller.key : '';
                const selectable = state === 'selectable' && key.length > 0;
                const product = typeof controller.product === 'string' && controller.product.length > 0 ?
                    controller.product : 'Unnamed OpenRGB controller';
                const row = $('<article>').addClass(
                    'settings-row flex-column align-items-stretch text-start'
                );
                const header = $('<div>').addClass(
                    'd-flex flex-wrap align-items-start justify-content-between gap-2 w-100'
                );
                const identity = $('<div>').addClass('d-flex align-items-start gap-2 flex-grow-1');
                const headingGroup = $('<div>').addClass('flex-grow-1');
                const heading = $('<h6>').addClass('mb-1 text-break').text(product);
                const badge = $('<span>').addClass(
                    'badge ' + statePresentation.badgeClass
                ).text(statePresentation.label);
                const metadata = $('<div>').addClass(
                    'd-grid gap-1 small w-100'
                );

                if (selectable) {
                    const label = $('<label>').addClass(
                        'd-flex align-items-center gap-2 mb-0'
                    );
                    const checkbox = $('<input>')
                        .attr('type', 'checkbox')
                        .addClass('form-check-input openrgb-import-checkbox')
                        .val(key);
                    const labelText = $('<span>').addClass('visually-hidden').text(
                        'Select ' + product + ' for import'
                    );

                    checkbox.attr('aria-label', 'Select ' + product + ' for import');
                    label.append(checkbox, labelText);
                    identity.append(label);
                } else {
                    const label = $('<label>').addClass(
                        'd-flex align-items-center gap-2 mb-0'
                    );
                    const checkbox = $('<input>')
                        .attr({
                            type: 'checkbox',
                            disabled: true,
                            'aria-label': statePresentation.label + ' controller cannot be selected for import'
                        })
                        .addClass('form-check-input');
                    const labelText = $('<span>').addClass('visually-hidden').text(
                        statePresentation.label + ' controller cannot be selected for import'
                    );

                    label.append(checkbox, labelText);
                    identity.append(label);
                    row.addClass('opacity-75').attr('aria-disabled', 'true');
                }

                headingGroup.append(heading);
                identity.append(headingGroup);
                header.append(identity, badge);

                appendOpenRGBDetail(metadata, 'Vendor', controller.vendor, false);
                appendOpenRGBDetail(metadata, 'Version', controller.version, false);
                appendOpenRGBDetail(metadata, 'Description', controller.description, false);

                if (typeof controller.displaySerial === 'string' &&
                    controller.displaySerial.length > 0) {
                    const displaySerialLabel =
                        typeof controller.displaySerialLabel === 'string' &&
                        controller.displaySerialLabel.length > 0 ?
                            controller.displaySerialLabel : 'Display serial';
                    appendOpenRGBDetail(
                        metadata,
                        displaySerialLabel,
                        controller.displaySerial,
                        true
                    );
                }

                appendOpenRGBDetail(
                    metadata,
                    'Zones',
                    Number.isFinite(controller.zoneCount) ? controller.zoneCount : 'Not reported',
                    false
                );
                appendOpenRGBDetail(
                    metadata,
                    'LEDs',
                    Number.isFinite(controller.ledCount) ? controller.ledCount : 'Not reported',
                    false
                );

                const identityLabel = openRGBIdentityLabel(controller.identityKind);
                appendOpenRGBDetail(metadata, 'Identity', identityLabel, false);

                if (state === 'imported' &&
                    typeof controller.configuredSerial === 'string' &&
                    controller.configuredSerial.length > 0) {
                    appendOpenRGBDetail(
                        metadata,
                        'Configured internal serial',
                        controller.configuredSerial,
                        true
                    );
                }

                renderOpenRGBZones(metadata, controller.zones);

                if ((state === 'ambiguous' || state === 'invalid') &&
                    typeof controller.reason === 'string' &&
                    controller.reason.length > 0) {
                    metadata.append(
                        $('<div>')
                            .addClass(state === 'invalid' ?
                                'alert alert-danger py-2 px-3 mt-2 mb-0' :
                                'alert alert-warning py-2 px-3 mt-2 mb-0')
                            .text(controller.reason)
                    );
                }

                row.append(header, metadata);
                container.append(row);
            });
        }

        function renderOpenRGBDiscovery(data) {
            const discovery = $.isPlainObject(data) ? data : {};
            const configured = Array.isArray(discovery.configured) ? discovery.configured : [];
            const controllers = Array.isArray(discovery.controllers) ? discovery.controllers : [];
            const activeConfigured = configured.filter(function (entry) {
                return $.isPlainObject(entry) && entry.disabled !== true;
            }).length;
            const preservedConfigured = configured.filter(function (entry) {
                return $.isPlainObject(entry) && entry.disabled === true;
            }).length;
            const availableControllers = controllers.filter(function (entry) {
                return $.isPlainObject(entry) &&
                    entry.state === 'selectable' &&
                    typeof entry.key === 'string' &&
                    entry.key.length > 0;
            }).length;

            openRGBState.latestDiscoveryData = {
                discoveryState: typeof discovery.discoveryState === 'string' ?
                    discovery.discoveryState : '',
                error: typeof discovery.error === 'string' ? discovery.error : '',
                configured: configured,
                controllers: controllers
            };

            resetOpenRGBSelections();
            renderConfiguredImports(configured);
            renderControllerPreviews(controllers);
            $('#openrgbCardSummary').text(
                configured.length + ' configured (' + activeConfigured + ' imported, ' +
                preservedConfigured + ' preserved); ' + controllers.length +
                ' discovered (' + availableControllers + ' available).'
            );
        }

        function showMalformedOpenRGBResponse() {
            const message = 'The server returned a malformed OpenRGB response.';
            setOpenRGBDiscoveryAlert(message, 'danger');
            setOpenRGBCardStatus(message, 'danger');
            toastr.error(message);
        }

        function handleOpenRGBTransportFailure(xhr, textStatus) {
            let message;
            if (textStatus === 'parsererror') {
                showMalformedOpenRGBResponse();
                return;
            }
            if (!xhr || xhr.status === 0) {
                message = 'Unable to connect to LumenForge for the OpenRGB request.';
            } else {
                message = 'The OpenRGB request could not be completed by LumenForge (HTTP ' +
                    xhr.status + ').';
            }
            setOpenRGBDiscoveryAlert(message, 'danger');
            setOpenRGBCardStatus(message, 'danger');
            toastr.error(message);
        }

        function discoverOpenRGBControllers() {
            if (!beginOpenRGBRequest('discover')) {
                return;
            }

            resetOpenRGBSelections();
            setOpenRGBDiscoveryAlert(
                'Contacting the local OpenRGB SDK server...',
                'secondary'
            );
            setOpenRGBCardStatus('OpenRGB controller discovery is in progress.', 'muted');

            $.ajax({
                url: '/api/openrgbimport/discover',
                type: 'POST',
                data: JSON.stringify({}),
                contentType: 'application/json',
                dataType: 'json',
                cache: false
            }).done(function (response) {
                if (!isOpenRGBResponse(response)) {
                    showMalformedOpenRGBResponse();
                    return;
                }

                if ($.isPlainObject(response.data)) {
                    renderOpenRGBDiscovery(response.data);
                }

                const message = responseMessage(
                    response,
                    response.status === 1 ?
                        'OpenRGB controller discovery completed.' :
                        'OpenRGB controller discovery could not be completed.'
                );

                if (response.status === 1) {
                    const noControllers = openRGBState.latestDiscoveryData &&
                        openRGBState.latestDiscoveryData.controllers.length === 0;
                    setOpenRGBDiscoveryAlert(message, noControllers ? 'secondary' : 'success');
                    setOpenRGBCardStatus(message, 'success');
                } else {
                    const conflict = openRGBState.latestDiscoveryData &&
                        openRGBState.latestDiscoveryData.discoveryState === 'conflict';
                    setOpenRGBDiscoveryAlert(message, conflict ? 'danger' : 'warning');
                    setOpenRGBCardStatus(message, conflict ? 'danger' : 'warning');
                    toast.warning(message);
                }
            }).fail(function (xhr, textStatus) {
                handleOpenRGBTransportFailure(xhr, textStatus);
            }).always(function () {
                finishOpenRGBRequest('discover');
            });
        }

        function importSelectedOpenRGBControllers() {
            const keys = checkedOpenRGBValues('.openrgb-import-checkbox');
            openRGBState.selectedImportKeys = new Set(keys);

            if (keys.length === 0) {
                const message = 'Select at least one available OpenRGB controller to import.';
                setOpenRGBDiscoveryAlert(message, 'warning');
                toast.warning(message);
                updateOpenRGBActionButtons();
                return;
            }
            if (!beginOpenRGBRequest('import')) {
                return;
            }

            setOpenRGBDiscoveryAlert('Importing the selected OpenRGB controllers...', 'secondary');
            setOpenRGBCardStatus('Selected OpenRGB controllers are being imported.', 'muted');

            $.ajax({
                url: '/api/openrgbimport/import',
                type: 'POST',
                data: JSON.stringify({keys: keys}),
                contentType: 'application/json',
                dataType: 'json',
                cache: false
            }).done(function (response) {
                if (!isOpenRGBResponse(response)) {
                    showMalformedOpenRGBResponse();
                    return;
                }

                const message = responseMessage(
                    response,
                    response.status === 1 ?
                        'OpenRGB controllers imported.' :
                        'The selected OpenRGB controllers could not be imported.'
                );

                if (response.status === 1) {
                    if ($.isPlainObject(response.data) &&
                        Array.isArray(response.data.controllers)) {
                        $('#openrgbCardSummary').text(
                            response.data.controllers.length +
                            ' controller import result(s) received. Reloading Settings...'
                        );
                    }
                    setOpenRGBDiscoveryAlert(message, 'success');
                    setOpenRGBCardStatus(message, 'success');
                    toast.success(message);
                    window.location.reload();
                } else {
                    setOpenRGBDiscoveryAlert(message, 'warning');
                    setOpenRGBCardStatus(message, 'warning');
                    toast.warning(message);
                }
            }).fail(function (xhr, textStatus) {
                handleOpenRGBTransportFailure(xhr, textStatus);
            }).always(function () {
                finishOpenRGBRequest('import');
            });
        }

        function configuredSelectionDetails(serials) {
            const configured = openRGBState.latestDiscoveryData &&
                Array.isArray(openRGBState.latestDiscoveryData.configured) ?
                openRGBState.latestDiscoveryData.configured : [];

            return serials.map(function (serial) {
                const match = configured.find(function (entry) {
                    return $.isPlainObject(entry) &&
                        entry.disabled !== true &&
                        entry.serial === serial;
                });
                return {
                    serial: serial,
                    product: match && typeof match.product === 'string' &&
                        match.product.length > 0 ? match.product : 'OpenRGB import'
                };
            });
        }

        function removeSelectedOpenRGBImports() {
            const serials = checkedOpenRGBValues('.openrgb-remove-checkbox');
            openRGBState.selectedRemovalSerials = new Set(serials);

            if (serials.length === 0) {
                const message = 'Select at least one imported OpenRGB controller to remove.';
                setOpenRGBDiscoveryAlert(message, 'warning');
                toast.warning(message);
                updateOpenRGBActionButtons();
                return;
            }

            const selections = configuredSelectionDetails(serials);
            const selectionText = selections.map(function (selection) {
                return selection.product + ' — ' + selection.serial;
            }).join('\n');
            const confirmed = window.confirm(
                'Remove these live OpenRGB imports?\n\n' +
                selectionText +
                '\n\nTheir profiles and RGB configuration files will be preserved for later reimport.'
            );

            if (!confirmed || !beginOpenRGBRequest('remove')) {
                return;
            }

            setOpenRGBDiscoveryAlert('Removing the selected live OpenRGB imports...', 'secondary');
            setOpenRGBCardStatus('Selected OpenRGB imports are being removed.', 'muted');

            $.ajax({
                url: '/api/openrgbimport/remove',
                type: 'POST',
                data: JSON.stringify({serials: serials}),
                contentType: 'application/json',
                dataType: 'json',
                cache: false
            }).done(function (response) {
                if (!isOpenRGBResponse(response)) {
                    showMalformedOpenRGBResponse();
                    return;
                }

                const message = responseMessage(
                    response,
                    response.status === 1 ?
                        'OpenRGB imports removed.' :
                        'The selected OpenRGB imports could not be removed.'
                );

                if (response.status === 1) {
                    setOpenRGBDiscoveryAlert(message, 'success');
                    setOpenRGBCardStatus(message, 'success');
                    toast.success(message);
                    window.location.reload();
                } else {
                    setOpenRGBDiscoveryAlert(message, 'warning');
                    setOpenRGBCardStatus(message, 'warning');
                    toast.warning(message);
                }
            }).fail(function (xhr, textStatus) {
                handleOpenRGBTransportFailure(xhr, textStatus);
            }).always(function () {
                finishOpenRGBRequest('remove');
            });
        }

        function refreshOpenRGBImports() {
            if (!beginOpenRGBRequest('refresh')) {
                return;
            }

            setOpenRGBCardStatus('Refreshing imported OpenRGB controllers...', 'muted');

            $.ajax({
                url: '/api/openrgbimport/refresh',
                type: 'POST',
                data: JSON.stringify({}),
                contentType: 'application/json',
                dataType: 'json',
                cache: false
            }).done(function (response) {
                if (!isOpenRGBResponse(response)) {
                    showMalformedOpenRGBResponse();
                    return;
                }

                const message = responseMessage(
                    response,
                    response.status === 1 ?
                        'OpenRGB import refresh requested.' :
                        'OpenRGB imports are not configured for refresh.'
                );

                if (response.status === 1) {
                    setOpenRGBCardStatus(message, 'success');
                    toast.success(message);
                } else {
                    setOpenRGBCardStatus(message, 'warning');
                    toast.warning(message);
                }
            }).fail(function (xhr, textStatus) {
                handleOpenRGBTransportFailure(xhr, textStatus);
            }).always(function () {
                finishOpenRGBRequest('refresh');
            });
        }

        $('#btnOpenRGBDiscover').on('click', function () {
            if (openRGBState.activeRequest !== null) {
                return;
            }
            $('#openrgbImportModal').modal('show');
            discoverOpenRGBControllers();
        });
        $('#btnOpenRGBDiscoverAgain').on('click', discoverOpenRGBControllers);
        $('#btnOpenRGBImportSelected').on('click', importSelectedOpenRGBControllers);
        $('#btnOpenRGBRemoveSelected').on('click', removeSelectedOpenRGBImports);
        $('#btnOpenRGBRefresh').on('click', refreshOpenRGBImports);
        $('#openrgbConfiguredImports').on(
            'change',
            '.openrgb-remove-checkbox',
            syncOpenRGBSelections
        );
        $('#openrgbControllerPreviews').on(
            'change',
            '.openrgb-import-checkbox',
            syncOpenRGBSelections
        );
        $('#openrgbImportModal')
            .on('show.bs.modal', function () {
                openRGBState.modalOpen = true;
            })
            .on('shown.bs.modal', function () {
                $('#openrgbImportModalLabel').trigger('focus');
            })
            .on('hidden.bs.modal', function () {
                openRGBState.modalOpen = false;
            });

        updateOpenRGBActionButtons();
    }

    initializeOpenRGBImportUI();


    $("#btnBackup").on("click", function() {
        window.location.href = "/api/backup";
    });

    $('.saveRgbControl').on('click', function () {
        const rgbControl = $("#rgbControl").is(':checked');
        const rgbOff = $("#rgbOff").val();
        const rgbOn = $("#rgbOn").val();
        const lcdControl = $("#lcdControl").is(':checked');

        const pf = {};
        pf["rgbControl"] = rgbControl;
        pf["rgbOff"] = rgbOff;
        pf["rgbOn"] = rgbOn;
        pf["lcdControl"] = lcdControl;

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/scheduler/rgb',
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
    
    $("#restoreForm").on("submit", function (e) {
        e.preventDefault();

        var formData = new FormData();
        var file = $("#backupFile")[0].files[0];
        if (!file) {
            toast.warning('Please select a .zip file first!');
            return;
        }
        formData.append("backupFile", file);

        $.ajax({
            url: "/api/restore",
            type: "POST",
            data: formData,
            processData: false,
            contentType: false,
            success: function (response) {
                toast.success(response);
            },
            error: function (xhr) {
                toast.warning("Restore failed: " + xhr.responseText);
            }
        });
    });

    $.ajax({
        url: '/api/getSupportedDevices',
        dataType: 'JSON',
        success: function(response) {
            if (response.code === 0) {
                toast.warning(response.message);
            } else {
                dt.clear();
                dt.rows.add(response.data);
                dt.draw();
            }
        }
    });

    dt.on('change', '.device-checkbox', function() {
        const productId = $(this).data('id');
        const enabled = $(this).prop('checked');

        // Optional: store in DataTables row data if needed
        const row = dt.row($(this).closest('tr'));
        const rowData = row.data();
        rowData.Enabled = enabled;
        row.data(rowData); // update row
    });

    $('#btnSaveSupportedDevices').on('click', function() {
        const supportedDevices = {};
        const pf = {};
        dt.rows().every(function() {
            const data = this.data();
            supportedDevices[data.ProductId] = data.Enabled; // true/false
        });
        pf["supportedDevices"] = supportedDevices;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/setSupportedDevices',
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

    $('.enableVirtualAudio').on('click', function () {
        const v_virtualAudio = $("#virtualAudio").is(':checked');

        const pf = {};
        pf["enabled"] = v_virtualAudio;
        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/audio/update',
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

    $('.updateDisplay').on('click', function () {
        const index = $(this).data('info');

        const width = parseInt($('.displayWidth_' + index).val());
        const height = parseInt($('.displayHeight_' + index).val());
        const position = parseInt($('.displayPosition_' + index).val());

        if (Number.isNaN(width) || width <= 0) {
            toastr.warning('Invalid display width');
            return;
        }

        if (Number.isNaN(height) || height <= 0) {
            toastr.warning('Invalid display height');
            return;
        }

        const left = position === 1;
        const top = position === 2;

        const pf = {};
        pf["displayIndex"] = index;
        pf["displayWidth"] = width;
        pf["displayHeight"] = height;
        pf["displayLeft"] = left;
        pf["displayTop"] = top;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/display/update',
            type: 'POST',
            dataType: 'json',
            data: json,
            cache: false,
            success: function (response) {
                if (response.status === 1) {
                    toast.success(response.message);
                } else {
                    toastr.error(response.message || 'Unable to update display');
                }
            },
            error: function () {
                toastr.error('Unable to update display');
            }
        });
    });

    $('.setTargetDevice').on('click', function () {
        const outputDevice = $("#outputDevice").val();
        const data = outputDevice.split(";");

        if (data.length < 2) {
            toast.warning('Invalid target device');
            return false;
        }

        const deviceSerial = parseInt(data[2]);
        const deviceDesc = data[1];
        const deviceName = data[0];
        
        const pf = {};
        pf["outputDeviceSerial"] = deviceSerial;
        pf["outputDeviceName"] = deviceName;
        pf["outputDeviceDesc"] = deviceDesc;

        const json = JSON.stringify(pf, null, 2);
        $.ajax({
            url: '/api/audio/outputDevice',
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

    const checkboxCpu = $('#checkbox-cpu');
    const checkboxGpu = $('#checkbox-gpu');
    const checkboxStorage = $('#checkbox-storage');
    const checkboxDevices = $('#checkbox-devices');
    const checkboxDeviceLabels = $('#checkbox-deviceLabels');
    const checkboxCelsius = $('#checkbox-celsius');
    const checkboxBattery = $('#checkbox-battery');
    const checkboxTemperatureBar = $('#checkbox-temperatureBar');
    const checkboxAddDeviceToDashboard = $('#checkbox-addDeviceToDashboard');

    function loadDashboardSettings() {
        // Load current settings
        $.ajax({
            url: '/api/dashboard',
            type: 'GET',
            cache: false,
            success: function(response) {
                if (response.status === 1) {
                    if (response.dashboard.showCpu === true) {
                        checkboxCpu.attr('Checked','Checked');
                    }
                    if (response.dashboard.showGpu === true) {
                        checkboxGpu.attr('Checked','Checked');
                    }
                    if (response.dashboard.showDisk === true) {
                        checkboxStorage.attr('Checked','Checked');
                    }
                    if (response.dashboard.showDevices === true) {
                        checkboxDevices.attr('Checked','Checked');
                    }
                    if (response.dashboard.showLabels === true) {
                        checkboxDeviceLabels.attr('Checked','Checked');
                    }
                    if (response.dashboard.celsius === true) {
                        checkboxCelsius.attr('Checked','Checked');
                    }
                    if (response.dashboard.showBattery === true) {
                        checkboxBattery.attr('Checked','Checked');
                    }
                    if (response.dashboard.temperatureBar === true) {
                        checkboxTemperatureBar.attr('Checked','Checked');
                    }
                    if (response.dashboard.addDeviceToDashboard === true) {
                        checkboxAddDeviceToDashboard.attr('Checked','Checked');
                    }
                }
            }
        });

        $('#btnSaveDashboardSettings').on('click', function () {
            const v_checkboxCpu = checkboxCpu.is(':checked');
            const v_checkboxGpu = checkboxGpu.is(':checked');
            const v_checkboxStorage = checkboxStorage.is(':checked');
            const v_checkboxDevices = checkboxDevices.is(':checked');
            const v_checkboxDeviceLabels = checkboxDeviceLabels.is(':checked');
            const v_checkboxCelsius = checkboxCelsius.is(':checked');
            const v_checkboxBattery = checkboxBattery.is(':checked');
            const v_checkboxTemperatureBar = checkboxTemperatureBar.is(':checked');
            const v_checkboxAddDeviceToDashboard = checkboxAddDeviceToDashboard.is(':checked');
            const v_languageCode = $("#userLanguage").val();
            const v_theme = $("#theme").val();
            const v_keyboardLayout = $("#keyboardLayout").val();

            const pf = {};
            pf["showCpu"] = v_checkboxCpu;
            pf["showGpu"] = v_checkboxGpu;
            pf["showDisk"] = v_checkboxStorage;
            pf["showDevices"] = v_checkboxDevices;
            pf["showLabels"] = v_checkboxDeviceLabels;
            pf["celsius"] = v_checkboxCelsius;
            pf["showBattery"] = v_checkboxBattery;
            pf["temperatureBar"] = v_checkboxTemperatureBar;
            pf["addDeviceToDashboard"] = v_checkboxAddDeviceToDashboard;
            pf["languageCode"] = v_languageCode;
            pf["theme"] = v_theme;
            pf["keyboardLayout"] = parseInt(v_keyboardLayout);

            const json = JSON.stringify(pf, null, 2);

            $.ajax({
                url: '/api/dashboard/update',
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
    }
    loadDashboardSettings();
});
