# TKSTAR Gateway

After the App starts, use **Open Web UI** or the **TKSTAR Gateway** sidebar entry.

For the first tracker:

1. Open **Devices**.
2. Press **Pair new tracker (10 min)**.
3. Configure the tracker to send to your public hostname/IP and the appropriate TCP port shown in the App Network settings.
4. Forward only that tracker TCP port through your router to the Home Assistant host.
5. Wait for the tracker to appear, then close pairing.

Default listeners are Watch 5093/tcp, H02 5013/tcp and GT06 5023/tcp. These are Gateway listener choices, not universal model guarantees.

Do not expose port 8099. The web interface is intended to run through Home Assistant Ingress.
