"use strict";

document.addEventListener("DOMContentLoaded", function () {
    /* ==========================================================
         NoUI Slider
     ========================================================== */
    var basicNoUISlider = document.getElementById("basicNoUISlider");
    if (basicNoUISlider) {
        noUiSlider.create(basicNoUISlider, {
            // we need to pass only the element, not jQuery object
            start: [20, 80],
            range: {
                min: [0],
                max: [100],
            },
        });
    }

    var stepNoUISlider = document.getElementById("stepNoUISlider");
    if (stepNoUISlider) {
        noUiSlider.create(stepNoUISlider, {
            // we need to pass only the element, not jQuery object
            start: [200, 1000],
            range: {
                min: [0],
                max: [1800],
            },
            step: 100,
            tooltips: true,
            connect: true,
        });
    }

    /* ==========================================================
         Vanillajs Datepicker
    ========================================================== */
    const datepicker = new Datepicker(document.querySelector(".input-datepicker"), {
        buttonClass: "btn",
        format: "mm/dd/yyyy",
    });

    const datepickerAutoClose = new Datepicker(document.querySelector(".input-datepicker-autoclose"), {
        buttonClass: "btn",
        autohide: true,
    });

    const datepickerMultiple = new Datepicker(document.querySelector(".input-datepicker-multiple"), {
        buttonClass: "btn",
        maxNumberOfDates: 3,
    });

    /* ==========================================================
         iMask
    ========================================================== */
    var element = document.getElementById("isbn1");
    if (element) {
        var maskOptions = {
            mask: "000-00-000-0000-0",
        };
        var mask = IMask(element, maskOptions);
    }

    var element = document.getElementById("isbn2");
    if (element) {
        var maskOptions = {
            mask: "000 00 000 0000 0",
        };
        var mask = IMask(element, maskOptions);
    }

    var element = document.getElementById("isbn3");
    if (element) {
        var maskOptions = {
            mask: "000/00/000/0000/0",
        };
        var mask = IMask(element, maskOptions);
    }
    var element = document.getElementById("ip4");
    if (element) {
        var maskOptions = {
            mask: "000.000.000.000'",
        };
        var mask = IMask(element, maskOptions);
    }

    var element = document.getElementById("currency");
    if (element) {
        var maskOptions = {
            mask: "$ num",
            blocks: {
                num: {
                    // nested masks are available!
                    mask: Number,
                    thousandsSeparator: ",",
                    radix: ".",
                },
            },
        };
        var mask = IMask(element, maskOptions);
    }
    var element = document.getElementById("date");
    if (element) {
        var maskOptions = {
            mask: Date, // enable date mask

            // other options are optional
            pattern: "Y-m-d", // Pattern mask with defined blocks, default is 'd{.}`m{.}`Y'
            // you can provide your own blocks definitions, default blocks for date mask are:
            blocks: {
                d: {
                    mask: IMask.MaskedRange,
                    from: 1,
                    to: 31,
                    maxLength: 2,
                },
                m: {
                    mask: IMask.MaskedRange,
                    from: 1,
                    to: 12,
                    maxLength: 2,
                },
                Y: {
                    mask: IMask.MaskedRange,
                    from: 1900,
                    to: 9999,
                },
            },
            // define date -> str convertion
            format: function (date) {
                var day = date.getDate();
                var month = date.getMonth() + 1;
                var year = date.getFullYear();

                if (day < 10) day = "0" + day;
                if (month < 10) month = "0" + month;

                return [year, month, day].join("-");
            },
            // define str -> date convertion
            parse: function (str) {
                var yearMonthDay = str.split("-");
                return new Date(yearMonthDay[0], yearMonthDay[1] - 1, yearMonthDay[2]);
            },
        };
        var mask = IMask(element, maskOptions);
    }

    var element = document.getElementById("phone");
    if (element) {
        var maskOptions = {
            mask: "+{1}-000-000-0000",
        };
        var mask = IMask(element, maskOptions);
    }

    var element = document.getElementById("taxId");
    if (element) {
        var maskOptions = {
            mask: "00-000000",
        };
        var mask = IMask(element, maskOptions);
    }

    /* ==========================================================
         MultiSelect
    ========================================================== */
    var select_element = document.getElementById("multiselect1");
    multi(select_element);

    /* =====================================================
        Choices.JS
    ===================================================== */
    /* Add custom Bootstrap classes to choices*/
    function injectClassess(x) {
        let pickerCustomClass = x.dataset.customclass;
        let pickerSevClasses = pickerCustomClass.split(" ");
        x.parentElement.classList.add.apply(x.parentElement.classList, pickerSevClasses);
    }

    // Variant 1 - Primary
    const choicesPrimary = document.querySelector(".choices-primary");
    const defaultChoices = new Choices(choicesPrimary, {
        searchEnabled: false,
        callbackOnInit: () => injectClassess(choicesPrimary),
    });

    // Variant 1 - Secondary
    const choicesSecondary = document.querySelector(".choices-secondary");
    const secondaryChoices = new Choices(choicesSecondary, {
        searchEnabled: false,
        callbackOnInit: () => injectClassess(choicesSecondary),
    });

    // Variant 1 - Outline dark
    const choicesOutlined = document.querySelector(".choices-outlined");
    const outlinedChoices = new Choices(choicesOutlined, {
        searchEnabled: false,
        placeholder: true,
        removeItemButton: true,
        placeholderValue: "Choose your country",
        callbackOnInit: () => injectClassess(choicesOutlined),
    });

    /* =====================================================
        CHOICES.JS Tags Input
    ===================================================== */

    // Variant 1 - With remove button
    const choicesTags = document.querySelector(".tags-input");
    const tagsChoices = new Choices(choicesTags, {
        searchEnabled: false,
        removeItemButton: true,
        callbackOnInit: () => injectClassess(choicesTags),
    });

    // Variant 2 - Email address only
    const choicesTagsEmail = document.querySelector(".tags-email");
    const tagsEmailChoices = new Choices(choicesTagsEmail, {
        callbackOnInit: () => injectClassess(choicesTagsEmail),
        addItemFilter: function (value) {
            if (!value) {
                return false;
            }
            const regex =
                /^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
            const expression = new RegExp(regex.source, "i");
            return expression.test(value);
        },
    });

    // Variant 3 - Disabled
    const choicesTagsDisabled = document.querySelector(".tags-disabled");
    const tagsDisabledChoices = new Choices(choicesTagsDisabled, {
        searchEnabled: false,
        removeItemButton: true,
        callbackOnInit: () => injectClassess(choicesTagsDisabled),
    }).disable();
});
