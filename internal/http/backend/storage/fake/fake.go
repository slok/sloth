package storage

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
	commonerrors "github.com/slok/sloth/pkg/common/errors"
)

type FakeRepository struct {
	services         []model.Service
	slos             []model.SLO
	sloBudgetDetails []model.SLOBudgetDetails
	sloAlerts        []model.SLOAlerts
}

var (
	days30 = 30 * 24 * time.Hour
)

func NewFakeRepository() *FakeRepository {
	r := FakeRepository{}
	r.genFakeData()

	return &r
}

func (f *FakeRepository) genFakeData() {
	groupedLabelFakeResources := []string{"", "api", "auth", "billing", "checkout", "notification", "order", "payment", "product", "recommendation", "search", "user", "warehouse"}

	f.services = []model.Service{
		{ID: "api-gateway"},
		{ID: "auth-service"},
		{ID: "billing-service"},
		{ID: "checkout-service"},
		{ID: "notification-service"},
		{ID: "order-service"},
		{ID: "payment-service"},
		{ID: "product-catalog"},
		{ID: "recommendation-engine"},
		{ID: "search-service"},
		{ID: "user-service"},
		{ID: "warehouse-service"},
		{ID: "payment-service"},
		{ID: "reporting-service"},
		{ID: "analytics-service"},
		{ID: "shipping-service"},
		{ID: "inventory-service"},
		{ID: "customer-service"},
		{ID: "review-service"},
		{ID: "loyalty-service"},
		{ID: "discount-service"},
		{ID: "fraud-detection-service"},
		{ID: "email-service"},
		{ID: "sms-service"},
		{ID: "push-notification-service"},
		{ID: "content-management-service"},
		{ID: "media-service"},
		{ID: "search-indexer-service"},
		{ID: "data-warehouse-service"},
		{ID: "etl-service"},
		{ID: "ad-serving-service"},
		{ID: "data-processing-service"},
		{ID: "real-time-analytics-service"},
		{ID: "session-management-service"},
		{ID: "api-rate-limiting-service"},
		{ID: "load-balancing-service"},
		{ID: "caching-service"},
		{ID: "logging-service"},
		{ID: "monitoring-service"},
		{ID: "alerting-service"},
		{ID: "backup-service"},
		{ID: "disaster-recovery-service"},
		{ID: "configuration-service"},
		{ID: "feature-flag-service"},
		{ID: "ab-testing-service"},
		{ID: "user-profile-service"},
		{ID: "session-storage-service"},
		{ID: "oauth-service"},
		{ID: "saml-service"},
		{ID: "openid-connect-service"},
		{ID: "two-factor-authentication-service"},
		{ID: "password-reset-service"},
		{ID: "account-management-service"},
		{ID: "billing-integration-service"},
		{ID: "tax-calculation-service"},
		{ID: "shipping-integration-service"},
		{ID: "third-party-api-integration-service"},
		{ID: "webhook-service"},
		{ID: "chat-service"},
		{ID: "video-conferencing-service"},
		{ID: "file-storage-service"},
		{ID: "image-processing-service"},
		{ID: "pdf-generation-service"},
		{ID: "search-optimization-service"},
		{ID: "performance-monitoring-service"},
		{ID: "user-behavior-tracking-service"},
		{ID: "heatmap-service"},
		{ID: "session-replay-service"},
		{ID: "conversion-tracking-service"},
		{ID: "funnel-analysis-service"},
		{ID: "customer-segmentation-service"},
		{ID: "a-b-testing-analytics-service"},
		{ID: "recommendation-algorithm-service"},
		{ID: "personalization-service"},
		{ID: "search-personalization-service"},
		{ID: "dynamic-content-service"},
		{ID: "real-time-bidding-service"},
		{ID: "ad-targeting-service"},
		{ID: "ad-frequency-capping-service"},
		{ID: "ad-performance-tracking-service"},
		{ID: "campaign-management-service"},
		{ID: "budget-optimization-service"},
		{ID: "bid-management-service"},
		{ID: "creative-management-service"},
		{ID: "audience-insights-service"},
		{ID: "marketplace-integration-service"},
		{ID: "affiliate-marketing-service"},
		{ID: "influencer-marketing-service"},
		{ID: "content-distribution-service"},
		{ID: "cdn-integration-service"},
		{ID: "video-streaming-service"},
		{ID: "live-broadcasting-service"},
		{ID: "virtual-reality-service"},
		{ID: "augmented-reality-service"},
		{ID: "blockchain-integration-service"},
		{ID: "cryptocurrency-payment-service"},
		{ID: "nft-management-service"},
		{ID: "smart-contract-service"},
		{ID: "iot-integration-service"},
		{ID: "edge-computing-service"},
		{ID: "fog-computing-service"},
		{ID: "quantum-computing-service"},
		{ID: "ai-integration-service"},
		{ID: "machine-learning-service"},
		{ID: "deep-learning-service"},
		{ID: "natural-language-processing-service"},
		{ID: "computer-vision-service"},
		{ID: "speech-recognition-service"},
		{ID: "chatbot-service"},
		{ID: "virtual-assistant-service"},
		{ID: "robotic-process-automation-service"},
		{ID: "process-mining-service"},
		{ID: "business-intelligence-service"},
		{ID: "data-visualization-service"},
		{ID: "predictive-analytics-service"},
		{ID: "prescriptive-analytics-service"},
		{ID: "data-governance-service"},
		{ID: "data-quality-service"},
		{ID: "master-data-management-service"},
		{ID: "metadata-management-service"},
		{ID: "data-lineage-service"},
		{ID: "compliance-management-service"},
		{ID: "risk-management-service"},
		{ID: "fraud-prevention-service"},
		{ID: "cybersecurity-service"},
		{ID: "identity-and-access-management-service"},
		{ID: "security-information-and-event-management-service"},
		{ID: "vulnerability-management-service"},
		{ID: "penetration-testing-service"},
		{ID: "incident-response-service"},
		{ID: "disaster-recovery-planning-service"},
		{ID: "business-continuity-planning-service"},
	}

	slos := []model.SLO{}
	objectiveValues := []float64{
		99, 95, 97, 98, 96, 99.9, 92, 99.5, 99.99, 80, 85, 99,
		99.97, 94.5, 96.7, 98.3, 97.5, 99.2, 93.8, 99.8,
		90, 88.5, 91.2, 94.8, 97.1, 95.6, 99.3, 96.4, 98.7, 92.9,
		99.6, 97.9, 94.2, 95.3, 98.1, 93.5, 99.4, 96.8, 97.6, 91.7,
		89.9, 92.5, 94.1, 95.8, 98.9, 99.1, 97.3, 96.2, 93.7, 99.95, 98.5, 99.999,
	}
	for i, svc := range f.services {
		sloQuantity := i * 456 % len(objectiveValues)
		if sloQuantity == 0 {
			sloQuantity = 2
		}
		for j, obj := range objectiveValues[:sloQuantity] {
			sloName := fmt.Sprintf("slo-%04d-%04d", i, j)
			// Single SLO or grouped SLO.
			if (j+i)%3 != 0 {
				slos = append(slos, model.SLO{
					ID:             svc.ID + "-" + sloName,
					SlothID:        svc.ID + "-" + sloName,
					Name:           sloName,
					ServiceID:      svc.ID,
					Objective:      obj,
					PeriodDuration: days30,
				})
			} else {
				for k := 0; k < j; k++ {
					resource := groupedLabelFakeResources[(j+i+k)%len(groupedLabelFakeResources)]
					groupedLabels := map[string]string{
						"handler":  fmt.Sprintf("/api/v1/group-%d%d%d", i, j, k),
						"resource": resource,
					}
					slothID := svc.ID + "-" + sloName
					slos = append(slos, model.SLO{
						ID:             model.SLOGroupLabelsIDMarshal(slothID, groupedLabels),
						SlothID:        slothID,
						Name:           sloName,
						ServiceID:      svc.ID,
						Objective:      obj,
						PeriodDuration: days30,
						GroupLabels:    groupedLabels,
						IsGrouped:      true,
					})
				}
			}
		}
	}

	f.slos = slos

	allSLOBudgets := []model.SLOBudgetDetails{}
	budgetCatalog := []model.SLOBudgetDetails{
		{BurningBudgetPercent: 93.5, BurnedBudgetWindowPercent: 40},
		{BurningBudgetPercent: 71, BurnedBudgetWindowPercent: 30.2},
		{BurningBudgetPercent: 90, BurnedBudgetWindowPercent: 71},
		{BurningBudgetPercent: 292, BurnedBudgetWindowPercent: 150},
		{BurningBudgetPercent: 191, BurnedBudgetWindowPercent: 101.2},
		{BurningBudgetPercent: 622, BurnedBudgetWindowPercent: 225.1},
		{BurningBudgetPercent: 33, BurnedBudgetWindowPercent: 20.9},
		{BurningBudgetPercent: 32, BurnedBudgetWindowPercent: 25.1},
		{BurningBudgetPercent: 174, BurnedBudgetWindowPercent: 135.8},
		{BurningBudgetPercent: 89, BurnedBudgetWindowPercent: 42},
		{BurningBudgetPercent: 150, BurnedBudgetWindowPercent: 80.5},
		{BurningBudgetPercent: 87, BurnedBudgetWindowPercent: 33.3},
		{BurningBudgetPercent: 86, BurnedBudgetWindowPercent: 25.6},
		{BurningBudgetPercent: 88, BurnedBudgetWindowPercent: 57.9},
		{BurningBudgetPercent: 187, BurnedBudgetWindowPercent: 130},
		{BurningBudgetPercent: 85, BurnedBudgetWindowPercent: 12.89},
		{BurningBudgetPercent: 3, BurnedBudgetWindowPercent: 1.2},
		{BurningBudgetPercent: 881, BurnedBudgetWindowPercent: 0},
		{BurningBudgetPercent: 79, BurnedBudgetWindowPercent: 12.3},
		{BurningBudgetPercent: 77, BurnedBudgetWindowPercent: 0.5},
		{BurningBudgetPercent: 76, BurnedBudgetWindowPercent: 5.6},
		{BurningBudgetPercent: 75, BurnedBudgetWindowPercent: 110.7},
		{BurningBudgetPercent: 174, BurnedBudgetWindowPercent: 135.8},
		{BurningBudgetPercent: 33, BurnedBudgetWindowPercent: 20.9},
		{BurningBudgetPercent: 32, BurnedBudgetWindowPercent: 25.1},
		{BurningBudgetPercent: 622, BurnedBudgetWindowPercent: 225.1},
		{BurningBudgetPercent: 21, BurnedBudgetWindowPercent: 130.2},
		{BurningBudgetPercent: 15, BurnedBudgetWindowPercent: 10.5},
		{BurningBudgetPercent: 14, BurnedBudgetWindowPercent: 5.6},
	}
	for i, slo := range f.slos {
		b := budgetCatalog[i%len(budgetCatalog)]
		allSLOBudgets = append(allSLOBudgets, model.SLOBudgetDetails{
			SLOID:                     slo.ID,
			BurningBudgetPercent:      b.BurningBudgetPercent,
			BurnedBudgetWindowPercent: b.BurnedBudgetWindowPercent,
		})
	}
	f.sloBudgetDetails = allSLOBudgets

	f.sloAlerts = []model.SLOAlerts{
		{SLOID: f.slos[0].ID, FiringPage: &model.Alert{Name: "api-gateway-page"}},
		{SLOID: f.slos[90].ID, FiringWarning: &model.Alert{Name: "auth-service-warning"}},
		{SLOID: f.slos[5].ID},
		{SLOID: f.slos[36].ID, FiringPage: &model.Alert{Name: "checkout-service-page"}, FiringWarning: &model.Alert{Name: "checkout-service-warning"}},
		{SLOID: f.slos[1].ID, FiringPage: &model.Alert{Name: "order-service-page"}},
		{SLOID: f.slos[6].ID},
		{SLOID: f.slos[121].ID, FiringWarning: &model.Alert{Name: "payment-service-warning"}},
		{SLOID: f.slos[10].ID, FiringWarning: &model.Alert{Name: "product-catalog-warning"}},
		{SLOID: f.slos[67].ID, FiringPage: &model.Alert{Name: "recommendation-engine-page"}, FiringWarning: &model.Alert{Name: "recommendation-engine-warning"}},
		{SLOID: f.slos[35].ID, FiringWarning: &model.Alert{Name: "search-service-warning"}},
		{SLOID: f.slos[83].ID, FiringPage: &model.Alert{Name: "user-service-page"}},
		{SLOID: f.slos[102].ID, FiringWarning: &model.Alert{Name: "warehouse-service-warning"}},
	}
}

