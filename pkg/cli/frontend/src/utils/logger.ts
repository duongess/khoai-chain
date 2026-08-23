/**
 * Console Logger Utility
 * Outputs timestamped messages to the docked FastAPI console window.
 */

export type LogLevel = 'info' | 'success' | 'error' | 'tx';

export function logConsole(type: LogLevel, message: string, txHash: string | null = null): void {
  const logsContainer = document.getElementById('console-logs');
  if (!logsContainer) return;

  const time = new Date().toLocaleTimeString('en-US', { hour12: false });
  const line = document.createElement('div');
  line.className = 'console-line';

  let tagClass = 'tag-info';
  let tagText = 'INFO';
  if (type === 'success') { tagClass = 'tag-success'; tagText = '200 OK'; }
  if (type === 'error') { tagClass = 'tag-error'; tagText = 'ERROR'; }
  if (type === 'tx') { tagClass = 'tag-tx'; tagText = 'TX CONFIRMED'; }

  let txHtml = '';
  if (txHash) {
    txHtml = ` | TxHash: <span class="tx-hash-link" onclick="navigator.clipboard.writeText('${txHash}')" title="Click to copy">${txHash.slice(0, 16)}...</span>`;
  }

  line.innerHTML = `
    <span style="color: #64748b;">[${time}]</span>
    <span class="console-tag ${tagClass}">${tagText}</span>
    <span>${message}${txHtml}</span>
  `;

  logsContainer.appendChild(line);
  logsContainer.scrollTop = logsContainer.scrollHeight;
}

export function clearTerminalLogs(): void {
  const logsContainer = document.getElementById('console-logs');
  if (logsContainer) {
    logsContainer.innerHTML = '';
    logConsole('info', 'Console logs cleared.');
  }
}
