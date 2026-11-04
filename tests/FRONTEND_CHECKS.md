# Frontend verification — 2026-11-04

The UI was rendered with system Chromium and **mock API responses** (not the live Go server). This validates the browser-side layout and interactions independently from PostgreSQL.

- Desktop 1440×900: 9 cards initially, no JavaScript page errors, no horizontal overflow.
- Mobile 390×844: 9 cards initially, no JavaScript page errors, no horizontal overflow.
- Show more: 9 → 10 cards, button hides at the end.
- Demo login role: choosing Buyer populates the expected demo email and password.
- JS syntax checks: `node --check web/app.js`, `node --check web/voice.js` — passed.
- Voice test: `node tests/voice_smoke.cjs` — 7 mocked scenarios passed, including automatic search after recognition.
- Screenshots: `docs/preview/*-mock.png`. They use mock data and are **not proof of a live database run**.

Still required on a Windows computer with Docker Desktop: `docker compose up --build -d`, verify `/api/health`, run buyer/supplier/admin flows, test a real microphone. `tests/api_regression.py` modifies the database; use a disposable test database only.
