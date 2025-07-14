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
}

function handleFormSubmit(event) {
    event.preventDefault(); // Prevent default form submission

    const submitBtn = document.getElementById('submitBtn');
    const loading = document.getElementById('loading');
    const btnText = document.querySelector('.btn-text');

    // Validate form first
    if (!validateForm()) {
        return false;
    }

    if (submitBtn && loading && btnText) {
        // Show loading state
        submitBtn.disabled = true;
        btnText.style.display = 'none';
        loading.style.display = 'inline-block';
    }

    // Submit form using FormData and fetch for better control
    const form = document.getElementById('uploadForm');
    const formData = new FormData(form);

    fetch('/upload', {
        method: 'POST',
        body: formData
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`Server error: ${response.status}`);
            }

            // Get filename from Content-Disposition header if available
            const disposition = response.headers.get('Content-Disposition');
            let filename = 'processed_file';
            if (disposition) {
                const matches = disposition.match(/filename="([^"]+)"/);
                if (matches) {
                    filename = matches[1];
                }
            }

            // Convert response to blob and trigger download
            return response.blob().then(blob => ({ blob, filename }));
        })
        .then(({ blob, filename }) => {
            // Create download link and trigger it
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = filename;
            document.body.appendChild(a);
            a.click();

            // Cleanup
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);

            // Show success message
            showSuccess('File processed and downloaded successfully!');
            resetSubmitButton();
            resetForm();
        })
        .catch(error => {
            console.error('Upload error:', error);
            showError('Error processing file: ' + error.message);
            resetSubmitButton();
        });

    return false;
}

function resetForm() {
    const form = document.getElementById('uploadForm');
    const fileInfo = document.getElementById('fileInfo');

    if (form) {
        form.reset();
    }

    if (fileInfo) {
        fileInfo.style.display = 'none';
    }

    // Clear saved form data except for operation and format preferences
    const savedData = localStorage.getItem('fileProcessorFormData');
    if (savedData) {
        try {
            const formData = JSON.parse(savedData);
            // Keep operation and format, but clear options
            formData.options = '';
            localStorage.setItem('fileProcessorFormData', JSON.stringify(formData));
        } catch (e) {
            localStorage.removeItem('fileProcessorFormData');
        }
    }
}

function validateForm() {
    const fileInput = document.getElementById('file');
    const operation = document.getElementById('operation');
    const format = document.getElementById('format');

    if (!fileInput.files.length) {
        showError('Please select a file');
        return false;
    }

    if (!operation.value) {
        showError('Please select an operation');
        return false;
    }

    if (!format.value) {
        showError('Please select output format');
        return false;
    }

    // Validate file type
    const file = fileInput.files[0];
    const allowedTypes = ['.gcode'];
    const fileExtension = '.' + file.name.split('.').pop().toLowerCase();

    if (!allowedTypes.includes(fileExtension)) {
        showError('Please select a valid gcode file');
        return false;
    }

    // Validate file size (10MB max)
    const maxSize = 10 * 1024 * 1024; // 10MB
    if (file.size > maxSize) {
        showError('File size must be less than 10MB');
        return false;
    }

    return true;
}

function handleFileSelect(event) {
    const file = event.target.files[0];
    const fileInfo = document.getElementById('fileInfo');

    if (file && fileInfo) {
        const size = formatFileSize(file.size);
        const type = file.type || 'Unknown';

        fileInfo.innerHTML = `
            <strong>Selected file:</strong> ${file.name}<br>
            <strong>Size:</strong> ${size}<br>
            <strong>Type:</strong> ${type}
        `;
        fileInfo.style.display = 'block';
        fileInfo.classList.add('fade-in');
    }
}

