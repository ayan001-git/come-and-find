(() => {
  const root = document.documentElement;

  function updateToggle() {
    const button = document.querySelector('#themeToggle');
    if (!button || typeof window.t !== 'function') return;

    const dark = root.dataset.theme === 'dark';
    const label = window.t(dark ? 'theme.toLight' : 'theme.toDark');
    button.setAttribute('aria-pressed', String(dark));
    button.setAttribute('aria-label', label);
    document.querySelector('#themeToggleIcon').textContent = dark ? '☀' : '☾';
  }

  const savedTheme = localStorage.getItem('cf-theme');
  root.dataset.theme = savedTheme === 'dark' ? 'dark' : 'light';

  window.toggleCfTheme = () => {
    const nextTheme = root.dataset.theme === 'dark' ? 'light' : 'dark';
    root.dataset.theme = nextTheme;
    localStorage.setItem('cf-theme', nextTheme);
    updateToggle();
  };
  window.updateCfThemeToggle = updateToggle;
  window.addEventListener('cf-language-change', updateToggle);
})();
