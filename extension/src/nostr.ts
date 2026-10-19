// Thin wrapper over nostr-tools — the only place this extension does
// any Nostr-specific cryptography. Per A012's own Decisions: never
// hand-rolled, nostr-tools is the de facto reference implementation
// (built on @noble/curves and @noble/hashes) essentially every other
// NIP-07 extension already relies on for the same BIP340
// Schnorr-signing and NIP-01 event-id computation persona's own Go
// side (go-nostr) does independently.

import { generateSecretKey, getPublicKey as nostrToolsGetPublicKey, finalizeEvent } from "nostr-tools/pure";
import { nip19 } from "nostr-tools";

export interface UnsignedEventTemplate {
  kind: number;
  created_at: number;
  tags: string[][];
  content: string;
}

export interface SignedEvent extends UnsignedEventTemplate {
  id: string;
  pubkey: string;
  sig: string;
}

export function hexToBytes(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  }
  return bytes;
}

export function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

export function generatePersonaSecret(): Uint8Array {
  return generateSecretKey();
}

export function getPublicKey(skHex: string): string {
  return nostrToolsGetPublicKey(hexToBytes(skHex));
}

export function signEvent(skHex: string, template: UnsignedEventTemplate): SignedEvent {
  return finalizeEvent(template, hexToBytes(skHex)) as SignedEvent;
}

/** Decodes a bech32 nsec to its raw hex secret key, or throws. */
export function decodeNsec(nsec: string): string {
  const decoded = nip19.decode(nsec.trim());
  if (decoded.type !== "nsec") {
    throw new Error(`expected an nsec, got a ${decoded.type}`);
  }
  return bytesToHex(decoded.data as Uint8Array);
}

export function encodeNsec(skHex: string): string {
  return nip19.nsecEncode(hexToBytes(skHex));
}

export function encodeNpub(pubkeyHex: string): string {
  return nip19.npubEncode(pubkeyHex);
}
