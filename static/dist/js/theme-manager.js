// Theme and Language Manager
class ThemeLanguageManager {
    constructor() {
        this.init();
    }
    
    init() {
        // Initialize theme
        this.initializeTheme();
        
        // Initialize language/direction
        this.initializeLanguage();
        
        // Bind events
        this.bindEvents();
        
        // Update UI based on current settings
        this.updateUI();
    }
    
    initializeTheme() {
        // Get saved theme or default to 'light'
        const savedTheme = localStorage.getItem('theme');
        const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
        this.currentTheme = savedTheme || systemTheme;
        this.applyTheme(this.currentTheme);
    }
    
    initializeLanguage() {
        // Get saved language or default based on browser/page
        const savedLang = localStorage.getItem('language');
        const htmlLang = document.documentElement.lang || 'en';
        this.currentLang = savedLang || htmlLang;
        this.applyLanguage(this.currentLang);
    }
    
    bindEvents() {
        // Theme toggle button
        const themeToggle = document.getElementById('theme-toggle');
        if (themeToggle) {
            themeToggle.addEventListener('click', () => this.toggleTheme());
        }
        
        // Language toggle button
        const langToggle = document.getElementById('language-toggle');
        if (langToggle) {
            langToggle.addEventListener('click', () => this.toggleLanguage());
        }
        
        // Listen for system theme changes
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
            if (!localStorage.getItem('theme')) {
                this.applyTheme(e.matches ? 'dark' : 'light');
            }
        });
        
        // Sidebar toggle
        const sidebarToggle = document.querySelector('[data-widget="pushmenu"]');
        if (sidebarToggle) {
            sidebarToggle.addEventListener('click', () => this.toggleSidebar());
        }
    }
    
    toggleTheme() {
        const newTheme = this.currentTheme === 'light' ? 'dark' : 'light';
        this.applyTheme(newTheme);
        localStorage.setItem('theme', newTheme);
    }
    
    applyTheme(theme) {
        this.currentTheme = theme;
        
        if (theme === 'dark') {
            document.documentElement.classList.add('dark');
            document.body.setAttribute('data-theme', 'dark');
        } else {
            document.documentElement.classList.remove('dark');
            document.body.setAttribute('data-theme', 'light');
        }
        
        this.updateThemeIcon();
    }
    
    updateThemeIcon() {
        const themeToggle = document.getElementById('theme-toggle');
        if (themeToggle) {
            const icon = themeToggle.querySelector('i');
            if (icon) {
                icon.className = this.currentTheme === 'dark' ? 'fas fa-sun' : 'fas fa-moon';
            }
        }
    }
    
    toggleLanguage() {
        const newLang = this.currentLang === 'en' ? 'fa' : 'en';
        this.applyLanguage(newLang);
        localStorage.setItem('language', newLang);
        
        // Reload page to apply language changes
        window.location.reload();
    }
    
    applyLanguage(lang) {
        this.currentLang = lang;
        
        // Set document language and direction
        document.documentElement.lang = lang;
        
        if (lang === 'fa') {
            document.documentElement.dir = 'rtl';
            document.body.classList.add('rtl');
            document.body.classList.remove('ltr');
        } else {
            document.documentElement.dir = 'ltr';
            document.body.classList.add('ltr');
            document.body.classList.remove('rtl');
        }
        
        // Update font family for Persian
        if (lang === 'fa') {
            document.body.style.fontFamily = "'Vazirmatn', 'IranYekan', 'Tahoma', sans-serif";
        } else {
            document.body.style.fontFamily = "'Inter', 'system-ui', sans-serif";
        }
        
        this.updateLanguageIcon();
    }
    
    updateLanguageIcon() {
        const langToggle = document.getElementById('language-toggle');
        if (langToggle) {
            const text = langToggle.querySelector('span');
            if (text) {
                text.textContent = this.currentLang === 'fa' ? 'EN' : 'فا';
            }
        }
    }
    
    toggleSidebar() {
        const sidebar = document.querySelector('.sidebar-modern');
        const overlay = document.querySelector('.sidebar-overlay');
        
        if (sidebar) {
            sidebar.classList.toggle('collapsed');
            
            // Create/remove overlay for mobile
            if (window.innerWidth <= 768) {
                if (sidebar.classList.contains('collapsed')) {
                    if (overlay) overlay.remove();
                } else {
                    if (!overlay) {
                        const newOverlay = document.createElement('div');
                        newOverlay.className = 'sidebar-overlay fixed inset-0 bg-black bg-opacity-50 z-40 lg:hidden';
                        newOverlay.addEventListener('click', () => this.toggleSidebar());
                        document.body.appendChild(newOverlay);
                    }
                }
            }
        }
    }
    
    updateUI() {
        // Update icons and text based on current theme and language
        this.updateThemeIcon();
        this.updateLanguageIcon();
        
        // Add smooth transitions to all elements
        this.addTransitions();
    }
    
    addTransitions() {
        // Add transition classes to elements that need smooth theme transitions
        const elements = document.querySelectorAll('.card-modern, .btn-modern, .input-modern, .sidebar-modern');
        elements.forEach(el => {
            el.classList.add('transition-all', 'duration-300');
        });
    }
    
    // Utility methods for other scripts
    isRTL() {
        return this.currentLang === 'fa' || document.documentElement.dir === 'rtl';
    }
    
    isDark() {
        return this.currentTheme === 'dark';
    }
    
    getCurrentTheme() {
        return this.currentTheme;
    }
    
    getCurrentLanguage() {
        return this.currentLang;
    }
}

