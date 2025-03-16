const translations = {
    en: {
        // Navigation
        "MAIN": "MAIN",
        "SETTINGS": "SETTINGS",
        "UTILITIES": "UTILITIES",
        "Help": "Help",
        "Logout": "Logout",
        
        // Menu Items
        "System Monitor": "System Monitor",
        "Wireguard Clients": "Wireguard Clients",
        "Wireguard Server": "Wireguard Server",
        "Global Settings": "Global Settings",
        "Users Settings": "Users Settings",
        "Status": "Status",
        
        // Login Page
        "Sign in to start your session": "Sign in to start your session",
        "Username": "Username",
        "Password": "Password",
        "Remember Me": "Remember Me",
        "Sign In": "Sign In",
        
        // User Panel
        "Administrator": "Administrator",
        "Manager": "Manager",
        
        // Buttons & Actions
        "New Client": "New Client",
        "Apply Config": "Apply Config",
        
        // Client Modal
        "New Wireguard Client": "New Wireguard Client",
        "Name": "Name",
        "Email": "Email",
        "Quota": "Quota",
        "Custom": "Custom",
        "Expiration Date": "Expiration Date",
        "Subnet range": "Subnet range",
        "IP Allocation": "IP Allocation",
        "Allowed IPs": "Allowed IPs",
        "Extra Allowed IPs": "Extra Allowed IPs",
        "Endpoint": "Endpoint",
        "Use server DNS": "Use server DNS",
        "Enable after creation": "Enable after creation",
        "Public Key": "Public Key",
        "Preshared Key": "Preshared Key",
        "Additional configuration": "Additional configuration",
        "Telegram userid": "Telegram userid",
        "Notes": "Notes",
        "Cancel": "Cancel",
        "Submit": "Submit",
        
        // Messages
        "Select a subnet range": "Select a subnet range",
        "Enter custom quota in bytes": "Enter custom quota in bytes",
        "Additional notes about this client": "Additional notes about this client",
        
        // Search
        "Search": "Search",
        "All": "All",
        "Enabled": "Enabled",
        "Disabled": "Disabled",
        "Connected": "Connected",
        "Disconnected": "Disconnected",
        
        // Utility Page
        "System Utilities": "System Utilities",
        "System Status": "System Status",
        "System Information": "System Information",
        "CPU Usage": "CPU Usage",
        "Memory Usage": "Memory Usage",
        "Disk Usage": "Disk Usage",
        "Network Status": "Network Status",
        "Interface Status": "Interface Status",
        "Active Connections": "Active Connections",
        "Total Data Transfer": "Total Data Transfer",
        "System Tools": "System Tools",
        "Restart WireGuard Service": "Restart WireGuard Service",
        "Flush DNS Cache": "Flush DNS Cache",
        "Check for Updates": "Check for Updates",
        "Generate System Report": "Generate System Report",
        "Logs": "Logs",
        "Log Level": "Log Level",
        "System Logs": "System Logs",
        "Refresh Logs": "Refresh Logs",
        "Clear Logs": "Clear Logs",
        "Error": "Error",
        "Warning": "Warning",
        "Info": "Info",
        "Debug": "Debug",
        "Operation Status": "Operation Status",
        "Close": "Close",

        // Status Messages
        "Service restart initiated": "Service restart initiated",
        "DNS cache flushed successfully": "DNS cache flushed successfully",
        "Checking for updates...": "Checking for updates...",
        "Generating system report...": "Generating system report...",
        "Refreshing logs...": "Refreshing logs...",
        "Logs cleared successfully": "Logs cleared successfully",
        
        // Global Settings
        "Language Settings": "Language Settings",
        "Select Language": "Select Language",
        "Wireguard Global Settings": "Wireguard Global Settings",
        "Endpoint Address": "Endpoint Address",
        "DNS Servers": "DNS Servers",
        "MTU": "MTU",
        "Persistent Keepalive": "Persistent Keepalive",
        "Firewall Mark": "Firewall Mark",
        "Table": "Table",
        "Wireguard Config File Path": "Wireguard Config File Path",
        "Save": "Save",
        "Suggest": "Suggest",

        // Help Section
        "The public IP address of your Wireguard server": "The public IP address of your Wireguard server",
        "Click on Suggest button to auto detect": "Click on Suggest button to auto detect",
        "The DNS servers will be set to client config": "The DNS servers will be set to client config",
        "The MTU will be set to server and client config": "The MTU will be set to server and client config",
        "By default it is": "By default it is",
        "Leave blank to omit this setting": "Leave blank to omit this setting",
        "The path of your Wireguard server config file": "The path of your Wireguard server config file",

        // Validation Messages
        "msg_mtu_digits": "MTU must be an integer",
        "msg_mtu_range": "MTU must be in range 68..65535",
        "msg_keepalive_digits": "Persistent keepalive must be an integer",
        "msg_config_required": "Please enter WireGuard config file path",
        "msg_success": "Settings saved successfully",
        "msg_error": "Error",

        // Modal Titles
        "Endpoint Address Suggestion": "Endpoint Address Suggestion",
        "IP addresses for your consideration": "Following is the list of public and local IP addresses for your consideration",
        "Use selected IP address": "Use selected IP address"
    },
    fa: {
        // Navigation
        "MAIN": "اصلی",
        "SETTINGS": "تنظیمات",
        "UTILITIES": "ابزارها",
        "Help": "راهنما",
        "Logout": "خروج",
        
        // Menu Items
        "System Monitor": "نظارت سیستم",
        "Wireguard Clients": "کلاینت‌های وایرگارد",
        "Wireguard Server": "سرور وایرگارد",
        "Global Settings": "تنظیمات عمومی",
        "Users Settings": "تنظیمات کاربران",
        "Status": "وضعیت",
        
        // Login Page
        "Sign in to start your session": "برای شروع جلسه وارد شوید",
        "Username": "نام کاربری",
        "Password": "رمز عبور",
        "Remember Me": "مرا به خاطر بسپار",
        "Sign In": "ورود",
        
        // User Panel
        "Administrator": "مدیر",
        "Manager": "کاربر",
        
        // Buttons & Actions
        "New Client": "کلاینت جدید",
        "Apply Config": "اعمال تنظیمات",
        
        // Client Modal
        "New Wireguard Client": "کلاینت جدید وایرگارد",
        "Name": "نام",
        "Email": "ایمیل",
        "Quota": "سهمیه",
        "Custom": "سفارشی",
        "Expiration Date": "تاریخ انقضا",
        "Subnet range": "محدوده شبکه",
        "IP Allocation": "تخصیص IP",
        "Allowed IPs": "IP های مجاز",
        "Extra Allowed IPs": "IP های مجاز اضافی",
        "Endpoint": "نقطه پایانی",
        "Use server DNS": "استفاده از DNS سرور",
        "Enable after creation": "فعال‌سازی پس از ایجاد",
        "Public Key": "کلید عمومی",
        "Preshared Key": "کلید از پیش تعیین شده",
        "Additional configuration": "تنظیمات اضافی",
        "Telegram userid": "شناسه تلگرام",
        "Notes": "یادداشت‌ها",
        "Cancel": "لغو",
        "Submit": "ثبت",
        
        // Messages
        "Select a subnet range": "یک محدوده شبکه انتخاب کنید",
        "Enter custom quota in bytes": "سهمیه سفارشی را به بایت وارد کنید",
        "Additional notes about this client": "یادداشت‌های اضافی درباره این کلاینت",
        
        // Search
        "Search": "جستجو",
        "All": "همه",
        "Enabled": "فعال",
        "Disabled": "غیرفعال",
        "Connected": "متصل",
        "Disconnected": "قطع",
        
        // Utility Page
        "System Utilities": "ابزارهای سیستم",
        "System Status": "وضعیت سیستم",
        "System Information": "اطلاعات سیستم",
        "CPU Usage": "مصرف پردازنده",
        "Memory Usage": "مصرف حافظه",
        "Disk Usage": "مصرف دیسک",
        "Network Status": "وضعیت شبکه",
        "Interface Status": "وضعیت رابط",
        "Active Connections": "اتصالات فعال",
        "Total Data Transfer": "کل انتقال داده",
        "System Tools": "ابزارهای سیستم",
        "Restart WireGuard Service": "راه‌اندازی مجدد سرویس وایرگارد",
        "Flush DNS Cache": "پاک کردن حافظه DNS",
        "Check for Updates": "بررسی به‌روزرسانی",
        "Generate System Report": "ایجاد گزارش سیستم",
        "Logs": "گزارش‌ها",
        "Log Level": "سطح گزارش",
        "System Logs": "گزارش‌های سیستم",
        "Refresh Logs": "تازه‌سازی گزارش‌ها",
        "Clear Logs": "پاک کردن گزارش‌ها",
        "Error": "خطا",
        "Warning": "هشدار",
        "Info": "اطلاعات",
        "Debug": "اشکال‌زدایی",
        "Operation Status": "وضعیت عملیات",
        "Close": "بستن",

        // Status Messages
        "Service restart initiated": "راه‌اندازی مجدد سرویس آغاز شد",
        "DNS cache flushed successfully": "حافظه DNS با موفقیت پاک شد",
        "Checking for updates...": "در حال بررسی به‌روزرسانی...",
        "Generating system report...": "در حال ایجاد گزارش سیستم...",
        "Refreshing logs...": "در حال تازه‌سازی گزارش‌ها...",
        "Logs cleared successfully": "گزارش‌ها با موفقیت پاک شدند",
        
        // Global Settings
        "Language Settings": "تنظیمات زبان",
        "Select Language": "انتخاب زبان",
        "Wireguard Global Settings": "تنظیمات عمومی وایرگارد",
        "Endpoint Address": "آدرس نقطه پایانی",
        "DNS Servers": "سرورهای DNS",
        "MTU": "MTU",
        "Persistent Keepalive": "نگهداری ارتباط",
        "Firewall Mark": "نشانه فایروال",
        "Table": "جدول",
        "Wireguard Config File Path": "مسیر فایل پیکربندی وایرگارد",
        "Save": "ذخیره",
        "Suggest": "پیشنهاد",

        // Help Section
        "The public IP address of your Wireguard server": "آدرس IP عمومی سرور وایرگارد شما",
        "Click on Suggest button to auto detect": "برای تشخیص خودکار روی دکمه پیشنهاد کلیک کنید",
        "The DNS servers will be set to client config": "سرورهای DNS در تنظیمات کلاینت تنظیم خواهند شد",
        "The MTU will be set to server and client config": "MTU در تنظیمات سرور و کلاینت تنظیم خواهد شد",
        "By default it is": "به طور پیش‌فرض",
        "Leave blank to omit this setting": "برای حذف این تنظیم خالی بگذارید",
        "The path of your Wireguard server config file": "مسیر فایل پیکربندی سرور وایرگارد شما",

        // Validation Messages
        "msg_mtu_digits": "MTU باید یک عدد صحیح باشد",
        "msg_mtu_range": "MTU باید بین 68 تا 65535 باشد",
        "msg_keepalive_digits": "نگهداری ارتباط باید یک عدد صحیح باشد",
        "msg_config_required": "لطفاً مسیر فایل پیکربندی وایرگارد را وارد کنید",
        "msg_success": "تنظیمات با موفقیت ذخیره شد",
        "msg_error": "خطا",

        // Modal Titles
        "Endpoint Address Suggestion": "پیشنهاد آدرس نقطه پایانی",
        "IP addresses for your consideration": "لیست آدرس‌های IP عمومی و محلی برای انتخاب شما",
        "Use selected IP address": "استفاده از آدرس IP انتخاب شده"
    }
}; 