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

    $('.saveRgbControl').on('click', function () {
        const rgbControl = $("#rgbControl").is(':checked');
        const rgbOff = $("#rgbOff").val();
        const rgbOn = $("#rgbOn").val();

        const pf = {};
        pf["rgbControl"] = rgbControl;
        pf["rgbOff"] = rgbOff;
        pf["rgbOn"] = rgbOn;

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
});