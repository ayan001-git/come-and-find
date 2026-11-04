# Regression checks

api_regression.py uses Python's standard library against a running application.
Use a fresh, disposable database: the test changes stock, orders and accounts.
The port defaults to 8080; override with TEST_BASE_URL.

frontend_dom.mjs exercises the actual app.js against a live API using jsdom.
It verifies behavior and DOM updates, not browser layout or rendering.
From tests/, run `npm install --no-save jsdom`, then `node frontend_dom.mjs`.
Run API and DOM checks against separate fresh databases; the DOM test assumes
initial demo product ID 1 and order ID 1. No node_modules is required to run the app.

Native PostgreSQL concurrent-checkout testing and a real-browser/mobile visual
check remain required before claiming production readiness.

Voice recognition is mocked in DOM tests: real microphone, permission UI and external recognition service must be tested on the demonstration computer. Run `go test ./...` for standalone smart-parser unit tests.
