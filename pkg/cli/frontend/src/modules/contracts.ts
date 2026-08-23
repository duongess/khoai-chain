/**
 * Contract Management Module
 * Handles ABI-driven dynamic parameter inputs, Ed25519 signing intercept, and POST /contract/call
 */
import { CONFIG } from '../config.ts';
import { state } from '../state.ts';
import { logConsole } from '../utils/logger.ts';

export interface ContractInput {
  name: string;
  type: string;
  default?: string;
}

export interface ContractFunction {
  name: string;
  description: string;
  inputs: ContractInput[];
}

export interface ContractSpec {
  name: string;
  functions: ContractFunction[];
}

export type ContractsABIMap = Record<string, ContractSpec>;

// Helper function để lấy ABI map từ window hoặc fallback về object rỗng
function getContractAbis(): ContractsABIMap {
  return window.contractAbi || {};
}

export function initContractDropdowns(): void {
  const contractSelect = document.getElementById('contract-select') as HTMLSelectElement;
  if (!contractSelect) return;
  contractSelect.innerHTML = '';

  const abis = getContractAbis();
  const keys = Object.keys(abis);
  console.log(abis, keys)

  if (keys.length === 0) {
    const opt = document.createElement('option');
    opt.value = '';
    opt.textContent = 'No contracts available';
    contractSelect.appendChild(opt);
    return;
  }

  keys.forEach((key) => {
    const opt = document.createElement('option');
    opt.value = key;
    opt.textContent = abis[key].name || key;
    contractSelect.appendChild(opt);
  });

  handleContractChange();
}

export function handleContractChange(): void {
  const contractSelect = document.getElementById('contract-select') as HTMLSelectElement;
  if (!contractSelect) return;

  state.selectedContract = contractSelect.value;
  const abis = getContractAbis();
  const contract = abis[state.selectedContract];

  const fnSelect = document.getElementById('function-select') as HTMLSelectElement;
  if (!fnSelect) return;
  fnSelect.innerHTML = '';

  if (!contract || !contract.functions) return;

  contract.functions.forEach((fn) => {
    const opt = document.createElement('option');
    opt.value = fn.name;
    opt.textContent = `${fn.name}(...) - ${fn.description || ''}`;
    fnSelect.appendChild(opt);
  });

  handleFunctionChange();
}

export function handleFunctionChange(): void {
  const fnSelect = document.getElementById('function-select') as HTMLSelectElement;
  if (!fnSelect) return;

  state.selectedFunction = fnSelect.value;
  renderDynamicInputs();
}

export function renderDynamicInputs(): void {
  const tbody = document.getElementById('dynamic-contract-params-body');
  if (!tbody) return;
  tbody.innerHTML = '';

  const abis = getContractAbis();
  const contract = abis[state.selectedContract];
  if (!contract || !contract.functions) return;

  const fn = contract.functions.find((f: ContractFunction) => f.name === state.selectedFunction);

  if (!fn || !fn.inputs || fn.inputs.length === 0) {
    const tr = document.createElement('tr');
    tr.innerHTML = `<td colspan="2" style="color: var(--text-muted); font-style: italic;">Function takes no input parameters.</td>`;
    tbody.appendChild(tr);
    return;
  }

  fn.inputs.forEach((inp) => {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td>
        <div class="param-name">${inp.name} <span class="param-required">* required</span></div>
        <div class="param-type">${inp.type} (body)</div>
      </td>
      <td>
        <input 
          type="text" 
          id="dyn-param-${inp.name}" 
          class="form-control mono" 
          value="${inp.default || ''}" 
          placeholder="Enter ${inp.type} value..."
          required
        />
      </td>
    `;
    tbody.appendChild(tr);
  });
}

export function executeSignAndSendContract(): void {
  // Check authorization
  if (!state.auth.isAuthorized) {
    alert('Ed25519 Authorization is required before dispatching transactions! Please click "Authorize" at the top.');
    const authModal = document.getElementById('auth-modal');
    if (authModal) authModal.classList.remove('hidden');
    return;
  }

  const abis = getContractAbis();
  const contract = abis[state.selectedContract];
  if (!contract) return;

  const fn = contract.functions.find((f: ContractFunction) => f.name === state.selectedFunction);

  // Extract dynamic parameters
  const params: Record<string, string> = {};
  if (fn && fn.inputs) {
    fn.inputs.forEach((inp) => {
      const inputEl = document.getElementById(`dyn-param-${inp.name}`) as HTMLInputElement;
      if (inputEl) {
        params[inp.name] = inputEl.value;
      }
    });
  }

  const gasLimitInput = document.getElementById('input-gas-limit') as HTMLInputElement;
  const nonceInput = document.getElementById('input-nonce') as HTMLInputElement;

  const gasLimit = gasLimitInput ? parseInt(gasLimitInput.value, 10) : 250000;
  const nonce = nonceInput ? parseInt(nonceInput.value, 10) : 42;

  // Build Raw Transaction Payload
  const rawTx = {
    function_name: state.selectedFunction,
    parameters: params,
    gas_limit: gasLimit,
    nonce: nonce,
    sender_pubkey: state.auth.publicKey,
    timestamp: Math.floor(Date.now() / 1000)
  };

  // Simulated Ed25519 cryptographic signature digest
  const mockSignature = '0x' + Array.from({ length: 128 }, () => Math.floor(Math.random() * 16).toString(16)).join('');
  const mockTxHash = '0x' + Array.from({ length: 64 }, () => Math.floor(Math.random() * 16).toString(16)).join('');

  const signedPayload = {
    ...rawTx,
    signature: mockSignature,
    signature_algorithm: 'Ed25519'
  };

  const url = `${CONFIG.TARGET_NODE}/contract/call`;

  const receipt = {
    status: 'success',
    code: 200,
    tx_hash: mockTxHash,
    block_number: 4892109,
    function_invoked: state.selectedFunction,
    gas_used: Math.floor(gasLimit * 0.68),
    execution_output: {
      result: "0x0000000000000000000000000000000000000000000000000000000000000001",
      events_emitted: [
        {
          event: `${state.selectedFunction.toUpperCase()}_EXECUTED`,
          caller: state.auth.publicKey.slice(0, 20) + '...'
        }
      ]
    },
    signer: state.auth.publicKey
  };

  const payloadBox = document.getElementById('payload-contract-call');
  const curlBox = document.getElementById('curl-contract-call');
  const bodyBox = document.getElementById('body-contract-call');
  const responsesWrapper = document.getElementById('responses-contract-call');

  if (payloadBox) payloadBox.textContent = JSON.stringify(signedPayload, null, 2);
  if (curlBox) {
    curlBox.textContent = `curl -X 'POST' '${url}' \\\n  -H 'accept: application/json' \\\n  -H 'Content-Type: application/json' \\\n  -H 'X-Ed25519-Signature: ${mockSignature.slice(0, 32)}...' \\\n  -d '${JSON.stringify(signedPayload)}'`;
  }
  if (bodyBox) bodyBox.textContent = JSON.stringify(receipt, null, 2);
  if (responsesWrapper) responsesWrapper.style.display = 'block';

  logConsole('tx', `POST ${url} -> ${contract.name}::${state.selectedFunction}()`, mockTxHash);
}