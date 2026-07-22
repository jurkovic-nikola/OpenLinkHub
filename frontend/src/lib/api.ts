export type ApiEnvelope<T = unknown> = {
	code: number;
	status?: number;
	message?: string;
	device?: T;
	devices?: T;
	data?: T;
	dashboard?: unknown;
};

async function request<T>(path: string, init?: RequestInit): Promise<ApiEnvelope<T>> {
	const res = await fetch(path, {
		headers: { Accept: 'application/json', ...(init?.headers ?? {}) },
		...init
	});
	if (!res.ok) {
		throw new Error(`HTTP ${res.status} ${path}`);
	}
	return (await res.json()) as ApiEnvelope<T>;
}

export function getDevices() {
	return request<Record<string, DeviceSummary>>('/api/');
}

export function getDevice(serial: string) {
	return request(`/api/devices/${encodeURIComponent(serial)}`);
}

export function getTemperatures() {
	return request('/api/temperatures/');
}

export function getDashboard() {
	return request('/api/dashboard');
}

export function getLanguage() {
	return request<Record<string, string>>('/api/language/');
}

export function postJson(path: string, body: unknown) {
	return request(path, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(body)
	});
}

export function putJson(path: string, body: unknown) {
	return request(path, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(body)
	});
}

export function deleteJson(path: string, body: unknown) {
	return request(path, {
		method: 'DELETE',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(body)
	});
}

export type DeviceSummary = {
	ProductType?: number;
	Product?: string;
	Serial?: string;
	Firmware?: string;
	Image?: string;
	Hidden?: boolean;
	GetDevice?: unknown;
};
