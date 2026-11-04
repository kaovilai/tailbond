// The phone page ("Get tailbond", agent/internal/agent/landing.html): copy
// buttons, and a better guess of this device than the server's (an iPad
// asks for the desktop site and calls itself a Mac). The page works
// without it: the platform sections switch with the address's #android,
// #mac, #iphone, #linux.
(function () {
  'use strict';
  var main = document.querySelector('main');
  var plats = ['android', 'mac', 'iphone', 'linux'];

  function detect() {
    var ua = navigator.userAgent || '';
    var uad = navigator.userAgentData;
    var p = uad && uad.platform ? uad.platform.toLowerCase() : '';
    if (p === 'android' || /Android/.test(ua)) return 'android';
    if (/iPhone|iPad|iPod/.test(ua)) return 'iphone';
    // iPadOS shows the desktop site: a "Mac" with a touch screen
    if (/Macintosh/.test(ua) && navigator.maxTouchPoints > 1) return 'iphone';
    if (p === 'macos' || /Macintosh|Mac OS X/.test(ua)) return 'mac';
    if (p === 'linux' || p === 'chrome os' || /Linux|X11|CrOS/.test(ua)) return 'linux';
    return '';
  }

  function show(p) {
    plats.forEach(function (x) {
      var s = document.getElementById(x);
      if (s) s.classList.toggle('first', x === p);
    });
    var links = document.querySelectorAll('nav.plats a');
    for (var i = 0; i < links.length; i++) {
      links[i].classList.toggle('on', links[i].getAttribute('href') === '#' + p);
    }
  }

  // the server's guess stands unless this one is clearer (iPad, client hints)
  var guess = detect();
  if (!location.hash && guess && guess !== main.getAttribute('data-detected')) show(guess);
  // the switcher: keep the highlighted tab in step with the address
  window.addEventListener('hashchange', function () {
    var h = location.hash.slice(1);
    if (plats.indexOf(h) >= 0) show(h);
  });
  var h0 = location.hash.slice(1);
  if (plats.indexOf(h0) >= 0) {
    show(h0);
    // a link to a section (#android from the QR code) opens at the top of
    // the page, title included, not scrolled down to the switcher
    window.addEventListener('load', function () { setTimeout(function () { window.scrollTo(0, 0); }, 0); });
  }

  // Copy: navigator.clipboard needs a secure context, which http://homebox/
  // isn't; fall back to selecting the text and execCommand('copy').
  function copied(btn, ok) {
    btn.textContent = ok ? 'Copied' : 'Select and copy';
    btn.classList.toggle('done', ok);
    setTimeout(function () { btn.textContent = 'Copy'; btn.classList.remove('done'); }, 2000);
  }
  function fallback(text) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.setAttribute('readonly', '');
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    var ok = false;
    try { ok = document.execCommand('copy'); } catch (e) { ok = false; }
    document.body.removeChild(ta);
    return ok;
  }
  document.addEventListener('click', function (e) {
    var btn = e.target.closest ? e.target.closest('button.copy') : null;
    if (!btn) return;
    var text = btn.getAttribute('data-copy') || '';
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text).then(function () { copied(btn, true); }, function () { copied(btn, fallback(text)); });
    } else {
      copied(btn, fallback(text));
    }
  });
})();
