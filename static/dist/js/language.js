class LanguageManager {
    constructor() {
        this.currentLang = localStorage.getItem('language') || 'en';
        this.translations = translations;
        this.init();
    }

    init() {
        this.updatePageDirection();
        this.translatePage();
        this.setupLanguageToggle();
    }

    updatePageDirection() {
        document.documentElement.dir = this.currentLang === 'fa' ? 'rtl' : 'ltr';
        document.documentElement.lang = this.currentLang;
        
        // Update body classes for RTL support
        if (this.currentLang === 'fa') {
            document.body.classList.add('rtl');
            document.body.classList.remove('ltr');
        } else {
            document.body.classList.add('ltr');
            document.body.classList.remove('rtl');
        }
    }

    translate(key) {
        return this.translations[this.currentLang][key] || key;
    }

    translatePage() {
        // Translate all elements with data-translate attribute
        document.querySelectorAll('[data-translate]').forEach(element => {
            const key = element.getAttribute('data-translate');
            if (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA') {
                element.placeholder = this.translate(key);
            } else {
                element.textContent = this.translate(key);
            }
        });

        // Update language toggle button text
        const langToggle = document.getElementById('language-toggle');
        if (langToggle) {
            langToggle.textContent = this.currentLang === 'fa' ? 'English' : 'فارسی';
        }
    }

    setupLanguageToggle() {
        const langToggle = document.createElement('button');
        langToggle.id = 'language-toggle';
        langToggle.className = 'language-toggle';
        langToggle.textContent = this.currentLang === 'fa' ? 'English' : 'فارسی';
        
        langToggle.addEventListener('click', () => {
            this.currentLang = this.currentLang === 'fa' ? 'en' : 'fa';
            localStorage.setItem('language', this.currentLang);
            this.init();
            window.location.reload();
        });

        // Add the button to the page if it doesn't exist
        if (!document.getElementById('language-toggle')) {
            document.body.appendChild(langToggle);
        }
    }
}

// Initialize language manager when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.langManager = new LanguageManager();
}); 