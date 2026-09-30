# TSIS 5 — Project Charter & Force Field Analysis

**Project:** Come and Find  
**Group:** [Group number]  
**PM:** [Member 1 — real name]  
**Date:** 24 September 2026 (planned launch meeting)

## 1. Launch summary

**Planned launch date:** 24 September 2026  
**Attendees:** [Member 1] — PM/QA; [Member 2] — Backend/Database; [Member 3] — Frontend/UI/UX.  
**Sponsor:** Course instructor.

Agreements for the project launch:

- The prototype category is **computer and office accessories**; example products include keyboards, mice, webcams, storage devices and monitors.
- The MVP keeps the TSIS 1–2 **Must** scope: registration/login, catalogue/product details, search/filtering, supplier information, cart/order creation, admin product approval and order status.
- **Simulated payment** and a **supplier dashboard** are included after the Must flow because they demonstrate the complete end-to-end scenario without a real bank gateway.
- The team will use **2-week sprints** and a hybrid approach: course milestones/charter are plan-driven, while implementation is managed as short backlog-driven iterations.
- Reviews/ratings, favourites, advanced comparison, real banking integration and production logistics are outside the charter MVP.

## 2. Project Charter

### Purpose

Buyers currently spend time searching across disconnected sources for suitable products and reliable suppliers. Come and Find provides one searchable web catalogue where users can review product information, identify the supplier, create an order and track the order state. For the semester prototype, the catalogue is focused on computer and office accessories in Kazakhstan so the team can demonstrate a realistic but controlled buyer–supplier workflow.

**Business Case.** The project creates user value by reducing fragmented product search and making supplier information visible in the same flow as ordering. For the course, it also provides a complete IT project on which the team can apply project-management artifacts from initiation through charter, backlog, implementation, testing and final demonstration. The semester MVP is not expected to generate real revenue; success is a working, testable prototype that demonstrates the promised workflow.

### Objectives

1. Deliver a locally runnable and deployable Come and Find MVP by **22 November 2026** using Go, PostgreSQL and a web frontend.
2. Implement and demonstrate all eight **Must** capabilities from TSIS 1–2: authentication, catalogue, search/filtering, product details, supplier information, cart, order creation/status and admin product approval.
3. Enable at least **3 peer testers** to complete the buyer flow “find product → inspect supplier → add to cart → create order → simulate payment” in **5 minutes or less** without team assistance.
4. Demonstrate the supplier/admin workflow: supplier submits a product, admin approves it, and the supplier can process a paid order through at least `processing` and `shipped` states.
5. Keep scope changes controlled: no new Must feature enters a sprint unless the PM and the two implementation owners agree and the change is recorded in the project notes.

### Overview

**Management approach.** The team uses a hybrid model. Fixed course submissions, deadlines and the charter are treated as plan-driven milestones. Software work is organized in 2-week sprints with a small prioritized backlog and a short review at the end of each sprint.

**Technical approach.** Backend: Go HTTP server exposing REST-style JSON endpoints. Database: PostgreSQL. Frontend: HTML, CSS and vanilla JavaScript. Source control: Git/GitHub. The application stores users, suppliers, categories, products, carts, orders and order items and provides separate buyer, supplier and admin flows.

**Out of scope:**
- Reviews and ratings.
- Favourites.
- Advanced product comparison.
- Real banking/payment gateway integration.
- Physical warehouse management and production-scale logistics.

### Schedules

| Milestone | Target |
|---|---|
| TSIS 4 competency matrix finalized | 25 September 2026 |
| Charter / Force Field approved by team | 2 October 2026 |
| Sprint 1 — project skeleton, DB, auth, basic catalogue | 28 Sep – 11 Oct |
| Sprint 2 — search/filter, product details, supplier info, admin approval | 12 Oct – 25 Oct |
| Sprint 3 — cart, order creation, status, simulated payment | 26 Oct – 8 Nov |
| Sprint 4 — supplier dashboard, end-to-end testing, deployment and fixes | 9 Nov – 22 Nov |
| Final polishing / course demo | Instructor-announced final demo date |

**Next two sprint goals.** Sprint 1 produces a runnable application with PostgreSQL, user roles and a public approved-product catalogue. Sprint 2 completes the discovery and verification flow so a buyer can find a product and understand who supplies it while an admin controls publication.

