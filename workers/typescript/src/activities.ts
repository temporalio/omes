export async function noop() {
  return undefined;
}

export async function delay(delay_for_ms: number) {
  return new Promise((resolve) => setTimeout(resolve, delay_for_ms));
}