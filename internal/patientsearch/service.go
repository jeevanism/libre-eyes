package patientsearch

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/auth"
)

const defaultStatementTimeout = 3 * time.Second

type operationAuthorizer interface {
	AuthorizeOperation(context.Context, auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error)
}

// ServiceConfig controls bounded patient-search infrastructure policy.
type ServiceConfig struct {
	StatementTimeout time.Duration
}

// Service implements authorization, exact search, protected cursors, and disclosure audit.
type Service struct {
	pool       *pgxpool.Pool
	authorizer operationAuthorizer
	limiter    *principalLimiter
	cursors    *cursorStore
	config     ServiceConfig
	now        func() time.Time
}

// Authorization proves that this Service validated one endpoint operation.
type Authorization struct {
	principal auth.OperationPrincipal
	operation Operation
	metadata  auth.RequestMetadata
	service   *Service
}

// NewService constructs the patient-search application service.
func NewService(pool *pgxpool.Pool, authorizer operationAuthorizer, cfg ServiceConfig) (*Service, error) {
	if pool == nil || authorizer == nil {
		return nil, errors.New("patient search database and authorizer are required")
	}
	if cfg.StatementTimeout == 0 {
		cfg.StatementTimeout = defaultStatementTimeout
	}
	if cfg.StatementTimeout <= 0 {
		return nil, errors.New("patient search statement timeout must be positive")
	}
	return &Service{
		pool: pool, authorizer: authorizer, limiter: newPrincipalLimiter(),
		cursors: newCursorStore(), config: cfg, now: time.Now,
	}, nil
}

// Authorize enforces session, CSRF, current context, permission, and the combined principal rate limit.
func (s *Service) Authorize(
	ctx context.Context,
	token string,
	csrf string,
	operation Operation,
	metadata auth.RequestMetadata,
) (Authorization, error) {
	permission, deniedEvent, ok := operationPolicy(operation)
	if !ok {
		return Authorization{}, ErrInvalidRequest
	}
	principal, err := s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{
		Token: token, CSRFToken: csrf, Permission: permission,
		Capability:      "patient_search",
		DeniedEventType: deniedEvent, Metadata: metadata,
	})
	if err != nil {
		if errors.Is(err, auth.ErrUnauthenticated) || errors.Is(err, auth.ErrCSRF) || errors.Is(err, auth.ErrForbidden) {
			return Authorization{}, err
		}
		return Authorization{}, classifyInfrastructureError(err)
	}
	if allowed, retryAfter := s.limiter.allow(principal.UserID); !allowed {
		s.bestEffortDeniedAudit(ctx, principal, metadata, deniedEvent, "rate_limited")
		return Authorization{}, &RateLimitError{RetryAfter: retryAfter}
	}
	return Authorization{principal: principal, operation: operation, metadata: metadata, service: s}, nil
}

// Search executes an authorized exact patient search and durably audits before disclosure.
func (s *Service) Search(ctx context.Context, authorization Authorization, request SearchRequest) (SearchPage, error) {
	if !s.validAuthorization(authorization, OperationSearch) {
		return SearchPage{}, auth.ErrForbidden
	}
	limit := request.Limit
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit < 1 || limit > maximumPageSize {
		return SearchPage{}, ErrInvalidRequest
	}

	started := s.now().UTC()
	tx, repository, err := s.begin(ctx)
	if err != nil {
		return SearchPage{}, err
	}
	defer rollback(tx)

	criteria, digest, form, err := s.prepareCriteria(ctx, repository, authorization.principal, OperationSearch, request.Criteria)
	if err != nil {
		return SearchPage{}, err
	}
	cursorBinding := s.cursorBinding(authorization.principal, digest, limit)
	var boundary *PageBoundary
	if request.Cursor != "" {
		value, err := s.cursors.get(request.Cursor, cursorBinding)
		if err != nil {
			return SearchPage{}, err
		}
		boundary = &value
	}

	page, err := executeSearch(ctx, repository, authorization.principal, criteria, limit, boundary)
	if err != nil {
		return SearchPage{}, classifyInfrastructureError(err)
	}
	results, err := mapPatients(page.Items)
	if err != nil {
		return SearchPage{}, ErrUnavailable
	}
	outcome := "matches"
	if len(results) == 0 {
		outcome = "no_matches"
	}
	if err := appendExecutedAudit(ctx, tx, authorization, "patient_search.executed", outcome, form, len(results), s.now().UTC().Sub(started)); err != nil {
		return SearchPage{}, ErrUnavailable
	}
	if err := tx.Commit(ctx); err != nil {
		return SearchPage{}, classifyInfrastructureError(err)
	}

	result := SearchPage{Items: results, HasMore: page.HasMore}
	if page.HasMore {
		if page.NextBoundary == nil {
			return SearchPage{}, ErrUnavailable
		}
		cursorBinding.boundary = *page.NextBoundary
		token, err := s.cursors.put(cursorBinding)
		if err != nil {
			return SearchPage{}, ErrUnavailable
		}
		result.NextCursor = &token
	}
	return result, nil
}

