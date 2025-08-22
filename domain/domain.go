package domain

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blevesearch/bleve/v2"
	blevequery "github.com/blevesearch/bleve/v2/search/query"
)

func New() *Store {
	return &Store{
		indexByID:   make(map[int64]*Todo),
		searchIndex: mustMakeBleveIndex(),
	}
}

func mustMakeBleveIndex() bleve.Index {
	doc := bleve.NewDocumentMapping()

	title := bleve.NewTextFieldMapping()
	title.Store = false
	title.Analyzer = "en"
	doc.AddFieldMappingsAt("Title", title)

	desc := bleve.NewTextFieldMapping()
	desc.Store = false
	desc.Analyzer = "en"
	doc.AddFieldMappingsAt("Description", desc)

	arch := bleve.NewBooleanFieldMapping()
	arch.Store = false
	doc.AddFieldMappingsAt("Archived", arch)

	m := bleve.NewIndexMapping()
	m.DefaultAnalyzer = "en"
	m.DefaultMapping = doc

	idx, err := bleve.NewMemOnly(m)
	if err != nil {
		panic(err)
	}
	return idx
}

type Status int8

const (
	_ Status = iota
	StatusOpen
	StatusDone
)

type Todo struct {
	ID          int64
	Title       string
	Description string
	Status      Status
	Archived    bool
	Created     time.Time
	Due         time.Time
}

type Store struct {
	lock        sync.Mutex
	todos       []*Todo
	indexByID   map[int64]*Todo
	searchIndex bleve.Index
}

var idCounter atomic.Int64

const (
	TitleMaxLength       = 1024      // 1 KiB
	DescriptionMaxLength = 16 * 1024 // 16 KiB
)

type ErrorValidation struct {
	TitleEmpty         bool
	TitleTooLong       bool
	DescriptionTooLong bool
}

func Validate(title, description string) ErrorValidation {
	return ErrorValidation{
		TitleEmpty:         title == "",
		TitleTooLong:       len(title) > TitleMaxLength,
		DescriptionTooLong: len(description) > DescriptionMaxLength,
	}
}

func (v ErrorValidation) IsErr() bool {
	return v.TitleEmpty ||
		v.TitleTooLong ||
		v.DescriptionTooLong
}

func (v ErrorValidation) Error() string { return "invalid" }

