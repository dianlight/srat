import { createHash, createPrivateKey, sign, generateKeyPairSync } from "node:crypto";

// ---- base64url helpers ----
export function base64UrlEncode(bytes: Uint8Array): string {
  if (typeof Buffer !== "undefined" && typeof Buffer.from === "function") {
    return (Buffer.from(bytes) as unknown as { toString(enc: string): string }).toString("base64url");
  }
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  const b64 = btoa(binary);
  return b64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function base64UrlDecode(s: string): Uint8Array {
  if (typeof Buffer !== "undefined" && typeof Buffer.from === "function") {
    return new Uint8Array(Buffer.from(s, "base64url"));
  }
  // Web fallback
  let b64 = s.replace(/-/g, "+").replace(/_/g, "/");
  while (b64.length % 4) b64 += "=";
  const binary = atob(b64);
  const out = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) out[i] = binary.charCodeAt(i);
  return out;
}

export function sha256Bytes(data: string | Uint8Array): Uint8Array {
  const h = createHash("sha256");
  if (typeof data === "string") h.update(data, "utf8");
  else h.update(data);
  return new Uint8Array(h.digest());
}

export function bodyHashBase64Url(rawBody: string): string {
  return base64UrlEncode(sha256Bytes(rawBody));
}

// client_id = base64url(SHA256(rawPubkey 32B))
export function computeClientId(publicKeyB64Url: string): string {
  const pub = base64UrlDecode(publicKeyB64Url);
  return base64UrlEncode(sha256Bytes(pub));
}

export function isValidPublicKeyB64Url(s: string): boolean {
  try {
    const b = base64UrlDecode(s);
    return b.length === 32;
  } catch {
    return false;
  }
}

export function isValidClientId(s: string): boolean {
  // SHA256 hash => 32 bytes => 43 chars b64url (no padding)
  return /^[A-Za-z0-9_-]{43}$/.test(s);
}

// stringToSign = client_id + "\n" + METHOD + "\n" + PATH + "\n" + t + "\n" + nonce + "\n" + bodyHash
export function buildStringToSign(
  clientId: string,
  method: string,
  path: string,
  t: string,
  nonce: string,
  bodyHash: string,
): string {
  return `${clientId}\n${method.toUpperCase()}\n${path}\n${t}\n${nonce}\n${bodyHash}`;
}

export type ParsedSignature = {
  clientId: string;
  t: string;
  nonce: string;
  sig: string;
};

export function parseSratSignature(header: string | null | undefined): ParsedSignature | null {
  if (!header) return null;
  // Expect: SRAT-Signature client_id="...", t="...", nonce="...", sig="..."
  const prefix = "SRAT-Signature";
  if (!header.startsWith(prefix)) return null;
  const rest = header.slice(prefix.length).trim();
  // parse key="value" pairs
  const params: Record<string, string> = {};
  const re = /(\w+)\s*=\s*"([^"]*)"/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(rest)) !== null) {
    params[m[1]] = m[2];
  }
  const clientId = params.client_id || params.clientId || "";
  const t = params.t || "";
  const nonce = params.nonce || "";
  const sig = params.sig || "";
  if (!clientId || !t || !nonce || !sig) return null;
  return { clientId, t, nonce, sig };
}

export function isValidNonce(s: string): boolean {
  return /^[A-Za-z0-9_-]{16,128}$/.test(s);
}

// Verify Ed25519 using WebCrypto subtle (portable) with Node fallback
export async function verifyEd25519(publicKeyB64Url: string, message: string, sigB64Url: string): Promise<boolean> {
  const pubBytes = base64UrlDecode(publicKeyB64Url);
  const sigBytes = base64UrlDecode(sigB64Url);
  const msgBytes = new TextEncoder().encode(message);
  if (pubBytes.length !== 32 || sigBytes.length !== 64) return false;

  // Try WebCrypto subtle first (Workers, Bun, Node 20+)
  try {
    const subtle = (globalThis.crypto as unknown as { subtle: SubtleCrypto })?.subtle;
    if (subtle) {
      const key = await subtle.importKey("raw", pubBytes as unknown as ArrayBuffer, { name: "Ed25519" } as unknown as string, false, ["verify"]);
      return await subtle.verify({ name: "Ed25519" } as unknown as string, key, sigBytes as unknown as ArrayBuffer, msgBytes as unknown as ArrayBuffer);
    }
  } catch {
    // fall through to Node crypto
  }

  // Node crypto fallback
  try {
    const { verify } = await import("node:crypto");
    // Node verify wants KeyObject; create public key from raw
    const { createPublicKey } = await import("node:crypto");
    // Ed25519 raw key import via JWK
    const jwk = { kty: "OKP", crv: "Ed25519", x: publicKeyB64Url } as unknown as string;
    const pubKey = createPublicKey({ key: jwk as unknown as string, format: "jwk" } as unknown as never);
    return verify(null, msgBytes, pubKey, sigBytes);
  } catch {
    return false;
  }
}

export async function generateEd25519KeyPair(): Promise<{ publicKeyB64Url: string; privateKeyB64Url: string; clientId: string }> {
  const { publicKey, privateKey } = generateKeyPairSync("ed25519");
  const pubJwk = publicKey.export({ format: "jwk" }) as unknown as { x: string };
  const privJwk = privateKey.export({ format: "jwk" }) as unknown as { d: string };
  const pubB64 = pubJwk.x;
  const privB64 = privJwk.d;
  const clientId = computeClientId(pubB64);
  return { publicKeyB64Url: pubB64, privateKeyB64Url: privB64, clientId };
}

export function generateEd25519KeyPairSync(): { publicKeyB64Url: string; privateKeyB64Url: string; clientId: string } {
  const { publicKey, privateKey } = generateKeyPairSync("ed25519");
  const pubJwk = publicKey.export({ format: "jwk" }) as unknown as { x: string };
  const privJwk = privateKey.export({ format: "jwk" }) as unknown as { d: string };
  const pubB64 = pubJwk.x;
  const privB64 = privJwk.d;
  const clientId = computeClientId(pubB64);
  return { publicKeyB64Url: pubB64, privateKeyB64Url: privB64, clientId };
}

export async function signStringToSign(privateKeyB64Url: string, publicKeyB64Url: string, message: string): Promise<string> {
  const privJwk = { kty: "OKP", crv: "Ed25519", d: privateKeyB64Url, x: publicKeyB64Url } as unknown as string;
  const privKey = createPrivateKey({ key: privJwk as unknown as string, format: "jwk" } as unknown as never);
  const sig = sign(null, Buffer.from(message, "utf8"), privKey);
  return base64UrlEncode(new Uint8Array(sig));
}

export function signStringToSignSync(privateKeyB64Url: string, publicKeyB64Url: string, message: string): string {
  const privJwk = { kty: "OKP", crv: "Ed25519", d: privateKeyB64Url, x: publicKeyB64Url } as unknown as string;
  const privKey = createPrivateKey({ key: privJwk as unknown as string, format: "jwk" } as unknown as never);
  const sig = sign(null, Buffer.from(message, "utf8"), privKey);
  return base64UrlEncode(new Uint8Array(sig));
}
