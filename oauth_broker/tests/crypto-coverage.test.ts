import { describe, it, expect, vi, afterEach } from "vitest";
import {
  base64UrlEncode,
  base64UrlDecode,
  sha256Bytes,
  bodyHashBase64Url,
  computeClientId,
  isValidPublicKeyB64Url,
  isValidClientId,
  isValidNonce,
  parseSratSignature,
  buildStringToSign,
  verifyEd25519,
  generateEd25519KeyPair,
  generateEd25519KeyPairSync,
  signStringToSign,
  signStringToSignSync,
} from "../src/crypto.js";

describe("crypto.ts – coverage stubs (patch 52% → 85%+)", () => {
  afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });
  it("base64UrlEncode / Decode – Buffer vs Web fallback", () => {
    const bytes = new Uint8Array([1,2,3,255,0,128]);
    const encViaBuffer = base64UrlEncode(bytes);
    expect(base64UrlDecode(encViaBuffer)).toEqual(bytes);
    vi.stubGlobal("Buffer", undefined as unknown as typeof Buffer);
    const encViaWeb = base64UrlEncode(bytes);
    expect(encViaWeb).toBe(encViaBuffer);
    expect(base64UrlDecode(encViaWeb)).toEqual(bytes);
    vi.unstubAllGlobals();
    vi.stubGlobal("Buffer", undefined as unknown as typeof Buffer);
    const tiny = new Uint8Array([42]);
    expect(base64UrlDecode(base64UrlEncode(tiny))).toEqual(tiny);
    vi.unstubAllGlobals();
  });
  it("sha256Bytes Uint8Array branch + bodyHash", () => {
    expect(sha256Bytes("hello")).toEqual(sha256Bytes(new TextEncoder().encode("hello")));
    expect(bodyHashBase64Url("")).toBe("47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU");
  });
  it("isValidPublicKeyB64Url wrong length", async () => {
    expect(isValidPublicKeyB64Url("!!!not-b64!!!")).toBe(false);
    const valid = (await generateEd25519KeyPair()).publicKeyB64Url;
    expect(isValidPublicKeyB64Url(valid)).toBe(true);
    expect(isValidPublicKeyB64Url(base64UrlEncode(new Uint8Array(31)))).toBe(false);
    expect(isValidPublicKeyB64Url(base64UrlEncode(new Uint8Array(33)))).toBe(false);
  });
  it("isValidClientId / isValidNonce boundaries", () => {
    expect(isValidClientId("short")).toBe(false);
    expect(isValidClientId("A".repeat(43))).toBe(true);
    expect(isValidNonce("a".repeat(15))).toBe(false);
    expect(isValidNonce("a".repeat(16))).toBe(true);
    expect(isValidNonce("a".repeat(128))).toBe(true);
    expect(isValidNonce("a".repeat(129))).toBe(false);
  });
  it("parseSratSignature edge cases", () => {
    expect(parseSratSignature(null)).toBeNull();
    expect(parseSratSignature("Bearer token")).toBeNull();
    expect(parseSratSignature('SRAT-Signature client_id="cid"')).toBeNull();
    expect(parseSratSignature('SRAT-Signature clientId="cid-alt", t="1", nonce="n-1234567890123456", sig="s"')?.clientId).toBe("cid-alt");
  });
  it("buildStringToSign uppercases method", () => {
    expect(buildStringToSign("cid","post","/v1/start","123","n-1234567890123456","hash")).toBe("cid\nPOST\n/v1/start\n123\nn-1234567890123456\nhash");
  });
  it("computeClientId roundtrip", async () => {
    const kp = await generateEd25519KeyPair();
    expect(computeClientId(kp.publicKeyB64Url)).toBe(kp.clientId);
    expect(computeClientId(generateEd25519KeyPairSync().publicKeyB64Url)).toMatch(/^[A-Za-z0-9_-]{43}$/);
  });
  it("sign sync vs async verifiable", async () => {
    const kp = await generateEd25519KeyPair();
    const msg = buildStringToSign(kp.clientId,"POST","/v1/start","999","n-cover-1234567890",bodyHashBase64Url("{}"));
    const sigAsync = await signStringToSign(kp.privateKeyB64Url,kp.publicKeyB64Url,msg);
    const sigSync = signStringToSignSync(kp.privateKeyB64Url,kp.publicKeyB64Url,msg);
    expect(await verifyEd25519(kp.publicKeyB64Url,msg,sigAsync)).toBe(true);
    expect(await verifyEd25519(kp.publicKeyB64Url,msg,sigSync)).toBe(true);
    expect(await verifyEd25519(kp.publicKeyB64Url,msg+"x",sigAsync)).toBe(false);
  });
  it("verifyEd25519 length guards", async () => {
    const kp = await generateEd25519KeyPair();
    expect(await verifyEd25519(base64UrlEncode(new Uint8Array(31)),"msg",base64UrlEncode(new Uint8Array(64)))).toBe(false);
    expect(await verifyEd25519(kp.publicKeyB64Url,"msg",base64UrlEncode(new Uint8Array(63)))).toBe(false);
  });
  it("verifyEd25519 fallback paths", async () => {
    const kp = await generateEd25519KeyPair();
    const msg="fallback-test";
    const sig = await signStringToSign(kp.privateKeyB64Url,kp.publicKeyB64Url,msg);
    vi.stubGlobal("crypto", { subtle: { importKey: async () => { throw new Error("boom"); }, verify: async () => { throw new Error("boom"); } } } as unknown as Crypto);
    expect(await verifyEd25519(kp.publicKeyB64Url,msg,sig)).toBe(true);
    vi.stubGlobal("crypto", undefined as unknown as Crypto);
    expect(await verifyEd25519(kp.publicKeyB64Url,msg,sig)).toBe(true);
    vi.unstubAllGlobals();
  });
});
