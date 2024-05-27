"use strict";

document.addEventListener("DOMContentLoaded", function () {
    // Insert Boostrap Validation Classes
    function bsValidationBehavior(errorInputs, form) {
        function watchError() {
            errorInputs.forEach((input) => {
                if (input.classList.contains("js-validate-error-field")) {
                    input.classList.add("is-invalid");
                    input.classList.remove("is-valid");
                } else {
                    input.classList.remove("is-invalid");
                    input.classList.add("is-valid");
                }
            });
        }
        watchError();
    }

    // Login form validation
    new window.JustValidate("#simpleLoginForm", {
        rules: {
            email: {
                required: true,
                email: true,
            },
            password: {
                required: true,
            },
        },
        messages: {
            email: "Please enter a valid email",
            password: "Please enter your password",
        },
        invalidFormCallback: function () {
            let errorInputs = document.querySelectorAll("#simpleLoginForm input[required]");
            let form = document.querySelector("#simpleLoginForm");
            bsValidationBehavior(errorInputs, form);
            form.addEventListener("keyup", () => bsValidationBehavior(errorInputs, form));
        },
    });

    // Editor form validation
    new window.JustValidate("#editorForm", {
        rules: {
            editorName: {
                required: true,
            },
            editorContent: {
                required: true,
            },
        },
        messages: {
            editorName: "Name is required",
            editorContent: "Please write something :)",
        },
        invalidFormCallback: function () {
            let errorInputs = document.querySelectorAll("#editorForm [required]");
            let form = document.querySelector("#editorForm");
            bsValidationBehavior(errorInputs, form);
            form.addEventListener("keyup", () => bsValidationBehavior(errorInputs, form));
        },
    });

    // Initiate editor
    var toolbarOptions = [
        ["bold", "italic", "underline", "strike"], // toggled buttons
        [{ header: 1 }, { header: 2 }], // custom button values
        [{ list: "ordered" }, { list: "bullet" }],
        [{ indent: "-1" }, { indent: "+1" }], // outdent/indent
        [{ direction: "rtl" }], // text direction
        [{ size: ["small", false, "large", "huge"] }], // custom dropdown
        [{ header: [1, 2, 3, 4, 5, 6, false] }],
        [{ color: [] }, { background: [] }], // dropdown with defaults from theme
        [{ font: [] }],
        [{ align: [] }],

        ["clean"], // remove formatting button
    ];
    const quill = new Quill("#editor", {
        modules: {
            toolbar: toolbarOptions,
        },
        placeholder: "Compose an epic...",
        theme: "snow",
    });

    // Sighup form validation
    new window.JustValidate("#form2", {
        rules: {
            email: {
                required: true,
                email: true,
            },
            name: {
                required: true,
            },
            password: {
                required: true,
                minLength: 5,
            },
            passwordConfirmation: {
                required: true,
                function: (name, value) => {
                    let form2PassFieldVal = document.getElementById("password").value;
                    return value == form2PassFieldVal;
                },
            },
        },

        messages: {
            name: "Please enter a your name",
            email: "Please enter a valid email",
            password: "Please enter your password",
            passwordConfirmation: "password doesnt match",
        },
        invalidFormCallback: function () {
            let errorInputs = document.querySelectorAll("#form2 input[required]");
            let form = document.querySelector("#form2");
            bsValidationBehavior(errorInputs, form);
            form.addEventListener("keyup", () => bsValidationBehavior(errorInputs, form));
        },
    });
});
