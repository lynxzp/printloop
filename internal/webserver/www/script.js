document.addEventListener('DOMContentLoaded', function() {
    initializeApp();
});

function initializeApp() {
    const form = document.getElementById('uploadForm');
    const fileInput = document.getElementById('file');
    const uploadArea = document.getElementById('uploadArea');
    const submitBtn = document.getElementById('submitBtn');
    const customSubmitBtn = document.getElementById('submitCustomBtn');
    const loading = document.getElementById('loading');
    const btnText = document.querySelector('.btn-text');
    const showTemplateBtn = document.getElementById('showTemplateBtn');
    const hideTemplateBtn = document.getElementById('hideTemplateBtn');
    const editParametersBtn = document.getElementById('editParametersBtn');

    // Documentation panel elements
    const docsPanel = document.getElementById('docsPanel');
    const closeDocs = document.getElementById('closeDocs');
    const mainContent = document.getElementById('mainContent');

    // Error panel elements
    const errorPanel = document.getElementById('errorPanel');
    const closeError = document.getElementById('closeError');

    // Form submission handling
    if (form) {
        form.addEventListener('submit', handleFormSubmit);
    }

    // Custom template button handling
    if (customSubmitBtn) {
        customSubmitBtn.addEventListener('click', handleCustomFormSubmit);
    }

    // File input change handling
    if (fileInput) {
        fileInput.addEventListener('change', handleFileSelect);
    }

    // Drag and drop functionality
    if (uploadArea) {
        setupDragAndDrop(uploadArea, fileInput);
    }

    // Template button handling
    if (showTemplateBtn) {
        showTemplateBtn.addEventListener('click', showTemplate);
    }

    if (hideTemplateBtn) {
        hideTemplateBtn.addEventListener('click', hideTemplate);
    }

    // Edit parameters button handling
    if (editParametersBtn) {
        editParametersBtn.addEventListener('click', toggleParameters);
    }

    // Documentation panel handling
    if (closeDocs) {
        closeDocs.addEventListener('click', closeDocsPanel);
    }

    // Error panel handling
    if (closeError) {
        closeError.addEventListener('click', closeErrorPanel);
    }

    // Add fade-in animation to container
    const container = document.querySelector('.container');
    if (container) {
        container.classList.add('fade-in');
    }

    // Add input validation and formatting
    setupInputValidation();
    setupSelectValidation();

    // Initialize hint system
    initializeHintSystem();
}

function setupInputValidation() {
    const numberInputs = document.querySelectorAll('input[type="number"]');

    numberInputs.forEach(input => {
        input.addEventListener('input', function() {
            const value = parseInt(this.value);
            const min = parseInt(this.min);
            const max = parseInt(this.max);

            if (value < min) this.value = min;
            if (value > max) this.value = max;

            // Add visual feedback for valid values
            if (value >= min && value <= max) {
                this.style.borderColor = '#00b894';
                this.style.boxShadow = '0 0 0 3px rgba(0, 184, 148, 0.1)';
            } else {
                this.style.borderColor = '#e74c3c';
                this.style.boxShadow = '0 0 0 3px rgba(231, 76, 60, 0.1)';
            }
        });

        input.addEventListener('blur', function() {
            this.style.borderColor = '#e9ecef';
            this.style.boxShadow = 'none';
        });
    });
}

function setupSelectValidation() {
    const selectInputs = document.querySelectorAll('.form-select');

    selectInputs.forEach(select => {
        select.addEventListener('change', function() {
            // Add visual feedback for selection
            if (this.value) {
                this.style.borderColor = '#00b894';
                this.style.boxShadow = '0 0 0 3px rgba(0, 184, 148, 0.1)';

                setTimeout(() => {
                    this.style.borderColor = '#e9ecef';
                    this.style.boxShadow = 'none';
                }, 800);
            }
        });

        select.addEventListener('focus', function() {
            this.style.borderColor = '#667eea';
            this.style.boxShadow = '0 0 0 3px rgba(102, 126, 234, 0.1)';
        });

        select.addEventListener('blur', function() {
            this.style.borderColor = '#e9ecef';
            this.style.boxShadow = 'none';
        });
    });
}

function handleFormSubmit(event) {
    event.preventDefault();
    submitForm(false); // false = use default template
    return false;
}

function handleCustomFormSubmit(event) {
    event.preventDefault();
    submitForm(true); // true = use custom template
    return false;
}

