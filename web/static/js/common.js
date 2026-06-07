/**
 * goform — shared utilities used across all pages
 */
(function (root) {
  /* ===== HTML escape ===== */
  function esc(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, function (m) {
      return { '&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&quot;', "'":'&#39;' }[m];
    });
  }
  root.esc = esc;
  root.escapeHtml = esc;

  /* ===== HTTP helper ===== */
  async function api(path, opts) {
    opts = opts || {};
    var headers = Object.assign({ 'Content-Type': 'application/json' }, opts.headers || {});
    var res = await fetch(path, {
      method: opts.method || 'GET',
      headers: headers,
      body: opts.body ? JSON.stringify(opts.body) : undefined,
      credentials: 'same-origin',
    });
    var text = await res.text();
    var data = null;
    try { data = text ? JSON.parse(text) : null; } catch (e) { data = text; }
    if (!res.ok) {
      var msg = (data && data.error) ? data.error : ('HTTP ' + res.status);
      var err = new Error(msg);
      err.status = res.status;
      err.data = data;
      throw err;
    }
    return data;
  }
  root.api = api;

  /* ===== Toast ===== */
  root.toast = function (msg, type) {
    var el = document.createElement('div');
    el.className = 'toast' + (type ? ' ' + type : '');
    var icon = type === 'error' ? Icons.warning({ size: 14 }) : Icons.check({ size: 14 });
    el.innerHTML = icon + ' ' + esc(msg);
    document.body.appendChild(el);
    setTimeout(function () { el.remove(); }, 3000);
  };

  /* ===== Modal helpers ===== */
  root.modal = function (opts) {
    var backdrop = document.createElement('div');
    backdrop.className = 'modal-backdrop';
    backdrop.innerHTML =
      '<div class="modal' + (opts.lg ? ' lg' : '') + '">' +
        '<div class="modal-header">' +
          '<div><h3>' + esc(opts.title || '') + '</h3>' +
          (opts.subtitle ? '<p>' + esc(opts.subtitle) + '</p>' : '') +
          '</div>' +
          '<button class="modal-close" data-modal-close>' + Icons.x({ size: 16 }) + '</button>' +
        '</div>' +
        '<div class="modal-body">' + (opts.body || '') + '</div>' +
        (opts.footer ? '<div class="modal-footer">' + opts.footer + '</div>' : '') +
      '</div>';
    document.body.appendChild(backdrop);
    backdrop.addEventListener('click', function (e) {
      if (e.target === backdrop || e.target.closest('[data-modal-close]')) {
        close();
      }
    });
    function close() {
      backdrop.remove();
      if (opts.onClose) opts.onClose();
    }
    if (opts.onMount) opts.onMount(backdrop, close);
    if (root.attachPasswordToggles) root.attachPasswordToggles(backdrop);
    return { el: backdrop, close: close };
  };

  /* ===== Time formatting (Turkish) ===== */
  root.timeAgo = function (ts) {
    var ms = Number(ts) * 1000;
    if (ts > 1e12) ms = ts;
    var diff = Math.floor((Date.now() - ms) / 1000);
    if (diff < 5) return 'az önce';
    if (diff < 60) return diff + ' sn önce';
    if (diff < 3600) return Math.floor(diff / 60) + ' dk önce';
    if (diff < 86400) return Math.floor(diff / 3600) + ' sa önce';
    if (diff < 604800) return Math.floor(diff / 86400) + ' gün önce';
    return new Date(ms).toLocaleDateString('tr-TR', { day: 'numeric', month: 'short', year: 'numeric' });
  };

  root.formatDateTime = function (ts) {
    var ms = Number(ts) * 1000;
    if (ts > 1e12) ms = ts;
    return new Date(ms).toLocaleString('tr-TR', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
  };

  /* ===== Theme ===== */
  root.theme = {
    get: function () { return localStorage.getItem('goform-theme') || 'system'; },
    set: function (val) {
      localStorage.setItem('goform-theme', val);
      this.apply();
      // Notify any open theme switches
      document.querySelectorAll('[data-theme-switch]').forEach(function (sw) {
        sw.querySelectorAll('button').forEach(function (b) {
          b.classList.toggle('active', b.dataset.themeVal === val);
        });
      });
    },
    apply: function () {
      var v = this.get();
      if (v === 'system') {
        document.documentElement.removeAttribute('data-theme');
      } else {
        document.documentElement.setAttribute('data-theme', v);
      }
    },
    cycle: function () {
      var cur = this.get();
      var next = cur === 'system' ? 'light' : (cur === 'light' ? 'dark' : 'system');
      this.set(next);
    },
    currentIcon: function () {
      var v = this.get();
      if (v === 'light') return Icons.sun({ size: 14 });
      if (v === 'dark') return Icons.moon({ size: 14 });
      return Icons.monitor({ size: 14 });
    },
  };
  root.theme.apply();

  /* ===== Password toggle ===== */
  root.attachPasswordToggles = function (parent) {
    (parent || document).querySelectorAll('[data-toggle-pw]').forEach(function (btn) {
      if (btn.dataset.bound) return;
      btn.dataset.bound = '1';
      btn.innerHTML = Icons.eye({ size: 14 });
      btn.addEventListener('click', function (e) {
        e.preventDefault();
        var wrap = btn.closest('.input-wrap');
        var input = wrap && wrap.querySelector('input');
        if (!input) return;
        if (input.type === 'password') {
          input.type = 'text';
          btn.innerHTML = Icons.eyeOff({ size: 14 });
          btn.title = 'Şifreyi gizle';
        } else {
          input.type = 'password';
          btn.innerHTML = Icons.eye({ size: 14 });
          btn.title = 'Şifreyi göster';
        }
      });
      btn.title = 'Şifreyi göster';
    });
  };

  /* Auto-attach on DOM ready and after modal opens */
  if (document.readyState !== 'loading') root.attachPasswordToggles();
  else document.addEventListener('DOMContentLoaded', function () { root.attachPasswordToggles(); });

  /* ===== Password field helper ===== */
  root.passwordField = function (id, opts) {
    opts = opts || {};
    return '<div class="input-wrap">' +
      '<input type="password" id="' + id + '" class="input"' +
        (opts.autocomplete ? ' autocomplete="' + opts.autocomplete + '"' : '') +
        (opts.placeholder ? ' placeholder="' + esc(opts.placeholder) + '"' : '') +
        (opts.minlength ? ' minlength="' + opts.minlength + '"' : '') +
        (opts.required ? ' required' : '') +
        (opts.autofocus ? ' autofocus' : '') +
      '>' +
      '<button type="button" class="input-action" data-toggle-pw aria-label="Şifreyi göster"></button>' +
    '</div>';
  };

  /* ===== Common header ===== */
  root.renderNav = function (opts) {
    opts = opts || {};
    var user = opts.user;
    var html = '<nav class="nav">' +
      '<a href="' + (user ? '/dashboard' : '/') + '" class="brand">' +
        '<img src="/static/img/logo.svg" alt="" class="logo">' +
        '<span class="brand-wordmark">goform</span>' +
      '</a>' +
      (opts.center || '') +
      '<div class="nav-spacer"></div>' +
      '<div class="nav-actions">' +
      (opts.actions || '');

    if (user) {
      // Theme switch
      var cur = theme.get();
      html += '<div class="theme-switch" data-theme-switch title="Tema">' +
        '<button data-theme-val="light" title="Açık"' + (cur === 'light' ? ' class="active"' : '') + '>' + Icons.sun({ size: 13 }) + '</button>' +
        '<button data-theme-val="dark" title="Koyu"' + (cur === 'dark' ? ' class="active"' : '') + '>' + Icons.moon({ size: 13 }) + '</button>' +
        '<button data-theme-val="system" title="Sistem"' + (cur === 'system' ? ' class="active"' : '') + '>' + Icons.monitor({ size: 13 }) + '</button>' +
      '</div>';

      var initials = (user.name || user.email || '?').slice(0, 1).toUpperCase();
      html += '<div class="user-menu">' +
        '<button class="user-trigger" data-user-trigger>' +
          '<div class="user-avatar">' + esc(initials) + '</div>' +
          '<span class="user-name">' + esc(user.name || user.email) + '</span>' +
          Icons.chevronDown({ size: 14 }) +
        '</button>' +
        '<div class="user-menu-pop" data-user-pop>' +
          '<div class="head">' +
            '<div class="who">' + esc(user.name) + (user.role === 'admin' ? ' <span class="badge admin">admin</span>' : '') + '</div>' +
            '<div class="em">' + esc(user.email) + '</div>' +
          '</div>' +
          '<a class="item" href="/dashboard">' + Icons.layout({ size: 14 }) + ' Panel</a>' +
          '<a class="item" href="/settings">' + Icons.settings({ size: 14 }) + ' Ayarlar</a>' +
          '<button class="item danger" data-logout>' + Icons.logout({ size: 14 }) + ' Çıkış yap</button>' +
        '</div>' +
      '</div>';
    }
    html += '</div></nav>';
    document.body.insertAdjacentHTML('afterbegin', html);

    // Wire up theme switch
    document.querySelectorAll('[data-theme-switch] button').forEach(function (b) {
      b.addEventListener('click', function (e) {
        e.stopPropagation();
        theme.set(b.dataset.themeVal);
      });
    });

    // Wire up user menu
    var trigger = document.querySelector('[data-user-trigger]');
    var pop = document.querySelector('[data-user-pop]');
    if (trigger) {
      trigger.addEventListener('click', function (e) {
        e.stopPropagation();
        pop.classList.toggle('open');
      });
      document.addEventListener('click', function () { pop.classList.remove('open'); });
    }
    var logoutBtn = document.querySelector('[data-logout]');
    if (logoutBtn) {
      logoutBtn.addEventListener('click', async function () {
        await api('/api/auth/logout', { method: 'POST' });
        location.href = '/login';
      });
    }
  };

  /* ===== Powered by footer ===== */
  root.renderPoweredBy = function () {
    var html = '<div class="poweredby">' +
      '<a href="https://github.com/benyigiteren/goform" target="_blank" rel="noopener" class="pb-logo">' +
        Icons.sparkles({ size: 12 }) +
        '<span>powered by goform</span>' +
      '</a></div>';
    return html;
  };

  /* ===== Confirm dialog ===== */
  root.confirmDialog = function (opts) {
    return new Promise(function (resolve) {
      var m = modal({
        title: opts.title || 'Onayla',
        subtitle: opts.subtitle || '',
        body: '<p style="color:var(--text-2)">' + esc(opts.message || '') + '</p>',
        footer:
          '<button class="btn" data-modal-close>İptal</button>' +
          '<button class="btn ' + (opts.danger ? 'btn-danger btn-primary' : 'btn-primary') + '" data-confirm>' +
            esc(opts.confirmText || 'Onayla') + '</button>',
        onMount: function (el) {
          el.querySelector('[data-confirm]').onclick = function () {
            m.close();
            resolve(true);
          };
        },
        onClose: function () { resolve(false); },
      });
    });
  };
})(window);