func (s *Store) Add(
	_ context.Context, title, description string, now, due time.Time,
) (id int64, err error) {
	if err := Validate(title, description); err.IsErr() {
		return 0, err
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	newID := idCounter.Add(1)
	t := &Todo{
		ID:          newID,
		Title:       title,
		Description: description,
		Status:      StatusOpen,
		Created:     now,
		Due:         due,
	}

	if err := s.searchIndex.Index(strconv.FormatInt(newID, 10), map[string]any{
		"Title":       t.Title,
		"Description": t.Description,
		"Archived":    t.Archived,
	}); err != nil {
		// Roll back in case of index failure.
		return 0, err
	}

	s.todos = append(s.todos, t)
	s.indexByID[newID] = t
	return newID, nil
}

func (s *Store) findByID(id int64) (*Todo, error) {
	t, ok := s.indexByID[id]
	if !ok {
		return nil, ErrNotExists
	}
	return t, nil
}

type SearchFilters struct {
	Archived  bool
	TextMatch string
}

func (s *Store) Search(_ context.Context, filters SearchFilters) (res []*Todo, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if strings.TrimSpace(filters.TextMatch) == "" {
		// Fast search with simple filters.
		for _, t := range s.todos {
			if !filters.Archived && t.Archived || filters.Archived && !t.Archived {
				continue
			}
			res = append(res, t)
		}
		return res, nil
	}

	// Slow search by text match.
	q := buildBleveQuery(filters)
	req := bleve.NewSearchRequest(q)
	req.Size = len(s.todos)
	idxRes, err := s.searchIndex.Search(req)
	if err != nil {
		return nil, fmt.Errorf("searching index: %w", err)
	}

	byID := make(map[int64]*Todo, len(s.todos))
	for _, t := range s.todos {
		byID[t.ID] = t
	}

	for _, h := range idxRes.Hits {
		id64, err := strconv.ParseInt(h.ID, 10, 64)
		if err != nil {
			continue
		}
		t := byID[id64]
		if t == nil {
			continue
		}
		if !filters.Archived && t.Archived || filters.Archived && !t.Archived {
			continue
		}
		res = append(res, t)
	}
	return res, nil
}

func buildBleveQuery(f SearchFilters) blevequery.Query {
	terms := strings.Fields(strings.TrimSpace(f.TextMatch))

	var archQ blevequery.Query
	if !f.Archived {
		q := blevequery.NewBoolFieldQuery(false)
		q.SetField("Archived")
		archQ = q
	}

	if len(terms) == 0 {
		if archQ != nil {
			return bleve.NewConjunctionQuery(archQ)
		}
		return bleve.NewMatchAllQuery()
	}

	// Strategy 1: Try exact phrase match first (highest priority)
	var exactQueries []blevequery.Query
	fullText := strings.Join(terms, " ")

	titlePhraseQuery := bleve.NewMatchPhraseQuery(fullText)
	titlePhraseQuery.SetField("Title")
	titlePhraseQuery.SetBoost(10.0) // High boost for exact title matches
	exactQueries = append(exactQueries, titlePhraseQuery)

	descPhraseQuery := bleve.NewMatchPhraseQuery(fullText)
	descPhraseQuery.SetField("Description")
	descPhraseQuery.SetBoost(2.0) // Lower boost for description matches
	exactQueries = append(exactQueries, descPhraseQuery)

	// Strategy 2: Individual term matches (for partial matches)
	var termQueries []blevequery.Query
	for _, term := range terms {
		titleMatch := bleve.NewMatchQuery(term)
		titleMatch.SetField("Title")
		titleMatch.SetBoost(3.0) // Boost title matches

		descMatch := bleve.NewMatchQuery(term)
		descMatch.SetField("Description")
		descMatch.SetBoost(1.0) // Normal boost for description

		termQueries = append(termQueries, titleMatch, descMatch)
	}

	// Strategy 3: Fuzzy matching for typos
	var fuzzyQueries []blevequery.Query
	for _, term := range terms {
		if len(term) > 3 { // Only fuzzy match longer terms
			titleFuzzy := bleve.NewFuzzyQuery(term)
			titleFuzzy.SetField("Title")
			titleFuzzy.SetFuzziness(1) // Allow 1 character difference
			titleFuzzy.SetBoost(0.5)   // Lower boost for fuzzy matches

			descFuzzy := bleve.NewFuzzyQuery(term)
			descFuzzy.SetField("Description")
			descFuzzy.SetFuzziness(1)
			descFuzzy.SetBoost(0.3)

			fuzzyQueries = append(fuzzyQueries, titleFuzzy, descFuzzy)
		}
	}

	// Combine all strategies with OR (disjunction)
	var allQueries []blevequery.Query
	allQueries = append(allQueries, exactQueries...)
	allQueries = append(allQueries, termQueries...)
	allQueries = append(allQueries, fuzzyQueries...)

	contentQuery := bleve.NewDisjunctionQuery(allQueries...)

	// Apply archive filter if needed
	if archQ != nil {
		return bleve.NewConjunctionQuery(contentQuery, archQ)
	}

	return contentQuery
}

var ErrNotExists = errors.New("not exists")

func (s *Store) Edit(_ context.Context, id int64, mutate func(*Todo) error) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	todo, err := s.findByID(id)
	if err != nil {
		return err
	}
	original := *todo
	if err := mutate(todo); err != nil {
		return err
	}
	if todo.ID != id {
		panic("don't mutate todo IDs")
	}
	if err := Validate(todo.Title, todo.Description); err.IsErr() {
		// Rollback
		*todo = original
		return err
	}

	return s.searchIndex.Index(strconv.FormatInt(id, 10), map[string]any{
		"Title":       todo.Title,
		"Description": todo.Description,
		"Archived":    todo.Archived,
	})
}

func (s *Store) Archive(_ context.Context, id int64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	todo, err := s.findByID(id)
	if err != nil {
		return err
	}
	if err := s.searchIndex.Index(strconv.FormatInt(id, 10), map[string]any{
		"Title":       todo.Title,
		"Description": todo.Description,
		"Archived":    todo.Archived,
	}); err != nil {
		return err
	}
	todo.Archived = true

	return nil
}

func (s *Store) Delete(_ context.Context, id int64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	todo, err := s.findByID(id)
	if err != nil {
		return err
	}
	todo.Archived = true

	s.todos = slices.DeleteFunc(s.todos, func(t *Todo) bool { return t.ID == id })
	delete(s.indexByID, id)
	return s.searchIndex.Delete(strconv.FormatInt(id, 10))
}