// FindDuplicates executes an authorized, exact-only duplicate candidate check.
func (s *Service) FindDuplicates(ctx context.Context, authorization Authorization, request DuplicateRequest) (DuplicateResult, error) {
	if !s.validAuthorization(authorization, OperationDuplicateCheck) {
		return DuplicateResult{}, auth.ErrForbidden
	}
	started := s.now().UTC()
	tx, repository, err := s.begin(ctx)
	if err != nil {
		return DuplicateResult{}, err
	}
	defer rollback(tx)

	criteria, _, form, err := s.prepareCriteria(ctx, repository, authorization.principal, OperationDuplicateCheck, request.Criteria)
	if err != nil {
		return DuplicateResult{}, err
	}
	candidates, err := executeDuplicates(ctx, repository, authorization.principal, criteria)
	if err != nil {
		return DuplicateResult{}, classifyInfrastructureError(err)
	}
	mapped, err := mapDuplicateCandidates(candidates.Candidates)
	if err != nil {
		return DuplicateResult{}, ErrUnavailable
	}
	outcome := "candidates"
	if candidates.HardConflict {
		outcome = "hard_conflict"
	} else if len(mapped) == 0 {
		outcome = "no_candidates"
	}
	if err := appendExecutedAudit(ctx, tx, authorization, "patient_duplicate_check.executed", outcome, form, len(mapped), s.now().UTC().Sub(started)); err != nil {
		return DuplicateResult{}, ErrUnavailable
	}
	if err := tx.Commit(ctx); err != nil {
		return DuplicateResult{}, classifyInfrastructureError(err)
	}
	return DuplicateResult{HardConflict: candidates.HardConflict, Truncated: candidates.Truncated, Candidates: mapped}, nil
}

type preparedCriteria struct {
	kind             CriteriaKind
	identifierTypeID int64
	canonicalValue   string
	demographic      DemographicCriteria
	duplicate        DuplicateDemographicCriteria
}

func (s *Service) prepareCriteria(
	ctx context.Context,
	repository *Repository,
	principal auth.OperationPrincipal,
	operation Operation,
	criteria SearchCriteria,
) (preparedCriteria, [32]byte, string, error) {
	switch criteria.Kind {
	case CriteriaIdentifier:
		if criteria.Identifier == nil || criteria.Demographic != nil || criteria.Identifier.IdentifierTypeID < 1 {
			return preparedCriteria{}, [32]byte{}, "", ErrInvalidRequest
		}
		config, err := repository.LoadIdentifierType(ctx, principal.InstitutionID, principal.SiteID, criteria.Identifier.IdentifierTypeID)
		if err != nil {
			if errors.Is(err, ErrIdentifierTypeUnavailable) || errors.Is(err, ErrInvalidRepositoryRequest) {
				return preparedCriteria{}, [32]byte{}, "", ErrInvalidRequest
			}
			return preparedCriteria{}, [32]byte{}, "", classifyInfrastructureError(err)
		}
		normalizer, err := CompileIdentifierRule(IdentifierRule{
			Kind: config.NormalizationKind, ValidationPattern: config.ValidationPattern,
			MaximumCanonicalLength: config.MaximumCanonicalLength, ZeroPadWidth: config.ZeroPadWidth,
		})
		if err != nil {
			return preparedCriteria{}, [32]byte{}, "", ErrInvalidRequest
		}
		canonical, err := normalizer.Canonicalize(criteria.Identifier.Value)
		if err != nil {
			return preparedCriteria{}, [32]byte{}, "", ErrInvalidRequest
		}
		prepared := preparedCriteria{kind: CriteriaIdentifier, identifierTypeID: config.ID, canonicalValue: canonical}
		return prepared, digestIdentifier(config.ID, config.ConfigurationVersion, canonical), "identifier", nil
	case CriteriaDemographic:
		if criteria.Demographic == nil || criteria.Identifier != nil {
			return preparedCriteria{}, [32]byte{}, "", ErrInvalidRequest
		}
		family, err := NormalizeName(criteria.Demographic.FamilyName, 100)
		if err != nil {
			return preparedCriteria{}, [32]byte{}, "", ErrInvalidRequest
		}
		dateOfBirth, err := parseDateOfBirth(criteria.Demographic.DateOfBirth, s.now())
		if err != nil {
			return preparedCriteria{}, [32]byte{}, "", ErrInvalidRequest
		}
		if operation == OperationSearch && !validGender(criteria.Demographic.Gender) {
			return preparedCriteria{}, [32]byte{}, "", ErrInvalidRequest
		}
		var given *string
		if criteria.Demographic.GivenName != nil {
			value, err := NormalizeName(*criteria.Demographic.GivenName, 300)
			if err != nil {
				return preparedCriteria{}, [32]byte{}, "", ErrInvalidRequest
			}
			given = &value
		}
		prepared := preparedCriteria{
			kind:        CriteriaDemographic,
			demographic: DemographicCriteria{FamilyNameNormalized: family, GivenNameNormalized: given, DateOfBirth: dateOfBirth, Gender: criteria.Demographic.Gender},
		}
		if given != nil {
			prepared.duplicate = DuplicateDemographicCriteria{FamilyNameNormalized: family, GivenNameNormalized: *given, DateOfBirth: dateOfBirth}
		}
		return prepared, digestDemographic(family, given, dateOfBirth, criteria.Demographic.Gender), "demographic", nil
	default:
		return preparedCriteria{}, [32]byte{}, "", ErrInvalidRequest
	}
}

