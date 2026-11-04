(() => {
  'use strict';
  const button = document.getElementById('voiceBtn');
  const language = document.getElementById('voiceLanguage');
  const status = document.getElementById('voiceStatus');
  const input = document.getElementById('searchInput');
  const Recognition = window.SpeechRecognition || window.webkitSpeechRecognition;
  if (!button || !language || !status || !input) {
    console.error('[Come and Find] Voice controls are missing from the page.');
    return;
  }

  let recognition = null;
  let listening = false;
  let timer = null;

  function show(message, kind = 'info') {
    status.textContent = message;
    status.dataset.state = kind;
    console.info('[Come and Find voice]', message);
  }
  function reset() {
    clearTimeout(timer);
    timer = null;
    listening = false;
    recognition = null;
    button.textContent = 'Voice input';
    button.setAttribute('aria-pressed', 'false');
    language.disabled = false;
  }
  function explainUnavailable() {
    if (!window.isSecureContext) {
      return 'Voice input is unavailable: open http://localhost:8080 (not a LAN IP) or HTTPS. Text search is available.';
    }
    if (!Recognition) {
      return 'Voice input is unavailable in this browser. Try current Google Chrome or Microsoft Edge; text search is available.';
    }
    return '';
  }
  const unavailable = explainUnavailable();
  if (unavailable) show(unavailable, 'error');
  else show('Ready. Click Voice input and allow microphone access. Speech recognition may need internet.');

  button.addEventListener('click', () => {
    if (listening) {
      show('Stopping voice input…');
      try { recognition?.stop(); }
      catch (err) { show('Could not stop recognition: ' + err.message, 'error'); reset(); }
      return;
    }
    const problem = explainUnavailable();
    if (problem) { show(problem, 'error'); return; }

    let current;
    try { current = new Recognition(); }
    catch (err) { show('Cannot create speech recognition: ' + err.message, 'error'); return; }
    recognition = current;
    listening = true;
    language.disabled = true;
    button.textContent = 'Stop voice input';
    button.setAttribute('aria-pressed', 'true');
    current.lang = language.value;
    current.continuous = false;
    current.interimResults = false;
    current.maxAlternatives = 1;
    let received = false;
    let failed = false;
    let timedOut = false;

    // Immediate visual feedback, even if the browser never calls onstart.
    show('Starting microphone… If prompted, click Allow.');
    current.onstart = () => {
      if (recognition !== current) return;
      show('Listening… Speak a product name and price, then wait.');
    };
    current.onresult = event => {
      if (recognition !== current) return;
      const transcript = Array.from(event.results || [])
        .map(result => result[0]?.transcript || '').join(' ').trim();
      if (!transcript) {
        show('No speech recognized. Try again or use text search.', 'error');
        return;
      }
      received = true;
      input.value = transcript;
      input.dispatchEvent(new Event('input', { bubbles: true }));
      input.focus();
      show('Recognized: ' + transcript + '. Updating results; you can edit the query.', 'success');
      const searchButton = document.getElementById('searchBtn');
      if(searchButton && !searchButton.disabled) searchButton.click();
    };
    current.onerror = event => {
      if (recognition !== current) return;
      failed = true;
      const messages = {
        'not-allowed': 'Microphone permission denied. Click the site controls next to the address, allow Microphone, and reload.',
        'service-not-allowed': 'Speech service unavailable or blocked by the browser. Try Chrome/Edge or text search.',
        'audio-capture': 'No working microphone found. Check Windows microphone settings and the input device.',
        'no-speech': 'No speech detected. Check microphone input and try again.',
        'network': 'Speech recognition network error. Check internet access or use text search.',
        'language-not-supported': 'Recognition language is not supported. Select Russian or English.',
        'aborted': 'Voice input stopped.'
      };
      show(messages[event.error] || ('Voice recognition failed: ' + (event.error || 'unknown error')), 'error');
      reset();
    };
    current.onend = () => {
      if (recognition !== current) return;
      if (!received && !failed && !timedOut) show('No speech recognized. Try again or type your query.', 'error');
      reset();
    };
    timer = setTimeout(() => {
      if (recognition !== current) return;
      timedOut = true;
      show('Voice input timed out. Check microphone permissions, internet and browser support.', 'error');
      try { current.abort(); } catch (_) { /* browser may already be stopped */ }
      reset();
    }, 15000);
    try { current.start(); }
    catch (err) {
      show('Cannot start voice input: ' + err.message + '. Try reloading or use text search.', 'error');
      reset();
    }
  });

  window.addEventListener('pagehide', () => {
    try { recognition?.abort(); } catch (_) { /* already stopped */ }
    reset();
  });
})();
