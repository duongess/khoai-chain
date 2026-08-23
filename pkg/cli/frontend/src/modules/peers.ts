/**
 * Peer Management Module
 * Handles GET /peers, POST /add-peer, and POST /remove-peer
 */
import { CONFIG } from '../config.ts';
import { state, PeerItem } from '../state.ts';
import { logConsole } from '../utils/logger.ts';

export function updatePeersDropdown(): void {
  const select = document.getElementById('input-remove-peer-id') as HTMLSelectElement;
  if (!select) return;
  select.innerHTML = '';

  state.peers.forEach((p) => {
    const opt = document.createElement('option');
    opt.value = p.peer_id;
    opt.textContent = `${p.tag} (${p.peer_id.slice(0, 14)}...)`;
    select.appendChild(opt);
  });

  const countBadge = document.getElementById('peers-count-tag');
  if (countBadge) countBadge.textContent = String(state.peers.length);
}

export function executeGetPeers(): void {
  const url = `${CONFIG.TARGET_NODE}/peers`;
  const curlBox = document.getElementById('curl-get-peers');
  const urlBox = document.getElementById('url-get-peers');
  const bodyBox = document.getElementById('body-get-peers');
  const responsesWrapper = document.getElementById('responses-get-peers');

  if (curlBox) curlBox.textContent = `curl -X 'GET' '${url}' -H 'accept: application/json'`;
  if (urlBox) urlBox.textContent = url;
  if (bodyBox) bodyBox.textContent = JSON.stringify(state.peers, null, 2);
  if (responsesWrapper) responsesWrapper.style.display = 'block';

  logConsole('info', `GET ${url} -> 200 OK (${state.peers.length} peers active)`);
}

export function executeAddPeer(): void {
  const addrInput = document.getElementById('input-add-peer-address') as HTMLInputElement;
  const pubInput = document.getElementById('input-add-peer-pubkey') as HTMLInputElement;
  const tagInput = document.getElementById('input-add-peer-tag') as HTMLInputElement;

  const addr = addrInput ? addrInput.value.trim() : '';
  const pub = pubInput ? pubInput.value.trim() : '';
  const tag = (tagInput && tagInput.value.trim()) || 'custom-peer';

  if (!addr) {
    alert('Peer multiaddr address is required.');
    return;
  }

  const newPeer: PeerItem = {
    peer_id: '12D3KooW' + Math.random().toString(36).substring(2, 10).toUpperCase() + 'xyz',
    multiaddr: addr,
    latency_ms: Math.floor(Math.random() * 35) + 10,
    status: 'connected',
    tag: tag
  };

  state.peers.push(newPeer);
  updatePeersDropdown();

  const url = `${CONFIG.TARGET_NODE}/add-peer`;
  const payload = { address: addr, pubkey: pub || undefined, tag: tag };

  const curlAdd = document.getElementById('curl-add-peer');
  const bodyAdd = document.getElementById('body-add-peer');
  const responsesAdd = document.getElementById('responses-add-peer');

  if (curlAdd) {
    curlAdd.textContent = `curl -X 'POST' '${url}' \\
  -H 'accept: application/json' \\
  -H 'Content-Type: application/json' \\
  -d '${JSON.stringify(payload)}'`;
  }

  const responseJson = {
    status: "success",
    message: "Peer successfully dialed and added to routing table",
    peer: newPeer
  };

  if (bodyAdd) bodyAdd.textContent = JSON.stringify(responseJson, null, 2);
  if (responsesAdd) responsesAdd.style.display = 'block';

  logConsole('success', `POST ${url} -> 200 OK Connected peer ${newPeer.peer_id.slice(0, 12)} (${tag})`);
}

export function executeRemovePeer(): void {
  const select = document.getElementById('input-remove-peer-id') as HTMLSelectElement;
  const reasonInput = document.getElementById('input-remove-reason') as HTMLInputElement;

  const peerId = select ? select.value : '';
  const reason = reasonInput ? reasonInput.value : 'Manual prune';

  if (!peerId) {
    alert('No peer selected to remove.');
    return;
  }

  const idx = state.peers.findIndex((p) => p.peer_id === peerId);
  if (idx !== -1) {
    const removed = state.peers.splice(idx, 1)[0];
    updatePeersDropdown();

    const url = `${CONFIG.TARGET_NODE}/remove-peer`;
    const responseJson = {
      status: "success",
      message: "Peer pruned from routing table",
      disconnected_peer: removed,
      reason: reason
    };

    const bodyRemove = document.getElementById('body-remove-peer');
    const responsesRemove = document.getElementById('responses-remove-peer');

    if (bodyRemove) bodyRemove.textContent = JSON.stringify(responseJson, null, 2);
    if (responsesRemove) responsesRemove.style.display = 'block';

    logConsole('info', `POST ${url} -> 200 OK Removed peer ${peerId.slice(0, 12)}`);
  }
}
