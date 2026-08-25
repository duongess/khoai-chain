/**
 * Contract ABI Definitions & Types
 * Provides mock JSON ABIs used to dynamically generate form inputs.
 */

export interface ContractABIInput {
  name: string;
  type: string;
  default: string;
}

export interface ContractABIFunction {
  name: string;
  description: string;
  inputs: ContractABIInput[];
}

export interface ContractABISpec {
  name: string;
  address: string;
  functions: ContractABIFunction[];
}

export const CONTRACT_ABIS: Record<string, ContractABISpec> = {
  TokenVault: {
    name: 'TokenVault (ERC-4626 Vault)',
    address: '0x8b329482701b7a2d8329482701b7a2d832948270',
    functions: [
      {
        name: 'deposit',
        description: 'Deposit assets into the vault to mint share tokens to recipient.',
        inputs: [
          { name: 'amount', type: 'uint256', default: '5000000000000000000' },
          { name: 'recipient', type: 'address', default: '0x9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b' }
        ]
      },
      {
        name: 'withdraw',
        description: 'Burn vault shares to withdraw underlying assets.',
        inputs: [
          { name: 'shares', type: 'uint256', default: '2500000000000000000' },
          { name: 'recipient', type: 'address', default: '0x9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b' }
        ]
      },
      {
        name: 'transferOwnership',
        description: 'Transfer vault control to new Ed25519 controller.',
        inputs: [
          { name: 'newOwner', type: 'address', default: '0x4321098765432109876543210987654321098765' }
        ]
      }
    ]
  },
  GovernanceDAO: {
    name: 'GovernanceDAO (Voting Module)',
    address: '0x3a99281745672201991823746281923847192834',
    functions: [
      {
        name: 'propose',
        description: 'Create a new governance action proposal.',
        inputs: [
          { name: 'title', type: 'string', default: 'CIP-42: Increase Peer Mesh Capacity' },
          { name: 'actionHash', type: 'bytes32', default: '0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855' },
          { name: 'votingBlocks', type: 'uint32', default: '40320' }
        ]
      },
      {
        name: 'castVote',
        description: 'Submit an Ed25519-signed vote (0=Against, 1=For, 2=Abstain).',
        inputs: [
          { name: 'proposalId', type: 'uint256', default: '104' },
          { name: 'support', type: 'uint8', default: '1' }
        ]
      }
    ]
  },
  OracleBridge: {
    name: 'OracleBridge (Price Feed)',
    address: '0xf49281a8c9201923485710293847102938471029',
    functions: [
      {
        name: 'updatePriceFeed',
        description: 'Publish oracle rate report with cryptographic proof.',
        inputs: [
          { name: 'symbol', type: 'string', default: 'ETH/USD' },
          { name: 'price', type: 'uint256', default: '354250000000' },
          { name: 'timestamp', type: 'uint64', default: String(Math.floor(Date.now() / 1000)) }
        ]
      }
    ]
  }
};
