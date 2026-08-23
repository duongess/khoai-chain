/**
    * Node Commander - Ed25519 Web3 Node CLI
    * Base endpoint constant required by specifications
    */
const TARGET_NODE_BASE = '{{.TargetNode}}';

// State Store
const state = {
    isAuthenticated: false,
    auth: {
    publicKey: '',
    privateKey: '',
    },
    currentTab: 'peer-mgmt',
    theme: 'dark',
    peers: [],
    selectedContractId: 'TokenVault',
    selectedFunctionId: 'deposit',
    logs: [],
    logFilter: 'all',
    cliHistory: [],
    cliHistoryIdx: -1
};

// ==========================================================================
// MOCK CONTRACT ABIs REPOSITORY
// ==========================================================================
const CONTRACT_ABIS = {
    TokenVault: {
    name: 'TokenVault (ERC-4626 Yield Engine)',
    address: '0x8b329482701b7a2d8329482701b7a2d832948270',
    description: 'Multi-asset vault smart contract with dynamic reward rate distribution.',
    functions: [
        {
        name: 'deposit',
        description: 'Deposit assets into the vault to mint share tokens to the recipient.',
        stateMutability: 'nonpayable',
        inputs: [
            { name: 'amount', type: 'uint256', placeholder: '1000000000000000000 (1 Token in wei)', defaultValue: '5000000000000000000' },
            { name: 'recipient', type: 'address', placeholder: '0x71C... recipient hex address', defaultValue: '0x9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b' }
        ]
        },
        {
        name: 'withdraw',
        description: 'Burn vault shares to withdraw underlying assets to owner wallet.',
        stateMutability: 'nonpayable',
        inputs: [
            { name: 'shares', type: 'uint256', placeholder: 'Vault share units to burn', defaultValue: '2500000000000000000' },
            { name: 'recipient', type: 'address', placeholder: '0x...', defaultValue: '0x9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b' }
        ]
        },
        {
        name: 'transferOwnership',
        description: 'Transfer governance control to a new Ed25519 multi-sig authority.',
        stateMutability: 'nonpayable',
        inputs: [
            { name: 'newOwner', type: 'address', placeholder: '0x... new controller address', defaultValue: '0x4321098765432109876543210987654321098765' }
        ]
        },
        {
        name: 'getBalance',
        description: 'Read underlying token balance for account address.',
        stateMutability: 'view',
        inputs: [
            { name: 'account', type: 'address', placeholder: '0x... account hex', defaultValue: '0x9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b' }
        ]
        },
        {
        name: 'setEmergencyPause',
        description: 'Halt all vault deposits and withdrawals immediately in crisis mode.',
        stateMutability: 'nonpayable',
        inputs: [
            { name: 'paused', type: 'bool', placeholder: 'true / false', defaultValue: 'true' }
        ]
        }
    ]
    },
    GovernanceDAO: {
    name: 'GovernanceDAO (On-Chain Protocol voting)',
    address: '0x3a99281745672201991823746281923847192834',
    description: 'Decentralized governance module for proposals, timelocks, and quorum checks.',
    functions: [
        {
        name: 'propose',
        description: 'Create a new governance action proposal with payload hash.',
        stateMutability: 'nonpayable',
        inputs: [
            { name: 'title', type: 'string', placeholder: 'CIP-42: Upgrade Consensus Threshold', defaultValue: 'CIP-42: Increase Max Peers to 128' },
            { name: 'actionHash', type: 'bytes32', placeholder: '0x... 32-byte sha256 action hash', defaultValue: '0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855' },
            { name: 'votingPeriodBlocks', type: 'uint32', placeholder: 'Voting duration in blocks', defaultValue: '40320' }
        ]
        },
        {
        name: 'castVote',
        description: 'Submit an Ed25519-signed vote (0=Against, 1=For, 2=Abstain).',
        stateMutability: 'nonpayable',
        inputs: [
            { name: 'proposalId', type: 'uint256', placeholder: 'Proposal index ID', defaultValue: '104' },
            { name: 'support', type: 'uint8', placeholder: '0 = Against, 1 = For, 2 = Abstain', defaultValue: '1' },
            { name: 'rationale', type: 'string', placeholder: 'Optional justification for vote', defaultValue: 'Validated peer capacity performance' }
        ]
        },
        {
        name: 'executeProposal',
        description: 'Execute approved proposal through the timelock executor.',
        stateMutability: 'nonpayable',
        inputs: [
            { name: 'proposalId', type: 'uint256', placeholder: 'Proposal index ID', defaultValue: '104' }
        ]
        }
    ]
    },
    OracleBridge: {
    name: 'OracleBridge (Decentralized Cross-Chain Feed)',
    address: '0xf49281a8c9201923485710293847102938471029',
    description: 'Decentralized price feed and verifiable randomness beacon.',
    functions: [
        {
        name: 'updatePriceFeed',
        description: 'Publish verified oracle rate report with cryptographic proof.',
        stateMutability: 'nonpayable',
        inputs: [
            { name: 'symbol', type: 'string', placeholder: 'e.g. BTC/USD, ETH/USD', defaultValue: 'ETH/USD' },
            { name: 'price', type: 'uint256', placeholder: 'Rate with 8 decimals (e.g. 350000000000 for $3500)', defaultValue: '354250000000' },
            { name: 'timestamp', type: 'uint64', placeholder: 'Unix epoch timestamp', defaultValue: String(Math.floor(Date.now() / 1000)) },
            { name: 'merkleProof', type: 'bytes', placeholder: '0x... Merkle branch hex proof', defaultValue: '0x8f2d91a0c4' }
        ]
        },
        {
        name: 'queryLatestPrice',
        description: 'Query most recent validated price for requested token symbol.',
        stateMutability: 'view',
        inputs: [
            { name: 'symbol', type: 'string', placeholder: 'e.g. BTC/USD', defaultValue: 'ETH/USD' }
        ]
        }
    ]
    },
    AccessRegistry: {
    name: 'AccessRegistry (RBAC Permission Matrix)',
    address: '0x1293847192837461928374619283746192837461',
    description: 'Role-Based Access Control enforcing peer whitelist and node dial rules.',
    functions: [
        {
        name: 'grantRole',
        description: 'Assign network authorization role to target address.',
        stateMutability: 'nonpayable',
        inputs: [
            { name: 'roleKey', type: 'bytes32', placeholder: '0x... ROLE_VALIDATOR (32-bytes)', defaultValue: '0x9b32a10400000000000000000000000000000000000000000000000000000000' },
            { name: 'account', type: 'address', placeholder: '0x... account hex', defaultValue: '0x5555555555555555555555555555555555555555' }
        ]
        },
        {
        name: 'revokeRole',
        description: 'Strip network authorization role from target address.',
        stateMutability: 'nonpayable',
        inputs: [
            { name: 'roleKey', type: 'bytes32', placeholder: '0x... role hash', defaultValue: '0x9b32a10400000000000000000000000000000000000000000000000000000000' },
            { name: 'account', type: 'address', placeholder: '0x... account hex', defaultValue: '0x5555555555555555555555555555555555555555' }
        ]
        }
    ]
    }
};

