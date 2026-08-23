/**
 * Global App State Store
 */

export interface PeerItem {
  peer_id: string;
  multiaddr: string;
  latency_ms: number;
  status: 'connected' | 'dialing' | 'disconnected';
  tag: string;
}

export interface AuthState {
  publicKey: string;
  privateKey: string;
  isAuthorized: boolean;
}

export interface AppState {
  theme: 'light' | 'dark';
  auth: AuthState;
  peers: PeerItem[];
  selectedContract: string;
  selectedFunction: string;
}

export const state: AppState = {
  theme: 'light',
  auth: {
    publicKey: '',
    privateKey: '',
    isAuthorized: false
  },
  peers: [
    {
      peer_id: '12D3KooWStZ1k4P8ZJ9mY7vU82aWc78e1',
      multiaddr: '/ip4/198.51.100.24/tcp/9000/p2p/12D3KooWStZ1k4P8ZJ9mY7vU82aWc78e1',
      latency_ms: 18,
      status: 'connected',
      tag: 'validator-frankfurt'
    },
    {
      peer_id: '12D3KooWNx9Bv3T4qL2eR5kY61aZb09f4',
      multiaddr: '/ip4/203.0.113.88/tcp/9000/p2p/12D3KooWNx9Bv3T4qL2eR5kY61aZb09f4',
      latency_ms: 32,
      status: 'connected',
      tag: 'sentry-singapore'
    },
    {
      peer_id: '12D3KooW8kP2qV9mB7xL1yZ34eR5aT6u7',
      multiaddr: '/ip4/192.0.2.14/tcp/9000/p2p/12D3KooW8kP2qV9mB7xL1yZ34eR5aT6u7',
      latency_ms: 12,
      status: 'connected',
      tag: 'bootnode-us-east'
    }
  ],
  selectedContract: 'TokenVault',
  selectedFunction: 'deposit'
};
