# TSIS 4 — Team Competencies

**Project:** Come and Find  
**Team size:** 3  
**Date:** 23 September 2026

> **Before submission:** replace the three member placeholders with real names and confirm the competency levels. The role split below is designed to match the RACI from TSIS 1–2 without inventing a fourth person.

## Team

| Member | Role |
|---|---|
| **[Member 1 — real name]** | Project Manager + QA / Documentation |
| **[Member 2 — real name]** | Backend Developer + Database |
| **[Member 3 — real name]** | Frontend Developer + UI/UX |

The team keeps four RACI responsibility areas (PM, DEV, UX and QA), but because the real team has three people, the PM also coordinates QA/documentation while the developer roles stay specialized.

## Required competencies

| Competency | Required level | Why it is required for Come and Find |
|---|---|---|
| Go / REST API development | Intermediate | Authentication, catalogue, cart, order, supplier and admin endpoints |
| PostgreSQL / SQL | Intermediate | Users, suppliers, products, carts, orders and reporting data |
| HTML / CSS / JavaScript | Intermediate | Buyer catalogue, product pages, cart and role dashboards |
| UI/UX design | Intermediate | Search/filter flow, product details and understandable checkout/dashboard screens |
| Git / GitHub collaboration | Intermediate | Three members must work on one codebase without overwriting each other |
| API & functional testing | Intermediate | Verify login, product approval, ordering and role permissions |
| Project planning & weekly reporting | Intermediate | TSIS reports, sprint goals, scope tracking and milestone preparation |
| Team communication / coordination | Intermediate | Soft skill needed to manage dependencies between backend, frontend and testing |

## Current levels — provisional self-assessment to confirm

| Competency | [Member 1 — PM/QA] | [Member 2 — Backend/DB] | [Member 3 — Frontend/UX] |
|---|---|---|---|
| Go / REST API development | Beginner | Intermediate | Beginner |
| PostgreSQL / SQL | Beginner | Intermediate | Beginner |
| HTML / CSS / JavaScript | Beginner | Beginner | Intermediate |
| UI/UX design | Intermediate | Beginner | Intermediate |
| Git / GitHub collaboration | Intermediate | Intermediate | Intermediate |
| API & functional testing | Intermediate | Intermediate | Beginner |
| Project planning & weekly reporting | Intermediate | Beginner | Beginner |
| Team communication / coordination | Intermediate | Intermediate | Intermediate |

## Gaps & actions

- **Frontend-to-API integration:** [Member 3] needs more REST integration practice. [Member 2] will document request/response examples and pair with [Member 3] for one integration session **by 25 September**.
- **PostgreSQL confidence outside the backend role:** [Member 1] and [Member 3] will review the project schema and run five basic SQL queries against users/products/orders **by 27 September**.
- **End-to-end testing:** [Member 3] is currently weaker in API/functional testing. [Member 1] will prepare a buyer → payment → supplier → admin test checklist and run it together with [Member 3] **by the end of Sprint 1 (11 October)**.
- **Deployment:** no member should be assumed Advanced in deployment. [Member 2] will own the first deployment rehearsal using the Docker setup and document the exact commands **by 2 October**.
- **Workload concentration:** the three-person team has four RACI responsibility areas. The team will keep Must features fixed and postpone Could features until the complete buyer–supplier–admin workflow is stable.

## AI Disclosure

Generative AI was used to structure the competency tables and suggest concrete gap-closing actions. The team must replace placeholders and confirm all self-assessed skill levels before submission.