function submitForm(useCustomTemplate) {
    const submitBtn = document.getElementById('submitBtn');
    const customSubmitBtn = document.getElementById('submitCustomBtn');
    const loading = document.getElementById('loading');
    const customLoading = document.getElementById('customLoading');
    const btnText = document.querySelector('.btn-text');
    const customBtnText = customSubmitBtn.querySelector('.btn-text');

    // Disable both buttons and show appropriate loading state
    if (submitBtn && loading && btnText) {
        submitBtn.disabled = true;
    }
    if (customSubmitBtn && customLoading && customBtnText) {
        customSubmitBtn.disabled = true;
    }

    if (useCustomTemplate) {
        customBtnText.style.display = 'none';
        customLoading.style.display = 'inline-block';
    } else {
        btnText.style.display = 'none';
        loading.style.display = 'inline-block';
    }

    const form = document.getElementById('uploadForm');
    const formData = new FormData();

    // Add file
    const fileInput = document.getElementById('file');
    if (fileInput.files[0]) {
        formData.append('file', fileInput.files[0]);
    }

    if (useCustomTemplate) {
        const templateContent = document.getElementById('templateContent');
        if (templateContent && templateContent.value.trim()) {
            formData.append('custom_template', templateContent.value);
        } else {
            showError('Template content is empty. Please edit the template or use the standard processing button.');
            resetSubmitButtons();
            return;
        }
    }

    // Add only enabled parameters
    const checkboxConfigs = [
        { checkboxId: 'printer_checkbox', inputId: 'printer', name: 'printer' },
        { checkboxId: 'iterations_checkbox', inputId: 'iterations', name: 'iterations' },
        { checkboxId: 'waitBedCooldownTempCheckbox', inputId: 'waitBedCooldownTemp', name: 'waitBedCooldownTemp' },
        { checkboxId: 'wait_min_checkbox', inputId: 'wait_min', name: 'wait_min' },
        { checkboxId: 'extra_extrude_checkbox', inputId: 'extra_extrude', name: 'extra_extrude' }
    ];

    checkboxConfigs.forEach(config => {
        const checkbox = document.getElementById(config.checkboxId);
        const input = document.getElementById(config.inputId);

        if (checkbox && checkbox.checked && input) {
            formData.append(config.name, input.value);
        }
    });

    // Get current language from URL for error messages (only if explicitly set)
    const urlParams = new URLSearchParams(window.location.search);
    const currentLang = urlParams.get('lang');
    const uploadUrl = currentLang ? `./upload?lang=${encodeURIComponent(currentLang)}` : './upload';

    fetch(uploadUrl, {
        method: 'POST',
        body: formData
    })
        .then(response => {
            if (!response.ok) {
                // Try to parse error response as JSON
                return response.text().then(text => {
                    try {
                        const errorData = JSON.parse(text);
                        throw { structured: true, ...errorData };
                    } catch (parseError) {
                        // Fallback to simple error if JSON parsing fails
                        throw new Error(`Server error: ${response.status} - ${text}`);
                    }
                });
            }

            const disposition = response.headers.get('Content-Disposition');
            let filename = 'processed_file';
            if (disposition) {
                const matches = disposition.match(/filename="([^"]+)"/);
                if (matches) {
                    filename = matches[1];
                }
            }

            return response.blob().then(blob => ({ blob, filename }));
        })
        .then(({ blob, filename }) => {
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = filename;
            document.body.appendChild(a);
            a.click();

            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);

            const message = useCustomTemplate ?
                'File processed with custom template and downloaded successfully!' :
                'File processed and downloaded successfully!';
            showSuccess(message);
            resetSubmitButtons();
        })
        .catch(error => {
            console.error('Upload error:', error);
            
            // Check if this is our structured error
            if (error && typeof error === 'object' && error.structured && error.type) {
                // Handle structured error response
                showStructuredError(error);
            } else if (error && error.message && error.message.includes('{"type":')) {
                // Extract JSON from error message and parse it
                const jsonMatch = error.message.match(/\{.*\}/);
                if (jsonMatch) {
                    try {
                        const errorData = JSON.parse(jsonMatch[0]);
                        showStructuredError({ structured: true, ...errorData });
                    } catch (parseError) {
                        console.error('Failed to parse error JSON:', parseError);
                        showError('Error processing file: ' + error.message);
                    }
                } else {
                    showError('Error processing file: ' + error.message);
                }
            } else {
                // Handle simple error
                showError('Error processing file: ' + (error.message || 'Unknown error'));
            }
            resetSubmitButtons();
        });
}

