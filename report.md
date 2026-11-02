# Focused project audit

**Subject:** ft_transcendence v21.2. **Checked:** 2026-11-02. **Evaluated version:** local working tree and GitHub `master`, both at [e6fde3c](https://github.com/ef3d0c3e/transcendance/commit/e6fde3ca634cf59853950aeaad05f488619d2f9).

This shorter audit covers mandatory requirements and modules supported by actual project work, the existing project plan, or an open issue. Unrelated optional modules are omitted. The full audit from 2026-10-30 remains a separate historical snapshot; this light audit reflects the newer version.

**Potential** is the subject value if the whole module is completed. **Earned** counts only complete implementation in the evaluated `master`; partial work, unmerged branches, open PRs and issue-only plans earn **0**. Mandatory requirements have no separate points but can block acceptance. ✅ = implemented; ⚠️ = partial or verification incomplete; 🔄 = unmerged implementation; ❎ = no implementation, even if planned.

## Mandatory requirements

### Application, security and deployment

| Requirement | Potential | Earned | Status / integration | Evidence and remaining work |
| --- | ---: | ---: | --- | --- |
| Web application with frontend, backend and database | 0 | 0 | ✅ Merged | Functional Svelte UI, Gin backend and MariaDB/GORM models, now including chat, friend/block relations, trades and economy tables. [Frontend][frontend-main], [routes][routes], [models][models-auth]. |
| Git with meaningful commits and contributions from all team members | 0 | 0 | ⚠️ Partial | Meaningful commits and merged PRs exist from four authors. Complete member/author mapping and evidence of proper work distribution are missing from [README][readme]. |
| Container deployment runnable with one command | 0 | 0 | ✅ Merged — PRs #41, #103 | Configured production Compose builds/starts the stack. Merged [PR #103](https://github.com/ef3d0c3e/transcendance/pull/103) preserves the browser Host/port, configures trusted proxy/client IP handling and updates outer-proxy WebSocket instructions. [Compose][compose], [DEPLOY][deploy]. Fresh deployment is unverified; no additional module point for the fix. |
| Latest stable Chrome compatibility | 0 | 0 | ⚠️ Browser verification pending; homepage fixed | `/` has a real [index controller][index-controller], template and [English/French tests][index-tests]; PR #102 later refreshed the [index page][index-template]. Full latest-Chrome feature/console/device verification remains unproven. |
| No JavaScript warnings/errors in browser console | 0 | 0 | ⚠️ Unverified | Exact-commit frontend build and ESLint pass; these do not prove a clean runtime console. No browser-console acceptance test was found. |
| Accessible, relevant Privacy Policy | 0 | 0 | ✅ Merged — PR #66 | Footer links `/legal`; substantive personal-data, retention, rights and cookie content exists. [Page][legal-template], [legal tests][legal-tests]. |
| Accessible, relevant Terms of Service | 0 | 0 | ✅ Merged — PR #66 | Footer and registration link `/terms`; account, conduct, content, virtual-item and termination terms exist. [Page][terms-template], [legal tests][legal-tests]. |
| Multiple users logged in and active simultaneously | 0 | 0 | ✅ Merged | Persistent per-user sessions and request-local authentication; multiple-account/session tests. [auth.go][auth], [models][models-auth]. |
| Concurrent actions without corruption, races or performance conflicts | 0 | 0 | ⚠️ Partial | Transactional conditional debits, booster-pile consumption and trade locking remain implemented; cards locked in open trades cannot be resold or reused, and a trading data race was fixed. CI uses the Go race detector and [economy][economy-tests]/[trade][trade-tests] tests cover the flows, but comprehensive concurrent-user/load verification remains absent. [Shop][shop], [economy][economy], [trade][trade]. |
| Real-time updates across connected users when applicable | 0 | 0 | ✅ Merged — PR #63 | Chat broadcast and private delivery now run in master over WebSockets, with gateway forwarding support from merged PR #103. [Transport][chat-transport], [hub][chat-hub]. Non-chat notification feed refreshes through the notifications API rather than a pushed channel. |
| Clear, responsive and accessible frontend across devices | 0 | 0 | ⚠️ Partial; mobile fixes merged — PR #93; WCAG work merged — PR #107 | Viewport metadata and phone/landscape pack-fitting changes from [PR #93](https://github.com/ef3d0c3e/transcendance/pull/93) and WCAG/pack-opening accessibility work from [PR #107](https://github.com/ef3d0c3e/transcendance/pull/107) are master functionality. Document language is now set in [base][base]. Full custom-modal focus trap/restore and device/assistive-technology verification remain incomplete. [Theme][theme], [popup][form-popup]. Do not keep reporting viewport metadata as missing. |
| CSS framework or chosen styling solution | 0 | 0 | ✅ Merged | Custom theme tokens, typography and shared/Svelte styles meet the styling-solution requirement. [theme.css][theme]. |
| Credentials in Git-ignored `.env`; provide `.env.example` | 0 | 0 | ⚠️ Partial | Production example/generator and ignored `docker/.env` exist, but **root `.env` is tracked and not ignored**. [Ignore rules][gitignore], [example][env-example]. Values are not reproduced. |
| Clear database schema and relations | 0 | 0 | ✅ Merged | Users, sessions, profiles, notifications, friend/block relations, cards, boosters/piles, trades and daily streaks have keys/indexes/relations and migrations. [Models][models-tree], [migration][database]. README schema description remains separately missing. |
| Secure signup/login, including email/password and salted password hashes | 0 | 0 | ✅ Merged — PR #90 | Required unique email, email/username login, bcrypt hashes and random expiring sessions. [auth.go][auth], [PR #90](https://github.com/ef3d0c3e/transcendance/pull/90). |
| Frontend and backend validation for all forms/inputs | 0 | 0 | ⚠️ Partial | Many constraints exist, but backend email syntax validation and client avatar size/dimension checks are still missing; file `accept` is not full validation. [Auth][auth], [avatar][avatar], [profile UI][profile-ui]. |
| HTTPS for every external backend connection | 0 | 0 | ⚠️ Partial; production proxy fix merged | Production TLS gateway/internal app ports exist, and PR #103 corrects Host/port forwarding, trusted client IP handling and outer-proxy upgrade instructions. Development still exposes HTTP directly, so all-mode HTTPS remains incomplete. [Gateway][gateway], [main][main], [DEV][dev], [DEPLOY][deploy]. |

### Team, README and evaluation

Each row below also has **0 potential / 0 earned** module points; these are mandatory acceptance requirements, not scoring modules.

| Requirement | Potential | Earned | Status / integration | Evidence and remaining work |
| --- | ---: | ---: | --- | --- |
| Team of 4–5; everyone contributes to mandatory work and modules | 0 | 0 | ⚠️ Partly evidenced | Four team handles are listed and multiple authors contributed code. Per-member contributions to both parts are not documented or fully verifiable. [Team handles][legal-code], Git history. |
| Assign and document PO, PM/Scrum Master, Technical Lead/Architect and Developer roles | 0 | 0 | ❎ Missing | No role/responsibility mapping in README. Include product/backlog/validation, coordination/progress/blockers, architecture/quality/review and implementation/test/documentation duties. [README][readme]. |
| Explain work division, communication and individual contributions | 0 | 0 | ⚠️ Partial | [ORG][org] describes the issue/PR/review workflow; actual roles, coordination, communication channels and contribution breakdown remain undocumented. |
| Root English README with exact italic 42 attribution first line | 0 | 0 | ⚠️ File exists; required content missing | Root README is five CI badges. Open [issue #14](https://github.com/ef3d0c3e/transcendance/issues/14); unmerged [README branch][readme-branch] is mainly empty headings. |
| README Description and Instructions | 0 | 0 | ❎ Missing | Must state project name, goal, overview/features, prerequisites/tools/versions/configuration and compilation/install/startup steps. DEV/DEPLOY exist but are not linked as README instructions. |
| README Resources and AI disclosure | 0 | 0 | ❎ Missing | Must provide references and specify AI-assisted tasks and affected project parts. |
| README Team Information and Project Management | 0 | 0 | ❎ Missing | Must identify each member's roles/responsibilities, work organization, management tools and communication channels. |
| README Technical Stack and Database Schema | 0 | 0 | ❎ Missing | Must explain frontend/backend/database/libraries and technical choices; describe tables, relations, key fields and types. |
| README Features List, Modules and Individual Contributions | 0 | 0 | ❎ Missing | Must list functionality, module choices/implementation/justifications and point calculation, with contributor attribution, challenges and resolutions. |
| Understand/review/test AI-generated work and seek peer review | 0 | 0 | ⚠️ Not fully verifiable | Tests/CI/PR workflow exist. Individual understanding and AI-specific peer validation require team evidence, not just repository inspection. |
| Submit repository work; explain and demonstrate modules; perform a small change if requested | 0 | 0 | ⚠️ Evaluation pending | Only integrated/submitted work counts for this version. Oral understanding, feature demonstrations and live-modification readiness cannot be certified here. This report is untracked and does not replace README. |

## Modules within project scope

### Web

| Module | Potential | Earned | Status / integration | Evidence and remaining work |
| --- | ---: | ---: | --- | --- |
| Frameworks for frontend and backend | 2 | 2 | ✅ Merged | Svelte UI components are mounted; Gin supplies routing/middleware/controllers. [Frontend][frontend-main], [routes][routes]. The separate 1-point frontend/backend alternatives are covered by these 2 points and are not added again. |
| Real-time features using WebSockets | 2 | 2 | ✅ Merged — PR #63 | Transport/hub implements broadcasts, private delivery, queues, ping/cleanup/session checks and reconnect handling, with same-host gateway-origin acceptance and regression tests. Chat now runs in master behind the merged proxy support of PR #103. [Transport][chat-transport], [hub][chat-hub], [socket][chat-socket]. |
| Allow users to interact: chat + profiles + friends | 2 | 2 | ✅ Merged — PRs #63, #108 | All three subfeatures are integrated and functional in master: shoutbox chat and private messages, editable profiles, and typed friend/block relations with request accept/deny/cancel/remove flows and friends/sent/received lists. Friend actions are POST-only; the earlier GET-mutation and filtering defects are gone. [Chat][chat-transport], [friends][friends-code], [list query][friends-list], [button][friend-button], [tests][friends-tests]. Live-browser functional verification remains a global limit. |
| ORM | 1 | 1 | ✅ Merged | GORM models, relations, migrations and transactions are used functionally. [Database][database], [shop][shop]. |
| Complete notifications for creation, update and deletion actions | 1 | 0 | ⚠️ Partial, merged | Persisted feed/read/unread/dismiss and ownership checks work, and PR #63/#108/#111 added private-message (grouped per unread sender), friend-request/accepted, trade and trade-counter types. Creation and update actions of friends/trades/messages are covered; deletion actions (trade cancel/decline, content removal) still produce no notifications, and application-wide lifecycle coverage remains incomplete. [Controller][notifications], [types][notif-types], [tests][notification-tests]. |
| Server-Side Rendering (SSR) | 1 | 1 | ✅ Merged | Go templates render translated pages before Svelte enhancement, including the homepage, redesigned inventory/collection pages and the daily-streak tile. [Renderer][renderer], [index controller][index-controller], [catalog][catalog]. The shop experience lives in the inventory; `/shop` redirects. These extend the existing single SSR point. |
| Progressive Web App (PWA): offline support and installability | 1 | 0 | ⚠️ Partial, merged — PRs #101, #102 | Installability now exists in master: [manifest][web-manifest], icons, standalone launch and apple-touch/theme metadata (PR #101), plus homepage install instructions (PR #102). **No service worker and no offline handling exist**, and the module requires both offline support and installability, so earned stays 0. |
| Custom design system, minimum 10 reusable components | 1 | 1 | ✅ Merged; component inventory refreshed | Palette/typography/icons and well over ten reusable units: FormPopup/FormEnhancement, ThemeSwitcher, LanguageSwitcher, Notification, InventoryCardPopup, ShopBuyPopup, FriendButton, Shoutbox/ChatConversations/ChatUserMenu, TradeMenu/TradeUpPicker/TradeUpResult, PackOpening/PackBackground, Navbar; shared [inventory tabs][inventory-tabs]/[top-up popup][topup-popup]/[daily streak][streak-partial]/[booster pile][pile-partial]/[card][card-partial]/[collection][collection-partial]/[paginator][paginator-partial] add reuse. [Theme][theme], [components][frontend-tree], [partials][partials-tree]. PR #93 removed InventoryBoosterPopup, so it is no longer counted. |
| Advanced search: filters, sorting and pagination | 1 | 1 | ✅ Merged, including collection-detail filters — PR #93 | Catalog/inventory/user searching meets the module, with collection-scoped rarity/text filters, sorting/pagination and owned/missing views. [Catalog][catalog], [collection tests][catalog-tests], [user search][user-search], [tests][search-tests]. Issue #79 is now closed and its feature was already implemented; no extra search point is added. |
| File upload and management | 1 | 0 | ⚠️ Partial, merged | Avatar upload handles JPEG/PNG/WebP, server limits/re-encoding, owner/admin changes, preview and deletion. Missing client size/dimension validation and upload progress indicators. [Avatar][avatar], [profile UI][profile-ui]. |

### Accessibility and Internationalization

| Module | Potential | Earned | Status / integration | Evidence and remaining work |
| --- | ---: | ---: | --- | --- |
| Complete WCAG 2.1 AA accessibility | 2 | 0 | ⚠️ Partial, merged UI work — PR #107 | Labels, focus styles, reduced motion and the document language are in place, and PR #107 added [accessibility tests][accessibility-tests] plus pack-opening accessibility work. Missing complete keyboard/custom-modal focus trap-restore work and screen-reader/assistive-technology compliance verification. [Base][base], [theme][theme], [popup][form-popup]. |
| At least 3 complete languages | 1 | 0 | ⚠️ Two languages merged; open issue #60 | Fluent and public language switching work; only English/French locales exist. Spanish remains a placeholder switcher entry, not a third translation; some UI strings remain hard-coded. [Localizer][localizer], [issue #60](https://github.com/ef3d0c3e/transcendance/issues/60). |
| Right-to-left language support | 1 | 0 | ❎ Issue only — #60 | Hebrew/RTL is requested in [issue #60](https://github.com/ef3d0c3e/transcendance/issues/60). No translation, complete layout mirroring or LTR/RTL switching implementation was found. An open issue is a plan, not implementation in progress. |
| Support for additional browsers | 1 | 0 | ⚠️ Existing plan; verification missing | Browser UI exists, but no complete test/fix matrix or limitations document. Subject requires **two additional browsers beyond Chrome**; Chrome/Chromium plus Firefox alone is insufficient. |

### User Management

| Module | Potential | Earned | Status / integration | Evidence and remaining work |
| --- | ---: | ---: | --- | --- |
| Standard user management and authentication | 2 | 2 | ✅ Merged — PRs #100, #108 | Editable profiles, default/uploaded avatars, online status and the required friend features (requests, lists, blocks) are master functionality. PR #100 fixed profile editing behind reverse proxies and is merged; the issue #95 mismatch is addressed on paper, with deployment regression confirmation part of the global unverified-deployment limit. [Profile][profile], [friends][friends-code], [online tests][online-tests], [friends tests][friends-tests]. |
| Advanced permissions system | 2 | 2 | ✅ Merged — PR #83; issue #96 closed | Account creation/view/edit/delete, admin-controlled roles and role-dependent UI/actions remain implemented. [Profile controller][profile], [permission tests][account-tests]. Master renders rank names and an admin rank selector in [profile.html][profile-template]; the formerly open issue #96 asked for exactly that and is now closed. |

### Gaming and user experience

| Module | Potential | Earned | Status / integration | Evidence and remaining work |
| --- | ---: | ---: | --- | --- |
| Advanced chat features | 1 | 0 | ⚠️ Partial, now merged — PR #63 | History, profile links and PM-blocking temporary ignores exist in master, with grouped unread PM notifications. Missing typing indicators, sender-visible read receipts, game invitations and game/tournament notices; the game-related features also need a game, and master contains none. [PM controller][chat-private], [ignore controller][chat-ignore]. |
| Gamification system | 1 | 1 | ✅ Merged — PR #111; category call is borderline | PR #111 establishes three gamification categories in master: daily connection streaks with a claimable reward ladder ([streak][streak]), a coin economy earned through dailies and selling (including duplicates) and spent in the shop ([economy][economy], [sell][sell]), and a refilling free-booster pile ([pile][economy]). Trading and collection completion meters are not needed for the count and are not claimed as categories. An evaluator who rejects shop currency as a points category would score this row 0. |

### Devops

| Module | Potential | Earned | Status / integration | Evidence and remaining work |
| --- | ---: | ---: | --- | --- |
| ELK log management | 2 | 2 | ✅ Merged — PR #41 | GELF/Logstash processing, Elasticsearch indexing, Kibana dashboard, rollover/compressed warm retention/deletion and protected access are configured. [Pipeline][logstash], [policies][es-setup], [Compose][compose], [PR #41](https://github.com/ef3d0c3e/transcendance/pull/41). Compressed retention is not an independent backup. |
| Prometheus and Grafana monitoring | 2 | 2 | ✅ Merged — PR #41 | App metrics, multiple exporters, provisioned dashboards/data sources, alerts/Alertmanager and protected Grafana access exist. [Metrics][metrics], [scrapes][prometheus], [dashboards][grafana-tree], [alerts][alerts]. |
| Health checks/status page, automated backups and disaster recovery | 1 | 0 | ⚠️ Health/monitoring merged — PR #41 | Database-aware `/healthz`, probes and dashboards exist. Dedicated status-page system, scheduled backups and restore/disaster-recovery procedures are missing. [Health controller][health], [DEPLOY][deploy]. |

### Data and Analytics

| Module | Potential | Earned | Status / integration | Evidence and remaining work |
| --- | ---: | ---: | --- | --- |
| GDPR compliance features | 1 | 0 | ⚠️ Partial, merged | Confirmed permanent account deletion/cascading erasure exists, plus contact instructions for data requests. No implemented readable user-data export/fulfilment flow or confirmation emails. [Deletion][profile], [tests][account-tests], [legal content][en]. Privacy/ToS pages do not independently complete this module. |

## Points and remaining work

| Category | Potential in listed scope | Earned on master |
| --- | ---: | ---: |
| Web | 13 | 10 |
| Accessibility and Internationalization | 5 | 0 |
| User Management | 4 | 4 |
| Gaming and user experience | 2 | 1 |
| Devops | 5 | 4 |
| Data and Analytics | 1 | 0 |
| **Total** | **30** | **19** |

**19 earned / 14 required.** The **22 listed modules** offer **30 potential points if fully completed**. Counting stays capped at 14 required plus up to 5 bonus points, so the listed scope now fills that cap exactly; the remaining 11 potential points are scoped opportunity, not a further counted award. Incomplete/unmerged work and framework alternatives add no duplicate credit.

**Changes since the previous audit ([bebd391](https://github.com/ef3d0c3e/transcendance/commit/bebd391f1d038dd065daa68e488244d93ed71ae6)):** ten PRs merged and the score moved from 12 to 19. [PR #63](https://github.com/ef3d0c3e/transcendance/pull/63) integrated the chat, earning the WebSocket (2) and interaction (2) rows; [PR #108](https://github.com/ef3d0c3e/transcendance/pull/108) integrated friends/blocks, completing standard user management (2); [PR #111](https://github.com/ef3d0c3e/transcendance/pull/111) integrated the economy/trading work — daily streaks, coins, booster pile, selling and trades — earning gamification (1). [PR #101](https://github.com/ef3d0c3e/transcendance/pull/101) merged the PWA manifest/installability and [PR #102](https://github.com/ef3d0c3e/transcendance/pull/102) added homepage install instructions; without a service worker the PWA row stays 0. [PR #100](https://github.com/ef3d0c3e/transcendance/pull/100) merged the profile-edit proxy fix. [PR #107](https://github.com/ef3d0c3e/transcendance/pull/107) merged WCAG/pack-opening accessibility work with tests. [PR #110](https://github.com/ef3d0c3e/transcendance/pull/110) merged UI improvements. The [cacti][cacti-collection] ([PR #104](https://github.com/ef3d0c3e/transcendance/pull/104)) and [flowers][flowers-collection] ([PR #105](https://github.com/ef3d0c3e/transcendance/pull/105)) collections are also merged. Content additions extend existing modules and earn **0 additional standalone points**.

There is **one open PR: [#112](https://github.com/ef3d0c3e/transcendance/pull/112)** with the [gemstones and minerals collection][gemstones-collection] on head `eccfd87`, referencing collections issue #73. It is a content addition with **0 standalone potential / 0 earned** points. All six PRs open at the previous audit (#63, #100, #101, #102, #104, #105) are merged; PR #99 was closed without merge.

The largest mandatory gaps remain README/team documentation, tracked root credentials, complete validation/HTTPS coverage and browser/device/concurrency verification. Chat, friends, economy, PWA installability and the profile-proxy fix are no longer gaps. Advanced chat game features still need a game. Issues #56 (currency/boosters) and #55 (selling cards) remain open although PR #111 implemented them — open issue status does not make that functionality missing, the same caveat previously applied to #96/#79, both now closed. Open issues #14 (README), #60 (Hebrew/RTL), #89 (card watermark) and #5 (brute-force protection) have no implementation in master.

Friends, chat, daily economy and user trading are now master functionality. Installation/tutorial/profile fixes have **0 standalone potential / 0 earned** points beyond their relevant module rows. Trading, selling and trade-up have no independently claimed subject module or README-justified custom award; they do not automatically complete gamification or a competitive game. The combined `testing` branch is not the evaluated master.

**Verification limits:** the actual master delta since `bebd391` (75 commits, 10 merged PRs), surviving open work, related issue states and current CI were checked. All **64** GitHub branch heads match local remote-tracking refs. For exactly `e6fde3c`, [Go tests/race CI](https://github.com/ef3d0c3e/transcendance/actions/runs/41375857507), [Go lints](https://github.com/ef3d0c3e/transcendance/actions/runs/41375857474), [frontend build](https://github.com/ef3d0c3e/transcendance/actions/runs/41375857481), [ESLint](https://github.com/ef3d0c3e/transcendance/actions/runs/41375857473) and [Lingora](https://github.com/ef3d0c3e/transcendance/actions/runs/41375857632) report success. No fresh local build, deployment, browser/accessibility or load test was run. Mandatory gaps may still reject the project despite the implementation subtotal. The full audit remains unchanged and historical.

<!-- Evidence pinned to the evaluated commit or explicitly unmerged head. -->
[frontend-main]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/frontend/src/main.js
[routes]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/routes/routes.go
[models-auth]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/models/auth.go
[readme]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/README.md
[compose]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/docker/docker-compose.yml
[deploy]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/docs/DEPLOY.md
[index-controller]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/data/index.go
[index-template]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/pages/index.html
[index-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/index_test.go
[legal-template]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/pages/legal.html
[legal-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/legal_test.go
[terms-template]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/pages/terms.html
[auth]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/auth.go
[pack-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/pack_test.go
[shop]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/shop.go
[pack]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/data/pack.go
[base]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/layouts/base.html
[theme]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/static/theme.css
[form-popup]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/frontend/src/FormPopup.svelte
[gitignore]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/.gitignore
[env-example]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/docker/.env.example
[models-tree]: https://github.com/ef3d0c3e/transcendance/tree/e6fde3ca634cf59853950aeaad05f488619d2f9/models
[database]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/config/database.go
[avatar]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/avatar.go
[profile-ui]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/frontend/src/ProfileEnhancement.svelte
[gateway]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/docker/gateway/templates/gateway.conf.template
[main]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/main.go
[dev]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/docs/DEV.md
[legal-code]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/legal.go
[org]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/docs/ORG.md
[readme-branch]: https://github.com/ef3d0c3e/transcendance/tree/8b3d14f802a49af73148f664589e25f295b2ece2
[chat-transport]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/shoutbox.go
[chat-hub]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/shoutbox_hub.go
[chat-socket]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/frontend/src/shoutbox-socket.js
[chat-private]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/chat_private.go
[chat-ignore]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/chat_ignore.go
[friends-code]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/friends.go
[friends-list]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/search_friends.go
[friends-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/friends_test.go
[friend-button]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/frontend/src/FriendButton.svelte
[notifications]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/notifications.go
[notif-types]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/models/notifications.go
[notification-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/notifications_test.go
[economy]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/economy.go
[economy-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/economy_test.go
[sell]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/sell.go
[streak]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/data/streak.go
[trade]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/trade.go
[trade-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/trade_test.go
[renderer]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/views/renderer.go
[catalog]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/data/catalog.go
[web-manifest]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/static/manifest.json
[accessibility-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/accessibility_test.go
[inventory-tabs]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/partials/inventory-tabs.html
[topup-popup]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/partials/topup-popup.html
[streak-partial]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/partials/daily-streak.html
[pile-partial]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/partials/booster-pile.html
[card-partial]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/partials/card.html
[collection-partial]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/partials/collection.html
[paginator-partial]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/partials/paginator.html
[frontend-tree]: https://github.com/ef3d0c3e/transcendance/tree/e6fde3ca634cf59853950aeaad05f488619d2f9/frontend
[partials-tree]: https://github.com/ef3d0c3e/transcendance/tree/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/partials
[catalog-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/catalog_test.go
[user-search]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/user_search.go
[search-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/search_test.go
[localizer]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/localization/localization.go
[profile]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/profile.go
[online-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/online_test.go
[account-tests]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/account_test.go
[profile-template]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/templates/pages/profile.html
[logstash]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/docker/logstash/pipeline/logs.conf
[es-setup]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/docker/elasticsearch/setup.sh
[metrics]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/metrics/metrics.go
[prometheus]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/docker/prometheus/prometheus.yml
[grafana-tree]: https://github.com/ef3d0c3e/transcendance/tree/e6fde3ca634cf59853950aeaad05f488619d2f9/docker/grafana/dashboards
[alerts]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/docker/prometheus/rules/alerts.yml
[health]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/controllers/health.go
[en]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/localization/locales/en/main.ftl
[cacti-collection]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/data/collections/cacti/collection.toml
[flowers-collection]: https://github.com/ef3d0c3e/transcendance/blob/e6fde3ca634cf59853950aeaad05f488619d2f9/data/collections/flowers/collection.toml
[gemstones-collection]: https://github.com/ef3d0c3e/transcendance/tree/eccfd87615e052ba96a7e082c97c8c63139562c2/data/collections/gemstones
