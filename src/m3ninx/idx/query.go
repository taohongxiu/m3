// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package idx

import (
	"github.com/m3db/m3ninx/search"
	"github.com/m3db/m3ninx/search/query"
)

// Query encapsulates a search query for an index.
type Query struct {
	query search.Query
}

// NewTermQuery returns a new query for finding documents which match a term exactly.
func NewTermQuery(field, term []byte) Query {
	return Query{
		query: query.NewTermQuery(field, term),
	}
}

// NewRegexpQuery returns a new query for finding documents which match a regular expression.
func NewRegexpQuery(field, regexp []byte) (Query, error) {
	q, err := query.NewRegexpQuery(field, regexp)
	if err != nil {
		return Query{}, err
	}
	return Query{
		query: q,
	}, nil
}

// NewConjunctionQuery returns a new query for finding documents which match each of the
// given queries.
func NewConjunctionQuery(queries ...Query) (Query, error) {
	qs := make([]search.Query, 0, len(queries))
	for _, q := range queries {
		qs = append(qs, q.query)
	}
	return Query{
		query: query.NewConjuctionQuery(qs),
	}, nil
}

// NewDisjunctionQuery returns a new query for finding documents which match at least one
// of the given queries.
func NewDisjunctionQuery(queries ...Query) (Query, error) {
	qs := make([]search.Query, 0, len(queries))
	for _, q := range queries {
		qs = append(qs, q.query)
	}
	return Query{
		query: query.NewDisjuctionQuery(qs),
	}, nil
}