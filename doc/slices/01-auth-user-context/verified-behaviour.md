# Verified Behaviour

## Status

Forty-one claims extracted under `AUTH-EXT-001` through `AUTH-EXT-008` (covering identity resolution, password policy, session lifecycle, institution/site/firm context, RBAC inheritance, PIN & electronic signatures, schema migrations, and LDAP/SSO boundaries) have been independently verified against the legacy source code (`OpenEyes/openeyes/protected/`) and accepted.

## Accepted Claims

| Claim ID | Behaviour | Source | Verification |
| --- | --- | --- | --- |
| AUTH-CLAIM-0001 | LoginForm requires username, password, site_id, and institution_id (when institution_required setting is on). | `LoginForm.php:39-51,96-107` | Accepted via `AUTH-CLAIM-0001.yaml` |
| AUTH-CLAIM-0002 | SiteController actionLogin binds form POST, validates credentials when SSO is off or username is in local_users, sets session flags (`confirm_site_and_firm`, `shown_version_reminder`), and redirects. | `SiteController.php:160-168,188-230` | Accepted via `AUTH-CLAIM-0002.yaml` |
| AUTH-CLAIM-0003 | UserIdentity resolves available authentications at construction and evaluates EXACT_MATCH before PERMISSIVE_MATCH. | `UserIdentity.php:47-80`, `LoginForm.php:126-140` | Accepted via `AUTH-CLAIM-0003.yaml` |
| AUTH-CLAIM-0004 | UserAuthentication::findAvailableAuthentications queries active=1 rows, returning exact/permissive matches, and distinguishes active=0 (deactivated account) from no-match invalid login. | `UserAuthentication.php:334-374` | Accepted via `AUTH-CLAIM-0004.yaml` |
| AUTH-CLAIM-0005 | Authentication fails and logs login-failed audit when no authentications exist or when multiple authentications exist for a match type. | `UserIdentity.php:61-70,83-89` | Accepted via `AUTH-CLAIM-0005.yaml` |
| AUTH-CLAIM-0006 | UserIdentity rejects inactive UserAuthentication (active != 1) or users missing OprnLogin permission with login-failed audit. | `UserIdentity.php:344-363` | Accepted via `AUTH-CLAIM-0006.yaml` |
| AUTH-CLAIM-0007 | Unsalted passwords use password_verify; legacy salted passwords verify using old hash, auto-encrypt to new password_verify format, clear salt, audit, and return success. | `UserAuthentication.php:244-274` | Accepted via `AUTH-CLAIM-0007.yaml` |
| AUTH-CLAIM-0008 | Password check fails if password status is inactive or softlocked; failed attempts increment failed_tries and log login-failed audit. | `UserIdentity.php:415-429` | Accepted via `AUTH-CLAIM-0008.yaml` |
| AUTH-CLAIM-0009 | setSessionDataForUser populates session institution, firm, site, user, user_auth. Throws Exception if user has no firm rights; selects firm/site defaults. | `UserIdentity.php:500-547,565-600` | Accepted via `AUTH-CLAIM-0009.yaml` |
| AUTH-CLAIM-0010 | Successful login updates last_successful_login_date on UserAuthentication and writes login-successful audit logs. | `UserIdentity.php:101-108,441-453,594-608` | Accepted via `AUTH-CLAIM-0010.yaml` |
| AUTH-CLAIM-0011 | LoginForm adds error message to both username and password fields on failure, and only calls web-user login when authenticated. | `LoginForm.php:126-161` | Accepted via `AUTH-CLAIM-0011.yaml` |
| AUTH-CLAIM-0012 | UserIdentityTest boolean assertions conflict with UserIdentity::authenticate() array return contract [bool, string]. | `UserIdentityTest.php:56-145`, `UserIdentity.php:61-80` | Accepted via `AUTH-CLAIM-0012.yaml` |
| AUTH-CLAIM-0013 | Password restrictions enforce min length (default 8), max length (70), and complexity regex requiring uppercase, lowercase, numeric, and special characters. | `PasswordUtils.php:30-63` | Accepted via `AUTH-CLAIM-0013.yaml` |
| AUTH-CLAIM-0014 | Password status hierarchy ranks severity: locked (0), softlocked (1), expired (2), current (3), stale (4). Status only changes if target rank <= current rank. | `PasswordUtils.php:19-25,117-139` | Accepted via `AUTH-CLAIM-0014.yaml` |
| AUTH-CLAIM-0015 | Incrementing failed tries tracks password_failed_tries; upon reaching pw_tries (default 3), status transitions to pw_tries_failed (default locked). | `PasswordUtils.php:141-159` | Accepted via `AUTH-CLAIM-0015.yaml` |
| AUTH-CLAIM-0016 | Softlocked status calculates and sets password_softlocked_until using pw_softlock_timeout (default '10 mins'). | `PasswordUtils.php:127-132` | Accepted via `AUTH-CLAIM-0016.yaml` |
| AUTH-CLAIM-0017 | Password expiry calculates age relative to password_last_changed_date against pw_days_stale, pw_days_expire, and pw_days_lock; local_users bypass expiry. | `PasswordUtils.php:161-191` | Accepted via `AUTH-CLAIM-0017.yaml` |
| AUTH-CLAIM-0018 | Identity changes force session ID regeneration via session_regenerate_id(true) to prevent session fixation vulnerabilities. | `OEWebUser.php:24-29` | Accepted via `AUTH-CLAIM-0018.yaml` |
| AUTH-CLAIM-0019 | Application init checks active sessions against DB session table via isValidSession(), forcing logout if blacklisted session ID matches. | `OEWebUser.php:37-44,50-60` | Accepted via `AUTH-CLAIM-0019.yaml` |
| AUTH-CLAIM-0020 | Logout invalidates sessions by storing '_' + session_id[1:] in session table with 48-hour expire and 'Logged out.' data payload. | `OEWebUser.php:46-48,62-76,82-86,111-114` | Accepted via `AUTH-CLAIM-0020.yaml` |
| AUTH-CLAIM-0021 | Login requirements check if URL contains '/eventImage', throwing HTTP 403 exception to preserve returnUrl integrity. | `OEWebUser.php:93-104` | Accepted via `AUTH-CLAIM-0021.yaml` |
| AUTH-CLAIM-0022 | Global firm rights (global_firm_rights == 1) grant access to all active institution firms, whereas global_firm_rights == 0 restricts available firms to explicit assignments in firm_user_assignment, user_firm_rights, or user_service_rights. | `User.php:636-657` | Accepted via `AUTH-CLAIM-0022.yaml` |
| AUTH-CLAIM-0023 | Firm context selection defaults to (1) last_firm_id if valid for institution, (2) first assigned firm in userFirms[0], or (3) key(firms); throws Exception if 0 firm rights exist. | `UserIdentity.php:525-548` | Accepted via `AUTH-CLAIM-0023.yaml` |
| AUTH-CLAIM-0024 | Site context selection defaults to (1) explicit site_id passed at login, (2) last_site_id if matching current institution, or (3) Site::getDefaultSite(); throws CException if unresolved. | `UserIdentity.php:580-593` | Accepted via `AUTH-CLAIM-0024.yaml` |
| AUTH-CLAIM-0025 | Changing active firm (User::changeFirm) updates last_firm_id and increments UserFirmPreference position order for recent context firm limit tracking. | `User.php:209-234` | Accepted via `AUTH-CLAIM-0025.yaml` |
| AUTH-CLAIM-0026 | RBAC evaluation supports namespaced business rules ('core' or module namespaces) via AuthManager::executeBizRule, dynamically invoking rule methods on registered ruleset instances. | `AuthManager.php:52-84` | Accepted via `AUTH-CLAIM-0026.yaml` |
| AUTH-CLAIM-0027 | AuthManager caches user auth assignments locally per request ($user_assignments[$user_id]) to minimize repetitive database assignment lookups. | `AuthManager.php:104-111` | Accepted via `AUTH-CLAIM-0027.yaml` |
| AUTH-CLAIM-0028 | Role assignment privileges (getAssignableRoles) restrict non-admin users from assigning the 'admin' role to other users. | `AuthManager.php:183-194` | Accepted via `AUTH-CLAIM-0028.yaml` |
| AUTH-CLAIM-0029 | User role updates (saveRoles) compute added and removed roles, calling assign and revoke, while triggering secondary module assignment handlers (e.g. OETrial permission sync). | `User.php:549-598` | Accepted via `AUTH-CLAIM-0029.yaml` |
| AUTH-CLAIM-0030 | PIN code generation enforces 6-digit zero-padded numbers, rejecting sequential digits, 6-digit repeated numbers, previously used PINs, or the global secretary_pin. | `PincodeHelper.php:5-52` | Accepted via `AUTH-CLAIM-0030.yaml` |
| AUTH-CLAIM-0031 | PIN code regeneration is rate-limited to a maximum threshold (PIN_REGEN_LIMIT = 5), preventing further user PIN regeneration once the limit is reached. | `ProfileController.php:282-327` | Accepted via `AUTH-CLAIM-0031.yaml` |
| AUTH-CLAIM-0032 | Revealing PIN codes requires password re-verification for local accounts, external identity verification for LDAP accounts, and bypasses password check for SSO authenticated sessions. | `ProfileController.php:225-248` | Accepted via `AUTH-CLAIM-0032.yaml` |
| AUTH-CLAIM-0033 | User electronic signatures are stored as protected file assets (signature_file_id), and signature decryption or verification requires valid PIN authentication. | `ProfileController.php:471-500`, `User.php:795` | Accepted via `AUTH-CLAIM-0033.yaml` |
| AUTH-CLAIM-0034 | Auth schema migration (m200517_044325) decoupled user credentials from legacy user table into normalized institution_authentication and user_authentication tables supporting LOCAL, LDAP, and SSO methods. | `m200517_044325...php:12-67` | Accepted via `AUTH-CLAIM-0034.yaml` |
| AUTH-CLAIM-0035 | The user_authentication schema encapsulates credential state fields including password_hash, password_salt, password_softlocked_until, password_last_changed_date, password_failed_tries, password_status, last_successful_login_date, and active flag. | `m200517_044325...php:48-67` | Accepted via `AUTH-CLAIM-0035.yaml` |
| AUTH-CLAIM-0036 | RBAC database tables (authitem, authitemchild, authassignment) structure permission inheritance across operations (type=0), tasks (type=1), and roles (type=2) linked to user IDs. | `AuthManager.php:20-40,130-172` | Accepted via `AUTH-CLAIM-0036.yaml` |
| AUTH-CLAIM-0037 | System service accounts (special_usernames such as docman_user, api, portal_user) bypass institution authentication by setting institution_authentication_id to NULL and having global firm rights. | `m200517_044325...php:173-195` | Accepted via `AUTH-CLAIM-0037.yaml` |
| AUTH-CLAIM-0038 | Native LDAP authentication (authenticateNativeLDAP) connects via server and port, performs an admin bind with ldap_admin_dn, searches for (sAMAccountName=$username), and verifies credentials with a user DN bind. | `UserIdentity.php:181-268` | Accepted via `AUTH-CLAIM-0038.yaml` |
| AUTH-CLAIM-0039 | Other LDAP authentication (authenticateOtherLDAP) binds directly using $ldap_user_prefix=$username,$ldap_dn and supports configurable search retries (ldap_info_retries) and delay (ldap_info_retry_delay). | `UserIdentity.php:270-331` | Accepted via `AUTH-CLAIM-0039.yaml` |
| AUTH-CLAIM-0040 | SSO / SAML / OIDC authentication delegates identity validation to external identity providers, provisioning user attributes (givenname, sn, mail) and bypassing local password checks. | `UserIdentity.php:170-179` | Accepted via `AUTH-CLAIM-0040.yaml` |
| AUTH-CLAIM-0041 | Multi-tenant LDAP configurations persist JSON settings in ldap_config (ldap_json) linked to institution_authentication to support per-institution directory integration. | `m200517_044325...php:19-46,124-165` | Accepted via `AUTH-CLAIM-0041.yaml` |

