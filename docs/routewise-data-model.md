# RouteWise Data Model Documentation

**Version:** 1.0  
**Date:** January 28, 2026  
**Database:** PostgreSQL 15.x  
**Migrations:** 000001 - 000007

---

## Architecture Overview

**Multi-tenant SaaS:** Organization-based isolation  
**User Types:** 
- Office Users (web dashboard) - `organization_users` table
- Field Workers (mobile app) - `workers` table

**Polymorphic Relationships:** Files and assignments support both user types

---

## Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        ORGANIZATIONS                             │
│  (Root - Multi-tenant isolation)                                │
└────────────┬────────────────────────────────────────────────────┘
             │
             │ Has Many
             │
     ┌───────┼───────┬──────────────┬──────────────┐
     │       │       │              │              │
     ▼       ▼       ▼              ▼              ▼
┌─────────┐ ┌──────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│ ORG     │ │WORKER│ │CUSTOMER  │ │  JOBS/   │ │  ...     │
│ USERS   │ │      │ │          │ │ PROJECTS │ │          │
└────┬────┘ └──┬───┘ └─────┬────┘ └────┬─────┘ └──────────┘
     │         │            │           │
     │         │            └───────────┼─── Belongs To ───┐
     │         │                        │                  │
     │         │                        │ Has Many         │
     │         │                        │                  │
     │         │                        ▼                  │
     │         │              ┌──────────────────┐         │
     │         │              │ PROJECT_FILES    │         │
     │         │              │                  │         │
     │         └──────────────┤ Uploaded By:     │         │
     │                        │ - user OR worker │         │
     │                        │ (polymorphic)    │         │
     └────────────────────────┤                  │         │
                              └──────────────────┘         │
                                                           │
                                      ┌────────────────────┘
                                      │
                                      │ Has Many
                                      │
                              ┌───────▼──────────┐
                              │ PROJECT          │
                              │ ASSIGNMENTS      │
                              │                  │
                              │ Assigned To:     │
                ┌─────────────┤ - user OR worker │
                │             │ (polymorphic)    │
                │             └──────────────────┘
                │
                │
    ┌───────────┴────────────┐
    │                        │
    ▼                        ▼
