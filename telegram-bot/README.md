# SHAKIL INTEL X

The bot uses a separate PostgreSQL database. Users without an active plan receive three one-time trial searches with results. Each search with results, including a repeat, uses one search; no-result and failed searches do not. Paid daily allowances reset at midnight IST. Plain phone numbers, emails and document numbers can be sent directly; all-numeric inputs search both phone and document-number fields in one request.

The backend expands exact document IDs and one bounded hop through alternate phone numbers, without following names or addresses. It scans at most 500 records per stage and returns up to five records, marking incomplete expansion. Search results can be copied from Telegram and downloaded as CSV. Reports are encrypted at rest, available to the requesting user until midnight IST on the search date and cleaned on a later search. CSV cells that could execute formulas in spreadsheet software are escaped.

Telegram does not reveal the originating device. Customer workload limits reduce abuse but cannot prevent users sharing an account. Payment verification requires the administrator to check the actual UPI receipt.

## Local Telegram testing

The bot can be tested from Telegram without deploying it. `BOT_MODE=polling` uses Telegram's getUpdates, so no public webhook or tunnel is needed. Only one process may poll a bot token, and polling will not work while that bot has a webhook configured. Use a separate test bot token; avoid interfering with a production bot. Configure the initial admin's numeric Telegram ID and keep your bot token private.

1. Copy `.env.example` to `.env`, if it does not already exist. Set `BOT_TOKEN`, `ADMIN_IDS`, `WEBHOOK_SECRET`, `POSTGRES_PASSWORD`, `DATABASE_URL`, and `TELEGRAM_SEARCH_SECRET`. Secrets must be independent, random and at least 32 characters. The database password and URL must match. Set `BOT_MODE=polling`.
2. Start local PostgreSQL and the bot with `docker compose up -d --build`. Data lives in the `bot-db` volume; back it up separately.
3. For local searches, run the backend and use host networking or another secure local forwarding arrangement to reach its loopback-only endpoint at `127.0.0.1:8081/internal/telegram/search`. Set the same `TELEGRAM_SEARCH_SECRET` on both services. The backend dependencies, including PostgreSQL and OpenSearch, must be running. Do not expose this endpoint publicly.
4. In Telegram, send `/start` from the administrator account. Create a plan, for example `/setplan Plus 500 100 30` (name, whole INR price, daily allowance, days). Create as many plans as needed; prices are stored in paise. A user starts the bot, then the admin assigns access with `/assign USER_ID Plus`. Verify their identity, authorization and payment independently before assigning.
5. Use the search buttons, `/account`, `/history`, `/plans` and `/admin`. `/account` displays the remaining daily search allowance. `/history` shows masked search records, ten per page; use `/history 2` for the next page. Quota resets do not filter or delete history. Full search inputs are encrypted for admin history. Results are encrypted while delivery is pending; downloadable reports remain encrypted and expire at midnight IST on the search date. Back up `HISTORY_ENCRYPTION_KEY` securely and separately from the database, and keep it stable across restarts and database migrations. Losing or rotating it without a migration makes historical values unreadable.

## Webhook deployment

For deployment, set `BOT_MODE=webhook`; reverse-proxy HTTPS `/telegram/webhook` to `127.0.0.1:8091` and register it with Telegram `setWebhook`, including `WEBHOOK_SECRET` as the secret token. The backend endpoint listens on `127.0.0.1:8081` when `TELEGRAM_SEARCH_SECRET` is configured. The production Compose file uses host networking so it can access that private listener. Store secrets securely. Set the bot profile image through @BotFather using `profile.jpg`.

Each account can have one search in progress, with default workload limits of 10 per rolling minute and 500 per rolling hour (`SEARCH_WORKLOAD_PER_MINUTE` and `SEARCH_WORKLOAD_PER_HOUR`). Repeats, empty results and failed searches count against workload limits. Repeats with results consume credits; empty results and failures do not. Telegram does not reveal a user's device identity, so these controls cannot prevent sharing a Telegram account.

Configured `ADMIN_IDS` are exempt from bot request throttles, daily search credits and the private backend workload limits; they do not need a paid plan to search. A blocked administrator remains unable to search. Telegram and upstream services may still enforce their own limits. Keep both the administrator IDs and the private search secret secure.

Local testing does not deploy code to `api.knotoriousai.online`. Its existing search endpoint behaves differently; the new restricted integration endpoint must be deployed separately before the bot can call it. Migrating to Neon later requires transferring the database, not simply changing `DATABASE_URL`.

## Manual UPI orders

Set `UPI_ID` in `.env` to the actual merchant UPI address. To show a QR image, set `UPI_QR_PATH` to the path **inside the container**, and add a read-only bind mount under the bot service, for example `- ./upi-qr.png:/run/upi-qr.png:ro` under `volumes`, with `UPI_QR_PATH=/run/upi-qr.png`. Confirm the QR belongs to the same payee. Without a QR image the bot shows the UPI ID; orders cannot be created until `UPI_ID` is configured. No gateway or automatic payment confirmation is used.

Create a positive-price plan with `/setplan Plus 500 100 30`. A user starts the bot, checks `/plans`, sends `/buy Plus`, pays the displayed amount, and enters the transaction reference. The most recently selected unpaid plan is submitted for admin review, and the customer is shown its terms. Only letters and digits are accepted in references. `/mypayments` shows order status.

Admins use `/pending` to see up to 20 pending reviews and unpaid selections; submitted reviews appear first. Verify the reference, amount, date and payee on the actual bank/UPI statement. If needed, use `/orderplan ORDER_ID PLAN` to correct the selected plan, then use `/approvepayment ORDER_ID`. Use `/rejectpayment ORDER_ID REASON` for rejected claims. Approval stores the original plan terms and starts access immediately. Existing active plans and blocked users require manual resolution before approval. Duplicate transaction references are rejected, including after rejection; use support to investigate errors. Admin and user notifications are best effort; admins should check `/pending`. Screenshots are not currently collected.

Payment and entitlement updates use one transaction, but production work remains: durable notification delivery, review and refund tools, complete search pagination, secure backups, and deployment testing. Do not treat submitted UPI references as proof of payment.

Admins can run `/resetquota USER_ID` to restore the user’s remaining allowance for the current IST day without deleting search history. Resets are audited and cannot run during an in-progress search. `/userhistory USER_ID [PAGE]` lets authorized admins view full queries, ten per page, and audits each lookup. Older records created before encryption was enabled remain masked. Customers see masked history only. A retention/deletion policy has not yet been implemented.

Admins can use `/users` or `/users PAGE` to list users, Telegram IDs, plans and statuses. `/user USER_ID` shows an account profile with plan expiry, usage and pending-payment count. `/userhistory USER_ID [PAGE]` displays full stored search inputs. These views are restricted to configured admins and audited; they do not include search-result records.
