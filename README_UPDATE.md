# 🔄 راهنمای به‌روزرسانی سریع vWireguard

## دو روش ساده برای به‌روزرسانی:

### روش 1: با همان script نصب
```bash
sudo bash install.sh update
```

### روش 2: با script اختصاصی
```bash
sudo bash update.sh
```

## ✅ چه کارهایی انجام می‌شود:

- ✅ **پشتیبان خودکار** از تمام تنظیمات
- ✅ **حفظ دیتابیس** کلاینت‌ها و کاربران
- ✅ **دانلود نسخه جدید** از GitHub
- ✅ **بازگردانی خودکار** در صورت خطا
- ✅ **تست سلامت** پس از به‌روزرسانی

## 📁 فایل‌های محفوظ:
- دیتابیس: `/opt/vwireguard/db/`
- تنظیمات WireGuard: `/etc/wireguard/`
- تنظیمات SSL و Nginx

## 🆘 اگر مشکلی پیش آمد:
```bash
# لاگ‌ها را ببینید
sudo journalctl -u vwireguard -f

# برگردید به نسخه قبل
cd /opt/vwireguard
sudo systemctl stop vwireguard
sudo cp vwireguard.old vwireguard
sudo systemctl start vwireguard
```

## ⚡ نکات مهم:
- هیچ تنظیماتی از دست نمی‌رود
- کلاینت‌های موجود تحت تأثیر قرار نمی‌گیرند
- از نسخه‌های جدید، دکمه Apply Config بدون ریستارت سرویس و با استفاده از `wg syncconf` تغییرات را اعمال می‌کند
- حین به‌روزرسانی، سرویس موقتاً متوقف می‌شود (حدود 30 ثانیه)

**نگران نباشید! همه چیز ایمن است 🛡️** 
 