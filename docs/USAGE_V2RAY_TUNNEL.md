# WireGuard ➜ V2Ray Tunnel

این ویژگی امکان تونل‌کردن ترافیک کلاینت‌های WireGuard را از طریق یک سرور V2Ray فراهم می‌کند. در صفحه Tunnels تب **WireGuard ➜ V2Ray** را انتخاب کرده و تنظیمات مربوط به پروتکل (VMess/VLESS/Trojan) را وارد کنید. پس از ذخیره، یک فایل پیکربندی در `/etc/vwireguard/tunnels/` ایجاد و سرویس systemd متناظر فعال می‌شود.

![screenshot](../assets/v2ray_tunnel.png)

## Paste a V2Ray link
در ابتدای فرم، لینک `vmess://`، `vless://` یا `trojan://` خود را در کادر **Link** وارد کنید و دکمهٔ **Parse** را بزنید تا مقادیر فرم به صورت خودکار پر شود.

![parse](../assets/v2ray_link_parse.png)
