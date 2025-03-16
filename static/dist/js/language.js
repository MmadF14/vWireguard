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
            const translation = this.translate(key);
            
            if (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA') {
                // For input elements, update both placeholder and value if it matches the key
                if (element.placeholder === key) {
                    element.placeholder = translation;
                }
                if (element.value === key) {
                    element.value = translation;
                }
            } else {
                element.textContent = translation;
            }

            // Update title attribute if it exists and matches the key
            if (element.title === key) {
                element.title = translation;
            }
        });

        // Translate placeholders that match translation keys
        document.querySelectorAll('input[placeholder], textarea[placeholder]').forEach(element => {
            const placeholder = element.getAttribute('placeholder');
            if (this.translations[this.currentLang][placeholder]) {
                element.placeholder = this.translate(placeholder);
            }
        });

        // Translate titles that match translation keys
        document.querySelectorAll('[title]').forEach(element => {
            const title = element.getAttribute('title');
            if (this.translations[this.currentLang][title]) {
                element.title = this.translate(title);
            }
        });
    }

    setupLanguageToggle() {
        // Remove existing language toggle if any
        const existingToggle = document.getElementById('language-toggle');
        if (existingToggle) {
            existingToggle.remove();
        }

        // Create new language toggle button
        const langToggle = document.createElement('button');
        langToggle.id = 'language-toggle';
        langToggle.className = 'language-toggle';
        langToggle.innerHTML = `
            <span>${this.currentLang === 'fa' ? 'English' : 'فارسی'}</span>
            <i class="fas ${this.currentLang === 'fa' ? 'fa-language' : 'fa-language'}"></i>
        `;
        
        // Add click event
        langToggle.addEventListener('click', () => {
            this.currentLang = this.currentLang === 'fa' ? 'en' : 'fa';
            localStorage.setItem('language', this.currentLang);
            window.location.reload();
        });

        // Add to navbar if exists, otherwise add to body
        const navbar = document.querySelector('.navbar-nav.ml-auto');
        if (navbar) {
            const li = document.createElement('li');
            li.className = 'nav-item';
            li.appendChild(langToggle);
            navbar.insertBefore(li, navbar.firstChild);
        } else {
            document.body.appendChild(langToggle);
        }
    }
}

// Initialize language manager when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.langManager = new LanguageManager();
}); 