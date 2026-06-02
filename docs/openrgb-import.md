OpenRGB Import – Design & Behavior
Overview

OpenRGB import in OpenLinkHub (OLH) is designed to prioritize stability, usability, and persistence over perfect automatic detection.

Instead of attempting to fully parse OpenRGB’s internal payload structures (which vary significantly by device), OLH uses a config-driven model:

Detect devices
Create a usable default configuration
Allow user edits
Persist the result as the source of truth
Core Principles
1. Saved Config is the Source of Truth

The file:

database/openrgbimport-zones.json

represents the last known good configuration.

If a valid config exists → it is always used
Auto-detection does not override saved configs
User edits persist across restarts
2. Auto-Detection is Only for First-Time Setup

Auto-detection is used only when:

A device is first discovered
No saved config exists

It provides a starting point, not a final answer.

3. Zone Names Matter More Than LED Counts
Correct zone structure and naming is the primary goal
LED counts are best-effort defaults
Users are expected to verify counts via OpenRGB if needed
4. LED Counts Are User-Editable

OLH intentionally allows users to:

Modify LED counts per zone
Use OpenRGB as the reference for correct values

OLH does not enforce strict limits or attempt to fully infer counts.

5. Known Devices Use Trusted Defaults

For certain devices, OLH uses predefined configurations instead of relying on parsing.

GPU (Generic)
1 zone
Default LED count: 1
Fully user-editable
ASUS Motherboard
Aura Mainboard
RGB Header 1
RGB Header 2
RGB Header 3
Default LED count: 1 per zone
Logical mapping (not parser-driven)
Lian Li Strimer
Each strip is treated as a zone:
24 Pin ATX Strip 0–5
Default LED count: 20 per strip
Fully user-editable
Unknown Devices
Fallback:
Zone 1
LED count: 1
Safety Model
Problem

Setting LED counts too high can cause the OpenRGB server to crash.

Solution

OLH uses a soft warning + runtime validation model:

Step 1: Detect Risky Change

If a user increases LED count beyond the current saved value:

Show warning:

Increasing LED count beyond the current saved value may crash the OpenRGB server.

Step 2: Apply First

The new configuration is applied to OpenRGB before saving.

Step 3: Health Check

OLH performs a short retry-based health check:

4 attempts
500ms delay between attempts
Verifies OpenRGB is still reachable
Step 4: Save or Reject
✅ If OpenRGB is still healthy → save config
❌ If OpenRGB is unavailable → do not save, revert to previous config
Result

The JSON config always represents a working configuration, not just the last attempted value.

Why Not Fully Parse OpenRGB?

OpenRGB payloads:

vary significantly by device/vendor
contain inconsistent structures
are not reliably parseable in a generic way

Attempting full parsing leads to:

fragile logic
incorrect assumptions
inconsistent results across devices

Instead, OLH uses:

trusted defaults for known devices
user correction for fine-tuning
runtime validation for safety
What NOT to Do

❌ Do not:

attempt to fully parse all OpenRGB payload structures generically
rely on parser-derived LED counts as authoritative
overwrite saved config after risky changes without validation
enforce strict LED caps based on assumptions
Debugging / Usage Notes
Starting the OpenRGB Server

OpenLinkHub is designed to connect to your existing standalone OpenRGB server. You do not need to run complicated background terminal commands to start the server! 
You can simply use the official OpenRGB GUI:
1. Open the official OpenRGB application.
2. Navigate to the **SDK Server** tab.
3. Click **Start Server** (ensure it binds to the default port `6742`).
4. Launch OpenLinkHub. It will automatically detect and connect to your running OpenRGB GUI instance.

Finding Correct LED Counts

Use OpenRGB:

Open OpenRGB UI
Select the device
Inspect zones and LED counts
Enter those values into OLH
Resetting Config

If needed, delete or edit:

database/openrgbimport-zones.json

Then restart OLH to regenerate defaults.

Summary

OLH’s OpenRGB import is designed to be:

Stable (saved config persists)
Flexible (user-editable)
Safe (runtime validation prevents bad saves)
Practical (no reliance on fragile parsing)

The system favors real-world behavior and user control over perfect automatic detection.
