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
        "IP Allocation": "IP Allocation",
        "Endpoint": "Endpoint",
        "Telegram userid": "Telegram userid",
        "Notes": "Notes",
        "Additional notes about this client": "Additional notes about this client",
        "Use server DNS": "Use server DNS",
        "Public Key": "Public Key",
        "Preshared Key": "Preshared Key",
        "Additional configuration": "Additional configuration",
        "Allowed IPs": "Allowed IPs",
        "Extra Allowed IPs": "Extra Allowed IPs",
        "Subnet range": "Subnet range",
        "Select a subnet range": "Select a subnet range",
        "Cancel": "Cancel",
        "Send": "Send",
        "Save": "Save",
        
        // Messages
        "Enter custom quota in bytes": "Enter custom quota in bytes",
        
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
        "Use selected IP address": "Use selected IP address",

        // System Monitor
        "System Overview": "System Overview",
        "CPU": "CPU",
        "Memory": "Memory",
        "Storage": "Storage",
        "Network Traffic": "Network Traffic",
        "Upload": "Upload",
        "Download": "Download",
        "Last 24 Hours": "Last 24 Hours",
        "Last 7 Days": "Last 7 Days",
        "Last 30 Days": "Last 30 Days",
        "Real-time": "Real-time",
        "Refresh": "Refresh",
        "Auto Refresh": "Auto Refresh",
        "System Load": "System Load",
        "Used": "Used",
        "Free": "Free",
        "Total": "Total",
        "System Monitor": "System Monitor",
        "System Status": "System Status",
        "Running": "Running",
        "Network": "Network",
        "Total Out": "Total Out",
        "Total In": "Total In",
        "Uptime": "Uptime",
        "KB/s": "KB/s",
        "GB": "GB",
        "MB": "MB",
        "Cores": "Cores",
        "Restoring...": "Restoring...",

        // Client Creation
        "Create New Client": "Create New Client",
        "Client Information": "Client Information",
        "Client Name": "Client Name",
        "Client Email": "Client Email",
        "Data Quota": "Data Quota",
        "Unlimited": "Unlimited",
        "Network Settings": "Network Settings",
        "Client IP": "Client IP",
        "DNS Settings": "DNS Settings",
        "Advanced Settings": "Advanced Settings",
        "Generate Keys": "Generate Keys",
        "Show QR Code": "Show QR Code",
        "Download Config": "Download Config",
        "Client Status": "Client Status",
        "Created Successfully": "Created Successfully",
        "Creation Failed": "Creation Failed",
        "Validate Settings": "Validate Settings",
        "Generate New Keys": "Generate New Keys",

        // Wireguard Server
        "Server Configuration": "Server Configuration",
        "Server Status": "Server Status",
        "Server IP": "Server IP",
        "Listen Port": "Listen Port",
        "Private Key": "Private Key",
        "Server Settings": "Server Settings",
        "Interface Name": "Interface Name",
        "Post Up Script": "Post Up Script",
        "Post Down Script": "Post Down Script",
        "Save Configuration": "Save Configuration",
        "Restart Server": "Restart Server",
        "Stop Server": "Stop Server",
        "Start Server": "Start Server",
        "Stopped": "Stopped",
        "Apply Changes": "Apply Changes",
        "Configuration Saved": "Configuration Saved",
        "Server Restarted": "Server Restarted",

        // Additional Global Settings
        "Interface Settings": "Interface Settings",
        "Routing Settings": "Routing Settings",
        "Security Settings": "Security Settings",
        "Advanced Configuration": "Advanced Configuration",
        "Network Interface": "Network Interface",
        "Port Settings": "Port Settings",
        "IP Range": "IP Range",
        "Subnet Mask": "Subnet Mask",
        "Default Gateway": "Default Gateway",
        "Apply Settings": "Apply Settings",
        "Reset Settings": "Reset Settings",
        "Import Configuration": "Import Configuration",
        "Export Configuration": "Export Configuration",
        "Backup Configuration": "Backup Configuration",
        "Restore Configuration": "Restore Configuration",

        // User Management
        "User Management": "User Management",
        "Add User": "Add User",
        "Edit User": "Edit User",
        "Delete User": "Delete User",
        "User Role": "User Role",
        "Change Password": "Change Password",
        "User Permissions": "User Permissions",
        "Access Level": "Access Level",
        "Account Status": "Account Status",
        "Last Login": "Last Login",
        "Created Date": "Created Date",
        "Modified Date": "Modified Date",
        "Active": "Active",
        "Inactive": "Inactive",
        "Suspended": "Suspended",
        "Reset Password": "Reset Password",
        "Confirm Password": "Confirm Password",

        // System Monitor Additional
        "Disk": "Disk",
        "Swap": "Swap",
        "RAM": "RAM",
        "System Load": "System Load",
        "+Wireguard": "+Wireguard",

        // System Backup & Restore
        "System Backup & Restore": "System Backup & Restore",
        "Create Backup": "Create Backup",
        "Download a complete backup of your system configuration and database.": "Download a complete backup of your system configuration and database.",
        "Download Backup": "Download Backup",
        "Restore Backup": "Restore Backup",
        "Select a backup file to restore your system configuration and database.": "Select a backup file to restore your system configuration and database.",
        "Choose backup file:": "Choose backup file:",
        "Restoring...": "Restoring...",

        // Persistent Keepalive Help
        "keepalive_help": "By default, WireGuard peers remain silent while they do not need to communicate, so peers located behind a NAT and/or firewall may be unreachable from other peers until they reach out to other peers themselves. Adding PersistentKeepalive can ensure that the connection remains open.",

        // Server Page
        "Wireguard Server Settings": "Wireguard Server Settings",
        "Server Interface Addresses": "Server Interface Addresses",
        "Listen Port": "Listen Port",
        "Post Up Script": "Post Up Script",
        "Pre Down Script": "Pre Down Script",
        "Post Down Script": "Post Down Script",
        "Check Interval (minutes)": "Check Interval (minutes)",
        "Key Pair": "Key Pair",
        "Private Key": "Private Key",
        "Public Key": "Public Key",
        "Show": "Show",
        "Generate": "Generate",
        "KeyPair Generation": "KeyPair Generation",
        "keypair_generation_warning": "Are you sure to generate a new key pair for the Wireguard server?\nThe existing Client's peer public key need to be updated to keep the connection working.",
        "Add More": "Add More",
        "Please enter a port": "Please enter a port",
        "Port must be an integer": "Port must be an integer",
        "Port must be in range 1..65535": "Port must be in range 1..65535",
        "Updated Wireguard server interface addresses successfully": "Updated Wireguard server interface addresses successfully",
        "Generate new key pair successfully": "Generate new key pair successfully",

        "Auto Refresh": "Auto Refresh",
        "5 seconds": "5 seconds",
        "10 seconds": "10 seconds",
        "30 seconds": "30 seconds",
        "60 seconds": "60 seconds",
        "Refresh Status": "Refresh Status",
        "Email Configuration": "Email Configuration",
        "Email address": "Email address",
        "QR Code": "QR Code",
        "Telegram Configuration": "Telegram Configuration",
        "Edit Client": "Edit Client",
        "Select Quota": "Select Quota",
        "Enter custom quota in GB": "Enter custom quota in GB",
        "Enable this client": "Enable this client",
        "Public and Preshared Keys": "Public and Preshared Keys",
        "Update the server stored client Public and Preshared keys.": "Update the server stored client Public and Preshared keys.",
        "Disable": "Disable",
        "Remove": "Remove",
        "Apply": "Apply",
        "Do you want to write config file and restart WireGuard server?": "Do you want to write config file and restart WireGuard server?",
        "Enable after creation": "Enable after creation",
        "Autogenerated": "Autogenerated",
        "Autogenerated - enter \"-\" to skip generation": "Autogenerated - enter \"-\" to skip generation",
        "Specify a list of addresses that will get routed to the server. These addresses will be included in 'AllowedIPs' of client config": "Specify a list of addresses that will get routed to the server. These addresses will be included in 'AllowedIPs' of client config",
        "Specify a list of addresses that will get routed to the client. These addresses will be included in 'AllowedIPs' of WG server config": "Specify a list of addresses that will get routed to the client. These addresses will be included in 'AllowedIPs' of WG server config",
        "If you don't want to let the server generate and store the client's private key, you can manually specify its public and preshared key here. Note: QR code will not be generated": "If you don't want to let the server generate and store the client's private key, you can manually specify its public and preshared key here. Note: QR code will not be generated",
        "Submit": "Submit",

        // Users Settings
        "Users Settings": "Users Settings",
        "Edit User": "Edit User",
        "Edit user": "Edit user",
        "Add new user": "Add new user",
        "New User": "New User",
        "Name": "Name",
        "Password": "Password",
        "Role": "Role",
        "User": "User",
        "Manager": "Manager",
        "Admin": "Admin",
        "Remove": "Remove",
        "You are about to remove user": "You are about to remove user",
        "Leave empty to keep the password unchanged": "Leave empty to keep the password unchanged",
        "Removed user successfully": "Removed user successfully",
        "Created user successfully": "Created user successfully",
        "Updated user information successfully": "Updated user information successfully",

        // Wake On Lan Hosts
        "Wake On Lan Hosts": "Wake On Lan Hosts",
        "New Wake On Lan Host": "New Wake On Lan Host",
        "Mac Address": "Mac Address",
        "Wake On": "Wake On",
        "Edit": "Edit",
        "Remove": "Remove",
        "Unused": "Unused",
        "You are about to remove host": "You are about to remove host",
        "Host removed successfully": "Host removed successfully",
        "Host created successfully": "Host created successfully",
        "Host updated successfully": "Host updated successfully",
        "Failed to wake up host": "Failed to wake up host",
        "Host woken up successfully": "Host woken up successfully",

        "About": "About",
        "About vWireguard": "About vWireguard",
        "Current version": "Current version",
        "git commit hash": "git commit hash",
        "Current version release date": "Current version release date",
        "Latest release": "Latest release",
        "Latest release date": "Latest release date",
        "Author": "Author",
        "Contributors": "Contributors",
        "Copyright": "Copyright",
        "All rights reserved": "All rights reserved",
        "Could not find this version on GitHub.com": "Could not find this version on GitHub.com",
        "Could not connect to GitHub.com": "Could not connect to GitHub.com",
        "Current version is out of date": "Current version is out of date"
    },
    fa: {
        // Navigation
        "MAIN": "اصلی",
        "SETTINGS": "تنظیمات",
        "UTILITIES": "ابزارها",
        "Help": "راهنما",
        "Logout": "خروج",
        
        // Menu Items
        "System Monitor": "نمایشگر سیستم",
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
        "IP Allocation": "تخصیص IP",
        "Endpoint": "نقطه پایانی",
        "Telegram userid": "شناسه تلگرام",
        "Notes": "یادداشت‌ها",
        "Additional notes about this client": "یادداشت‌های اضافی درباره این کلاینت",
        "Use server DNS": "استفاده از DNS سرور",
        "Public Key": "کلید عمومی",
        "Preshared Key": "کلید از پیش تعیین شده",
        "Additional configuration": "تنظیمات اضافی",
        "Allowed IPs": "IP های مجاز",
        "Extra Allowed IPs": "IP های مجاز اضافی",
        "Subnet range": "محدوده شبکه",
        "Select a subnet range": "محدوده شبکه را انتخاب کنید",
        "Cancel": "لغو",
        "Send": "ارسال",
        "Save": "ذخیره",
        
        // Messages
        "Enter custom quota in bytes": "سهمیه سفارشی را به بایت وارد کنید",
        
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
        "Use selected IP address": "استفاده از آدرس IP انتخاب شده",

        // System Monitor
        "System Overview": "نمای کلی سیستم",
        "CPU": "پردازنده",
        "Memory": "حافظه",
        "Storage": "فضای ذخیره‌سازی",
        "Network Traffic": "ترافیک شبکه",
        "Upload": "آپلود",
        "Download": "دانلود",
        "Last 24 Hours": "۲۴ ساعت گذشته",
        "Last 7 Days": "۷ روز گذشته",
        "Last 30 Days": "۳۰ روز گذشته",
        "Real-time": "بلادرنگ",
        "Refresh": "تازه‌سازی",
        "Auto Refresh": "تازه‌سازی خودکار",
        "System Load": "بار سیستم",
        "Used": "استفاده شده",
        "Free": "آزاد",
        "Total": "کل",
        "System Monitor": "نمایشگر سیستم",
        "System Status": "وضعیت سیستم",
        "Running": "در حال اجرا",
        "Network": "شبکه",
        "Total Out": "کل خروجی",
        "Total In": "کل ورودی",
        "Uptime": "زمان کارکرد",
        "KB/s": "کیلوبایت/ثانیه",
        "GB": "گیگابایت",
        "MB": "مگابایت",
        "Cores": "هسته",
        "Restoring...": "در حال بازیابی...",

        // Client Creation
        "Create New Client": "ایجاد کلاینت جدید",
        "Client Information": "اطلاعات کلاینت",
        "Client Name": "نام کلاینت",
        "Client Email": "ایمیل کلاینت",
        "Data Quota": "سهمیه داده",
        "Unlimited": "نامحدود",
        "Network Settings": "تنظیمات شبکه",
        "Client IP": "آی‌پی کلاینت",
        "DNS Settings": "تنظیمات DNS",
        "Advanced Settings": "تنظیمات پیشرفته",
        "Generate Keys": "تولید کلیدها",
        "Show QR Code": "نمایش کد QR",
        "Download Config": "دانلود پیکربندی",
        "Client Status": "وضعیت کلاینت",
        "Created Successfully": "با موفقیت ایجاد شد",
        "Creation Failed": "ایجاد ناموفق بود",
        "Validate Settings": "اعتبارسنجی تنظیمات",
        "Generate New Keys": "تولید کلیدهای جدید",

        // Wireguard Server
        "Server Configuration": "پیکربندی سرور",
        "Server Status": "وضعیت سرور",
        "Server IP": "آی‌پی سرور",
        "Listen Port": "پورت شنود",
        "Private Key": "کلید خصوصی",
        "Server Settings": "تنظیمات سرور",
        "Interface Name": "نام رابط",
        "Post Up Script": "اسکریپت پس از راه‌اندازی",
        "Post Down Script": "اسکریپت پس از توقف",
        "Save Configuration": "ذخیره پیکربندی",
        "Restart Server": "راه‌اندازی مجدد سرور",
        "Stop Server": "توقف سرور",
        "Start Server": "شروع سرور",
        "Stopped": "متوقف شده",
        "Apply Changes": "اعمال تغییرات",
        "Configuration Saved": "پیکربندی ذخیره شد",
        "Server Restarted": "سرور مجدداً راه‌اندازی شد",

        // Additional Global Settings
        "Interface Settings": "تنظیمات رابط",
        "Routing Settings": "تنظیمات مسیریابی",
        "Security Settings": "تنظیمات امنیتی",
        "Advanced Configuration": "پیکربندی پیشرفته",
        "Network Interface": "رابط شبکه",
        "Port Settings": "تنظیمات پورت",
        "IP Range": "محدوده IP",
        "Subnet Mask": "ماسک شبکه",
        "Default Gateway": "دروازه پیش‌فرض",
        "Apply Settings": "اعمال تنظیمات",
        "Reset Settings": "بازنشانی تنظیمات",
        "Import Configuration": "وارد کردن پیکربندی",
        "Export Configuration": "خروجی گرفتن از پیکربندی",
        "Backup Configuration": "پشتیبان‌گیری از پیکربندی",
        "Restore Configuration": "بازیابی پیکربندی",

        // User Management
        "User Management": "مدیریت کاربران",
        "Add User": "افزودن کاربر",
        "Edit User": "ویرایش کاربر",
        "Delete User": "حذف کاربر",
        "User Role": "نقش کاربر",
        "Change Password": "تغییر رمز عبور",
        "User Permissions": "مجوزهای کاربر",
        "Access Level": "سطح دسترسی",
        "Account Status": "وضعیت حساب",
        "Last Login": "آخرین ورود",
        "Created Date": "تاریخ ایجاد",
        "Modified Date": "تاریخ ویرایش",
        "Active": "فعال",
        "Inactive": "غیرفعال",
        "Suspended": "معلق",
        "Reset Password": "بازنشانی رمز عبور",
        "Confirm Password": "تأیید رمز عبور",

        // System Monitor Additional
        "Disk": "دیسک",
        "Swap": "سواپ",
        "RAM": "رم",
        "System Load": "بار سیستم",
        "+Wireguard": "+وایرگارد",

        // System Backup & Restore
        "System Backup & Restore": "پشتیبان‌گیری و بازیابی سیستم",
        "Create Backup": "ایجاد پشتیبان",
        "Download a complete backup of your system configuration and database.": "دانلود یک نسخه پشتیبان کامل از پیکربندی سیستم و پایگاه داده.",
        "Download Backup": "دانلود پشتیبان",
        "Restore Backup": "بازیابی پشتیبان",
        "Select a backup file to restore your system configuration and database.": "یک فایل پشتیبان برای بازیابی پیکربندی سیستم و پایگاه داده انتخاب کنید.",
        "Choose backup file:": "انتخاب فایل پشتیبان:",
        "Restoring...": "در حال بازیابی...",

        // Persistent Keepalive Help
        "keepalive_help": "به طور پیش‌فرض، همتایان وایرگارد در زمانی که نیازی به ارتباط ندارند ساکت می‌مانند، بنابراین همتایانی که پشت NAT و/یا فایروال قرار دارند ممکن است تا زمانی که خودشان با همتایان دیگر ارتباط برقرار نکنند، از دسترس خارج باشند. افزودن PersistentKeepalive می‌تواند اطمینان دهد که ارتباط باز می‌ماند.",

        // Server Page
        "Wireguard Server Settings": "تنظیمات سرور وایرگارد",
        "Server Interface Addresses": "آدرس‌های رابط سرور",
        "Listen Port": "پورت شنود",
        "Post Up Script": "اسکریپت پس از راه‌اندازی",
        "Pre Down Script": "اسکریپت قبل از توقف",
        "Post Down Script": "اسکریپت پس از توقف",
        "Check Interval (minutes)": "فاصله بررسی (دقیقه)",
        "Key Pair": "جفت کلید",
        "Private Key": "کلید خصوصی",
        "Public Key": "کلید عمومی",
        "Show": "نمایش",
        "Generate": "تولید",
        "KeyPair Generation": "تولید جفت کلید",
        "keypair_generation_warning": "آیا مطمئن هستید که می‌خواهید یک جفت کلید جدید برای سرور وایرگارد تولید کنید؟\nکلید عمومی همتای کلاینت‌های موجود باید به‌روزرسانی شود تا اتصال کار کند.",
        "Add More": "افزودن بیشتر",
        "Please enter a port": "لطفاً یک پورت وارد کنید",
        "Port must be an integer": "پورت باید یک عدد صحیح باشد",
        "Port must be in range 1..65535": "پورت باید بین 1 تا 65535 باشد",
        "Updated Wireguard server interface addresses successfully": "آدرس‌های رابط سرور وایرگارد با موفقیت به‌روزرسانی شد",
        "Generate new key pair successfully": "جفت کلید جدید با موفقیت تولید شد",

        "Auto Refresh": "بروزرسانی خودکار",
        "5 seconds": "۵ ثانیه",
        "10 seconds": "۱۰ ثانیه",
        "30 seconds": "۳۰ ثانیه",
        "60 seconds": "۶۰ ثانیه",
        "Refresh Status": "بروزرسانی وضعیت",
        "Email Configuration": "تنظیمات ایمیل",
        "Email address": "آدرس ایمیل",
        "QR Code": "کد QR",
        "Telegram Configuration": "تنظیمات تلگرام",
        "Edit Client": "ویرایش کلاینت",
        "Select Quota": "انتخاب سهمیه",
        "Enter custom quota in GB": "سهمیه سفارشی را به گیگابایت وارد کنید",
        "Enable this client": "فعال‌سازی این کلاینت",
        "Public and Preshared Keys": "کلیدهای عمومی و از پیش تعیین شده",
        "Update the server stored client Public and Preshared keys.": "بروزرسانی کلیدهای عمومی و از پیش تعیین شده کلاینت در سرور",
        "Disable": "غیرفعال‌سازی",
        "Remove": "حذف",
        "Apply": "اعمال",
        "Do you want to write config file and restart WireGuard server?": "آیا می‌خواهید فایل پیکربندی را بنویسید و سرور وایرگارد را مجدداً راه‌اندازی کنید؟",
        "Enable after creation": "فعال‌سازی پس از ایجاد",
        "Autogenerated": "تولید خودکار",
        "Autogenerated - enter \"-\" to skip generation": "تولید خودکار - برای رد کردن تولید \"-\" وارد کنید",
        "Specify a list of addresses that will get routed to the server. These addresses will be included in 'AllowedIPs' of client config": "لیستی از آدرس‌هایی که به سرور مسیریابی می‌شوند را مشخص کنید. این آدرس‌ها در 'AllowedIPs' پیکربندی کلاینت گنجانده خواهند شد",
        "Specify a list of addresses that will get routed to the client. These addresses will be included in 'AllowedIPs' of WG server config": "لیستی از آدرس‌هایی که به کلاینت مسیریابی می‌شوند را مشخص کنید. این آدرس‌ها در 'AllowedIPs' پیکربندی سرور وایرگارد گنجانده خواهند شد",
        "If you don't want to let the server generate and store the client's private key, you can manually specify its public and preshared key here. Note: QR code will not be generated": "اگر نمی‌خواهید سرور کلید خصوصی کلاینت را تولید و ذخیره کند، می‌توانید کلید عمومی و کلید از پیش تعیین شده را در اینجا به صورت دستی مشخص کنید. توجه: کد QR تولید نخواهد شد",
        "Submit": "ثبت",

        // Users Settings
        "Users Settings": "تنظیمات کاربران",
        "Edit User": "ویرایش کاربر",
        "Edit user": "ویرایش کاربر",
        "Add new user": "افزودن کاربر جدید",
        "New User": "کاربر جدید",
        "Name": "نام",
        "Password": "رمز عبور",
        "Role": "نقش",
        "User": "کاربر",
        "Manager": "مدیر",
        "Admin": "مدیر کل",
        "Remove": "حذف",
        "You are about to remove user": "شما در حال حذف کاربر زیر هستید",
        "Leave empty to keep the password unchanged": "برای حفظ رمز عبور فعلی، این فیلد را خالی بگذارید",
        "Removed user successfully": "کاربر با موفقیت حذف شد",
        "Created user successfully": "کاربر با موفقیت ایجاد شد",
        "Updated user information successfully": "اطلاعات کاربر با موفقیت به‌روزرسانی شد",

        // Wake On Lan Hosts
        "Wake On Lan Hosts": "میزبان‌های Wake On Lan",
        "New Wake On Lan Host": "میزبان Wake On Lan جدید",
        "Mac Address": "آدرس مک",
        "Wake On": "روشن کردن",
        "Edit": "ویرایش",
        "Remove": "حذف",
        "Unused": "استفاده نشده",
        "You are about to remove host": "شما در حال حذف میزبان زیر هستید",
        "Host removed successfully": "میزبان با موفقیت حذف شد",
        "Host created successfully": "میزبان با موفقیت ایجاد شد",
        "Host updated successfully": "میزبان با موفقیت به‌روزرسانی شد",
        "Failed to wake up host": "روشن کردن میزبان با خطا مواجه شد",
        "Host woken up successfully": "میزبان با موفقیت روشن شد",

        "About": "درباره",
        "About vWireguard": "درباره وایرگارد",
        "Current version": "نسخه فعلی",
        "git commit hash": "هش کامیت گیت",
        "Current version release date": "تاریخ انتشار نسخه فعلی",
        "Latest release": "آخرین نسخه",
        "Latest release date": "تاریخ آخرین نسخه",
        "Author": "نویسنده",
        "Contributors": "مشارکت‌کنندگان",
        "Copyright": "حق نشر",
        "All rights reserved": "تمامی حقوق محفوظ است",
        "Could not find this version on GitHub.com": "این نسخه در گیت‌هاب یافت نشد",
        "Could not connect to GitHub.com": "اتصال به گیت‌هاب امکان‌پذیر نیست",
        "Current version is out of date": "نسخه فعلی قدیمی است",
        "System Monitor Additional": "اطلاعات اضافی سیستم",
        "Disk": "دیسک",
        "Swap": "فضای مبادله",
        "RAM": "حافظه",
        "System Load": "بار سیستم",
        "+Wireguard": "+وایرگارد"
    }
}; 