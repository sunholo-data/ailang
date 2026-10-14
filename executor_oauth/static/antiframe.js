/*
 * Anti-clickjacking guard for the OAuth sign-in/consent page (RFC 9700 §4.16:
 * an authorization server must stop its consent page being framed).
 *
 * serve-api cannot send X-Frame-Options / CSP frame-ancestors yet (ailang
 * #1597), so the page defends itself with the OWASP pattern: oauth-login.html
 * ships with <html class="af-hidden"> (nothing rendered), and this script,
 * loaded synchronously in <head>, reveals the page only when it is the
 * top-level window. Framed, it stays blank and tries to break out. A sandboxed
 * frame that blocks scripts also stays blank, because the hiding is CSS.
 * External (not inline) so a later CSP without 'unsafe-inline' keeps working.
 */
(function () {
  var framed;
  try { framed = window.self !== window.top; } catch (e) { framed = true; }
  window.AX_FRAMED = framed;
  if (!framed) {
    document.documentElement.className = document.documentElement.className.replace(/\baf-hidden\b/, '').trim();
  } else {
    try { window.top.location = window.self.location.href; } catch (e) { /* blocked: stay hidden */ }
  }
})();