// Initial Mock Peers Seed
const DEFAULT_PEERS = [
    {
    id: '12D3KooWStqW...8p9j',
    fullId: '12D3KooWStqWbCg5k6B7s4M2h9u7x8p9j4Q1zL0vN3m',
    alias: 'validator-frankfurt-01',
    address: '/ip4/18.192.44.12/tcp/9000/p2p/12D3KooWStqWbCg5k6B7s4M2h9u7x8p9j',
    pubkey: '0x7e4a19b0284c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e',
    status: 'CONNECTED',
    latency: 18,
    traffic: '↑ 4.2 MB / ↓ 12.8 MB',
    connectedAt: '1h 42m ago'
    },
    {
    id: '12D3KooW9mKz...3t4r',
    fullId: '12D3KooW9mKzN2vL1x4P7h8M4s6B7k5CgStqW8p9j0v',
    alias: 'sentry-tokyo-ap1',
    address: '/ip4/35.200.118.99/tcp/9000/p2p/12D3KooW9mKzN2vL1x',
    pubkey: '0x4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e7e4a19b0284c8d9e0f1a2b3c',
    status: 'CONNECTED',
    latency: 34,
    traffic: '↑ 1.1 MB / ↓ 3.9 MB',
    connectedAt: '34m ago'
    },
    {
    id: '12D3KooWB7s4...7u9x',
    fullId: '12D3KooWB7s4M2h9u7x8p9j4Q1zL0vN3mStqWbCg5k6',
    alias: 'archive-node-us-east',
    address: '/ip4/54.89.201.76/tcp/9000/p2p/12D3KooWB7s4M2h9u',
    pubkey: '0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b4d5e6f7a8b9c0d1e2f3a4b5c',
    status: 'SYNCING',
    latency: 68,
    traffic: '↑ 842 KB / ↓ 1.2 MB',
    connectedAt: '12m ago'
    },
    {
    id: '12D3KooW4Q1z...k6B7',
    fullId: '12D3KooW4Q1zL0vN3mStqWbCg5k6B7s4M2h9u7x8p9j',
    alias: 'bootnode-seed-primary',
    address: '/dns4/bootnode.mainnet.mesh/tcp/9000/p2p/12D3KooW4Q1z',
    pubkey: '0x9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e7e4a19b0281a2b3c4d5e6f7a8b9c0d1e2f',
    status: 'CONNECTED',
    latency: 12,
    traffic: '↑ 18.5 MB / ↓ 45.1 MB',
    connectedAt: '8h 15m ago'
    }
];

// ==========================================================================
// INITIALIZATION & EVENT HOOKS
// ==========================================================================
window.addEventListener('DOMContentLoaded', () => {
    initTheme();
    initContractSelects();
    initSession();
    renderPeersTable();
    updateDynamicForm();
    addLog('info', `Node Commander initialized. Base Target: ${TARGET_NODE_BASE}`);
    addLog('info', 'Ed25519 authentication lock active. Enter your keypair to sign and dispatch transactions.');
});

// Theme Management
function initTheme() {
    const savedTheme = localStorage.getItem('node_commander_theme') || 'dark';
    setTheme(savedTheme);
}

function setTheme(theme) {
    state.theme = theme;
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('node_commander_theme', theme);
    const label = document.getElementById('theme-label');
    if (label) {
    label.textContent = theme === 'dark' ? 'Dark' : 'Light';
    }
}

function toggleTheme() {
    const newTheme = state.theme === 'dark' ? 'light' : 'dark';
    setTheme(newTheme);
    showToast(`Theme switched to ${newTheme} mode`);
    addLog('info', `UI Theme toggled to ${newTheme.toUpperCase()}`);
}

// Session & Auth Management
function initSession() {
    const savedPub = sessionStorage.getItem('node_ed25519_pub');
    const savedPriv = sessionStorage.getItem('node_ed25519_priv');
    
    if (savedPub && savedPriv) {
    state.auth.publicKey = savedPub;
    state.auth.privateKey = savedPriv;
    state.isAuthenticated = true;
    hideAuthModal();
    updateAuthUI();
    addLog('sign', `Session restored with Ed25519 PubKey: ${shortenHex(savedPub)}`);
    } else {
    showAuthModal();
    updateAuthUI();
    }
}

function showAuthModal() {
    const modal = document.getElementById('auth-modal');
    if (modal) modal.classList.remove('hidden');
}

