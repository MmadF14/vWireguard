# vWireguard

<div align="center">
  <img src="assets/images/vwireguard-logo.png" alt="vWireguard Logo" width="200"/>
  <br>
  <strong>A modern web interface for managing WireGuard VPN</strong>
  <br>
  <br>
  <img src="https://img.shields.io/github/v/release/MmadF14/vWireguard?include_prereleases&sort=semver" alt="Release">
  <img src="https://img.shields.io/github/license/MmadF14/vWireguard" alt="License">
  <img src="https://img.shields.io/github/last-commit/MmadF14/vWireguard" alt="Last Commit">
  <img src="https://img.shields.io/github/issues/MmadF14/vWireguard" alt="Issues">
  <img src="https://img.shields.io/github/pull-requests/MmadF14/vWireguard" alt="Pull Requests">
</div>

<div align="center">
  <h3>
    <a href="#english">English</a> |
    <a href="#فارسی">فارسی</a>
  </h3>
</div>

---

<div id="english">

## English

### Overview
vWireguard is a modern, user-friendly web interface for managing WireGuard VPN servers. It provides a comprehensive set of features for managing VPN clients, server configuration, and system monitoring.

### Features
- 🔒 Secure client management
- 📊 Real-time system monitoring
- 🔄 Automatic configuration generation
- 📧 Email and Telegram integration
- 👥 Multi-user support with role-based access
- 🌐 Multi-language support (English & Persian)
- 📱 Responsive design
- 🔍 System utilities and tools
- 📝 Comprehensive logging system

### Screenshots
<div align="center">
  <img src="assets/images/dashboard.png" alt="Dashboard" width="400"/>
  <br>
  <em>Dashboard Overview</em>
</div>

<div align="center">
  <img src="assets/images/client-management.png" alt="Client Management" width="400"/>
  <br>
  <em>Client Management Interface</em>
</div>

<div align="center">
  <img src="assets/images/system-monitor.png" alt="System Monitor" width="400"/>
  <br>
  <em>System Monitoring Dashboard</em>
</div>

### Installation

#### Prerequisites
- Go 1.16 or higher
- WireGuard installed on your system
- Root/sudo access for system operations

#### Quick Start
```bash
# Clone the repository
git clone https://github.com/MmadF14/vWireguard.git
cd vWireguard

# Build the application
go build

# Run the application
./vWireguard
```

#### Configuration
The application can be configured using environment variables or command-line flags:

```bash
# Basic configuration
./vWireguard --bind-address=0.0.0.0:5000 --disable-login=false

# With email configuration
./vWireguard --email-from=admin@example.com --smtp-hostname=smtp.example.com

# With Telegram integration
./vWireguard --telegram-token=YOUR_BOT_TOKEN
```

### Usage
1. Access the web interface at `http://your-server:5000`
2. Log in with your credentials (default: admin/admin)
3. Start managing your WireGuard VPN

### API Documentation
The application provides a RESTful API for programmatic access:

```bash
# Get all clients
GET /api/clients

# Create a new client
POST /new-client

# Update client status
POST /client/set-status

# Get system metrics
GET /api/system-metrics
```

### Contributing
Contributions are welcome! Please feel free to submit a Pull Request.

### License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

</div>

---

<div id="فارسی">

## فارسی

### معرفی
vWireguard یک رابط کاربری وب مدرن و کاربرپسند برای مدیریت سرورهای WireGuard VPN است. این برنامه مجموعه‌ای جامع از ویژگی‌ها را برای مدیریت کلاینت‌های VPN، پیکربندی سرور و نظارت بر سیستم ارائه می‌دهد.

### ویژگی‌ها
- 🔒 مدیریت امن کلاینت‌ها
- 📊 نظارت بر سیستم در زمان واقعی
- 🔄 تولید خودکار پیکربندی
- 📧 یکپارچه‌سازی ایمیل و تلگرام
- 👥 پشتیبانی از چند کاربر با دسترسی مبتنی بر نقش
- 🌐 پشتیبانی از چند زبان (انگلیسی و فارسی)
- 📱 طراحی واکنش‌گرا
- 🔍 ابزارها و امکانات سیستم
- 📝 سیستم ثبت رویداد جامع

### تصاویر
<div align="center">
  <img src="assets/images/dashboard.png" alt="داشبورد" width="400"/>
  <br>
  <em>نمای کلی داشبورد</em>
</div>

<div align="center">
  <img src="assets/images/client-management.png" alt="مدیریت کلاینت" width="400"/>
  <br>
  <em>رابط مدیریت کلاینت</em>
</div>

<div align="center">
  <img src="assets/images/system-monitor.png" alt="نظارت بر سیستم" width="400"/>
  <br>
  <em>داشبورد نظارت بر سیستم</em>
</div>

### نصب

#### پیش‌نیازها
- Go 1.16 یا بالاتر
- WireGuard نصب شده روی سیستم
- دسترسی root/sudo برای عملیات سیستم

#### شروع سریع
```bash
# کلون کردن مخزن
git clone https://github.com/MmadF14/vWireguard.git
cd vWireguard

# ساخت برنامه
go build

# اجرای برنامه
./vWireguard
```

#### پیکربندی
برنامه را می‌توان با استفاده از متغیرهای محیطی یا پرچم‌های خط فرمان پیکربندی کرد:

```bash
# پیکربندی پایه
./vWireguard --bind-address=0.0.0.0:5000 --disable-login=false

# با پیکربندی ایمیل
./vWireguard --email-from=admin@example.com --smtp-hostname=smtp.example.com

# با یکپارچه‌سازی تلگرام
./vWireguard --telegram-token=YOUR_BOT_TOKEN
```

### استفاده
1. به رابط وب در آدرس `http://your-server:5000` دسترسی پیدا کنید
2. با اطلاعات ورود خود وارد شوید (پیش‌فرض: admin/admin)
3. شروع به مدیریت VPN WireGuard خود کنید

### مستندات API
برنامه یک API RESTful برای دسترسی برنامه‌نویسی ارائه می‌دهد:

```bash
# دریافت تمام کلاینت‌ها
GET /api/clients

# ایجاد کلاینت جدید
POST /new-client

# به‌روزرسانی وضعیت کلاینت
POST /client/set-status

# دریافت معیارهای سیستم
GET /api/system-metrics
```

### مشارکت
مشارکت‌ها مورد استقبال قرار می‌گیرند! لطفاً آزادانه یک Pull Request ارسال کنید.

### مجوز
این پروژه تحت مجوز MIT است - برای جزئیات به فایل [LICENSE](LICENSE) مراجعه کنید.

</div>
