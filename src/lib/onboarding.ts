const KEY = "cleansign_pending_consent";

export type PendingConsent = {
  terms: boolean;
  privacy: boolean;
  marketing: boolean;
};

export function savePendingConsent(consent: PendingConsent) {
  sessionStorage.setItem(KEY, JSON.stringify(consent));
}

export function getPendingConsent(): PendingConsent | null {
  const raw = sessionStorage.getItem(KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as PendingConsent;
  } catch {
    return null;
  }
}

export function clearPendingConsent() {
  sessionStorage.removeItem(KEY);
}
