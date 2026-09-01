import { createBrokerApp } from "../src/app.js";
import { MemoryClientStore, MemoryInstanceStore, MemoryNonceStore, MemorySessionStore } from "../src/session.js";
import { bodyHashBase64Url, buildStringToSign, generateEd25519KeyPair, signStringToSign } from "../src/crypto.js";

export function testEnv(overrides: Record<string, string | undefined> = {}): Record<string, string | undefined> {
  return {
    BROKER_PUBLIC_URL: "https://broker.example.com",
    SESSION_TTL: "600",
    DROPBOX_CLIENT_ID: "dropbox-id",
    DROPBOX_CLIENT_SECRET: "dropbox-secret",
    ...overrides,
  };
}

export function createTestApp(
  env: Record<string, string | undefined> = testEnv(),
  opts: {
    store?: MemorySessionStore;
    instanceStore?: MemoryInstanceStore;
    clientStore?: MemoryClientStore;
    nonceStore?: MemoryNonceStore;
    fetchImpl?: typeof fetch;
  } = {},
) {
  const store = opts.store ?? new MemorySessionStore();
  const instanceStore = opts.instanceStore ?? new MemoryInstanceStore();
  const clientStore = opts.clientStore ?? new MemoryClientStore();
  const nonceStore = opts.nonceStore ?? new MemoryNonceStore();
  const app = createBrokerApp({ store, instanceStore, clientStore, nonceStore, env, fetchImpl: opts.fetchImpl });
  return { app, store, instanceStore, clientStore, nonceStore, env };
}

export type TestKeyPair = { publicKeyB64Url: string; privateKeyB64Url: string; clientId: string };

export async function generateTestKeyPair(): Promise<TestKeyPair> {
  return generateEd25519KeyPair();
}

export async function registerClient(app: any, kp: TestKeyPair) {
  return app.request("/v1/clients", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ client_id: kp.clientId, public_key: kp.publicKeyB64Url }),
  });
}

export async function signedHeaders(kp: TestKeyPair, method: string, path: string, body: string): Promise<Record<string, string>> {
  const t = String(Math.floor(Date.now() / 1000));
  const nonce = `n-${Math.random().toString(36).slice(2, 10)}-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`;
  const bh = bodyHashBase64Url(body);
  const sts = buildStringToSign(kp.clientId, method, path, t, nonce, bh);
  const sig = await signStringToSign(kp.privateKeyB64Url, kp.publicKeyB64Url, sts);
  return { authorization: `SRAT-Signature client_id="${kp.clientId}", t="${t}", nonce="${nonce}", sig="${sig}"` };
}

const defaultKeyPairs = new WeakMap<any, TestKeyPair>();

export async function getOrCreateDefaultKeyPair(app: any): Promise<TestKeyPair> {
  let kp = defaultKeyPairs.get(app);
  if (kp) return kp;
  kp = await generateTestKeyPair();
  defaultKeyPairs.set(app, kp);
  await registerClient(app, kp);
  return kp;
}

export async function registerInstance(app: any, instanceId: string, redirectUrl: string, kp?: TestKeyPair) {
  const key = kp ?? (await getOrCreateDefaultKeyPair(app));
  const body = JSON.stringify({ instance_id: instanceId, redirect_url: redirectUrl });
  const headers = await signedHeaders(key, "POST", "/v1/instances/register", body);
  return app.request("/v1/instances/register", {
    method: "POST",
    headers: { "content-type": "application/json", ...headers },
    body,
  });
}

export async function signedStartRequest(app: any, bodyObj: Record<string, unknown>, kp?: TestKeyPair) {
  const key = kp ?? (await getOrCreateDefaultKeyPair(app));
  const body = JSON.stringify(bodyObj);
  const headers = await signedHeaders(key, "POST", "/v1/start", body);
  return app.request("/v1/start", {
    method: "POST",
    headers: { "content-type": "application/json", ...headers },
    body,
  });
}

export async function signedSessionRequest(app: any, sessionId: string, kp?: TestKeyPair) {
  const key = kp ?? (await getOrCreateDefaultKeyPair(app));
  const headers = await signedHeaders(key, "GET", `/v1/session/${sessionId}`, "");
  return app.request(`/v1/session/${sessionId}`, {
    headers: { ...headers },
  });
}

export async function jsonBody(resp: Response): Promise<Record<string, unknown>> {
  const text = await resp.text();
  try {
    return JSON.parse(text) as Record<string, unknown>;
  } catch {
    return { _raw: text };
  }
}
