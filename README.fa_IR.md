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

برای نصب سریع می‌توانید اسکریپت `install.sh` را دانلود و اجرا کنید تا آخرین نسخه آماده از گیت‌هاب دریافت و پیکربندی شود.

> **دو نکته که تقریباً همه را موقع بیلد از سورس گیر می‌اندازد:**
>
> ۱. **قبل از `go build` حتماً باید `./prepare_assets.sh` را اجرا کنی.** پوشهٔ
> `assets/` در گیت نادیده گرفته شده (فقط `.gitkeep` کامیت شده) ولی `main.go`
> آن را با `//go:embed assets/*` داخل باینری جاسازی می‌کند. اگر این مرحله را رد
> کنی، یا بیلد با خطای `pattern assets/*: no matching files found` می‌شکند، یا
> باینری‌ای می‌سازی که رابط وبش هیچ CSS/JS ندارد.
>
> ۲. **از `apt install golang-go` استفاده نکن.** نسخهٔ Go در مخزن دبیان/اوبونتو
> قدیمی است (اوبونتو ۲۲.۰۴ نسخهٔ ۱.۱۸ دارد) ولی `go.mod` حداقل **Go 1.21**
> می‌خواهد و بیلد با `go.mod requires go >= 1.21` شکست می‌خورد.

**۱. بسته‌های سیستمی:**
```bash
sudo apt-get update
sudo apt-get install -y git curl wireguard wireguard-tools
```

**۲. نصب Go 1.21 یا بالاتر (از go.dev، نه apt):**
```bash
GO_VERSION=1.21.13
# روی سرور ARM به‌جای linux-amd64 بنویس linux-arm64
curl -fsSLO "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
export PATH=$PATH:/usr/local/go/bin
go version
```

**۳. نصب Node.js و Yarn (فقط برای ساخت فایل‌های رابط کاربری):**
```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs
sudo npm install -g yarn
```

**۴. کلون کردن مخزن:**
```bash
git clone https://github.com/MmadF14/vwireguard.git
cd vwireguard
```

**۵. ساخت فایل‌های رابط کاربری ← همان مرحله‌ای که فراموش می‌شود:**
```bash
chmod +x prepare_assets.sh
./prepare_assets.sh

# بررسی کن که واقعاً ساخته شده باشد
ls assets/dist/js assets/dist/css assets/plugins
```

**۶. ساخت برنامه:**
```bash
go mod download
go build -trimpath -ldflags "-s -w" -o vwireguard .
```

**۷. اجرای برنامه:**
```bash
./vwireguard
```

> **یادآوری مهم:** فعال کردن سرویس پنل، تانل را فعال نمی‌کند. اگر
> `systemctl enable wg-quick@wg0` را نزنی، بعد از ریبوت پنل بالا می‌آید ولی
> `wg show` خالی است و هیچ‌کس وصل نمی‌شود تا وقتی دستی «Apply Config» بزنی.

### بیلد مجدد بعد از تغییر کد

```bash
git pull
./prepare_assets.sh          # فقط اگر فایل‌های فرانت‌اند یا وابستگی‌ها عوض شده
go build -trimpath -o vwireguard .
sudo systemctl restart vwireguard
```

### رفع اشکال

| خطا | علت | راه‌حل |
|---|---|---|
| `pattern assets/*: no matching files found` | `prepare_assets.sh` اجرا نشده | مرحلهٔ ۵ |
| رابط وب بدون استایل بالا می‌آید | موقع بیلد `assets/` خالی بوده | مرحلهٔ ۵ و بیلد مجدد |
| `go.mod requires go >= 1.21` | Go نصب‌شده از apt قدیمی است | مرحلهٔ ۲ |
| `yarn: command not found` | Node/Yarn نصب نیست | مرحلهٔ ۳ |
| بعد از ریبوت کسی وصل نمی‌شود | `wg-quick@wg0` فعال نشده | `sudo systemctl enable wg-quick@wg0` |

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
- نام کاربری و رمز عبور به‌صورت تصادفی ایجاد شده و پس از نصب نمایش داده می‌شود. این اطلاعات در فایل `/root/vwireguard_credentials.txt` نیز ذخیره می‌شود.

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