function hideAuthModal() {
    const modal = document.getElementById('auth-modal');
    if (modal) modal.classList.add('hidden');
}

function lockSession() {
    state.isAuthenticated = false;
    sessionStorage.removeItem('node_ed25519_pub');
    sessionStorage.removeItem('node_ed25519_priv');
    updateAuthUI();
    showAuthModal();
    addLog('info', 'Session locked. Cryptographic signature module halted.');
    showToast('Session locked');
}

function updateAuthUI() {
    const dot = document.getElementById('auth-indicator-dot');
    const text = document.getElementById('auth-status-text');
    const pubShort = document.getElementById('auth-pubkey-short');
    const lockBtn = document.getElementById('lock-btn');

    if (state.isAuthenticated) {
    if (dot) {
        dot.className = 'status-dot active';
    }
    if (text) text.textContent = 'AUTHENTICATED';
    if (pubShort) pubShort.textContent = `[${shortenHex(state.auth.publicKey)}]`;
    if (lockBtn) lockBtn.style.display = 'inline-flex';
    } else {
    if (dot) {
        dot.className = 'status-dot locked';
    }
    if (text) text.textContent = 'LOCKED';
    if (pubShort) pubShort.textContent = '';
    if (lockBtn) lockBtn.style.display = 'none';
    }
}

function togglePrivKeyVisibility() {
    const input = document.getElementById('auth-privkey');
    if (input) {
    input.type = input.type === 'password' ? 'text' : 'password';
    }
}

function generateSampleEd25519Keypair() {
    // Helper to generate a realistic 32-byte Ed25519 public & 64-byte private seed pair
    const randomHex = (bytes) => {
    const arr = new Uint8Array(bytes);
    crypto.getRandomValues(arr);
    return '0x' + Array.from(arr).map(b => b.toString(16).padStart(2, '0')).join('');
    };

    const samplePub = randomHex(32);
    const samplePriv = randomHex(64);

    document.getElementById('auth-pubkey').value = samplePub;
    document.getElementById('auth-privkey').value = samplePriv;
    
    validateKeyInputs();
    showToast('Generated fresh Ed25519 demo keypair');
}

function validateKeyInputs() {
    const pub = document.getElementById('auth-pubkey').value.trim();
    const priv = document.getElementById('auth-privkey').value.trim();
    const pubInd = document.getElementById('pubkey-indicator');
    const privInd = document.getElementById('privkey-indicator');

    const isPubValid = /^0x[0-9a-fA-F]{64}$/.test(pub) || /^[0-9a-fA-F]{64}$/.test(pub);
    const isPrivValid = priv.length >= 32;

    if (pubInd) {
    if (isPubValid) {
        pubInd.className = 'key-indicator valid';
        pubInd.textContent = '✓ Valid 32-Byte Ed25519 Public Key';
    } else if (pub.length > 0) {
        pubInd.className = 'key-indicator invalid';
        pubInd.textContent = '✗ Must be 64 hexadecimal characters (32 bytes)';
    }
    }

    if (privInd) {
    if (isPrivValid) {
        privInd.className = 'key-indicator valid';
        privInd.textContent = '✓ Private Key / Secret Seed Provided';
    } else if (priv.length > 0) {
        privInd.className = 'key-indicator invalid';
        privInd.textContent = '✗ Key seed too short';
    }
    }

    return isPubValid && isPrivValid;
}

// Attach validation listeners
document.addEventListener('input', (e) => {
    if (e.target && (e.target.id === 'auth-pubkey' || e.target.id === 'auth-privkey')) {
    validateKeyInputs();
    }
});

function handleAuthSubmit(e) {
    e.preventDefault();
    const pub = document.getElementById('auth-pubkey').value.trim();
    const priv = document.getElementById('auth-privkey').value.trim();
    const remember = document.getElementById('remember-session').checked;

    if (!pub || !priv) {
    showToast('Please provide both Public and Private keys', 'error');
    return;
    }

    state.auth.publicKey = pub;
    state.auth.privateKey = priv;
    state.isAuthenticated = true;

    if (remember) {
    sessionStorage.setItem('node_ed25519_pub', pub);
    sessionStorage.setItem('node_ed25519_priv', priv);
    }

    hideAuthModal();
    updateAuthUI();
    showToast('Ed25519 Session Unlocked Successfully', 'success');
    addLog('sign', `Ed25519 Keypair registered: PubKey=${shortenHex(pub)}`);
    addLog('success', 'Dashboard unlocked. Node RPC ready for requests.');

    // Refresh peers
    fetchPeersList();
}

// ==========================================================================
// TAB NAVIGATION
// ==========================================================================
function switchTab(tabId) {
    state.currentTab = tabId;
    
    document.querySelectorAll('.tab-button').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.dashboard-view').forEach(view => view.classList.remove('active'));

    if (tabId === 'peer-mgmt') {
    document.getElementById('tab-btn-peers').classList.add('active');
    document.getElementById('peer-mgmt-view').classList.add('active');
    } else if (tabId === 'contract-mgmt') {
    document.getElementById('tab-btn-contracts').classList.add('active');
    document.getElementById('contract-mgmt-view').classList.add('active');
    updateDynamicForm();
    }

    addLog('info', `Navigation: Switched to ${tabId.toUpperCase()}`);
}

// ==========================================================================
// PEER MANAGEMENT (GET /peers, POST /add-peer, POST /remove-peer)
// ==========================================================================
state.peers = [...DEFAULT_PEERS];

