// Storage model, split deliberately across two browser storage areas:
//
//   - chrome.storage.local: persisted to disk, holds only what's safe
//     to persist encrypted-at-rest — each persona's label, pubkey,
//     and its passphrase-encrypted secret key (crypto.ts's
//     EncryptedBlob). Also the per-origin active-persona mapping
//     (A012's own multi-persona Decision: never an implicit
//     cross-site default).
//   - chrome.storage.session: cleared when the browser closes, never
//     written to disk. Holds the *decrypted* secret keys for personas
//     unlocked this session — the "unlock once per session" UX this
//     action's Decisions commit to, without ever writing a decrypted
//     key to disk.

import type { EncryptedBlob } from "./crypto";

export interface StoredPersona {
  label: string;
  pubkeyHex: string;
  encrypted: EncryptedBlob;
}

interface LocalState {
  personas: Record<string, StoredPersona>; // keyed by pubkeyHex
  originPersona: Record<string, string>; // origin -> pubkeyHex
}

const LOCAL_KEY = "personaState";

async function readLocal(): Promise<LocalState> {
  const result = await chrome.storage.local.get(LOCAL_KEY);
  const state = result[LOCAL_KEY] as LocalState | undefined;
  return state ?? { personas: {}, originPersona: {} };
}

async function writeLocal(state: LocalState): Promise<void> {
  await chrome.storage.local.set({ [LOCAL_KEY]: state });
}

export async function listPersonas(): Promise<StoredPersona[]> {
  const state = await readLocal();
  return Object.values(state.personas);
}

export async function savePersona(persona: StoredPersona): Promise<void> {
  const state = await readLocal();
  state.personas[persona.pubkeyHex] = persona;
  await writeLocal(state);
}

export async function deletePersona(pubkeyHex: string): Promise<void> {
  const state = await readLocal();
  delete state.personas[pubkeyHex];
  for (const origin of Object.keys(state.originPersona)) {
    if (state.originPersona[origin] === pubkeyHex) delete state.originPersona[origin];
  }
  await writeLocal(state);
  await clearUnlocked(pubkeyHex);
}

export async function getActivePersonaForOrigin(origin: string): Promise<string | undefined> {
  const state = await readLocal();
  return state.originPersona[origin];
}

export async function setActivePersonaForOrigin(origin: string, pubkeyHex: string): Promise<void> {
  const state = await readLocal();
  state.originPersona[origin] = pubkeyHex;
  await writeLocal(state);
}

// --- Session-only decrypted key material ---

const SESSION_KEY = "unlockedSecrets";

async function readSession(): Promise<Record<string, string>> {
  const result = await chrome.storage.session.get(SESSION_KEY);
  return (result[SESSION_KEY] as Record<string, string> | undefined) ?? {};
}

/** Holds a decrypted secret key (hex) in memory for this browser session only. */
export async function setUnlocked(pubkeyHex: string, skHex: string): Promise<void> {
  const secrets = await readSession();
  secrets[pubkeyHex] = skHex;
  await chrome.storage.session.set({ [SESSION_KEY]: secrets });
}

export async function getUnlocked(pubkeyHex: string): Promise<string | undefined> {
  const secrets = await readSession();
  return secrets[pubkeyHex];
}

export async function clearUnlocked(pubkeyHex: string): Promise<void> {
  const secrets = await readSession();
  delete secrets[pubkeyHex];
  await chrome.storage.session.set({ [SESSION_KEY]: secrets });
}

export async function lockAll(): Promise<void> {
  await chrome.storage.session.remove(SESSION_KEY);
}
