/**
 * Ed25519 Authentication & Modal Management
 */
import { state } from '../state.ts';
import { logConsole } from '../utils/logger.ts';

export function initAuth(): void {
  const savedPub = localStorage.getItem('ed25519_pubkey');
  const savedPriv = localStorage.getItem('ed25519_privkey');
  if (savedPub && savedPriv) {
    state.auth.publicKey = savedPub;
    state.auth.privateKey = savedPriv;
    state.auth.isAuthorized = true;
    updateAuthUI();
  }
}

export function openAuthModal(): void {
  const modalPubkey = document.getElementById('modal-pubkey') as HTMLInputElement;
  const modalPrivkey = document.getElementById('modal-privkey') as HTMLInputElement;
  const authModal = document.getElementById('auth-modal');

  if (modalPubkey) modalPubkey.value = state.auth.publicKey;
  if (modalPrivkey) modalPrivkey.value = state.auth.privateKey;
  if (authModal) authModal.classList.remove('hidden');
}

export function closeAuthModal(): void {
  const authModal = document.getElementById('auth-modal');
  if (authModal) authModal.classList.add('hidden');
}

export function saveAuthModal(): void {
  const modalPubkey = document.getElementById('modal-pubkey') as HTMLInputElement;
  const modalPrivkey = document.getElementById('modal-privkey') as HTMLInputElement;

  const pub = modalPubkey ? modalPubkey.value.trim() : '';
  const priv = modalPrivkey ? modalPrivkey.value.trim() : '';

  if (!pub || !priv) {
    alert('Please provide both Ed25519 Public and Private keys.');
    return;
  }

  state.auth.publicKey = pub;
  state.auth.privateKey = priv;
  state.auth.isAuthorized = true;
  localStorage.setItem('ed25519_pubkey', pub);
  localStorage.setItem('ed25519_privkey', priv);

  updateAuthUI();
  closeAuthModal();
  logConsole('success', `Ed25519 Keypair authorized (${pub.slice(0, 10)}...${pub.slice(-6)})`);
}

export function logoutAuth(): void {
  state.auth.publicKey = '';
  state.auth.privateKey = '';
  state.auth.isAuthorized = false;
  localStorage.removeItem('ed25519_pubkey');
  localStorage.removeItem('ed25519_privkey');
  updateAuthUI();
  closeAuthModal();
  logConsole('info', 'Logged out and cleared authorization keys.');
}

export function updateAuthUI(): void {
  const authBtn = document.getElementById('auth-btn');
  const lockIcon = document.getElementById('auth-lock-icon');
  const btnText = document.getElementById('auth-btn-text');

  if (!authBtn || !lockIcon || !btnText) return;

  if (state.auth.isAuthorized) {
    authBtn.classList.add('authorized');
    lockIcon.textContent = '🔓';
    btnText.textContent = `Authorized (${state.auth.publicKey.slice(0, 8)}...)`;
  } else {
    authBtn.classList.remove('authorized');
    lockIcon.textContent = '🔒';
    btnText.textContent = 'Authorize';
  }
}

export function generateDemoKeypair(): void {
  const hexChars = '0123456789abcdef';
  let pub = '0x';
  let priv = '0x';
  for (let i = 0; i < 64; i++) pub += hexChars[Math.floor(Math.random() * 16)];
  for (let i = 0; i < 64; i++) priv += hexChars[Math.floor(Math.random() * 16)];

  const modalPubkey = document.getElementById('modal-pubkey') as HTMLInputElement;
  const modalPrivkey = document.getElementById('modal-privkey') as HTMLInputElement;

  if (modalPubkey) modalPubkey.value = pub;
  if (modalPrivkey) modalPrivkey.value = priv;
}
