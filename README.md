# vWireguard - WireGuard VPN Management System
# vWireguard - سیستم مدیریت VPN وایرفارد

<div align="center">
  <h3>
    <a href="#english">English</a> |
    <a href="#فارسی">فارسی</a>
  </h3>
</div>

[![Go Report Card](https://goreportcard.com/badge/github.com/MmadF14/vwireguard)](https://goreportcard.com/report/github.com/MmadF14/vwireguard)
[![GoDoc](https://godoc.org/github.com/MmadF14/vwireguard?status.svg)](https://godoc.org/github.com/MmadF14/vwireguard)
[![License](https://img.shields.io/github/license/MmadF14/vwireguard)](LICENSE)

<div id="english">

## 📸 Screenshots

<div align="center">
  <h3>Dashboard</h3>
  <img src="assets/images/dashboard.png" alt="Dashboard" width="800"/>
  <p><em>Main dashboard showing system overview and statistics</em></p>
</div>

<div align="center">
  <h3>Client Management</h3>
  <img src="assets/images/client-management.png" alt="Client Management" width="800"/>
  <p><em>Client management interface with connection status and configuration options</em></p>
</div>

<div align="center">
  <h3>System Monitor</h3>
  <img src="assets/images/system-monitor.png" alt="System Monitor" width="800"/>
  <p><em>Real-time system monitoring with resource usage graphs</em></p>
</div>

## 🌟 Features

- 🔒 Secure WireGuard VPN management
- 👥 Multi-user support with role-based access control
- 🌐 Wake-on-LAN functionality for remote devices
- 📊 Real-time system monitoring
- 🔄 Automatic client configuration generation
- 📱 Responsive web interface
- 🌍 Bilingual support (English/Persian)
- 📝 Comprehensive logging system
- 🔧 System utilities and maintenance tools
- 🔐 Secure key management

## 🚀 Quick Start

1. Clone the repository:
```bash
git clone https://github.com/MmadF14/vwireguard.git
cd vwireguard
```

2. Install dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build
```

4. Run the application:
```bash
./vwireguard
```

## 📋 Prerequisites

- Go 1.16 or higher
- WireGuard installed on the server
- Linux-based system (Ubuntu recommended)
- Root privileges for system operations

## 🛠️ Configuration

1. Configure WireGuard:
```bash
wg-quick up wg0
```

2. Access the web interface:
```
http://localhost:8080
```

3. Default credentials:
- Username: admin
- Password: admin

## 🔒 Security

- All passwords are hashed using bcrypt
- HTTPS support for secure communication
- Role-based access control
- Secure key storage and management
- Regular security updates

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Authors

- [MmadF14](https://github.com/MmadF14)

## 🙏 Acknowledgments

- WireGuard team for their excellent VPN solution
- Echo framework for the web framework
- All contributors and users of this project

</div>

---

<div id="فارسی">

# vWireguard - سیستم مدیریت VPN وایرفارد

<div dir="rtl" align="center">
  <h3>
    <a href="#english">English</a> |
    <a href="#فارسی">فارسی</a>
  </h3>
</div>

[![Go Report Card](https://goreportcard.com/badge/github.com/MmadF14/vwireguard)](https://goreportcard.com/report/github.com/MmadF14/vwireguard)
[![GoDoc](https://godoc.org/github.com/MmadF14/vwireguard?status.svg)](https://godoc.org/github.com/MmadF14/vwireguard)
[![License](https://img.shields.io/github/license/MmadF14/vwireguard)](LICENSE)

## 📸 تصاویر

<div dir="rtl" align="center">
  <h3>داشبورد</h3>
  <img src="assets/images/dashboard.png" alt="داشبورد" width="800"/>
  <p><em>داشبورد اصلی با نمایش کلی سیستم و آمار</em></p>
</div>

<div dir="rtl" align="center">
  <h3>مدیریت کلاینت‌ها</h3>
  <img src="assets/images/client-management.png" alt="مدیریت کلاینت" width="800"/>
  <p><em>رابط مدیریت کلاینت با وضعیت اتصال و گزینه‌های پیکربندی</em></p>
</div>

<div dir="rtl" align="center">
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

- Go 1.16 یا بالاتر
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
http://localhost:8080
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
  <img src="https://img.shields.io/github/stars/MmadF14/vwireguard?style=social" alt="GitHub Stars">
  <img src="https://img.shields.io/github/forks/MmadF14/vwireguard?style=social" alt="GitHub Forks">
  <img src="https://img.shields.io/github/watchers/MmadF14/vwireguard?style=social" alt="GitHub Watchers">
</div>

<div dir="rtl" align="center">
  <img src="https://img.shields.io/github/stars/MmadF14/vwireguard?style=social" alt="ستاره‌های گیت‌هاب">
  <img src="https://img.shields.io/github/forks/MmadF14/vwireguard?style=social" alt="فورک‌های گیت‌هاب">
  <img src="https://img.shields.io/github/watchers/MmadF14/vwireguard?style=social" alt="مشاهده‌کنندگان گیت‌هاب">
</div>
