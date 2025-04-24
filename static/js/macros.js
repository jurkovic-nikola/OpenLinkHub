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

    // Init dataTable
    const dt = $('#table').DataTable(
        {
            order: [[0, 'asc']],
            select: {
                style: 'os',
                selector: 'td:first-child'
            },
            paging: false,
            searching: false,
            language: {
                emptyTable: "No profile selected or profile has no macros defined. Select profile from left side or define macros"
            }
        }
    );

    $('#btnSaveNewMacroProfile').on('click', function(){
        const profile = $("#profileName").val();
        if (profile.length < 3) {
            toast.warning('Enter your profile name. Minimum length is 3 characters');
            return false;
        }

        const pf = {};
        pf["macroName"] = profile;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/macro/',
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
                            default:
                                actionType = 'n/a';
                                break;
                        }

                        if (item.actionType === 5) { // 5 is always Delay option
                            dt.row.add([
                                i,
                                actionType,
                                item.actionDelay,
                                '<input class="btn btn-danger deleteMacroValue" id="deleteMacroValue" data-id="' + pf + ';' + i + '" type="button" value="DELETE" style="width: 100%;">'
                            ]).draw();
                        } else {
                            // Render row if we have actual key
                            getKeyName(item.actionCommand, function(result) {
                                dt.row.add([
                                    i,
                                    actionType,
                                    result,
                                    '<input class="btn btn-danger deleteMacroValue" id="deleteMacroValue" data-id="' + pf + ';' + i + '" type="button" value="DELETE" style="width: 100%;">'
                                ]).draw();
                            });
                        }
                    });

                    dt.on('click', '.deleteMacroValue', function () {
                        const $btn = $(this); // Save reference to the clicked button
                        const macroInfo = $btn.data('id');
                        const macro = macroInfo.split(";");

                        if (macro.length < 2 || macro.length > 2) {
                            toast.warning('Invalid macro profile selected');
                            return false;
                        }

                        const pf = {
                            macroId: parseInt(macro[0]),
                            macroIndex: parseInt(macro[1])
                        };
                        const json = JSON.stringify(pf, null, 2);

                        $.ajax({
                            url: '/api/macro/value',
                            type: 'DELETE',
                            data: json,
                            cache: false,
                            success: function(response) {
                                try {
                                    if (response.status === 1) {
                                        // Remove the row from DataTable
                                        dt.row($btn.closest('tr')).remove().draw();
                                        toast.success("Macro value deleted successfully.");
                                    } else {
                                        toast.warning(response.message);
                                    }
                                } catch (err) {
                                    toast.warning("Error occurred while processing response.");
                                }
                            }
                        });
                    });
                    $("#deleteBtn").show();
                    $("#newMacroValue").show();
                }
            }
        });
    }

    $('.macroList').on('click', function(){
        const profile = $(this).attr('id');
        $('.macroList').removeClass('selected-effect');
        $(this).addClass('selected-effect');
        let pf = parseInt(profile);
        loadMacroValues(pf);
    });

    $('#deleteMacroProfile').on('click', function(){
        const macroId = $("#profile").val();
        const pf = {};
        pf["macroId"] = parseInt(macroId);
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/macro/',
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

    $('#addMacroValue').on('click', function(){
        const macroId = $("#profile").val();
        const macroType = $("#macroType").val();
        const macroValue = $("#macroKeyId").val();
        const macroDelay = $("#macroDelay").val();

        if (parseInt(macroType) === 0) {
            toast.warning('Select macro type');
            return false;
        }

        if (parseInt(macroType) === 3 && parseInt(macroValue) === 0) {
            toast.warning('Select macro value');
            return false;
        }

        if (parseInt(macroType) === 5 && parseInt(macroDelay) < 1) {
            toast.warning('Macro delay requires definition of delay in milliseconds');
            return false;
        }

        const pf = {};
        pf["macroId"] = parseInt(macroId);
        pf["macroType"] = parseInt(macroType);
        pf["macroValue"] = parseInt(macroValue);
        pf["macroDelay"] = parseInt(macroDelay);
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/macro/',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
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

    $('#macroType').on('change', function () {
        const selectedValue = parseInt($(this).val());

        switch (selectedValue) {
            case 0: {
                $("#macroDelay").hide();
                $("#macroKeyId").hide();
            }
            break;
            case 3: {
                $("#macroDelay").hide();
                $.ajax({
                    url:'/api/input/keyboard',
                    type:'get',
                    success:function(result){
                        let macroKeyId = $("#macroKeyId");
                        macroKeyId.empty();
                        $.each(result.data, function( index, value ) {
                            macroKeyId.append($('<option>', { value: index, text: value.Name }));
                        });
                    }
                });
                $("#macroKeyId").show();
            }
            break;
            case 5: {
                $("#macroDelay").show();
                $("#macroKeyId").hide();
            }
            break;
        }
    });

    function getKeyName(keyIndex, callback) {
        $.ajax({
            url: '/api/macro/keyInfo/' + parseInt(keyIndex),
            type: 'GET',
            cache: false,
            success: function(response) {
                if (response.status === 1) {
                    callback(response.data);
                } else {
                    callback('');
                }
            },
            error: function() {
                callback('');
            }
        });
    }
});