func parseDateOfBirth(value string, now time.Time) (time.Time, error) {
	date, err := time.Parse(dateLayout, value)
	if err != nil || date.Format(dateLayout) != value || isFutureDate(date, now) {
		return time.Time{}, ErrInvalidRequest
	}
	return date, nil
}

func isFutureDate(date, now time.Time) bool {
	location := now.Location()
	calendarDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	return calendarDate.After(today)
}

func (s *Service) begin(ctx context.Context) (pgx.Tx, *Repository, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, nil, classifyInfrastructureError(err)
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('statement_timeout', $1, true)`, s.config.StatementTimeout.String()); err != nil {
		_ = tx.Rollback(ctx)
		return nil, nil, classifyInfrastructureError(err)
	}
	repository, err := NewRepository(tx)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, nil, ErrUnavailable
	}
	return tx, repository, nil
}

func executeSearch(ctx context.Context, repository *Repository, principal auth.OperationPrincipal, criteria preparedCriteria, limit int, boundary *PageBoundary) (PatientPage, error) {
	if criteria.kind == CriteriaIdentifier {
		return repository.SearchByIdentifier(ctx, principal.InstitutionID, principal.SiteID, criteria.identifierTypeID, criteria.canonicalValue, limit, boundary)
	}
	return repository.SearchByDemographics(ctx, principal.InstitutionID, principal.SiteID, criteria.demographic, limit, boundary)
}

func executeDuplicates(ctx context.Context, repository *Repository, principal auth.OperationPrincipal, criteria preparedCriteria) (DuplicateCandidates, error) {
	if criteria.kind == CriteriaIdentifier {
		return repository.FindIdentifierDuplicates(ctx, principal.InstitutionID, principal.SiteID, criteria.identifierTypeID, criteria.canonicalValue)
	}
	if criteria.duplicate.GivenNameNormalized == "" {
		return DuplicateCandidates{}, ErrInvalidRequest
	}
	return repository.FindDemographicDuplicates(ctx, principal.InstitutionID, principal.SiteID, criteria.duplicate)
}

func appendExecutedAudit(ctx context.Context, tx pgx.Tx, authorization Authorization, eventType, operationOutcome, form string, count int, latency time.Duration) error {
	attributes, err := json.Marshal(map[string]any{
		"permission":        operationPermission(authorization.operation),
		"permissionOutcome": "allowed",
		"searchForm":        form,
		"resultCountBucket": resultCountBucket(count),
		"latencyMs":         max(latency.Milliseconds(), 0),
		"operationOutcome":  operationOutcome,
	})
	if err != nil {
		return fmt.Errorf("encode patient search audit attributes: %w", err)
	}
	principal := authorization.principal
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			event_type, actor_user_id, session_id, institution_id, site_id, firm_id,
			outcome, reason_code, correlation_id, source_ip_class, attributes
		) VALUES ($1, $2, $3, $4, $5, $6, 'success', $7, $8, $9, $10::jsonb)`,
		eventType, principal.UserID, principal.SessionID, principal.InstitutionID,
		principal.SiteID, principal.FirmID, operationOutcome,
		authorization.metadata.CorrelationID, authorization.metadata.SourceIPClass, attributes,
	); err != nil {
		return fmt.Errorf("append patient search audit: %w", err)
	}
	return nil
}