// Animation utilities
class AnimationManager {
    static fadeIn(element, duration = 300) {
        element.style.opacity = '0';
        element.style.transition = `opacity ${duration}ms ease-in-out`;
        
        requestAnimationFrame(() => {
            element.style.opacity = '1';
        });
    }
    
    static slideIn(element, direction = 'left', duration = 300) {
        const translateValue = direction === 'left' ? '-100%' : '100%';
        element.style.transform = `translateX(${translateValue})`;
        element.style.transition = `transform ${duration}ms ease-out`;
        
        requestAnimationFrame(() => {
            element.style.transform = 'translateX(0)';
        });
    }
    
    static scaleIn(element, duration = 200) {
        element.style.transform = 'scale(0.95)';
        element.style.opacity = '0';
        element.style.transition = `transform ${duration}ms ease-out, opacity ${duration}ms ease-out`;
        
        requestAnimationFrame(() => {
            element.style.transform = 'scale(1)';
            element.style.opacity = '1';
        });
    }
    
    static pulse(element, duration = 1000) {
        element.style.animation = `pulse ${duration}ms ease-in-out infinite`;
    }
    
    static removePulse(element) {
        element.style.animation = '';
    }
}

// Toast notification system
class ToastManager {
    constructor() {
        this.container = this.createContainer();
    }
    
    createContainer() {
        const container = document.createElement('div');
        container.className = 'fixed top-4 right-4 z-50 space-y-2 max-w-sm';
        container.id = 'toast-container';
        document.body.appendChild(container);
        return container;
    }
    
    show(message, type = 'info', duration = 3000) {
        const toast = document.createElement('div');
        toast.className = `
            toast-modern transform translate-x-full opacity-0 transition-all duration-300
            p-4 rounded-xl shadow-lg border-l-4 text-white font-medium
            ${this.getTypeClasses(type)}
        `;
        
        toast.innerHTML = `
            <div class="flex items-center justify-between">
                <div class="flex items-center">
                    <i class="${this.getTypeIcon(type)} mr-2"></i>
                    <span>${message}</span>
                </div>
                <button onclick="this.parentElement.parentElement.remove()" class="ml-4 text-white hover:text-gray-200">
                    <i class="fas fa-times"></i>
                </button>
            </div>
        `;
        
        this.container.appendChild(toast);
        
        // Trigger animation
        requestAnimationFrame(() => {
            toast.classList.remove('translate-x-full', 'opacity-0');
            toast.classList.add('translate-x-0', 'opacity-100');
        });
        
        // Auto remove
        if (duration > 0) {
            setTimeout(() => {
                this.remove(toast);
            }, duration);
        }
        
        return toast;
    }
    
    remove(toast) {
        toast.classList.add('translate-x-full', 'opacity-0');
        setTimeout(() => {
            if (toast.parentElement) {
                toast.parentElement.removeChild(toast);
            }
        }, 300);
    }
    
    getTypeClasses(type) {
        const classes = {
            success: 'bg-green-500 border-green-400',
            error: 'bg-red-500 border-red-400',
            warning: 'bg-yellow-500 border-yellow-400',
            info: 'bg-blue-500 border-blue-400'
        };
        return classes[type] || classes.info;
    }
    
    getTypeIcon(type) {
        const icons = {
            success: 'fas fa-check-circle',
            error: 'fas fa-exclamation-circle',
            warning: 'fas fa-exclamation-triangle',
            info: 'fas fa-info-circle'
        };
        return icons[type] || icons.info;
    }
}

// Modal management
class ModalManager {
    static show(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.remove('hidden');
            modal.classList.add('flex');
            
            // Animate in
            const content = modal.querySelector('.modal-content');
            if (content) {
                AnimationManager.scaleIn(content);
            }
            
            // Prevent body scroll
            document.body.style.overflow = 'hidden';
        }
    }
    
    static hide(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.add('hidden');
            modal.classList.remove('flex');
            
            // Restore body scroll
            document.body.style.overflow = '';
        }
    }
    
    static toggle(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            if (modal.classList.contains('hidden')) {
                this.show(modalId);
            } else {
                this.hide(modalId);
            }
        }
    }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    // Initialize theme and language manager
    window.themeManager = new ThemeLanguageManager();
    
    // Initialize toast manager
    window.toastManager = new ToastManager();
    
    // Initialize animation manager
    window.animationManager = AnimationManager;
    
    // Initialize modal manager
    window.modalManager = ModalManager;
    
    // Add global utility functions
    window.showToast = (message, type, duration) => window.toastManager.show(message, type, duration);
    window.showModal = (modalId) => window.modalManager.show(modalId);
    window.hideModal = (modalId) => window.modalManager.hide(modalId);
    
    // Apply initial animations to page elements
    const cards = document.querySelectorAll('.card-modern');
    cards.forEach((card, index) => {
        card.style.animationDelay = `${index * 0.1}s`;
        card.classList.add('animate-fade-in');
    });
    
    console.log('Theme and Language Manager initialized');
}); 