package domain_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/romshark/todostar/domain"

	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	s := domain.New()

	c := collectAll(t, s, domain.SearchFilters{})
	require.Len(t, c, 0)

	now := time.Now()

	id, err := s.Add(t.Context(), "New Todo", "some description", now, now.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(1), id)

	c = collectAll(t, s, domain.SearchFilters{})
	require.Equal(t, []*domain.Todo{
		{
			ID:          id,
			Title:       "New Todo",
			Description: "some description",
			Status:      domain.StatusOpen,
			Created:     now,
			Due:         now.Add(time.Hour),
		},
	}, c)

	err = s.Edit(t.Context(), id, func(t *domain.Todo) error {
		t.Status = domain.StatusDone
		return nil
	})
	require.NoError(t, err)

	c = collectAll(t, s, domain.SearchFilters{})
	require.Equal(t, []*domain.Todo{
		{
			ID:          id,
			Title:       "New Todo",
			Description: "some description",
			Status:      domain.StatusDone, // changed.
			Created:     now,
			Due:         now.Add(time.Hour),
		},
	}, c)

	err = s.Archive(t.Context(), id)
	require.NoError(t, err)

	c = collectAll(t, s, domain.SearchFilters{})
	require.Len(t, c, 0)

	err = s.Delete(t.Context(), id)
	require.NoError(t, err)

	c = collectAll(t, s, domain.SearchFilters{Archived: true})
	require.Len(t, c, 0)
}

func TestSearch(t *testing.T) {
	s := domain.New()
	now := time.Now()

	_, err := s.Add(t.Context(),
		"New Todo", "some description", now, now.Add(time.Hour))
	require.NoError(t, err)
	_, err = s.Add(t.Context(),
		"Won't match", "this will not match by text", now, now.Add(time.Hour))
	require.NoError(t, err)
	_, err = s.Add(t.Context(),
		"Another New Todo", "", now, now.Add(time.Hour))
	require.NoError(t, err)

	c := collectAll(t, s, domain.SearchFilters{
		TextMatch: "New Todo",
	})
	require.Len(t, c, 2)
	require.Equal(t, "New Todo", c[0].Title)
	require.Equal(t, "Another New Todo", c[1].Title)
}

func TestStoreChangeStatusErrNotExist(t *testing.T) {
	s := domain.New()
	err := s.Edit(t.Context(), 1, func(t *domain.Todo) error {
		t.Status = domain.StatusDone
		t.Due = t.Due.Add(2 * time.Hour)
		return nil
	})
	require.ErrorIs(t, err, domain.ErrNotExists)
}

func TestValidate(t *testing.T) {
	s := domain.New()

	{ // Title
		id, err := s.Add(t.Context(), "", "", time.Now(), time.Now().Add(time.Minute))
		var v domain.ErrorValidation
		require.ErrorAs(t, err, &v)
		require.Zero(t, id)
		require.Equal(t, domain.ErrorValidation{
			TitleEmpty: true,
		}, v)
		c := collectAll(t, s, domain.SearchFilters{})
		require.Len(t, c, 0)
	}

	{ // Title and description too long
		title := strings.Repeat("x", domain.TitleMaxLength+1)
		desc := strings.Repeat("x", domain.DescriptionMaxLength+1)
		id, err := s.Add(
			t.Context(), title, desc, time.Now(), time.Now().Add(time.Minute),
		)
		var v domain.ErrorValidation
		require.ErrorAs(t, err, &v)
		require.Zero(t, id)
		require.Equal(t, domain.ErrorValidation{
			TitleTooLong:       true,
			DescriptionTooLong: true,
		}, v)
		c := collectAll(t, s, domain.SearchFilters{})
		require.Len(t, c, 0)
	}
}

func collectAll(
	t *testing.T, s *domain.Store, filters domain.SearchFilters,
) []*domain.Todo {
	t.Helper()
	l, err := s.Search(context.Background(), filters)
	require.NoError(t, err)
	return l
}
