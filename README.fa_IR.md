[English](README.md) | فارسی

<div dir="rtl">

# vWireguard - سیستم مدیریت VPN وایرفارد

[![Go Report Card](https://goreportcard.com/badge/github.com/MmadF14/vwireguard)](https://goreportcard.com/report/github.com/MmadF14/vwireguard)
[![GoDoc](https://godoc.org/github.com/MmadF14/vwireguard?status.svg)](https://godoc.org/github.com/MmadF14/vwireguard)
[![License](https://img.shields.io/github/license/MmadF14/vwireguard)](LICENSE)

## 📸 تصاویر

<div align="center">
  <h3>داشبورد</h3>
  <img src="assets/images/dashboard.png" alt="داشبورد" width="800"/>
  <p><em>داشبورد اصلی با نمایش کلی سیستم و آمار</em></p>
</div>

<div align="center">
  <h3>مدیریت کلاینت‌ها</h3>
  <img src="assets/images/client-management.png" alt="مدیریت کلاینت" width="800"/>
  <p><em>رابط مدیریت کلاینت با وضعیت اتصال و گزینه‌های پیکربندی</em></p>
</div>

<div align="center">
  <h3>نظارت بر سیستم</h3>
  <img src="assets/images/system-monitor.png" alt="نظارت بر سیستم" width="800"/>
  <p><em>نظارت بر سیستم در زمان واقعی با نمودارهای مصرف منابع</em></p>
</div>

## 🌟 ویژگی‌ها

- 🔒 مدیریت امن VPN وایرفارد
- 👥 پشتیبانی از چند کاربر با کنترل دسترسی مبتنی بر نقش
- 🌐 قابلیت Wake-on-LAN برای دستگاه‌های راه‌دور
- 📊 نظارت بر سیستم در زمان واقعی
- 🔄 تولید خودکار پیکربندی کلاینت
- 📱 رابط کاربری واکنش‌گرا
- 🌍 پشتیبانی دو زبانه (فارسی/انگلیسی)
- 📝 سیستم ثبت رویداد جامع
- 🔧 ابزارهای سیستم و نگهداری
- 🔐 مدیریت امن کلیدها

## 🚀 شروع سریع

1. کلون کردن مخزن:
```bash
git clone https://github.com/MmadF14/vwireguard.git
cd vwireguard
```

2. نصب وابستگی‌ها:
```bash
go mod download
```

3. ساخت برنامه:
```bash
go build
```

4. اجرای برنامه:
```bash
./vwireguard
```

## 📋 پیش‌نیازها

- Go 1.21 یا بالاتر
- نصب شده وایرفارد روی سرور
- سیستم مبتنی بر لینوکس (توصیه شده اوبونتو)
- دسترسی root برای عملیات سیستم

## 🛠️ پیکربندی

1. پیکربندی وایرفارد:
```bash
wg-quick up wg0
```

2. دسترسی به رابط وب:
```
http://localhost:5000
```

3. اطلاعات پیش‌فرض:
- نام کاربری: admin
- رمز عبور: admin

## 🔒 امنیت

- تمام رمزهای عبور با bcrypt هش می‌شوند
- پشتیبانی از HTTPS برای ارتباط امن
- کنترل دسترسی مبتنی بر نقش
- ذخیره‌سازی و مدیریت امن کلیدها
- به‌روزرسانی‌های منظم امنیتی

## 🤝 مشارکت

مشارکت‌ها مورد استقبال قرار می‌گیرند! لطفاً Pull Request ارسال کنید.

## 📝 مجوز

این پروژه تحت مجوز MIT است - برای جزئیات به فایل [LICENSE](LICENSE) مراجعه کنید.

## 👥 نویسندگان

- [MmadF14](https://github.com/MmadF14)

## 🙏 قدردانی

- تیم وایرفارد برای راه‌حل VPN عالی‌شان
- فریم‌ورک Echo برای فریم‌ورک وب
- تمام مشارکت‌کنندگان و کاربران این پروژه

</div>

---

<div align="center">
  <img src="https://img.shields.io/github/stars/MmadF14/vwireguard?style=social" alt="ستاره‌های گیت‌هاب">
  <img src="https://img.shields.io/github/forks/MmadF14/vwireguard?style=social" alt="فورک‌های گیت‌هاب">
  <img src="https://img.shields.io/github/watchers/MmadF14/vwireguard?style=social" alt="مشاهده‌کنندگان گیت‌هاب">
</div> 