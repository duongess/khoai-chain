export interface EXECUTE {
    type: string;
    sender: string;
    contract: string;
    function: string;
    args: string[];
    nonce: string;
    signature: string;
}

export interface PEER {
    Type: string;
    Address: string;
    PublicKey: string;
}

export async function sendP2p(payload: EXECUTE | PEER): Promise<any> {
    console.log(payload);
    try {
        const response = await fetch('/api/p2p/message', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            },
            body: JSON.stringify(payload)
        });

        if (!response.ok) {
            throw new Error(`Error P2P: ${response.statusText} (Code: ${response.status})`);
        }

        const data = await response.json();
        
        if (data.status !== "success") {
            throw new Error(data.error || "HTTP Bridge error");
        }

        // Them trim() de loai bo ky tu \n thua tu server Go
        const p2pResult = JSON.parse(data.result.trim());
        
        if (p2pResult.status === "Error") {
            throw new Error(`[Node Error]: ${p2pResult.error}`);
        }

        return p2pResult;

    } catch (err: any) {
        console.error("Unable to send P2P message:", err.message || err);
        throw err;
    }
}