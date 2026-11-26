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
  const voiceText = {
    ru: {button:'Голосовой поиск',stop:'Остановить голосовой ввод',secure:'Голосовой ввод доступен только на localhost или HTTPS. Текстовый поиск работает.',unsupported:'Голосовой ввод не поддерживается этим браузером. Попробуйте Chrome или Edge; текстовый поиск доступен.',ready:'Готово. Нажмите «Голосовой поиск» и разрешите доступ к микрофону. Для распознавания может понадобиться интернет.',stopping:'Останавливаем голосовой ввод…',stopError:'Не удалось остановить распознавание: ',createError:'Не удалось запустить распознавание речи: ',starting:'Включаем микрофон… Если появится запрос, нажмите «Разрешить».',listening:'Слушаю… Назовите товар и цену, затем подождите.',empty:'Речь не распознана. Попробуйте ещё раз или введите запрос.',recognized:'Распознано: ',update:'Результаты обновляются. Запрос можно отредактировать.',permission:'Нет доступа к микрофону. Разрешите микрофон в настройках сайта рядом с адресной строкой и перезагрузите страницу.',service:'Служба распознавания речи недоступна. Попробуйте Chrome/Edge или текстовый поиск.',audio:'Микрофон не найден. Проверьте настройки микрофона Windows и устройство ввода.',noSpeech:'Речь не обнаружена. Проверьте микрофон и попробуйте ещё раз.',network:'Ошибка сети распознавания речи. Проверьте интернет или используйте текстовый поиск.',unsupportedLang:'Этот язык распознавания не поддерживается. Выберите русский, казахский или английский.',aborted:'Голосовой ввод остановлен.',failed:'Ошибка распознавания речи: ',timeout:'Время голосового ввода истекло. Проверьте микрофон, интернет и поддержку браузера.',startError:'Не удалось включить голосовой ввод: '},
    kk: {button:'Дауыспен іздеу',stop:'Дауыстық енгізуді тоқтату',secure:'Дауыстық енгізу тек localhost немесе HTTPS арқылы қолжетімді. Мәтінмен іздеу жұмыс істейді.',unsupported:'Бұл браузер дауыстық енгізуді қолдамайды. Chrome не Edge қолданыңыз немесе мәтінмен іздеңіз.',ready:'Дайын. «Дауыспен іздеу» түймесін басып, микрофонға рұқсат беріңіз. Сөйлеуді тану үшін интернет қажет болуы мүмкін.',stopping:'Дауыстық енгізу тоқтатылуда…',stopError:'Тануды тоқтату мүмкін болмады: ',createError:'Сөйлеуді тануды қосу мүмкін болмады: ',starting:'Микрофон қосылуда… Сұрау шықса, «Рұқсат беру» түймесін басыңыз.',listening:'Тыңдап тұрмын… Тауар мен бағасын айтып, күтіңіз.',empty:'Сөйлеу танылмады. Қайталап көріңіз немесе мәтінмен іздеңіз.',recognized:'Танылды: ',update:'Нәтижелер жаңартылуда. Сұрауды түзете аласыз.',permission:'Микрофонға рұқсат жоқ. Мекенжай жолағындағы сайт баптауларынан микрофонға рұқсат беріп, бетті жаңартыңыз.',service:'Сөйлеуді тану қызметі қолжетімсіз. Chrome/Edge қолданыңыз немесе мәтінмен іздеңіз.',audio:'Жұмыс істейтін микрофон табылмады. Windows микрофоны мен енгізу құрылғысын тексеріңіз.',noSpeech:'Сөйлеу анықталмады. Микрофонды тексеріп, қайталап көріңіз.',network:'Сөйлеуді тану желісінде қате шықты. Интернетті тексеріңіз немесе мәтінмен іздеңіз.',unsupportedLang:'Бұл тану тілі қолдау таппайды. Орыс, қазақ немесе ағылшын тілін таңдаңыз.',aborted:'Дауыстық енгізу тоқтатылды.',failed:'Сөйлеуді тану қатесі: ',timeout:'Дауыстық енгізу уақыты аяқталды. Микрофонды, интернетті және браузерді тексеріңіз.',startError:'Дауыстық енгізуді бастау мүмкін болмады: '},
    en: {button:'Voice search',stop:'Stop voice input',secure:'Voice input is available only on localhost or HTTPS. Text search still works.',unsupported:'Voice input is not supported in this browser. Try Chrome or Edge; text search is available.',ready:'Ready. Click Voice search and allow microphone access. Speech recognition may need internet.',stopping:'Stopping voice input…',stopError:'Could not stop recognition: ',createError:'Cannot create speech recognition: ',starting:'Starting microphone… If prompted, click Allow.',listening:'Listening… Say a product name and price, then wait.',empty:'No speech recognized. Try again or type your query.',recognized:'Recognized: ',update:'Updating results. You can edit the query.',permission:'Microphone access denied. Allow Microphone in the site controls beside the address bar, then reload.',service:'Speech recognition is unavailable. Try Chrome/Edge or text search.',audio:'No working microphone found. Check your Windows microphone settings and input device.',noSpeech:'No speech detected. Check the microphone and try again.',network:'Speech recognition network error. Check internet access or use text search.',unsupportedLang:'This recognition language is not supported. Select Russian, Kazakh, or English.',aborted:'Voice input stopped.',failed:'Voice recognition failed: ',timeout:'Voice input timed out. Check microphone permissions, internet, and browser support.',startError:'Cannot start voice input: '}
  };
  const tr = key => (voiceText[window.cfLang] || voiceText.ru)[key];

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
    button.textContent = tr('button');
    button.setAttribute('aria-pressed', 'false');
    language.disabled = false;
  }
  function explainUnavailable() {
    if (!window.isSecureContext) {
      return tr('secure');
    }
    if (!Recognition) {
      return tr('unsupported');
    }
    return '';
  }
  const unavailable = explainUnavailable();
  if (unavailable) show(unavailable, 'error');
  else show(tr('ready'));

  window.addEventListener('cf-language-change', () => {
    if (!listening) show(explainUnavailable() || tr('ready'));
    if (!listening) button.textContent = tr('button');
  });

  button.addEventListener('click', () => {
    if (listening) {
      show(tr('stopping'));
      try { recognition?.stop(); }
      catch (err) { show(tr('stopError') + err.message, 'error'); reset(); }
      return;
    }
    const problem = explainUnavailable();
    if (problem) { show(problem, 'error'); return; }

    let current;
    try { current = new Recognition(); }
    catch (err) { show(tr('createError') + err.message, 'error'); return; }
    recognition = current;
    listening = true;
    language.disabled = true;
    button.textContent = tr('stop');
    button.setAttribute('aria-pressed', 'true');
    current.lang = language.value;
    current.continuous = false;
    current.interimResults = false;
    current.maxAlternatives = 1;
    let received = false;
    let failed = false;
    let timedOut = false;

    // Immediate visual feedback, even if the browser never calls onstart.
    show(tr('starting'));
    current.onstart = () => {
      if (recognition !== current) return;
      show(tr('listening'));
    };
    current.onresult = event => {
      if (recognition !== current) return;
      const transcript = Array.from(event.results || [])
        .map(result => result[0]?.transcript || '').join(' ').trim();
      if (!transcript) {
        show(tr('empty'), 'error');
        return;
      }
      received = true;
      input.value = transcript;
      input.dispatchEvent(new Event('input', { bubbles: true }));
      input.focus();
      show(tr('recognized') + transcript + '. ' + tr('update'), 'success');
      const searchButton = document.getElementById('searchBtn');
      if(searchButton && !searchButton.disabled) searchButton.click();
    };
    current.onerror = event => {
      if (recognition !== current) return;
      failed = true;
      const messages = {
        'not-allowed': tr('permission'),
        'service-not-allowed': tr('service'),
        'audio-capture': tr('audio'),
        'no-speech': tr('noSpeech'),
        'network': tr('network'),
        'language-not-supported': tr('unsupportedLang'),
        'aborted': tr('aborted')
      };
      show(messages[event.error] || (tr('failed') + (event.error || 'unknown error')), 'error');
      reset();
    };
    current.onend = () => {
      if (recognition !== current) return;
      if (!received && !failed && !timedOut) show(tr('empty'), 'error');
      reset();
    };
    timer = setTimeout(() => {
      if (recognition !== current) return;
      timedOut = true;
      show(tr('timeout'), 'error');
      try { current.abort(); } catch (_) { /* browser may already be stopped */ }
      reset();
    }, 15000);
    try { current.start(); }
    catch (err) {
      show(tr('startError') + err.message, 'error');
      reset();
    }
  });

  window.addEventListener('pagehide', () => {
    try { recognition?.abort(); } catch (_) { /* already stopped */ }
    reset();
  });
})();
