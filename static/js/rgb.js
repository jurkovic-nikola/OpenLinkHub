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

    $('.rgbList').on('click', function(){
        const profile = $(this).attr('id');
        $('.rgbList').removeClass('selected-effect');
        $(this).addClass('selected-effect');
        $.ajax({
            url: '/api/color/' + profile,
            dataType: 'JSON',
            success: function(response) {
                if (response.code === 0) {
                    toast.warning(response.message);
                } else {
                    const data = response.data;
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
                    } else {
                        $("#profile").val(profile);
                        dt.row.add([
                            data.speed,
                            data.brightness,
                            data.smoothness,
                            '<div style="width:100%; height:20px;background-color: rgb(' + data.start.red + ', ' + data.start.green + ', ' + data.start.blue + ')"></div',
                            '<div style="width:100%; height:20px;background-color: rgb(' + data.end.red + ', ' + data.end.green + ', ' + data.end.blue + ')"></div'
                        ]).draw();
                    }
                }
            }
        });
    });
});