function handleFileSelect(event) {
    const file = event.target.files[0];
    const fileInfo = document.getElementById('fileInfo');

    if (file && fileInfo) {
        const size = formatFileSize(file.size);

        fileInfo.innerHTML = `
            <strong>Selected file:</strong> ${file.name}<br>
            <strong>Size:</strong> ${size}
        `;
        fileInfo.style.display = 'block';
        fileInfo.classList.add('fade-in');
    }
}

function setupDragAndDrop(uploadArea, fileInput) {
    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        uploadArea.addEventListener(eventName, preventDefaults, false);
        document.body.addEventListener(eventName, preventDefaults, false);
    });

    ['dragenter', 'dragover'].forEach(eventName => {
        uploadArea.addEventListener(eventName, highlight, false);
    });

    ['dragleave', 'drop'].forEach(eventName => {
        uploadArea.addEventListener(eventName, unhighlight, false);
    });

    uploadArea.addEventListener('drop', handleDrop, false);

    uploadArea.addEventListener('click', (e) => {
        if (e.target.type !== 'file' && e.target.tagName !== 'BUTTON') {
            fileInput.click();
        }
    });

    function preventDefaults(e) {
        e.preventDefault();
        e.stopPropagation();
    }

    function highlight(e) {
        uploadArea.classList.add('dragover');
    }

    function unhighlight(e) {
        uploadArea.classList.remove('dragover');
    }

    function handleDrop(e) {
        const dt = e.dataTransfer;
        const files = dt.files;

        if (files.length > 0) {
            fileInput.files = files;
            handleFileSelect({ target: { files: files } });
        }
    }
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function showError(message) {
    const errorMessage = document.getElementById('errorMessage');
    if (!errorMessage) return;

    // Set the error message content
    errorMessage.innerHTML = `<strong>Error:</strong> ${message}`;

    // Open the error panel
    openErrorPanel();
}

function showStructuredError(errorData) {
    const errorMessage = document.getElementById('errorMessage');
    if (!errorMessage) return;

    // Build comprehensive error display
    let errorHtml = `
        <div class="error-header-info">
            <h3 class="error-type-title">${errorData.title || 'Processing Error'}</h3>
            <div class="error-type-badge error-type-${errorData.type || 'internal'}">${getErrorTypeBadge(errorData.type)}</div>
        </div>
        <div class="error-description">
            <p><strong>Description:</strong> ${errorData.description || 'An error occurred during processing.'}</p>
        </div>
    `;

    if (errorData.details) {
        errorHtml += `
            <div class="error-details">
                <p><strong>Technical Details:</strong></p>
                <div class="error-details-content">${escapeHtml(errorData.details)}</div>
            </div>
        `;
    }

    if (errorData.suggestions && errorData.suggestions.length > 0) {
        errorHtml += `
            <div class="error-suggestions">
                <p><strong>Suggestions:</strong></p>
                <ul class="error-suggestions-list">
        `;
        errorData.suggestions.forEach(suggestion => {
            errorHtml += `<li>${escapeHtml(suggestion)}</li>`;
        });
        errorHtml += `
                </ul>
            </div>
        `;
    }

    errorMessage.innerHTML = errorHtml;

    // Open the error panel
    openErrorPanel();
}

function getErrorTypeBadge(errorType) {
    const badges = {
        'file_processing': 'File Processing',
        'template': 'Template',
        'validation': 'Validation',
        'configuration': 'Configuration',
        'file_io': 'File I/O',
        'upload': 'Upload',
        'internal': 'Internal'
    };
    return badges[errorType] || 'Unknown';
}

