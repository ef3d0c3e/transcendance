*This Project has been created as part of the 42 curriculum by gdelhota, jeel-qai, lgamba, thschnei*

![Go tests badge](https://github.com/ef3d0c3e/transcendance/actions/workflows/go_test.yml/badge.svg)
![Go lints badge](https://github.com/ef3d0c3e/transcendance/actions/workflows/go_lint.yml/badge.svg)
![ESLint lints badge](https://github.com/ef3d0c3e/transcendance/actions/workflows/js_lint.yml/badge.svg)
![Svelte build badge](https://github.com/ef3d0c3e/transcendance/actions/workflows/svelte_build.yml/badge.svg)
![Lingora checks badge](https://github.com/ef3d0c3e/transcendance/actions/workflows/lingora.yml/badge.svg)

# Description

 `ft_transcendance` is the last project of the 42 common-core. It is a group project intended to boost creativity, adaptability to new technologies and teamwork skills. The goal of the project is to create a web application.
We chose to create a trading card game with various collections about different subjects.

The name of the project is **AnyCardz**. Its core features are the following:
- Booster packs to open for cards
- Viewing your card collection and the ones of other users
- Trading cards with other players
- Selling / Upgrading cards on the shop
- Account creation and management
- Live chat with other users
- Friend System

# Instructions

## First Start

Simply run `make` from the repository root to generate `docker/.env` and build, then start the stack.
`make down` stops it and `make logs` follows the logs.
More detailed information and admin access information can be found in `docs/DEV.md`

# Resources

 - [Gin Gonic documentation](https://gin-gonic.com/en/docs/)
 - [Gorm documentation](https://gorm.io/docs/)
 - [Svelte documentation](https://svelte.dev/docs)
 - [E-layout documentation](https://yukimi-inu.github.io/e-layout/)

# AI Usage

We used AI extensively for testing:
 - Playwright tests
 - Tests in headless Firefox/Chromium for frontend features
 - Go test writing
 - Regressions tests

We also used AI for UI mock-ups and frontend. It proved useful at writing ad-hoc user interfaces, but they generally had poorer user experience than hand-written ones.

# Team Information
 - _gdelhota_: Product Manager, Developer
 Responsibilities: organization and leading of team meetings, progression tracking using github issues
 - _jeel-qai_, Tech Lead, Developer
 Responsibilities: Choice of technical stack, code reviews for major changes
 - _lgamba_: Product Owner, Developer
 Responsibilities: Creating and updating pending tickets following team meetings, defining wanted features. DevOps.
 - _thschnei_: Product Owner, Developer
 Responsibilities: UX, UI, Gamification and design choices and reviews of related tickets.

# Project Management:

Our organization was roughly as follows:
 - Weekly meetings to discuss new modules and project advancement
 - Daily through a messaging app for support, administrative tasks, and general discussions around the project

We followed a simplified version of [Feature Driven Development](https://en.wikipedia.org/wiki/Feature-driven_development#Design_by_feature), where we had people test our application as we iterated on design choices.
'Simplified' because we decided on the basic set of features as required by the subject: authentication, containerization, DevOps, etc.

We adopted this method as soon as we started the project, see [DEV.md](./docs/DEV.md). 
It proved efficient when working on separate features, especially at prevention friction between branches.

To summarize our method works as follows:
 - Define *modules*: modules are a complete set of features that we want to implement, for instance: user profiles, chat, cards, etc.
 - Inside each module, define *tickets*. Tickets are meant to represent one unit of work that shouldn't be split into smaller items.
 For instance: `auth: account deletion`.
 The purpose of defining tickets is to have something that can be tested and verified against a given specification.
 While also being possible to review independently, and trying to be the least blocking possible for other team members.
 - Each ticket is implemented in its own branch, which makes it easy to check out while reviewing
 - Each review must first pass the CI (through GitHub action) before it can be reviewed.
 Then the ticket is reviewed by a member of the team before it can be merged.

For the sake of clarity we also opted in favor of using [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

Additionally, we ran `golangci-lint` and `eslint` on every commit, in order to enforce coding style guidelines.
As we weren't experienced enough in both languages, we used basic settings for both tools. They proved very valuable and caught numerous bugs that would otherwise have made it to `master`.

#### Returns

Overall this organization method proved very useful for things like one shot feature (chat, profiles, authentication). However, some items weren't properly described as tickets: accessibility, responsiveness, style/theme. 

The likely explanation is that we should probably have set higher standards from the beginning of the project, so accessibility and responsiveness would have been continuously reviewed and implemented.

That said, this mode of organization was really helpful when someone was stuck on a ticket.
Because they could make a partial PR, and as long as they didn't break API for what was already working, other people could build on top of their branch.

## GitHub Actions

We extensively used GitHub Action through this project.
 * It is free for public repositories
 * It is a practical tool to use when the project starts to grow and requires multiple tools to run all checks and tests
 * It helped us find many bugs through the codebase

Mainly we ran tests for the backend and linters. As the project grew, backend tests proved very practical at preventing regressions.

# Technical Stack:
 - `gin` (go) Go framework for backend web
 - `gorm` (go) A go ORM
 - `e-layout` (CSS/JS) A light CSS framework dedicated to layouts
 - `MariaDB` (DB) A database compatible with MySQL
 - `SvelteJS` (JS/TS) Frontend framework

We decided to use `Go` for our backend because it is a very simple language and it's syntax/semantics are very similar to C and C++; which we all had experience with.

Once we had chosen `Go`, we went with `gin` and `gorm`.
Go by itself is fully capable of powering web applications, however `gin` handles a lot of the work we would have had to do, if we were using a plain go HTTP server.
`gorm` followed naturally because it is really easy to use with `gin` and handled our migrations automatically until the very end.

`Go` has a fantastic HTML template rendering library built-in. So we decided to use it and use SSE on most of the application.
We decided to use `Svelte` as our frontend framework because it worked well with the 'islands' design: go renders the page and Svelte injects code the make the page interactive.


 # Database Schema:

 ```mermaid
erDiagram
    User {
        uint ID PK
        string Email UK
        string Username UK
        uint Rank
        uint Coins
    }
    UserSession {
        uint ID PK
        uint UserID FK
        string Token UK
        time ExpiresAt
    }
    UserProfile {
        uint UserID PK,FK
        string Bio
        time BirthDate
    }
    DailyStreak {
        uint UserID PK,FK
        uint Day
        uint Best
    }
    Card {
        uint ID PK
        uint OwnerID FK
        uint CardID "catalog id"
        string Foil
    }
    Booster {
        uint ID PK
        uint OwnerID FK
        uint CollectionID "catalog id"
        uint Size
    }
    BoosterPile {
        uint OwnerID PK,FK
        uint Count
    }
    Trade {
        uint ID PK
        uint FromID FK
        uint ToID FK
        string State
        uint Coins
        uint WantCoins
    }
    TradeLine {
        uint ID PK
        uint TradeID FK
        string Side
        uint CardID "catalog id"
        string Foil
        uint Count
    }
    Relation {
        uint ID PK
        uint UserID FK
        uint TargetID FK
        string Type
    }
    Notification {
        uint ID PK
        uint UserID FK
        string Type
        time ReadAt
    }
    ShoutboxMessage {
        uint ID PK
        uint UserID FK
        string Body
    }
    PrivateMessage {
        uint ID PK
        uint SenderID FK
        uint RecipientID FK
        string Body
        time ReadAt
    }
    ShoutboxMute {
        uint UserID PK,FK
        uint ModeratorID FK
        time Until
    }
    UserIgnore {
        uint UserID PK,FK
        uint IgnoredID PK,FK
        time Until
    }

    User ||--o{ UserSession : "has"
    User ||--o| UserProfile : "has"
    User ||--o| DailyStreak : "has"
    User ||--o{ Card : "owns"
    User ||--o{ Booster : "owns"
    User ||--o| BoosterPile : "owns"
    User ||--o{ Trade : "sends (From)"
    User ||--o{ Trade : "receives (To)"
    Trade ||--o{ TradeLine : "contains"
    User ||--o{ Relation : "user"
    User ||--o{ Relation : "target"
    User ||--o{ Notification : "receives"
    User ||--o{ ShoutboxMessage : "posts"
    User ||--o{ PrivateMessage : "sends"
    User ||--o{ PrivateMessage : "receives"
    User ||--o| ShoutboxMute : "muted"
    User ||--o{ ShoutboxMute : "moderator"
    User ||--o{ UserIgnore : "ignores"
    User ||--o{ UserIgnore : "ignored"
```

As the project is centered around the user and their gamified experience, almost every table directly depends on the `users` table.

# Features List:
 - User Profiles: jeel-qai
 Profile page for each user with customizable avatar, bio, username, etc.
 - Translations and global UI: lgamba, thschnei
 Fully responsive UI with language switcher
 - Authentication: lgamba
 Easy registration, login & account management through forms
 - Notifications: gdelhota
 Text notifications system with redirection and read statuses
 - Friend system: gdelhota
 User relations (friends/blocked)
 - Chat: lgamba
 Real-time public chat and private messages using WebSockets
 - Shop: thschnei
 Simple economy with virtual money and card boosters buying
 - Inventory: thschnei
 Extensive Inventory system with search filtering, pagination and fast visual loading
 - Card Appearance: thschnei, lgamba
 Random Foil patterns on cards and pretty animations on mouse hover.
 - Gamification: thschnei, lgamba
 Daily bonuses, design choices and animations for maximum user engagement.

# Modules

*Overall, we've decided to defend the following modules.*

## Web (12 points)

>**Major** Backend + Frontend framework

Gin (back) and Svelte (front)

>**Major** Real-time feature with WebSockets

Private messages and global chat
Assignees: _lgamba_

>**Major** User <-> User interactions

Chat, profile, friend requests, blocking
Assignees: _lgamba_ _jeel-qai_ _gdelhota_

>**Major** Standard User Management

Profile modifications, avatar, friends, online status, profile page
Assignees: _jeel-qai_ _gdelhota_

>**Minor** ORM for the database

Gorm

>**Minor** Complete Notification System

Notifications for all user relations actions
Assignees: _gdelhota_

>**Minor** Server-side Rendering

Through go's `html/templates` and Svelte islands

>**Minor** Advanced search

Filters, sorting and pagination through `bleve` and `gorm`.
Assignees: _lgamba_

## Accessibility and Internationalization (3 points)

>**Major** WCAG AA Compatibility

Verified through the site with Lightroom and axe. All pages are keyboard-navigation friendly.
Assignees: _thschnei_

>**Minor** Additional browsers

Website has been tested and works on Firefox, Chromium, Chrome (Android), Fennec, Edge and Brave.

## User Management (4 points)

>**Major** Standard user management and authentication

Customizable profile page with avatar and online status, friend system
Assignees: _jeel-qai_, _gdelhota_

>**Major** Advanced permissions system

CRUD, admin roles
Assignees: _jeel-qai_

## Devops (4 points)

>**Major** ELK Log Management

Assignees: _lgamba_

>**Major** Monitoring with Grafana and Prometheus

Assignees: _lgamba_

## Total: 23 points


 # Individual Contributions:

 ## gdelhota
 Friends system and Notifications

 ## jeel-qai
 Profile page, admin permissions, CRUD

 ## lgamba
 Devops, Authentication, Chat, Advanced search

 ## thschnei
Card collection & Inventory, shop and booster opening, gamified experience