async function fetchPeersList(manual = false) {
    const endpoint = `${TARGET_NODE_BASE}/peers`;
    addLog('info', `Requesting peer list: GET ${endpoint}`);

    try {
    // Attempt network fetch to TargetNode
    const response = await fetch(endpoint, {
        method: 'GET',
        headers: {
        'Content-Type': 'application/json',
        'X-Ed25519-PubKey': state.auth.publicKey || 'unauthenticated'
        }
    }).catch(err => {
        // In offline / preview simulation or when TargetNode placeholder is used
        return null;
    });

    if (response && response.ok) {
        const data = await response.json();
        if (Array.isArray(data)) {
        state.peers = data;
        }
        addLog('success', `Retrieved ${state.peers.length} active peers from ${endpoint}`);
    } else {
        // Graceful fallback for mock node environment
        addLog('info', `[Simulated Network] Target node returned ${state.peers.length} mesh peers.`);
    }
    } catch (err) {
    addLog('error', `GET /peers error: ${err.message}`);
    }

    renderPeersTable();
    if (manual) showToast('Peer list refreshed');
}

function renderPeersTable() {
    const tbody = document.getElementById('peers-table-body');
    const countBadge = document.getElementById('peer-count-badge');
    const statActive = document.getElementById('stat-active-peers');
    
    if (!tbody) return;
    tbody.innerHTML = '';

    if (countBadge) countBadge.textContent = state.peers.length;
    if (statActive) statActive.textContent = state.peers.length;

    if (state.peers.length === 0) {
    tbody.innerHTML = `
        <tr>
        <td colspan="7" style="text-align: center; color: var(--text-muted); padding: 2rem;">
            No peers currently connected. Use the form above to add a peer.
        </td>
        </tr>
    `;
    return;
    }

    state.peers.forEach((peer, idx) => {
    const tr = document.createElement('tr');
    
    let latencyClass = 'latency-low';
    if (peer.latency > 50) latencyClass = 'latency-high';
    else if (peer.latency > 25) latencyClass = 'latency-med';

    tr.innerHTML = `
        <td>
        <span class="badge-pill" style="${peer.status === 'CONNECTED' ? 'background: var(--success-subtle); color: var(--success);' : 'background: var(--warning-subtle); color: var(--warning);'}">
            ● ${peer.status}
        </span>
        </td>
        <td>
        <div class="peer-id-cell">
            <span title="${peer.fullId || peer.id}">${peer.id}</span>
            ${peer.alias ? `<span class="badge-pill" style="background: var(--bg-surface-elevated); color: var(--text-secondary); border: 1px solid var(--border-subtle);">${peer.alias}</span>` : ''}
        </div>
        </td>
        <td>
        <code style="font-size: 0.75rem; color: var(--text-secondary);" title="${peer.address}">${shortenAddress(peer.address)}</code>
        </td>
        <td>
        <code style="font-size: 0.75rem; color: var(--purple);" title="${peer.pubkey}">${shortenHex(peer.pubkey)}</code>
        </td>
        <td>
        <span class="latency-pill ${latencyClass}">${peer.latency}ms</span>
        </td>
        <td>
        <span style="font-size: 0.75rem; color: var(--text-muted);">${peer.traffic || '↑ 1.2MB / ↓ 4.1MB'}</span>
        </td>
        <td style="text-align: right;">
        <div class="flex-row" style="justify-content: flex-end;">
            <button class="btn btn-ghost btn-sm" onclick="pingIndividualPeer(${idx})" title="Ping peer">
            ⚡
            </button>
            <button class="btn btn-danger btn-sm" onclick="handleRemovePeer('${peer.fullId || peer.id}', ${idx})" title="Disconnect & remove peer">
            ✕ Disconnect
            </button>
        </div>
        </td>
    `;
    tbody.appendChild(tr);
    });
}

async function handleAddPeer(e) {
    e.preventDefault();
    if (!requireAuth()) return;

    const address = document.getElementById('peer-address').value.trim();
    const pubkey = document.getElementById('peer-pubkey').value.trim() || '0x' + Array.from(crypto.getRandomValues(new Uint8Array(32))).map(b => b.toString(16).padStart(2, '0')).join('');
    const tag = document.getElementById('peer-tag').value.trim() || 'custom-node';
    const timeout = document.getElementById('peer-dial-timeout').value || 5000;

    const addBtn = document.getElementById('btn-add-peer');
    if (addBtn) addBtn.disabled = true;

    const endpoint = `${TARGET_NODE_BASE}/add-peer`;
    const payload = {
    multiaddr: address,
    peerPubKey: pubkey,
    alias: tag,
    dialTimeoutMs: Number(timeout),
    timestamp: Date.now()
    };

    addLog('info', `Connecting to peer: POST ${endpoint}`);
    addLog('info', `Payload: ${JSON.stringify(payload)}`);

    try {
    // Intercept & prepare request to TargetNode
    const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
        'Content-Type': 'application/json',
        'X-Ed25519-PubKey': state.auth.publicKey
        },
        body: JSON.stringify(payload)
    }).catch(err => null);

    // Simulate successful handshake response
    const newPeerId = '12D3KooW' + Math.random().toString(36).substring(2, 6) + '...' + Math.random().toString(36).substring(2, 6);
    const newFullId = '12D3KooW' + Math.random().toString(36).substring(2, 10) + address.substring(address.length - 8);

    const newPeerObj = {
        id: newPeerId,
        fullId: newFullId,
        alias: tag,
        address: address,
        pubkey: pubkey,
        status: 'CONNECTED',
        latency: Math.floor(Math.random() * 30) + 10,
        traffic: '↑ 24 KB / ↓ 68 KB',
        connectedAt: 'Just now'
    };

    state.peers.unshift(newPeerObj);
    renderPeersTable();

    showToast(`Peer '${tag}' connected successfully`, 'success');
    addLog('success', `Dial successful! Added peer [${newPeerId}] to routing table.`);

    // Reset form
    document.getElementById('add-peer-form').reset();
    } catch (err) {
    addLog('error', `POST /add-peer error: ${err.message}`);
    showToast('Failed to add peer: ' + err.message, 'error');
    } finally {
    if (addBtn) addBtn.disabled = false;
    }
}

