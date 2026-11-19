/**
 * Sign-in page for the hosted AILANG executor's MCP OAuth (static/
 * oauth-login.html), served by the executor itself with serve-api --static, so
 * every request below is same-origin. Adapted from AILANG Parse's page.
 *
 * Flow: GET /oauth/login-config (which Firebase project) -> GET /oauth/request
 * (who is asking) -> Google sign-in popup (or email + password, below) ->
 * consent -> top-level form POST of the ID token to /oauth/login/complete,
 * whose 302 returns the user to the app with an authorization code.
 *
 * Email + password is for accounts created by hand in the Firebase console
 * (directory reviewers need a login with no MFA, emailed codes or magic links:
 * m-executor-directory-submission-kit.md). The page only ever SIGNS IN with it;
 * it never creates an account or sends a password reset, so there is no self
 * sign-up here. It needs the Email/Password provider enabled in the
 * Firebase project (console only), and the server's gate still requires a
 * verified email address, so a hand-made account must be marked verified.
 *
 * Never log the handle or the ID token: the handle is a one-time credential
 * for this sign-in, and the token is the user's.
 */
(function () {
  'use strict';

  // Public Firebase browser configs, keyed by the project id that
  // /oauth/login-config returns; an unknown id is refused. The key's referrer
  // list must admit this service's host (ailang-multivac terraform
  // allowed_referrers) and the host must be a Firebase authorized domain.
  // Each apiKey is that project's "firebase-browser-docparse" key, the one
  // AILANG Parse's sign-in page uses too. Test has no executor service, so no
  // entry.
  var FIREBASE = {
    'ailang-multivac-dev': {
      apiKey: 'AIzaSyDfYyF0UR5OaUGe9DWIx2nzcy625_txN1o',
      authDomain: 'ailang-multivac-dev.firebaseapp.com',
      projectId: 'ailang-multivac-dev'
    },
    'ailang-multivac': {
      apiKey: 'AIzaSyBqHyoveqcGyL7LLnlvQ2FtHxdy6evXbHo',
      authDomain: 'ailang-multivac.firebaseapp.com',
      projectId: 'ailang-multivac'
    }
  };

  var handle = new URLSearchParams(window.location.search).get('handle') || '';
  var auth = null;
  var request = null;
  var user = null;
  var submitting = false;

  function $(id) { return document.getElementById(id); }
  function setText(id, text) { $(id).textContent = text; }

  function showStatus(msg, isError) {
    var el = $('status');
    el.textContent = msg;
    el.className = 'status visible ' + (isError ? 'error' : 'success');
  }

  // Values from the request come from a document the client publishes:
  // textContent only, never markup.
  function addRow(label, value) {
    var row = document.createElement('div');
    row.className = 'row';
    var l = document.createElement('span'); l.className = 'label'; l.textContent = label;
    var v = document.createElement('span'); v.className = 'value'; v.textContent = value;
    row.appendChild(l); row.appendChild(v);
    $('trust-rows').appendChild(row);
  }

  function render() {
    var prompt = $('signin-prompt');
    var form = $('approve-form');
    if (!request || !auth) { prompt.style.display = 'none'; form.style.display = 'none'; return; }
    if (user) {
      prompt.style.display = 'none';
      form.style.display = 'block';
      setText('signed-in-as', 'Signed in as ' + (user.email || user.displayName || user.uid));
      setText('allow-btn', 'Allow ' + (request.display || request.client_host));
      // The server refuses an unverified address (executor_oauth/oauth.ail
      // gate); say so here instead of letting Allow end on its error page.
      if (user.emailVerified === false) {
        $('allow-btn').disabled = true;
        showStatus('This account\'s email address is not verified, so the executor will not accept it. ' +
          'Sign in with a different account.', true);
      } else {
        $('allow-btn').disabled = submitting;
      }
    } else {
      prompt.style.display = 'block';
      form.style.display = 'none';
    }
  }

  function startSignIn() {
    // Popup, not redirect: the page is not on the firebaseapp.com auth domain,
    // and the redirect flow breaks where third-party storage is blocked.
    var provider = new firebase.auth.GoogleAuthProvider();
    provider.setCustomParameters({ prompt: 'select_account' });
    auth.signInWithPopup(provider).catch(function (e) {
      if (e && e.code === 'auth/popup-closed-by-user') return;
      showStatus('Sign-in failed. Allow pop-ups for this page and try again.', true);
    });
  }

  // Email + password: sign in only (see the header). Firebase's messages can
  // say whether an address has an account; ours never do.
  function emailSignIn(ev) {
    ev.preventDefault();
    if (!auth) return;
    var email = $('email-input').value.trim();
    var password = $('password-input').value;
    if (!email || !password) return;
    $('email-btn').disabled = true;
    auth.signInWithEmailAndPassword(email, password).then(function () {
      $('password-input').value = '';
      $('status').className = 'status';
    }).catch(function (e) {
      var code = e && e.code;
      if (code === 'auth/operation-not-allowed') {
        showStatus('Email sign-in is not enabled on this server. Sign in with Google.', true);
      } else if (code === 'auth/too-many-requests') {
        showStatus('Too many attempts. Wait a few minutes and try again.', true);
      } else if (code === 'auth/network-request-failed') {
        showStatus('Could not reach the sign-in service. Check your connection and try again.', true);
      } else {
        showStatus('That email and password did not match an account.', true);
      }
    }).then(function () { $('email-btn').disabled = false; });
  }

  function allow() {
    if (window.AX_FRAMED !== false) return;  // never consent from inside a frame
    if (submitting || !request || !user) return;
    submitting = true;
    $('allow-btn').disabled = true;
    user.getIdToken().then(function (idToken) {
      var form = $('complete-form');
      form.action = '/oauth/login/complete';
      $('f-handle').value = handle;
      $('f-id-token').value = idToken;
      form.submit();
    }).catch(function () {
      submitting = false;
      $('allow-btn').disabled = false;
      showStatus('Could not read your sign-in. Sign in again and retry.', true);
    });
  }

  function deny() {
    request = null;
    render();
    showStatus('Not connected. You can close this tab.', false);
  }

  function getJson(url) {
    return fetch(url, { referrerPolicy: 'no-referrer', credentials: 'omit' })
      .then(function (r) { return r.json(); });
  }

  function init() {
    // Never run the consent flow inside a frame (antiframe.js keeps the page
    // hidden too; this stops the flow even if the CSS were overridden).
    if (window.AX_FRAMED !== false) return;
    $('signin-btn').addEventListener('click', startSignIn);
    $('email-form').addEventListener('submit', emailSignIn);
    $('allow-btn').addEventListener('click', allow);
    $('cancel-btn').addEventListener('click', deny);
    $('switch-user').addEventListener('click', function () {
      $('status').className = 'status';
      if (auth) auth.signOut();
    });

    if (!handle) {
      setText('consent-text', 'This link is incomplete. Start the connection again from the app.');
      return;
    }
    if (typeof firebase === 'undefined') {
      setText('consent-text', 'The sign-in library did not load. Check your connection and reload.');
      return;
    }

    Promise.all([
      getJson('/oauth/login-config'),
      getJson('/oauth/request?handle=' + encodeURIComponent(handle))
    ]).then(function (results) {
      var cfg = FIREBASE[results[0] && results[0].project_id];
      var info = results[1];
      if (!cfg) {
        setText('consent-text', 'Sign-in is not configured for this server.');
        return;
      }
      if (!info || info.error || !info.client_host) {
        setText('consent-text',
          'This sign-in request has expired or was already used. Start the connection again from the app.');
        return;
      }
      request = info;
      // A registered client's id is an opaque mco_ string: name it by where the
      // code goes instead. A DCR client registered itself, so anyone could have
      // created it (sunholo/mcp_oauth AGENT.md): say it is not verified.
      var registered = request.registered === 'dcr' || request.registered === 'operator';
      request.display = registered ? request.redirect_host : request.client_host;
      var unverified = request.registered === 'dcr'
        ? ' This app registered itself and is not verified by AILANG.' : '';
      setText('consent-text',
        request.display + ' wants to run AILANG code for you in the hosted sandbox. ' +
        'Only continue if you started this from that app.' + unverified);
      addRow('App', request.display + (unverified ? ' (unverified)' : ''));
      addRow('Returns to', request.redirect_host);
      $('trust-panel').style.display = 'block';

      firebase.initializeApp({ apiKey: cfg.apiKey, authDomain: cfg.authDomain, projectId: cfg.projectId });
      auth = firebase.auth();
      auth.onAuthStateChanged(function (u) { user = u; render(); });
      render();
    }).catch(function () {
      showStatus('Could not load the sign-in request. Check your connection and reload.', true);
    });
  }

  document.addEventListener('DOMContentLoaded', init);
})();
