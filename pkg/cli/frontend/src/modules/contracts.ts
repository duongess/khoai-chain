/**
 * Contract Management Module
 * Handles ABI-driven dynamic parameter inputs, Ed25519 signing intercept, and POST /contract/call
 */
import { CONFIG } from '../config.ts';
import { state } from '../state.ts';
import { logConsole } from '../utils/logger.ts';
import { sendP2p, EXECUTE } from './network.ts'

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

// Hàm phụ: Biến chuỗi Hex thành mảng Uint8Array
function hexToUint8Array(hexString: string): Uint8Array<ArrayBuffer> {
  const cleanHex = hexString.startsWith('0x') ? hexString.slice(2) : hexString;
  const numBytes = cleanHex.length / 2;
  const byteArray = new Uint8Array(numBytes) as Uint8Array<ArrayBuffer>;
  for (let i = 0; i < numBytes; i++) {
    byteArray[i] = parseInt(cleanHex.substr(i * 2, 2), 16);
  }
  return byteArray;
}

// Hàm phụ: Biến mảng Uint8Array thành chuỗi Hex
function uint8ArrayToHex(byteArray: Uint8Array): string {
  return Array.from(byteArray, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

/**
 * Hàm ký tin nhắn dùng Web Crypto API đã bọc ASN.1 PKCS#8
 */
async function signMessageWebCrypto(privateKeyHex: string, messageObj: EXECUTE): Promise<string> {
  const encoder = new TextEncoder();
  
  // 1. ĐỒNG BỘ THỨ TỰ JSON CHUẨN VỚI GO STRUCT
  // JavaScript bảo lưu thứ tự chèn object. Ta phải tái tạo object đúng thứ tự struct của Go.
  const orderedMessage = {
    type: messageObj.type,
    sender: messageObj.sender,
    contract: messageObj.contract,
    function: messageObj.function,
    args: messageObj.args,
    nonce: messageObj.nonce,
    signature: "" // Go backend gán rỗng signature trước khi mã hóa
  };

  const messageString = JSON.stringify(orderedMessage);
  const dataBytes = encoder.encode(messageString) as Uint8Array<ArrayBuffer>;

  // 2. BỌC PRIVATE KEY THÔ THÀNH CHUẨN PKCS#8 DER
  const cleanHex = privateKeyHex.startsWith('0x') ? privateKeyHex.slice(2) : privateKeyHex;
  // Private Key của Go dài 64 bytes (128 hex chars), ta chỉ lấy 32 bytes đầu (seed) cho PKCS#8
  const seedHex = cleanHex.slice(0, 64); 
  
  // Header tĩnh của ASN.1 cho thuật toán Ed25519
  const pkcs8Prefix = "302e020100300506032b657004220420";
  const pkcs8Hex = pkcs8Prefix + seedHex;
  
  const pkcs8Bytes = hexToUint8Array(pkcs8Hex);
  const privateKeyBuffer = pkcs8Bytes.buffer.slice(pkcs8Bytes.byteOffset, pkcs8Bytes.byteOffset + pkcs8Bytes.byteLength);
  
  const privateKey = await window.crypto.subtle.importKey(
    "pkcs8", 
    privateKeyBuffer,
    { name: "Ed25519", namedCurve: "Ed25519" }, // Web Crypto API chuẩn
    true,
    ["sign"]
  );

  // 3. KÝ TÊN LÊN MẢNG BYTE DỮ LIỆU
  const signatureBuffer = await window.crypto.subtle.sign(
    { name: "Ed25519" },
    privateKey,
    dataBytes
  );

  return uint8ArrayToHex(new Uint8Array(signatureBuffer));
}

export async function executeSignAndSendContract(): Promise<void> {
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

  // Extract dynamic parameters into an args array
  const args: string[] = [];
  if (fn && fn.inputs) {
    fn.inputs.forEach((inp) => {
      const inputEl = document.getElementById(`dyn-param-${inp.name}`) as HTMLInputElement;
      if (inputEl) {
        args.push(inputEl.value);
      }
    });
  }

  // --- BƯỚC 1: LẤY NONCE TỪ SERVER ---
  let receivedNonce = "42"; 
  try {
    const nonceRes = await sendP2p({
      type: "NONCE",
      sender: state.auth.publicKey,
      contract: "",
      function: "",
      args: [],
      nonce: ""
    } as unknown as EXECUTE);
    
    
    receivedNonce = nonceRes.result;

    console.log(receivedNonce)
    
  } catch (err) {
    console.warn("Could not fetch dynamic nonce, falling back to default:", err);
  }

  const commandMessage: EXECUTE = {
    type: "EXECUTE",
    sender: state.auth.publicKey,
    contract: state.selectedContract,
    function: state.selectedFunction,
    args: args,
    nonce: receivedNonce,
  } as any;

  const mockSignature = '0x' + Array.from({ length: 128 }, () => Math.floor(Math.random() * 16).toString(16)).join('');
  const mockTxHash = '0x' + Array.from({ length: 64 }, () => Math.floor(Math.random() * 16).toString(16)).join('');

  const signature = await signMessageWebCrypto(state.auth.privateKey, commandMessage)
  commandMessage.signature = signature

  let serverResponseResult = "";
  try {
    const result = await sendP2p(commandMessage);
    serverResponseResult = result.result || JSON.stringify(result);
  } catch (err: any) {
    serverResponseResult = `Error: ${err.message}`;
  }

  const payloadBox = document.getElementById('payload-contract-call');
  const curlBox = document.getElementById('curl-contract-call');
  const bodyBox = document.getElementById('body-contract-call');
  const responsesWrapper = document.getElementById('responses-contract-call');

  if (payloadBox) payloadBox.textContent = JSON.stringify(commandMessage, null, 2);
  if (curlBox) {
    curlBox.textContent = `curl -X 'POST' '/api/p2p/message' \\\n  -H 'accept: application/json' \\\n  -H 'Content-Type: application/json' \\\n  -d '${JSON.stringify(commandMessage)}'`;
  }
  if (bodyBox) bodyBox.textContent = serverResponseResult;
  if (responsesWrapper) responsesWrapper.style.display = 'block';

  logConsole('tx', `POST /api/p2p/message -> ${state.selectedContract}::${state.selectedFunction}()`, mockTxHash);
}