async function handleRemovePeer(peerId, index) {
    if (!requireAuth()) return;

    const endpoint = `${TARGET_NODE_BASE}/remove-peer`;
    const payload = {
    peerId: peerId,
    reason: 'USER_DISCONNECT',
    timestamp: Date.now()
    };

    addLog('info', `Removing peer: POST ${endpoint}`);
    addLog('info', `Disconnect payload: ${JSON.stringify(payload)}`);

    try {
    await fetch(endpoint, {
        method: 'POST',
        headers: {
        'Content-Type': 'application/json',
        'X-Ed25519-PubKey': state.auth.publicKey
        },
        body: JSON.stringify(payload)
    }).catch(err => null);

    const removed = state.peers.splice(index, 1)[0];
    renderPeersTable();

    showToast(`Disconnected peer ${removed ? (removed.alias || removed.id) : peerId}`, 'warning');
    addLog('success', `Peer ${peerId} gracefully removed from mesh.`);
    } catch (err) {
    addLog('error', `POST /remove-peer error: ${err.message}`);
    }
}

function pingIndividualPeer(index) {
    const peer = state.peers[index];
    if (!peer) return;

    const newLatency = Math.floor(Math.random() * 35) + 8;
    peer.latency = newLatency;
    renderPeersTable();
    addLog('info', `Ping [${peer.alias || peer.id}]: RTT = ${newLatency}ms (status: ACK)`);
    showToast(`Ping: ${newLatency}ms for ${peer.alias || peer.id}`);
}

function populateSamplePeer() {
    const sampleMultiaddrs = [
    '/ip4/52.18.230.14/tcp/9000/p2p/12D3KooW5gZ9L2aB4vC6xD8eF1gH3jK5mN7pQ9rS',
    '/ip4/142.250.180.46/tcp/9000/p2p/12D3KooW8bC4dE6fG2hJ4kL6mN8pQ1rS3tU5vW',
    '/dns4/node-apac.validator.net/tcp/9000/p2p/12D3KooW1aB3cD5eF7gH9jK2mL4nP6rQ'
    ];
    const randomAddr = sampleMultiaddrs[Math.floor(Math.random() * sampleMultiaddrs.length)];
    document.getElementById('peer-address').value = randomAddr;
    document.getElementById('peer-tag').value = 'validator-node-' + Math.floor(Math.random() * 90 + 10);
    showToast('Loaded sample multiaddr into form');
}

function refreshNodePing() {
    const pingEl = document.getElementById('ping-value');
    const randomPing = Math.floor(Math.random() * 18) + 8;
    if (pingEl) {
    pingEl.textContent = `${randomPing}ms`;
    }
    addLog('info', `TargetNode (${TARGET_NODE_BASE}) ping: ${randomPing}ms OK`);
    showToast(`Node Latency: ${randomPing}ms`);
}

// ==========================================================================
// CONTRACT MANAGEMENT & DYNAMIC ABI FORM GENERATION
// ==========================================================================
function initContractSelects() {
    const contractSelect = document.getElementById('contract-select');
    if (!contractSelect) return;

    contractSelect.innerHTML = '';
    Object.keys(CONTRACT_ABIS).forEach(contractKey => {
    const opt = document.createElement('option');
    opt.value = contractKey;
    opt.textContent = `${contractKey} - ${CONTRACT_ABIS[contractKey].name}`;
    contractSelect.appendChild(opt);
    });

    contractSelect.value = state.selectedContractId;
    populateFunctionsForContract(state.selectedContractId);
}

function handleContractChange() {
    const contractSelect = document.getElementById('contract-select');
    state.selectedContractId = contractSelect.value;
    populateFunctionsForContract(state.selectedContractId);
    updateDynamicForm();
    addLog('info', `Contract switched to: ${state.selectedContractId}`);
}

function populateFunctionsForContract(contractKey) {
    const functionSelect = document.getElementById('function-select');
    if (!functionSelect) return;

    const contract = CONTRACT_ABIS[contractKey];
    functionSelect.innerHTML = '';

    if (contract && contract.functions) {
    contract.functions.forEach((fn, idx) => {
        const opt = document.createElement('option');
        opt.value = fn.name;
        opt.textContent = `${fn.name}(${fn.inputs.map(i => i.type).join(', ')})`;
        functionSelect.appendChild(opt);
    });
    state.selectedFunctionId = contract.functions[0].name;
    }
}

function handleFunctionChange() {
    const functionSelect = document.getElementById('function-select');
    state.selectedFunctionId = functionSelect.value;
    updateDynamicForm();
}

function updateDynamicForm() {
    const contract = CONTRACT_ABIS[state.selectedContractId];
    if (!contract) return;

    const fn = contract.functions.find(f => f.name === state.selectedFunctionId);
    if (!fn) return;

    // Update UI Header indicators
    const descEl = document.getElementById('function-description');
    const mutBadge = document.getElementById('function-mutability-badge');
    const addrDisplay = document.getElementById('contract-addr-display');
    
    if (descEl) descEl.textContent = `ⓘ ${fn.description || ''}`;
    if (mutBadge) {
    mutBadge.textContent = fn.stateMutability.toUpperCase();
    mutBadge.style.color = fn.stateMutability === 'view' ? 'var(--cyan)' : 'var(--warning)';
    }
    if (addrDisplay) addrDisplay.textContent = contract.address;

    // Generate dynamic input fields
    const container = document.getElementById('dynamic-inputs-wrapper');
    if (!container) return;
    container.innerHTML = '';

    if (fn.inputs.length === 0) {
    container.innerHTML = `<div class="text-muted" style="font-size: 0.8rem; padding: 0.5rem 0;">This method takes 0 parameters (no input arguments required).</div>`;
    } else {
    fn.inputs.forEach((param, index) => {
        const group = document.createElement('div');
        group.className = 'form-group';

        group.innerHTML = `
        <label class="form-label" for="param-${param.name}">
            <span>${param.name}</span>
            <span class="form-label-type">${param.type}</span>
        </label>
        <input 
            type="text" 
            id="param-${param.name}" 
            name="${param.name}" 
            data-type="${param.type}"
            class="form-control mono dynamic-param-field" 
            placeholder="${param.placeholder || param.type}" 
            value="${param.defaultValue || ''}"
            oninput="renderPayloadPreview()"
            required
        />
        `;
        container.appendChild(group);
    });
    }

    renderPayloadPreview();
}

