/**
 * goform — field definitions and renderer
 * Shared between builder (preview) and view (interactive)
 */
(function (root) {
  root.FieldTypes = [
    { type: 'short_text', label: 'Kısa metin',     icon: 'text',     group: 'Metin' },
    { type: 'long_text',  label: 'Uzun metin',     icon: 'paragraph',group: 'Metin' },
    { type: 'email',      label: 'E-posta',        icon: 'email',    group: 'Metin' },
    { type: 'url',        label: 'Web adresi',     icon: 'url',      group: 'Metin' },
    { type: 'phone',      label: 'Telefon',        icon: 'phone',    group: 'Metin' },
    { type: 'radio',      label: 'Çoktan seçmeli', icon: 'radio',    group: 'Seçim' },
    { type: 'checkbox',   label: 'Onay kutuları',  icon: 'checkbox', group: 'Seçim' },
    { type: 'dropdown',   label: 'Açılır menü',    icon: 'dropdown', group: 'Seçim' },
    { type: 'number',     label: 'Sayı',           icon: 'number',   group: 'Veri' },
    { type: 'date',       label: 'Tarih',          icon: 'date',     group: 'Veri' },
    { type: 'time',       label: 'Saat',           icon: 'time',     group: 'Veri' },
    { type: 'rating',     label: 'Yıldız puan',    icon: 'star',     group: 'Ölçek' },
    { type: 'scale',      label: 'Lineer ölçek',   icon: 'scale',    group: 'Ölçek' },
    { type: 'section',    label: 'Bölüm başlığı',  icon: 'section',  group: 'Düzen' },
  ];

  root.createField = function (type) {
    var id = 'f_' + Math.random().toString(36).slice(2, 10);
    var base = { id: id, type: type, label: '', description: '', required: false };
    var defaults = {
      short_text: { label: 'Kısa metin sorusu', placeholder: '' },
      long_text:  { label: 'Uzun metin sorusu', placeholder: '' },
      email:      { label: 'E-posta adresiniz', placeholder: 'ornek@mail.com' },
      url:        { label: 'Web sitesi', placeholder: 'https://...' },
      phone:      { label: 'Telefon numaranız', placeholder: '5xx xxx xx xx' },
      radio:      { label: 'Çoktan seçmeli soru', options: ['Seçenek 1', 'Seçenek 2'] },
      checkbox:   { label: 'Birden fazla seçenek seçilebilir', options: ['Seçenek 1', 'Seçenek 2'] },
      dropdown:   { label: 'Bir seçenek belirleyin', options: ['Seçenek 1', 'Seçenek 2'] },
      number:     { label: 'Sayı sorusu', placeholder: '', min: null, max: null },
      date:       { label: 'Bir tarih seçin' },
      time:       { label: 'Bir saat seçin' },
      rating:     { label: 'Lütfen değerlendirin', maxStars: 5 },
      scale:      { label: 'Bir değer seçin', min: 1, max: 10, minLabel: '', maxLabel: '' },
      section:    { label: 'Bölüm başlığı', description: 'İsteğe bağlı açıklama metni' },
    };
    return Object.assign(base, defaults[type] || {});
  };

  function esc(s) { return root.esc(s); }

  root.renderFieldPreview = function (field, opts) {
    opts = opts || {};
    var interactive = !!opts.interactive;
    var value = opts.value == null ? '' : opts.value;
    var disabled = interactive ? '' : 'disabled';

    switch (field.type) {
      case 'short_text':
        return '<input type="text" class="input" ' + disabled + ' placeholder="' + esc(field.placeholder || 'Cevabınız') + '" value="' + esc(value) + '" data-field="' + field.id + '">';
      case 'long_text':
        return '<textarea class="textarea" ' + disabled + ' placeholder="' + esc(field.placeholder || 'Cevabınız') + '" data-field="' + field.id + '">' + esc(value) + '</textarea>';
      case 'email':
        return '<input type="email" class="input" autocomplete="email" ' + disabled + ' placeholder="' + esc(field.placeholder || 'ornek@mail.com') + '" value="' + esc(value) + '" data-field="' + field.id + '">';
      case 'url':
        return '<input type="url" class="input" ' + disabled + ' placeholder="' + esc(field.placeholder || 'https://...') + '" value="' + esc(value) + '" data-field="' + field.id + '">';
      case 'phone':
        return '<input type="tel" class="input" autocomplete="tel" ' + disabled + ' placeholder="' + esc(field.placeholder || '') + '" value="' + esc(value) + '" data-field="' + field.id + '">';
      case 'number':
        var minAttr = field.min != null ? 'min="' + field.min + '"' : '';
        var maxAttr = field.max != null ? 'max="' + field.max + '"' : '';
        return '<input type="number" class="input" ' + disabled + ' placeholder="' + esc(field.placeholder || '') + '" value="' + esc(value) + '" ' + minAttr + ' ' + maxAttr + ' data-field="' + field.id + '">';
      case 'date':
        return '<input type="date" class="input" ' + disabled + ' value="' + esc(value) + '" data-field="' + field.id + '">';
      case 'time':
        return '<input type="time" class="input" ' + disabled + ' value="' + esc(value) + '" data-field="' + field.id + '">';
      case 'radio':
        return '<div class="choices">' + (field.options || []).map(function (opt) {
          return '<label class="choice-row" style="cursor:' + (interactive ? 'pointer' : 'default') + '">' +
            '<input type="radio" name="' + field.id + '" value="' + esc(opt) + '" ' + disabled + (value === opt ? ' checked' : '') + ' data-field="' + field.id + '">' +
            '<span>' + esc(opt) + '</span></label>';
        }).join('') + '</div>';
      case 'checkbox':
        var arr = Array.isArray(value) ? value : [];
        return '<div class="choices">' + (field.options || []).map(function (opt) {
          return '<label class="choice-row" style="cursor:' + (interactive ? 'pointer' : 'default') + '">' +
            '<input type="checkbox" name="' + field.id + '" value="' + esc(opt) + '" ' + disabled + (arr.indexOf(opt) >= 0 ? ' checked' : '') + ' data-field="' + field.id + '">' +
            '<span>' + esc(opt) + '</span></label>';
        }).join('') + '</div>';
      case 'dropdown':
        return '<select class="select" ' + disabled + ' data-field="' + field.id + '">' +
          '<option value="">Seçim yapın...</option>' +
          (field.options || []).map(function (opt) {
            return '<option value="' + esc(opt) + '"' + (value === opt ? ' selected' : '') + '>' + esc(opt) + '</option>';
          }).join('') + '</select>';
      case 'rating': {
        var max = field.maxStars || 5;
        var current = Number(value) || 0;
        var html = '<div class="rating" data-field="' + field.id + '" data-value="' + current + '">';
        for (var i = 1; i <= max; i++) {
          html += '<div class="star' + (i <= current ? ' active' : '') + '" data-star="' + i + '">' + Icons.star({ size: 24, stroke: 1.5 }) + '</div>';
        }
        return html + '</div>';
      }
      case 'scale': {
        var smin = field.min == null ? 1 : field.min;
        var smax = field.max == null ? 10 : field.max;
        var sval = value;
        var sh = '<div class="scale" data-field="' + field.id + '">';
        if (field.minLabel || field.maxLabel) {
          sh += '<div class="scale-row"><span>' + esc(field.minLabel || '') + '</span><span>' + esc(field.maxLabel || '') + '</span></div>';
        }
        sh += '<div class="scale-buttons">';
        for (var j = smin; j <= smax; j++) {
          sh += '<div class="scale-btn' + (sval == j ? ' active' : '') + '" data-val="' + j + '">' + j + '</div>';
        }
        return sh + '</div></div>';
      }
      case 'section':
        return '';
      default:
        return '<div class="text-muted">Bilinmeyen alan tipi</div>';
    }
  };

  root.attachFieldInteractive = function (rootEl, field, setVal) {
    if (field.type === 'rating') {
      var wrap = rootEl.querySelector('[data-field="' + field.id + '"]');
      if (!wrap) return;
      wrap.querySelectorAll('.star').forEach(function (starEl) {
        starEl.addEventListener('click', function () {
          var v = Number(starEl.dataset.star);
          setVal(field.id, v);
          wrap.dataset.value = v;
          wrap.querySelectorAll('.star').forEach(function (s) {
            s.classList.toggle('active', Number(s.dataset.star) <= v);
          });
        });
      });
    } else if (field.type === 'scale') {
      var swrap = rootEl.querySelector('[data-field="' + field.id + '"]');
      if (!swrap) return;
      swrap.querySelectorAll('.scale-btn').forEach(function (btn) {
        btn.addEventListener('click', function () {
          var v = Number(btn.dataset.val);
          setVal(field.id, v);
          swrap.querySelectorAll('.scale-btn').forEach(function (b) { b.classList.toggle('active', b === btn); });
        });
      });
    } else if (field.type === 'checkbox') {
      rootEl.querySelectorAll('[data-field="' + field.id + '"]').forEach(function (input) {
        input.addEventListener('change', function () {
          var checked = Array.from(rootEl.querySelectorAll('[data-field="' + field.id + '"]:checked')).map(function (i) { return i.value; });
          setVal(field.id, checked);
        });
      });
    } else if (field.type === 'radio') {
      rootEl.querySelectorAll('[data-field="' + field.id + '"]').forEach(function (input) {
        input.addEventListener('change', function () {
          if (input.checked) setVal(field.id, input.value);
        });
      });
    } else {
      var ip = rootEl.querySelector('[data-field="' + field.id + '"]');
      if (!ip) return;
      ip.addEventListener('input', function () { setVal(field.id, ip.value); });
      ip.addEventListener('change', function () { setVal(field.id, ip.value); });
    }
  };

  root.validateField = function (field, val) {
    if (field.required) {
      var empty = val === undefined || val === null || val === '';
      if (Array.isArray(val) && val.length === 0) empty = true;
      if (empty) return 'Bu alan zorunludur.';
    }
    if (val === undefined || val === null || val === '') return null;
    if (field.type === 'email') {
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(val)) return 'Geçerli bir e-posta adresi girin.';
    }
    if (field.type === 'url') {
      if (!/^https?:\/\/.+/i.test(val)) return 'Geçerli bir URL girin (http:// veya https:// ile başlamalı).';
    }
    if (field.type === 'number') {
      var n = Number(val);
      if (isNaN(n)) return 'Geçerli bir sayı girin.';
      if (field.min != null && n < field.min) return 'En az ' + field.min + ' olmalı.';
      if (field.max != null && n > field.max) return 'En fazla ' + field.max + ' olmalı.';
    }
    return null;
  };
})(window);
