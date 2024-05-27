// ------------------------------------------------------ //
// For demo purposes, can be deleted
// ------------------------------------------------------ //

// Asigning Alternative stylesheet & insert it in its place
var stylesheet = document.getElementById("theme-stylesheet");
var alternateStylesheet = document.createElement("link");
alternateStylesheet.setAttribute("id", "new-stylesheet");
alternateStylesheet.setAttribute("rel", "stylesheet");
stylesheet.parentNode.insertBefore(alternateStylesheet, stylesheet.nextSibling);

// Style Switcher
var styleSwitcher = document.getElementById("colour");
styleSwitcher.addEventListener("change", function () {
    var alternateColor = styleSwitcher.value;
    alternateStylesheet.setAttribute("href", alternateColor);
    Cookies.set("switcherColor", alternateColor, { expires: 365, path: "/" });
});

var theCookie = Cookies.get("switcherColor");
if (theCookie) {
    alternateStylesheet.setAttribute("href", theCookie);
}