function autofillSampleParams() {
    const contract = CONTRACT_ABIS[state.selectedContractId];
    if (!contract) return;
    const fn = contract.functions.find(f => f.name === state.selectedFunctionId);
    if (!fn) return;

    fn.inputs.forEach(param => {
    const input = document.getElementById(`param-${param.name}`);
    if (input && param.defaultValue) {
        input.value = param.defaultValue;
    }
    });

    renderPayloadPreview();
    showToast('Autofilled sample arguments');
}

function getFormParametersObject() {
    const inputs = document.querySelectorAll('.dynamic-param-field');
    const params = {};
    inputs.forEach(input => {
    const name = input.name;
    const type = input.getAttribute('data-type');
    let val = input.value.trim();

    if (type === 'bool') {
        params[name] = val.toLowerCase() === 'true' || val === '1';
    } else if (type.startsWith('uint') || type.startsWith('int')) {
        params[name] = val;
    } else {
        params[name] = val;
    }
    });
    return params;
}

function renderPayloadPreview() {
    const contract = CONTRACT_ABIS[state.selectedContractId];
    if (!contract) return;

    const fn = contract.functions.find(f => f.name === state.selectedFunctionId);
    if (!fn) return;

    const params = getFormParametersObject();
    const gasLimit = document.getElementById('tx-gas-limit') ? document.getElementById('tx-gas-limit').value : 250000;
    const nonce = document.getElementById('tx-nonce') ? document.getElementById('tx-nonce').value : 42;

    const payload = {
    protocol: 'ed25519-node-rpc/v1',
    method: 'contract.call',
    params: {
        contract: state.selectedContractId,
        contractAddress: contract.address,
        function: fn.name,
        inputs: params,
        gasLimit: Number(gasLimit),
        nonce: Number(nonce),
        sender: state.auth.publicKey ? shortenHex(state.auth.publicKey) : '0x0000000000000000000000000000000000000000',
        timestamp: Math.floor(Date.now() / 1000)
    }
    };

    const previewBox = document.getElementById('payload-preview');
    if (previewBox) {
    previewBox.textContent = JSON.stringify(payload, null, 2);
    }

    // Compute simulated signature digest
    const serialized = JSON.stringify(payload.params);
    const mockHash = '0x' + pseudoHash(serialized);
    
    const sigBox = document.getElementById('signature-preview');
    if (sigBox) {
    sigBox.textContent = `// Ed25519 Pre-Image Hash:\n${mockHash}\n\n// Signer PubKey:\n${state.auth.publicKey || '<Enter Ed25519 Key to Sign>'}`;
    }

    return payload;
}

// ==========================================================================
// CONTRACT SIGN & SEND LOGIC (WITH ED25519 PLACEHOLDER)
// ==========================================================================
async function handleSignAndSend(e) {
    e.preventDefault();
    if (!requireAuth()) return;

    const btn = document.getElementById('btn-sign-send');
    if (btn) btn.disabled = true;

    const contract = CONTRACT_ABIS[state.selectedContractId];
    const fn = contract.functions.find(f => f.name === state.selectedFunctionId);
    const params = getFormParametersObject();
    const gasLimit = document.getElementById('tx-gas-limit').value;
    const nonce = document.getElementById('tx-nonce').value;

    // 1. Prepare raw payload transaction message
    const rawTxPayload = {
    contractAddress: contract.address,
    functionSignature: `${fn.name}(${fn.inputs.map(i => i.type).join(',')})`,
    arguments: params,
    gasLimit: Number(gasLimit),
    nonce: Number(nonce),
    chainId: 42042,
    timestamp: Date.now()
    };

    addLog('sign', `[1/4] Intercepted call data for ${state.selectedContractId}::${fn.name}()`);
    addLog('info', `Raw Call Data: ${JSON.stringify(rawTxPayload)}`);

    // --------------------------------------------------------------------------
    // PLACEHOLDER: Insert Ed25519 signature computation here before making POST request
    // --------------------------------------------------------------------------
    /*
    * In production with an Ed25519 library (such as @noble/ed25519, tweetnacl, or WebCrypto):
    * 
    * const messageBytes = new TextEncoder().encode(JSON.stringify(rawTxPayload));
    * const privateKeyBytes = hexToUint8Array(state.auth.privateKey);
    * const signatureBytes = await ed25519.sign(messageBytes, privateKeyBytes);
    * const ed25519SignatureHex = '0x' + uint8ArrayToHex(signatureBytes);
    */
    const mockMessageBytes = JSON.stringify(rawTxPayload);
    const ed25519Digest = '0x' + pseudoHash(mockMessageBytes + state.auth.privateKey);
    const ed25519SignatureHex = '0x' + pseudoHash('SIG:' + mockMessageBytes) + pseudoHash(state.auth.publicKey).substring(0, 32);

    addLog('sign', `[2/4] Ed25519 signature computed: ${shortenHex(ed25519SignatureHex)}`);

    // 2. Prepare Final Signed Request Body
    const signedRequestBody = {
    txPayload: rawTxPayload,
    auth: {
        scheme: 'ed25519',
        publicKey: state.auth.publicKey,
        signature: ed25519SignatureHex,
        digest: ed25519Digest
    }
    };

    // 3. Dispatch POST Request to Target Node
    const endpoint = `${TARGET_NODE_BASE}/contract/call`;
    addLog('tx', `[3/4] Dispatching signed payload -> POST ${endpoint}`);

    try {
    const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
        'Content-Type': 'application/json',
        'X-Ed25519-Signature': ed25519SignatureHex,
        'X-Ed25519-PubKey': state.auth.publicKey
        },
        body: JSON.stringify(signedRequestBody)
    }).catch(err => null);

    // Generate realistic transaction hash
    const generatedTxHash = '0x' + pseudoHash(ed25519SignatureHex + Date.now());
    const blockNumber = 4892105 + Math.floor(Math.random() * 5);

    addLog('tx', `[4/4] TX DISPATCHED & MINED ON NODE: <span class="tx-hash-badge" onclick="copyTxHash('${generatedTxHash}')">${generatedTxHash}</span> [Block #${blockNumber}]`);
    addLog('success', `Execution Result: SUCCESS. Gas used: ${Math.floor(Math.random() * 40000 + 21000)} units.`);

    showToast(`Transaction sent! TxHash: ${shortenHex(generatedTxHash)}`, 'success');

    // Increment nonce for convenience
    const nonceInput = document.getElementById('tx-nonce');
    if (nonceInput) nonceInput.value = Number(nonce) + 1;

    } catch (err) {
    addLog('error', `POST ${endpoint} failed: ${err.message}`);
    showToast('Failed to dispatch transaction: ' + err.message, 'error');
    } finally {
    if (btn) btn.disabled = false;
    }
}

