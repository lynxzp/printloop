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

    // Add fade-in animation to container
    const container = document.querySelector('.container');
    if (container) {
        container.classList.add('fade-in');
    }

    // Add input validation and formatting
    setupInputValidation();
    setupSelectValidation();
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
        { checkboxId: 'wait_temp_checkbox', inputId: 'wait_temp', name: 'wait_temp' },
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

    fetch('/upload', {
        method: 'POST',
        body: formData
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`Server error: ${response.status}`);
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
            showError('Error processing file: ' + error.message);
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
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error-notification fade-in';
    errorDiv.innerHTML = `
        <strong>Error:</strong> ${message}
        <button onclick="this.parentElement.remove()" style="float: right; background: none; border: none; color: white; cursor: pointer; font-size: 1.1rem; padding: 0; margin-left: 12px;">&times;</button>
    `;

    const container = document.querySelector('.container');
    const firstChild = container.firstElementChild;
    container.insertBefore(errorDiv, firstChild);

    // Scroll to top to make error visible
    window.scrollTo({
        top: 0,
        behavior: 'smooth'
    });

    setTimeout(() => {
        if (errorDiv.parentElement) {
            errorDiv.style.opacity = '0';
            errorDiv.style.transform = 'translateY(-15px)';
            setTimeout(() => errorDiv.remove(), 250);
        }
    }, 4000);
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

    // Show loading state
    const showTemplateBtn = document.getElementById('showTemplateBtn');
    const originalText = showTemplateBtn.textContent;
    showTemplateBtn.textContent = 'Loading...';
    showTemplateBtn.disabled = true;

    fetch(`/template?printer=${encodeURIComponent(printerName)}`)
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
}
