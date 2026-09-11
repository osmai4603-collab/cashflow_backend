package recruitmentstorage

import (
	"context"
	"fmt"

	"cashflow_backend/internal/domain/recruitment"
	platformerrors "cashflow_backend/internal/platform/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct{ pool *pgxpool.Pool }

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo { return &PostgresRepo{pool: pool} }
func (repo *PostgresRepo) CreateStage(ctx context.Context, stage *recruitment.RecruitmentStage) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO recruitment_stages (name, sequence, folded, company_id) VALUES ($1,$2,$3,$4) RETURNING id`, stage.Name, stage.Sequence, stage.Folded, stage.CompanyID).Scan(&stage.ID)
}
func (repo *PostgresRepo) ListStages(ctx context.Context, companyID int64) ([]recruitment.RecruitmentStage, error) {
	rows, err := repo.pool.Query(ctx, `SELECT id, name, sequence, folded, company_id FROM recruitment_stages WHERE company_id=$1 ORDER BY sequence, id`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]recruitment.RecruitmentStage, 0)
	for rows.Next() {
		var stage recruitment.RecruitmentStage
		if err := rows.Scan(&stage.ID, &stage.Name, &stage.Sequence, &stage.Folded, &stage.CompanyID); err != nil {
			return nil, err
		}
		result = append(result, stage)
	}
	return result, rows.Err()
}
func (repo *PostgresRepo) CreateApplicant(ctx context.Context, applicant *recruitment.Applicant) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO recruitment_applicants (partner_name, email, phone, job_id, department_id, stage_id, recruiter_user_id, priority, salary_expected, salary_proposed, availability, refusal_reason, resume_url, employee_id, company_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id, created_at, updated_at`, applicant.PartnerName, applicant.Email, applicant.Phone, applicant.JobID, applicant.DepartmentID, applicant.StageID, applicant.RecruiterUserID, applicant.Priority, applicant.SalaryExpected, applicant.SalaryProposed, applicant.Availability, applicant.RefusalReason, applicant.ResumeURL, applicant.EmployeeID, applicant.CompanyID).Scan(&applicant.ID, &applicant.CreatedAt, &applicant.UpdatedAt)
}
func (repo *PostgresRepo) GetApplicant(ctx context.Context, id int64) (*recruitment.Applicant, error) {
	applicant := &recruitment.Applicant{ID: id}
	err := repo.pool.QueryRow(ctx, `SELECT partner_name, email, phone, job_id, department_id, stage_id, recruiter_user_id, priority, salary_expected, salary_proposed, availability, refusal_reason, resume_url, employee_id, company_id, created_at, updated_at FROM recruitment_applicants WHERE id=$1`, id).Scan(&applicant.PartnerName, &applicant.Email, &applicant.Phone, &applicant.JobID, &applicant.DepartmentID, &applicant.StageID, &applicant.RecruiterUserID, &applicant.Priority, &applicant.SalaryExpected, &applicant.SalaryProposed, &applicant.Availability, &applicant.RefusalReason, &applicant.ResumeURL, &applicant.EmployeeID, &applicant.CompanyID, &applicant.CreatedAt, &applicant.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound(fmt.Sprintf("applicant %d not found", id))
	}
	return applicant, err
}
func (repo *PostgresRepo) UpdateApplicant(ctx context.Context, applicant *recruitment.Applicant) error {
	result, err := repo.pool.Exec(ctx, `UPDATE recruitment_applicants SET stage_id=$1, refusal_reason=$2, salary_proposed=$3, employee_id=$4, updated_at=NOW() WHERE id=$5`, applicant.StageID, applicant.RefusalReason, applicant.SalaryProposed, applicant.EmployeeID, applicant.ID)
	if err == nil && result.RowsAffected() == 0 {
		return platformerrors.NotFound("applicant not found")
	}
	return err
}
func (repo *PostgresRepo) ListApplicants(ctx context.Context, companyID int64) ([]recruitment.Applicant, error) {
	rows, err := repo.pool.Query(ctx, `SELECT id, partner_name, email, phone, job_id, department_id, stage_id, recruiter_user_id, priority, salary_expected, salary_proposed, availability, refusal_reason, resume_url, employee_id, company_id, created_at, updated_at FROM recruitment_applicants WHERE company_id=$1 ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]recruitment.Applicant, 0)
	for rows.Next() {
		var applicant recruitment.Applicant
		if err := rows.Scan(&applicant.ID, &applicant.PartnerName, &applicant.Email, &applicant.Phone, &applicant.JobID, &applicant.DepartmentID, &applicant.StageID, &applicant.RecruiterUserID, &applicant.Priority, &applicant.SalaryExpected, &applicant.SalaryProposed, &applicant.Availability, &applicant.RefusalReason, &applicant.ResumeURL, &applicant.EmployeeID, &applicant.CompanyID, &applicant.CreatedAt, &applicant.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, applicant)
	}
	return result, rows.Err()
}
func (repo *PostgresRepo) CreateInterview(ctx context.Context, interview *recruitment.ApplicantInterview) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO recruitment_interviews (applicant_id, interviewer_id, event_id, interview_date, score, feedback, recommendation) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, interview.ApplicantID, interview.InterviewerID, interview.EventID, interview.InterviewDate, interview.Score, interview.Feedback, interview.Recommendation).Scan(&interview.ID)
}
