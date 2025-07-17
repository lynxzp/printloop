// File Processing Service - Client Side JavaScript

document.addEventListener('DOMContentLoaded', function() {
    initializeApp();
});

function initializeApp() {
    const form = document.getElementById('uploadForm');
    const fileInput = document.getElementById('file');
    const uploadArea = document.getElementById('uploadArea');
    const submitBtn = document.getElementById('submitBtn');
    const loading = document.getElementById('loading');
    const btnText = document.querySelector('.btn-text');

    // Form submission handling
    if (form) {
        form.addEventListener('submit', handleFormSubmit);
    }

    // File input change handling
    if (fileInput) {
        fileInput.addEventListener('change', handleFileSelect);
    }

    // Drag and drop functionality
    if (uploadArea) {
        setupDragAndDrop(uploadArea, fileInput);
    }

    // Add fade-in animation to container
    const container = document.querySelector('.container');
    if (container) {
        container.classList.add('fade-in');
    }

    // Add input validation and formatting
    setupInputValidation();
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

function handleFormSubmit(event) {
    event.preventDefault();

    const submitBtn = document.getElementById('submitBtn');
    const loading = document.getElementById('loading');
    const btnText = document.querySelector('.btn-text');

    if (submitBtn && loading && btnText) {
        submitBtn.disabled = true;
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

    // Add only enabled parameters
    const checkboxConfigs = [
        { checkboxId: 'iterations_checkbox', inputId: 'iterations', name: 'iterations' },
        { checkboxId: 'wait_temp_checkbox', inputId: 'wait_temp', name: 'wait_temp' },
        { checkboxId: 'wait_min_checkbox', inputId: 'wait_min', name: 'wait_min' }
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

            showSuccess('🎉 File processed and downloaded successfully!');
            resetSubmitButton();
            resetForm();
        })
        .catch(error => {
            console.error('Upload error:', error);
            showError('❌ Error processing file: ' + error.message);
            resetSubmitButton();
        });

    return false;
}

function resetForm() {
    const form = document.getElementById('uploadForm');
    const fileInfo = document.getElementById('fileInfo');

    if (form) {
        // Reset only file input, keep parameter values
        const fileInput = document.getElementById('file');
        if (fileInput) {
            fileInput.value = '';
        }
    }

    if (fileInfo) {
        fileInfo.style.display = 'none';
    }
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
        <button onclick="this.parentElement.remove()" style="float: right; background: none; border: none; color: white; cursor: pointer; font-size: 1.2rem; padding: 0; margin-left: 15px;">&times;</button>
    `;

    const container = document.querySelector('.container');
    const firstChild = container.firstElementChild;
    container.insertBefore(errorDiv, firstChild);

    setTimeout(() => {
        if (errorDiv.parentElement) {
            errorDiv.style.opacity = '0';
            errorDiv.style.transform = 'translateY(-20px)';
            setTimeout(() => errorDiv.remove(), 300);
        }
    }, 5000);
}

function resetSubmitButton() {
    const submitBtn = document.getElementById('submitBtn');
    const loading = document.getElementById('loading');
    const btnText = document.querySelector('.btn-text');

    if (submitBtn && loading && btnText) {
        submitBtn.disabled = false;
        btnText.style.display = 'inline';
        loading.style.display = 'none';
    }
}

function showSuccess(message) {
    const successDiv = document.createElement('div');
    successDiv.className = 'success-notification fade-in';
    successDiv.innerHTML = `
        <strong>Success:</strong> ${message}
        <button onclick="this.parentElement.remove()" style="float: right; background: none; border: none; color: white; cursor: pointer; font-size: 1.2rem; padding: 0; margin-left: 15px;">&times;</button>
    `;

    const container = document.querySelector('.container');
    const firstChild = container.firstElementChild;
    container.insertBefore(successDiv, firstChild);

    setTimeout(() => {
        if (successDiv.parentElement) {
            successDiv.style.opacity = '0';
            successDiv.style.transform = 'translateY(-20px)';
            setTimeout(() => successDiv.remove(), 300);
        }
    }, 5000);
}

// Enhanced keyboard shortcuts
document.addEventListener('keydown', function(event) {
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
        const form = document.getElementById('uploadForm');
        if (form) {
            form.requestSubmit();
        }
    }

    if (event.key === 'Escape') {
        const fileInput = document.getElementById('file');
        const fileInfo = document.getElementById('fileInfo');
        const uploadIcon = document.querySelector('.upload-icon');

        if (fileInput && fileInfo) {
            fileInput.value = '';
            fileInfo.style.display = 'none';

            if (uploadIcon) {
                uploadIcon.textContent = '📁';
                uploadIcon.style.color = '#667eea';
            }
        }
    }
});

// Add smooth scrolling for mobile
if ('scrollBehavior' in document.documentElement.style) {
    document.documentElement.style.scrollBehavior = 'smooth';
}