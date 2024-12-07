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
            order: [[1, 'asc']],
            select: {
                style: 'os',
                selector: 'td:first-child'
            },
            paging: false,
            searching: false,
            language: {
                emptyTable: "No profile selected. Select profile from left side"
            }
        }
    );

    $('#btnSaveNewProfile').on('click', function(){
        const profile = $("#profileName").val();
        const sensor = $("#sensor").val();

        if (profile.length < 3) {
            toast.warning('Enter your profile name. Minimum length is 3 characters');
            return false;
        }

        const staticMode = $('#staticCheckbox').is(':checked');
        const zeroRpmMode = $('#zeroRpmCheckbox').is(':checked');

        const pf = {};
        pf["profile"] = profile;
        pf["static"] = staticMode;
        pf["zeroRpm"] = zeroRpmMode;
        pf["sensor"] = parseInt(sensor);
        if (parseInt(sensor) === 3) {
            pf["hwmonDeviceId"] = $("#hwmonDeviceId").val();
        }
        if (parseInt(sensor) === 4) {
            const probeData = $("#probeData").val().split(';')
            pf["deviceId"] = probeData[0];
            pf["channelId"] = parseInt(probeData[1]);
        }
        const json = JSON.stringify(pf, null, 2);

        console.log(json)
        $.ajax({
            url: '/api/temperatures',
            type: 'POST',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        window.location.href = '/temperature';
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('.tempList').on('click', function(){
        const profile = $(this).attr('id');
        $('.tempList').removeClass('selected-effect');
        $(this).addClass('selected-effect');
        $.ajax({
            url: '/api/temperatures/' + profile,
            dataType: 'JSON',
            success: function(response) {
                if (response.code === 0) {
                    toast.warning(response.message);
                } else {
                    const data = response.data.profiles;
                    dt.clear();
                    if (profile === 'Quiet' || profile === 'Normal' || profile === 'Performance') {
                        // Those profiles are not editable
                        $.each(data, function(i, item) {
                            dt.row.add([
                                item.id,
                                item.min,
                                item.max,
                                item.fans,
                                item.pump
                            ]).draw();
                        });
                        $("#deleteBtn").hide();
                        $("#updateBtn").hide();
                    } else {
                        $("#profile").val(profile);
                        $.each(data, function(i, item) {
                            dt.row.add([
                                item.id,
                                item.min,
                                item.max,
                                '<input class="form-control" id="pf-fans-' + item.id + '" type="text" value="' + item.fans + '">',
                                '<input class="form-control" id="pf-pump-' + item.id + '" type="text" value="' + item.pump + '">'
                            ]).draw();
                        });
                        $("#deleteBtn").show();
                        $("#updateBtn").show();
                    }
                }
            }
        });
    });

    $('#delete').on('click', function(){
        const profile = $("#profile").val();

        const pf = {};
        pf["profile"] = profile;
        const json = JSON.stringify(pf, null, 2);

        $.ajax({
            url: '/api/temperatures',
            type: 'DELETE',
            data: json,
            cache: false,
            success: function(response) {
                try {
                    if (response.status === 1) {
                        toast.success(response.message);
                        $('#' + profile).remove();
                        $('#deleteTempModal').modal('hide');
                        $("#profile").val('');
                    } else {
                        toast.warning(response.message);
                    }
                } catch (err) {
                    toast.warning(response.message);
                }
            }
        });
    });

    $('#updateBtn').on('click', function(){
        const profile = $("#profile").val();
        const dict = {}; // Create an empty array
        dt.rows().every( function ( rowIdx, tableLoop, rowLoop ) {
            const data = this.data();
            let fans = $("#pf-fans-" + data[0] + "").val();
            let pump = $("#pf-pump-" + data[0] + "").val();
            if (pump < 20) {
                pump = 50;
            }

            pump = parseInt(pump);
            fans = parseInt(fans);
            dict[parseInt(data[0])] = {fans, pump};
        } );

        const json = JSON.stringify(dict, null, 2);

        $.ajax({
            url: '/api/temperatures',
            type: 'PUT',
            data: {
                profile: profile,
                data: json
            },
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

    $('#sensor').on('change', function () {
        const value = $(this).val();
        if (value === "3") {
            $("#storage-data").show();
            $("#temperature-probe-data").hide();
        } else if (value === "4") {
            $("#storage-data").hide();
            $("#temperature-probe-data").show();
        } else {
            $("#storage-data").hide();
            $("#temperature-probe-data").hide();
        }
    });
});