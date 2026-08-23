/**
 * Theme Management Module (Light / Dark Mode)
 */
import { state } from '../state.ts';

export function initTheme(): void {
  const savedTheme = (localStorage.getItem('fastapi_theme') as 'light' | 'dark') || 'light';
  setTheme(savedTheme);
}

export function toggleTheme(): void {
  const nextTheme = state.theme === 'light' ? 'dark' : 'light';
  setTheme(nextTheme);
}

export function setTheme(t: 'light' | 'dark'): void {
  state.theme = t;
  document.documentElement.setAttribute('data-theme', t);
  localStorage.setItem('fastapi_theme', t);

  const themeIcon = document.getElementById('theme-icon');
  const themeText = document.getElementById('theme-text');
  if (themeIcon) themeIcon.textContent = t === 'light' ? '🌓' : '☀️';
  if (themeText) themeText.textContent = t === 'light' ? 'Dark Mode' : 'Light Mode';
}
