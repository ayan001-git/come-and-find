# TSIS 3 — Weekly Plan & Fact Report

**Project:** Come and Find  
**Week:** Week 3  
**Dates:** 14–20 September 2026  
**PM:** [Member 1 — real name]  
**Overall status:** **Delayed**

## Week summary

The main goal of the week was to move Come and Find from the TSIS 1–2 initiation stage into concrete technical planning. The team already had the project concept, stakeholders, RACI, Idea List and MVP feature priorities, but development had not started yet. During the week the team did not complete the planned technical artifacts, so the overall status is **Delayed**. The recovery action is to freeze the prototype category and frontend choice, prepare the database/API structure and start the first implementation sprint.

## Plan for the week

1. Select one product category for the prototype.
2. Finalize the frontend technology.
3. Create the first database model for users, suppliers, products and orders.
4. Draft the REST API endpoints for the MVP.
5. Create the first buyer/supplier/admin user-flow wireframes.
6. Prepare the Git repository structure and implementation tasks.

## Plan & Fact

| Task | Planned result | Actual result | Status | Blocker | Next step |
|---|---|---|---|---|---|
| Product category | One category selected for realistic demo data | Not finalized during the reporting week | Delayed | Team focused on TSIS 1–2 submission | Freeze **computer & office accessories** as prototype category |
| Frontend technology | Frontend stack agreed | TSIS 1–2 still stated “to be finalized” | Delayed | No final team decision | Use HTML/CSS/JavaScript for a small, defendable MVP |
| Database design | Initial ER/database structure | Only high-level entities were known | At Risk | No implementation session | Create PostgreSQL tables for users, suppliers, categories, products, cart, orders and order items |
| REST API design | Endpoint list for buyer/supplier/admin flows | Not completed in the reporting week | Delayed | Database structure not frozen | Define auth, catalogue, cart, order, supplier and admin endpoints |
| UI/user flows | Wireframes for catalogue and dashboards | Workflow existed only as TSIS 1–2 process description | At Risk | Frontend choice unresolved | Build catalogue, product detail, cart and role dashboards |
| Repository setup | Runnable starter project | Not started | Delayed | Technical decisions not finalized | Create one Go project with `/web`, database schema and README |

## What we learned

- A broad project idea is not enough for development; the prototype needs one concrete product category and demo dataset.
- The MVP should follow the **Must** features from TSIS 1–2 before adding reviews, favourites or advanced comparison.
- For a three-person team, a simple frontend reduces integration risk while preserving the planned Go + REST API + PostgreSQL direction.
- Weekly reporting is more useful when delayed work is recorded honestly and connected to a specific recovery action.

## AI Disclosure

Generative AI was used to help structure the report and improve wording. The project status and factual statements must be reviewed by the team before submission.
