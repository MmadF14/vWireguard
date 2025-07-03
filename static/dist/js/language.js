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
            // Add Vazirmatn font for Persian
            document.body.style.fontFamily = 'Vazirmatn, system-ui, -apple-system, sans-serif';
        } else {
            document.body.classList.add('ltr');
            document.body.classList.remove('rtl');
            // Reset to default font
            document.body.style.fontFamily = 'Inter, system-ui, -apple-system, sans-serif';
        }
    }

    translate(key) {
        if (!key) return '';
        return this.translations[this.currentLang][key] || key;
    }

    translatePage() {
        console.log('Translating page to:', this.currentLang);
        // First, translate all elements with data-translate attribute
        const elementsToTranslate = document.querySelectorAll('[data-translate]');
        console.log('Found', elementsToTranslate.length, 'elements to translate');
        
        elementsToTranslate.forEach(element => {
            const key = element.getAttribute('data-translate');
            if (!key) return;
            
            const translation = this.translate(key);
            
            if (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA') {
                // For input elements, update placeholder and value if they match the key
                if (element.placeholder === key) {
                    element.placeholder = translation;
                }
                if (element.value === key) {
                    element.value = translation;
                }
            } else if (element.tagName === 'OPTION') {
                // For select options, update both text and value if they match
                element.textContent = translation;
                if (element.value === key) {
                    element.value = translation;
                }
            } else {
                // For other elements, update text content
                // Only update if the current text is the key or empty
                if (element.textContent.trim() === key || element.textContent.trim() === '') {
                    element.textContent = translation;
                }
            }

            // Update title attribute if it exists and matches the key
            if (element.title === key) {
                element.title = translation;
            }
        });

        // Then translate all placeholders that match translation keys
        document.querySelectorAll('input[placeholder], textarea[placeholder]').forEach(element => {
            const placeholder = element.getAttribute('placeholder');
            if (placeholder && this.translations[this.currentLang][placeholder]) {
                element.placeholder = this.translate(placeholder);
            }
        });

        // Translate all titles that match translation keys
        document.querySelectorAll('[title]').forEach(element => {
            const title = element.getAttribute('title');
            if (title && this.translations[this.currentLang][title]) {
                element.title = this.translate(title);
            }
        });

        // Translate validation messages
        if (window.jQuery && jQuery.validator) {
            jQuery.extend(jQuery.validator.messages, {
                required: this.translate('This field is required'),
                email: this.translate('Please enter a valid email address'),
                number: this.translate('Please enter a valid number'),
                digits: this.translate('Please enter only digits'),
                equalTo: this.translate('Please enter the same value again'),
                maxlength: jQuery.validator.format(this.translate('Please enter no more than {0} characters')),
                minlength: jQuery.validator.format(this.translate('Please enter at least {0} characters')),
                range: jQuery.validator.format(this.translate('Please enter a value between {0} and {1}'))
            });
        }

        // Translate Select2 placeholders if Select2 is present
        if (window.jQuery && jQuery.fn.select2) {
            document.querySelectorAll('select').forEach(select => {
                const $select = jQuery(select);
                if ($select.data('select2')) {
                    const placeholder = $select.data('placeholder');
                    if (placeholder && this.translations[this.currentLang][placeholder]) {
                        $select.data('placeholder', this.translate(placeholder));
                        $select.select2(); // Reinitialize to update placeholder
                    }
                }
            });
        }
    }

    // Method to translate a specific element
    translateElement(element) {
        if (!element) return;
        
        const key = element.getAttribute('data-translate');
        if (!key) return;
        
        const translation = this.translate(key);
        
        if (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA') {
            if (element.placeholder === key) {
                element.placeholder = translation;
            }
            if (element.value === key) {
                element.value = translation;
            }
        } else if (element.tagName === 'OPTION') {
            element.textContent = translation;
            if (element.value === key) {
                element.value = translation;
            }
        } else {
            element.textContent = translation;
        }

        if (element.title === key) {
            element.title = translation;
        }
    }

    // Force retranslate the entire page
    forceRetranslate() {
        this.translatePage();
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
        langToggle.className = 'language-toggle btn btn-outline-light btn-sm';
        langToggle.innerHTML = `
            <span>${this.currentLang === 'fa' ? 'English' : 'فارسی'}</span>
            <i class="fas fa-language ml-1"></i>
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
    
    // Make applyTranslations available globally
    window.applyTranslations = function() {
        if (window.langManager) {
            window.langManager.translatePage();
        }
    };
    
    // Only retranslate when new content is added to the DOM
    const observer = new MutationObserver((mutations) => {
        let shouldRetranslate = false;
        mutations.forEach((mutation) => {
            if (mutation.type === 'childList' && mutation.addedNodes.length > 0) {
                mutation.addedNodes.forEach((node) => {
                    if (node.nodeType === Node.ELEMENT_NODE) {
                        if (node.hasAttribute && node.hasAttribute('data-translate')) {
                            shouldRetranslate = true;
                        }
                        if (node.querySelectorAll && node.querySelectorAll('[data-translate]').length > 0) {
                            shouldRetranslate = true;
                        }
                    }
                });
            }
        });
        
        if (shouldRetranslate && window.langManager) {
            setTimeout(() => window.langManager.forceRetranslate(), 100);
        }
    });
    
    observer.observe(document.body, {
        childList: true,
        subtree: true
    });
}); 