// Passphrase-derived encryption for everything this extension stores
// at rest. Per A012's own Decisions: this is non-negotiable v1 scope,
// not deferred hardening — chrome.storage.local has no encryption of
// its own beyond normal browser-profile access control, and every
// private key held here is exactly as safe as this file.
//
// PBKDF2 (derive a key from the user's passphrase) -> AES-GCM
// (authenticated encryption of the actual secret). Both are WebCrypto
// primitives, never hand-rolled — same "don't hand-roll crypto"
// discipline as this project's own Go side (go-nostr, nostr-tools).

const PBKDF2_ITERATIONS = 600_000; // OWASP's 2023 minimum for PBKDF2-SHA256
const SALT_BYTES = 16;
const IV_BYTES = 12; // AES-GCM's standard nonce size

export interface EncryptedBlob {
  saltB64: string;
  ivB64: string;
  ciphertextB64: string;
}

function bytesToB64(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes));
}

function b64ToBytes(b64: string): Uint8Array {
  return Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
}

async function deriveKey(passphrase: string, salt: Uint8Array): Promise<CryptoKey> {
  const material = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(passphrase),
    "PBKDF2",
    false,
    ["deriveKey"],
  );
  return crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: salt as BufferSource, iterations: PBKDF2_ITERATIONS, hash: "SHA-256" },
    material,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
}

/** Encrypts plaintext (a secret key's raw bytes) under passphrase. */
export async function encrypt(passphrase: string, plaintext: Uint8Array): Promise<EncryptedBlob> {
  const salt = crypto.getRandomValues(new Uint8Array(SALT_BYTES));
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES));
  const key = await deriveKey(passphrase, salt);
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: iv as BufferSource },
    key,
    plaintext as BufferSource,
  );
  return {
    saltB64: bytesToB64(salt),
    ivB64: bytesToB64(iv),
    ciphertextB64: bytesToB64(new Uint8Array(ciphertext)),
  };
}

/**
 * Decrypts a blob under passphrase. Throws (AES-GCM's own
 * authentication check fails) if the passphrase is wrong — there is
 * no separate "is this the right passphrase" check, the failed
 * decrypt itself is the check.
 */
export async function decrypt(passphrase: string, blob: EncryptedBlob): Promise<Uint8Array> {
  const salt = b64ToBytes(blob.saltB64);
  const iv = b64ToBytes(blob.ivB64);
  const key = await deriveKey(passphrase, salt);
  const plaintext = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: iv as BufferSource },
    key,
    b64ToBytes(blob.ciphertextB64) as BufferSource,
  );
  return new Uint8Array(plaintext);
}