function escapeHtml(unsafe) {
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

function resetSubmitButtons() {
    const submitBtn = document.getElementById('submitBtn');
    const customSubmitBtn = document.getElementById('submitCustomBtn');
    const loading = document.getElementById('loading');
    const customLoading = document.getElementById('customLoading');
    const btnText = document.querySelector('.btn-text');
    const customBtnText = customSubmitBtn?.querySelector('.btn-text');

    if (submitBtn && loading && btnText) {
        submitBtn.disabled = false;
        btnText.style.display = 'inline';
        loading.style.display = 'none';
    }

    if (customSubmitBtn && customLoading && customBtnText) {
        customSubmitBtn.disabled = false;
        customBtnText.style.display = 'inline';
        customLoading.style.display = 'none';
    }
}

function showSuccess(message) {
    const successDiv = document.createElement('div');
    successDiv.className = 'success-notification fade-in';
    successDiv.innerHTML = `
        <strong>Success:</strong> ${message}
        <button onclick="this.parentElement.remove()" style="float: right; background: none; border: none; color: white; cursor: pointer; font-size: 1.1rem; padding: 0; margin-left: 12px;">&times;</button>
    `;

    const container = document.querySelector('.container');
    const firstChild = container.firstElementChild;
    container.insertBefore(successDiv, firstChild);

    setTimeout(() => {
        if (successDiv.parentElement) {
            successDiv.style.opacity = '0';
            successDiv.style.transform = 'translateY(-15px)';
            setTimeout(() => successDiv.remove(), 250);
        }
    }, 4000);
}

// Add smooth scrolling for mobile
if ('scrollBehavior' in document.documentElement.style) {
    document.documentElement.style.scrollBehavior = 'smooth';
}

function showTemplate() {
    const printerSelect = document.getElementById('printer');
    const printerName = printerSelect.value;

    if (!printerName) {
        showError('Please select a printer first');
        return;
    }

    // Show documentation panel
    openDocsPanel();

    // Show loading state
    const showTemplateBtn = document.getElementById('showTemplateBtn');
    const originalText = showTemplateBtn.textContent;
    showTemplateBtn.textContent = 'Loading...';
    showTemplateBtn.disabled = true;

    fetch(`./template?printer=${encodeURIComponent(printerName)}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Failed to load template: ${response.status}`);
            }
            return response.text();
        })
        .then(template => {
            const templateContent = document.getElementById('templateContent');
            templateContent.value = template;
            templateContent.readOnly = false;
            templateContent.classList.add('editable');

            document.getElementById('templateSection').style.display = 'block';

            // Show custom template button
            const customSubmitBtn = document.getElementById('submitCustomBtn');
            if (customSubmitBtn) {
                customSubmitBtn.style.display = 'block';
            }

            // Reset button
            showTemplateBtn.textContent = originalText;
            showTemplateBtn.disabled = false;

            // Scroll to template section
            document.getElementById('templateSection').scrollIntoView({
                behavior: 'smooth',
                block: 'start'
            });
        })
        .catch(error => {
            console.error('Template error:', error);
            showError('Failed to load template: ' + error.message);

            // Reset button
            showTemplateBtn.textContent = originalText;
            showTemplateBtn.disabled = false;
        });
}

function hideTemplate() {
    const templateSection = document.getElementById('templateSection');
    const templateContent = document.getElementById('templateContent');
    const customSubmitBtn = document.getElementById('submitCustomBtn');

    templateSection.style.display = 'none';
    templateContent.value = '';
    templateContent.readOnly = false;
    templateContent.classList.remove('editable');

    // Hide custom template button
    if (customSubmitBtn) {
        customSubmitBtn.style.display = 'none';
    }

    // Close documentation panel
    closeDocsPanel();
}

function openDocsPanel() {
    const docsPanel = document.getElementById('docsPanel');
    const mainContent = document.getElementById('mainContent');

    docsPanel.classList.add('open');
    mainContent.classList.add('docs-open');
}

function closeDocsPanel() {
    const docsPanel = document.getElementById('docsPanel');
    const mainContent = document.getElementById('mainContent');

    docsPanel.classList.remove('open');
    mainContent.classList.remove('docs-open');
}

function openErrorPanel() {
    const errorPanel = document.getElementById('errorPanel');
    const mainContent = document.getElementById('mainContent');

    errorPanel.classList.add('open');
    mainContent.classList.add('error-open');
    document.body.classList.add('error-panel-open');
}

function closeErrorPanel() {
    const errorPanel = document.getElementById('errorPanel');
    const mainContent = document.getElementById('mainContent');

    errorPanel.classList.remove('open');
    mainContent.classList.remove('error-open');
    document.body.classList.remove('error-panel-open');
}