func (f FakeRepository) ListAllServiceAndAlerts(ctx context.Context) ([]storage.ServiceAndAlerts, error) {
	data := make([]storage.ServiceAndAlerts, 0, len(f.services))
	indexedSLOBudgets := make(map[string]model.SLOBudgetDetails)
	for _, bd := range f.sloBudgetDetails {
		indexedSLOBudgets[bd.SLOID] = bd
	}

	for _, svc := range f.services {
		svcAlerts := make([]model.SLOAlerts, 0)
		stats := model.ServiceStats{ServiceID: svc.ID}
		for _, slo := range f.slos {
			if slo.ServiceID != svc.ID {
				continue
			}

			budgetDetails, ok := indexedSLOBudgets[slo.ID]
			if !ok {
				continue
			}
			if budgetDetails.BurningBudgetPercent > 100 {
				stats.SLOsCurrentlyBurningOverBudget++
			}
			if budgetDetails.BurnedBudgetWindowPercent > 100 {
				stats.SLOsAlreadyConsumedBudgetOnPeriod++
			}

			stats.TotalSLOs++

			for _, sloAlert := range f.sloAlerts {
				if sloAlert.SLOID != slo.ID {
					continue
				}
				svcAlerts = append(svcAlerts, sloAlert)
			}
		}
		data = append(data, storage.ServiceAndAlerts{
			Service:      svc,
			ServiceStats: stats,
			Alerts:       svcAlerts,
		})
	}

	return data, nil
}