┌─────────┐              ┌──────┐
│ ORG     │              │WORKER│
│ USERS   │              │      │
└─────────┘              └──────┘
```

---

## Table Definitions

### 1. organizations

**Purpose:** Root entity for multi-tenant isolation. Each company is an organization.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | SERIAL | PRIMARY KEY | Unique organization ID |
| name | VARCHAR(255) | NOT NULL | Company name |
| phone | VARCHAR(20) | | Company phone |
| industry | VARCHAR(50) | | 'construction', 'hvac', etc. |
| created_at | TIMESTAMP | DEFAULT NOW() | Record creation time |
| updated_at | TIMESTAMP | DEFAULT NOW() | Last update time |

**Indexes:**
- PRIMARY KEY on `id`

**Relationships:**
- Has many `organization_users`
- Has many `workers`
- Has many `customers`
- Has many `jobs`

---

### 2. organization_users

**Purpose:** Office staff who manage via web dashboard (owners, admins, managers).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | SERIAL | PRIMARY KEY | Unique user ID |
| organization_id | INTEGER | NOT NULL, FK → organizations(id) | Organization membership |
| email | VARCHAR(255) | NOT NULL, UNIQUE | Login email |
| password | VARCHAR(255) | NOT NULL | bcrypt hashed password |
| name | VARCHAR(255) | NOT NULL | Full name |
| phone | VARCHAR(20) | | Contact phone |
| role | VARCHAR(50) | NOT NULL, CHECK | 'owner', 'admin', 'manager' |
| created_by | INTEGER | FK → organization_users(id), NULLABLE | Who added this user |
| created_at | TIMESTAMP | DEFAULT NOW() | Record creation time |
| updated_at | TIMESTAMP | DEFAULT NOW() | Last update time |

**Constraints:**
- `CHECK (role IN ('owner', 'admin', 'manager'))`

**Indexes:**
- PRIMARY KEY on `id`
- INDEX on `organization_id`
- INDEX on `email`
- INDEX on `created_by`

**Relationships:**
- Belongs to `organizations`
- Created by `organization_users` (self-referential, nullable)
- Creates `workers` (via `workers.created_by`)
- Can upload to `project_files` (via `uploaded_by_user`)
- Can be assigned to `project_assignments` (polymorphic)

**Authentication:** Email + Password  
**Access:** Web dashboard

---

### 3. workers

**Purpose:** Field workers who use mobile app (technicians, crew members).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | SERIAL | PRIMARY KEY | Unique worker ID |
| organization_id | INTEGER | NOT NULL, FK → organizations(id) | Organization membership |
| name | VARCHAR(255) | NOT NULL | Full name |
| phone | VARCHAR(20) | NOT NULL, UNIQUE | Mobile phone (for login) |
| email | VARCHAR(255) | NULLABLE | Optional email |
| role | VARCHAR(50) | NULLABLE | 'foreman', 'electrician', 'technician', etc. |
| is_active | BOOLEAN | DEFAULT true | Soft delete flag |
| created_by | INTEGER | FK → organization_users(id), NULLABLE | Who added this worker |
| created_at | TIMESTAMP | DEFAULT NOW() | Record creation time |
| updated_at | TIMESTAMP | DEFAULT NOW() | Last update time |

**Constraints:**
- `UNIQUE(organization_id, phone)`

**Indexes:**
- PRIMARY KEY on `id`
- INDEX on `organization_id`
- INDEX on `phone`
- INDEX on `organization_id, is_active`
- INDEX on `created_by`

**Relationships:**
- Belongs to `organizations`
- Created by `organization_users`
- Can upload to `project_files` (via `uploaded_by_worker`)
- Can be assigned to `project_assignments` (via `worker_id`)

**Authentication:** Phone + Password (or OTP - future)  
**Access:** Mobile app  
**Note:** Separate from organization_users to avoid nullable password constraints

---

### 4. customers

**Purpose:** End customers who need services (homeowners, businesses).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | SERIAL | PRIMARY KEY | Unique customer ID |
| organization_id | INTEGER | NOT NULL, FK → organizations(id) | Organization ownership |
| name | VARCHAR(255) | NOT NULL | Customer name |
| email | VARCHAR(255) | NULLABLE | Contact email |
| phone | VARCHAR(20) | NULLABLE | Contact phone |
| address | TEXT | NULLABLE | Customer address |
| created_at | TIMESTAMP | DEFAULT NOW() | Record creation time |
| updated_at | TIMESTAMP | DEFAULT NOW() | Last update time |

**Indexes:**
- PRIMARY KEY on `id`
- INDEX on `organization_id`

**Relationships:**
- Belongs to `organizations`
- Has many `jobs`

---

### 5. jobs (Projects/Service Calls)

**Purpose:** Work orders - can be HVAC service calls OR construction projects.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | SERIAL | PRIMARY KEY | Unique job/project ID |
| organization_id | INTEGER | NOT NULL, FK → organizations(id) | Organization ownership |
| customer_id | INTEGER | FK → customers(id), NULLABLE | Customer for this job |
| created_by | INTEGER | FK → organization_users(id), NULLABLE | Who created this job |
| technician_id | INTEGER | FK → workers(id), NULLABLE | **HVAC:** Single tech assignment |
| title | VARCHAR(255) | NOT NULL | Job/project title |
| description | TEXT | NULLABLE | Detailed description |
| status | VARCHAR(50) | | 'pending', 'in_progress', 'completed', 'cancelled' |
| address | TEXT | NULLABLE | Job site address |
| scheduled_date | TIMESTAMP | NULLABLE | When job starts |
| completed_at | TIMESTAMP | NULLABLE | When finished |
| duration_minutes | INTEGER | NULLABLE | Estimated/actual duration |
| price | DECIMAL(10,2) | NULLABLE | Job cost |
| project_type | VARCHAR(100) | NULLABLE | **Construction:** 'residential', 'commercial', 'renovation' |
| project_value | DECIMAL(12,2) | NULLABLE | **Construction:** Total project cost |
| estimated_duration | INTEGER | NULLABLE | **Construction:** Days or **HVAC:** Hours |
| actual_start_date | DATE | NULLABLE | **Construction:** Actual start |
| actual_end_date | DATE | NULLABLE | **Construction:** Actual end |
| metadata | JSONB | NULLABLE | Flexible industry-specific data |
| created_at | TIMESTAMP | DEFAULT NOW() | Record creation time |
| updated_at | TIMESTAMP | DEFAULT NOW() | Last update time |

**Indexes:**
- PRIMARY KEY on `id`
- INDEX on `organization_id`
- INDEX on `customer_id`
- INDEX on `technician_id`
- INDEX on `status`
- INDEX on `scheduled_date`

**Relationships:**
- Belongs to `organizations`
- Belongs to `customers` (nullable)
- Created by `organization_users` (nullable)
- Has one `worker` assigned via `technician_id` (HVAC pattern)
- Has many `project_files`
- Has many `project_assignments` (Construction pattern)

**Industry Usage:**
- **HVAC:** Short service calls, single tech via `technician_id`
- **Construction:** Long projects, multiple workers via `project_assignments`

---

### 6. project_files

**Purpose:** Files uploaded to projects (photos, PDFs, documents).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | SERIAL | PRIMARY KEY | Unique file ID |
| project_id | INTEGER | NOT NULL, FK → jobs(id) ON DELETE CASCADE | Project ownership |
| uploaded_by_user | INTEGER | FK → organization_users(id), NULLABLE | If office user uploaded |
| uploaded_by_worker | INTEGER | FK → workers(id), NULLABLE | If worker uploaded |
| file_type | VARCHAR(50) | NOT NULL | 'photo', 'document', 'report' |
| file_category | VARCHAR(50) | NULLABLE | 'progress', 'contract', 'site_photo', 'invoice' |
| file_name | VARCHAR(255) | NOT NULL | Display filename |
| original_file_name | VARCHAR(255) | NOT NULL | Original upload filename |
| mime_type | VARCHAR(100) | NOT NULL | 'image/jpeg', 'application/pdf' |
| file_size | INTEGER | NULLABLE | Bytes |
| file_extension | VARCHAR(10) | NULLABLE | 'jpg', 'pdf' |
| s3_bucket | VARCHAR(100) | NOT NULL | AWS S3 bucket name |
| s3_key | TEXT | NOT NULL | S3 object key |
| s3_url | TEXT | NULLABLE | Presigned download URL (temporary) |
| description | TEXT | NULLABLE | User-provided description |
| taken_at | TIMESTAMP | NULLABLE | When photo was taken |
| created_at | TIMESTAMP | DEFAULT NOW() | Upload time |
| updated_at | TIMESTAMP | DEFAULT NOW() | Last update time |

**Constraints:**
- `CHECK (uploaded_by_user IS NOT NULL AND uploaded_by_worker IS NULL) OR (uploaded_by_user IS NULL AND uploaded_by_worker IS NOT NULL)`

**Indexes:**
- PRIMARY KEY on `id`
- INDEX on `project_id`
- INDEX on `file_type`
- INDEX on `file_category`
- INDEX on `uploaded_by_user`
- INDEX on `uploaded_by_worker`
- INDEX on `created_at DESC`

**Relationships:**
- Belongs to `jobs` (projects)
- Uploaded by `organization_users` OR `workers` (polymorphic - only one is set)

**Storage:**
- Files stored in AWS S3
- Database stores metadata only
- S3 key format: `organizations/{org_id}/projects/{project_id}/{type}/{timestamp}_{filename}`

---

### 7. project_assignments

**Purpose:** Track which workers are assigned to which projects (Construction pattern).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | SERIAL | PRIMARY KEY | Unique assignment ID |
| project_id | INTEGER | NOT NULL, FK → jobs(id) ON DELETE CASCADE | Project reference |
| worker_id | INTEGER | NOT NULL, FK → workers(id) ON DELETE CASCADE | Worker reference |
| role | VARCHAR(50) | NULLABLE | Worker's role on this project |
| assigned_at | TIMESTAMP | DEFAULT NOW() | When assigned |
| removed_at | TIMESTAMP | NULLABLE | When removed (NULL = still assigned) |

**Constraints:**
- `UNIQUE(project_id, worker_id, assigned_at)` - Prevent duplicate assignments

**Indexes:**
- PRIMARY KEY on `id`
- INDEX on `project_id`
- INDEX on `worker_id`
- INDEX on `(project_id, worker_id) WHERE removed_at IS NULL`

**Relationships:**
- Belongs to `jobs` (projects)
- Belongs to `workers`

**Usage:**
- **Construction:** Multiple workers per project, long-term assignments
- **HVAC:** Optional (can use `jobs.technician_id` for single tech)

**Active assignment:** `removed_at IS NULL`  
**Removed assignment:** `removed_at IS NOT NULL`

---

## Key Design Patterns

### Multi-Tenant Isolation
Every table (except `organizations`) has `organization_id` foreign key.  
All queries filter by `organization_id` to prevent cross-tenant data access.

### Two User Types
- **organization_users:** Office staff, email/password auth, web access
- **workers:** Field staff, phone auth, mobile access
- Separate tables avoid nullable password constraints

### Polymorphic Relationships
**project_files:**
- `uploaded_by_user` → office user uploaded
- `uploaded_by_worker` → worker uploaded
- CHECK constraint ensures only one is set

**project_assignments (future):**
- Could support both user types via polymorphic pattern
- Currently only workers

### Soft Deletes
**workers.is_active:**
- `true` = active worker
- `false` = deactivated (soft deleted)
- Preserves audit trail and historical data

### Self-Referential
**organization_users.created_by:**
- References `organization_users(id)`
- Tracks who added each user
- NULL for self-registered owners

### Industry Flexibility
**jobs table supports both:**
- **HVAC:** `technician_id`, `scheduled_date`, `duration_minutes`
- **Construction:** `project_assignments`, `project_type`, `project_value`, `actual_start_date`

---

## Migration History

| Migration | Description | Date |
|-----------|-------------|------|
| 000001 | Initial schema | Jan 2026 |
| 000002-000004 | Core tables | Jan 2026 |
| 000005 | Replace job_photos with project_files | Jan 28, 2026 |
| 000006 | Add construction fields & project_assignments | Jan 28, 2026 |
| 000007 | Consolidate technicians → workers | Jan 28, 2026 |

---

## Future Enhancements

### Considered but Deferred:
- **UUIDs:** Using integer IDs for MVP, can add UUIDs later if needed
- **Polymorphic assignments:** project_assignments currently workers-only
- **Manager role:** Only owner/admin for MVP
- **Complex permissions:** Simple owner/admin split for now
- **Hebrew i18n:** Database supports UTF-8, UI translation deferred

### Post-MVP (Week 4+):
- Materials/equipment tracking
- Time tracking
- Project phases/milestones
- Budget tracking
- Worker availability/scheduling
- Customer portal
- Reporting tables
- Audit log table

---

## Technology Stack

**Database:** PostgreSQL 15.x  
**Hosting:** AWS EC2 (planned migration to RDS)  
**File Storage:** AWS S3  
**Backend:** Go with Gin framework  
**Frontend Web:** React on Vercel  
**Frontend Mobile:** React Native  
**Authentication:** JWT tokens

---

## Security Considerations

**Multi-tenant isolation:** All queries scoped by organization_id  
**Soft deletes:** Workers deactivated, not deleted (audit trail)  
**File access:** Presigned S3 URLs with expiration  
**Authentication:** Separate auth flows for users vs workers  
**No PII leakage:** Sequential IDs are org-scoped, not global

---

## Notes

**HVAC vs Construction:**
- Same database schema supports both industries
- Different usage patterns (single tech vs crew)
- `technician_id` for HVAC, `project_assignments` for construction
- Both can coexist in same organization

**Permission Model:**
- Owner: Full control
- Admin: Cannot create other admins
- Workers: Mobile app only, no dashboard access

**File Management:**
- PostgreSQL stores metadata
- S3 stores actual files
- Polymorphic uploader tracking

**Current State:**
- Production ready
- File upload working
- Ready for mobile app (Week 2)
- First customer demo in 13 days

---

**Generated:** January 28, 2026  
**Status:** Production - Week 1 Complete ✅
