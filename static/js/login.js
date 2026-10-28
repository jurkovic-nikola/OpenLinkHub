"use strict";

const loginForm = document.getElementById("loginForm");
const loginSubmit = document.getElementById("loginSubmit");
const loginError = document.getElementById("loginError");

loginForm.addEventListener("submit", async (event) => {
    event.preventDefault();

    if (loginSubmit.disabled) {
        return;
    }

    loginError.hidden = true;
    loginSubmit.disabled = true;
    loginSubmit.textContent = "Signing in…";

    try {
        const response = await fetch("/auth/login", {
            method: "POST",
            credentials: "same-origin",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({
                username: document.getElementById("username").value,
                password: document.getElementById("password").value
            })
        });

        if (response.status === 204) {
            document.getElementById("password").value = "";
            window.location.replace("/");
            return;
        }

        switch (response.status) {
            case 401:
                loginError.textContent = "Invalid username or password.";
                break;
            case 429:
                loginError.textContent = "Too many login attempts. Please try again shortly.";
                break;
            case 403:
                loginError.textContent = "This address does not match the configured authentication origin.";
                break;
            default:
                loginError.textContent = "Unable to sign in. Please try again.";
        }

        loginError.hidden = false;
    } catch {
        loginError.textContent = "Unable to connect to OpenLinkHub.";
        loginError.hidden = false;
    } finally {
        loginSubmit.disabled = false;
        loginSubmit.textContent = "Sign in";
    }
});