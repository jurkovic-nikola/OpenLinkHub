"use strict";

function hidePreloader() {
    let preloader = document.querySelector(".spinner-wrapper");

    setTimeout(function () {
        preloader.style.opacity = "0";
    }, 1000);
    setTimeout(function () {
        preloader.remove();
    }, 1500);
}

window.addEventListener("load", function () {
    hidePreloader();
});
