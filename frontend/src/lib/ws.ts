export type WsServerMessage = {
	type: string;
	topic?: string;
	serial?: string;
	data?: unknown;
};

type Handler = (msg: WsServerMessage) => void;

function wsUrl(): string {
	const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
	return `${proto}//${location.host}/api/ws`;
}

class WsClient {
	private socket: WebSocket | null = null;
	private handlers = new Set<Handler>();
	private wanted = new Set<string>();
	private retryMs = 1000;
	private closed = false;

	connect() {
		this.closed = false;
		if (this.socket && (this.socket.readyState === WebSocket.OPEN || this.socket.readyState === WebSocket.CONNECTING)) {
			return;
		}
		const sock = new WebSocket(wsUrl());
		this.socket = sock;

		sock.onopen = () => {
			this.retryMs = 1000;
			for (const topic of this.wanted) {
				this.sendSubscribe(topic);
			}
		};

		sock.onmessage = (ev) => {
			try {
				const msg = JSON.parse(String(ev.data)) as WsServerMessage;
				for (const h of this.handlers) h(msg);
			} catch {
				/* ignore */
			}
		};

		sock.onclose = () => {
			this.socket = null;
			if (this.closed) return;
			const wait = this.retryMs;
			this.retryMs = Math.min(this.retryMs * 2, 15000);
			setTimeout(() => this.connect(), wait);
		};
	}

	disconnect() {
		this.closed = true;
		this.socket?.close();
		this.socket = null;
	}

	onMessage(handler: Handler) {
		this.handlers.add(handler);
		return () => this.handlers.delete(handler);
	}

	subscribe(topic: string) {
		this.wanted.add(topic);
		if (this.socket?.readyState === WebSocket.OPEN) {
			this.sendSubscribe(topic);
		} else {
			this.connect();
		}
	}

	unsubscribe(topic: string) {
		this.wanted.delete(topic);
		if (this.socket?.readyState === WebSocket.OPEN) {
			this.socket.send(JSON.stringify({ type: 'unsubscribe', topic }));
		}
	}

	subscribeDevice(serial: string) {
		this.subscribe(`device:${serial}`);
	}

	unsubscribeDevice(serial: string) {
		this.unsubscribe(`device:${serial}`);
	}

	private sendSubscribe(topic: string) {
		const payload =
			topic.startsWith('device:')
				? { type: 'subscribe', serial: topic.slice(7), topic }
				: { type: 'subscribe', topic };
		this.socket?.send(JSON.stringify(payload));
	}
}

export const ws = new WsClient();
