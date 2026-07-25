const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

test("temperature probe options include the configured sensor label", () => {
    const source = fs.readFileSync(path.join(__dirname, "overview.js"), "utf8");
    assert.match(
        source,
        /value:\s*label\.ChannelId,\s*text:\s*`\$\{label\.Name\} - \$\{label\.Label\}`/
    );
});