function setupDragAndDrop(uploadArea, fileInput) {
    // Prevent default drag behaviors
    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        uploadArea.addEventListener(eventName, preventDefaults, false);
        document.body.addEventListener(eventName, preventDefaults, false);
    });

    // Highlight drop area when item is dragged over it
    ['dragenter', 'dragover'].forEach(eventName => {
        uploadArea.addEventListener(eventName, highlight, false);
    });

    ['dragleave', 'drop'].forEach(eventName => {
        uploadArea.addEventListener(eventName, unhighlight, false);
    });

    // Handle dropped files
    uploadArea.addEventListener('drop', handleDrop, false);

    // Click to select file
    uploadArea.addEventListener('click', () => {
        fileInput.click();
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
    // Create error notification
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error-notification';
    errorDiv.innerHTML = `
        <strong>Error:</strong> ${message}
        <button onclick="this.parentElement.remove()" style="float: right; background: none; border: none; color: white; cursor: pointer;">&times;</button>
    `;

    // Add error styles
    errorDiv.style.cssText = `
        background: linear-gradient(135deg, #e74c3c, #c0392b);
        color: white;
        padding: 15px;
        margin-bottom: 20px;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(231, 76, 60, 0.3);
        animation: fadeIn 0.3s ease-out;
    `;

    // Insert at top of container
    const container = document.querySelector('.container');
    const firstChild = container.firstElementChild;
    container.insertBefore(errorDiv, firstChild);

    // Auto remove after 5 seconds
    setTimeout(() => {
        if (errorDiv.parentElement) {
            errorDiv.remove();
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

// Utility function to show success messages
function showSuccess(message) {
    const successDiv = document.createElement('div');
    successDiv.className = 'success-notification';
    successDiv.innerHTML = `
        <strong>Success:</strong> ${message}
        <button onclick="this.parentElement.remove()" style="float: right; background: none; border: none; color: white; cursor: pointer;">&times;</button>
    `;

    successDiv.style.cssText = `
        background: linear-gradient(135deg, #00b894, #00a085);
        color: white;
        padding: 15px;
        margin-bottom: 20px;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 184, 148, 0.3);
        animation: fadeIn 0.3s ease-out;
    `;

    const container = document.querySelector('.container');
    const firstChild = container.firstElementChild;
    container.insertBefore(successDiv, firstChild);

    setTimeout(() => {
        if (successDiv.parentElement) {
            successDiv.remove();
        }
    }, 5000);
}

// Add keyboard shortcuts
document.addEventListener('keydown', function(event) {
    // Ctrl/Cmd + Enter to submit form
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
        const form = document.getElementById('uploadForm');
        if (form) {
            form.requestSubmit();
        }
    }

    // Escape to clear file selection
    if (event.key === 'Escape') {
        const fileInput = document.getElementById('file');
        const fileInfo = document.getElementById('fileInfo');

        if (fileInput && fileInfo) {
            fileInput.value = '';
            fileInfo.style.display = 'none';
        }
    }
});

// Progress enhancement for file upload (if needed in future)
function createProgressBar() {
    const progressContainer = document.createElement('div');
    progressContainer.innerHTML = `
        <div style="background: #e9ecef; border-radius: 10px; overflow: hidden; margin: 10px 0;">
            <div id="progressBar" style="height: 20px; background: linear-gradient(135deg, #667eea, #764ba2); width: 0%; transition: width 0.3s ease;"></div>
        </div>
        <div style="text-align: center; font-size: 0.9rem; color: #6c757d;">
            <span id="progressText">Preparing upload...</span>
        </div>
    `;

    return progressContainer;
}

// Form auto-save to localStorage (for development convenience)
function saveFormData() {
    const form = document.getElementById('uploadForm');
    if (!form) return;

    const formData = {
        operation: document.getElementById('operation').value,
        format: document.getElementById('format').value,
        options: document.getElementById('options').value
    };

    localStorage.setItem('fileProcessorFormData', JSON.stringify(formData));
}

function loadFormData() {
    const savedData = localStorage.getItem('fileProcessorFormData');
    if (!savedData) return;

    try {
        const formData = JSON.parse(savedData);

        if (formData.operation) {
            document.getElementById('operation').value = formData.operation;
        }
        if (formData.format) {
            document.getElementById('format').value = formData.format;
        }
        if (formData.options) {
            document.getElementById('options').value = formData.options;
        }
    } catch (e) {
        console.warn('Failed to load saved form data:', e);
    }
}

// Save form data on change
document.addEventListener('change', function(event) {
    if (event.target.matches('#operation, #format, #options')) {
        saveFormData();
    }
});

// Load saved form data on page load
document.addEventListener('DOMContentLoaded', function() {
    loadFormData();
});