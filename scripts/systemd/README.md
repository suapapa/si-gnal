# Systemd Service Registration

This directory contains example unit files to run the `si-gnal` server and player as background services on Linux (e.g., Raspberry Pi).

## Prerequisites

1. Build the binaries.
2. Move the project to a stable location like `/home/admin/ws/si-gnal/bin`.

## Registration Steps

1. Copy the service files to `/etc/systemd/system/`:
   ```bash
   sudo cp scripts/systemd/*.service /etc/systemd/system/
   ```

2. Edit the service files to match your environment:
   - Change `User=pi` and `WorkingDirectory` if necessary.
   - Update `GEMINI_API_KEY` in `si-gnal-server.service`.
   - Update `-gpio` flag in `si-gnal-player.service` to match your pin.

3. Reload systemd and enable services:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable si-gnal-server
   sudo systemctl enable si-gnal-player
   ```

4. Start the services:
   ```bash
   sudo systemctl start si-gnal-server
   sudo systemctl start si-gnal-player
   ```

## Checking Status

```bash
sudo systemctl status si-gnal-server
sudo systemctl status si-gnal-player
journalctl -u si-gnal-server -f
```

## Note for Player Service

The player's interactive keyboard control will not work when running as a `systemd` service. Ensure you have a GPIO pin configured to trigger playback, or modify the code to ignore keyboard initialization errors.
