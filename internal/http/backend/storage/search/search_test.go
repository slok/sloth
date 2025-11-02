package search_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
	"github.com/slok/sloth/internal/http/backend/storage/search"
	"github.com/slok/sloth/internal/http/backend/storage/storagemock"
)

func TestSearchRepositoryWrapperListServiceAndAlertsByServiceSearch(t *testing.T) {
	tests := map[string]struct {
		searchInput string
		mocks       func(*storagemock.ServiceGetter, *storagemock.SLOGetter)
		expRes      []storage.ServiceAndAlerts
		expErr      bool
	}{
		"Should return matching services when search input matches service IDs.": {
			searchInput: "mt",
			mocks: func(msg *storagemock.ServiceGetter, slg *storagemock.SLOGetter) {
				msg.On("ListAllServiceAndAlerts", mock.Anything).Once().Return([]storage.ServiceAndAlerts{
					{Service: model.Service{ID: "service-mt"}},
					{Service: model.Service{ID: "service-xy"}},
					{Service: model.Service{ID: "another-service-mt"}},
					{Service: model.Service{ID: "unrelated"}},
					{Service: model.Service{ID: "service-mt-2"}},
					{Service: model.Service{ID: "service-z"}},
				}, nil)
			},
			expRes: []storage.ServiceAndAlerts{
				{Service: model.Service{ID: "service-mt"}},
				{Service: model.Service{ID: "another-service-mt"}},
				{Service: model.Service{ID: "service-mt-2"}},
			},
		},

		"Shouldn't return services when search input doesn't matches service IDs.": {
			searchInput: "zzzzz",
			mocks: func(msg *storagemock.ServiceGetter, slg *storagemock.SLOGetter) {
				msg.On("ListAllServiceAndAlerts", mock.Anything).Once().Return([]storage.ServiceAndAlerts{
					{Service: model.Service{ID: "service-mt"}},
					{Service: model.Service{ID: "service-xy"}},
					{Service: model.Service{ID: "another-service-mt"}},
					{Service: model.Service{ID: "unrelated"}},
					{Service: model.Service{ID: "service-mt-2"}},
					{Service: model.Service{ID: "service-z"}},
				}, nil)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			msg := storagemock.NewServiceGetter(t)
			slg := storagemock.NewSLOGetter(t)
			test.mocks(msg, slg)

			repo, err := search.NewSearchRepositoryWrapper(msg, slg)
			require.NoError(err)

			res, err := repo.ListServiceAndAlertsByServiceSearch(t.Context(), test.searchInput)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRes, res)
			}
		})
	}
}

func TestSearchRepositoryWrapperListSLOInstantDetailsServiceBySLOSearch(t *testing.T) {
	tests := map[string]struct {
		service     string
		searchInput string
		mocks       func(*storagemock.ServiceGetter, *storagemock.SLOGetter)
		expRes      []storage.SLOInstantDetails
		expErr      bool
	}{
		"Should return matching SLOs when search input matches SLO IDs.": {
			service:     "my-service",
			searchInput: "mt",
			mocks: func(msg *storagemock.ServiceGetter, slg *storagemock.SLOGetter) {
				slg.On("ListSLOInstantDetailsService", mock.Anything, "my-service").Once().Return([]storage.SLOInstantDetails{
					{SLO: model.SLO{ID: "service-mt"}},
					{SLO: model.SLO{ID: "service-xy"}},
					{SLO: model.SLO{ID: "another-service-mt"}},
					{SLO: model.SLO{ID: "unrelated"}},
					{SLO: model.SLO{ID: "service-mt-2"}},
					{SLO: model.SLO{ID: "service-z"}},
				}, nil)
			},
			expRes: []storage.SLOInstantDetails{
				{SLO: model.SLO{ID: "service-mt"}},
				{SLO: model.SLO{ID: "another-service-mt"}},
				{SLO: model.SLO{ID: "service-mt-2"}},
			},
		},

		"Shouldn't return SLOs when search input doesn't matches SLO IDs.": {
			service:     "my-service",
			searchInput: "zzzzz",
			mocks: func(msg *storagemock.ServiceGetter, slg *storagemock.SLOGetter) {
				slg.On("ListSLOInstantDetailsService", mock.Anything, "my-service").Once().Return([]storage.SLOInstantDetails{
					{SLO: model.SLO{ID: "service-mt"}},
					{SLO: model.SLO{ID: "service-xy"}},
					{SLO: model.SLO{ID: "another-service-mt"}},
					{SLO: model.SLO{ID: "unrelated"}},
					{SLO: model.SLO{ID: "service-mt-2"}},
					{SLO: model.SLO{ID: "service-z"}},
				}, nil)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			msg := storagemock.NewServiceGetter(t)
			slg := storagemock.NewSLOGetter(t)
			test.mocks(msg, slg)

			repo, err := search.NewSearchRepositoryWrapper(msg, slg)
			require.NoError(err)

			res, err := repo.ListSLOInstantDetailsServiceBySLOSearch(t.Context(), test.service, test.searchInput)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRes, res)
			}
		})
	}

}

func TestSearchRepositoryWrapperListSLOInstantDetailsBySLOSearch(t *testing.T) {
	tests := map[string]struct {
		searchInput string
		mocks       func(*storagemock.ServiceGetter, *storagemock.SLOGetter)
		expRes      []storage.SLOInstantDetails
		expErr      bool
	}{
		"Should return matching SLOs when search input matches SLO IDs.": {
			searchInput: "mt",
			mocks: func(msg *storagemock.ServiceGetter, slg *storagemock.SLOGetter) {
				slg.On("ListSLOInstantDetails", mock.Anything).Once().Return([]storage.SLOInstantDetails{
					{SLO: model.SLO{ID: "service-mt"}},
					{SLO: model.SLO{ID: "service-xy"}},
					{SLO: model.SLO{ID: "another-service-mt"}},
					{SLO: model.SLO{ID: "unrelated"}},
					{SLO: model.SLO{ID: "service-mt-2"}},
					{SLO: model.SLO{ID: "service-z"}},
				}, nil)
			},
			expRes: []storage.SLOInstantDetails{
				{SLO: model.SLO{ID: "service-mt"}},
				{SLO: model.SLO{ID: "another-service-mt"}},
				{SLO: model.SLO{ID: "service-mt-2"}},
			},
		},

		"Shouldn't return SLOs when search input doesn't matches SLO IDs.": {
			searchInput: "zzzzz",
			mocks: func(msg *storagemock.ServiceGetter, slg *storagemock.SLOGetter) {
				slg.On("ListSLOInstantDetails", mock.Anything).Once().Return([]storage.SLOInstantDetails{
					{SLO: model.SLO{ID: "service-mt"}},
					{SLO: model.SLO{ID: "service-xy"}},
					{SLO: model.SLO{ID: "another-service-mt"}},
					{SLO: model.SLO{ID: "unrelated"}},
					{SLO: model.SLO{ID: "service-mt-2"}},
					{SLO: model.SLO{ID: "service-z"}},
				}, nil)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			msg := storagemock.NewServiceGetter(t)
			slg := storagemock.NewSLOGetter(t)
			test.mocks(msg, slg)

			repo, err := search.NewSearchRepositoryWrapper(msg, slg)
			require.NoError(err)

			res, err := repo.ListSLOInstantDetailsBySLOSearch(t.Context(), test.searchInput)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRes, res)
			}
		})
	}
}