function toggleParameters() {
    const parametersContainer = document.getElementById('parametersContainer');
    const editBtn = document.getElementById('editParametersBtn');
    
    if (parametersContainer.style.display === 'none') {
        // Show parameters
        parametersContainer.style.display = 'block';
        editBtn.textContent = editBtn.getAttribute('data-hide-text');
        
        // Scroll to parameters section
        parametersContainer.scrollIntoView({
            behavior: 'smooth',
            block: 'start'
        });
    } else {
        // Hide parameters
        parametersContainer.style.display = 'none';
        editBtn.textContent = editBtn.getAttribute('data-edit-text');
    }
}

// Hint system functionality
function initializeHintSystem() {
    const hintIcons = document.querySelectorAll('.hint-icon');
    const hintPopup = document.getElementById('hintPopup');
    const hintPopupClose = document.getElementById('hintPopupClose');
    const hintPopupBody = document.getElementById('hintPopupBody');
    
    let showTimeout = null;
    let hideTimeout = null;

    // Add click and hover handlers to hint icons
    hintIcons.forEach(icon => {
        const hintKey = icon.getAttribute('data-hint');
        const label = icon.closest('label');
        
        // Click handler for icon
        icon.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            clearAllTimeouts();
            showHint(hintKey);
        });

        // Hover handlers for icon
        icon.addEventListener('mouseenter', function(e) {
            clearAllTimeouts();
            showTimeout = setTimeout(() => showHint(hintKey), 300);
        });

        icon.addEventListener('mouseleave', function(e) {
            clearAllTimeouts();
            hideTimeout = setTimeout(closeHint, 200);
        });

        // Hover handlers for label (if exists)
        if (label) {
            label.addEventListener('mouseenter', function(e) {
                clearAllTimeouts();
                showTimeout = setTimeout(() => showHint(hintKey), 300);
            });

            label.addEventListener('mouseleave', function(e) {
                clearAllTimeouts();
                hideTimeout = setTimeout(closeHint, 200);
            });
        }
    });

    // Close popup handlers
    if (hintPopupClose) {
        hintPopupClose.addEventListener('click', function() {
            clearAllTimeouts();
            closeHint();
        });
    }

    // Keep popup open when hovering over it
    if (hintPopup) {
        hintPopup.addEventListener('mouseenter', function() {
            clearAllTimeouts();
        });

        hintPopup.addEventListener('mouseleave', function() {
            clearAllTimeouts();
            hideTimeout = setTimeout(closeHint, 200);
        });

        // Close on background click (only if clicking the transparent background)
        hintPopup.addEventListener('click', function(e) {
            if (e.target === this) {
                clearAllTimeouts();
                closeHint();
            }
        });
    }

    // Close on Escape key
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && hintPopup && hintPopup.classList.contains('show')) {
            clearAllTimeouts();
            closeHint();
        }
    });

    function clearAllTimeouts() {
        clearTimeout(showTimeout);
        clearTimeout(hideTimeout);
    }

    function showHint(hintKey) {
        clearAllTimeouts();
        
        // Get current language from page's lang attribute, which matches server-side language detection
        const currentLang = document.documentElement.lang || 'en';
        
        // Fetch the hint content
        fetch(`./hint?key=${encodeURIComponent(hintKey)}&lang=${encodeURIComponent(currentLang)}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error(`Failed to load hint: ${response.status}`);
                }
                return response.text();
            })
            .then(hintText => {
                if (hintPopupBody) {
                    // Split hint text into paragraphs using double newline as separator
                    const paragraphs = hintText.split('\n\n').filter(p => p.trim().length > 0);
                    
                    if (paragraphs.length > 1) {
                        // Multiple paragraphs - format each as a separate <p> element
                        const paragraphHtml = paragraphs.map(p => `<p>${escapeHtml(p.trim())}</p>`).join('');
                        hintPopupBody.innerHTML = paragraphHtml;
                    } else {
                        // Single paragraph - display as before
                        hintPopupBody.innerHTML = `<p>${escapeHtml(hintText)}</p>`;
                    }
                }
                if (hintPopup) {
                    hintPopup.classList.add('show');
                }
            })
            .catch(error => {
                console.error('Hint error:', error);
                if (hintPopupBody) {
                    hintPopupBody.innerHTML = '<p>Unable to load hint information.</p>';
                }
                if (hintPopup) {
                    hintPopup.classList.add('show');
                }
            });
    }

    function closeHint() {
        clearAllTimeouts();
        if (hintPopup) {
            hintPopup.classList.remove('show');
        }
    }
}