func (f FakeRepository) ListServiceAndAlertsByServiceSearch(ctx context.Context, serviceSearchInput string) ([]storage.ServiceAndAlerts, error) {
	return nil, fmt.Errorf("search not supported on storage")
}

func (f FakeRepository) ListSLOInstantDetailsService(ctx context.Context, serviceID string) ([]storage.SLOInstantDetails, error) {
	data := make([]storage.SLOInstantDetails, 0, len(f.slos))
	all, err := f.ListSLOInstantDetails(ctx)
	if err != nil {
		return nil, err
	}
	for _, sloDetail := range all {
		if sloDetail.SLO.ServiceID != serviceID {
			continue
		}
		data = append(data, sloDetail)
	}
	return data, nil
}

func (f FakeRepository) ListSLOInstantDetailsServiceBySLOSearch(ctx context.Context, serviceID, sloSearchInput string) ([]storage.SLOInstantDetails, error) {
	return nil, fmt.Errorf("search not supported on storage")
}

func (f FakeRepository) ListSLOInstantDetails(ctx context.Context) ([]storage.SLOInstantDetails, error) {
	data := make([]storage.SLOInstantDetails, 0, len(f.slos))
	for _, slo := range f.slos {
		var budgetDetails model.SLOBudgetDetails
		for _, bd := range f.sloBudgetDetails {
			if bd.SLOID == slo.ID {
				budgetDetails = bd
				break
			}
		}
		var alerts model.SLOAlerts
		for _, alert := range f.sloAlerts {
			if alert.SLOID == slo.ID {
				alerts = alert
				break
			}
		}
		data = append(data, storage.SLOInstantDetails{
			SLO:           slo,
			BudgetDetails: budgetDetails,
			Alerts:        alerts,
		})
	}
	return data, nil
}