### Resources

- **Team capacity:** target **6 hours per person per week** (about 18 team-hours/week) outside class; adjust downward during exam-heavy weeks instead of silently expanding deadlines.
- **Software/tools:** Go, PostgreSQL, Git/GitHub, VS Code, browser developer tools, optional Docker Desktop, free/educational hosting if used.
- **Budget:** target **0 KZT** for the semester MVP; use local development and free tiers only.
- **Scope-change rule:** a new Must feature requires agreement by the PM, Backend/DB owner and Frontend/UX owner and must be written into the backlog before work starts. Could features cannot replace an unfinished Must feature.

### Personnel

- **[Member 1] — Project Manager + QA/Documentation:** coordinates scope, weekly reports, acceptance checklist and final presentation.
- **[Member 2] — Backend Developer + Database:** owns PostgreSQL schema, Go REST API, authentication and order logic.
- **[Member 3] — Frontend Developer + UI/UX:** owns catalogue/product/cart/dashboard UI and frontend integration.
- **Sponsor:** Course instructor.

This 3-person assignment preserves the TSIS 1–2 RACI responsibility areas; QA is combined with the PM role because the actual team has three members.

### Risk Management Plans

| Risk | Impact | Mitigation |
|---|---|---|
| Scope creep from reviews, favourites or advanced comparison | High | Keep Could features outside the MVP until all Must acceptance checks pass |
| Three people must cover four responsibility areas | High | Combine PM + QA; use clear task ownership and pair only on integration points |
| Skill gap in Go/PostgreSQL/frontend integration | High | Pair backend and frontend owners on API contracts; maintain example requests/responses |
| Academic workload / exams reduce available hours | Medium–High | Commit to smaller sprint scope; protect Must features; move polish/Should items later |
| Local/deployment environment differences | Medium | Keep one documented Docker path and one native PostgreSQL path; rehearse deployment before final week |

### Evaluation Methods

The project will be evaluated by the course instructor through required TSIS artifacts, consistency between the charter and delivered scope, and the final working demonstration. The team will use a Must-feature acceptance checklist and record defects before the final demo.

User testing criteria:
1. At least 3 peer testers can find a product, open its details, identify the supplier, add it to cart and create/pay a simulated order in 5 minutes or less.
2. In an admin/supplier test, a newly submitted product remains hidden until approval and becomes visible in the public catalogue after approval.

## 3. Force Field Analysis

**Central MVP statement:** **“Deliver a working Come and Find marketplace MVP for computer and office accessories, demonstrating buyer, supplier and admin workflows by 22 November 2026.”**

| Driving force (+) | Strength | Why it helps | Restraining force (−) | Strength | Why it blocks |
|---|---:|---|---|---:|---|
| Clear buyer problem and workflow already defined in TSIS 1–2 | 5 | Team does not need to redesign the project concept | Only three team members | 5 | Each person covers more than one responsibility area |
| Must/Should/Could priorities already exist | 5 | Gives a defensible order of implementation | Skill gaps across full stack | 4 | Integration can take longer than isolated coding |
| Go + PostgreSQL direction already selected | 4 | Reduces technology-choice uncertainty | Academic workload and parallel courses | 4 | Available weekly hours can change |
| Product category is now narrowed for the prototype | 4 | Makes seed data and testing concrete | Scope creep from attractive Could features | 4 | Extra features can delay the core flow |
| Open-source tools and zero-budget target | 3 | No procurement dependency | Deployment/environment differences | 3 | “Works on my machine” risk before demo |

### Actions

1. **Freeze the Must scope** and move reviews/favourites/comparison to a parking list — **Owner: PM, by 25 September**.
2. **Create and review the API/database contract** before frontend integration — **Owner: Backend/DB member, by 2 October**.
3. **Run one full buyer → payment → supplier → admin test every sprint review** — **Owner: PM/QA, starting 11 October**.
4. **Rehearse deployment from a clean machine/environment** — **Owner: Backend/DB member with Frontend member, by 8 November**.
5. **Book three peer testers** and use the same 5-minute scenario — **Owner: Frontend/UX member, by 15 November**.

## 4. AI Disclosure

Generative AI was used to help structure the charter, convert previously selected features into measurable objectives, and improve wording. The team must review the dates, personnel names, hours/week estimate, competency assumptions and risk priorities before submission.
