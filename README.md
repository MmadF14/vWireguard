# vWireguard - WireGuard VPN Management System
# vWireguard - سیستم مدیریت VPN وایرفارد

[![Go Report Card](https://goreportcard.com/badge/github.com/MmadF14/vwireguard)](https://goreportcard.com/report/github.com/MmadF14/vwireguard)
[![GoDoc](https://godoc.org/github.com/MmadF14/vwireguard?status.svg)](https://godoc.org/github.com/MmadF14/vwireguard)
[![License](https://img.shields.io/github/license/MmadF14/vwireguard)](LICENSE)

<div dir="rtl">

[![Go Report Card](https://goreportcard.com/badge/github.com/MmadF14/vwireguard)](https://goreportcard.com/report/github.com/MmadF14/vwireguard)
[![GoDoc](https://godoc.org/github.com/MmadF14/vwireguard?status.svg)](https://godoc.org/github.com/MmadF14/vwireguard)
[![License](https://img.shields.io/github/license/MmadF14/vwireguard)](LICENSE)

</div>

## 📸 Screenshots | تصاویر

### English
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

### فارسی
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

## 🌟 Features | ویژگی‌ها

### English
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

### فارسی
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

## 🚀 Quick Start | شروع سریع

### English
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

### فارسی
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

## 📋 Prerequisites | پیش‌نیازها

### English
- Go 1.16 or higher
- WireGuard installed on the server
- Linux-based system (Ubuntu recommended)
- Root privileges for system operations

### فارسی
- Go 1.16 یا بالاتر
- نصب شده وایرفارد روی سرور
- سیستم مبتنی بر لینوکس (توصیه شده اوبونتو)
- دسترسی root برای عملیات سیستم

## 🛠️ Configuration | پیکربندی

### English
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

### فارسی
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

## 🔒 Security | امنیت

### English
- All passwords are hashed using bcrypt
- HTTPS support for secure communication
- Role-based access control
- Secure key storage and management
- Regular security updates

### فارسی
- تمام رمزهای عبور با bcrypt هش می‌شوند
- پشتیبانی از HTTPS برای ارتباط امن
- کنترل دسترسی مبتنی بر نقش
- ذخیره‌سازی و مدیریت امن کلیدها
- به‌روزرسانی‌های منظم امنیتی

## 🤝 Contributing | مشارکت

### English
Contributions are welcome! Please feel free to submit a Pull Request.

### فارسی
مشارکت‌ها مورد استقبال قرار می‌گیرند! لطفاً Pull Request ارسال کنید.

## 📝 License | مجوز

### English
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

### فارسی
این پروژه تحت مجوز MIT است - برای جزئیات به فایل [LICENSE](LICENSE) مراجعه کنید.

## 👥 Authors | نویسندگان

### English
- [MmadF14](https://github.com/MmadF14)

### فارسی
- [MmadF14](https://github.com/MmadF14)

## 🙏 Acknowledgments | قدردانی

### English
- WireGuard team for their excellent VPN solution
- Echo framework for the web framework
- All contributors and users of this project

### فارسی
- تیم وایرفارد برای راه‌حل VPN عالی‌شان
- فریم‌ورک Echo برای فریم‌ورک وب
- تمام مشارکت‌کنندگان و کاربران این پروژه

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
