package service

import (
	"errors"
	"testing"

	"github.com/ireuven89/routewise/internal/models"
)

// mockJobRepoForDashboard is a minimal stand-in for *repository.JobRepository
// used to exercise JobSvc.GetDashboardStats. JobSvc.repo is a concrete
// *repository.JobRepository (not an interface), so - following the same
// convention used by testProviderService/testWorkerService elsewhere in this
// package - we mirror the thin wrapper logic against a function-field mock
// rather than the real DB-backed repository.
type mockJobRepoForDashboard struct {
	GetRevenueStatsFunc func(organizationID uint) (*models.RevenueStats, error)
}

// testJobDashboardService mirrors JobSvc.GetDashboardStats exactly.
type testJobDashboardService struct {
	repo *mockJobRepoForDashboard
}

func (s *testJobDashboardService) GetDashboardStats(organizationID uint) (*models.DashboardStats, error) {
	revenue, err := s.repo.GetRevenueStatsFunc(organizationID)
	if err != nil {
		return nil, err
	}
	return &models.DashboardStats{Revenue: revenue}, nil
}

func TestJobService_GetDashboardStats_Success(t *testing.T) {
	expectedRevenue := &models.RevenueStats{
		Total:       10000.0,
		ThisMonth:   2200.0,
		ThisWeek:    500.0,
		AvgJobValue: 333.33,
		RevenueByMonth: []models.MonthlyRevenue{
			{Month: "2026-06", Revenue: 1000.0},
			{Month: "2026-07", Revenue: 2200.0},
		},
	}

	var capturedOrgID uint
	svc := &testJobDashboardService{repo: &mockJobRepoForDashboard{
		GetRevenueStatsFunc: func(organizationID uint) (*models.RevenueStats, error) {
			capturedOrgID = organizationID
			return expectedRevenue, nil
		},
	}}

	stats, err := svc.GetDashboardStats(42)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedOrgID != 42 {
		t.Errorf("expected organizationID 42 to be passed to repo, got %d", capturedOrgID)
	}
	if stats == nil {
		t.Fatal("expected non-nil DashboardStats")
	}
	if stats.Revenue != expectedRevenue {
		t.Errorf("expected Revenue to be the value returned by the repo, got %+v", stats.Revenue)
	}
	if stats.Revenue.Total != 10000.0 {
		t.Errorf("expected Total 10000.0, got %v", stats.Revenue.Total)
	}
	if len(stats.Revenue.RevenueByMonth) != 2 {
		t.Errorf("expected 2 months of revenue data, got %d", len(stats.Revenue.RevenueByMonth))
	}
}

func TestJobService_GetDashboardStats_RepoErrorPropagates(t *testing.T) {
	sentinel := errors.New("db unavailable")

	svc := &testJobDashboardService{repo: &mockJobRepoForDashboard{
		GetRevenueStatsFunc: func(organizationID uint) (*models.RevenueStats, error) {
			return nil, sentinel
		},
	}}

	stats, err := svc.GetDashboardStats(1)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected error to be/wrap %v, got %v", sentinel, err)
	}
	if stats != nil {
		t.Errorf("expected nil stats on repo error, got %+v", stats)
	}
}
