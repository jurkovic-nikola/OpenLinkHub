function CreateToastr() {
    toastr.options = {
        closeButton: true,
        debug: false,
        newestOnTop: false,
        progressBar: true,
        positionClass: "toast-top-right",
        preventDuplicates: true,
        showDuration: 300,
        hideDuration: 1000,
        timeOut: 7000,
        extendedTimeout: 1000,
        showEasing: "swing",
        hideEasing: "linear",
        showMethod: "fadeIn",
        hideMethod: "fadeOut"
    };
    return toastr;
}

// Global
window.toast = CreateToastr();