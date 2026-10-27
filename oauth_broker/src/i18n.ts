export type Locale = "en" | "it";

export type Messages = {
  titleSuccess: string;
  titleError: string;
  msgSuccess: string;
  msgRedirecting: string;
  btnContinue: string;
  msgNoRedirect: string;
  errorSessionExpired: string;
  errorInvalidRequest: string;
  errorInstanceNotFound: string;
  errorRedirectMismatch: string;
  errorTokenFailed: string;
  errorGeneric: string;
};

const CATALOG: Record<Locale, Messages> = {
  en: {
    titleSuccess: "Authorization successful",
    titleError: "Authorization failed",
    msgSuccess: "Your cloud storage has been linked successfully.",
    msgRedirecting: "You will be redirected to your Home Assistant shortly.",
    btnContinue: "Continue to Home Assistant",
    msgNoRedirect: "You can close this window and return to Home Assistant.",
    errorSessionExpired: "Session expired or not found. Please restart the authorization from Home Assistant.",
    errorInvalidRequest: "Invalid request. Missing code or session.",
    errorInstanceNotFound: "Instance not registered or expired. Please register again from Home Assistant.",
    errorRedirectMismatch: "Redirect URL does not match the registered instance.",
    errorTokenFailed: "Token exchange failed. Please try again.",
    errorGeneric: "An error occurred. Please try again from Home Assistant.",
  },
  it: {
    titleSuccess: "Autorizzazione completata",
    titleError: "Autorizzazione fallita",
    msgSuccess: "Il cloud storage è stato collegato con successo.",
    msgRedirecting: "Verrai reindirizzato a Home Assistant a breve.",
    btnContinue: "Continua su Home Assistant",
    msgNoRedirect: "Puoi chiudere questa finestra e tornare a Home Assistant.",
    errorSessionExpired: "Sessione scaduta o non trovata. Riavvia l'autorizzazione da Home Assistant.",
    errorInvalidRequest: "Richiesta non valida. Codice o sessione mancanti.",
    errorInstanceNotFound: "Istanza non registrata o scaduta. Registrala di nuovo da Home Assistant.",
    errorRedirectMismatch: "L'URL di reindirizzamento non corrisponde all'istanza registrata.",
    errorTokenFailed: "Scambio token fallito. Riprova.",
    errorGeneric: "Si è verificato un errore. Riprova da Home Assistant.",
  },
};

export function pickLocale(acceptLanguage: string | null | undefined): Locale {
  if (!acceptLanguage) return "en";
  const tags = acceptLanguage
    .split(",")
    .map((p) => p.split(";")[0]?.trim().toLowerCase() ?? "")
    .filter(Boolean);
  for (const tag of tags) {
    if (tag.startsWith("it")) return "it";
    if (tag.startsWith("en")) return "en";
  }
  // wildcard or first tag
  if (tags[0]?.startsWith("it")) return "it";
  return "en";
}

export function getMessages(locale: Locale): Messages {
  return CATALOG[locale];
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

export function renderHtmlPage(opts: {
  locale: Locale;
  success: boolean;
  redirectUrl?: string;
  errorMessage?: string;
  autoRedirectSeconds?: number;
}): string {
  const m = getMessages(opts.locale);
  const title = opts.success ? m.titleSuccess : m.titleError;
  const heading = esc(title);
  const bodyMsg = opts.success ? esc(m.msgSuccess) : esc(opts.errorMessage ?? m.errorGeneric);
  const redirecting = esc(m.msgRedirecting);
  const btn = esc(m.btnContinue);
  const langAttr = opts.locale;
  // CSP: allow inline style/script via nonce? Keep default-src none + style-src unsafe-inline for simplicity
  const redirectMeta = opts.success && opts.redirectUrl ? `<meta http-equiv="refresh" content="${opts.autoRedirectSeconds ?? 2};url=${esc(opts.redirectUrl)}">` : "";
  const jsRedirect =
    opts.success && opts.redirectUrl
      ? `<script>setTimeout(function(){location.href=${JSON.stringify(opts.redirectUrl)}}, ${(opts.autoRedirectSeconds ?? 2) * 1000});</script>`
      : "";
  const button =
    opts.success && opts.redirectUrl
      ? `<p><a href="${esc(opts.redirectUrl)}" class="btn">${btn}</a></p><p class="hint">${redirecting}</p>`
      : `<p class="hint">${esc(m.msgNoRedirect)}</p>`;
  const statusIcon = opts.success ? "✅" : "❌";
  return `<!doctype html><html lang="${langAttr}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">${redirectMeta}<title>${heading}</title>
<style>body{font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,sans-serif;background:#f5f5f5;margin:0;padding:24px;color:#222}
.card{max-width:560px;margin:48px auto;background:#fff;border-radius:12px;box-shadow:0 2px 12px rgba(0,0,0,.08);padding:28px 24px;text-align:center}
h1{font-size:20px;margin:8px 0 12px} .icon{font-size:36px} p{line-height:1.5} .btn{display:inline-block;margin-top:12px;padding:10px 18px;background:#03a9f4;color:#fff;border-radius:8px;text-decoration:none;font-weight:600}
.hint{color:#666;font-size:13px;margin-top:10px} .err{color:#b00020}</style>
</head><body><div class="card"><div class="icon">${statusIcon}</div><h1>${heading}</h1><p class="${opts.success ? "" : "err"}">${bodyMsg}</p>${opts.success ? button : ""}</div>${jsRedirect}</body></html>`;
}
