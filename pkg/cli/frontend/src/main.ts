/**
 * Main Application Entry Point
 * Initializes all sub-modules and exposes global handlers.
 */
import { CONFIG } from './config.ts';
import { logConsole, clearTerminalLogs } from './utils/logger.ts';
import { initTheme, toggleTheme } from './modules/theme.ts';
import { 
  initAuth, 
  openAuthModal, 
  closeAuthModal, 
  saveAuthModal, 
  logoutAuth, 
  generateDemoKeypair 
} from './modules/auth.ts';
import { 
  updatePeersDropdown, 
  executePeers,
} from './modules/peers.ts';
import { 
  initContractDropdowns, 
  handleContractChange, 
  handleFunctionChange, 
  executeSignAndSendContract, 
  ContractsABIMap
} from './modules/contracts.ts';

// UI Helpers
export function toggleOpblock(id: string): void {
  const el = document.getElementById(id);
  if (el) el.classList.toggle('is-open');
}

export function toggleTryOut(id: string): void {
  const el = document.getElementById(`try-${id}`);
  if (el) el.classList.toggle('active');
}

export function clearResponses(id: string): void {
  const el = document.getElementById(`responses-${id}`);
  if (el) el.style.display = 'none';
}

export function filterTag(tag: 'peers' | 'contracts'): void {
  const secPeers = document.getElementById('section-peers');
  const secContracts = document.getElementById('section-contracts');
  const tabPeers = document.getElementById('tab-peers');
  const tabContracts = document.getElementById('tab-contracts');

  if (tabPeers) tabPeers.classList.remove('active');
  if (tabContracts) tabContracts.classList.remove('active');

  if (tag === 'peers') {
    if (secPeers) secPeers.style.display = 'block';
    if (secContracts) secContracts.style.display = 'none';
    if (tabPeers) tabPeers.classList.add('active');
  } else if (tag === 'contracts') {
    if (secPeers) secPeers.style.display = 'none';
    if (secContracts) secContracts.style.display = 'block';
    if (tabContracts) tabContracts.classList.add('active');
  }
  const url = new URL(window.location.href);
  url.searchParams.set('tab', tag);
  window.history.pushState({}, '', url.toString());
}

async function fetchNodeConfig() {
    try {
        const response = await fetch('/api/config');
        const data = await response.json();
        
        window.KHOAI_TARGET_NODE = data.TargetNode;
        
        console.log("Retrieved the Target Node from Go:", data.TargetNode);
        

        window.contractAbi = data.contract
        initContractDropdowns();
        const targetElement = document.getElementById("target-node-display");
        if (targetElement) {
          targetElement.textContent = window.KHOAI_TARGET_NODE;
        }
        
    } catch (err) {
        console.error("Unable to retrieve configuration from the Go backend:", err);
    }
}

// Attach functions to global window for inline HTML onclick/onchange handlers
declare global {
  interface Window {
    toggleTheme: typeof toggleTheme;
    openAuthModal: typeof openAuthModal;
    closeAuthModal: typeof closeAuthModal;
    saveAuthModal: typeof saveAuthModal;
    logoutAuth: typeof logoutAuth;
    generateDemoKeypair: typeof generateDemoKeypair;
    toggleOpblock: typeof toggleOpblock;
    toggleTryOut: typeof toggleTryOut;
    clearResponses: typeof clearResponses;
    filterTag: typeof filterTag;
    executePeers: typeof executePeers;
    handleContractChange: typeof handleContractChange;
    handleFunctionChange: typeof handleFunctionChange;
    executeSignAndSendContract: typeof executeSignAndSendContract;
    clearTerminalLogs: typeof clearTerminalLogs;
    KHOAI_TARGET_NODE: string;
    contractAbi: ContractsABIMap;
  }
}

window.toggleTheme = toggleTheme;
window.openAuthModal = openAuthModal;
window.closeAuthModal = closeAuthModal;
window.saveAuthModal = saveAuthModal;
window.logoutAuth = logoutAuth;
window.generateDemoKeypair = generateDemoKeypair;
window.toggleOpblock = toggleOpblock;
window.toggleTryOut = toggleTryOut;
window.clearResponses = clearResponses;
window.filterTag = filterTag;
window.executePeers = executePeers;
window.handleContractChange = handleContractChange;
window.handleFunctionChange = handleFunctionChange;
window.executeSignAndSendContract = executeSignAndSendContract;
window.clearTerminalLogs = clearTerminalLogs;

// Bootstrap Application on DOM Ready
window.addEventListener('DOMContentLoaded', () => {
  initTheme();
  initAuth();
  fetchNodeConfig();
  updatePeersDropdown();

  const params = new URLSearchParams(window.location.search);
  const activeTab = (params.get('tab') as 'peers' | 'contracts') || 'peers';
  
  filterTag(activeTab);

  logConsole('info', `Khoai-Chain Swagger UI initialized. Target node: ${CONFIG.TARGET_NODE}`);
});
