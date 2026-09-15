# reMarkable Paper Pro — Network Setup

How to give your reMarkable Paper Pro a stable, reachable IP address on your home network so a backend service can SSH into it reliably, without needing USB plugged in.

> **⚠️ Do this before anything else.** Enabling Developer Mode (required for any SSH access on the Paper Pro, including over USB) **factory-resets the device**, wiping local storage. Back up your documents first — see Step 0.

## Step 0: Back up your documents (do this first)

Do this **before** enabling Developer Mode. None of these require SSH or Developer Mode.

1. **Confirm cloud sync is caught up** (your real restore mechanism): **Settings → About / Storage** → confirm no pending sync items. After the reset, signing back into your account pulls everything back down automatically.
2. **Bulk-export a local copy with the reMarkable desktop app**: open the app, sign in, select all documents (`Ctrl+A`) → right-click → **Export** → PDF → choose a destination folder. Fastest way to get everything onto your computer.
3. **(Optional, redundant check)** — with the tablet connected via USB and **Settings → Storage → USB web interface** toggled on, open `http://10.11.99.1` in a browser (not a terminal) for a per-document download page. No SSH or Developer Mode needed.

Only proceed past this point once you've confirmed at least one of the above.

## 1. Enable Developer Mode and the SSH/web interface

> On the Paper Pro, **Developer Mode is required for SSH access at all** — even over USB. The "USB web interface" toggle alone (a holdover from older reMarkable models) is not sufficient; without Developer Mode, `sshd` will accept the TCP connection but reset it before completing the handshake (`kex_exchange_identification: read: Connection reset by peer`).

1. **Enable Developer Mode**: **Settings → Software → tap the version number → Advanced → Developer settings**, then follow the prompt. This resets the device and you'll redo initial onboarding — confirm Step 0 is done first.
2. After the reset, sign back in and confirm your documents synced back down.
3. On the tablet: **Settings → Storage → USB web interface** → toggle on.
4. Get the root SSH password: **Settings → About / Help → General information** (device-specific, changes on factory reset/Developer Mode re-enable).
5. Connect the tablet to your home WiFi (**Settings → WiFi**).

> **Note:** the toggle above only enables SSH over **USB**. SSH over WiFi is disabled by default and needs the extra steps in section 1a below — skipping this is the most common cause of "ping works, SSH connection refused."

## 1a. Enable SSH over WiFi (required — off by default)

1. **Connect via USB** and SSH in over the USB link:
   ```bash
   ssh root@10.11.99.1
   ```
2. **From inside that USB SSH session**, turn on WiFi SSH:
   ```bash
   rm-ssh-over-wlan on
   ```
3. Disconnect USB. SSH over WiFi is now enabled and will stay enabled across reboots.

## 2. Find the tablet's MAC address

On the tablet: **Settings → About / Help → General information** — look for the WiFi MAC address (format `XX:XX:XX:XX:XX:XX`).

Alternatively, find it from your router's DHCP client list (see step 3) by matching the device name/current IP.

## 3. Find your router's gateway IP and subnet

On your computer:

```bash
ipconfig
```

Note:
- **Default Gateway** — e.g. `192.168.1.1`
- **Subnet Mask** — usually `255.255.255.0`

Any reservation IP must share the first three octets with the gateway (e.g. `192.168.1.x`) when the mask is `255.255.255.0`.

## 4. Reserve an IP for the tablet

1. Open your router's admin page in a browser (the gateway IP from step 3) and log in.
2. Find the DHCP settings — naming varies by brand:
   - **Netgear / TP-Link / Asus**: *LAN Setup* or *DHCP Server* → *Address Reservation*
   - **Google Wifi / Nest**: app → *Wifi* tab → tap device → gear icon → *Reserve IP*
   - **Ubiquiti / UniFi**: *Clients* → select device → *Config* → *Use Fixed IP*
   - **eero**: app → device → *Reserve this IP address*
3. Add a reservation:
   - **MAC address**: from step 2
   - **IP address**: an address in your LAN subnet (step 3), outside the router's dynamic DHCP pool if it shows one (e.g. pick `.50` if the dynamic pool is `.100`–`.200`)
   - **Hostname/label**: `remarkable-paper-pro` (for your own reference)
4. Save.

## 5. Apply the reservation

Reservations only take effect on the tablet's *next* DHCP request:

- Toggle WiFi off/on on the tablet (**Settings → WiFi**), or
- Reboot the tablet.

## 6. Verify

On the tablet, confirm the assigned IP: **Settings → WiFi → tap the connected network** — it should show the reserved address.

From your computer:

```bash
ping <reserved-ip>
ssh root@<reserved-ip>
```

### If ping fails with 100% loss

- **Client/AP isolation**: many routers (guest networks, mesh systems, IoT SSIDs) block devices on the same WiFi from reaching each other even on the same subnet. Check for a setting called **AP Isolation**, **Client Isolation**, or a separate "IoT"/guest network the tablet might have joined, and disable it or move the tablet to your main network.
- **WiFi sleep**: the tablet may disable its WiFi radio when the screen sleeps to save battery. Wake it and re-check **Settings → WiFi** shows "Connected" before pinging.
- **Stale IP**: confirm the tablet's current IP (step 6) actually matches the reserved one — if the reservation hasn't applied yet, repeat step 5.

## 7. Browse the device's document store

```bash
ssh root@<reserved-ip>
cd /home/root/.local/share/remarkable/xochitl/
ls -la
```

Each document is a UUID-named `.metadata` / `.content` pair (plus `.pagedata`, `.thumbnails/`, etc.):

```bash
cat <uuid>.metadata   # title, type, parent folder, timestamps
cat <uuid>.content    # page count, current page
```

Optional — copy the whole store locally to inspect without editing live files:

```bash
scp -r root@<reserved-ip>:/home/root/.local/share/remarkable/xochitl/ ./xochitl-backup
```

## Notes for the backend

- This reserved IP is what goes into the Go backend's config (`config.yaml` / env var) as the SSH adapter's target host.
- The SSH root password is device-specific — store it as a secret, not hardcoded. Consider `ssh-copy-id root@<reserved-ip>` to switch to key-based auth.
- Design the `Client` adapter interface to fall back to the reMarkable cloud API when this IP is unreachable (tablet off WiFi, asleep, or away from home).