func (f FakeRepository) ListSLOInstantDetailsBySLOSearch(ctx context.Context, sloSearchInput string) ([]storage.SLOInstantDetails, error) {
	return nil, fmt.Errorf("search not supported on storage")
}

func (f FakeRepository) GetSLOInstantDetails(ctx context.Context, sloID string) (*storage.SLOInstantDetails, error) {
	for _, slo := range f.slos {
		if slo.ID != sloID {
			continue
		}
		var budgetDetails model.SLOBudgetDetails
		for _, bd := range f.sloBudgetDetails {
			if bd.SLOID == slo.ID {
				budgetDetails = bd
				break
			}
		}
		var alerts model.SLOAlerts
		for _, alert := range f.sloAlerts {
			if alert.SLOID == slo.ID {
				alerts = alert
				break
			}
		}
		return &storage.SLOInstantDetails{
			SLO:           slo,
			BudgetDetails: budgetDetails,
			Alerts:        alerts,
		}, nil
	}
	return nil, commonerrors.ErrNotFound
}

func (f FakeRepository) GetSLIAvailabilityInRange(ctx context.Context, sloID string, from, to time.Time, step time.Duration) ([]model.DataPoint, error) {
	slo, err := f.GetSLOInstantDetails(ctx, sloID)
	if err != nil {
		return nil, err
	}

	// Fake data.
	factor := rand.Float64() + rand.Float64() + rand.Float64()
	dataPoints := []model.DataPoint{}
	for ts := from; ts.Before(to); ts = ts.Add(step) {
		perfectBurn := 100 - slo.SLO.Objective
		burned := perfectBurn * rand.Float64() * factor
		dataPoints = append(dataPoints, model.DataPoint{
			Value: 100 - burned,
			TS:    ts,
		})
	}

	return dataPoints, nil
}

func (f FakeRepository) GetSLIAvailabilityInRangeAutoStep(ctx context.Context, sloID string, from, to time.Time) ([]model.DataPoint, error) {
	autoStep := to.Sub(from) / 120
	return f.GetSLIAvailabilityInRange(ctx, sloID, from, to, autoStep)
}
