/**
 * goform — ready-made form templates
 */
(function (root) {
  function id() { return 'f_' + Math.random().toString(36).slice(2, 10); }

  root.Templates = [
    {
      key: 'blank',
      title: 'Boş form',
      description: 'Sıfırdan kendi formunuzu oluşturun.',
      icon: 'plus',
      color: 'indigo',
      build: function () {
        return { title: 'İsimsiz form', description: '', theme: { color: 'indigo', icon: '', mode: 'light' }, fields: [] };
      },
    },
    {
      key: 'satisfaction',
      title: 'Müşteri Memnuniyet Anketi',
      description: 'Hizmet kalitenizi puan ve yorumlarla ölçün.',
      icon: 'star',
      color: 'amber',
      build: function () {
        return {
          title: 'Müşteri Memnuniyet Anketi',
          description: 'Görüşleriniz bizim için çok değerli. Lütfen birkaç dakikanızı ayırın.',
          theme: { color: 'amber', icon: '⭐', mode: 'light' },
          fields: [
            { id: id(), type: 'short_text', label: 'Adınız Soyadınız', required: false, placeholder: 'İsteğe bağlı' },
            { id: id(), type: 'email', label: 'E-posta adresiniz', required: false, placeholder: 'sizin@mail.com' },
            { id: id(), type: 'rating', label: 'Bizi nasıl değerlendirirsiniz?', required: true, maxStars: 5 },
            { id: id(), type: 'scale', label: 'Bizi tavsiye etme olasılığınız', required: true, min: 0, max: 10, minLabel: 'Asla', maxLabel: 'Kesinlikle' },
            { id: id(), type: 'radio', label: 'Bizi nereden duydunuz?', required: false, options: ['Google', 'Sosyal medya', 'Arkadaş tavsiyesi', 'Reklam', 'Diğer'] },
            { id: id(), type: 'checkbox', label: 'Hangi hizmetlerimizi kullandınız?', required: false, options: ['Hızlı destek', 'Online sipariş', 'Ürün danışmanlığı', 'Teknik servis'] },
            { id: id(), type: 'long_text', label: 'Eklemek istediğiniz başka bir şey var mı?', required: false, placeholder: 'Yorumlarınız...' },
          ],
        };
      },
    },
    {
      key: 'event',
      title: 'Etkinlik Kayıt',
      description: 'Davetli takibi ve menü tercihleri için.',
      icon: 'date',
      color: 'rose',
      build: function () {
        return {
          title: 'Etkinlik Kayıt Formu',
          description: 'Katılımınızı onaylamak için bu formu doldurun. Yerimiz sınırlıdır.',
          theme: { color: 'rose', icon: '🎉', mode: 'light' },
          fields: [
            { id: id(), type: 'short_text', label: 'Ad Soyad', required: true },
            { id: id(), type: 'email', label: 'E-posta', required: true },
            { id: id(), type: 'phone', label: 'Telefon', required: false },
            { id: id(), type: 'radio', label: 'Katılım durumu', required: true, options: ['Katılacağım', 'Maalesef katılamayacağım'] },
            { id: id(), type: 'number', label: 'Kaç kişi geleceksiniz (kendiniz dahil)?', required: false, min: 1, max: 10 },
            { id: id(), type: 'dropdown', label: 'Yemek tercihi', required: false, options: ['Standart', 'Vejetaryen', 'Vegan', 'Glutensiz'] },
            { id: id(), type: 'long_text', label: 'Özel istekleriniz / alerjiler', required: false },
          ],
        };
      },
    },
    {
      key: 'job',
      title: 'İş Başvurusu',
      description: 'CV ve önemli bilgileri tek formda toplayın.',
      icon: 'file',
      color: 'emerald',
      build: function () {
        return {
          title: 'İş Başvuru Formu',
          description: 'Ekibimize katılmak istediğiniz için teşekkür ederiz. Lütfen aşağıdaki alanları doldurun.',
          theme: { color: 'emerald', icon: '💼', mode: 'light' },
          fields: [
            { id: id(), type: 'section', label: 'Kişisel bilgiler', description: '' },
            { id: id(), type: 'short_text', label: 'Ad Soyad', required: true },
            { id: id(), type: 'email', label: 'E-posta', required: true },
            { id: id(), type: 'phone', label: 'Telefon', required: true },
            { id: id(), type: 'date', label: 'Doğum tarihi', required: false },
            { id: id(), type: 'short_text', label: 'Yaşadığınız şehir', required: false },
            { id: id(), type: 'section', label: 'Başvuru detayları', description: '' },
            { id: id(), type: 'dropdown', label: 'Başvurduğunuz pozisyon', required: true, options: ['Yazılım Geliştirici', 'Tasarımcı', 'Pazarlama', 'Satış', 'Operasyon', 'Diğer'] },
            { id: id(), type: 'number', label: 'Toplam deneyim (yıl)', required: false, min: 0, max: 60 },
            { id: id(), type: 'url', label: 'LinkedIn profili', required: false, placeholder: 'https://www.linkedin.com/in/...' },
            { id: id(), type: 'url', label: 'Portföy / GitHub', required: false, placeholder: 'https://...' },
            { id: id(), type: 'long_text', label: 'Neden bizimle çalışmak istiyorsunuz?', required: true, placeholder: 'Birkaç cümleyle anlatın...' },
            { id: id(), type: 'date', label: 'Mümkün olduğunca erken başlama tarihi', required: false },
          ],
        };
      },
    },
    {
      key: 'contact',
      title: 'İletişim Formu',
      description: 'Web sitenize gömülecek basit iletişim formu.',
      icon: 'email',
      color: 'sky',
      build: function () {
        return {
          title: 'Bize ulaşın',
          description: 'Sorularınız, geri bildirimleriniz veya iş birliği önerileriniz için lütfen formu doldurun. 24 saat içinde dönüş yaparız.',
          theme: { color: 'sky', icon: '💬', mode: 'light' },
          fields: [
            { id: id(), type: 'short_text', label: 'Adınız', required: true },
            { id: id(), type: 'email', label: 'E-posta adresiniz', required: true },
            { id: id(), type: 'short_text', label: 'Konu', required: true },
            { id: id(), type: 'dropdown', label: 'Talep türü', required: true, options: ['Genel soru', 'Teknik destek', 'Satış', 'İş birliği', 'Diğer'] },
            { id: id(), type: 'long_text', label: 'Mesajınız', required: true, placeholder: 'Konunuzu detaylıca açıklayın...' },
          ],
        };
      },
    },
    {
      key: 'feedback',
      title: 'Hata / Geri Bildirim',
      description: 'Kullanıcılarınızdan hata raporu ve öneri toplayın.',
      icon: 'warning',
      color: 'indigo',
      build: function () {
        return {
          title: 'Hata Bildirim Formu',
          description: 'Karşılaştığınız sorunu ne kadar ayrıntılı anlatırsanız, çözmemiz o kadar kolay olur.',
          theme: { color: 'indigo', icon: '🐛', mode: 'light' },
          fields: [
            { id: id(), type: 'radio', label: 'Bildirim türü', required: true, options: ['Hata bildirimi', 'Özellik isteği', 'Genel geri bildirim'] },
            { id: id(), type: 'short_text', label: 'Başlık', required: true, placeholder: 'Kısa bir özet' },
            { id: id(), type: 'long_text', label: 'Detaylı açıklama', required: true, placeholder: 'Ne yapmaya çalışıyordunuz, ne oldu?' },
            { id: id(), type: 'long_text', label: 'Adımlar (yeniden üretmek için)', required: false, placeholder: '1. ... \n2. ...\n3. ...' },
            { id: id(), type: 'dropdown', label: 'Öncelik / şiddet', required: true, options: ['Düşük', 'Orta', 'Yüksek', 'Kritik'] },
            { id: id(), type: 'short_text', label: 'Tarayıcı / cihaz', required: false, placeholder: 'ör. Chrome 120 / iPhone 15' },
            { id: id(), type: 'email', label: 'İletişim için e-posta', required: false },
          ],
        };
      },
    },
  ];

  /* Form theme color palette */
  root.FormThemeColors = [
    { key: 'indigo',  label: 'İndigo',  from: '#6366f1', to: '#8b5cf6' },
    { key: 'rose',    label: 'Gül',     from: '#f43f5e', to: '#ec4899' },
    { key: 'emerald', label: 'Yeşil',   from: '#10b981', to: '#14b8a6' },
    { key: 'amber',   label: 'Amber',   from: '#f59e0b', to: '#ef4444' },
    { key: 'sky',     label: 'Mavi',    from: '#0ea5e9' ,to: '#6366f1' },
    { key: 'slate',   label: 'Gri',     from: '#475569', to: '#1e293b' },
  ];

  /* Form icon options (emojis for simplicity + universal support) */
  root.FormIcons = ['', '⭐', '🎉', '💼', '💬', '🐛', '📋', '🍕', '🏆', '🎓', '📚', '🛒', '✈️', '🎵', '💡', '❤️'];

  /* Helper to get gradient CSS */
  root.formThemeGradient = function (key) {
    var c = (root.FormThemeColors.find(function (x) { return x.key === key; })) || root.FormThemeColors[0];
    return 'linear-gradient(135deg, ' + c.from + ', ' + c.to + ')';
  };
})(window);