## Rejected Claims

| Claim ID | Rejected statement | Reason | Verification record |
| --- | --- | --- | --- |
| None | No claims rejected | N/A | N/A |

## Uncertain Claims

| Claim ID | Uncertain statement | Missing evidence | Next action |
| --- | --- | --- | --- |
| None | All extracted claims for AUTH-EXT-001 through AUTH-EXT-008 are verified | N/A | Complete Slice 1 Specification (AUTH-SPEC-001) |

## Verification Queue

| Claim range | Subject | Current state |
| --- | --- | --- |
| AUTH-CLAIM-0001-0004 | Form validation and credential resolution | Verified & Accepted |
| AUTH-CLAIM-0005-0008 | Authentication rejection, permission, password, and lock checks | Verified & Accepted |
| AUTH-CLAIM-0009-0011 | Session population, successful login audit, and form result handling | Verified & Accepted |
| AUTH-CLAIM-0012 | Legacy test and implementation contract mismatch | Verified & Accepted |
| AUTH-CLAIM-0013-0017 | Password restrictions, status hierarchy, failed tries escalation, softlock timeout, and password expiry | Verified & Accepted |
| AUTH-CLAIM-0018-0021 | Session ID regeneration, DB session validation, session revocation blacklisting, and returnUrl preservation | Verified & Accepted |
| AUTH-CLAIM-0022-0025 | Global vs. restricted firm access, firm/site fallback resolution, and active firm preference tracking | Verified & Accepted |
| AUTH-CLAIM-0026-0029 | RBAC namespaced bizRules, assignment caching, admin assignment restrictions, and module role sync hooks | Verified & Accepted |
| AUTH-CLAIM-0030-0033 | PIN validation rules, PIN regeneration threshold, PIN viewing step-up verification, and e-signature decryption | Verified & Accepted |
| AUTH-CLAIM-0034-0037 | Auth schema migration decoupling, credential state encapsulation, Yii DB RBAC table structure, and service account NULL institution handling | Verified & Accepted |
| AUTH-CLAIM-0038-0041 | Native LDAP admin bind search, direct LDAP user prefix bind & retries, SSO/SAML/OIDC claim provisioning, and multi-tenant LDAP JSON config | Verified & Accepted |