func (s *Service) bestEffortDeniedAudit(ctx context.Context, principal auth.OperationPrincipal, metadata auth.RequestMetadata, eventType, reason string) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return
	}
	defer rollback(tx)
	attributes, _ := json.Marshal(map[string]any{"permissionOutcome": "denied"})
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			event_type, actor_user_id, session_id, institution_id, site_id, firm_id,
			outcome, reason_code, correlation_id, source_ip_class, attributes
		) VALUES ($1, $2, $3, $4, $5, $6, 'denied', $7, $8, $9, $10::jsonb)`,
		eventType, principal.UserID, principal.SessionID, principal.InstitutionID,
		principal.SiteID, principal.FirmID, reason, metadata.CorrelationID,
		metadata.SourceIPClass, attributes,
	); err != nil {
		return
	}
	_ = tx.Commit(ctx)
}

func operationPolicy(operation Operation) (string, string, bool) {
	switch operation {
	case OperationSearch:
		return "patient.search", "patient_search.denied", true
	case OperationDuplicateCheck:
		return "patient.duplicate_check", "patient_duplicate_check.denied", true
	default:
		return "", "", false
	}
}

func operationPermission(operation Operation) string {
	permission, _, _ := operationPolicy(operation)
	return permission
}

func (s *Service) validAuthorization(authorization Authorization, operation Operation) bool {
	return authorization.service == s && authorization.operation == operation &&
		authorization.principal.UserID > 0 && authorization.principal.SessionID > 0 &&
		authorization.principal.InstitutionID > 0 && authorization.principal.SiteID > 0 &&
		authorization.principal.FirmID > 0
}

func (s *Service) cursorBinding(principal auth.OperationPrincipal, criteriaDigest [32]byte, limit int) cursorEntry {
	return cursorEntry{
		userID: principal.UserID, sessionID: principal.SessionID,
		institutionID: principal.InstitutionID, siteID: principal.SiteID,
		firmID: principal.FirmID, contextVersion: principal.ContextVersion,
		criteriaDigest: criteriaDigest, limit: limit,
		orderingVersion:      cursorOrderingVersion,
		normalizationVersion: NameNormalizationVersion,
	}
}

func digestIdentifier(typeID, version int64, canonical string) [32]byte {
	value := make([]byte, 16, 16+len(canonical))
	binary.BigEndian.PutUint64(value[:8], uint64(typeID))
	binary.BigEndian.PutUint64(value[8:16], uint64(version))
	value = append(value, canonical...)
	return sha256.Sum256(value)
}

func digestDemographic(family string, given *string, date time.Time, gender Gender) [32]byte {
	value := family + "\x00"
	if given != nil {
		value += *given
	}
	value += "\x00" + date.Format(dateLayout) + "\x00" + string(gender)
	return sha256.Sum256([]byte(value))
}

func mapPatients(records []PatientRecord) ([]PatientResult, error) {
	results := make([]PatientResult, len(records))
	for index, record := range records {
		mapped, err := mapPatient(record)
		if err != nil {
			return nil, err
		}
		results[index] = mapped
	}
	return results, nil
}

func mapDuplicateCandidates(candidates []DuplicateCandidate) ([]DuplicateResultCandidate, error) {
	results := make([]DuplicateResultCandidate, len(candidates))
	for index, candidate := range candidates {
		patient, err := mapPatient(candidate.Patient)
		if err != nil {
			return nil, err
		}
		results[index] = DuplicateResultCandidate{Reason: candidate.Reason, Patient: patient}
	}
	return results, nil
}

func formatIdentifier(identifier PrimaryIdentifierRecord) (string, error) {
	if identifier.TypeID < 1 || identifier.Label == "" || identifier.OriginalValue == "" {
		return "", ErrUnavailable
	}
	valueRunes := []rune(identifier.OriginalValue)
	formatted := ""
	consumed := 0
	if identifier.SpacingRule != nil {
		var builder strings.Builder
		for _, ruleRune := range []rune(*identifier.SpacingRule) {
			if consumed >= len(valueRunes) {
				break
			}
			if ruleRune == ' ' {
				builder.WriteRune(' ')
				continue
			}
			builder.WriteRune(valueRunes[consumed])
			consumed++
		}
		formatted = builder.String()
		if consumed < len(valueRunes) {
			if formatted != "" {
				formatted += " "
			}
			formatted += string(valueRunes[consumed:])
		}
	} else {
		formatted = identifier.OriginalValue
	}
	formatted = identifier.DisplayPrefix + formatted + identifier.DisplaySuffix
	if formatted == "" || utf8.RuneCountInString(formatted) > 255 {
		return "", ErrUnavailable
	}
	return formatted, nil
}

func resultCountBucket(count int) string {
	switch {
	case count == 0:
		return "0"
	case count == 1:
		return "1"
	case count <= 10:
		return "2-10"
	case count <= 25:
		return "11-25"
	default:
		return "26-100"
	}
}

func classifyInfrastructureError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ErrTimeout
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "57014" {
		return ErrTimeout
	}
	if errors.Is(err, ErrInvalidRepositoryRequest) {
		return ErrInvalidRequest
	}
	return ErrUnavailable
}

func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}
