const NON_HUB = new Set([8, 9]); // memory, nexus

export function isHubFamily(productType: number | undefined): boolean {
	return productType != null && productType < 100 && !NON_HUB.has(productType);
}

export function isKeyboardFamily(productType: number | undefined): boolean {
	return productType != null && productType >= 101 && productType < 200;
}

export function isMouseFamily(productType: number | undefined): boolean {
	return productType != null && productType >= 200 && productType < 300;
}

export function isHeadsetFamily(productType: number | undefined): boolean {
	return productType != null && productType >= 300 && productType < 400;
}

export type DeviceFamily = 'hub' | 'keyboard' | 'mouse' | 'headset' | 'generic';

export function deviceFamily(productType: number | undefined): DeviceFamily {
	if (isHubFamily(productType)) return 'hub';
	if (isKeyboardFamily(productType)) return 'keyboard';
	if (isMouseFamily(productType)) return 'mouse';
	if (isHeadsetFamily(productType)) return 'headset';
	return 'generic';
}