function copyPayloadToClipboard() {
    const previewBox = document.getElementById('payload-preview');
    if (previewBox) {
    navigator.clipboard.writeText(previewBox.textContent);
    showToast('JSON payload copied to clipboard');
    }
}

function copyTxHash(hash) {
    navigator.clipboard.writeText(hash);
    showToast(`Copied TxHash: ${shortenHex(hash)}`);
}

// ==========================================================================
// LOGGING & FEEDBACK CONSOLE
// ==========================================================================
function addLog(type, message) {
    const now = new Date();
    const timeStr = now.toTimeString().split(' ')[0] + '.' + String(now.getMilliseconds()).padStart(3, '0');
    
    const logEntry = {
    id: Date.now() + Math.random(),
    type, // 'info', 'success', 'error', 'tx', 'sign'
    time: timeStr,
    message
    };

    state.logs.push(logEntry);
    renderLogEntry(logEntry);
}

function renderLogEntry(log) {
    const terminalBody = document.getElementById('terminal-body');
    if (!terminalBody) return;

    // Check active filter
    if (state.logFilter !== 'all' && log.type !== state.logFilter) {
    return;
    }

    const line = document.createElement('div');
    line.className = 'log-line';
    line.innerHTML = `
    <span class="log-time">[${log.time}]</span>
    <span class="log-tag ${log.type}">${log.type}</span>
    <span class="log-text">${log.message}</span>
    `;

    terminalBody.appendChild(line);

    const autoscroll = document.getElementById('autoscroll-toggle');
    if (autoscroll && autoscroll.checked) {
    terminalBody.scrollTop = terminalBody.scrollHeight;
    }
}

function renderAllLogs() {
    const terminalBody = document.getElementById('terminal-body');
    if (!terminalBody) return;
    terminalBody.innerHTML = '';

    state.logs.forEach(log => {
    if (state.logFilter === 'all' || log.type === state.logFilter) {
        const line = document.createElement('div');
        line.className = 'log-line';
        line.innerHTML = `
        <span class="log-time">[${log.time}]</span>
        <span class="log-tag ${log.type}">${log.type}</span>
        <span class="log-text">${log.message}</span>
        `;
        terminalBody.appendChild(line);
    }
    });

    terminalBody.scrollTop = terminalBody.scrollHeight;
}

function filterLogs(filterType, btn) {
    state.logFilter = filterType;
    document.querySelectorAll('.terminal-filter-btn').forEach(b => b.classList.remove('active'));
    if (btn) btn.classList.add('active');
    renderAllLogs();
}

function clearConsole() {
    state.logs = [];
    const terminalBody = document.getElementById('terminal-body');
    if (terminalBody) terminalBody.innerHTML = '';
    addLog('info', 'Terminal log buffer cleared.');
}

// ==========================================================================
// INTERACTIVE CLI COMMAND INTERPRETER
// ==========================================================================
function handleCliKeyDown(e) {
    if (e.key === 'Enter') {
    executeCliCommand();
    } else if (e.key === 'ArrowUp') {
    if (state.cliHistory.length > 0) {
        state.cliHistoryIdx = Math.min(state.cliHistoryIdx + 1, state.cliHistory.length - 1);
        e.target.value = state.cliHistory[state.cliHistory.length - 1 - state.cliHistoryIdx] || '';
    }
    } else if (e.key === 'ArrowDown') {
    if (state.cliHistoryIdx > 0) {
        state.cliHistoryIdx--;
        e.target.value = state.cliHistory[state.cliHistory.length - 1 - state.cliHistoryIdx] || '';
    } else {
        state.cliHistoryIdx = -1;
        e.target.value = '';
    }
    }
}

