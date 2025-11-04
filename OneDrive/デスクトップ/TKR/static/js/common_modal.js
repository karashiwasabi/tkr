let modalOverlay, modalTitle, modalInput, okButton, cancelButton;
let currentResolve = null;

function init() {
    modalOverlay = document.getElementById('input-modal-overlay');
    modalTitle = document.getElementById('input-modal-title');
    modalInput = document.getElementById('input-modal-input');
    okButton = document.getElementById('input-modal-ok');
    cancelButton = document.getElementById('input-modal-cancel');

    if (!modalOverlay) {
        console.error('Input modal not found in DOM');
        return;
    }

    okButton.addEventListener('click', () => {
        if (currentResolve) {
            currentResolve(modalInput.value);
            currentResolve = null;
        }
        hide();
    });

    cancelButton.addEventListener('click', () => {
        if (currentResolve) {
            currentResolve(null); // Resolve with null on cancel
            currentResolve = null;
        }
        hide();
    });

    modalInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            okButton.click();
        }
    });
}

function hide() {
    modalOverlay.classList.add('hidden');
}

export function showInputModal(title) {
    return new Promise((resolve) => {
        currentResolve = resolve;
        modalTitle.textContent = title;
        modalInput.value = '';
        modalOverlay.classList.remove('hidden');
        setTimeout(() => modalInput.focus(), 50);
    });
}

// Initialize on script load
init();
