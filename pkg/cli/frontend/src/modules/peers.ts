/**
 * Peer Management Module
 * Handles GET /peers, POST /add-peer, and POST /remove-peer
 */
import { CONFIG } from '../config.ts';
import { state, PeerItem } from '../state.ts';
import { logConsole } from '../utils/logger.ts';
import { sendP2p, PEER } from './network.ts';

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

export async function executePeers(type : string, isAddress: boolean = false) {
  const id = type.toLowerCase()

  var address
  if (isAddress) {
    const addressOJ = document.getElementById(`input-${id}-address`)
    if (addressOJ) {
      address = addressOJ.textContent
    }
  }
  const payload : PEER = {
    type: type,
    sender: state.auth.publicKey,
    address: address || ""
  }
 try {
    const peers = await sendP2p(payload);

    // Mo hien thi khoi ket qua
    const wrapper = document.getElementById(`responses-${id}`);
    if (wrapper) wrapper.style.display = 'block';

    // In data ra box, format JSON voi 2 space de de doc
    const bodyBox = document.getElementById(`body-${id}`);
    if (bodyBox) {
      bodyBox.textContent = JSON.stringify(peers, null, 2);
    }

    logConsole('info', `(${peers?.length || 0} peers active)`);
  } catch (error: any) {
    // In loi ra man hinh neu viec lay peers that bai
    const wrapper = document.getElementById(id);
    if (wrapper) wrapper.style.display = 'block';

    const bodyBox = document.getElementById(`body-${id}`);
    if (bodyBox) {
      bodyBox.textContent = `Error: ${error.message}`;
    }
    
    logConsole('error', `Get peers failed: ${error.message}`);
  }
}