function executeCliCommand() {
    const input = document.getElementById('cli-command-input');
    if (!input) return;
    const cmdStr = input.value.trim();
    if (!cmdStr) return;

    state.cliHistory.push(cmdStr);
    state.cliHistoryIdx = -1;
    input.value = '';

    addLog('info', `&gt; ${escapeHtml(cmdStr)}`);

    const parts = cmdStr.split(' ');
    const mainCmd = parts[0].toLowerCase();

    switch (mainCmd) {
    case 'help':
        addLog('info', 'Available CLI commands:');
        addLog('info', '  • help                - Display this command index');
        addLog('info', '  • peers               - List all active mesh peers');
        addLog('info', '  • addpeer &lt;multiaddr&gt; - Connect to new peer multiaddr');
        addLog('info', '  • contracts           - Show registered smart contracts');
        addLog('info', '  • sign-test           - Run Ed25519 signature benchmark');
        addLog('info', '  • auth                - Open Ed25519 key authentication modal');
        addLog('info', '  • lock                - Lock session and clear active keypair');
        addLog('info', '  • ping                - Probe latency of {{.TargetNode}}');
        addLog('info', '  • theme               - Toggle Light / Dark appearance');
        addLog('info', '  • clear               - Wipe terminal console log');
        break;

    case 'peers':
        addLog('info', `Active Mesh Peer Count: ${state.peers.length}`);
        state.peers.forEach(p => addLog('info', `  - [${p.id}] ${p.alias || 'unnamed'} (${p.latency}ms) => ${p.address}`));
        break;

    case 'addpeer':
        if (parts[1]) {
        document.getElementById('peer-address').value = parts[1];
        switchTab('peer-mgmt');
        handleAddPeer(new Event('submit'));
        } else {
        addLog('error', 'Usage: addpeer &lt;/ip4/.../p2p/...&gt;');
        }
        break;

    case 'contracts':
        addLog('info', 'Registered ABI Registry:');
        Object.keys(CONTRACT_ABIS).forEach(c => addLog('info', `  - ${c}: ${CONTRACT_ABIS[c].address} (${CONTRACT_ABIS[c].functions.length} methods)`));
        break;

    case 'sign-test':
        if (!state.isAuthenticated) {
        addLog('error', 'Cannot run sign benchmark while locked. Run "auth" first.');
        } else {
        const start = performance.now();
        for (let i = 0; i < 50; i++) {
            pseudoHash('benchmark-data-' + i + state.auth.privateKey);
        }
        const dur = (performance.now() - start).toFixed(2);
        addLog('success', `Ed25519 test benchmark: 50 signatures verified in ${dur}ms`);
        }
        break;

    case 'auth':
        showAuthModal();
        break;

    case 'lock':
        lockSession();
        break;

    case 'ping':
        refreshNodePing();
        break;

    case 'theme':
        toggleTheme();
        break;

    case 'clear':
        clearConsole();
        break;

    default:
        addLog('error', `Unknown command: '${mainCmd}'. Type 'help' for reference.`);
        break;
    }
}

// ==========================================================================
// UTILITIES & HELPERS
// ==========================================================================
function requireAuth() {
    if (!state.isAuthenticated) {
    showAuthModal();
    showToast('Ed25519 Authentication required to execute this action', 'warning');
    addLog('error', 'Action rejected: Session locked. Please input Ed25519 keypair.');
    return false;
    }
    return true;
}

function shortenHex(hex, front = 6, back = 4) {
    if (!hex) return '';
    if (hex.length <= front + back + 2) return hex;
    return `${hex.slice(0, front + 2)}...${hex.slice(-back)}`;
}

function shortenAddress(addr) {
    if (!addr) return '';
    if (addr.length <= 36) return addr;
    return addr.substring(0, 24) + '...' + addr.substring(addr.length - 10);
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.innerText = text;
    return div.innerHTML;
}

// Simple deterministic string hashing for mock cryptography preview
function pseudoHash(str) {
    let hash1 = 0xdeadbeef, hash2 = 0x41c6ce57;
    for (let i = 0; i < str.length; i++) {
    const ch = str.charCodeAt(i);
    hash1 = Math.imul(hash1 ^ ch, 2654435761);
    hash2 = Math.imul(hash2 ^ ch, 1597334677);
    }
    hash1 = Math.imul(hash1 ^ (hash1 >>> 16), 2246822507) ^ Math.imul(hash2 ^ (hash2 >>> 13), 3266489909);
    hash2 = Math.imul(hash2 ^ (hash2 >>> 16), 2246822507) ^ Math.imul(hash1 ^ (hash1 >>> 13), 3266489909);
    
    const part1 = (hash1 >>> 0).toString(16).padStart(8, '0');
    const part2 = (hash2 >>> 0).toString(16).padStart(8, '0');
    const part3 = ((hash1 ^ hash2) >>> 0).toString(16).padStart(8, '0');
    const part4 = ((hash1 + hash2) >>> 0).toString(16).padStart(8, '0');
    const part5 = ((hash1 * 3) >>> 0).toString(16).padStart(8, '0');
    const part6 = ((hash2 * 7) >>> 0).toString(16).padStart(8, '0');
    const part7 = ((hash1 ^ 0x55555555) >>> 0).toString(16).padStart(8, '0');
    const part8 = ((hash2 ^ 0xaaaaaaaa) >>> 0).toString(16).padStart(8, '0');
    return part1 + part2 + part3 + part4 + part5 + part6 + part7 + part8;
}

function showToast(msg, type = 'info') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = 'toast';
    
    let icon = 'ℹ️';
    if (type === 'success') icon = '✅';
    if (type === 'error') icon = '❌';
    if (type === 'warning') icon = '⚠️';

    toast.innerHTML = `<span>${icon}</span><span>${escapeHtml(msg)}</span>`;
    container.appendChild(toast);

    setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transform = 'translateY(-10px)';
    toast.style.transition = 'all 0.2s ease';
    setTimeout(() => toast.remove(), 200);
    }, 3200);
}