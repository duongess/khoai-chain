interface EXECUTE {
    Type: string;
    Sender: string;
    Contract: string;
    Function: string;
    Args: string[];
    Nonce: string;
}

interface PEER {
    Type: string;
    Address: string;
    PublicKey: string;
}

export async function sendP2p(data: EXECUTE | PEER): Promise<any> {
    try {
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            },
            body: JSON.stringify(data)
        });

        if (!response.ok) {
            throw new Error(`Error P2P: ${response.statusText} (Mã: ${response.status})`);
        }

        const result = await response.json();
        return result;
    } catch (err: any) {
        console.error("Không thể gửi thông điệp P2P tới Node:", err);
        throw err;
    }
}