export function t(strings: Record<string, string>, key: string, fallback?: string): string {
	return strings[key] ?? fallback ?? key;
}
