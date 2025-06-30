# راهنمای به‌روزرسانی vWireguard Panel

## 🔄 روش‌های به‌روزرسانی

### روش اول: استفاده از script نصب
```bash
sudo bash install.sh update
```
یا
```bash
sudo bash install.sh --update
```
یا
```bash
sudo bash install.sh -u
```

### روش دوم: استفاده از script اختصاصی به‌روزرسانی
```bash
# ابتدا فایل را دانلود کنید
wget https://raw.githubusercontent.com/MmadF14/vwireguard/main/update.sh

# سپس اجرا کنید
sudo bash update.sh
```

## 📋 مراحل به‌روزرسانی

1. **ایجاد پشتیبان خودکار**: تمام تنظیمات و دیتابیس شما پشتیبان‌گیری می‌شود
2. **توقف سرویس**: سرویس vWireguard به طور موقت متوقف می‌شود
3. **دانلود نسخه جدید**: آخرین نسخه از GitHub دانلود می‌شود
4. **جایگزینی فایل‌ها**: فقط فایل‌های اجرایی و template ها جایگزین می‌شوند
5. **راه‌اندازی مجدد**: سرویس با تنظیمات جدید راه‌اندازی می‌شود

## 🛡️ ایمنی به‌روزرسانی

- **پشتیبان خودکار**: قبل از هر به‌روزرسانی، پشتیبان کامل ایجاد می‌شود
- **حفظ تنظیمات**: تمام کلاینت‌ها، کاربران و تنظیمات حفظ می‌شوند
- **بازگردانی خودکار**: در صورت خطا، به نسخه قبل بازمی‌گردد
- **تست سلامت**: پس از به‌روزرسانی، سلامت سرویس تست می‌شود

## 📁 فایل‌هایی که حفظ می‌شوند

- `/opt/vwireguard/db/` - دیتابیس کلاینت‌ها و تنظیمات
- `/etc/wireguard/wg0.conf` - تنظیمات WireGuard
- `/etc/nginx/sites-available/vwireguard` - تنظیمات Nginx
- فایل‌های SSL در صورت وجود

## 📁 فایل‌هایی که به‌روزرسانی می‌شوند

- `/opt/vwireguard/vwireguard` - فایل اجرایی اصلی
- `/opt/vwireguard/templates/` - قالب‌های وب
- `/opt/vwireguard/static/` - فایل‌های استاتیک

## 🔧 دستورات مفید

```bash
# بررسی وضعیت سرویس
sudo systemctl status vwireguard

# مشاهده لاگ‌ها
sudo journalctl -u vwireguard -f

# ری‌استارت سرویس
sudo systemctl restart vwireguard

# بررسی نسخه فعلی
cd /opt/vwireguard && ./vwireguard --version
```

## 🚨 در صورت بروز مشکل

اگر پس از به‌روزرسانی مشکلی پیش آمد:

1. **لاگ‌ها را بررسی کنید**:
   ```bash
   sudo journalctl -u vwireguard --no-pager -n 50
   ```

2. **به نسخه قبل بازگردید**:
   ```bash
   cd /opt/vwireguard
   sudo systemctl stop vwireguard
   sudo cp vwireguard.old vwireguard
   sudo systemctl start vwireguard
   ```

3. **یا از پشتیبان کامل استفاده کنید**:
   ```bash
   # پیدا کردن آخرین پشتیبان
   ls -la /opt/vwireguard-backup-*
   
   # بازگردانی از پشتیبان (مثال)
   sudo systemctl stop vwireguard
   sudo cp /opt/vwireguard-backup-YYYYMMDD_HHMMSS/vwireguard.old /opt/vwireguard/vwireguard
   sudo systemctl start vwireguard
   ```

## ✅ پس از به‌روزرسانی

1. بررسی کنید که پنل در مرورگر باز می‌شود
2. تست کنید که کلاینت‌های موجود کار می‌کنند
3. بررسی کنید که تانل‌های جدید (اگر اضافه شده) در دسترس هستند

## 📞 پشتیبانی

در صورت بروز مشکل، می‌توانید:
- Issue جدید در GitHub ایجاد کنید
- لاگ‌های سیستم را ضمیمه کنید
- مشخصات سیستم عامل و نسخه را اعلام کنید 